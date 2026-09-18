package process

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"wincmp/internal/i18n"
)

const redisServiceKey = "redis"

// RedisServiceKey 回傳 Redis 服務的唯一識別 key
func RedisServiceKey() string {
	return redisServiceKey
}

// StartRedis 啟動 Redis 服務 (預設 127.0.0.1:6379)
func (m *Manager) StartRedis(version, exePath string, port int) error {
	if m.IsRunning(redisServiceKey) {
		return fmt.Errorf("%s", i18n.T("Redis 已經在運行中"))
	}

	if port <= 0 {
		port = 6379
	}

	redisDir := filepath.Dir(exePath)
	confPath := filepath.Join(redisDir, "redis.windows.conf")
	if _, err := os.Stat(confPath); os.IsNotExist(err) {
		confPath = filepath.Join(m.baseDir, "conf", "redis.conf")
	}

	args := []string{}
	if _, err := os.Stat(confPath); err == nil {
		args = append(args, confPath)
	}

	args = append(args, "--bind", "127.0.0.1", "--port", fmt.Sprintf("%d", port))

	cmd := m.createCommand(exePath, args...)
	cmd.Dir = redisDir
	m.pipeOutput(cmd, "redis", "Redis")

	m.log("redis", "%s", i18n.T("🚀 啟動 Redis 服務..."))
	m.log("redis", "%s", i18n.Tfmt("  執行檔: %s", exePath))
	m.log("redis", "%s", i18n.Tfmt("  連線埠: 127.0.0.1:%d", port))

	if err := cmd.Start(); err != nil {
		m.errorLog("redis", i18n.T("Redis 啟動失敗"), err)
		return fmt.Errorf("%s: %w", i18n.T("Redis 啟動失敗"), err)
	}

	m.register(redisServiceKey, fmt.Sprintf("Redis (%s)", version), exePath, []*exec.Cmd{cmd})
	m.log("redis", "%s", i18n.Tfmt("✅ Redis (%s) 已啟動 (PID: %d)", version, cmd.Process.Pid))

	go m.waitForExit(cmd, redisServiceKey, "redis", "Redis")

	return nil
}

// StopRedis 停止 Redis 服務
func (m *Manager) StopRedis() error {
	if !m.IsRunning(redisServiceKey) {
		return fmt.Errorf("%s", i18n.T("Redis 未在運行"))
	}

	m.log("redis", "%s", i18n.T("🛑 停止 Redis..."))
	if err := m.stopService(redisServiceKey); err != nil {
		m.errorLog("redis", i18n.T("Redis 停止失敗"), err)
		return err
	}
	m.log("redis", "%s", i18n.T("✅ Redis 已停止"))
	return nil
}
