package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"golang.org/x/sys/windows"

	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"wincmp/internal/config"
	"wincmp/internal/detect"
	"wincmp/internal/hosts"
	"wincmp/internal/port"
	"wincmp/internal/process"
	"wincmp/internal/scanner"
	"wincmp/internal/singleinstance"

	"fyne.io/fyne/v2/data/binding"

	"sync"
	"sync/atomic"

	fynetooltip "github.com/dweymouth/fyne-tooltip"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
	_ "github.com/go-sql-driver/mysql"
	"github.com/ncruces/zenity"
	"gopkg.in/natefinch/lumberjack.v2"
)

// 全域變數
var (
	sysLog       = binding.NewString()
	caddyLog     = binding.NewString()
	dbLog        = binding.NewString()
	phpLog       = binding.NewString()
	logEntries   map[string]*container.Scroll
	logTabs      *container.AppTabs // 用於切換分頁
	procMgr      *process.Manager
	scanRes      *scanner.ScanResult
	appCfg       *config.WincmpConfig
	baseDir      string      // 專案根目錄
	isZenityOpen atomic.Bool // 防止重複開啟檔案選擇器
	myApp        fyne.App    // Fyne App 實例，用於主題切換

	// 切換分頁的節流 / 防抖相關
	tabSwitchMu sync.Mutex
	lastSwitch  time.Time
	tabTimer    *time.Timer

	// 主分頁
	mainTabs *container.AppTabs
)

func addLog(category string, msg string) {
	now := time.Now()
	timeStr := now.Format("15:04:05")
	newText := fmt.Sprintf("[%s] %s\n", timeStr, msg)

	var logSource binding.String
	catKey := strings.ToLower(category)
	tabIndex := 0

	switch catKey {
	case "caddy":
		logSource = caddyLog
		tabIndex = 1
	case "mariadb", "db", "mysql":
		catKey = "mariadb"
		logSource = dbLog
		tabIndex = 2
	case "php":
		logSource = phpLog
		tabIndex = 3
	default:
		catKey = "system"
		logSource = sysLog
		tabIndex = 0
	}

	if logSource != nil {
		oldText, _ := logSource.Get()
		logSource.Set(oldText + newText)

		if logEntries != nil {
			if scroll, ok := logEntries[catKey]; ok && scroll != nil {
				fyne.Do(func() {
					scroll.ScrollToBottom()
				})
			}
		}

		// 自動切換分頁機制 (Throttling + Debounce)
		if logTabs != nil {
			tabSwitchMu.Lock()
			now := time.Now()
			// 如果離上次切換已經超過 500ms，就立刻切換 (Leading edge)
			if now.Sub(lastSwitch) >= 500*time.Millisecond {
				lastSwitch = now
				fyne.Do(func() {
					if logTabs != nil && logTabs.CurrentTabIndex() != tabIndex {
						logTabs.SelectIndex(tabIndex)
					}
				})
			} else {
				// 否則延遲 500ms 後切換 (Trailing edge)
				if tabTimer != nil {
					tabTimer.Stop()
				}
				tabTimer = time.AfterFunc(500*time.Millisecond, func() {
					tabSwitchMu.Lock()
					lastSwitch = time.Now()
					tabSwitchMu.Unlock()
					fyne.Do(func() {
						if logTabs != nil && logTabs.CurrentTabIndex() != tabIndex {
							logTabs.SelectIndex(tabIndex)
						}
					})
				})
			}
			tabSwitchMu.Unlock()
		}
	}

	// 確保日誌目錄存在
	logDir := filepath.Join(baseDir, "logs")
	os.MkdirAll(logDir, 0755)

	// 按日期分檔: wincmp-YYYY-MM-DD.log
	dateStr := now.Format("2006-01-02")
	appLogPath := filepath.Join(logDir, fmt.Sprintf("wincmp-%s.log", dateStr))

	retention := 0
	if appCfg != nil {
		retention = appCfg.Global.MaxLogRetention
	}

	// 使用 lumberjack 處理日誌滾動與保留
	l := &lumberjack.Logger{
		Filename:   appLogPath,
		MaxSize:    10, // megabytes
		MaxBackups: 0,  // 不限制數量，改用天數限制
		MaxAge:     retention,
		Compress:   true,
	}
	defer l.Close()
	l.Write([]byte(newText))
}

// addErrorLog 全域錯誤日誌機制
func addErrorLog(category string, contextMsg string, err error) {
	if err == nil {
		return
	}

	// 格式化輸出到 UI 的訊息
	uiMsg := fmt.Sprintf("❌ %s: %v", contextMsg, err)
	addLog(category, uiMsg)

	now := time.Now()
	dateStr := now.Format("2006-01-02")
	timeStr := now.Format("15:04:05")

	// 寫入獨立錯誤日誌檔: error-YYYY-MM-DD.log
	logDir := filepath.Join(baseDir, "logs")
	os.MkdirAll(logDir, 0755)

	errLogPath := filepath.Join(logDir, fmt.Sprintf("error-%s.log", dateStr))
	detailMsg := fmt.Sprintf("[%s] [%s] %s: %+v\n", timeStr, category, contextMsg, err)

	retention := 0
	if appCfg != nil {
		retention = appCfg.Global.MaxLogRetention
	}

	// 使用 lumberjack 處理日誌滾動與保留
	l := &lumberjack.Logger{
		Filename:   errLogPath,
		MaxSize:    10, // megabytes
		MaxBackups: 0,
		MaxAge:     retention,
		Compress:   true,
	}
	defer l.Close()
	l.Write([]byte(detailMsg))
}

// openZenitySelector 統一管理對話框，並強制阻擋主視窗互動 (Modal 模式)
func openZenitySelector(win fyne.Window, currentPath, fallbackPath string, isDir bool, callback func(string), opts ...zenity.Option) {
	// 1. 檢查並鎖定 (避免連點開啟多個)
	if !isZenityOpen.CompareAndSwap(false, true) {
		return
	}

	// 2. 建立並掛載「互動阻擋器」到 Fyne 視窗最上層
	blocker := newModalBlocker()
	blocker.Resize(win.Canvas().Size())
	win.Canvas().Overlays().Add(blocker)

	go func() {
		// 3. 確保對話框關閉後，移除阻擋器並解除鎖定
		defer func() {
			win.Canvas().Overlays().Remove(blocker)
			isZenityOpen.Store(false)
		}()

		startPath := fallbackPath

		// 檢查路徑是否存在
		if currentPath != "" {
			if info, err := os.Stat(currentPath); err == nil {
				if isDir && info.IsDir() {
					startPath = currentPath
				} else if !isDir && !info.IsDir() {
					startPath = currentPath
				}
			} else if !isDir {
				dir := filepath.Dir(currentPath)
				if dirInfo, err := os.Stat(dir); err == nil && dirInfo.IsDir() {
					startPath = dir + string(os.PathSeparator)
				}
			}
		}

		finalOpts := append([]zenity.Option{zenity.Filename(startPath)}, opts...)
		if isDir {
			finalOpts = append(finalOpts, zenity.Directory())
		}

		// 呼叫 Zenity (此時因為 go routine 阻塞在這裡，Fyne 視窗仍被 blocker 覆蓋)
		path, err := zenity.SelectFile(finalOpts...)
		if err == nil && path != "" {
			callback(path)
		}
	}()
}

// openLocalPath 透過系統預設程式開啟檔案或資料夾
func openLocalPath(path string) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		addErrorLog("system", "路徑無效: "+path, err)
		return
	}
	err = windows.ShellExecute(0, toPtr("open"), toPtr(absPath), nil, nil, 1)
	if err != nil {
		addErrorLog("system", "開啟失敗: "+absPath, err)
	}
}

func toPtr(s string) *uint16 {
	return windows.StringToUTF16Ptr(s)
}

// monitorUptime 輔助函式：用來背景更新運行時間
func monitorUptime(serviceKey string, uptimeData binding.String) {
	ctx := procMgr.GetContext(serviceKey)
	startTime := procMgr.GetStartTime(serviceKey)

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				uptimeData.Set("")
				return
			case now := <-ticker.C:
				dur := now.Sub(startTime)
				uptimeData.Set(fmt.Sprintf("%02d:%02d:%02d", int(dur.Hours()), int(dur.Minutes())%60, int(dur.Seconds())%60))
			}
		}
	}()
}

// saveLastServiceState 擷取當前所有服務狀態並寫入設定檔
func saveLastServiceState() {
	if appCfg == nil {
		return
	}
	appCfg.Global.LastServiceState.Caddy = procMgr.IsRunning("caddy")

	mariadbRunning := false
	for _, info := range scanRes.MariaDBList {
		if procMgr.IsRunning(process.MariaDBServiceKey(info.Version)) {
			mariadbRunning = true
			break
		}
	}
	appCfg.Global.LastServiceState.MariaDB = mariadbRunning

	if appCfg.Global.LastServiceState.PHP == nil {
		appCfg.Global.LastServiceState.PHP = make(map[string]bool)
	}
	for _, p := range scanRes.PHPList {
		appCfg.Global.LastServiceState.PHP[p.Version] = procMgr.IsRunning(process.PHPServiceKey(p.Version))
	}

	cfgPath := filepath.Join(baseDir, "conf", "wincmp.json")
	if err := appCfg.Save(cfgPath); err != nil {
		addErrorLog("system", "無法儲存服務狀態至設定檔", err)
	}
}

// saveAndQuit 儲存目前服務狀態並關閉應用程式
func saveAndQuit(myApp fyne.App) {
	addLog("system", "正在儲存狀態與關閉所有服務...")
	saveLastServiceState()
	procMgr.StopAll()
	myApp.Quit()
}


func main() {
	// ─── 單一執行個體防護 ──────────────────────────────
	isFirst, err := singleinstance.TryAcquire()
	if err != nil {
		// Mutex 建立失敗，保守地繼續執行
		// (極少見，通常是系統資源問題)
	} else if !isFirst {
		// 已有另一個 WinCMP 在執行
		singleinstance.BringExistingToFront() // 透過管道喚回現有視窗
		os.Exit(0)                            // 靜默退出
	}
	defer singleinstance.Release()
	// ──────────────────────────────────────────────────

	// 將 Tooltip 的字體大小設定為標準內文大小 (預設是較小的 CaptionText)
	fynetooltip.SetToolTipTextSizeName(theme.SizeNameText)

	// 取得執行檔所在目錄作為基準路徑
	execPath, err := os.Executable()
	if err != nil {
		// Fallback: 使用當前工作目錄
		baseDir, _ = os.Getwd()
	} else {
		baseDir = filepath.Dir(execPath)
	}

	// 開發模式：如果 bin/ 目錄不在執行檔目錄下，嘗試用當前工作目錄
	if _, err := os.Stat(filepath.Join(baseDir, "bin")); os.IsNotExist(err) {
		cwd, _ := os.Getwd()
		if _, err := os.Stat(filepath.Join(cwd, "bin")); err == nil {
			baseDir = cwd
		}
	}

	myApp = app.New()
	myApp.SetIcon(resourceIconSvg)
	// myWindow := myApp.NewWindow("WinCMP Control Panel - Development Stack of Caddy MariaDB PHP")
	myWindow := myApp.NewWindow("WinCMP")
	myWindow.Resize(fyne.NewSize(1000, 700))

	// 啟動啟動訊號監聽（用於單一執行個體喚回）
	singleinstance.ListenForActivation(func() {
		fyne.Do(func() {
			myWindow.Show()
			singleinstance.ActivateWindow("WinCMP")
		})
	})

	// --- Log 區塊初始化 (Tabs + Binding) ---
	logEntries = make(map[string]*container.Scroll)

	createLogTab := func(data binding.String, category string) fyne.CanvasObject {
		// 使用 RichText 取代 MultiLineEntry，避免灰色禁用背景問題並提升效能 (解法 A)
		richText := widget.NewRichText()
		richText.Wrapping = fyne.TextWrapBreak

		// 監聽數據變化並更新 RichText
		data.AddListener(binding.NewDataListener(func() {
			val, _ := data.Get()
			// 由於 Terminal Log 主要是純文字，直接更新 Segments 效率最高
			richText.Segments = []widget.RichTextSegment{
				&widget.TextSegment{
					Style: widget.RichTextStyleCodeBlock, // 使用等寬字體樣式更有 Terminal 感
					Text:  val,
				},
			}
			richText.Refresh()
		}))

		scroll := container.NewVScroll(richText)
		logEntries[category] = scroll

		// 仍套用 logTheme 來控制邊距 (InnerPadding=0)
		return container.NewThemeOverride(scroll, &logTheme{Theme: theme.DefaultTheme()})
	}

	logTabs = container.NewAppTabs(
		container.NewTabItem("System", createLogTab(sysLog, "system")),
		container.NewTabItem("Caddy", createLogTab(caddyLog, "caddy")),
		container.NewTabItem("MariaDB", createLogTab(dbLog, "mariadb")),
		container.NewTabItem("PHP", createLogTab(phpLog, "php")),
	)

	addLog("system", "正在初始化 WinCMP...")
	addLog("system", fmt.Sprintf("專案根目錄: %s", baseDir))

	// --- 建立程序管理器 ---
	procMgr = process.NewManager(baseDir, addLog, addErrorLog)

	// --- 掃描已安裝的服務版本 ---
	addLog("system", "掃描 ./bin 目錄中的服務版本...")
	scanRes, err = scanner.ScanBinDir(baseDir)
	if err != nil {
		addErrorLog("system", "掃描服務版本失敗", err)
	} else {
		if len(scanRes.CaddyList) > 0 {
			versions := make([]string, len(scanRes.CaddyList))
			for i, c := range scanRes.CaddyList {
				versions[i] = c.Version
			}
			addLog("system", fmt.Sprintf("  ✓ 找到 Caddy 版本: [%s]", strings.Join(versions, ", ")))
		} else {
			addLog("system", "  ✗ 未找到 Caddy")
		}
		if len(scanRes.MariaDBList) > 0 {
			versions := make([]string, len(scanRes.MariaDBList))
			for i, m := range scanRes.MariaDBList {
				versions[i] = m.Version
			}
			addLog("system", fmt.Sprintf("  ✓ 找到 MariaDB 版本: [%s]", strings.Join(versions, ", ")))
		} else {
			addLog("system", "  ✗ 未找到 MariaDB")
		}
		if len(scanRes.PHPList) > 0 {
			versions := make([]string, len(scanRes.PHPList))
			for i, p := range scanRes.PHPList {
				versions[i] = p.Version
			}
			addLog("system", fmt.Sprintf("  ✓ 找到 PHP 版本: [%s]", strings.Join(versions, ", ")))
			// 顯示略過的版本
			if len(scanRes.SkippedPHP) > 0 {
				addLog("system", fmt.Sprintf("  ℹ 略過舊 Patch 版本 (僅保留最新): [%s]", strings.Join(scanRes.SkippedPHP, ", ")))
			}
		} else {
			addLog("system", "  ✗ 未找到 PHP")
		}
	}

	// --- 載入設定檔 ---
	cfgPath := filepath.Join(baseDir, "conf", "wincmp.json")
	appCfg, err = config.Load(cfgPath)
	if err != nil {
		addErrorLog("system", "無法載入設定檔", err)
		// 使用預設設定
		appCfg = &config.WincmpConfig{
			Global: config.GlobalConfig{
				DefaultWWW:      "www",
				DefaultSSL:      "conf/ssl",
				RestoreLastState: bool(true), // 用於相容性
				MinimizeToTray:   false,
				AutoUpdateHosts: true,
			},
		}
	} else {
		addLog("system", fmt.Sprintf("  ✓ 設定檔已載入 (%d 個專案)", len(appCfg.Projects)))
	}

	// --- 套用已儲存的主題設定 ---
	applyTheme(appCfg.Global.Theme)

	// 初始化 PHP 設定 ( map 需要 make)
	if appCfg.Global.PHP.Processes == nil {
		appCfg.Global.PHP.Processes = make(map[string]int)
	}
	if appCfg.Global.PHP.ProcessesPerVersion == 0 {
		appCfg.Global.PHP.ProcessesPerVersion = 3 // 預設 3 個
	}

	// --- 根據上次狀態自動啟動服務 ---
	if appCfg.Global.RestoreLastState {
		if appCfg.Global.LastServiceState.Caddy && len(scanRes.CaddyList) > 0 {
			info := scanRes.CaddyList[0]
			addLog("system", "自動啟動上次執行的服務: Caddy")
			if err := checkSSLCerts(); err == nil && generateCaddyfiles() == nil {
				procMgr.StartCaddy(info.Version, info.ExePath)
				triggerHostsUpdate()
			}
		}

		if appCfg.Global.LastServiceState.MariaDB && len(scanRes.MariaDBList) > 0 {
			info := scanRes.MariaDBList[0]
			addLog("system", "自動啟動上次執行的服務: MariaDB")
			procMgr.StartMariaDB(info.Version)
		}
	}

	// 初始化 PHP 進程數設定並套用到掃描結果
	for i := range scanRes.PHPList {
		info := &scanRes.PHPList[i]
		count := appCfg.Global.PHP.ProcessesPerVersion
		if c, ok := appCfg.Global.PHP.Processes[info.MajorMin]; ok {
			count = c
		}
		info.PortCount = count
		// 同步回 map，確保 UI 初始化時比對成功，避免誤發變更日誌
		appCfg.Global.PHP.Processes[info.MajorMin] = count
	}

	if appCfg.Global.RestoreLastState && appCfg.Global.LastServiceState.PHP != nil {
		for i := range scanRes.PHPList {
			info := &scanRes.PHPList[i]
			if appCfg.Global.LastServiceState.PHP[info.Version] {
				addLog("system", fmt.Sprintf("自動啟動上次執行的服務: PHP-CGI %s", info.Version))
				procMgr.StartPHPCGI(*info)
			}
		}
	}

	// 自動啟動完成後，檢查 Caddy 啟用的專案是否需要未啟動的 PHP
	if procMgr.IsRunning("caddy") {
		checkPHPForProjects(myWindow)
	}

	logPanel := container.NewBorder(
		widget.NewLabelWithStyle("Terminal Logs", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil, nil, nil, logTabs,
	)

	// --- 各 Tab 內容 ---
	dashboardContent := createDashboard(myWindow, func() {
		// 這裡傳入一個重新整理 Projects Tab 的內容的閉包，如果需要的話
	})
	projectsContent := createProjectsTab(myWindow)
	dbExplorerContent := createDatabaseExplorerTab()
	settingsContent := createSettingsTab(myWindow)

	// --- 左側選單 (Sidebar) ---
	mainTabs = container.NewAppTabs(
		container.NewTabItemWithIcon("Dashboard", theme.HomeIcon(), dashboardContent),
		container.NewTabItemWithIcon("Projects", theme.FolderIcon(), projectsContent),
		container.NewTabItemWithIcon("DB Explorer", theme.StorageIcon(), dbExplorerContent),
		container.NewTabItemWithIcon("Settings", theme.SettingsIcon(), settingsContent),
	)
	mainTabs.SetTabLocation(container.TabLocationLeading)

	// 上下分割 (上方功能區 65%，下方 Log 區 35%)
	mainLayout := container.NewVSplit(mainTabs, logPanel)
	mainLayout.SetOffset(0.65)

	// --- System Tray (系統匣支援) ---
	refreshSystemTray(myApp, myWindow)

	// 程式關閉時縮小到系統匣
	myWindow.SetCloseIntercept(func() {
		if appCfg.Global.MinimizeToTray {
			myWindow.Hide()
		} else {
			saveAndQuit(myApp)
		}
	})

	myWindow.SetContent(fynetooltip.AddWindowToolTipLayer(mainLayout, myWindow.Canvas()))
	myWindow.ShowAndRun()
}

// ===== 自定義主題解決 Log 對比度 =====
type logTheme struct {
	fyne.Theme
}

func (m *logTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNameDisabled {
		return theme.ForegroundColor() // 讓 Disabled 文字顏色跟一般文字一樣
	}
	return m.Theme.Color(name, variant)
}

func (m *logTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNameInnerPadding || name == theme.SizeNameInputBorder {
		return 0
	}
	return m.Theme.Size(name)
}

// ===== 自定義按鈕主題（保留 Start/Stop 狀態顏色）====
type coloredButtonTheme struct {
	fyne.Theme
	isStop func() bool
}

func (m *coloredButtonTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	isStop := m.isStop != nil && m.isStop()

	switch name {
	case theme.ColorNameForeground:
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}
	case theme.ColorNameButton:
		if isStop {
			return color.RGBA{R: 220, G: 53, B: 69, A: 255}
		}
		return color.RGBA{R: 40, G: 167, B: 69, A: 255}
	}
	return m.Theme.Color(name, variant)
}

// ===== 自訂深色主題工廠 =====
var darkThemeColors = map[string]struct {
	background color.Color
	cardBg     color.Color
	inputBg    color.Color
	border     color.Color
}{
	"blue": {
		background: color.RGBA{R: 0x1a, G: 0x1a, B: 0x2e, A: 255},
		cardBg:     color.RGBA{R: 0x16, G: 0x21, B: 0x3e, A: 255},
		inputBg:    color.RGBA{R: 0x12, G: 0x19, B: 0x2c, A: 255},
		border:     color.RGBA{R: 0x2a, G: 0x3a, B: 0x5e, A: 255},
	},
	"gray": {
		background: color.RGBA{R: 0x2d, G: 0x2d, B: 0x2d, A: 255},
		cardBg:     color.RGBA{R: 0x3d, G: 0x3d, B: 0x3d, A: 255},
		inputBg:    color.RGBA{R: 0x35, G: 0x35, B: 0x35, A: 255},
		border:     color.RGBA{R: 0x4d, G: 0x4d, B: 0x4d, A: 255},
	},
	"twilight": {
		background: color.RGBA{R: 0x1e, G: 0x1e, B: 0x2e, A: 255},
		cardBg:     color.RGBA{R: 0x2a, G: 0x2a, B: 0x3e, A: 255},
		inputBg:    color.RGBA{R: 0x24, G: 0x24, B: 0x34, A: 255},
		border:     color.RGBA{R: 0x3a, G: 0x3a, B: 0x5e, A: 255},
	},
}

type customDarkTheme struct {
	fyne.Theme
	background color.Color
	cardBg     color.Color
	inputBg    color.Color
	border     color.Color
}

func (t *customDarkTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return t.background
	case theme.ColorNameButton:
		return t.cardBg
	case theme.ColorNameInputBackground:
		return t.inputBg
	case theme.ColorNameSeparator:
		return t.border
	}
	return t.Theme.Color(name, variant)
}

func createDarkTheme(variant string) fyne.Theme {
	base := theme.DarkTheme()
	colors, ok := darkThemeColors[variant]
	if !ok {
		colors = darkThemeColors["blue"]
	}
	return &customDarkTheme{
		Theme:      base,
		background: colors.background,
		cardBg:     colors.cardBg,
		inputBg:    colors.inputBg,
		border:     colors.border,
	}
}

// ===== 1. Dashboard（動態版本 + 實際程序管理）=====

func createDashboard(win fyne.Window, refreshProjects func()) fyne.CanvasObject {
	header := container.NewGridWithColumns(6,
		widget.NewLabelWithStyle("Service", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Version", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Status / PID", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Uptime", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Port(s)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Action", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
	)

	rows := []fyne.CanvasObject{
		widget.NewLabelWithStyle("Service Modules Manager", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		header,
		widget.NewSeparator(),
	}

	// Caddy 行
	for _, caddyInfo := range scanRes.CaddyList {
		rows = append(rows, createCaddyRow(win, caddyInfo, refreshProjects))
	}

	// MariaDB 行
	for _, mariaDBInfo := range scanRes.MariaDBList {
		rows = append(rows, createMariaDBRow(mariaDBInfo))
	}

	if len(scanRes.PHPList) > 0 {
		rows = append(rows,
			widget.NewSeparator(),
			widget.NewLabelWithStyle("PHP FastCGI Processes (Multi-port)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		)
		for _, phpInfo := range scanRes.PHPList {
			rows = append(rows, createPHPRow(phpInfo))
		}
	}

	vbox := container.NewVBox(rows...)
	return container.NewPadded(vbox)
}

var domainReg = regexp.MustCompile(`[^a-z0-9]+`)

func GenerateValidDomain(folderName string) string {
	domain := strings.ToLower(folderName)
	domain = domainReg.ReplaceAllString(domain, "-")
	domain = strings.Trim(domain, "-")
	if domain == "" {
		domain = "project"
	}
	return fmt.Sprintf("local-%s.test", domain)
}

// checkSSLCerts 檢查所有專案中的 SSL 憑證是否存在，若使用自動 TLS 可忽略
func checkSSLCerts() error {
	for _, proj := range appCfg.Projects {
		if !proj.Enabled || !proj.UseSSL {
			continue
		}
		crt := appCfg.GetSSLCertPath(proj, baseDir)
		key := appCfg.GetSSLKeyPath(proj, baseDir)
		certExists := crt != "" && key != ""
		if certExists {
			if _, err := os.Stat(crt); os.IsNotExist(err) {
				certExists = false
				addLog("caddy", fmt.Sprintf("⚠️ 專案 %s: 憑證遺失，將使用自動 TLS", proj.Name))
			} else if _, err := os.Stat(key); os.IsNotExist(err) {
				certExists = false
				addLog("caddy", fmt.Sprintf("⚠️ 專案 %s: 金鑰遺失，將使用自動 TLS", proj.Name))
			}
		}
	}
	return nil
}

// scanDefaultWWW 掃描預設目錄加入新專案
func scanDefaultWWW() {
	if appCfg.Global.DefaultWWW == "" {
		return
	}
	// 判斷是否為絕對路徑
	var wwwDir string
	if filepath.IsAbs(appCfg.Global.DefaultWWW) {
		wwwDir = appCfg.Global.DefaultWWW
	} else {
		wwwDir = filepath.Join(baseDir, appCfg.Global.DefaultWWW)
	}
	entries, err := os.ReadDir(wwwDir)
	if err != nil {
		addErrorLog("system", "無法掃描 www 目錄", err)
		return
	}

	added := 0
	for _, d := range entries {
		if !d.IsDir() {
			continue
		}
		name := d.Name()
		// 檢查是否已存在
		exists := false
		for _, p := range appCfg.Projects {
			if p.Name == name {
				exists = true
				break
			}
		}
		if !exists {
			projectPath := filepath.Join(wwwDir, name)
			res := detect.DetectLaravel(projectPath)
			projectType := ""
			phpVersion := ""
			if res.IsLaravel {
				projectType = "laravel"
				phpVersion = "8.2"
			}

			appCfg.Projects = append(appCfg.Projects, config.ProjectConfig{
				Name:       name,
				Domains:    []string{GenerateValidDomain(name)},
				PHPVersion: phpVersion,
				Type:       projectType,
				UseSSL:     true,
				Enabled:    false,
			})
			added++

			if res.IsLaravel {
				addLog("system", fmt.Sprintf("  ↳ %s: 偵測為 Laravel (Confidence: %d, Reasons: %s)", name, res.Confidence, strings.Join(res.Reasons, ", ")))
			}
		}
	}
	if added > 0 {
		appCfg.Save(filepath.Join(baseDir, "conf", "wincmp.json"))
		addLog("system", fmt.Sprintf("📌 已自動掃描並加入 %d 個新專案", added))
	}
}

// generatePHPUpstream 產生 PHP 負載平衡設定檔
func generatePHPUpstream() error {
	snippetDir := filepath.Join(baseDir, "conf", "snippets")
	os.MkdirAll(snippetDir, 0755)
	upstreamPath := filepath.Join(snippetDir, "php-upstream.caddy")

	var content strings.Builder
	for _, info := range scanRes.PHPList {
		// 根據設定取得進程數
		count := appCfg.Global.PHP.ProcessesPerVersion
		if c, ok := appCfg.Global.PHP.Processes[info.MajorMin]; ok {
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

	return os.WriteFile(upstreamPath, []byte(content.String()), 0644)
}

// generateCaddyfiles 產生所有子專案的 .caddy 設定檔
func generateCaddyfiles() error {
	// 先產生 PHP Upstream
	if err := generatePHPUpstream(); err != nil {
		return err
	}

	sitesDir := filepath.Join(baseDir, "conf", "sites")
	os.MkdirAll(sitesDir, 0755)

	// 清除舊的 .caddy 檔
	if oldFiles, err := filepath.Glob(filepath.Join(sitesDir, "*.caddy")); err == nil {
		for _, f := range oldFiles {
			os.Remove(f)
		}
	}

	for _, proj := range appCfg.Projects {
		if !proj.Enabled {
			continue
		}

		// 產生站點檔案
		caddyPath := filepath.Join(sitesDir, proj.Name+".caddy")

		var domainsStr string
		if len(proj.Domains) > 0 {
			domainsStr = strings.Join(proj.Domains, ", ")
		} else {
			domainsStr = "local-" + proj.Name + ".test"
		}

		content := fmt.Sprintf("%s {\n", domainsStr)

		// SSL
		if proj.UseSSL {
			crt := appCfg.GetSSLCertPath(proj, baseDir)
			key := appCfg.GetSSLKeyPath(proj, baseDir)
			certExists := crt != "" && key != ""
			if certExists {
				// 檢查憑證與金鑰檔案是否存在
				if _, err := os.Stat(crt); os.IsNotExist(err) {
					certExists = false
					addErrorLog("caddy", fmt.Sprintf("專案 %s: 憑證遺失，使用自動 TLS", proj.Name), nil)
				} else if _, err := os.Stat(key); os.IsNotExist(err) {
					certExists = false
					addErrorLog("caddy", fmt.Sprintf("專案 %s: 金鑰遺失，使用自動 TLS", proj.Name), nil)
				}
			}
			if certExists {
				crt = strings.ReplaceAll(crt, "\\", "/")
				key = strings.ReplaceAll(key, "\\", "/")
				content += fmt.Sprintf("\ttls %s %s\n", crt, key)
			} else {
				content += "\ttls internal\n"
			}
		}

		// IP Allow rule or others
		content += "\timport common_dev\n"

		// Root
		root := appCfg.GetProjectRoot(proj, baseDir)
		root = strings.ReplaceAll(root, "\\", "/")
		content += fmt.Sprintf("\troot * %s\n", root)

		if proj.PHPVersion != "" {
			phpVerStr := strings.ReplaceAll(proj.PHPVersion, ".", "")
			content += fmt.Sprintf("\timport php%s\n", phpVerStr)
		}

		content += "\timport static_site\n"
		content += "}\n"

		os.WriteFile(caddyPath, []byte(content), 0644)
	}
	return nil
}

// checkPHPForProjects 檢查已啟用專案所需的 PHP 版本是否已啟動，
// 若有未啟動的 PHP 版本，輸出 Log 並彈出 Dialog 提醒。
func checkPHPForProjects(win fyne.Window) {
	type projectPHPStatus struct {
		ProjectName string
		PHPVersion  string // MajorMin, e.g. "8.2"
		IsRunning   bool
	}

	var statuses []projectPHPStatus

	for _, proj := range appCfg.Projects {
		if !proj.Enabled || proj.PHPVersion == "" {
			continue
		}

		// 透過 MajorMin 找到對應的完整版本號，檢查 PHP-CGI 是否已啟動
		running := false
		for _, phpInfo := range scanRes.PHPList {
			if phpInfo.MajorMin == proj.PHPVersion {
				serviceKey := process.PHPServiceKey(phpInfo.Version)
				if procMgr.IsRunning(serviceKey) {
					running = true
				}
				break
			}
		}

		statuses = append(statuses, projectPHPStatus{
			ProjectName: proj.Name,
			PHPVersion:  proj.PHPVersion,
			IsRunning:   running,
		})
	}

	if len(statuses) == 0 {
		return
	}

	// 方案 A: Log 提示
	hasUnstarted := false
	for _, s := range statuses {
		if !s.IsRunning {
			hasUnstarted = true
			addLog("system", fmt.Sprintf("⚠️ 專案 %s 需要 PHP %s，但 PHP %s 尚未啟動！",
				s.ProjectName, s.PHPVersion, s.PHPVersion))
		}
	}
	if hasUnstarted {
		addLog("system", "💡 請在 Dashboard 的 PHP FastCGI 區塊啟動對應版本")
	}

	// 方案 B: Dialog 彈窗（僅在有未啟動的 PHP 時顯示）
	if !hasUnstarted {
		return
	}

	var msgBuilder strings.Builder
	msgBuilder.WriteString("以下專案需要啟動 PHP-CGI 才能正常運作：\n\n")
	for _, s := range statuses {
		status := "✅ 已啟動"
		if !s.IsRunning {
			status = "❌ 未啟動"
		}
		msgBuilder.WriteString(fmt.Sprintf("  • %s → PHP %s (%s)\n", s.ProjectName, s.PHPVersion, status))
	}
	msgBuilder.WriteString("\n請在 Dashboard 的 PHP FastCGI 區塊啟動對應版本。")

	fyne.Do(func() {
		dialog.ShowInformation("⚠️ PHP Service Required", msgBuilder.String(), win)
	})
}

// createCaddyRow 建立 Caddy 服務列
func createCaddyRow(win fyne.Window, info scanner.ServiceInfo, refreshProjects func()) fyne.CanvasObject {
	statusLabel := widget.NewLabel("Stopped")
	uptimeData := binding.NewString()
	uptimeData.Set("")
	uptimeLabel := widget.NewLabelWithData(uptimeData)

	var actionBtn *widget.Button
	var reloadBtn *widget.Button

	reloadBtn = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		/*if err := checkSSLCerts(); err != nil {
			dialog.ShowError(err, win)
			addErrorLog("caddy", "SSL 檢查失敗", err)
			return
		}
		if err := generateCaddyfiles(); err != nil {
			addErrorLog("caddy", "生成配置失敗", err)
			return
		}*/
		if err := procMgr.ReloadCaddy(info.ExePath); err != nil {
			addErrorLog("caddy", "重載 Caddy 失敗", err)
		} else {
			triggerHostsUpdate()
		}
	})
	reloadBtn.Disable() // 初始禁用

	actionBtn = widget.NewButton("Start", func() {
		if !procMgr.IsRunning("caddy") {
			blocked := port.CheckPorts([]port.PortInfo{
				{"Caddy", 80},
				{"Caddy", 443},
			})
			if len(blocked) > 0 {
				for _, p := range blocked {
					addErrorLog("caddy", fmt.Sprintf("通訊埠 %d 被佔用，無法啟動 Caddy", p.Port), nil)
				}
				return
			}

			if err := checkSSLCerts(); err != nil {
				dialog.ShowError(err, win)
				addErrorLog("caddy", "SSL 檢查失敗", err)
				return
			}
			if err := generateCaddyfiles(); err != nil {
				addErrorLog("caddy", "生成配置失敗", err)
				return
			}
			if err := procMgr.StartCaddy(info.Version, info.ExePath); err != nil {
				addErrorLog("caddy", "啟動 Caddy 失敗", err)
				return
			}
			triggerHostsUpdate()
			pids := procMgr.GetPIDs("caddy")
			if len(pids) > 0 {
				statusLabel.SetText(fmt.Sprintf("Running (PID: %d)", pids[0]))
			}
			actionBtn.SetText("Stop")
			reloadBtn.Enable()
			monitorUptime("caddy", uptimeData)
			checkPHPForProjects(win)
			saveLastServiceState()
		} else {
			if err := procMgr.StopCaddy(); err != nil {
				addErrorLog("caddy", "停止 Caddy 失敗", err)
				return
			}
			statusLabel.SetText("Stopped")
			actionBtn.SetText("Start")
			reloadBtn.Disable()
			uptimeData.Set("")
			saveLastServiceState()
		}
	})

	if procMgr.IsRunning("caddy") {
		pids := procMgr.GetPIDs("caddy")
		if len(pids) > 0 {
			statusLabel.SetText(fmt.Sprintf("Running (PID: %d)", pids[0]))
		}
		actionBtn.SetText("Stop")
		reloadBtn.Enable()
		monitorUptime("caddy", uptimeData)
	}

	// 初始化主題包裝器 (使用閉包檢查按鈕文字)
	actionBtnWrapper := container.NewThemeOverride(actionBtn, &coloredButtonTheme{
		Theme:  theme.DefaultTheme(),
		isStop: func() bool { return actionBtn.Text == "Stop" },
	})

	// 更新按鈕點擊後的顏色切換邏輯 (SetText 會觸發 Refresh)
	originalCallback := actionBtn.OnTapped
	actionBtn.OnTapped = func() {
		originalCallback()
		actionBtn.Refresh()
	}

	actionGroup := container.NewBorder(nil, nil, nil, reloadBtn, actionBtnWrapper)

	return container.NewGridWithColumns(6,
		widget.NewLabel("Caddy"),
		widget.NewLabel(info.Version),
		statusLabel,
		uptimeLabel,
		widget.NewLabel("80, 443"),
		actionGroup,
	)
}

// createMariaDBRow 建立 MariaDB 服務列
func createMariaDBRow(info scanner.ServiceInfo) fyne.CanvasObject {
	statusLabel := widget.NewLabel("Stopped")
	uptimeData := binding.NewString()
	uptimeData.Set("")
	uptimeLabel := widget.NewLabelWithData(uptimeData)

	var actionBtn *widget.Button
	serviceKey := process.MariaDBServiceKey(info.Version)

	actionBtn = widget.NewButton("Start", func() {
		if !procMgr.IsRunning(serviceKey) {
			blocked := port.CheckPorts([]port.PortInfo{
				{"MariaDB", 3306},
			})
			if len(blocked) > 0 {
				for _, p := range blocked {
					addErrorLog("mariadb", fmt.Sprintf("通訊埠 %d 被佔用，無法啟動 MariaDB", p.Port), nil)
				}
				return
			}

			if err := procMgr.StartMariaDB(info.Version); err != nil {
				addErrorLog("mariadb", "啟動 MariaDB 失敗", err)
				return
			}
			pids := procMgr.GetPIDs(serviceKey)
			if len(pids) > 0 {
				statusLabel.SetText(fmt.Sprintf("Running (PID: %d)", pids[0]))
			}
			actionBtn.SetText("Stop")
			monitorUptime(serviceKey, uptimeData)
			saveLastServiceState()
		} else {
			if err := procMgr.StopMariaDB(info.Version); err != nil {
				addErrorLog("mariadb", "停止 MariaDB 失敗", err)
				return
			}
			statusLabel.SetText("Stopped")
			actionBtn.SetText("Start")
			uptimeData.Set("")
			saveLastServiceState()
		}
	})

	if procMgr.IsRunning(serviceKey) {
		pids := procMgr.GetPIDs(serviceKey)
		if len(pids) > 0 {
			statusLabel.SetText(fmt.Sprintf("Running (PID: %d)", pids[0]))
		}
		actionBtn.SetText("Stop")
		monitorUptime(serviceKey, uptimeData)
	}

	actionBtnWrapper := container.NewThemeOverride(actionBtn, &coloredButtonTheme{
		Theme:  theme.DefaultTheme(),
		isStop: func() bool { return actionBtn.Text == "Stop" },
	})
	originalCallback := actionBtn.OnTapped
	actionBtn.OnTapped = func() {
		originalCallback()
		actionBtn.Refresh()
	}

	return container.NewGridWithColumns(6,
		widget.NewLabel("MariaDB"),
		widget.NewLabel(info.Version),
		statusLabel,
		uptimeLabel,
		widget.NewLabel("3306"),
		actionBtnWrapper,
	)
}

// createPHPRow 建立 PHP-CGI 服務列
func createPHPRow(info scanner.PHPVersionInfo) fyne.CanvasObject {
	statusLabel := widget.NewLabel("Stopped")
	uptimeData := binding.NewString()
	uptimeData.Set("")
	uptimeLabel := widget.NewLabelWithData(uptimeData)

	options := []string{"3", "10"}
	for i := 20; i <= 100; i += 10 {
		options = append(options, fmt.Sprintf("%d", i))
	}
	currentCount := appCfg.Global.PHP.ProcessesPerVersion
	if c, ok := appCfg.Global.PHP.Processes[info.MajorMin]; ok {
		currentCount = c
	}

	found := false
	currentStr := fmt.Sprintf("%d", currentCount)
	for _, opt := range options {
		if opt == currentStr {
			found = true
			break
		}
	}
	if !found {
		options = append(options, currentStr)
		sort.Slice(options, func(i, j int) bool {
			var a, b int
			fmt.Sscanf(options[i], "%d", &a)
			fmt.Sscanf(options[j], "%d", &b)
			return a < b
		})
	}

	processSelect := widget.NewSelect(options, nil)
	processSelect.SetSelected(fmt.Sprintf("%d", currentCount))
	processSelect.OnChanged = func(val string) {
		count := 3
		fmt.Sscanf(val, "%d", &count)

		old, ok := appCfg.Global.PHP.Processes[info.MajorMin]
		if ok && old == count {
			return
		}
		if !ok && count == appCfg.Global.PHP.ProcessesPerVersion {
			return
		}

		appCfg.Global.PHP.Processes[info.MajorMin] = count
		cfgPath := filepath.Join(baseDir, "conf", "wincmp.json")
		appCfg.Save(cfgPath)

		addLog("php", fmt.Sprintf("PHP %s 進程數變更為 %d (重啟後生效)", info.MajorMin, count))

		if procMgr.IsRunning("caddy") {
			addLog("caddy", "PHP 進程配置已更變，請重載 (Reload) 或重啟 Caddy 以套用新端口")
		}
	}

	var actionBtn *widget.Button
	serviceKey := process.PHPServiceKey(info.Version)

	actionBtn = widget.NewButton("Start", func() {
		if !procMgr.IsRunning(serviceKey) {
			ports := info.GetPHPPorts()
			var portInfos []port.PortInfo
			for _, p := range ports {
				portInfos = append(portInfos, port.PortInfo{
					Service: fmt.Sprintf("PHP-%s", info.MajorMin),
					Port:    p,
				})
			}
			blocked := port.CheckPorts(portInfos)
			if len(blocked) > 0 {
				for _, p := range blocked {
					addErrorLog("php", fmt.Sprintf("通訊埠 %d 被佔用，無法啟動 PHP-CGI %s", p.Port, info.Version), nil)
				}
				return
			}

			if count, ok := appCfg.Global.PHP.Processes[info.MajorMin]; ok {
				info.PortCount = count
			} else {
				info.PortCount = appCfg.Global.PHP.ProcessesPerVersion
			}

			if err := procMgr.StartPHPCGI(info); err != nil {
				addErrorLog("php", "啟動 PHP-CGI 失敗", err)
				return
			}
			pids := procMgr.GetPIDs(serviceKey)
			statusLabel.SetText(fmt.Sprintf("Running (%d PIDs)", len(pids)))
			actionBtn.SetText("Stop")
			processSelect.Disable()
			monitorUptime(serviceKey, uptimeData)
			saveLastServiceState()
		} else {
			if err := procMgr.StopPHPCGI(info.Version); err != nil {
				addErrorLog("php", "停止 PHP-CGI 失敗", err)
				return
			}
			statusLabel.SetText("Stopped")
			actionBtn.SetText("Start")
			processSelect.Enable()
			uptimeData.Set("")
			saveLastServiceState()
		}
	})

	if procMgr.IsRunning(serviceKey) {
		pids := procMgr.GetPIDs(serviceKey)
		statusLabel.SetText(fmt.Sprintf("Running (%d PIDs)", len(pids)))
		actionBtn.SetText("Stop")
		processSelect.Disable()
		monitorUptime(serviceKey, uptimeData)
	}

	actionBtnWrapper := container.NewThemeOverride(actionBtn, &coloredButtonTheme{
		Theme:  theme.DefaultTheme(),
		isStop: func() bool { return actionBtn.Text == "Stop" },
	})
	originalCallback := actionBtn.OnTapped
	actionBtn.OnTapped = func() {
		originalCallback()
		actionBtn.Refresh()
	}

	return container.NewGridWithColumns(6,
		widget.NewLabel("PHP-CGI"),
		widget.NewLabel(info.Version),
		statusLabel,
		uptimeLabel,
		container.NewHBox(widget.NewLabel("Proc:"), processSelect),
		actionBtnWrapper,
	)
}

// ===== 2. 網頁專案管理 =====

// showProjectEditor 顯示專案編輯器
func showProjectEditor(win fyne.Window, proj *config.ProjectConfig, onSave func()) {
	// --- 基本設定 ---
	nameEntry := widget.NewEntry()
	nameEntry.SetText(proj.Name)

	// 使用多行輸入框解決捲軸擋住文字的問題
	domainsEntry := widget.NewMultiLineEntry()
	domainsEntry.SetMinRowsVisible(3)
	domainsEntry.Wrapping = fyne.TextWrapWord
	domainsEntry.SetText(strings.Join(proj.Domains, ", "))
	domainsEntry.PlaceHolder = "e.g. local-project.test, www.project.test"

	rootPathEntry := widget.NewEntry()
	rootPathEntry.SetText(proj.RootPath)
	rootBrowse := widget.NewButtonWithIcon("Browse", theme.FolderOpenIcon(), func() {
		openZenitySelector(
			win,
			rootPathEntry.Text, // 讀取當前輸入框的值
			baseDir,
			true, // 選取目錄
			func(path string) { rootPathEntry.SetText(path) },
			zenity.Title("Select Project Root"),
		)
	})

	phpVersions := []string{"None (Static)"} // 提示空值含義
	for _, p := range scanRes.PHPList {
		phpVersions = append(phpVersions, p.MajorMin)
	}
	phpSelect := widget.NewSelect(phpVersions, nil)
	if proj.PHPVersion == "" {
		phpSelect.SetSelected("None (Static)")
	} else {
		phpSelect.SetSelected(proj.PHPVersion)
	}

	basicForm := widget.NewForm(
		widget.NewFormItem("Project Name", nameEntry),
		widget.NewFormItem("Domains (CSV)", domainsEntry),
		widget.NewFormItem("Root Path", container.NewBorder(nil, nil, nil, rootBrowse, rootPathEntry)),
		widget.NewFormItem("PHP Version", container.NewVBox(
			phpSelect,
			widget.NewLabelWithStyle("(Empty or 'None' means PHP is disabled for this project)", fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
		)),
	)

	// --- 狀態 ---
	enabledCheck := widget.NewCheck("Project Enabled", nil)
	enabledCheck.Checked = proj.Enabled

	typeSelect := widget.NewSelect([]string{"None (Static)", "Laravel"}, nil)
	if proj.Type == "laravel" {
		typeSelect.SetSelected("Laravel")
	} else {
		typeSelect.SetSelected("None (Static)")
	}

	// --- 進階設定 ---
	useSSLCheck := widget.NewCheck("Enable SSL", nil)
	useSSLCheck.Checked = proj.UseSSL

	sslCrtEntry := widget.NewEntry()
	sslCrtEntry.SetText(proj.SSLCrt)
	crtBrowse := widget.NewButtonWithIcon("Browse", theme.FileIcon(), func() {
		openZenitySelector(
			win,
			sslCrtEntry.Text,
			filepath.Join(baseDir, "conf", "ssl")+string(os.PathSeparator),
			false, // 選取檔案
			func(path string) { sslCrtEntry.SetText(path) },
			zenity.Title("Select SSL Certificate"),
			zenity.FileFilters{{Name: "Certificate Files", Patterns: []string{"*.crt", "*.pem", "*.cert"}}},
		)
	})

	sslKeyEntry := widget.NewEntry()
	sslKeyEntry.SetText(proj.SSLKey)
	keyBrowse := widget.NewButtonWithIcon("Browse", theme.FileIcon(), func() {
		openZenitySelector(
			win,
			sslKeyEntry.Text,
			filepath.Join(baseDir, "conf", "ssl")+string(os.PathSeparator),
			false, // 選取檔案
			func(path string) { sslKeyEntry.SetText(path) },
			zenity.Title("Select SSL Key"),
			zenity.FileFilters{{Name: "Key Files", Patterns: []string{"*.key", "*.pem"}}},
		)
	})

	advTitle := widget.NewLabelWithStyle("Advanced Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	advForm := widget.NewForm(
		widget.NewFormItem("SSL Toggle", useSSLCheck),
		widget.NewFormItem("SSL CRT", container.NewBorder(nil, nil, nil, crtBrowse, sslCrtEntry)),
		widget.NewFormItem("SSL KEY", container.NewBorder(nil, nil, nil, keyBrowse, sslKeyEntry)),
	)

	openDirBtn := widget.NewButtonWithIcon("Open Project Directory", theme.FolderIcon(), func() {
		openLocalPath(rootPathEntry.Text)
	})
	openCaddyfileBtn := widget.NewButtonWithIcon("Open Caddyfile", theme.DocumentIcon(), func() {
		openLocalPath(filepath.Join(baseDir, "conf", "sites", proj.Name+".caddy"))
	})

	actionRow := container.NewHBox(openDirBtn, openCaddyfileBtn)

	// 組合佈局
	content := container.NewVBox(
		actionRow,
		widget.NewSeparator(),
		basicForm,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Availability & Type", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		enabledCheck,
		container.NewHBox(widget.NewLabel("Project Type: "), typeSelect),
		widget.NewSeparator(),
		advTitle,
		advForm,
	)

	d := dialog.NewCustomConfirm("Edit Project", "Save", "Cancel", container.NewVScroll(content), func(save bool) {
		if save {
			proj.Name = nameEntry.Text
			rawDomains := domainsEntry.Text
			// 處理換行或逗號分隔
			rawDomains = strings.ReplaceAll(rawDomains, "\n", ",")
			doms := strings.Split(rawDomains, ",")
			var finalDomains []string
			for _, d := range doms {
				trimmed := strings.TrimSpace(d)
				if trimmed != "" {
					finalDomains = append(finalDomains, trimmed)
				}
			}
			proj.Domains = finalDomains

			if phpSelect.Selected == "None (Static)" {
				proj.PHPVersion = ""
			} else {
				proj.PHPVersion = phpSelect.Selected
			}

			proj.RootPath = rootPathEntry.Text
			proj.UseSSL = useSSLCheck.Checked
			proj.SSLCrt = sslCrtEntry.Text
			proj.SSLKey = sslKeyEntry.Text
			proj.Enabled = enabledCheck.Checked

			if typeSelect.Selected == "Laravel" {
				proj.Type = "laravel"
			} else {
				proj.Type = ""
			}

			onSave()
		}
	}, win)

	// 增加預設尺寸以便輸入長路徑
	d.Resize(fyne.NewSize(650, 520))
	d.Show()
}

func createProjectsTab(win fyne.Window) fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Web Projects", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// 定義固定寬度的透明矩形來撐開欄位
	projectRect := canvas.NewRectangle(color.Transparent)
	projectRect.SetMinSize(fyne.NewSize(120, 0))
	projectH := container.NewStack(widget.NewLabelWithStyle("Project", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), projectRect)

	availabilityRect := canvas.NewRectangle(color.Transparent)
	availabilityRect.SetMinSize(fyne.NewSize(120, 0))
	availabilityH := container.NewStack(widget.NewLabelWithStyle("Availability", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), availabilityRect)

	stateRect := canvas.NewRectangle(color.Transparent)
	stateRect.SetMinSize(fyne.NewSize(80, 0))
	stateH := container.NewStack(widget.NewLabelWithStyle("State", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), stateRect)

	header := container.NewBorder(nil, nil,
		container.NewHBox(projectH, availabilityH, stateH),
		nil,
		container.NewGridWithColumns(2,
			widget.NewLabelWithStyle("Domains", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabelWithStyle("Root Path", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		),
	)

	actionH := widget.NewLabelWithStyle("Action", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	// 使用 Container 包裝 Action 並設定最小寬度以對齊下方的兩個按鈕 (約 70-80px)
	actionRect := canvas.NewRectangle(color.Transparent)
	actionRect.SetMinSize(fyne.NewSize(76, 0))
	actionHStack := container.NewStack(actionH, actionRect)

	headerContainer := container.NewBorder(nil, nil, nil, actionHStack, header)

	var list *widget.List
	list = widget.NewList(
		func() int { return len(appCfg.Projects) },
		func() fyne.CanvasObject {
			btns := container.NewHBox(
				widget.NewButtonWithIcon("", theme.SettingsIcon(), nil),
				widget.NewButtonWithIcon("", theme.DeleteIcon(), nil),
			)

			projectBox := container.NewStack()
			availabilityBox := container.NewStack()
			stateBox := container.NewStack()
			domainsBox := container.NewStack()
			pathBox := container.NewStack()

			leftHBox := container.NewHBox(projectBox, availabilityBox, stateBox)
			centerGrid := container.NewGridWithColumns(2, domainsBox, pathBox)

			dataFields := container.NewBorder(nil, nil, leftHBox, nil, centerGrid)

			return container.NewBorder(nil, nil, nil, btns, dataFields)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if int(i) >= len(appCfg.Projects) {
				return
			}
			proj := &appCfg.Projects[i]

			border := o.(*fyne.Container)
			dataFields := border.Objects[0].(*fyne.Container)
			btns := border.Objects[1].(*fyne.Container)

			centerGrid := dataFields.Objects[0].(*fyne.Container)
			leftHBox := dataFields.Objects[1].(*fyne.Container)

			projectBox := leftHBox.Objects[0].(*fyne.Container)
			availabilityBox := leftHBox.Objects[1].(*fyne.Container)
			stateBox := leftHBox.Objects[2].(*fyne.Container)

			domainsBox := centerGrid.Objects[0].(*fyne.Container)
			pathBox := centerGrid.Objects[1].(*fyne.Container)

			// Project Name (Hover 顯示完整名稱)
			projectNameHover := ttwidget.NewLabel(proj.Name)
			projectNameHover.SetToolTip(proj.Name)
			projectNameHover.TextStyle = fyne.TextStyle{Bold: true}
			projectNameHover.Truncation = fyne.TextTruncateEllipsis
			nameRect := canvas.NewRectangle(color.Transparent)
			nameRect.SetMinSize(fyne.NewSize(120, 0))
			projectBox.Objects = []fyne.CanvasObject{container.NewStack(projectNameHover, nameRect)}
			projectBox.Refresh()

			// Availability
			availText := "Disabled"
			availColor := color.NRGBA{R: 244, G: 67, B: 54, A: 255} // Red
			if proj.Enabled {
				availText = "Enabled"
				availColor = color.NRGBA{R: 76, G: 175, B: 80, A: 255} // Green
			}
			availLabel := canvas.NewText(availText, availColor)
			availLabel.TextStyle = fyne.TextStyle{Bold: true}

			availBoxRect := canvas.NewRectangle(color.Transparent)
			availBoxRect.SetMinSize(fyne.NewSize(120, 0))
			availabilityBox.Objects = []fyne.CanvasObject{container.NewStack(availBoxRect, container.NewHBox(availLabel))}
			availabilityBox.Refresh()

			// State (Running/Stopped)
			// 判定：可用 Caddy config 中存在 且 Caddy 正在運行
			caddyRunning := procMgr.IsRunning("caddy")
			caddyConfigPath := filepath.Join(baseDir, "conf", "sites", proj.Name+".caddy")
			configExists := false
			if _, err := os.Stat(caddyConfigPath); err == nil {
				configExists = true
			}

			stateText := "Stopped"
			var stateColor color.Color = color.NRGBA{R: 158, G: 158, B: 158, A: 255} // Light Gray
			if caddyRunning && configExists {
				stateText = "Running"
				stateColor = theme.ForegroundColor() // Normal color
			}
			stateLabel := canvas.NewText(stateText, stateColor)
			stateLabel.TextStyle = fyne.TextStyle{Bold: true}

			stateBoxRect := canvas.NewRectangle(color.Transparent)
			stateBoxRect.SetMinSize(fyne.NewSize(80, 0))
			stateBox.Objects = []fyne.CanvasObject{container.NewStack(stateBoxRect, container.NewHBox(stateLabel))}
			stateBox.Refresh()

			// Domains & Path
			domainsStr := strings.Join(proj.Domains, ", ")
			hoverDomains := strings.Join(proj.Domains, ",\n")
			domainsHover := ttwidget.NewLabel(domainsStr)
			domainsHover.SetToolTip(hoverDomains)
			domainsHover.Truncation = fyne.TextTruncateEllipsis

			copyBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
				scheme := "http://"
				if proj.UseSSL {
					scheme = "https://"
				}
				domain := ""
				if len(proj.Domains) > 0 {
					domain = proj.Domains[0]
				} else {
					domain = GenerateValidDomain(proj.Name)
				}
				url := scheme + domain
				win.Clipboard().SetContent(url)
				addLog("system", fmt.Sprintf("📋 已複製網址到剪貼簿: %s", url))
			})

			domainsBox.Objects = []fyne.CanvasObject{container.NewBorder(nil, nil, nil, copyBtn, domainsHover)}
			domainsBox.Refresh()

			rootPath := appCfg.GetProjectRoot(*proj, baseDir)
			pathHover := ttwidget.NewLabel(rootPath)
			pathHover.SetToolTip(rootPath)
			pathHover.Truncation = fyne.TextTruncateEllipsis
			pathBox.Objects = []fyne.CanvasObject{pathHover}
			pathBox.Refresh()

			btns.Objects[0].(*widget.Button).OnTapped = func() {
				showProjectEditor(win, proj, func() {
					appCfg.Save(filepath.Join(baseDir, "conf", "wincmp.json"))
					generateCaddyfiles()
					list.Refresh()
					addLog("system", fmt.Sprintf("✅ 已更新專案: %s", proj.Name))
				})
			}
			btns.Objects[1].(*widget.Button).OnTapped = func() {
				dialog.ShowConfirm("刪除專案", fmt.Sprintf("確定要從清單移除 %s 嗎？\n(不會刪除實際檔案)", proj.Name), func(b bool) {
					if b {
						appCfg.Projects = append(appCfg.Projects[:i], appCfg.Projects[i+1:]...)
						appCfg.Save(filepath.Join(baseDir, "conf", "wincmp.json"))
						generateCaddyfiles()
						list.Refresh()
						addLog("system", fmt.Sprintf("✅ 已移除專案: %s", proj.Name))
					}
				}, win)
			}
		},
	)

	addBtn := widget.NewButtonWithIcon("Add Project", theme.ContentAddIcon(), func() {
		openZenitySelector(
			win,
			"", // 新增沒有預設路徑，直接用 fallback
			baseDir,
			true,
			func(path string) {
				name := filepath.Base(path)
				for _, p := range appCfg.Projects {
					if p.Name == name {
						dialog.ShowError(fmt.Errorf("專案 %s 已存在", name), win)
						return
					}
				}

				res := detect.DetectLaravel(path)
				projectType := ""
				phpVersion := ""
				if res.IsLaravel {
					projectType = "laravel"
					phpVersion = "8.2"
				}

				newProj := config.ProjectConfig{
					Name:       name,
					Domains:    []string{GenerateValidDomain(name)},
					PHPVersion: phpVersion,
					Type:       projectType,
					RootPath:   path,
					UseSSL:     true,
					Enabled:    false,
				}

				appCfg.Projects = append(appCfg.Projects, newProj)
				appCfg.Save(filepath.Join(baseDir, "conf", "wincmp.json"))
				generateCaddyfiles()

				// 更新 UI
				list.Refresh()
				addLog("system", fmt.Sprintf("📌 已新增專案: %s", name))
				if res.IsLaravel {
					addLog("system", fmt.Sprintf("  ↳ 偵測為 Laravel (Confidence: %d, Reasons: %s)", res.Confidence, strings.Join(res.Reasons, ", ")))
				}
				dialog.ShowInformation("成功", fmt.Sprintf("專案 %s 已加入。\n請在 Dashboard 點擊 Reload Caddy 生效。", name), win)
			},
			zenity.Title("Select Project Folder"),
		)
	})

	syncBtn := widget.NewButtonWithIcon("Re-gen Caddy", theme.ViewRefreshIcon(), func() {
		dialog.ShowConfirm("重整 Caddy 設定", "您確定要根據 wincmp.json 重新整理所有 Caddy 設定檔嗎？\n\n這將會移除已停用或不存在專案的設定。", func(confirm bool) {
			if confirm {
				if err := generateCaddyfiles(); err != nil {
					addErrorLog("system", "Re-gen Caddy 失敗", err)
					dialog.ShowError(err, win)
					return
				}
				if procMgr.IsRunning("caddy") {
					exePath := procMgr.GetExePath("caddy")
					if err := procMgr.ReloadCaddy(exePath); err != nil {
						addErrorLog("system", "Reload Caddy 失敗", err)
						dialog.ShowError(err, win)
					} else {
						addLog("system", "✅ Caddy 重整配置並載入完成")
						triggerHostsUpdate()
					}
				} else {
					addLog("system", "✅ Caddy 設定檔已重整 (Caddy 目前未啟動)")
				}
			}
		}, win)
	})
	scanBtn := widget.NewButtonWithIcon("Scan WWW", theme.SearchIcon(), func() {
		if appCfg.Global.DefaultWWW == "" {
			dialog.ShowInformation("提示", "尚未設定預設 WWW 目錄，請至 Settings 頁面設定。", win)
			return
		}

		// 取得完整路徑以顯示給用戶看 (邏輯與 scanDefaultWWW 一致)
		var wwwDir string
		if filepath.IsAbs(appCfg.Global.DefaultWWW) {
			wwwDir = appCfg.Global.DefaultWWW
		} else {
			wwwDir = filepath.Join(baseDir, appCfg.Global.DefaultWWW)
		}

		msg := fmt.Sprintf("確定要自動掃描預設目錄嗎？\n\n路徑：%s\n\n系統將會嘗試將此目錄下的所有「子資料夾」加入為網頁專案清單。\n(已存在的專案將不會重複加入)", wwwDir)

		dialog.ShowConfirm("掃描確認", msg, func(confirm bool) {
			if confirm {
				scanDefaultWWW()
				list.Refresh()
				addLog("system", "🔍 手動掃描預設目錄完成")
			}
		}, win)
	})

	// 加上提示
	scanBtnWrap := container.NewHBox(syncBtn, scanBtn, addBtn)

	topBar := container.NewBorder(nil, nil, nil, scanBtnWrap, title)
	content := container.NewBorder(container.NewVBox(topBar, headerContainer), nil, nil, nil, list)
	return content
}

// ===== 3. 簡易資料庫檢視器 (Database Explorer) =====

func createDatabaseExplorerTab() fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Database Explorer", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// 狀態提示標籤
	statusLabel := widget.NewLabel("")
	statusLabel.Wrapping = fyne.TextWrapWord

	// 右側內容區域（資料表列表）
	tableListData := binding.NewStringList()
	tableList := widget.NewListWithData(
		tableListData,
		func() fyne.CanvasObject {
			return widget.NewLabel("table_name")
		},
		func(item binding.DataItem, o fyne.CanvasObject) {
			val, _ := item.(binding.String).Get()
			o.(*widget.Label).SetText(val)
		},
	)

	tableHeader := widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	rightPanel := container.NewBorder(tableHeader, nil, nil, nil, tableList)

	// 左側 Schema 列表
	schemaListData := binding.NewStringList()
	schemaList := widget.NewListWithData(
		schemaListData,
		func() fyne.CanvasObject {
			return widget.NewLabel("schema_name")
		},
		func(item binding.DataItem, o fyne.CanvasObject) {
			val, _ := item.(binding.String).Get()
			o.(*widget.Label).SetText(val)
		},
	)

	// 主要瀏覽區域（Split）
	split := container.NewHSplit(schemaList, container.NewPadded(rightPanel))
	split.SetOffset(0.3)

	// --- 1. 定義資料庫查詢與更新邏輯 (提前定義以解決 Closure 引用問題) ---

	// queryDatabases 查詢所有 Schema
	queryDatabases := func() ([]string, error) {
		db, err := sql.Open("mysql", "root@tcp(127.0.0.1:3306)/")
		if err != nil {
			return nil, err
		}
		defer db.Close()
		db.SetConnMaxLifetime(5 * time.Second)
		db.SetMaxOpenConns(1)

		rows, err := db.Query("SHOW DATABASES")
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var databases []string
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err == nil {
				databases = append(databases, name)
			}
		}
		return databases, rows.Err()
	}

	// queryTables 查詢指定 Schema 的資料表
	queryTables := func(schema string) ([]string, error) {
		db, err := sql.Open("mysql", "root@tcp(127.0.0.1:3306)/"+schema)
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

		var tables []string
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

	isMariaDBRunning := func() bool {
		for _, info := range scanRes.MariaDBList {
			if procMgr.IsRunning(process.MariaDBServiceKey(info.Version)) {
				return true
			}
		}
		return false
	}

	// 先宣告控制變數，以便在 refreshUI 中存取
	notRunningMsg := widget.NewLabel("請先至 Dashboard 頁面啟動 MariaDB 服務，\n再使用 Database Explorer。")
	notRunningMsg.Wrapping = fyne.TextWrapWord
	notRunningMsg.Alignment = fyne.TextAlignCenter
	var notRunningBox *fyne.Container

	refreshUI := func() {
		if !isMariaDBRunning() {
			split.Hide()
			if notRunningBox != nil {
				notRunningBox.Show()
			}
			//statusLabel.SetText("MariaDB 未運行")
			addLog("system", "DB Explorer: MariaDB 未運行")
			return
		}

		// MariaDB 運行中
		if notRunningBox != nil {
			notRunningBox.Hide()
		}
		split.Show()
		statusLabel.SetText("已連線")

		databases, err := queryDatabases()
		if err != nil {
			statusLabel.SetText(fmt.Sprintf("連線失敗: %v", err))
			addLog("system", fmt.Sprintf("DB Explorer: 連線失敗 - %v", err))
			split.Hide()
			notRunningMsg.SetText(fmt.Sprintf("⚠️ 無法連線到 MariaDB\n\n錯誤: %v\n\n請確認 MariaDB 已正常啟動並運行中。", err))
			if notRunningBox != nil {
				notRunningBox.Show()
			}
			return
		}

		schemaListData.Set(databases)
		tableHeader.SetText("選擇左側的資料庫以檢視資料表")
		tableListData.Set([]string{})
		addLog("system", fmt.Sprintf("DB Explorer: 已載入 %d 個資料庫", len(databases)))
	}

	// --- 2. 建立連線提示 UI (優化佈局) ---
	notRunningIcon := widget.NewIcon(theme.WarningIcon())
	// 使用 Container 控制 Icon 大小
	iconContainer := container.NewGridWrap(fyne.NewSize(64, 64), notRunningIcon)

	notRunningTitle := widget.NewLabelWithStyle("MariaDB 尚未啟動", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// 使用透明矩形撐開寬度，解決「直排文字」Bug
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(450, 0))

	dashboardBtn := widget.NewButtonWithIcon("前往 Dashboard", theme.HomeIcon(), func() {
		if mainTabs != nil {
			mainTabs.SelectIndex(0)
		}
	})

	notRunningBox = container.NewVBox(
		container.NewCenter(iconContainer),
		notRunningTitle,
		container.NewStack(spacer, notRunningMsg),
		layout.NewSpacer(),
		container.NewCenter(container.NewHBox(
			widget.NewButtonWithIcon("重新整理狀態", theme.ViewRefreshIcon(), func() {
				notRunningMsg.SetText("請先至 Dashboard 頁面啟動 MariaDB 服務，\n再使用 Database Explorer。")
				refreshUI()
			}),
			dashboardBtn,
		)),
	)

	// 使用 Stack 在連線提示和 split 之間切換
	contentStack := container.NewStack(split, container.NewCenter(notRunningBox))

	// 選擇 Schema 時查詢資料表
	schemaList.OnSelected = func(id widget.ListItemID) {
		schemas, _ := schemaListData.Get()
		if id >= len(schemas) {
			return
		}
		selectedSchema := schemas[id]
		tableHeader.SetText(fmt.Sprintf("資料庫 '%s' 的資料表：", selectedSchema))
		tables, err := queryTables(selectedSchema)
		if err != nil {
			tableListData.Set([]string{fmt.Sprintf("查詢失敗: %v", err)})
			return
		}
		if len(tables) == 0 {
			tableListData.Set([]string{"（此資料庫沒有資料表）"})
		} else {
			tableListData.Set(tables)
		}
	}

	// 頂部按鈕列
	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		notRunningMsg.SetText("請先至 Dashboard 頁面啟動 MariaDB 服務，\n再使用 Database Explorer。")
		refreshUI()
	})

	heidiSQLBtn := widget.NewButtonWithIcon("Open in HeidiSQL", theme.ComputerIcon(), func() {
		heidiPath := scanRes.HeidiSQLPath
		if heidiPath == "" {
			addLog("system", "DB Explorer: 找不到 HeidiSQL 執行檔")
			return
		}
		cmd := exec.Command(heidiPath, "-h=127.0.0.1", "-P=3306", "-u=root")
		if err := cmd.Start(); err != nil {
			addErrorLog("system", "啟動 HeidiSQL 失敗", err)
			return
		}
		go cmd.Wait()
		addLog("system", "DB Explorer: 已啟動 HeidiSQL")
	})

	toolbar := container.NewHBox(refreshBtn, heidiSQLBtn)
	topBar := container.NewBorder(nil, nil, nil, toolbar, title)

	// 初始化刷新
	go func() {
		time.Sleep(300 * time.Millisecond)
		refreshUI()
	}()

	return container.NewBorder(container.NewVBox(topBar, statusLabel), nil, nil, nil, contentStack)
}

func applyTheme(themeName string) {
	appCfg.Global.Theme = themeName
	switch themeName {
	case "Light":
		myApp.Settings().SetTheme(theme.LightTheme())
	case "Dark":
		myApp.Settings().SetTheme(theme.DarkTheme())
	case "Dark Blue":
		myApp.Settings().SetTheme(createDarkTheme("blue"))
	case "Dark Gray":
		myApp.Settings().SetTheme(createDarkTheme("gray"))
	case "Twilight":
		myApp.Settings().SetTheme(createDarkTheme("twilight"))
	default:
		myApp.Settings().SetTheme(theme.DefaultTheme())
	}
}

// ===== 4. 全域設定 (Settings) =====

func createSettingsTab(win fyne.Window) fyne.CanvasObject {
	title := widget.NewLabelWithStyle("WinCMP Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	hint := canvas.NewText(" (Changes will be automatically saved immediately with debouncing)", color.NRGBA{R: 128, G: 128, B: 128, A: 255})
	hint.TextSize = 11

	header := container.NewHBox(
		title,
		hint,
	)

	// --- 1. Basic Settings 組件聲明 ---
	wwwDirEntry := widget.NewEntry()
	wwwDirEntry.SetText(appCfg.Global.DefaultWWW)
	wwwDirBox := container.NewBorder(nil, nil, nil, widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		openZenitySelector(
			win,
			wwwDirEntry.Text,
			baseDir,
			true,
			func(path string) { wwwDirEntry.SetText(path) },
			zenity.Title("Select Default WWW Directory"),
		)
	}), wwwDirEntry)

	sslDirEntry := widget.NewEntry()
	sslDirEntry.SetText(appCfg.Global.DefaultSSL)
	sslDirBox := container.NewBorder(nil, nil, nil, widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		openZenitySelector(
			win,
			sslDirEntry.Text,
			baseDir,
			true,
			func(path string) { sslDirEntry.SetText(path) },
			zenity.Title("Select Default SSL Directory"),
		)
	}), sslDirEntry)

	// --- 2. System Settings 組件聲明 ---
	restoreLastStateCheck := widget.NewCheck("Restore Previous Session", nil)
	restoreLastStateCheck.Checked = appCfg.Global.RestoreLastState

	minToTrayCheck := widget.NewCheck("Minimize to System Tray on Close", nil)
	minToTrayCheck.Checked = appCfg.Global.MinimizeToTray

	runOnBootCheck := widget.NewCheck("Run WinCMP on System Startup", nil)
	runOnBootCheck.Checked = appCfg.Global.RunOnBoot

	autoUpdateHostsCheck := widget.NewCheck("Auto Update System Hosts File", nil)
	autoUpdateHostsCheck.Checked = appCfg.Global.AutoUpdateHosts

	// --- 3. Log Settings 組件聲明 ---
	maxLogEntry := widget.NewEntry()
	maxLogEntry.SetText(fmt.Sprintf("%d", appCfg.Global.MaxLogRetention))
	if appCfg.Global.MaxLogRetention == 0 {
		maxLogEntry.SetText("30") // 預設 30 天
	}

	// --- 自動保存邏輯 (含防抖動與數值對比) ---
	var saveTimer *time.Timer
	var mu sync.Mutex

	debouncedSave := func(settingName string, oldVal, newVal interface{}) {
		mu.Lock()
		if saveTimer != nil {
			saveTimer.Stop()
		}

		saveTimer = time.AfterFunc(800*time.Millisecond, func() {
			appCfg.Global.DefaultWWW = wwwDirEntry.Text
			appCfg.Global.DefaultSSL = sslDirEntry.Text
			appCfg.Global.RestoreLastState = restoreLastStateCheck.Checked
			appCfg.Global.AutoUpdateHosts = autoUpdateHostsCheck.Checked
			appCfg.Global.MinimizeToTray = minToTrayCheck.Checked
			appCfg.Global.RunOnBoot = runOnBootCheck.Checked

			var days int
			fmt.Sscanf(maxLogEntry.Text, "%d", &days)
			appCfg.Global.MaxLogRetention = days

			cfgPath := filepath.Join(baseDir, "conf", "wincmp.json")
			if err := appCfg.Save(cfgPath); err != nil {
				addErrorLog("system", "自動儲存設定失敗", err)
			} else {
				addLog("system", fmt.Sprintf("⚙️ %s: [%v] ➔ [%v] (Auto Saved)", settingName, oldVal, newVal))
			}
		})
		mu.Unlock()
	}

	// --- 2.5 Appearance Settings 組件聲明 ---
	themeOptions := []string{"Light", "Dark", "Dark Blue", "Dark Gray", "Twilight", "System"}
	themeSelect := widget.NewSelect(themeOptions, nil)
	if appCfg.Global.Theme == "" {
		themeSelect.SetSelected("System")
	} else {
		themeSelect.SetSelected(appCfg.Global.Theme)
	}

	// 綁定回調
	wwwDirEntry.OnChanged = func(s string) {
		if s != appCfg.Global.DefaultWWW {
			debouncedSave("WWW Dir", appCfg.Global.DefaultWWW, s)
		}
	}
	sslDirEntry.OnChanged = func(s string) {
		if s != appCfg.Global.DefaultSSL {
			debouncedSave("SSL Dir", appCfg.Global.DefaultSSL, s)
		}
	}
	restoreLastStateCheck.OnChanged = func(b bool) {
		if b != appCfg.Global.RestoreLastState {
			debouncedSave("Previous Session", appCfg.Global.RestoreLastState, b)
		}
	}
	minToTrayCheck.OnChanged = func(b bool) {
		if b != appCfg.Global.MinimizeToTray {
			old := appCfg.Global.MinimizeToTray
			appCfg.Global.MinimizeToTray = b // 為了讓 refreshSystemTray 讀到最新狀態，先更新記憶體
			debouncedSave("Min to Tray", old, b)
			refreshSystemTray(fyne.CurrentApp(), win)
		}
	}
	runOnBootCheck.OnChanged = func(b bool) {
		if b != appCfg.Global.RunOnBoot {
			debouncedSave("Run on Boot", appCfg.Global.RunOnBoot, b)
		}
	}
	autoUpdateHostsCheck.OnChanged = func(b bool) {
		if b != appCfg.Global.AutoUpdateHosts {
			debouncedSave("Auto Update Hosts", appCfg.Global.AutoUpdateHosts, b)
		}
	}
	maxLogEntry.OnChanged = func(s string) {
		var current int
		fmt.Sscanf(s, "%d", &current)
		if current != appCfg.Global.MaxLogRetention {
			debouncedSave("Log Retention", appCfg.Global.MaxLogRetention, current)
		}
	}
	themeSelect.OnChanged = func(selected string) {
		if selected != appCfg.Global.Theme {
			oldTheme := appCfg.Global.Theme
			if oldTheme == "" {
				oldTheme = "System"
			}
			applyTheme(selected)
			debouncedSave("Theme", oldTheme, selected)
		}
	}

	// --- 佈局組合 ---
	basicTitle := widget.NewLabelWithStyle("Basic Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	basicForm := widget.NewForm(
		widget.NewFormItem("Default WWW Dir", wwwDirBox),
		widget.NewFormItem("Default SSL Dir", sslDirBox),
	)

	sysTitle := widget.NewLabelWithStyle("System Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	// 優化為雙欄佈局，更有效利用空間
	autoForm := widget.NewForm(
		widget.NewFormItem("Automation", container.NewVBox(restoreLastStateCheck, autoUpdateHostsCheck)),
	)
	behForm := widget.NewForm(
		widget.NewFormItem("Behavior", minToTrayCheck),
		widget.NewFormItem("Startup", runOnBootCheck),
	)
	sysGrid := container.NewGridWithColumns(2, autoForm, behForm)

	appearTitle := widget.NewLabelWithStyle("Appearance", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	appearForm := widget.NewForm(
		widget.NewFormItem("Theme", themeSelect),
	)

	logTitle := widget.NewLabelWithStyle("Log Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	logForm := widget.NewForm(
		widget.NewFormItem("Retention (Days)", maxLogEntry),
	)

	hostsBtn := widget.NewButtonWithIcon("hosts", theme.DocumentIcon(), func() {
		openLocalPath("C:\\Windows\\System32\\drivers\\etc\\hosts")
	})
	phpIniBtn := widget.NewButtonWithIcon("php.ini", theme.DocumentIcon(), func() {
		openLocalPath(filepath.Join(baseDir, "conf", "php", "php.ini"))
	})
	jsonConfigBtn := widget.NewButtonWithIcon("WinCMP Config", theme.DocumentIcon(), func() {
		openLocalPath(filepath.Join(baseDir, "conf", "wincmp.json"))
	})

	configTitle := widget.NewLabelWithStyle("Config", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// 優化：為 Config 按鈕加入簡單說明
	hostsHint := canvas.NewText("(系統 Hosts)", color.NRGBA{R: 128, G: 128, B: 128, A: 255})
	hostsHint.TextSize = 10
	phpIniHint := canvas.NewText("(PHP 全域設定)", color.NRGBA{R: 128, G: 128, B: 128, A: 255})
	phpIniHint.TextSize = 10
	winCMPHint := canvas.NewText("(核心設定)", color.NRGBA{R: 128, G: 128, B: 128, A: 255})
	winCMPHint.TextSize = 10

	configBtns := container.NewHBox(
		container.NewVBox(hostsBtn, container.NewCenter(hostsHint)),
		container.NewVBox(phpIniBtn, container.NewCenter(phpIniHint)),
		container.NewVBox(jsonConfigBtn, container.NewCenter(winCMPHint)),
	)

	scrollContent := container.NewVBox(
		header,
		widget.NewSeparator(),
		configTitle,
		configBtns,
		widget.NewSeparator(),
		basicTitle,
		basicForm,
		widget.NewSeparator(),
		sysTitle,
		sysGrid,
		widget.NewSeparator(),
		appearTitle,
		appearForm,
		widget.NewSeparator(),
		logTitle,
		logForm,
		layout.NewSpacer(),
	)

	return container.NewPadded(container.NewVScroll(scrollContent))
}

// refreshSystemTray 根據設定更新系統匣選單 (目前維持選單始終存在以確保穩定性)
func refreshSystemTray(myApp fyne.App, myWindow fyne.Window) {
	if desk, ok := myApp.(desktop.App); ok {
		// 始終保留系統匣選單，避免在切換設定時造成程式不穩定
		m := fyne.NewMenu("WinCMP",
			fyne.NewMenuItem("顯示 WinCMP", func() {
				myWindow.Show()
			}),
			fyne.NewMenuItem("完全退出 (Quit)", func() {
				saveAndQuit(myApp)
			}),
		)
		desk.SetSystemTrayMenu(m)
	}
}

// triggerHostsUpdate 檢查並更新系統 hosts 檔
func triggerHostsUpdate() {
	if !appCfg.Global.AutoUpdateHosts {
		return
	}

	// 收集所有啟用專案的網域
	var allDomains []string
	for _, proj := range appCfg.Projects {
		if proj.Enabled {
			allDomains = append(allDomains, proj.Domains...)
		}
	}

	if len(allDomains) == 0 {
		return
	}

	// 1. 檢查缺失網域
	missing, err := hosts.CheckHosts(allDomains)
	if err != nil {
		addErrorLog("system", "檢查 Hosts 失敗", err)
		return
	}

	if len(missing) == 0 {
		// addLog("system", "ℹ️ 系統 Hosts 已包含所有專案網域，無需更新")
		return
	}

	addLog("system", fmt.Sprintf("🔍 偵測到 %d 個網域不在系統 Hosts 中: %s", len(missing), strings.Join(missing, ", ")))

	// 2. 備份 Hosts
	backupPath, err := hosts.BackupHosts(baseDir)
	if err != nil {
		addErrorLog("system", "備份 Hosts 失敗 (將停止更新)", err)
		return
	}
	addLog("system", fmt.Sprintf("✅ 已備份現有 Hosts 到: %s", backupPath))

	// 3. 更新 Hosts
	err = hosts.UpdateHosts(missing)
	if err != nil {
		// 這裡通常是權限問題
		addErrorLog("system", "更新系統 Hosts 失敗 (請嘗試以管理員權限執行 WinCMP)", err)
		return
	}

	addLog("system", fmt.Sprintf("🚀 已成功將 %d 個網域寫入系統 Hosts 檔", len(missing)))
}

// ==== 自定義：模態互動阻擋器 (Modal Blocker) ====
type modalBlocker struct {
	widget.BaseWidget
}

func newModalBlocker() *modalBlocker {
	b := &modalBlocker{}
	b.ExtendBaseWidget(b)
	return b
}

func (b *modalBlocker) CreateRenderer() fyne.WidgetRenderer {
	// 這裡設為半透明黑色 (A: 100)，讓用戶視覺上知道主視窗被鎖定了 (類似網頁的 lightbox)
	// 如果你想要完全透明不想被發現，可以改為 color.NRGBA{R: 0, G: 0, B: 0, A: 1}
	rect := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: 100})
	return widget.NewSimpleRenderer(rect)
}

// 實作 Tapped 事件，但不做任何事情，藉此「吞噬」點擊
func (b *modalBlocker) Tapped(_ *fyne.PointEvent)          {}
func (b *modalBlocker) TappedSecondary(_ *fyne.PointEvent) {}
