package process

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

const caddyServiceKey = "caddy"

// StartCaddy 啟動 Caddy 服務
func (m *Manager) StartCaddy(version, exePath string) error {
	if m.IsRunning(caddyServiceKey) {
		return fmt.Errorf("Caddy 已經在運行中")
	}

	configPath := filepath.Join(m.baseDir, "conf", "Caddyfile")

	cmd := m.createCommand(exePath,
		"run",
		"--config", configPath,
		"--adapter", "caddyfile",
		"--watch",
	)

	// 擷取 stdout/stderr 輸出到 Terminal Logs
	m.pipeOutput(cmd, "caddy", "Caddy")

	m.log("caddy", "🚀 啟動 Caddy...")
	m.log("caddy", "  執行檔: %s", exePath)
	m.log("caddy", "  設定檔: %s", configPath)

	if err := cmd.Start(); err != nil {
		m.errorLog("caddy", "Caddy 啟動失敗", err)
		return fmt.Errorf("Caddy 啟動失敗: %w", err)
	}

	m.register(caddyServiceKey, fmt.Sprintf("Caddy (%s)", version), exePath, []*exec.Cmd{cmd})
	m.log("caddy", "✅ Caddy (%s) 已啟動 (PID: %d)", version, cmd.Process.Pid)

	// 監控程序退出
	go m.waitForExit(cmd, caddyServiceKey, "caddy", "Caddy")

	return nil
}

// StopCaddy 停止 Caddy 服務
func (m *Manager) StopCaddy() error {
	if !m.IsRunning(caddyServiceKey) {
		return fmt.Errorf("Caddy 未在運行")
	}

	m.log("caddy", "🛑 停止 Caddy...")
	if err := m.stopService(caddyServiceKey); err != nil {
		m.errorLog("caddy", "Caddy 停止失敗", err)
		return err
	}
	m.log("caddy", "✅ Caddy 已停止")
	return nil
}

// ReloadCaddy 重載 Caddy 設定
func (m *Manager) ReloadCaddy(exePath string) error {
	if !m.IsRunning(caddyServiceKey) {
		return fmt.Errorf("Caddy 未在運行，無法重載")
	}

	configPath := filepath.Join(m.baseDir, "conf", "Caddyfile")

	cmd := m.createCommand(exePath,
		"reload",
		"--config", configPath,
		"--adapter", "caddyfile",
	)

	m.log("caddy", "🔄 重載 Caddy 設定...")
	output, err := cmd.CombinedOutput()
	if err != nil {
		m.errorLog("caddy", fmt.Sprintf("Caddy 重載失敗\n%s", string(output)), err)
		return fmt.Errorf("Caddy 重載失敗: %w", err)
	}

	m.log("caddy", "✅ Caddy 設定已重載")
	return nil
}

