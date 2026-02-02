package controllers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"wx_channel/internal/utils"
)

func PlayVideo(w http.ResponseWriter, r *http.Request) {
	targetURL := r.URL.Query().Get("url")
	fmt.Printf("[PlayVideo] Target URL: %s\n", targetURL)
	if targetURL == "" {
		fmt.Println("[PlayVideo] Error: url parameter required")
		http.Error(w, "url parameter required", http.StatusBadRequest)
		return
	}

	// 获取可选的解密密钥
	decryptKeyStr := r.URL.Query().Get("key")
	fmt.Printf("[PlayVideo] Decrypt key string: %s\n", decryptKeyStr)
	var decryptKey uint64
	var needsDecryption bool

	if decryptKeyStr != "" {
		var err error
		decryptKey, err = strconv.ParseUint(decryptKeyStr, 10, 64)
		if err != nil {
			fmt.Printf("[PlayVideo] Error: failed to parse decrypt key: %v\n", err)
			http.Error(w, "invalid decryption key", http.StatusBadRequest)
			return
		}
		needsDecryption = true
		fmt.Printf("[PlayVideo] Decrypt key parsed: %d\n", decryptKey)
	}

	// 创建上游请求
	req, err := http.NewRequest(r.Method, targetURL, nil)
	if err != nil {
		fmt.Printf("[PlayVideo] Error: failed to create request: %v\n", err)
		http.Error(w, "invalid URL", http.StatusBadRequest)
		return
	}

	// 复制 Range 头（支持视频拖动）
	if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}

	// 发送请求
	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "failed to fetch video", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Range")

	// 复制响应头
	for k, v := range resp.Header {
		w.Header()[k] = v
	}

	// 确保设置 Accept-Ranges
	if w.Header().Get("Accept-Ranges") == "" {
		w.Header().Set("Accept-Ranges", "bytes")
	}

	// 如果需要解密
	if needsDecryption {
		// 解析 Content-Range 头以获取起始偏移
		var startOffset uint64 = 0
		if cr := resp.Header.Get("Content-Range"); cr != "" {
			// Content-Range 格式: "bytes start-end/total"
			parts := strings.Split(cr, " ")
			if len(parts) == 2 {
				rangePart := parts[1]
				dashIdx := strings.Index(rangePart, "-")
				if dashIdx > 0 {
					if v, err := strconv.ParseUint(rangePart[:dashIdx], 10, 64); err == nil {
						startOffset = v
					}
				}
			}
		}

		// 创建解密读取器
		// 加密区域大小为 131072 字节（128KB）
		decryptReader := utils.NewDecryptReader(resp.Body, decryptKey, startOffset, 131072)

		// 写入状态码
		w.WriteHeader(resp.StatusCode)

		// 如果是 HEAD 请求，不传输内容
		if r.Method == "HEAD" {
			return
		}

		// 流式复制解密后的数据到客户端
		io.Copy(w, decryptReader)
	} else {
		// 无需解密，直接代理
		w.WriteHeader(resp.StatusCode)

		// 如果是 HEAD 请求，不传输内容
		if r.Method == "HEAD" {
			return
		}

		// 流式复制数据到客户端
		io.Copy(w, resp.Body)
	}
}
