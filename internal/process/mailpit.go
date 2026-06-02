package process

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"wincmp/internal/i18n"
)

const mailpitServiceKey = "mailpit"

// MailpitServiceKey 回傳 Mailpit 服務的唯一識別 key
func MailpitServiceKey() string {
	return mailpitServiceKey
}

// StartMailpit 啟動 Mailpit 服務
func (m *Manager) StartMailpit(version, exePath string, smtpPort, httpPort int, useDB bool) error {
	if m.IsRunning(mailpitServiceKey) {
		return fmt.Errorf("%s", i18n.T("Mailpit 已經在運行中"))
	}

	args := []string{
		"-s", fmt.Sprintf("127.0.0.1:%d", smtpPort),
		"-l", fmt.Sprintf("127.0.0.1:%d", httpPort),
		"--smtp-auth-accept-any",
		"--smtp-auth-allow-insecure",
	}

	if useDB {
		dataDir := filepath.Join(m.baseDir, "data", "mailpit")
		if err := os.MkdirAll(dataDir, 0700); err != nil {
			return fmt.Errorf("%s: %w", i18n.T("建立 Mailpit 資料目錄失敗"), err)
		}
		args = append(args, "-d", dataDir)
		m.log("mailpit", "%s", i18n.Tfmt("  資料目錄: %s", dataDir))
	}

	cmd := m.createCommand(exePath, args...)
	m.pipeOutput(cmd, "mailpit", "Mailpit")

	m.log("mailpit", "%s", i18n.T("🚀 啟動 Mailpit..."))
	m.log("mailpit", "%s", i18n.Tfmt("  執行檔: %s", exePath))
	m.log("mailpit", "%s", i18n.Tfmt("  SMTP Port: %d", smtpPort))
	m.log("mailpit", "%s", i18n.Tfmt("  HTTP Port: %d", httpPort))
	if useDB {
		m.log("mailpit", "%s", i18n.T("  持久化: 啟用 (database)"))
	} else {
		m.log("mailpit", "%s", i18n.T("  持久化: 停用 (記憶體模式)"))
	}

	if err := cmd.Start(); err != nil {
		m.errorLog("mailpit", i18n.T("Mailpit 啟動失敗"), err)
		return fmt.Errorf("%s: %w", i18n.T("Mailpit 啟動失敗"), err)
	}

	m.register(mailpitServiceKey, fmt.Sprintf("Mailpit (%s)", version), exePath, []*exec.Cmd{cmd})
	m.log("mailpit", "%s", i18n.Tfmt("✅ Mailpit (%s) 已啟動 (PID: %d)", version, cmd.Process.Pid))

	go m.waitForExit(cmd, mailpitServiceKey, "mailpit", "Mailpit")

	return nil
}

// StopMailpit 停止 Mailpit 服務
func (m *Manager) StopMailpit() error {
	if !m.IsRunning(mailpitServiceKey) {
		return fmt.Errorf("%s", i18n.T("Mailpit 未在運行"))
	}

	m.log("mailpit", "%s", i18n.T("🛑 停止 Mailpit..."))
	if err := m.stopService(mailpitServiceKey); err != nil {
		m.errorLog("mailpit", i18n.T("Mailpit 停止失敗"), err)
		return err
	}
	m.log("mailpit", "%s", i18n.T("✅ Mailpit 已停止"))
	return nil
}
