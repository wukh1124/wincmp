package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gopkg.in/natefinch/lumberjack.v2"

	"wincmp/internal/config"
	"wincmp/internal/i18n"
	"wincmp/internal/process"
	"wincmp/internal/resource"
	"wincmp/internal/scanner"
	"wincmp/internal/singleinstance"
	"fyne.io/systray"

	"wincmp/internal/terminal"
)

// App struct
type App struct {
	ctx        context.Context
	baseDir    string
	procMgr    *process.Manager
	resMonitor *resource.Monitor
	appCfg     *config.WincmpConfig
	scanRes    *scanner.ScanResult
	termMgr    *terminal.Manager

	// 日誌寫入器相關
	appLogWriter      *lumberjack.Logger
	errorLogCache     sync.Map
	runtimeLogWriters map[string]*lumberjack.Logger
	runtimeLogMu      sync.RWMutex

	// 資料庫連線池相關
	dbPool    *sql.DB
	dbPoolMu  sync.Mutex
	dbPoolDSN string

	saveStateMu sync.Mutex

	quitting     bool
	quittingMu   sync.RWMutex
	trayShowItem *systray.MenuItem
	trayQuitItem *systray.MenuItem
}

// NewApp 建立一個新的 App 實例
func NewApp() *App {
	return &App{
		runtimeLogWriters: make(map[string]*lumberjack.Logger),
	}
}

// startup 在應用程式啟動時由 Wails 自動呼叫，保存 context 並初始化後端模組
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// 1. 取得執行檔所在目錄作為基準路徑
	execPath, err := os.Executable()
	if err != nil {
		a.baseDir, _ = os.Getwd()
	} else {
		a.baseDir = filepath.Dir(execPath)
	}

	// 開發模式：如果 bin/ 目錄不在執行檔目錄下，嘗試用當前工作目錄
	if _, err := os.Stat(filepath.Join(a.baseDir, "bin")); os.IsNotExist(err) {
		cwd, _ := os.Getwd()
		if _, err := os.Stat(filepath.Join(cwd, "bin")); err == nil {
			a.baseDir = cwd
		}
	}

	// 在載入設定檔前，自動檢測並釋放預設設定檔到 conf/
	if err := config.RestoreDefaultConf(a.baseDir); err != nil {
		a.handleErrorLog("system", i18n.T("釋放預設設定檔失敗"), err)
	}

	// 2. 載入設定檔以套用語言與全域設定
	cfgPath := filepath.Join(a.baseDir, "conf", "wincmp.json")
	a.appCfg, err = config.Load(cfgPath)
	if err != nil {
		// 載入失敗則初始化預設設定
		a.appCfg = &config.WincmpConfig{
			Global: config.GlobalConfig{
				DefaultWWW:       "www",
				DefaultSSL:       "conf/ssl",
				RestoreLastState: true,
				MinimizeToTray:   false,
				AutoUpdateHosts:  true,
				Language:         "zh-TW",
			},
		}
	}
	i18n.SetLanguage(a.appCfg.Global.Language)

	// 3. 初始化 Lumberjack 檔案日誌寫入器
	a.initLogWriters()

	// 自動清理過期日誌檔案
	go a.cleanExpiredLogs()

	a.handleLog("system", i18n.T("WinCMP 控制面板正在啟動..."))
	a.handleLog("system", i18n.Tfmt("專案根目錄: %s", a.baseDir))
	if err != nil {
		a.handleErrorLog("system", i18n.T("無法載入設定檔"), err)
	} else {
		a.handleLog("system", i18n.Tfmt("  ✓ 設定檔已載入 (%d 個專案)", len(a.appCfg.Projects)))
	}

	// 4. 定義並連接日誌推送函式到進程管理器
	logFn := func(category string, msg string) {
		a.handleLog(category, msg)
	}

	errLogFn := func(category string, contextMsg string, err error) {
		a.handleErrorLog(category, contextMsg, err)
	}

	// 5. 初始化程序管理器
	a.procMgr = process.NewManager(a.baseDir, logFn, errLogFn)

	// 5.5 初始化資源監控器
	a.resMonitor = resource.NewAppResourceMonitor(a.procMgr)

	// 5.6 初始化終端管理器
	a.termMgr = terminal.NewManager()

	a.handleLog("system", i18n.T("掃描 ./bin 目錄中的服務版本..."))

	// 6. 掃描已安裝的服務版本
	a.scanRes, err = scanner.ScanBinDir(a.baseDir)
	if err != nil {
		a.handleErrorLog("system", i18n.T("掃描服務版本失敗"), err)
	} else {
		if len(a.scanRes.CaddyList) > 0 {
			a.handleLog("system", i18n.Tfmt("  ✓ 找到 Caddy 版本: [%s]", a.scanRes.CaddyList[0].Version))
		} else {
			a.handleLog("system", i18n.T("  ✗ 未找到 Caddy"))
		}
		if len(a.scanRes.MariaDBList) > 0 {
			a.handleLog("system", i18n.Tfmt("  ✓ 找到 MariaDB 版本: [%s]", a.scanRes.MariaDBList[0].Version))
		} else {
			a.handleLog("system", i18n.T("  ✗ 未找到 MariaDB"))
		}
		if len(a.scanRes.MailpitList) > 0 {
			a.handleLog("system", i18n.Tfmt("  ✓ 找到 Mailpit 版本: [%s]", a.scanRes.MailpitList[0].Version))
		} else {
			a.handleLog("system", i18n.T("  ✗ 未找到 Mailpit"))
		}
		if len(a.scanRes.PHPList) > 0 {
			var phpVers []string
			for _, php := range a.scanRes.PHPList {
				phpVers = append(phpVers, php.Version)
			}
			a.handleLog("system", i18n.Tfmt("  ✓ 找到 PHP 版本: [%s]", strings.Join(phpVers, ", ")))
		} else {
			a.handleLog("system", i18n.T("  ✗ 未找到 PHP"))
		}
		if len(a.scanRes.NodeList) > 0 {
			var nodeVers []string
			for _, n := range a.scanRes.NodeList {
				nodeVers = append(nodeVers, n.Version)
			}
			a.handleLog("system", i18n.Tfmt("  ✓ 找到 Node 版本: [%s]", strings.Join(nodeVers, ", ")))
		} else {
			a.handleLog("system", i18n.T("  ✗ 未找到 Node"))
		}
		if len(a.scanRes.BunList) > 0 {
			var bunVers []string
			for _, b := range a.scanRes.BunList {
				bunVers = append(bunVers, b.Version)
			}
			a.handleLog("system", i18n.Tfmt("  ✓ 找到 Bun 版本: [%s]", strings.Join(bunVers, ", ")))
		}

		if len(a.scanRes.ComposerList) > 0 {
			a.handleLog("system", i18n.Tfmt("  ✓ 找到 Composer 版本: [%s]", a.scanRes.ComposerList[0].Version))
		} else {
			a.handleLog("system", i18n.T("  ✗ 未找到 Composer"))
		}
		if len(a.scanRes.HeidiSQLList) > 0 {
			a.handleLog("system", i18n.Tfmt("  ✓ 找到 HeidiSQL 版本: [%s]", a.scanRes.HeidiSQLList[0].Version))
		} else {
			a.handleLog("system", i18n.T("  ✗ 未找到 HeidiSQL"))
		}

		a.handleLog("system", i18n.T("掃描 bin/ 目錄完成"))
	}

	// 7. 啟動背景資源監控
	a.startResourceMonitoring()

	// 8. 恢復上次關閉前的服務狀態
	go a.restoreLastState()

	// 9. 初始化系統托盤
	a.setupSystray()

	// 10. 啟動啟動訊號監聽（用於單一實例喚回）
	singleinstance.ListenForActivation(func() {
		if a.ctx != nil {
			runtime.WindowShow(a.ctx)
			runtime.WindowUnminimise(a.ctx)
		}
		singleinstance.ActivateWindow("WinCMP Control Panel")
	})
}

// shutdown 在應用程式關閉時由 Wails 自動呼叫，安全停止所有背景服務與子進程並關閉日誌
func (a *App) shutdown(ctx context.Context) {
	a.saveLastServiceState()

	if a.procMgr != nil {
		a.procMgr.StopAll()
	}
	if a.termMgr != nil {
		a.termMgr.StopAll()
	}
	a.closeDBPool()

	// 關閉所有的日誌寫入器
	a.runtimeLogMu.Lock()
	defer a.runtimeLogMu.Unlock()
	for _, w := range a.runtimeLogWriters {
		w.Close()
	}
	if a.appLogWriter != nil {
		a.appLogWriter.Close()
	}
}

// beforeClose 在視窗關閉前由 Wails 呼叫。如果設定為「最小化到托盤」，則隱藏視窗並阻止退出。
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	a.quittingMu.RLock()
	quitting := a.quitting
	a.quittingMu.RUnlock()

	// 如果正在完全退出，就讓它正常關閉
	if quitting {
		return false
	}

	// 檢查是否設定了「點擊關閉視窗時縮小至系統托盤」
	if a.appCfg != nil && a.appCfg.Global.MinimizeToTray {
		runtime.WindowHide(ctx)
		return true // 阻止應用程式關閉
	}

	return false
}

// ==========================================
// 5. 日誌系統內部方法
// ==========================================

// initLogWriters 初始化全域應用程式日誌寫入器
func (a *App) initLogWriters() {
	logDir := filepath.Join(a.baseDir, "logs")
	os.MkdirAll(logDir, 0700)

	retention := 30
	if a.appCfg != nil && a.appCfg.Global.MaxLogRetention > 0 {
		retention = a.appCfg.Global.MaxLogRetention
	}

	a.appLogWriter = &lumberjack.Logger{
		Filename:   filepath.Join(logDir, fmt.Sprintf("wincmp-%s.log", time.Now().Format("2006-01-02"))),
		MaxSize:    10,
		MaxBackups: 0,
		MaxAge:     retention,
		Compress:   true,
	}
}

// getRuntimeLogWriter 取得（或建立）指定專案的 Runtime 日誌寫入器
func (a *App) getRuntimeLogWriter(projectName string) *lumberjack.Logger {
	if projectName == "" {
		return nil
	}

	a.runtimeLogMu.Lock()
	defer a.runtimeLogMu.Unlock()

	if a.runtimeLogWriters == nil {
		a.runtimeLogWriters = make(map[string]*lumberjack.Logger)
	}

	if w, ok := a.runtimeLogWriters[projectName]; ok {
		return w
	}

	logDir := filepath.Join(a.baseDir, "logs")
	os.MkdirAll(logDir, 0700)

	retention := 30
	if a.appCfg != nil && a.appCfg.Global.MaxLogRetention > 0 {
		retention = a.appCfg.Global.MaxLogRetention
	}

	w := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, fmt.Sprintf("runtime-%s-%s.log", projectName, time.Now().Format("2006-01-02"))),
		MaxSize:    10,
		MaxBackups: 0,
		MaxAge:     retention,
		Compress:   true,
	}
	a.runtimeLogWriters[projectName] = w
	return w
}

// getCategoryLogWriter 取得（或建立）指定分類的系統日誌寫入器
func (a *App) getCategoryLogWriter(category string) *lumberjack.Logger {
	catKey := strings.ToLower(category)
	if catKey == "" {
		return nil
	}

	a.runtimeLogMu.Lock()
	defer a.runtimeLogMu.Unlock()

	if a.runtimeLogWriters == nil {
		a.runtimeLogWriters = make(map[string]*lumberjack.Logger)
	}

	// 檔名格式為 wincmp-分類-日期.log，例如 wincmp-caddy-2026-06-04.log
	logKey := "wincmp-" + catKey

	if w, ok := a.runtimeLogWriters[logKey]; ok {
		return w
	}

	logDir := filepath.Join(a.baseDir, "logs")
	os.MkdirAll(logDir, 0700)

	retention := 30
	if a.appCfg != nil && a.appCfg.Global.MaxLogRetention > 0 {
		retention = a.appCfg.Global.MaxLogRetention
	}

	w := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, fmt.Sprintf("wincmp-%s-%s.log", catKey, time.Now().Format("2006-01-02"))),
		MaxSize:    10,
		MaxBackups: 0,
		MaxAge:     retention,
		Compress:   true,
	}
	a.runtimeLogWriters[logKey] = w
	return w
}

// cleanExpiredLogs 清除超出保存期限的歷史日誌檔案
func (a *App) cleanExpiredLogs() {
	retention := 30
	if a.appCfg != nil && a.appCfg.Global.MaxLogRetention > 0 {
		retention = a.appCfg.Global.MaxLogRetention
	}

	logDir := filepath.Join(a.baseDir, "logs")
	files, err := os.ReadDir(logDir)
	if err != nil {
		return
	}

	now := time.Now()
	cutoffDate := now.AddDate(0, 0, -retention)

	// 清理格式為 *-[YYYY-MM-DD].log 的日誌檔案
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if !strings.HasSuffix(name, ".log") {
			continue
		}

		dotLog := strings.LastIndex(name, ".log")
		// 確保檔名長度足夠，且日期前一個字元必須為 "-"
		if dotLog < 11 || name[dotLog-11] != '-' {
			continue
		}

		dateStr := name[dotLog-10 : dotLog]
		fileDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		// 如果檔案的日期在截止日期之前，則刪除 it
		if fileDate.Before(cutoffDate) {
			filePath := filepath.Join(logDir, name)
			_ = os.Remove(filePath)
			a.handleLog("system", i18n.Tfmt("ℹ️ 已自動刪除過期日誌檔: %s", name))
		}
	}
}

// parseProjectFromRuntimeMsg 從 runtime log 訊息中解析專案名稱
func parseProjectFromRuntimeMsg(msg string) string {
	if start := strings.Index(msg, "["); start >= 0 {
		if end := strings.Index(msg[start:], "]"); end > 1 {
			bracketContent := msg[start+1 : start+end]
			if pStart := strings.Index(bracketContent, "("); pStart >= 0 {
				if pEnd := strings.Index(bracketContent[pStart:], ")"); pEnd > 1 {
					return bracketContent[pStart+1 : pStart+pEnd]
				}
			}
			if !strings.Contains(bracketContent, ":") {
				return bracketContent
			}
		}
	}
	return ""
}

// handleLog 處理常規日誌（寫入檔案並推送至前端）
func (a *App) handleLog(category string, msg string) {
	now := time.Now()
	timeStr := now.Format("15:04:05")
	newText := fmt.Sprintf("[%s] %s\n", timeStr, msg)

	catKey := strings.ToLower(category)

	var projectName string
	if catKey == "node" || catKey == "runtime" {
		projectName = parseProjectFromRuntimeMsg(msg)
	}

	// 1. 推送到 Wails 前端 (前端 React 會監聽 "log" 事件)
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "log", map[string]interface{}{
			"category":    catKey,
			"message":     newText,
			"time":        timeStr,
			"projectName": projectName,
		})
	}

	// 2. 寫入硬碟日誌檔
	if catKey == "node" || catKey == "runtime" {
		projectName := parseProjectFromRuntimeMsg(msg)
		if projectName != "" {
			if w := a.getRuntimeLogWriter(projectName); w != nil {
				w.Write([]byte(newText))
			}
		}
	} else {
		// 按系統分類分別寫入對應的 wincmp-分類-日期.log 檔案
		if w := a.getCategoryLogWriter(catKey); w != nil {
			w.Write([]byte(newText))
		} else if a.appLogWriter != nil {
			// 兜底降級寫入通用 wincmp-日期.log
			a.appLogWriter.Write([]byte(newText))
		}
	}
}

// handleErrorLog 處理錯誤日誌（寫入檔案並推送至前端）
func (a *App) handleErrorLog(category string, contextMsg string, err error) {
	if err == nil {
		return
	}

	// 格式化輸出
	uiMsg := fmt.Sprintf("❌ %s: %v", contextMsg, err)
	a.handleLog(category, uiMsg)

	now := time.Now()
	dateStr := now.Format("2006-01-02")
	timeStr := now.Format("15:04:05")
	detailMsg := fmt.Sprintf("[%s] [%s] %s: %+v\n", timeStr, category, contextMsg, err)

	retention := 30
	if a.appCfg != nil && a.appCfg.Global.MaxLogRetention > 0 {
		retention = a.appCfg.Global.MaxLogRetention
	}

	// 使用快取管理每日錯誤日誌寫入器
	val, _ := a.errorLogCache.LoadOrStore(dateStr, &lumberjack.Logger{
		Filename:   filepath.Join(a.baseDir, "logs", fmt.Sprintf("error-%s.log", dateStr)),
		MaxSize:    10,
		MaxBackups: 0,
		MaxAge:     retention,
		Compress:   true,
	})
	l := val.(*lumberjack.Logger)
	l.Write([]byte(detailMsg))
}

// startResourceMonitoring 定時（每 2 秒）取得系統 CPU 與記憶體資訊並推送到前端
func (a *App) startResourceMonitoring() {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if a.ctx == nil {
					return
				}
				
				var cpu float64 = 0.0
				var mem uint64 = 0
				if a.resMonitor != nil {
					cpu, mem = a.resMonitor.GetCPUAndRAM()
				}

				runtime.EventsEmit(a.ctx, "resource_usage", map[string]interface{}{
					"cpu":    cpu,
					"memory": mem,
				})
			}
		}
	}()
}

// closeDBPool 關閉 MariaDB 連線池
func (a *App) closeDBPool() {
	a.dbPoolMu.Lock()
	defer a.dbPoolMu.Unlock()
	if a.dbPool != nil {
		a.dbPool.Close()
		a.dbPool = nil
		a.dbPoolDSN = ""
	}
}

// saveLastServiceState 擷取當前所有服務狀態並寫入設定檔
func (a *App) saveLastServiceState() {
	a.saveStateMu.Lock()
	defer a.saveStateMu.Unlock()

	if a.appCfg == nil {
		return
	}

	a.appCfg.Global.LastServiceState.Caddy = a.procMgr.IsRunning("caddy")

	mariadbRunning := false
	if a.scanRes != nil {
		if a.appCfg.Global.MariaDBExternal {
			mariadbRunning = a.procMgr.IsRunning(process.MariaDBExternalServiceKey)
		} else {
			for _, info := range a.scanRes.MariaDBList {
				if a.procMgr.IsRunning(process.MariaDBServiceKey(info.Version)) {
					mariadbRunning = true
					break
				}
			}
		}
	}
	a.appCfg.Global.LastServiceState.MariaDB = mariadbRunning

	a.appCfg.Global.LastServiceState.Mailpit = a.procMgr.IsRunning(process.MailpitServiceKey())

	if a.appCfg.Global.LastServiceState.PHP == nil {
		a.appCfg.Global.LastServiceState.PHP = make(map[string]bool)
	}
	if a.scanRes != nil {
		for _, p := range a.scanRes.PHPList {
			a.appCfg.Global.LastServiceState.PHP[p.Version] = a.procMgr.IsRunning(process.PHPServiceKey(p.Version))
		}
	}

	cfgPath := filepath.Join(a.baseDir, "conf", "wincmp.json")
	if err := a.appCfg.Save(cfgPath); err != nil {
		a.handleErrorLog("system", i18n.T("無法儲存服務狀態至設定檔"), err)
	}
}

// restoreLastState 根據上次關閉時的服務狀態進行恢復
func (a *App) restoreLastState() {
	if a.appCfg == nil || a.scanRes == nil || !a.appCfg.Global.RestoreLastState {
		return
	}

	// 1. Caddy
	if a.appCfg.Global.LastServiceState.Caddy && len(a.scanRes.CaddyList) > 0 {
		info := a.scanRes.CaddyList[0]
		a.handleLog("system", i18n.T("自動啟動上次執行的服務: Caddy"))
		a.checkSSLCerts()
		if err := a.generateCaddyfiles(); err == nil {
			if err := a.procMgr.StartCaddy(info.Version, info.ExePath); err != nil {
				a.handleErrorLog("caddy", "自動啟動 Caddy 失敗", err)
			} else {
				a.triggerHostsUpdate()
			}
		} else {
			a.handleErrorLog("caddy", "自動啟動 Caddy 失敗（無法產生設定檔）", err)
		}
	}

	// 2. MariaDB
	if a.appCfg.Global.LastServiceState.MariaDB && len(a.scanRes.MariaDBList) > 0 {
		info := a.scanRes.MariaDBList[0]
		a.handleLog("system", i18n.T("自動啟動上次執行的服務: MariaDB"))
		go func() {
			done, errCh := a.procMgr.StartMariaDBAsync(
				info.Version,
				a.appCfg.Global.MariaDBExternal,
				a.appCfg.Global.MariaDBBasedir,
				a.appCfg.Global.MariaDBDatadir,
				a.appCfg.Global.MariaDBType,
				a.appCfg.Global.MariaDBPort,
			)
			<-done
			if err := <-errCh; err != nil {
				a.handleErrorLog("mariadb", "自動啟動 MariaDB 失敗", err)
			}
		}()
	}

	// 3. Mailpit
	if a.appCfg.Global.LastServiceState.Mailpit && len(a.scanRes.MailpitList) > 0 {
		mpInfo := a.scanRes.MailpitList[0]
		mpSmtp := 1025
		mpHttp := 8025
		if a.appCfg.Global.MailpitSMTPPort > 0 {
			mpSmtp = a.appCfg.Global.MailpitSMTPPort
		}
		if a.appCfg.Global.MailpitHTTPPort > 0 {
			mpHttp = a.appCfg.Global.MailpitHTTPPort
		}
		a.handleLog("system", i18n.T("自動啟動上次執行的服務: Mailpit"))
		if err := a.procMgr.StartMailpit(mpInfo.Version, mpInfo.ExePath, mpSmtp, mpHttp, a.appCfg.Global.MailpitUseDB); err != nil {
			a.handleErrorLog("mailpit", "自動啟動 Mailpit 失敗", err)
		}
	}

	// 4. PHP-CGI
	if a.appCfg.Global.LastServiceState.PHP != nil {
		for i := range a.scanRes.PHPList {
			info := &a.scanRes.PHPList[i]
			if a.appCfg.Global.LastServiceState.PHP[info.Version] {
				a.handleLog("system", i18n.Tfmt("自動啟動上次執行的服務: PHP-CGI %s", info.Version))
				// 套用進程數配置
				count := a.appCfg.Global.PHP.ProcessesPerVersion
				if c, ok := a.appCfg.Global.PHP.Processes[info.MajorMin]; ok {
					count = c
				}
				info.PortCount = count

				if err := a.procMgr.StartPHPCGI(*info); err != nil {
					a.handleErrorLog("php", fmt.Sprintf("自動啟動 PHP-CGI %s 失敗", info.Version), err)
				}
			}
		}
	}
}

