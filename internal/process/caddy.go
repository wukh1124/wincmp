package process

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	// Caddy 停止後清理 timberjack 在 Windows 上無法刪除的殘留 log 檔
	m.cleanupStaleRotatedLogs()

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

// cleanupStaleRotatedLogs 清理 timberjack 在 Windows 上因檔案鎖定無法刪除的殘留 log 檔。
// 當 timberjack 進行 log rotation 時，會先壓縮舊 log 為 .gz，再刪除原始檔；
// 但在 Windows 上，若 Caddy 仍持有 file handle，刪除會失敗。
// 此函數在 Caddy 停止後呼叫，掃描 logs/ 目錄中同時存在 .log 和 .log.gz 的配對，
// 刪除已成功壓縮的原始 .log 檔。
func (m *Manager) cleanupStaleRotatedLogs() {
	logsDir := filepath.Join(m.baseDir, "logs")

	entries, err := os.ReadDir(logsDir)
	if err != nil {
		// 目錄不存在或無法讀取，靜默忽略
		return
	}

	// 建立已存在 .gz 檔的集合（不含 .gz 後綴），用於快速查詢配對
	gzBaseSet := make(map[string]bool)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".log.gz") {
			// "access-2026-04-09T23-46-26.714-size.log.gz" → "access-2026-04-09T23-46-26.714-size.log"
			baseName := strings.TrimSuffix(entry.Name(), ".gz")
			gzBaseSet[baseName] = true
		}
	}

	// 刪除有對應 .gz 的殘留 .log 檔
	cleaned := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// 只處理 .log 結尾（排除 .log.gz，那個是壓縮檔要保留）
		if !strings.HasSuffix(name, ".log") || strings.HasSuffix(name, ".log.gz") {
			continue
		}
		if gzBaseSet[name] {
			fullPath := filepath.Join(logsDir, name)
			if err := os.Remove(fullPath); err != nil {
				m.errorLog("caddy", fmt.Sprintf("清理殘留 log 檔失敗: %s", name), err)
			} else {
				cleaned++
				m.log("caddy", "🧹 清理殘留 log: %s", name)
			}
		}
	}

	if cleaned > 0 {
		m.log("caddy", "🧹 已清理 %d 個 timberjack 殘留 log 檔", cleaned)
	}
}
