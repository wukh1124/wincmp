package process

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"wincmp/internal/scanner"
)

// PHPServiceKey 產生 PHP-CGI 服務的唯一識別 key
func PHPServiceKey(version string) string {
	return "php-" + version
}

// StartPHPCGI 啟動指定版本的 PHP-CGI 行程（多 port 負載平衡）
// 對應 bat 指令:
//
//	start "" /b bin\php\php-8.2.30\php-cgi.exe -b 127.0.0.1:9821
//	start "" /b bin\php\php-8.2.30\php-cgi.exe -b 127.0.0.1:9822
//	start "" /b bin\php\php-8.2.30\php-cgi.exe -b 127.0.0.1:9823
func (m *Manager) StartPHPCGI(phpInfo scanner.PHPVersionInfo) error {
	serviceKey := PHPServiceKey(phpInfo.Version)

	if m.IsRunning(serviceKey) {
		return fmt.Errorf("PHP-CGI %s 已經在運行中", phpInfo.Version)
	}

	ports := phpInfo.GetPHPPorts()
	m.log("php", "🚀 啟動 PHP-CGI %s (%d 個行程)...", phpInfo.Version, len(ports))
	m.log("php", "  執行檔: %s", phpInfo.ExePath)

	// 為 PHP 動態注入 PATH 環境變數（確保 DLL 依賴正確載入）
	phpDir := filepath.Dir(phpInfo.ExePath)
	env := append(os.Environ(),
		fmt.Sprintf("PATH=%s;%s", phpDir, os.Getenv("PATH")),
	)

	// 使用通用的 conf/php/php.ini，並動態注入 extension_dir
	phpIniPath := filepath.Join(m.baseDir, "conf", "php", "php.ini")
	extDir := filepath.Join(phpDir, "ext")

	cmds := make([]*exec.Cmd, 0, len(ports))

	for _, port := range ports {
		bindAddr := fmt.Sprintf("127.0.0.1:%d", port)

		cmd := exec.Command(phpInfo.ExePath,
			"-c", phpIniPath,
			"-d", "extension_dir="+extDir,
			"-b", bindAddr,
		)
		cmd.Dir = m.baseDir
		cmd.Env = env
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}

		// PHP-CGI 的輸出通常不多，但仍擷取 stderr
		m.pipeOutput(cmd, "php", fmt.Sprintf("PHP-%s", phpInfo.MajorMin))

		if err := cmd.Start(); err != nil {
			// 有一個啟動失敗，終止已啟動的
			m.errorLog("php", fmt.Sprintf("PHP-CGI %s 在 Port %d 啟動失敗", phpInfo.Version, port), err)
			for _, c := range cmds {
				if c.Process != nil {
					_ = c.Process.Kill()
				}
			}
			return fmt.Errorf("PHP-CGI %s 啟動失敗: %w", phpInfo.Version, err)
		}

		m.log("php", "  ✓ PID %d → %s", cmd.Process.Pid, bindAddr)
		cmds = append(cmds, cmd)
	}

	m.register(serviceKey, "PHP-CGI "+phpInfo.Version, phpInfo.ExePath, cmds)
	m.log("php", "✅ PHP-CGI %s 已啟動 (%s)", phpInfo.Version, phpInfo.GetPortRangeStr())

	// 監控每個行程的退出
	for i, cmd := range cmds {
		go func(c *exec.Cmd, port int) {
			err := c.Wait()
			if m.IsRunning(serviceKey) {
				if err != nil {
					m.errorLog("php", fmt.Sprintf("PHP-CGI %s (Port %d) 異常退出", phpInfo.Version, port), err)
				}
				// 注意：這裡不自動 unregister，因為其他行程可能還在運行
				// 需要更精細的處理，暫時先記錄 log
			}
		}(cmd, ports[i])
	}

	return nil
}

// StopPHPCGI 停止指定版本的所有 PHP-CGI 行程
func (m *Manager) StopPHPCGI(version string) error {
	serviceKey := PHPServiceKey(version)

	if !m.IsRunning(serviceKey) {
		return fmt.Errorf("PHP-CGI %s 未在運行", version)
	}

	m.log("php", "🛑 停止 PHP-CGI %s...", version)
	if err := m.stopService(serviceKey); err != nil {
		m.errorLog("php", fmt.Sprintf("PHP-CGI %s 停止失敗", version), err)
		return err
	}
	m.log("php", "✅ PHP-CGI %s 已停止", version)
	return nil
}
