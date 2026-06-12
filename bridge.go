package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	mysql "github.com/go-sql-driver/mysql"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"wincmp/internal/config"
	"wincmp/internal/detect"
	"wincmp/internal/hosts"
	"wincmp/internal/i18n"
	"wincmp/internal/preset"
	"wincmp/internal/process"
	"wincmp/internal/resource"
	"wincmp/internal/scanner"
	"wincmp/internal/singleinstance"
	"wincmp/internal/updater"
)

// ==========================================
// 1. 設定與服務掃描 API
// ==========================================

// GetDetailedResources 獲取詳細的 CPU & RAM 監控數據
func (a *App) GetDetailedResources() (resource.DetailedResources, error) {
	if a.resMonitor == nil {
		return resource.DetailedResources{}, fmt.Errorf("%s", i18n.T("資源監控器未初始化"))
	}
	return a.resMonitor.GetDetailedResourceUsage()
}

// CheckPortConflicts 檢查核心服務端口是否被佔用
func (a *App) CheckPortConflicts() (map[string]bool, error) {
	var ports []int

	// MariaDB Port
	dbPort := a.appCfg.Global.MariaDBPort
	if dbPort <= 0 {
		dbPort = 3306
	}
	ports = append(ports, dbPort)

	// Mailpit Ports
	smtpPort := a.appCfg.Global.MailpitSMTPPort
	if smtpPort <= 0 {
		smtpPort = 1025
	}
	httpPort := a.appCfg.Global.MailpitHTTPPort
	if httpPort <= 0 {
		httpPort = 8025
	}
	ports = append(ports, smtpPort, httpPort)

	conflicts := make(map[string]bool)
	for _, p := range ports {
		isRunning := false
		if p == dbPort {
			isRunning = a.IsMariaDBRunning()
		} else if p == smtpPort || p == httpPort {
			isRunning = a.procMgr.IsRunning("mailpit")
		}

		// 只有當服務未運行且端口不可用時，才視為衝突
		if !isRunning && !process.IsPortAvailable(p) {
			conflicts[strconv.Itoa(p)] = true
		} else {
			conflicts[strconv.Itoa(p)] = false
		}
	}

	return conflicts, nil
}

// GetConfig 獲取當前記憶體中的全域設定檔
func (a *App) GetConfig() *config.WincmpConfig {
	return a.appCfg
}

// SaveConfig 將新的設定檔寫入記憶體並持久化保存到 conf/wincmp.json
func (a *App) SaveConfig(newCfg *config.WincmpConfig) error {
	if newCfg == nil {
		return fmt.Errorf("%s", i18n.T("設定檔為空，無法儲存"))
	}

	// 後端安全驗證：檢查專案名稱與網域格式是否合規
	projectNamePattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	for _, proj := range newCfg.Projects {
		name := strings.TrimSpace(proj.Name)
		if name == "" {
			return fmt.Errorf("%s", i18n.T("專案名稱不能為空"))
		}
		if !projectNamePattern.MatchString(name) {
			return fmt.Errorf("%s", i18n.Tfmt("專案名稱 '%s' 格式不正確。僅能包含英數字、連字號(-)與底線(_)。", name))
		}
		for _, dom := range proj.Domains {
			domTrimmed := strings.TrimSpace(dom)
			if domTrimmed == "" {
				continue
			}
			if !hosts.IsValidDomain(domTrimmed) {
				return fmt.Errorf("%s", i18n.Tfmt("網域 '%s' 格式不正確。僅能包含英數字、連字號(-)、底線(_)與點(.)，且不能包含埠號或路徑。", domTrimmed))
			}
		}
	}

	a.appCfg = newCfg
	cfgPath := filepath.Join(a.baseDir, "conf", "wincmp.json")
	
	if err := a.appCfg.Save(cfgPath); err != nil {
		return fmt.Errorf(i18n.T("無法儲存設定檔: %w"), err)
	}

	// 更新後端語系並同步刷新系統托盤選單
	i18n.SetLanguage(a.appCfg.Global.Language)
	a.updateTrayMenu()

	return nil
}

// SaveQuickSettings 快速儲存主題、語言與字體大小設定，避免與其他設定頁草稿衝突
func (a *App) SaveQuickSettings(theme string, language string, fontSize string) error {
	a.appCfg.Global.Theme = theme
	a.appCfg.Global.Language = language
	a.appCfg.Global.FontSize = fontSize
	cfgPath := filepath.Join(a.baseDir, "conf", "wincmp.json")

	if err := a.appCfg.Save(cfgPath); err != nil {
		return fmt.Errorf(i18n.T("無法儲存設定檔: %w"), err)
	}

	// 更新後端語系並同步刷新系統托盤選單
	i18n.SetLanguage(language)
	a.updateTrayMenu()

	return nil
}


// ScanServices 重新掃描 bin/ 目錄並更新二進位服務版本資訊
func (a *App) ScanServices() (*scanner.ScanResult, error) {
	res, err := scanner.ScanBinDir(a.baseDir)
	if err != nil {
		return nil, fmt.Errorf(i18n.T("服務掃描失敗: %w"), err)
	}
	a.scanRes = res
	return a.scanRes, nil
}

// GetScanResult 獲取當前已快取的服務掃描結果
func (a *App) GetScanResult() *scanner.ScanResult {
	if a.scanRes == nil {
		res, err := scanner.ScanBinDir(a.baseDir)
		if err == nil {
			a.scanRes = res
		}
	}
	return a.scanRes
}

// IsServiceRunning 檢查特定服務是否正在運行
func (a *App) IsServiceRunning(serviceKey string) bool {
	if a.procMgr == nil {
		return false
	}
	return a.procMgr.IsRunning(serviceKey)
}

// GetServicesStatus 獲取所有已偵測服務的運行狀態對照表
func (a *App) GetServicesStatus() map[string]bool {
	status := make(map[string]bool)
	if a.procMgr == nil || a.scanRes == nil {
		return status
	}

	// Caddy
	status["caddy"] = a.procMgr.IsRunning("caddy")

	// Mailpit
	status["mailpit"] = a.procMgr.IsRunning(process.MailpitServiceKey())

	// MariaDB
	for _, m := range a.scanRes.MariaDBList {
		key := process.MariaDBServiceKey(m.Version)
		status[key] = a.procMgr.IsRunning(key)
	}

	// PHP
	for _, p := range a.scanRes.PHPList {
		key := process.PHPServiceKey(p.Version)
		status[key] = a.procMgr.IsRunning(key)
	}

	// 專案 Runtime 狀態
	for _, proj := range a.appCfg.Projects {
		if proj.Enabled && preset.IsRuntimeProject(proj.Type) {
			key := process.RuntimeServiceKey(proj.Name)
			status[key] = a.procMgr.IsRunning(key)
		}
	}

	return status
}

// ==========================================
// 2. 服務控制 API (Caddy, MariaDB, PHP, Mailpit)
// ==========================================

// StartCaddy 啟動 Caddy 反向代理服務
func (a *App) StartCaddy(version string, exePath string) error {
	if a.procMgr == nil {
		return fmt.Errorf("%s", i18n.T("進程管理器未初始化"))
	}

	// 1. 檢查並更新憑證
	a.checkSSLCerts()

	// 2. 生成 Caddy 配置文件
	if err := a.generateCaddyfiles(); err != nil {
		return fmt.Errorf(i18n.T("產生 Caddy 配置文件失敗: %w"), err)
	}

	// 3. 啟動進程
	if err := a.procMgr.StartCaddy(version, exePath); err != nil {
		return fmt.Errorf(i18n.T("啟動 Caddy 失敗: %w"), err)
	}

	// 4. 自動更新 Hosts 檔案
	a.triggerHostsUpdate()

	a.saveLastServiceState()

	return nil
}

// StopCaddy 停止 Caddy 服務
func (a *App) StopCaddy() error {
	if a.procMgr == nil {
		return fmt.Errorf("%s", i18n.T("進程管理器未初始化"))
	}
	a.procMgr.StopCaddy()
	a.saveLastServiceState()
	return nil
}

// ReloadCaddy 重新載入 Caddy 設定檔，並觸發 Hosts 檔案檢查
func (a *App) ReloadCaddy() error {
	if a.procMgr == nil {
		return fmt.Errorf("%s", i18n.T("進程管理器未初始化"))
	}
	if err := a.regenerateCaddyAndReload(); err != nil {
		return fmt.Errorf(i18n.T("重新載入 Caddy 失敗: %w"), err)
	}
	return nil
}

// StartMariaDB 啟動 MariaDB 資料庫服務 (同步等待啟動完成)
func (a *App) StartMariaDB(version string) error {
	if a.procMgr == nil {
		return fmt.Errorf("%s", i18n.T("進程管理器未初始化"))
	}

	// 異步啟動
	done, errCh := a.procMgr.StartMariaDBAsync(
		version,
		a.appCfg.Global.MariaDBExternal,
		a.appCfg.Global.MariaDBBasedir,
		a.appCfg.Global.MariaDBDatadir,
		a.appCfg.Global.MariaDBType,
		a.appCfg.Global.MariaDBPort,
	)

	// 等待 Go routine 完成並回傳錯誤狀態
	<-done
	if err := <-errCh; err != nil {
		return fmt.Errorf(i18n.T("啟動 MariaDB 失敗: %w"), err)
	}

	a.saveLastServiceState()

	return nil
}

// StopMariaDB 停止 MariaDB 服務
func (a *App) StopMariaDB(version string) error {
	if a.procMgr == nil {
		return fmt.Errorf("%s", i18n.T("進程管理器未初始化"))
	}

	portVal := a.appCfg.Global.MariaDBPort
	if portVal <= 0 {
		portVal = 3306
	}

	err := a.procMgr.StopMariaDB(
		version,
		a.appCfg.Global.MariaDBExternal,
		a.appCfg.Global.MariaDBBasedir,
		a.appCfg.Global.MariaDBType,
		portVal,
	)
	if err != nil {
		return fmt.Errorf(i18n.T("停止 MariaDB 失敗: %w"), err)
	}

	a.saveLastServiceState()

	return nil
}

// StartMailpit 啟動 Mailpit 郵件測試服務
func (a *App) StartMailpit(version string, exePath string, smtpPort int, httpPort int, useDB bool) error {
	if a.procMgr == nil {
		return fmt.Errorf("%s", i18n.T("進程管理器未初始化"))
	}
	if err := a.procMgr.StartMailpit(version, exePath, smtpPort, httpPort, useDB); err != nil {
		return fmt.Errorf(i18n.T("啟動 Mailpit 失敗: %w"), err)
	}

	a.saveLastServiceState()

	return nil
}

// StopMailpit 停止 Mailpit 服務
func (a *App) StopMailpit() error {
	if a.procMgr == nil {
		return fmt.Errorf("%s", i18n.T("進程管理器未初始化"))
	}
	a.procMgr.StopMailpit()
	a.saveLastServiceState()
	return nil
}

// StartPHP 啟動特定版本的 PHP-CGI 多行程服務
func (a *App) StartPHP(version string) error {
	if a.procMgr == nil {
		return fmt.Errorf("%s", i18n.T("進程管理器未初始化"))
	}

	var targetPHP *scanner.PHPVersionInfo
	for i := range a.scanRes.PHPList {
		if a.scanRes.PHPList[i].Version == version {
			targetPHP = &a.scanRes.PHPList[i]
			break
		}
	}

	if targetPHP == nil {
		return fmt.Errorf(i18n.T("找不到指定版本 %s 的 PHP-CGI"), version)
	}

	// 套用進程數配置
	count := a.appCfg.Global.PHP.ProcessesPerVersion
	if c, ok := a.appCfg.Global.PHP.Processes[targetPHP.MajorMin]; ok {
		count = c
	}
	targetPHP.PortCount = count

	if err := a.procMgr.StartPHPCGI(*targetPHP); err != nil {
		return fmt.Errorf(i18n.T("啟動 PHP-CGI %s 失敗: %w"), version, err)
	}

	a.saveLastServiceState()

	return nil
}

// StopPHP 停止特定版本的 PHP-CGI 服務
func (a *App) StopPHP(version string) error {
	if a.procMgr == nil {
		return fmt.Errorf("%s", i18n.T("進程管理器未初始化"))
	}
	a.procMgr.StopPHPCGI(version)
	a.saveLastServiceState()
	return nil
}

// ==========================================
// 3. 專案 Runtime 啟停 API (Node/Bun/Python/Go)
// ==========================================

// StartProjectRuntime 啟動特定專案的背景/終端 Runtime
func (a *App) StartProjectRuntime(projectName string) error {
	if a.procMgr == nil {
		return fmt.Errorf("%s", i18n.T("進程管理器未初始化"))
	}

	var proj *config.ProjectConfig
	for i := range a.appCfg.Projects {
		if a.appCfg.Projects[i].Name == projectName {
			proj = &a.appCfg.Projects[i]
			break
		}
	}
	if proj == nil {
		return fmt.Errorf(i18n.T("找不到專案 %s"), projectName)
	}

	// 1. 檢查端口佔用
	portVal := proj.RuntimePort
	if portVal == 0 {
		portVal = 3000
	}
	if !process.IsPortAvailable(portVal) {
		return fmt.Errorf(i18n.T("端口 %d 已被其他進程佔用"), portVal)
	}

	// 2. 推導執行路徑 (Bundled Runtime 優先，否則尋找系統 PATH)
	exePath := ""
	resolvedRT := proj.RuntimeType
	hasNode := len(a.scanRes.NodeList) > 0
	hasBun := len(a.scanRes.BunList) > 0
	if resolvedRT == "auto" {
		if proj.UseWinCMPBin {
			if hasNode {
				resolvedRT = "node"
			} else if hasBun {
				resolvedRT = "bun"
			} else {
				resolvedRT = "node"
			}
		} else {
			if _, err := exec.LookPath("node"); err == nil {
				resolvedRT = "node"
			} else if _, err := exec.LookPath("bun"); err == nil {
				resolvedRT = "bun"
			} else {
				resolvedRT = "node"
			}
		}
	}

	if proj.UseWinCMPBin {
		if resolvedRT == "node" {
			for _, n := range a.scanRes.NodeList {
				if n.Version == proj.RuntimeVersion {
					exePath = n.ExePath
					break
				}
			}
		} else if resolvedRT == "bun" {
			for _, b := range a.scanRes.BunList {
				if b.Version == proj.RuntimeVersion {
					exePath = b.ExePath
					break
				}
			}
		}
	} else {
		// 使用系統環境變數中的執行檔
		switch resolvedRT {
		case "node":
			exePath = "npm"
		case "bun":
			exePath = "bun"
		case "python":
			exePath = "python"
		case "go_air":
			exePath = "air"
		case "go_run":
			exePath = "go"
		}
	}

	if exePath == "" && resolvedRT != "custom" {
		return fmt.Errorf(i18n.T("找不到可用的 %s 執行器，請檢查設定與 PATH"), resolvedRT)
	}

	// 3. 呼叫底層 process 啟動
	if err := a.procMgr.StartRuntime(*proj, proj.RuntimeMode, exePath); err != nil {
		return fmt.Errorf(i18n.T("啟動專案 Runtime 失敗: %w"), err)
	}

	return nil
}

// StopProjectRuntime 停止特定專案的 Runtime 服務
func (a *App) StopProjectRuntime(projectName string) error {
	if a.procMgr == nil {
		return fmt.Errorf("%s", i18n.T("進程管理器未初始化"))
	}

	var proj *config.ProjectConfig
	for i := range a.appCfg.Projects {
		if a.appCfg.Projects[i].Name == projectName {
			proj = &a.appCfg.Projects[i]
			break
		}
	}
	if proj == nil {
		return fmt.Errorf(i18n.T("找不到專案 %s"), projectName)
	}

	a.procMgr.StopRuntime(*proj)
	return nil
}

// ==========================================
// 4. 內建輔助方法 (Caddyfile 產生器與 SSL 檢查)
// ==========================================

// checkSSLCerts 檢查所有專案中的 SSL 憑證是否存在
func (a *App) checkSSLCerts() {
	for _, proj := range a.appCfg.Projects {
		if !proj.Enabled || !proj.UseSSL {
			continue
		}
		crt := a.appCfg.GetSSLCertPath(proj, a.baseDir)
		key := a.appCfg.GetSSLKeyPath(proj, a.baseDir)
		if crt != "" && key != "" {
			if _, err := os.Stat(crt); os.IsNotExist(err) {
				fmt.Printf("⚠️ 專案 %s: 憑證遺失，將使用自動 TLS\n", proj.Name)
			} else if _, err := os.Stat(key); os.IsNotExist(err) {
				fmt.Printf("⚠️ 專案 %s: 金鑰遺失，將使用自動 TLS\n", proj.Name)
			}
		}
	}
}

// generatePHPUpstream 產生 PHP 負載平衡設定檔
func (a *App) generatePHPUpstream() error {
	snippetDir := filepath.Join(a.baseDir, "conf", "snippets")
	os.MkdirAll(snippetDir, 0700)
	upstreamPath := filepath.Join(snippetDir, "php-upstream.caddy")

	var content strings.Builder
	for _, info := range a.scanRes.PHPList {
		count := a.appCfg.Global.PHP.ProcessesPerVersion
		if c, ok := a.appCfg.Global.PHP.Processes[info.MajorMin]; ok {
			count = c
		}
		info.PortCount = count

		phpID := strings.ReplaceAll(info.MajorMin, ".", "")
		content.WriteString(fmt.Sprintf("(php%s) {\n", phpID))
		content.WriteString("\tphp_fastcgi")
		ports := info.GetPHPPorts()
		for _, port := range ports {
			content.WriteString(fmt.Sprintf(" 127.0.0.1:%d", port))
		}
		content.WriteString("\n}\n")
	}

	return os.WriteFile(upstreamPath, []byte(content.String()), 0600)
}

// generateCaddyfiles 產生所有子專案的 .caddy 設定檔
func (a *App) generateCaddyfiles() error {
	if err := a.generatePHPUpstream(); err != nil {
		return err
	}

	sitesDir := filepath.Join(a.baseDir, "conf", "sites")
	os.MkdirAll(sitesDir, 0700)

	// 清除舊的 .caddy 檔
	if oldFiles, err := filepath.Glob(filepath.Join(sitesDir, "*.caddy")); err == nil {
		for _, f := range oldFiles {
			os.Remove(f)
		}
	}

	for _, proj := range a.appCfg.Projects {
		if !proj.Enabled {
			continue
		}

		caddyPath := filepath.Join(sitesDir, proj.Name+".caddy")
		content, err := a.buildCaddyfileContent(proj)
		if err != nil {
			return err
		}

		if err := os.WriteFile(caddyPath, []byte(content), 0600); err != nil {
			return fmt.Errorf(i18n.T("寫入 Caddy 設定檔 %s 失敗: %w"), caddyPath, err)
		}
	}
	return nil
}

// validateCaddyPath 驗證路徑安全性
func (a *App) validateCaddyPath(path string) (string, error) {
	cleaned := filepath.Clean(path)
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf(i18n.T("路徑含非法的目錄遍歷: %s"), path)
	}
	return strings.ReplaceAll(cleaned, "\\", "/"), nil
}

// regenerateCaddyAndReload 重新產生 Caddy 設定檔，若 Caddy 運行中則自動 Reload 並更新 Hosts
func (a *App) regenerateCaddyAndReload() error {
	if err := a.generateCaddyfiles(); err != nil {
		return err
	}
	if a.procMgr.IsRunning("caddy") {
		exePath := a.procMgr.GetExePath("caddy")
		if err := a.procMgr.ReloadCaddy(exePath); err != nil {
			fmt.Printf("❌ Reload Caddy 失敗: %v\n", err)
		} else {
			fmt.Println("✅ Caddy 設定已重新載入")
		}
	}
	a.triggerHostsUpdate()
	return nil
}

// triggerHostsUpdate 檢查並更新系統 hosts 檔
func (a *App) triggerHostsUpdate() {
	if !a.appCfg.Global.AutoUpdateHosts {
		return
	}

	var allDomains []string
	for _, proj := range a.appCfg.Projects {
		allDomains = append(allDomains, proj.Domains...)
	}

	if len(allDomains) == 0 {
		return
	}

	missing, err := hosts.CheckHosts(allDomains)
	if err != nil {
		a.handleErrorLog("system", i18n.T("檢查 Hosts 失敗"), err)
		return
	}

	if len(missing) == 0 {
		return
	}

	a.handleLog("system", i18n.Tfmt("🔍 偵測到 %d 個網域不在系統 Hosts 中: %s", len(missing), strings.Join(missing, ", ")))

	var invalidDomains []string
	var validMissing []string
	for _, d := range missing {
		if hosts.IsValidDomain(d) {
			validMissing = append(validMissing, d)
		} else {
			invalidDomains = append(invalidDomains, d)
		}
	}

	if len(invalidDomains) > 0 {
		a.handleLog("system", i18n.Tfmt("⚠️ 以下網域含非法字元，已跳過: %v", invalidDomains))
	}

	if len(validMissing) == 0 {
		return
	}

	// 備份 Hosts
	backupPath, err := hosts.BackupHosts(a.baseDir)
	if err != nil {
		a.handleErrorLog("system", i18n.T("備份 Hosts 失敗 (將停止更新)"), err)
		return
	}
	a.handleLog("system", i18n.Tfmt("✅ 已備份現有 Hosts 到: %s", backupPath))

	// 更新 Hosts
	err = hosts.UpdateHosts(validMissing)
	if err != nil {
		errMsg := i18n.T("更新系統 Hosts 失敗 (可能需要管理員權限)")
		a.handleErrorLog("system", errMsg, err)
		
		// 彈出 Wails 對話框提示用戶
		go func() {
			_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
				Type:          runtime.WarningDialog,
				Title:         i18n.T("Hosts 更新失敗"),
				Message:       i18n.T("無法寫入 Hosts 檔案。這通常是因為權限不足。\n請嘗試以「系統管理員身分」執行 WinCMP，或者手動將網域新增至 Hosts 檔案中。") + "\n\n" + fmt.Sprintf("Error: %v", err),
				Buttons:       []string{i18n.T("確定")},
			})
		}()
		return
	}

	a.handleLog("system", i18n.Tfmt("🚀 已成功將 %d 個網域寫入系統 Hosts 檔", len(validMissing)))
}

// OpenFolder 用系統預設檔案瀏覽器開啟指定的本機資料夾
func (a *App) OpenFolder(path string) error {
	cmd := exec.Command("explorer", filepath.Clean(path))
	return cmd.Start()
}

// SelectFolder 彈出 Wails 原生的目錄選擇對話框，並回傳選擇的路徑
func (a *App) SelectFolder() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("%s", i18n.T("應用程式 Context 未初始化"))
	}
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: i18n.T("選擇專案目錄"),
	})
	if err != nil {
		return "", err
	}
	return path, nil
}

// ==========================================
// 5. 資料庫 Explorer API
// ==========================================

// getDBPool 取得或建立 MariaDB 連線池
func (a *App) getDBPool() (*sql.DB, error) {
	portVal := a.appCfg.Global.MariaDBPort
	if portVal <= 0 {
		portVal = 3306
	}
	user := a.appCfg.Global.MariaDBUser
	if user == "" {
		user = "root"
	}
	password := a.appCfg.Global.MariaDBPassword
	cfg := mysql.NewConfig()
	cfg.User = user
	cfg.Passwd = password
	cfg.Net = "tcp"
	cfg.Addr = fmt.Sprintf("127.0.0.1:%d", portVal)
	cfg.Timeout = 5 * time.Second
	dsn := cfg.FormatDSN()

	a.dbPoolMu.Lock()
	defer a.dbPoolMu.Unlock()

	if a.dbPool != nil && a.dbPoolDSN == dsn {
		return a.dbPool, nil
	}

	if a.dbPool != nil {
		a.dbPool.Close()
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(3)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(30 * time.Second)
	db.SetConnMaxIdleTime(10 * time.Second)

	a.dbPool = db
	a.dbPoolDSN = dsn
	return a.dbPool, nil
}

// IsMariaDBRunning 檢查本機 MariaDB 服務是否運行中
func (a *App) IsMariaDBRunning() bool {
	if a.appCfg.Global.MariaDBExternal {
		return a.procMgr.IsRunning(process.MariaDBExternalServiceKey)
	}
	for _, info := range a.scanRes.MariaDBList {
		if a.procMgr.IsRunning(process.MariaDBServiceKey(info.Version)) {
			return true
		}
	}
	return false
}

// QueryDatabases 查詢本機資料庫所有的 Schema 列表
func (a *App) QueryDatabases() ([]string, error) {
	db, err := a.getDBPool()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query("SHOW DATABASES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var databases = []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			databases = append(databases, name)
		}
	}
	return databases, rows.Err()
}

// QueryTables 查詢指定 Schema 的所有資料表與行數資訊
func (a *App) QueryTables(schema string) ([]string, error) {
	portVal := a.appCfg.Global.MariaDBPort
	if portVal <= 0 {
		portVal = 3306
	}
	user := a.appCfg.Global.MariaDBUser
	if user == "" {
		user = "root"
	}
	password := a.appCfg.Global.MariaDBPassword
	cfg := mysql.NewConfig()
	cfg.User = user
	cfg.Passwd = password
	cfg.Net = "tcp"
	cfg.Addr = fmt.Sprintf("127.0.0.1:%d", portVal)
	cfg.DBName = schema
	cfg.Timeout = 5 * time.Second
	dsn := cfg.FormatDSN()

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	db.SetConnMaxLifetime(5 * time.Second)
	db.SetMaxOpenConns(1)

	rows, err := db.Query(
		"SELECT TABLE_NAME, TABLE_ROWS FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? ORDER BY TABLE_NAME",
		schema,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables = []string{}
	for rows.Next() {
		var tableName string
		var tableRows sql.NullInt64
		if err := rows.Scan(&tableName, &tableRows); err == nil {
			if tableRows.Valid {
				tables = append(tables, fmt.Sprintf("%-40s  (%d rows)", tableName, tableRows.Int64))
			} else {
				tables = append(tables, tableName)
			}
		}
	}
	return tables, rows.Err()
}

// OpenInHeidiSQL 使用 HeidiSQL 圖形介面軟體開啟 MariaDB
func (a *App) OpenInHeidiSQL() error {
	if len(a.scanRes.HeidiSQLList) == 0 {
		return fmt.Errorf("%s", i18n.T("找不到已安裝的 HeidiSQL 執行檔，請確認 HeidiSQL 是否在 bin/ 中"))
	}

	heidiPath := a.scanRes.HeidiSQLList[0].ExePath
	portVal := a.appCfg.Global.MariaDBPort
	if portVal <= 0 {
		portVal = 3306
	}
	user := a.appCfg.Global.MariaDBUser
	if user == "" {
		user = "root"
	}

	cmd := exec.Command(heidiPath, "-h=127.0.0.1", fmt.Sprintf("-P=%d", portVal), fmt.Sprintf("-u=%s", user))
	if err := cmd.Start(); err != nil {
		return fmt.Errorf(i18n.T("啟動 HeidiSQL 失敗: %w"), err)
	}

	go cmd.Wait()
	return nil
}

// ==========================================
// 6. 專案路徑與框架自動偵測 API
// ==========================================

// ProjectDetectResult 專案路徑偵測結果
type ProjectDetectResult struct {
	Name        string   `json:"name"`
	Domains     []string `json:"domains"`
	Type        string   `json:"type"`
	RuntimeType string   `json:"runtime_type"`
	RuntimePort int      `json:"runtime_port"`
	PHPVersion  string   `json:"php_version"`
}

// laravelPHPMapping 結構體用於解析 Laravel 與 PHP 版本的對應
type laravelPHPMapping struct {
	Mappings []struct {
		Laravel string `json:"laravel"`
		PHP     string `json:"php"`
	} `json:"mappings"`
	Fallback string `json:"fallback"`
}

// DetectProjectPath 偵測專案物理目錄的資訊，包含自動生成的名稱、網域別名、專案類型與執行環境等
func (a *App) DetectProjectPath(path string) (*ProjectDetectResult, error) {
	if path == "" {
		return nil, fmt.Errorf("%s", i18n.T("路徑不能為空"))
	}

	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf(i18n.T("路徑無效或不存在: %w"), err)
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("%s", i18n.T("選擇的路徑不是一個有效的資料夾"))
	}

	folderName := filepath.Base(path)

	// 1. 生成專案名稱：底線與空格轉橫線，並進行檔名/安全字元清洗
	sanitizedName := strings.ReplaceAll(folderName, "_", "-")
	sanitizedName = config.SanitizeProjectName(sanitizedName)

	// 2. 生成網域：僅保留英數字與橫線，前後加 local- 與 .test
	domainName := strings.ToLower(folderName)
	domainName = strings.ReplaceAll(domainName, "_", "-")
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	domainName = reg.ReplaceAllString(domainName, "-")
	domainName = strings.Trim(domainName, "-")
	if domainName == "" {
		domainName = "project"
	}
	recommendedDomain := fmt.Sprintf("local-%s.test", domainName)

	// 3. 偵測 Preset 專案類型與執行環境
	detRes := preset.DetectProjectPreset(path)
	projectType := detRes.Type
	runtimeType := detRes.Runtime
	runtimePort := detRes.Port
	phpVersion := ""

	// 4. Laravel PHP 專屬匹配
	if projectType == preset.TypeLaravel {
		laravelRes := detect.DetectLaravel(path)
		mapping, fallback := a.loadLaravelPHPMapping()
		phpVersion = a.getRecommendedPHPVersion(laravelRes.Version, mapping, fallback)
	}

	return &ProjectDetectResult{
		Name:        sanitizedName,
		Domains:     []string{recommendedDomain},
		Type:        projectType,
		RuntimeType: runtimeType,
		RuntimePort: runtimePort,
		PHPVersion:  phpVersion,
	}, nil
}

// loadLaravelPHPMapping 載入 Laravel PHP 版本推薦映射設定
func (a *App) loadLaravelPHPMapping() (map[string]string, string) {
	mappingFile := filepath.Join(a.baseDir, "conf", "php", "laravel-php-mapping.json")
	data, err := os.ReadFile(mappingFile)
	if err != nil {
		return nil, "8.2"
	}

	var m laravelPHPMapping
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, "8.2"
	}

	result := make(map[string]string)
	for _, item := range m.Mappings {
		result[item.Laravel] = item.PHP
	}
	return result, m.Fallback
}

// getRecommendedPHPVersion 依據 Laravel 版本取得推薦的 PHP 版本，並驗證是否已在本機安裝
func (a *App) getRecommendedPHPVersion(laravelVersion string, mapping map[string]string, fallback string) string {
	if laravelVersion == "" {
		return a.validatePHPMajorVersion(fallback)
	}

	if php, ok := mapping[laravelVersion]; ok {
		validated := a.validatePHPMajorVersion(php)
		if validated != "" {
			return validated
		}
	}

	if strings.HasPrefix(laravelVersion, "<") {
		if php, ok := mapping["<"+a.laravelMajorStr(laravelVersion)]; ok {
			validated := a.validatePHPMajorVersion(php)
			if validated != "" {
				return validated
			}
		}
		return a.validatePHPMajorVersion(fallback)
	}

	if strings.HasPrefix(laravelVersion, ">=") || strings.HasPrefix(laravelVersion, ">") {
		if php, ok := mapping[">="+a.laravelMajorStr(laravelVersion)]; ok {
			validated := a.validatePHPMajorVersion(php)
			if validated != "" {
				return validated
			}
		}
	}

	laravelMajor := a.parseLaravelMajor(laravelVersion)
	if laravelMajor == 0 {
		return a.validatePHPMajorVersion(fallback)
	}

	for laravelMajor > 0 {
		key := strconv.Itoa(laravelMajor) + ".x"
		if php, ok := mapping[key]; ok {
			validated := a.validatePHPMajorVersion(php)
			if validated != "" {
				return validated
			}
		}
		laravelMajor--
	}

	return a.validatePHPMajorVersion(fallback)
}

func (a *App) laravelMajorStr(version string) string {
	version = strings.TrimPrefix(version, "<")
	version = strings.TrimPrefix(version, ">=")
	version = strings.TrimPrefix(version, ">")
	return strings.TrimSuffix(version, ".x")
}

func (a *App) parseLaravelMajor(version string) int {
	majorStr := a.laravelMajorStr(version)
	if majorStr == "" {
		return 0
	}
	major, err := strconv.Atoi(majorStr)
	if err != nil {
		return 0
	}
	return major
}

func (a *App) validatePHPMajorVersion(majorVersion string) string {
	if a.scanRes == nil {
		return ""
	}
	for _, info := range a.scanRes.PHPList {
		if info.MajorMin == majorVersion {
			return majorVersion
		}
	}
	return ""
}

// StartTerminalSession 啟動一個新終端會話，並回傳會話 ID
func (a *App) StartTerminalSession(projName string, cols int, rows int) (string, error) {
	if a.termMgr == nil {
		return "", fmt.Errorf("%s", i18n.T("終端管理器未初始化"))
	}

	var proj *config.ProjectConfig
	for i := range a.appCfg.Projects {
		if a.appCfg.Projects[i].Name == projName {
			proj = &a.appCfg.Projects[i]
			break
		}
	}

	var cwd string
	if proj != nil {
		cwd = a.appCfg.GetProjectPhysicalRoot(*proj, a.baseDir)
	} else {
		cwd = filepath.Join(a.baseDir, "www")
	}

	shellPath := a.appCfg.Global.TerminalShell
	if shellPath == "" {
		shellPath = "powershell.exe"
	}

	var sessionID string
	var sessionIDMu sync.RWMutex

	realOnOutput := func(data string) {
		if a.ctx != nil {
			sessionIDMu.RLock()
			sID := sessionID
			sessionIDMu.RUnlock()
			runtime.EventsEmit(a.ctx, "terminal_output", map[string]string{
				"sessionId": sID,
				"data":      data,
			})
		}
	}

	realOnExit := func() {
		if a.ctx != nil {
			sessionIDMu.RLock()
			sID := sessionID
			sessionIDMu.RUnlock()
			runtime.EventsEmit(a.ctx, "terminal_exit", map[string]string{
				"sessionId": sID,
			})
		}
	}

	sID, err := a.termMgr.StartTerminal(projName, shellPath, cwd, cols, rows, realOnOutput, realOnExit)
	if err != nil {
		return "", err
	}

	sessionIDMu.Lock()
	sessionID = sID
	sessionIDMu.Unlock()

	return sID, nil
}

// SendTerminalInput 傳送輸入字元或指令到終端會話
func (a *App) SendTerminalInput(sessionID string, data string) error {
	if a.termMgr == nil {
		return fmt.Errorf("%s", i18n.T("終端管理器未初始化"))
	}
	return a.termMgr.Write(sessionID, data)
}

// ResizeTerminal 調整指定終端會話的視窗大小
func (a *App) ResizeTerminal(sessionID string, cols int, rows int) error {
	if a.termMgr == nil {
		return fmt.Errorf("%s", i18n.T("終端管理器未初始化"))
	}
	return a.termMgr.Resize(sessionID, cols, rows)
}

// StopTerminalSession 主動停止並銷毀指定終端會話
func (a *App) StopTerminalSession(sessionID string) {
	if a.termMgr != nil {
		a.termMgr.Stop(sessionID)
	}
}

// RestartApp 儲存目前服務狀態，安全釋放鎖，並重啟 WinCMP
func (a *App) RestartApp() error {
	a.quittingMu.Lock()
	a.quitting = true
	a.quittingMu.Unlock()

	a.handleLog("system", i18n.T("正在準備重啟 WinCMP..."))

	// 提前釋放單實例鎖，讓新啟動的行程順利取得鎖
	singleinstance.Release()

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf(i18n.T("無法取得執行檔路徑: %w"), err)
	}

	cmd := exec.Command(execPath, "--restart")
	cmd.Dir = a.baseDir
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x01000000, // CREATE_BREAKAWAY_FROM_JOB
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf(i18n.T("自動重啟失敗: %w"), err)
	}

	runtime.Quit(a.ctx)
	return nil
}

// ─── 系統版本與權限 API ──────────────────────────────

var (
	shell32           = syscall.NewLazyDLL("shell32.dll")
	procIsUserAnAdmin = shell32.NewProc("IsUserAnAdmin")
)

// GetAppVersion 獲取當前應用程式版本號
func (a *App) GetAppVersion() string {
	return AppVersion
}

// IsAdmin 檢查當前進程是否以系統管理員權限啟動
func (a *App) IsAdmin() bool {
	ret, _, _ := procIsUserAnAdmin.Call()
	return ret != 0
}

// buildCaddyfileContent 根據專案配置生成 Caddyfile 的內容字串
func (a *App) buildCaddyfileContent(proj config.ProjectConfig) (string, error) {
	var domainsStr string
	if len(proj.Domains) > 0 {
		domainsStr = strings.Join(proj.Domains, ", ")
	} else {
		domainsStr = "local-" + proj.Name + ".test"
	}

	content := fmt.Sprintf("%s {\n", domainsStr)

	// SSL 設定
	if proj.UseSSL {
		crt := a.appCfg.GetSSLCertPath(proj, a.baseDir)
		key := a.appCfg.GetSSLKeyPath(proj, a.baseDir)
		certExists := crt != "" && key != ""
		var safeCrt, safeKey string
		if certExists {
			var crtErr, keyErr error
			safeCrt, crtErr = a.validateCaddyPath(crt)
			safeKey, keyErr = a.validateCaddyPath(key)
			if crtErr != nil || keyErr != nil {
				certExists = false
			} else {
				if _, err := os.Stat(crt); os.IsNotExist(err) {
					certExists = false
				} else if _, err := os.Stat(key); os.IsNotExist(err) {
					certExists = false
				}
			}
		}
		if certExists {
			content += fmt.Sprintf("\ttls %s %s\n", safeCrt, safeKey)
		} else {
			content += "\ttls internal\n"
		}
	}

	content += "\timport common_dev\n"

	if preset.IsRuntimeProject(proj.Type) {
		port := proj.RuntimePort
		if port == 0 {
			port = 3000
		}
		content += fmt.Sprintf("\treverse_proxy localhost:%d\n", port)
	} else {
		root := a.appCfg.GetProjectRoot(proj, a.baseDir)
		root = strings.ReplaceAll(root, "\\", "/")
		content += fmt.Sprintf("\troot * %s\n", root)

		if proj.PHPVersion != "" {
			phpVerStr := strings.ReplaceAll(proj.PHPVersion, ".", "")
			content += fmt.Sprintf("\timport php%s\n", phpVerStr)
		}

		content += "\timport static_site\n"
	}
	content += "}\n"
	return content, nil
}

// OpenProjectCaddyfile 檢查並開啟指定專案的 Caddy 配置文件
func (a *App) OpenProjectCaddyfile(projectName string) error {
	if projectName == "" {
		return fmt.Errorf("%s", i18n.T("專案名稱不能為空"))
	}

	// 1. 尋找專案設定
	var targetProj *config.ProjectConfig
	for i := range a.appCfg.Projects {
		if a.appCfg.Projects[i].Name == projectName {
			targetProj = &a.appCfg.Projects[i]
			break
		}
	}

	if targetProj == nil {
		return fmt.Errorf(i18n.T("找不到專案 %s"), projectName)
	}

	// 2. 推導路徑
	sitesDir := filepath.Join(a.baseDir, "conf", "sites")
	caddyPath := filepath.Join(sitesDir, targetProj.Name+".caddy")

	// 3. 確保檔案存在，如果不存在且專案啟用，則為其產生
	if _, err := os.Stat(caddyPath); os.IsNotExist(err) {
		// 呼叫 generateCaddyfiles 重新產生所有設定檔
		if err := a.generateCaddyfiles(); err != nil {
			return fmt.Errorf(i18n.T("產生 Caddy 配置文件失敗: %w"), err)
		}
	}

	// 4. 再次檢查是否真的產生成功（如果專案沒啟用，generateCaddyfiles 不會產生它）
	// 如果專案未啟用，我們手動為其單獨產生一個臨時/預設 Caddyfile 以供查看
	if _, err := os.Stat(caddyPath); os.IsNotExist(err) {
		content, err := a.buildCaddyfileContent(*targetProj)
		if err != nil {
			return fmt.Errorf(i18n.T("產生 Caddy 配置文件內容失敗: %w"), err)
		}
		if err := os.WriteFile(caddyPath, []byte(content), 0600); err != nil {
			return fmt.Errorf(i18n.T("寫入 Caddy 設定檔 %s 失敗: %w"), caddyPath, err)
		}
	}

	// 5. 使用系統預設關聯程式開啟檔案
	cmd := exec.Command("cmd", "/c", "start", "", filepath.Clean(caddyPath))
	return cmd.Start()
}

// LogEntry 描述一筆日誌記錄
type LogEntry struct {
	Text string `json:"text"`
	Time string `json:"time"`
}

// GetCategoryLogs 獲取指定分類的當天日誌歷史紀錄
// 支援 system, caddy, mariadb, mailpit, php, runtime 等分類
func (a *App) GetCategoryLogs(category string, subCategory string) ([]LogEntry, error) {
	catKey := strings.ToLower(category)
	var fileName string
	dateStr := time.Now().Format("2006-01-02")
	logDir := filepath.Join(a.baseDir, "logs")

	if catKey == "runtime" {
		projName := subCategory
		if projName == "" {
			projName = "System"
		}
		fileName = fmt.Sprintf("runtime-%s-%s.log", projName, dateStr)
	} else {
		fileName = fmt.Sprintf("wincmp-%s-%s.log", catKey, dateStr)
	}

	filePath := filepath.Join(logDir, fileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []LogEntry{}, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("讀取日誌檔失敗: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var entries []LogEntry

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}

		// 解析格式例如: [15:04:05] 日誌內容
		if len(line) >= 11 && line[0] == '[' && line[9] == ']' {
			timeStr := line[1:9]
			textStr := line[11:]
			entries = append(entries, LogEntry{
				Time: timeStr,
				Text: textStr,
			})
		} else {
			// 兜底，萬一沒有時間戳記，使用當前時間作為時間戳記
			entries = append(entries, LogEntry{
				Time: time.Now().Format("15:04:05"),
				Text: line,
			})
		}
	}

	// 限制最多回傳行數（根據設定檔中的 max_log_lines）
	limit := 500
	if a.appCfg != nil && a.appCfg.Global.MaxLogLines > 0 {
		limit = a.appCfg.Global.MaxLogLines
	}
	if len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}

	return entries, nil
}

// CheckNewVersion 檢查是否有新版本可用
func (a *App) CheckNewVersion() (*updater.ReleaseInfo, error) {
	return updater.CheckNewVersion(AppVersion)
}

// StartAutoUpdate 啟動自動下載並更新覆蓋
func (a *App) StartAutoUpdate(downloadURL string, assetType string) error {
	a.handleLog("system", i18n.Tfmt("🚀 開始下載新版本：%s (類型: %s)...", downloadURL, assetType))

	// 在協程中執行更新，避免阻塞 Wails
	go func() {
		newExePath, err := updater.DownloadAndUpdate(downloadURL, assetType, a.baseDir, func(current, total int64) {
			var percent float64 = 0
			if total > 0 {
				percent = float64(current) / float64(total)
			}

			// 將下載進度推送到前端
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "update_progress", map[string]interface{}{
					"status":    "downloading",
					"percent":   percent,
					"currentMB": float64(current) / 1024 / 1024,
					"totalMB":   float64(total) / 1024 / 1024,
				})
			}
		})

		if err != nil {
			a.handleErrorLog("system", i18n.T("自動更新失敗"), err)
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "update_progress", map[string]interface{}{
					"status": "error",
					"error":  err.Error(),
				})
			}
			return
		}

		a.handleLog("system", i18n.T("✅ 自動更新成功，程式即將重啟！"))
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "update_progress", map[string]interface{}{
				"status": "completed",
			})
		}

		// 提前釋放單實例鎖，讓新啟動的行程順利取得鎖
		singleinstance.Release()

		// 啟動新版本
		cmd := exec.Command(newExePath, "--restart")
		cmd.Dir = a.baseDir
		cmd.SysProcAttr = &syscall.SysProcAttr{
			CreationFlags: 0x01000000, // CREATE_BREAKAWAY_FROM_JOB
		}

		if err := cmd.Start(); err != nil {
			a.handleErrorLog("system", i18n.T("自動重啟失敗"), err)
		}

		// 稍微延遲後退出舊進程
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()

	return nil
}



