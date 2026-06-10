package process

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"syscall"

	"wincmp/internal/config"
	"wincmp/internal/i18n"
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
		return fmt.Errorf("%s", i18n.Tfmt("PHP-CGI %s 已經在運行中", phpInfo.Version))
	}

	ports := phpInfo.GetPHPPorts()
	m.log("php", "%s", i18n.Tfmt("🚀 啟動 PHP-CGI %s (%d 個行程)...", phpInfo.Version, len(ports)))
	m.log("php", "%s", i18n.Tfmt("  執行檔: %s", phpInfo.ExePath))

	// 為 PHP 動態注入 PATH 環境變數（確保 DLL 依賴正確載入）
	phpDir := filepath.Dir(phpInfo.ExePath)
	env := append(os.Environ(),
		fmt.Sprintf("PATH=%s;%s", phpDir, os.Getenv("PATH")),
	)

	// 使用通用的 conf/php/php.ini，並動態注入 extension_dir
	phpIniPath := filepath.Join(m.baseDir, "conf", "php", "php.ini")
	extDir := filepath.Join(phpDir, "ext")

	// 檢查並自動下載 cacert.pem
	cacertPath := filepath.Join(m.baseDir, "conf", "ssl", "cacert.pem")
	if _, err := os.Stat(cacertPath); os.IsNotExist(err) {
		m.log("php", "%s", i18n.T("ℹ️ 未檢測到 cacert.pem，準備自動從網際網路下載..."))
		depPath := filepath.Join(m.baseDir, "conf", "dependencies.json")
		depCfg, err := config.LoadDependencies(depPath)
		url := "https://curl.se/ca/cacert.pem"
		if err == nil {
			if item, ok := depCfg["cacert"]; ok && item.URL != "" {
				url = item.URL
			}
		}
		m.log("php", "%s", i18n.Tfmt("🌐 正在從 %s 下載 cacert.pem...", url))
		if err := downloadCacert(url, cacertPath); err != nil {
			m.errorLog("php", i18n.T("❌ 下載 cacert.pem 失敗，可能會影響 PHP 的 SSL 連線功能"), err)
		} else {
			m.log("php", "%s", i18n.T("✅ cacert.pem 下載並設定成功！"))
		}
	}

	cmds := make([]*exec.Cmd, 0, len(ports))

	for _, port := range ports {
		bindAddr := fmt.Sprintf("127.0.0.1:%d", port)

		// 將 cacert.pem 的路徑轉換為正斜線，避免 Windows 路徑反斜線被 PHP 轉義或解析錯誤
		cacertSlashPath := filepath.ToSlash(cacertPath)
		cmd := exec.Command(phpInfo.ExePath,
			"-c", phpIniPath,
			"-d", "extension_dir="+extDir,
			"-d", "openssl.cafile="+cacertSlashPath,
			"-d", "curl.cainfo="+cacertSlashPath,
			"-b", bindAddr,
		)
		cmd.Dir = m.baseDir
		cmd.Env = env
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}

		// PHP-CGI 的輸出通常不多，但仍擷取 stderr
		m.pipeOutput(cmd, "php", fmt.Sprintf("PHP-%s", phpInfo.MajorMin))

		if err := cmd.Start(); err != nil {
			// 有一個啟動失敗，終止已啟動的
			m.errorLog("php", i18n.Tfmt("PHP-CGI %s 在 Port %d 啟動失敗", phpInfo.Version, port), err)
			for _, c := range cmds {
				if c.Process != nil {
					_ = c.Process.Kill()
				}
			}
			return fmt.Errorf("%s: %w", i18n.Tfmt("PHP-CGI %s 啟動失敗", phpInfo.Version), err)
		}

		m.log("php", "  ✓ PID %d → %s", cmd.Process.Pid, bindAddr)
		cmds = append(cmds, cmd)
	}

	m.register(serviceKey, "PHP-CGI "+phpInfo.Version, phpInfo.ExePath, cmds)
	m.log("php", "%s", i18n.Tfmt("✅ PHP-CGI %s 已啟動 (%s)", phpInfo.Version, phpInfo.GetPortRangeStr()))

	// 監控每個行程的退出，單一程序退出時更新 PID 列表，所有退出則 unregister
	var exitedCount int32
	for i, cmd := range cmds {
		go func(c *exec.Cmd, port int) {
			err := c.Wait()
			remaining := atomic.AddInt32(&exitedCount, 1)
			total := int32(len(cmds))
			stillRunning := total - remaining
			if stillRunning > 0 {
				// 還有程序在運行：從 PID 列表移除已退出的，但保持服務運行
				if c.Process != nil {
					m.RemovePID(serviceKey, c.Process.Pid)
				}
				if err != nil {
					m.errorLog("php", i18n.Tfmt("PHP-CGI %s (Port %d) 異常退出，剩餘 %d 個程序", phpInfo.Version, port, stillRunning), err)
				} else {
					m.log("php", "%s", i18n.Tfmt("ℹ️ PHP-CGI %s (Port %d) 已退出，剩餘 %d 個程序", phpInfo.Version, port, stillRunning))
				}
			} else {
				// 所有程序都已退出
				if m.IsRunning(serviceKey) {
					if err != nil {
						m.errorLog("php", i18n.Tfmt("PHP-CGI %s 最後一個程序 (Port %d) 異常退出", phpInfo.Version, port), err)
					} else {
						m.log("php", "%s", i18n.Tfmt("ℹ️ PHP-CGI %s 所有程序已退出", phpInfo.Version))
					}
					m.unregister(serviceKey)
				}
			}
		}(cmd, ports[i])
	}

	return nil
}

// StopPHPCGI 停止指定版本的所有 PHP-CGI 行程
func (m *Manager) StopPHPCGI(version string) error {
	serviceKey := PHPServiceKey(version)

	if !m.IsRunning(serviceKey) {
		return fmt.Errorf("%s", i18n.Tfmt("PHP-CGI %s 未在運行", version))
	}

	m.log("php", "%s", i18n.Tfmt("🛑 停止 PHP-CGI %s...", version))
	if err := m.stopService(serviceKey); err != nil {
		m.errorLog("php", i18n.Tfmt("PHP-CGI %s 停止失敗", version), err)
		return err
	}
	m.log("php", "%s", i18n.Tfmt("✅ PHP-CGI %s 已停止", version))
	return nil
}

// downloadCacert 從指定網址下載 cacert.pem 證書並保存到本地
func downloadCacert(url, dest string) error {
	// 確保目錄存在
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("無法建立目錄: %w", err)
	}

	// 建立臨時檔案，下載完成後再重命名，避免下載中斷產生損壞的檔案
	tmpDest := dest + ".tmp"
	out, err := os.Create(tmpDest)
	if err != nil {
		return fmt.Errorf("無法建立臨時檔案: %w", err)
	}
	defer func() {
		out.Close()
		_ = os.Remove(tmpDest) // 如果成功，Rename 後此處 Remove 會回傳錯誤（可忽略）
	}()

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("下載請求失敗: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("伺服器回應錯誤狀態碼: %d", resp.StatusCode)
	}

	if _, err = io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("寫入檔案失敗: %w", err)
	}
	out.Close()

	if err := os.Rename(tmpDest, dest); err != nil {
		return fmt.Errorf("重新命名檔案失敗: %w", err)
	}

	return nil
}
