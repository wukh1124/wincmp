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
	"wincmp/internal/scanner"
)

// App struct
type App struct {
	ctx     context.Context
	baseDir string
	procMgr *process.Manager
	appCfg  *config.WincmpConfig
	scanRes *scanner.ScanResult

	// 日誌寫入器相關
	appLogWriter      *lumberjack.Logger
	errorLogCache     sync.Map
	runtimeLogWriters map[string]*lumberjack.Logger
	runtimeLogMu      sync.RWMutex

	// 資料庫連線池相關
	dbPool    *sql.DB
	dbPoolMu  sync.Mutex
	dbPoolDSN string
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
		a.handleErrorLog("system", "釋放預設設定檔失敗", err)
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

	// 4. 定義並連接日誌推送函式到進程管理器
	logFn := func(category string, msg string) {
		a.handleLog(category, msg)
	}

	errLogFn := func(category string, contextMsg string, err error) {
		a.handleErrorLog(category, contextMsg, err)
	}

	// 5. 初始化程序管理器
	a.procMgr = process.NewManager(a.baseDir, logFn, errLogFn)

	// 6. 掃描已安裝的服務版本
	a.scanRes, err = scanner.ScanBinDir(a.baseDir)
	if err != nil {
		a.handleErrorLog("system", "掃描服務版本失敗", err)
	} else {
		a.handleLog("system", "掃描 bin/ 目錄完成")
	}

	// 7. 啟動背景資源監控
	a.startResourceMonitoring()
}

// shutdown 在應用程式關閉時由 Wails 自動呼叫，安全停止所有背景服務與子進程並關閉日誌
func (a *App) shutdown(ctx context.Context) {
	if a.procMgr != nil {
		a.procMgr.StopAll()
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

	// 1. 推送到 Wails 前端 (前端 React 會監聽 "log" 事件)
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "log", map[string]interface{}{
			"category": catKey,
			"message":  newText,
			"time":     timeStr,
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
	} else if a.appLogWriter != nil {
		a.appLogWriter.Write([]byte(newText))
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
				// 這裡可以透過 a.procMgr 或是 wincmp/internal/resource 模組獲取 CPU 與 RAM 佔用
				// 目前我們定時推送一個假的數據結構，待 Phase 4 完整移植資源監控模組
				runtime.EventsEmit(a.ctx, "resource_usage", map[string]interface{}{
					"cpu":    1.5, // %
					"memory": 45,  // MB
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
