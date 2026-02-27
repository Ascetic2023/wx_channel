package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"wx_channel/internal/database"
	"wx_channel/internal/utils"
)

// TranscriptionService 处理视频语音转文字业务逻辑（whisper-server 模式）
type TranscriptionService struct {
	settingsRepo  *database.SettingsRepository
	downloadRepo  *database.DownloadRecordRepository
	mu            sync.Mutex
	activeJobs    map[string]context.CancelFunc
	serverCmd     *exec.Cmd
	serverPort    int
	serverRunning bool
}

// NewTranscriptionService 创建一个新的 TranscriptionService
func NewTranscriptionService() *TranscriptionService {
	return &TranscriptionService{
		settingsRepo: database.NewSettingsRepository(),
		downloadRepo: database.NewDownloadRecordRepository(),
		activeJobs:   make(map[string]context.CancelFunc),
	}
}

// IsEnabled 检查转写功能是否启用
func (s *TranscriptionService) IsEnabled() bool {
	enabled, _ := s.settingsRepo.GetBool(database.SettingKeyTranscriptionEnabled, false)
	return enabled
}

// IsAutoRunEnabled 检查是否启用了自动转写
func (s *TranscriptionService) IsAutoRunEnabled() bool {
	if !s.IsEnabled() {
		return false
	}
	autoRun, _ := s.settingsRepo.GetBool(database.SettingKeyTranscriptionAutoRun, false)
	return autoRun
}

// ValidateTools 检测 FFmpeg 和 whisper-server 是否可用
func (s *TranscriptionService) ValidateTools() (bool, string) {
	ffmpegPath := s.getFFmpegPath()
	if ffmpegPath == "" {
		return false, "未找到 FFmpeg，请在设置中配置 FFmpeg 路径或将其添加到系统 PATH"
	}

	// 测试 FFmpeg
	cmd := exec.Command(ffmpegPath, "-version")
	if err := cmd.Run(); err != nil {
		return false, fmt.Sprintf("FFmpeg 执行失败: %v", err)
	}

	serverPath := s.getWhisperServerPath()
	if serverPath == "" {
		return false, "未找到 whisper-server 程序，请在设置中配置路径或将其添加到系统 PATH"
	}

	// 测试 whisper-server 可执行（使用 --help）
	cmd = exec.Command(serverPath, "--help")
	if err := cmd.Run(); err != nil {
		return false, fmt.Sprintf("whisper-server 执行失败: %v", err)
	}

	// 检查模型文件
	modelPath := s.getModelPath()
	if modelPath == "" {
		return false, "未配置 Whisper 模型文件路径"
	}
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return false, fmt.Sprintf("模型文件不存在: %s", modelPath)
	}

	return true, "FFmpeg 和 whisper-server 工具检测通过"
}

// TranscribeVideo 同步执行视频转写
func (s *TranscriptionService) TranscribeVideo(ctx context.Context, recordID string) error {
	// 获取下载记录
	record, err := s.downloadRepo.GetByID(recordID)
	if err != nil {
		return fmt.Errorf("获取下载记录失败: %w", err)
	}
	if record == nil {
		return fmt.Errorf("下载记录不存在: %s", recordID)
	}

	// 验证视频文件存在
	if _, err := os.Stat(record.FilePath); os.IsNotExist(err) {
		return fmt.Errorf("视频文件不存在: %s", record.FilePath)
	}

	// 检查是否已在转写
	s.mu.Lock()
	if _, exists := s.activeJobs[recordID]; exists {
		s.mu.Unlock()
		return fmt.Errorf("该视频正在转写中")
	}

	// 注册取消函数
	ctx, cancel := context.WithCancel(ctx)
	s.activeJobs[recordID] = cancel
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.activeJobs, recordID)
		s.mu.Unlock()
		cancel()
	}()

	// 计算输出路径
	ext := filepath.Ext(record.FilePath)
	txtPath := strings.TrimSuffix(record.FilePath, ext) + ".txt"

	// 标记状态为转写中
	if err := s.downloadRepo.UpdateTranscriptStatus(recordID, database.TranscriptStatusInProgress, ""); err != nil {
		utils.Error("更新转写状态失败: %v", err)
	}

	// 执行转写
	if err := s.doTranscribe(ctx, record.FilePath, txtPath); err != nil {
		_ = s.downloadRepo.UpdateTranscriptStatus(recordID, database.TranscriptStatusFailed, "")
		return fmt.Errorf("转写失败: %w", err)
	}

	// 验证输出文件
	if _, err := os.Stat(txtPath); os.IsNotExist(err) {
		_ = s.downloadRepo.UpdateTranscriptStatus(recordID, database.TranscriptStatusFailed, "")
		return fmt.Errorf("转写完成但输出文件不存在: %s", txtPath)
	}

	// 标记完成
	if err := s.downloadRepo.UpdateTranscriptStatus(recordID, database.TranscriptStatusCompleted, txtPath); err != nil {
		return fmt.Errorf("更新转写状态失败: %w", err)
	}

	utils.Info("✅ 语音转文字完成: %s -> %s", record.Title, txtPath)

	// 转写完成后删除视频文件（如果设置了）
	if s.isDeleteAfterTranscriptEnabled() {
		if err := os.Remove(record.FilePath); err != nil {
			utils.Warn("转写后删除视频文件失败: %v", err)
		} else {
			utils.Info("🗑️ 已删除视频文件: %s", record.FilePath)
		}
	}

	return nil
}

// TranscribeAsync 异步执行转写
func (s *TranscriptionService) TranscribeAsync(recordID string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		if err := s.TranscribeVideo(ctx, recordID); err != nil {
			utils.Error("异步转写失败 [%s]: %v", recordID, err)
		}
	}()
}

// CancelTranscription 取消正在进行的转写
func (s *TranscriptionService) CancelTranscription(recordID string) error {
	s.mu.Lock()
	cancel, exists := s.activeJobs[recordID]
	s.mu.Unlock()

	if !exists {
		return fmt.Errorf("没有正在进行的转写任务: %s", recordID)
	}

	cancel()
	_ = s.downloadRepo.UpdateTranscriptStatus(recordID, database.TranscriptStatusFailed, "")
	return nil
}

// GetTranscript 获取转写文本内容
func (s *TranscriptionService) GetTranscript(recordID string) (string, error) {
	record, err := s.downloadRepo.GetByID(recordID)
	if err != nil {
		return "", fmt.Errorf("获取下载记录失败: %w", err)
	}
	if record == nil {
		return "", fmt.Errorf("下载记录不存在: %s", recordID)
	}

	if record.TranscriptPath == "" || record.TranscriptStatus != database.TranscriptStatusCompleted {
		return "", fmt.Errorf("转写尚未完成")
	}

	data, err := os.ReadFile(record.TranscriptPath)
	if err != nil {
		return "", fmt.Errorf("读取转写文件失败: %w", err)
	}

	return string(data), nil
}

// GetTranscriptPath 获取转写文件路径
func (s *TranscriptionService) GetTranscriptPath(recordID string) (string, error) {
	record, err := s.downloadRepo.GetByID(recordID)
	if err != nil {
		return "", fmt.Errorf("获取下载记录失败: %w", err)
	}
	if record == nil {
		return "", fmt.Errorf("下载记录不存在: %s", recordID)
	}

	if record.TranscriptPath == "" {
		return "", fmt.Errorf("转写文件不存在")
	}

	return record.TranscriptPath, nil
}

// StopServer 停止 whisper-server 进程
func (s *TranscriptionService) StopServer() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.serverCmd != nil && s.serverCmd.Process != nil {
		utils.Info("正在停止 whisper-server...")
		_ = s.serverCmd.Process.Kill()
		_ = s.serverCmd.Wait()
		s.serverCmd = nil
		s.serverRunning = false
		utils.Info("whisper-server 已停止")
	}
}

// ensureServerRunning 确保 whisper-server 正在运行
func (s *TranscriptionService) ensureServerRunning() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查进程是否还活着
	if s.serverRunning && s.serverCmd != nil && s.serverCmd.Process != nil {
		port := s.serverPort
		s.mu.Unlock()
		alive := s.pingServer(port)
		s.mu.Lock()
		if alive {
			return nil
		}
		// 进程已死，清理
		s.serverRunning = false
		s.serverCmd = nil
	}

	return s.startServerLocked()
}

// startServerLocked 启动 whisper-server（调用方已持锁）
func (s *TranscriptionService) startServerLocked() error {
	serverPath := s.getWhisperServerPath()
	if serverPath == "" {
		return fmt.Errorf("whisper-server 路径未配置")
	}
	modelPath := s.getModelPath()
	if modelPath == "" {
		return fmt.Errorf("Whisper 模型路径未配置")
	}

	port := s.getServerPort()
	portStr := strconv.Itoa(port)

	utils.Info("🚀 正在启动 whisper-server (端口 %d)...", port)

	cmd := exec.Command(serverPath,
		"-m", modelPath,
		"--port", portStr,
		"--host", "127.0.0.1",
	)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 whisper-server 失败: %w", err)
	}

	s.serverCmd = cmd
	s.serverPort = port

	// 释放锁等待 server 就绪
	s.mu.Unlock()
	err := s.waitForServerReady(port, 120*time.Second)
	s.mu.Lock()

	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		s.serverCmd = nil
		return fmt.Errorf("whisper-server 启动超时: %w", err)
	}

	s.serverRunning = true
	utils.Info("✅ whisper-server 已就绪 (端口 %d)", port)

	// 后台监听进程退出
	go func() {
		_ = cmd.Wait()
		s.mu.Lock()
		if s.serverCmd == cmd {
			s.serverRunning = false
			s.serverCmd = nil
			utils.Warn("whisper-server 进程已退出")
		}
		s.mu.Unlock()
	}()

	return nil
}

// waitForServerReady 轮询等待 server 就绪
func (s *TranscriptionService) waitForServerReady(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if s.pingServer(port) {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("超时等待 whisper-server 启动（端口 %d）", port)
}

// pingServer 检查 server 是否可用
func (s *TranscriptionService) pingServer(port int) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
	if err != nil {
		return false
	}
	resp.Body.Close()
	return true
}

// doTranscribe 执行实际的转写流程: 确保 server → 提取音频 → HTTP POST → 保存结果
func (s *TranscriptionService) doTranscribe(ctx context.Context, videoPath, txtPath string) error {
	// 1. 确保 whisper-server 在运行
	if err := s.ensureServerRunning(); err != nil {
		return fmt.Errorf("whisper-server 未就绪: %w", err)
	}

	ffmpegPath := s.getFFmpegPath()
	if ffmpegPath == "" {
		return fmt.Errorf("FFmpeg 路径未配置")
	}

	// 2. 用 FFmpeg 提取音频
	wavPath := videoPath + ".tmp.wav"
	utils.Info("🎵 正在提取音频: %s", filepath.Base(videoPath))

	ffmpegArgs := []string{
		"-i", videoPath,
		"-ar", "16000",
		"-ac", "1",
		"-f", "wav",
		"-y",
		wavPath,
	}

	cmd := exec.CommandContext(ctx, ffmpegPath, ffmpegArgs...)
	if output, err := cmd.CombinedOutput(); err != nil {
		os.Remove(wavPath)
		return fmt.Errorf("FFmpeg 提取音频失败: %v, 输出: %s", err, string(output))
	}
	defer os.Remove(wavPath)

	// 3. HTTP POST multipart 到 whisper-server /inference
	utils.Info("🗣️ 正在识别语音: %s", filepath.Base(videoPath))

	text, err := s.postInference(ctx, wavPath)
	if err != nil {
		return fmt.Errorf("whisper-server 识别失败: %w", err)
	}

	// 4. 将结果写入 txt 文件
	if err := os.WriteFile(txtPath, []byte(strings.TrimSpace(text)), 0644); err != nil {
		return fmt.Errorf("写入转写文件失败: %w", err)
	}

	return nil
}

// postInference 向 whisper-server 发送音频文件进行识别
func (s *TranscriptionService) postInference(ctx context.Context, wavPath string) (string, error) {
	file, err := os.Open(wavPath)
	if err != nil {
		return "", fmt.Errorf("打开音频文件失败: %w", err)
	}
	defer file.Close()

	// 构造 multipart body
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filepath.Base(wavPath))
	if err != nil {
		return "", fmt.Errorf("创建 multipart 字段失败: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("写入音频数据失败: %w", err)
	}

	_ = writer.WriteField("response_format", "text")

	language := s.getLanguage()
	if language != "auto" {
		_ = writer.WriteField("language", language)
	}

	writer.Close()

	// 发送请求
	s.mu.Lock()
	port := s.serverPort
	s.mu.Unlock()

	url := fmt.Sprintf("http://127.0.0.1:%d/inference", port)
	req, err := http.NewRequestWithContext(ctx, "POST", url, &body)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求 whisper-server 失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("whisper-server 返回错误 %d: %s", resp.StatusCode, string(respBody))
	}

	return string(respBody), nil
}

// getFFmpegPath 获取 FFmpeg 路径
func (s *TranscriptionService) getFFmpegPath() string {
	path, _ := s.settingsRepo.Get(database.SettingKeyFFmpegPath)
	if path != "" {
		return path
	}
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		return p
	}
	return ""
}

// getWhisperServerPath 获取 whisper-server 路径
func (s *TranscriptionService) getWhisperServerPath() string {
	path, _ := s.settingsRepo.Get(database.SettingKeyWhisperServerPath)
	if path != "" {
		return path
	}
	for _, name := range []string{"whisper-server", "server"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

// getModelPath 获取模型文件路径
func (s *TranscriptionService) getModelPath() string {
	path, _ := s.settingsRepo.Get(database.SettingKeyWhisperModelPath)
	return path
}

// getServerPort 获取 whisper-server 端口
func (s *TranscriptionService) getServerPort() int {
	port, _ := s.settingsRepo.GetInt(database.SettingKeyWhisperServerPort, 8178)
	return port
}

// getLanguage 获取转写语言
func (s *TranscriptionService) getLanguage() string {
	lang, _ := s.settingsRepo.Get(database.SettingKeyTranscriptionLanguage)
	if lang == "" {
		return "zh"
	}
	return lang
}

// isDeleteAfterTranscriptEnabled 检查是否启用了转写后删除视频
func (s *TranscriptionService) isDeleteAfterTranscriptEnabled() bool {
	enabled, _ := s.settingsRepo.GetBool(database.SettingKeyDeleteVideoAfterTranscript, false)
	return enabled
}
