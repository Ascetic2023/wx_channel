package config

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestLoad_Defaults(t *testing.T) {
	// 确保没有环境变量或配置文件干扰
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if strings.HasPrefix(pair[0], "WX_CHANNEL_") {
			os.Unsetenv(pair[0])
		}
	}

	viper.Reset()
	globalConfig = nil // 重置单例

	// 设置临时目录作为 HOME 防止读取用户配置
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	cfg := Load()

	assert.NotNil(t, cfg)
	assert.Equal(t, 2025, cfg.Port)
	assert.Equal(t, "5.3.0", cfg.Version)
	assert.Equal(t, int64(2<<20), cfg.ChunkSize)
	assert.Equal(t, 500*time.Millisecond, cfg.SaveDelay)
}

func TestLoad_EnvVars(t *testing.T) {
	viper.Reset()
	globalConfig = nil // 重置单例

	// 清理相关环境变量
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if strings.HasPrefix(pair[0], "WX_CHANNEL_") {
			os.Unsetenv(pair[0])
		}
	}

	t.Setenv("WX_CHANNEL_PORT", "9999")
	t.Setenv("WX_CHANNEL_LOG_FILE", "test.log")

	cfg := Load()

	assert.Equal(t, 9999, cfg.Port)
	assert.Equal(t, "test.log", cfg.LogFile)
}

func TestSetPort(t *testing.T) {
	cfg := &Config{Port: 8080}
	cfg.SetPort(9090)
	assert.Equal(t, 9090, cfg.Port)
}
