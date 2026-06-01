package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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
	"fyne.io/systray"

	"wincmp/internal/config"
	"wincmp/internal/detect"
	"wincmp/internal/hosts"
	"wincmp/internal/port"
	"wincmp/internal/preset"
	"wincmp/internal/process"
	"wincmp/internal/resource"
	"wincmp/internal/scanner"
	"wincmp/internal/singleinstance"
	"wincmp/internal/downloader"

	"fyne.io/fyne/v2/data/binding"

	"sync"
	"sync/atomic"

	"github.com/BurntSushi/toml"
	fynetooltip "github.com/dweymouth/fyne-tooltip"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
	_ "github.com/go-sql-driver/mysql"
	mysql "github.com/go-sql-driver/mysql"
	"github.com/ncruces/zenity"
	"gopkg.in/natefinch/lumberjack.v2"
)

// 全域變數

// tabSwitchReq 分頁切換請求
type tabSwitchReq struct {
	tabIndex int
}

var (
	sysLog          = binding.NewString()
	caddyLog        = binding.NewString()
	dbLog           = binding.NewString()
	mailpitLog      = binding.NewString()
	phpLog          = binding.NewString()
	runtimeLog      = binding.NewString()
	logEntries      map[string]*container.Scroll
	logTabs         *container.AppTabs
	runtimeTabItem  *container.TabItem
	procMgr         *process.Manager
	resourceMonitor *resource.Monitor
	scanRes         *scanner.ScanResult
	appCfg          *config.WincmpConfig
	depCfg          config.DependencyConfig
	baseDir         string
	isZenityOpen    atomic.Bool
	myApp           fyne.App

	runtimeLogWriters map[string]*lumberjack.Logger
	appLogWriter      *lumberjack.Logger
	errorLogCache     sync.Map

	runtimeLogBindings   map[string]binding.String
	activeRuntimeProject string

	runtimeLogMu sync.RWMutex

	tabSwitchCh   chan tabSwitchReq
	tabSwitchDone chan struct{}

	// 主分頁切換鎖：防止 Tab 內容載入時切換到其他 Tab
	mainTabLock      sync.Mutex
	isMainTabLoading atomic.Bool

	// Log Tab 切換鎖：防止應用初始化期間自動切換到 Runtime Tab
	isLogTabSwitchAllowed atomic.Bool

	// saveStateMu 保護 saveLastServiceState，避免並行讀寫 scanRes 和 IsRunning
	saveStateMu sync.Mutex

	// 主分頁
	mainTabs *container.AppTabs

	// PHP Row UI 組件映射 (key: MajorMin, e.g. "8.2")
	phpRowUI map[string]*phpRowComponents

	// 退出標記，防止重複執行儲存與關閉邏輯
	isQuitting atomic.Bool

	// MariaDB 連線池（全域共用，避免頻繁開關連線）
	dbPool    *sql.DB
	dbPoolMu  sync.Mutex
	dbPoolDSN string
)

const (
	maxLogBytes = 200 * 1024
)

// validDomainPattern 用於驗證域名是否只含合法字元（防止 hosts / Caddy 注入攻擊）
var validDomainPattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?)*$`)

// validDbUserPattern 用於驗證資料庫使用者名稱（僅允許文數字、底線、減號）
var validDbUserPattern = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)

// newScrollPassthroughEntry 建立不攔截滾輪事件的單行輸入框
// Fyne 預設 Entry 的 Wrapping 為 TextTruncateClip，會創建內部 scroll widget 並攔截滾輪事件，
// 導致外層 VScroll 無法滾動。設為 TextWrapOff + ScrollNone 可避免此問題。
// 注意：MultiLineEntry 不可使用 ScrollNone，否則 SetMinRowsVisible() 會失效
// （因為 entryRenderer.MinSize 在 ScrollNone 模式下不使用 multiLineRows 計算高度）。
// Ref: https://github.com/fyne-io/fyne/issues/1939
func newScrollPassthroughEntry() *widget.Entry {
	e := widget.NewEntry()
	e.Wrapping = fyne.TextWrapOff
	e.Scroll = fyne.ScrollNone
	return e
}

func getTruncatedMsg() string {
	lines := 500
	if appCfg != nil && appCfg.Global.MaxLogLines > 0 {
		lines = appCfg.Global.MaxLogLines
	}
	return fmt.Sprintf("| 日誌已截斷，僅保留最後 %d 行 / 200KB |\n", lines)
}

// dynamicTooltipLabel 是一個自製的動態 Tooltip 標籤，支援在顯示期間即時刷新內容
type dynamicTooltipLabel struct {
	widget.Label
	hovered bool
	window  fyne.Window
	popup   *widget.PopUp
	content *widget.Label
	cancel  context.CancelFunc // 用於處理顯示延遲
}

func newDynamicTooltipLabel(text string, w fyne.Window) *dynamicTooltipLabel {
	l := &dynamicTooltipLabel{window: w}
	l.Text = text
	l.Alignment = fyne.TextAlignLeading
	l.ExtendBaseWidget(l)
	return l
}

func (l *dynamicTooltipLabel) MouseIn(e *desktop.MouseEvent) {
	l.hovered = true

	// 先取消舊的 context，防止快速反覆 hover 造成 goroutine 洩漏
	if l.cancel != nil {
		l.cancel()
	}

	// 加入 500ms 延遲顯示，避免滑鼠快速經過時狂閃
	ctx, cancel := context.WithCancel(context.Background())
	l.cancel = cancel

	go func() {
		time.Sleep(500 * time.Millisecond)
		select {
		case <-ctx.Done():
			return
		default:
			fyne.Do(func() {
				if !l.hovered {
					return
				}
				if l.popup == nil {
					l.content = widget.NewLabel("")
					l.content.TextStyle = fyne.TextStyle{Monospace: true}
					l.popup = widget.NewPopUp(container.NewPadded(l.content), l.window.Canvas())
				}

				// 取得全域座標來定位
				pos := e.AbsolutePosition
				pos.Y -= l.popup.MinSize().Height + 5 // 顯示在滑鼠上方
				if pos.Y < 0 {
					pos.Y = e.AbsolutePosition.Y + 25 // 空間不足就顯示在下方
				}
				l.popup.ShowAtPosition(pos)
			})
		}
	}()
}

func (l *dynamicTooltipLabel) MouseOut() {
	l.hovered = false
	if l.cancel != nil {
		l.cancel()
	}
	if l.popup != nil {
		l.popup.Hide()
	}
}

func (l *dynamicTooltipLabel) MouseMoved(e *desktop.MouseEvent) {}

func (l *dynamicTooltipLabel) SetToolTip(text string) {
	if l.content != nil {
		l.content.SetText(text)
		// 如果正在顯示，重新整理大小以適應內容
		if l.popup != nil && l.hovered {
			l.popup.Refresh()
		}
	}
}

func (l *dynamicTooltipLabel) IsHovered() bool {
	return l.hovered
}

func cleanupOldLogs(retention int) {
	if retention <= 0 {
		return
	}
	logDir := filepath.Join(baseDir, "logs")
	cutoff := time.Now().AddDate(0, 0, -retention)

	patterns := []string{"wincmp-*.log", "node-*.log", "error-*.log", "runtime-*.log"}
	for _, pattern := range patterns {
		files, _ := filepath.Glob(filepath.Join(logDir, pattern))
		for _, f := range files {
			info, err := os.Stat(f)
			if err != nil {
				continue
			}
			if info.IsDir() {
				continue
			}
			if info.ModTime().Before(cutoff) {
				os.Remove(f)
			}
		}
	}
}

func initLogWriters() {
	logDir := filepath.Join(baseDir, "logs")
	os.MkdirAll(logDir, 0700)

	runtimeLogWriters = make(map[string]*lumberjack.Logger)

	retention := 0
	if appCfg != nil {
		retention = appCfg.Global.MaxLogRetention
	}

	appLogWriter = &lumberjack.Logger{
		Filename:   filepath.Join(logDir, fmt.Sprintf("wincmp-%s.log", time.Now().Format("2006-01-02"))),
		MaxSize:    10,
		MaxBackups: 0,
		MaxAge:     retention,
		Compress:   true,
	}
}

// getRuntimeLogWriter 取得（或建立）指定項目的 Runtime log writer
func getRuntimeLogWriter(projectName string) *lumberjack.Logger {
	if projectName == "" {
		return nil
	}
	if w, ok := runtimeLogWriters[projectName]; ok {
		return w
	}
	logDir := filepath.Join(baseDir, "logs")
	os.MkdirAll(logDir, 0700)

	retention := 0
	if appCfg != nil {
		retention = appCfg.Global.MaxLogRetention
	}
	w := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, fmt.Sprintf("runtime-%s-%s.log", projectName, time.Now().Format("2006-01-02"))),
		MaxSize:    10,
		MaxBackups: 0,
		MaxAge:     retention,
		Compress:   true,
	}
	runtimeLogWriters[projectName] = w
	return w
}

type phpRowComponents struct {
	StatusLabel   *widget.Label
	ActionBtn     *widget.Button
	ProcessSelect *widget.Select
	UptimeData    binding.String
}

// parseProjectFromRuntimeMsg 從 runtime log 訊息中解析項目名稱
// 支援格式：
//   - "🚀 [projectName] ..." / "💻 [projectName] ..." (來自 StartRuntime)
//   - "[runtimeLabel (projectName)] ..." / "[runtimeLabel (projectName):err] ..." (來自 pipeRuntimeOutput)
func parseProjectFromRuntimeMsg(msg string) string {
	// 優先從方括號 [...] 中解析：先檢查是否有 "(projectName)" 格式，再取整個內容
	if start := strings.Index(msg, "["); start >= 0 {
		if end := strings.Index(msg[start:], "]"); end > 1 {
			bracketContent := msg[start+1 : start+end]
			// 格式 1: "[runtimeLabel (projectName)]" 或 "[runtimeLabel (projectName):err]"
			if pStart := strings.Index(bracketContent, "("); pStart >= 0 {
				if pEnd := strings.Index(bracketContent[pStart:], ")"); pEnd > 1 {
					return bracketContent[pStart+1 : pStart+pEnd]
				}
			}
			// 格式 2: "[projectName]"（排除時間戳 [HH:MM:SS]）
			if !strings.Contains(bracketContent, ":") {
				return bracketContent
			}
		}
	}
	return ""
}

// logRingBuffer 環形緩衝區，避免每次追加日誌都做全量字串 Get/Set 操作
type logRingBuffer struct {
	mu           sync.Mutex
	lines        []string
	maxLines     int
	maxBytes     int
	dirty        bool
	wasTruncated bool
	binding      binding.String
}

var (
	logBuffers   = make(map[binding.String]*logRingBuffer)
	logBuffersMu sync.Mutex
	logSyncStop  = make(chan struct{})
)

func getOrCreateLogBuffer(b binding.String) *logRingBuffer {
	logBuffersMu.Lock()
	defer logBuffersMu.Unlock()
	if buf, ok := logBuffers[b]; ok {
		return buf
	}
	maxLines := 500
	if appCfg != nil && appCfg.Global.MaxLogLines > 0 {
		maxLines = appCfg.Global.MaxLogLines
	}
	buf := &logRingBuffer{
		lines:    make([]string, 0, maxLines),
		maxLines: maxLines,
		maxBytes: maxLogBytes,
		binding:  b,
	}
	logBuffers[b] = buf
	return buf
}

func (rb *logRingBuffer) append(line string) {
	rb.mu.Lock()
	rb.lines = append(rb.lines, line)
	if len(rb.lines) > rb.maxLines {
		rb.lines = rb.lines[len(rb.lines)-rb.maxLines:]
		rb.wasTruncated = true
	}
	totalBytes := 0
	for i := len(rb.lines) - 1; i >= 0; i-- {
		totalBytes += len(rb.lines[i])
		if totalBytes > rb.maxBytes {
			rb.lines = rb.lines[i+1:]
			rb.wasTruncated = true
			break
		}
	}
	rb.dirty = true
	rb.mu.Unlock()
}

func (rb *logRingBuffer) sync() {
	rb.mu.Lock()
	if !rb.dirty {
		rb.mu.Unlock()
		return
	}
	rb.dirty = false
	content := strings.Join(rb.lines, "")
	if rb.wasTruncated {
		content = getTruncatedMsg() + content
	}
	rb.mu.Unlock()
	rb.binding.Set(content)
}

func startLogBufferSync() {
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				logBuffersMu.Lock()
				for _, buf := range logBuffers {
					buf.sync()
				}
				logBuffersMu.Unlock()
			case <-logSyncStop:
				return
			}
		}
	}()
}

func stopLogBufferSync() {
	close(logSyncStop)
}

// appendToLogBinding 將文字追加到指定的 binding.String（透過 ring buffer 批次更新）
func appendToLogBinding(logSource binding.String, newText string) {
	buf := getOrCreateLogBuffer(logSource)
	buf.append(newText)
}

// tabSwitchWorker 處理分頁切換請求的 goroutine
// 使用 channel-based 非同步節流：收到請求後 debounce 500ms 再切換
func tabSwitchWorker() {
	var lastReq tabSwitchReq
	hasPending := false
	debounceTimer := time.NewTimer(time.Hour)
	debounceTimer.Stop()

	// Tab index 對應的 category 名稱（用於滾動到對應的 log）
	tabIndexToCategory := []string{
		0: "system",
		1: "caddy",
		2: "mariadb",
		3: "mailpit",
		4: "php",
		5: "runtime",
	}

	for {
		select {
		case <-tabSwitchDone:
			debounceTimer.Stop()
			return
		case req := <-tabSwitchCh:
			lastReq = req
			hasPending = true
			debounceTimer.Reset(500 * time.Millisecond)
		case <-debounceTimer.C:
			if hasPending {
				hasPending = false
				fyne.Do(func() {
					if logTabs != nil && logTabs.CurrentTabIndex() != lastReq.tabIndex {
						logTabs.SelectIndex(lastReq.tabIndex)
					}
					// 切換 Tab 後，自動滾動到該 Tab 的 log 底部
					if logEntries != nil {
						cat := tabIndexToCategory[lastReq.tabIndex]
						if scroll, ok := logEntries[cat]; ok && scroll != nil {
							scroll.ScrollToBottom()
						}
					}
				})
			}
		}
	}
}

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
	case "mailpit":
		logSource = mailpitLog
		tabIndex = 3 // Mailpit is at index 3 (System=0, Caddy=1, MariaDB=2, Mailpit=3, PHP=4, Runtime=5)
	case "php":
		logSource = phpLog
		tabIndex = 4 // PHP is at index 4 (System=0, Caddy=1, MariaDB=2, Mailpit=3, PHP=4, Runtime=5)
	case "node", "runtime":
		// 解析項目名，寫入該項目的 binding
		projectName := parseProjectFromRuntimeMsg(msg)
		runtimeLogMu.RLock()
		bindings := runtimeLogBindings
		activeProj := activeRuntimeProject
		runtimeLogMu.RUnlock()
		runtimeLogUpdated := false
		if projectName != "" && bindings != nil {
			if projLog, ok := bindings[projectName]; ok {
				appendToLogBinding(projLog, newText)
			}
			// 如果該項目是當前顯示的項目，同步更新 runtimeLog
			if projectName == activeProj {
				appendToLogBinding(runtimeLog, newText)
				runtimeLogUpdated = true
			}
		}
		// 若無法解析項目名或無 activeRuntimeProject，fallback 寫入 runtimeLog
		// 確保所有 runtime log 至少都顯示在 Runtime 分頁
		if projectName == "" || activeProj == "" {
			appendToLogBinding(runtimeLog, newText)
			runtimeLogUpdated = true
		}
		// 自動切換到 Runtime 分頁（只在實際有新內容時才切換，且初始化完成後）
		if runtimeLogUpdated && logTabs != nil && isLogTabSwitchAllowed.Load() {
			select {
			case tabSwitchCh <- tabSwitchReq{tabIndex: 5}:
			default:
			}
		}
		// 滾動到底部
		if logEntries != nil {
			if scroll, ok := logEntries["runtime"]; ok && scroll != nil {
				fyne.Do(func() {
					scroll.ScrollToBottom()
				})
			}
		}
		goto fileWrite
	default:
		catKey = "system"
		logSource = sysLog
		tabIndex = 0
	}

	if logSource != nil {
		appendToLogBinding(logSource, newText)

		if logEntries != nil {
			if scroll, ok := logEntries[catKey]; ok && scroll != nil {
				fyne.Do(func() {
					scroll.ScrollToBottom()
				})
			}
		}

		// 自動切換分頁機制 (channel-based 非同步節流)
		if logTabs != nil {
			select {
			case tabSwitchCh <- tabSwitchReq{tabIndex: tabIndex}:
			default:
			}
		}
	}

fileWrite:
	// 寫入檔案日誌
	if catKey == "node" || catKey == "runtime" {
		projectName := parseProjectFromRuntimeMsg(msg)
		if projectName != "" {
			if w := getRuntimeLogWriter(projectName); w != nil {
				w.Write([]byte(newText))
			}
		}
	} else if appLogWriter != nil {
		appLogWriter.Write([]byte(newText))
	}
}

// switchRuntimeLog 切換 Runtime 分頁顯示的項目 log
func switchRuntimeLog(projectName string) {
	runtimeLogMu.Lock()
	activeRuntimeProject = projectName
	bindings := runtimeLogBindings
	runtimeLogMu.Unlock()

	if bindings != nil {
		if projLog, ok := bindings[projectName]; ok {
			content, _ := projLog.Get()
			runtimeLog.Set(content)
		} else {
			runtimeLog.Set("")
		}
	}

	// 更新 Runtime 分頁標籤
	if runtimeTabItem != nil && logTabs != nil {
		if projectName != "" {
			runtimeTabItem.Text = fmt.Sprintf("Runtime (%s)", projectName)
		} else {
			runtimeTabItem.Text = "Runtime"
		}
		logTabs.Refresh()
	}

	// 滾動到底部
	if logEntries != nil {
		if scroll, ok := logEntries["runtime"]; ok && scroll != nil {
			fyne.Do(func() {
				scroll.ScrollToBottom()
			})
		}
	}
}

// ensureRuntimeLogBinding 確保指定項目的 log binding 存在（不存在則建立）
func ensureRuntimeLogBinding(projectName string) {
	if projectName == "" {
		return
	}
	runtimeLogMu.Lock()
	if runtimeLogBindings == nil {
		runtimeLogBindings = make(map[string]binding.String)
	}
	if _, ok := runtimeLogBindings[projectName]; !ok {
		runtimeLogBindings[projectName] = binding.NewString()
	}
	runtimeLogMu.Unlock()
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

	detailMsg := fmt.Sprintf("[%s] [%s] %s: %+v\n", timeStr, category, contextMsg, err)

	retention := 0
	if appCfg != nil {
		retention = appCfg.Global.MaxLogRetention
	}

	// 使用 sync.Map 快取 error logger，按日期為 key 做單例管理
	val, _ := errorLogCache.LoadOrStore(dateStr, &lumberjack.Logger{
		Filename:   filepath.Join(baseDir, "logs", fmt.Sprintf("error-%s.log", dateStr)),
		MaxSize:    10,
		MaxBackups: 0,
		MaxAge:     retention,
		Compress:   true,
	})
	l := val.(*lumberjack.Logger)
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
			// 使用 fyne.Do 配合 channel 取代 fyne.DoAndWait，避免死鎖
			resultCh := make(chan struct{})
			fyne.Do(func() {
				callback(path)
				close(resultCh)
			})
			// 等待 callback 完成，帶超時保護
			select {
			case <-resultCh:
			case <-time.After(5 * time.Second):
				// 超時保護：避免 callback 阻塞永遠等待
			}
		}
	}()
}

// openLocalPath 透過系統預設程式開啟檔案或資料夾
// 安全檢查：路徑必須在 baseDir 內，且不允許開啟可執行檔
var blockedExecExts = map[string]bool{
	".exe": true, ".bat": true, ".cmd": true, ".ps1": true,
	".vbs": true, ".msi": true, ".com": true, ".scr": true,
}

func openLocalPath(path string, force bool) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		addErrorLog("system", "路徑無效: "+path, err)
		return
	}
	cleanBase := filepath.Clean(baseDir)
	cleanAbs := filepath.Clean(absPath)
	
	// 特別許可：若 force 為 true 則跳過目錄限制 (但仍保留副檔名檢查)
	if !force && !strings.HasPrefix(cleanAbs, cleanBase+string(os.PathSeparator)) && cleanAbs != cleanBase {
		addErrorLog("system", "路徑不在允許的目錄內: "+absPath, nil)
		return
	}
	info, err := os.Stat(absPath)
	if err == nil && !info.IsDir() {
		ext := strings.ToLower(filepath.Ext(absPath))
		if blockedExecExts[ext] {
			addErrorLog("system", "不允許開啟可執行檔: "+absPath, nil)
			return
		}
	}
	err = windows.ShellExecute(0, toPtr("open"), toPtr(absPath), nil, nil, 1)
	if err != nil {
		addErrorLog("system", "開啟失敗: "+absPath, err)
	}
}

func openLatestLog(prefix string) {
	logDir := filepath.Join(baseDir, "logs")
	pattern := filepath.Join(logDir, prefix+"-*.log")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		addErrorLog("system", "找不到日誌檔: "+prefix, nil)
		return
	}
	sort.Strings(matches)
	latest := matches[len(matches)-1]
	openLocalPath(latest, false)
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
	saveStateMu.Lock()
	defer saveStateMu.Unlock()

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

	appCfg.Global.LastServiceState.Mailpit = procMgr.IsRunning(process.MailpitServiceKey())

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
	// 防止重複執行 (例如 myApp.Quit() 觸發視窗關閉攔截器)
	if !isQuitting.CompareAndSwap(false, true) {
		return
	}

	addLog("system", "正在儲存狀態與關閉所有服務...")
	saveLastServiceState()
	closeDBPool()
	stopLogBufferSync()
	close(tabSwitchDone) // 通知 tabSwitchWorker goroutine 停止
	procMgr.StopAll()
	if resourceMonitor != nil {
		resourceMonitor.Stop()
	}
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

	// 初始化日誌寫入器 (lumberjack 全域實例)
	initLogWriters()
	startLogBufferSync()

	// 初始化分頁切換 channel 和處理 goroutine
	isLogTabSwitchAllowed.Store(false) // 初始化期間禁止 Log Tab 自動切換
	tabSwitchCh = make(chan tabSwitchReq, 32)
	tabSwitchDone = make(chan struct{})
	go tabSwitchWorker()

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
	runtimeLogBindings = make(map[string]binding.String)

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

		// 套用 logTheme，內部會動態跟隨全域主題並在深色模式下確保字體亮度
		return container.NewThemeOverride(scroll, &logTheme{})
	}

	logTabs = container.NewAppTabs(
		container.NewTabItem("System", createLogTab(sysLog, "system")),
		container.NewTabItem("Caddy", createLogTab(caddyLog, "caddy")),
		container.NewTabItem("MariaDB", createLogTab(dbLog, "mariadb")),
		container.NewTabItem("Mailpit", createLogTab(mailpitLog, "mailpit")),
		container.NewTabItem("PHP", createLogTab(phpLog, "php")),
	)
	runtimeTabItem = container.NewTabItem("Runtime", createLogTab(runtimeLog, "runtime"))
	logTabs.Append(runtimeTabItem)

	// 使用者手動點擊 Log Tab 時，自動滾動到底部
	logTabs.OnSelected = func(tab *container.TabItem) {
		fyne.Do(func() {
			if logEntries != nil {
				var cat string
				switch tab.Text {
				case "System":
					cat = "system"
				case "Caddy":
					cat = "caddy"
				case "MariaDB":
					cat = "mariadb"
				case "Mailpit":
					cat = "mailpit"
				case "PHP":
					cat = "php"
				case "Runtime":
					cat = "runtime"
				default:
					return
				}
				if scroll, ok := logEntries[cat]; ok && scroll != nil {
					scroll.ScrollToBottom()
				}
			}
		})
	}

	addLog("system", "正在初始化 WinCMP...")
	addLog("system", fmt.Sprintf("專案根目錄: %s", baseDir))

	// --- 建立程序管理器 ---
	procMgr = process.NewManager(baseDir, addLog, addErrorLog)

	// --- 建立資源監控器（顯示 WinCMP 主程序 + 可選 Stack Total） ---
	resourceMonitor = resource.NewAppResourceMonitor(procMgr)

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
		if len(scanRes.ComposerList) > 0 {
			versions := make([]string, len(scanRes.ComposerList))
			for i, c := range scanRes.ComposerList {
				versions[i] = c.Version
			}
			addLog("system", fmt.Sprintf("  ✓ 找到 Composer 版本: [%s]", strings.Join(versions, ", ")))
		} else {
			addLog("system", "  ✗ 未找到 Composer")
		}
		if len(scanRes.HeidiSQLList) > 0 {
			versions := make([]string, len(scanRes.HeidiSQLList))
			for i, h := range scanRes.HeidiSQLList {
				versions[i] = h.Version
			}
			addLog("system", fmt.Sprintf("  ✓ 找到 HeidiSQL 版本: [%s]", strings.Join(versions, ", ")))
		} else {
			addLog("system", "  ✗ 未找到 HeidiSQL")
		}
		if len(scanRes.MailpitList) > 0 {
			versions := make([]string, len(scanRes.MailpitList))
			for i, mp := range scanRes.MailpitList {
				versions[i] = mp.Version
			}
			addLog("system", fmt.Sprintf("  ✓ 找到 Mailpit 版本: [%s]", strings.Join(versions, ", ")))
		} else {
			addLog("system", "  ✗ 未找到 Mailpit")
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
		if len(scanRes.NodeList) > 0 {
			versions := make([]string, len(scanRes.NodeList))
			for i, n := range scanRes.NodeList {
				versions[i] = n.Version
			}
			addLog("system", fmt.Sprintf("  ✓ 找到 Node 版本: [%s]", strings.Join(versions, ", ")))
		} else {
			addLog("system", "  ✗ 未找到 Node")
		}
		if len(scanRes.PHPList) > 0 {
			versions := make([]string, len(scanRes.PHPList))
			for i, p := range scanRes.PHPList {
				versions[i] = p.Version
			}
			addLog("system", fmt.Sprintf("  ✓ 找到 PHP 版本: [%s]", strings.Join(versions, ", ")))
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
				DefaultWWW:       "www",
				DefaultSSL:       "conf/ssl",
				RestoreLastState: bool(true), // 用於相容性
				MinimizeToTray:   false,
				AutoUpdateHosts:  true,
			},
		}
	} else {
		addLog("system", fmt.Sprintf("  ✓ 設定檔已載入 (%d 個專案)", len(appCfg.Projects)))
	}

	// --- 載入依賴設定檔 ---
	depCfgPath := filepath.Join(baseDir, "conf", "dependencies.json")
	var depLoadErr error
	depCfg, depLoadErr = config.LoadDependencies(depCfgPath)
	if depLoadErr != nil {
		addErrorLog("system", "無法載入依賴設定檔，將使用預設設定", depLoadErr)
		depCfg = config.DefaultDependencies
	} else {
		addLog("system", "  ✓ 依賴設定檔已載入")
	}

	cleanupOldLogs(appCfg.Global.MaxLogRetention)

	// --- 預計算 Projects 的 ConfigExists 狀態（提升 UI 渲染效能）---
	appCfg.RefreshConfigExists(baseDir)

	// --- 補全舊版專案的框架資訊 (遷移至新 Preset 模型) ---
	needsSave := false
	for i := range appCfg.Projects {
		p := &appCfg.Projects[i]
		// 如果 Runtime 是 Node/Bun，但 Type 還在舊格式或為空，使用 Preset 偵測
		if (p.RuntimeType == "node" || p.RuntimeType == "bun") && (p.Type == "node" || p.Type == "bun" || p.Type == "") {
			root := appCfg.GetProjectRoot(*p, baseDir)
			detRes := preset.DetectProjectPreset(root)
			if detRes.Type != "" && detRes.Type != p.Type && detRes.Type != preset.TypeStatic {
				p.Type = detRes.Type
				if p.RuntimePort == 0 {
					p.RuntimePort = detRes.Port
				}
				if p.RuntimeType == "" || p.RuntimeType == "auto" {
					p.RuntimeType = detRes.Runtime
				}
				needsSave = true
				addLog("system", fmt.Sprintf("  ↳ 自動遷移專案 [%s] 的框架類型為: %s", p.Name, detRes.Type))
			}
		}
		// 如果 Type 是 "go" 舊值，嘗試判斷是 PocketBase 還是 Go API
		if p.Type == "go" {
			root := appCfg.GetProjectRoot(*p, baseDir)
			detRes := preset.DetectProjectPreset(root)
			if detRes.Type == preset.TypePocketBase {
				p.Type = preset.TypePocketBase
				p.RuntimeType = preset.RuntimeGoRun
				p.RuntimePort = 8090
			} else {
				p.Type = preset.TypeGoAPI
				p.RuntimeType = preset.RuntimeGoAir
			}
			needsSave = true
			addLog("system", fmt.Sprintf("  ↳ 自動遷移專案 [%s] 的類型為: %s", p.Name, p.Type))
		}
	}
	if needsSave {
		appCfg.Save(cfgPath)
	}

	// --- 檢查重複項目名稱 ---
	nameCount := make(map[string]int)
	for _, p := range appCfg.Projects {
		nameCount[p.Name]++
	}
	for name, count := range nameCount {
		if count > 1 {
			addLog("system", fmt.Sprintf("⚠️ 發現 %d 個重複項目名稱 [%s]，Runtime Log 可能出現混淆，建議修改項目名稱以區分", count, name))
		}
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
			if err := checkSSLCerts(); err == nil && regenerateCaddyAndReload() == nil {
				procMgr.StartCaddy(info.Version, info.ExePath)
			}
		}

		if appCfg.Global.LastServiceState.MariaDB && len(scanRes.MariaDBList) > 0 {
			info := scanRes.MariaDBList[0]
			addLog("system", "自動啟動上次執行的服務: MariaDB")
			go func() {
				done, errCh := procMgr.StartMariaDBAsync(
					info.Version,
					appCfg.Global.MariaDBExternal,
					appCfg.Global.MariaDBBasedir,
					appCfg.Global.MariaDBDatadir,
					appCfg.Global.MariaDBType,
					appCfg.Global.MariaDBPort,
				)
				<-done
				if err := <-errCh; err != nil {
					addErrorLog("mariadb", "自動啟動 MariaDB 失敗", err)
				}
			}()
		}

		if appCfg.Global.LastServiceState.Mailpit && len(scanRes.MailpitList) > 0 {
			mpInfo := scanRes.MailpitList[0]
			mpSmtp := 1025
			mpHttp := 8025
			if appCfg.Global.MailpitSMTPPort > 0 {
				mpSmtp = appCfg.Global.MailpitSMTPPort
			}
			if appCfg.Global.MailpitHTTPPort > 0 {
				mpHttp = appCfg.Global.MailpitHTTPPort
			}
			addLog("system", "自動啟動上次執行的服務: Mailpit")
			if err := procMgr.StartMailpit(mpInfo.Version, mpInfo.ExePath, mpSmtp, mpHttp, appCfg.Global.MailpitUseDB); err != nil {
				addErrorLog("mailpit", "自動啟動 Mailpit 失敗", err)
			}
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

	// 檢查核心依賴偵測提示
	checkCoreDependencies(myWindow)

	// 初始化完成，允許 Log Tab 自動切換
	isLogTabSwitchAllowed.Store(true)

	resourceStatusLabel := newDynamicTooltipLabel("WinCMP RAM: -- MB | CPU: -- %", myWindow)
	resourceStatusLabel.SetToolTip("WinCMP 資源監控\n\n載入中...")
	logTitle := widget.NewLabelWithStyle("Terminal Logs", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	caddyLogBtn := widget.NewButtonWithIcon("Caddy.log", theme.DocumentIcon(), func() {
		openLocalPath(filepath.Join(baseDir, "logs", "caddy.log"), false)
	})
	runtimeLogBtn := widget.NewButtonWithIcon("Runtime.log", theme.DocumentIcon(), func() {
		runtimeLogMu.RLock()
		activeProj := activeRuntimeProject
		runtimeLogMu.RUnlock()
		if activeProj != "" {
			// 開啟當前活動項目的最新 log 檔案
			pattern := filepath.Join(baseDir, "logs", fmt.Sprintf("runtime-%s-*", activeProj))
			matches, _ := filepath.Glob(pattern)
			if len(matches) > 0 {
				sort.Strings(matches)
				openLocalPath(matches[len(matches)-1], false)
				return
			}
		}
		openLatestLog("runtime")
	})
	wincmpLogBtn := widget.NewButtonWithIcon("Wincmp.log", theme.DocumentIcon(), func() {
		openLatestLog("wincmp")
	})
	logTopBar := container.NewHBox(logTitle, wincmpLogBtn, caddyLogBtn, runtimeLogBtn, layout.NewSpacer(), resourceStatusLabel)
	logPanel := container.NewBorder(
		logTopBar, nil, nil, nil, logTabs,
	)

	// --- 各 Tab 內容 ---
	dashboardContent := createDashboard(myWindow, func() {
		// 這裡傳入一個重新整理 Projects Tab 的內容的閉包，如果需要的話
	})
	projectsContent := createProjectsTab(myWindow)
	dbExplorerContent, refreshDBExplorer := createDatabaseExplorerTab()
	settingsContent := createSettingsTab(myWindow)
	runtimeContent, refreshRuntimeTab := createRuntimeTab(myWindow)

	// --- 左側選單 (Sidebar) ---
	mainTabs = container.NewAppTabs(
		container.NewTabItemWithIcon("Dashboard", theme.HomeIcon(), dashboardContent),
		container.NewTabItemWithIcon("Projects", theme.FolderIcon(), projectsContent),
		container.NewTabItemWithIcon("DB Explorer", theme.StorageIcon(), dbExplorerContent),
		container.NewTabItemWithIcon("Runtime", theme.ComputerIcon(), runtimeContent),
		container.NewTabItemWithIcon("Settings", theme.SettingsIcon(), settingsContent),
	)
	mainTabs.OnSelected = func(tab *container.TabItem) {
		if isMainTabLoading.Load() {
			return
		}
		if tab.Text == "DB Explorer" {
			refreshDBExplorer()
		} else if tab.Text == "Runtime" {
			refreshRuntimeTab()
		}
	}
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

	// 啟動資源監控 Goroutine（每秒更新 status label）
	go resourceMonitor.Start(resourceStatusLabel)

	defer func() {
		if runtimeLogWriters != nil {
			for _, w := range runtimeLogWriters {
				w.Close()
			}
		}
		if appLogWriter != nil {
			appLogWriter.Close()
		}
	}()

	myWindow.ShowAndRun()
}

// ===== 自定義主題解決 Log 對比度 (動態跟隨主程式主題) =====
// ===== 自定義主題解決 Log 對比度 (動態跟隨主程式主題) =====
type logTheme struct{}

func (m *logTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if variant == theme.VariantDark {
		if name == theme.ColorNameForeground {
			return color.NRGBA{R: 225, G: 225, B: 225, A: 255} // 優化後更亮的灰白色
		}
		if name == theme.ColorNameDisabled {
			return color.NRGBA{R: 190, G: 190, B: 190, A: 255} // 時間戳也稍微加亮
		}
	}

	// 其他顏色透過全域主題獲取
	t := fyne.CurrentApp().Settings().Theme()
	if name == theme.ColorNameDisabled {
		return t.Color(theme.ColorNameForeground, variant) // 讓 Disabled 文字顏色跟一般文字一樣
	}
	return t.Color(name, variant)
}

func (m *logTheme) Font(style fyne.TextStyle) fyne.Resource {
	return fyne.CurrentApp().Settings().Theme().Font(style)
}

func (m *logTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return fyne.CurrentApp().Settings().Theme().Icon(name)
}

func (m *logTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNameInnerPadding || name == theme.SizeNameInputBorder {
		return 0
	}
	return fyne.CurrentApp().Settings().Theme().Size(name)
}

// ===== 自定義按鈕主題（保留 Start/Stop 狀態顏色）====
type coloredButtonTheme struct {
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
	return fyne.CurrentApp().Settings().Theme().Color(name, variant)
}

func (m *coloredButtonTheme) Font(style fyne.TextStyle) fyne.Resource {
	return fyne.CurrentApp().Settings().Theme().Font(style)
}

func (m *coloredButtonTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return fyne.CurrentApp().Settings().Theme().Icon(name)
}

func (m *coloredButtonTheme) Size(name fyne.ThemeSizeName) float32 {
	return fyne.CurrentApp().Settings().Theme().Size(name)
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
	phpRowUI = make(map[string]*phpRowComponents)

	header := container.NewGridWithColumns(6,
		widget.NewLabelWithStyle("Service", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Version", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Status / PID", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Uptime", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Port(s)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Action", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
	)

	rows := []fyne.CanvasObject{
		container.NewBorder(
			nil, nil, nil,
			widget.NewButtonWithIcon("Manage Dependencies", theme.DownloadIcon(), func() {
				showDependencyManager(win)
			}),
			widget.NewLabelWithStyle("Service Modules Manager", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		),
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
		rows = append(rows, createMariaDBRow(win, mariaDBInfo))
	}

	// Mailpit 行
	for _, mailpitInfo := range scanRes.MailpitList {
		rows = append(rows, createMailpitRow(win, mailpitInfo))
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
	// 移除所有路徑部分 (如果 folderName 包含路徑，只取最後一部分)
	domain = filepath.Base(filepath.ToSlash(domain))

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

	mapping, fallback := loadLaravelPHPMapping()

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

			// 使用 Preset 系統偵測專案類型
			detRes := preset.DetectProjectPreset(projectPath)

			projectType := detRes.Type
			runtimeType := detRes.Runtime
			runtimePort := detRes.Port
			phpVersion := ""

			// Laravel 特殊處理
			if projectType == preset.TypeLaravel {
				res := detect.DetectLaravel(projectPath)
				phpVersion = getRecommendedPHPVersion(res.Version, mapping, fallback)
			}

			newProj := config.ProjectConfig{
				Name:        config.SanitizeProjectName(name),
				Domains:     []string{GenerateValidDomain(name)},
				RootPath:    projectPath,
				PHPVersion:  phpVersion,
				Type:        projectType,
				RuntimeType: runtimeType,
				UseSSL:      true,
				Enabled:     true,
				RuntimePort: runtimePort,
			}

			appCfg.Projects = append(appCfg.Projects, newProj)
			added++

			if projectType == preset.TypeLaravel {
				addLog("system", fmt.Sprintf("  ↳ %s: 偵測為 Laravel (Confidence: %d, Reasons: %s)", name, detRes.Confidence, strings.Join(detRes.Reasons, ", ")))
			} else if projectType != preset.TypeStatic && projectType != "" {
				addLog("system", fmt.Sprintf("  ↳ %s: 偵測為 %s (Runtime: %s, Confidence: %d, Reasons: %s)", name, preset.GetProjectTypeLabel(projectType), preset.GetRuntimeLabel(runtimeType), detRes.Confidence, strings.Join(detRes.Reasons, ", ")))
			}
		}
	}
	if added > 0 {
		appCfg.Save(filepath.Join(baseDir, "conf", "wincmp.json"))
		addLog("system", fmt.Sprintf("📌 已自動掃描並加入 %d 個新專案", added))
		triggerHostsUpdate()
	}
}

type laravelPHPMapping struct {
	Mappings []struct {
		Laravel string `json:"laravel"`
		PHP     string `json:"php"`
	} `json:"mappings"`
	Fallback string `json:"fallback"`
}

func loadLaravelPHPMapping() (map[string]string, string) {
	mappingFile := filepath.Join(baseDir, "conf", "php", "laravel-php-mapping.json")
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

func getRecommendedPHPVersion(laravelVersion string, mapping map[string]string, fallback string) string {
	if laravelVersion == "" {
		return validatePHPMajorVersion(fallback)
	}

	if php, ok := mapping[laravelVersion]; ok {
		validated := validatePHPMajorVersion(php)
		if validated != "" {
			return validated
		}
	}

	if strings.HasPrefix(laravelVersion, "<") {
		if php, ok := mapping["<"+laravelMajorStr(laravelVersion)]; ok {
			validated := validatePHPMajorVersion(php)
			if validated != "" {
				return validated
			}
		}
		return validatePHPMajorVersion(fallback)
	}

	if strings.HasPrefix(laravelVersion, ">=") || strings.HasPrefix(laravelVersion, ">") {
		if php, ok := mapping[">="+laravelMajorStr(laravelVersion)]; ok {
			validated := validatePHPMajorVersion(php)
			if validated != "" {
				return validated
			}
		}
	}

	laravelMajor := parseLaravelMajor(laravelVersion)
	if laravelMajor == 0 {
		return validatePHPMajorVersion(fallback)
	}

	for laravelMajor > 0 {
		key := strconv.Itoa(laravelMajor) + ".x"
		if php, ok := mapping[key]; ok {
			validated := validatePHPMajorVersion(php)
			if validated != "" {
				return validated
			}
		}
		laravelMajor--
	}

	return validatePHPMajorVersion(fallback)
}

func laravelMajorStr(version string) string {
	version = strings.TrimPrefix(version, "<")
	version = strings.TrimPrefix(version, ">=")
	version = strings.TrimPrefix(version, ">")
	return strings.TrimSuffix(version, ".x")
}

func parseLaravelMajor(version string) int {
	majorStr := laravelMajorStr(version)
	if majorStr == "" {
		return 0
	}
	major, err := strconv.Atoi(majorStr)
	if err != nil {
		return 0
	}
	return major
}

func validatePHPMajorVersion(majorVersion string) string {
	for _, info := range scanRes.PHPList {
		if info.MajorMin == majorVersion {
			return majorVersion
		}
	}
	return ""
}

// sanitizePath 驗證路徑不包含路徑遍歷攻擊（..）且為合法路徑
func sanitizePath(path string) (string, error) {
	cleaned := filepath.Clean(path)
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("路徑含非法的目錄遍歷: %s", path)
	}
	return cleaned, nil
}

// validateCaddyPath 驗證 Caddy 設定檔中的路徑安全性
func validateCaddyPath(path string) (string, error) {
	cleaned := filepath.Clean(path)
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("路徑含非法的目錄遍歷: %s", path)
	}
	// 轉換為正斜線（Caddy 設定檔格式）
	return strings.ReplaceAll(cleaned, "\\", "/"), nil
}

// generatePHPUpstream 產生 PHP 負載平衡設定檔
func generatePHPUpstream() error {
	snippetDir := filepath.Join(baseDir, "conf", "snippets")
	os.MkdirAll(snippetDir, 0700)
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

	return os.WriteFile(upstreamPath, []byte(content.String()), 0600)
}

// generateCaddyfiles 產生所有子專案的 .caddy 設定檔
func generateCaddyfiles() error {
	// 先產生 PHP Upstream
	if err := generatePHPUpstream(); err != nil {
		return err
	}

	sitesDir := filepath.Join(baseDir, "conf", "sites")
	os.MkdirAll(sitesDir, 0700)

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
			// 直接使用者的 domains，不做安全過濾
			// 無效的 domains (如含底線) 可能導致 hosts 更新失敗，但 Caddyfile 本身可正常運作
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
			var safeCrt, safeKey string
			if certExists {
				// 驗證路徑安全性（防止路徑遍歷攻擊）
				var crtErr, keyErr error
				safeCrt, crtErr = validateCaddyPath(crt)
				safeKey, keyErr = validateCaddyPath(key)
				if crtErr != nil || keyErr != nil {
					certExists = false
					if crtErr != nil {
						addErrorLog("caddy", fmt.Sprintf("專案 %s: 憑證路徑不安全 (%v)，使用自動 TLS", proj.Name, crtErr), nil)
					}
					if keyErr != nil {
						addErrorLog("caddy", fmt.Sprintf("專案 %s: 金鑰路徑不安全 (%v)，使用自動 TLS", proj.Name, keyErr), nil)
					}
				} else {
					// 檢查憑證與金鑰檔案是否存在
					if _, err := os.Stat(crt); os.IsNotExist(err) {
						certExists = false
						addErrorLog("caddy", fmt.Sprintf("專案 %s: 憑證遺失，使用自動 TLS", proj.Name), nil)
					} else if _, err := os.Stat(key); os.IsNotExist(err) {
						certExists = false
						addErrorLog("caddy", fmt.Sprintf("專案 %s: 金鑰遺失，使用自動 TLS", proj.Name), nil)
					}
				}
			}
			if certExists {
				content += fmt.Sprintf("\ttls %s %s\n", safeCrt, safeKey)
			} else {
				content += "\ttls internal\n"
			}
		}

		// IP Allow rule or others
		content += "\timport common_dev\n"

		if preset.IsRuntimeProject(proj.Type) {
			port := proj.RuntimePort
			if port == 0 {
				port = 3000
			}
			content += fmt.Sprintf("\treverse_proxy localhost:%d\n", port)
		} else {
			// Root
			root := appCfg.GetProjectRoot(proj, baseDir)
			root = strings.ReplaceAll(root, "\\", "/")
			content += fmt.Sprintf("\troot * %s\n", root)

			if proj.PHPVersion != "" {
				phpVerStr := strings.ReplaceAll(proj.PHPVersion, ".", "")
				content += fmt.Sprintf("\timport php%s\n", phpVerStr)
			}

			content += "\timport static_site\n"
		}
		content += "}\n"

		if err := os.WriteFile(caddyPath, []byte(content), 0600); err != nil {
			return fmt.Errorf("寫入 Caddy 設定檔 %s 失敗: %w", caddyPath, err)
		}
	}
	return nil
}

// regenerateCaddyAndReload 重新產生 Caddy 設定檔，若 Caddy 運行中則自動 Reload 並更新 Hosts
func regenerateCaddyAndReload() error {
	if err := generateCaddyfiles(); err != nil {
		return err
	}
	if procMgr.IsRunning("caddy") {
		exePath := procMgr.GetExePath("caddy")
		if err := procMgr.ReloadCaddy(exePath); err != nil {
			addErrorLog("system", "Reload Caddy 失敗", err)
		} else {
			addLog("system", "✅ Caddy 設定已重新載入")
		}
	}
	triggerHostsUpdate()
	return nil
}

// checkPHPForProjects 檢查已啟用專案所需的 PHP 版本是否已啟動，
// 若有未啟動的 PHP 版本，輸出 Log 並彈出 Dialog 提醒。
func checkPHPForProjects(win fyne.Window) {
	type projectPHPStatus struct {
		ProjectName string
		PHPVersion  string // MajorMin, e.g. "8.2"
		IsRunning   bool
		PHPInfo     *scanner.PHPVersionInfo
	}

	var statuses []projectPHPStatus

	for _, proj := range appCfg.Projects {
		if !proj.Enabled || proj.PHPVersion == "" {
			continue
		}

		running := false
		var matchedPHPInfo *scanner.PHPVersionInfo
		for i := range scanRes.PHPList {
			phpInfo := &scanRes.PHPList[i]
			if phpInfo.MajorMin == proj.PHPVersion {
				serviceKey := process.PHPServiceKey(phpInfo.Version)
				if procMgr.IsRunning(serviceKey) {
					running = true
				}
				matchedPHPInfo = phpInfo
				break
			}
		}

		statuses = append(statuses, projectPHPStatus{
			ProjectName: proj.Name,
			PHPVersion:  proj.PHPVersion,
			IsRunning:   running,
			PHPInfo:     matchedPHPInfo,
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

	var toStart []*scanner.PHPVersionInfo
	started := make(map[string]bool)
	for _, s := range statuses {
		if !s.IsRunning && s.PHPInfo != nil && !started[s.PHPInfo.MajorMin] {
			toStart = append(toStart, s.PHPInfo)
			started[s.PHPInfo.MajorMin] = true
		}
	}

	if len(toStart) > 0 {
		msgBuilder.WriteString("\n即將啟動：")
		for i, php := range toStart {
			if i > 0 {
				msgBuilder.WriteString("、")
			}
			msgBuilder.WriteString("PHP " + php.MajorMin)
		}
	}
	msgBuilder.WriteString("。")

	lbl := widget.NewLabel(msgBuilder.String())
	lbl.Wrapping = fyne.TextWrapWord
	dialogContent := container.NewVScroll(lbl)
	dialogContent.SetMinSize(fyne.NewSize(350, 200))

	fyne.Do(func() {
		dialog.NewCustomConfirm("⚠️ PHP Service Required", "Run PHP", "Cancel", dialogContent, func(confirm bool) {
			if !confirm {
				return
			}
			for _, phpInfo := range toStart {
				if err := procMgr.StartPHPCGI(*phpInfo); err != nil {
					addErrorLog("php", fmt.Sprintf("啟動 PHP %s 失敗", phpInfo.Version), err)
				} else {
					addLog("php", fmt.Sprintf("✅ PHP %s 已啟動", phpInfo.Version))
				}
			}
			refreshAllPHPStatus()
		}, win).Show()
	})
}

// checkCoreDependencies 檢查 Caddy、PHP 與 MariaDB 等核心元件是否缺失並進行提示
func checkCoreDependencies(win fyne.Window) {
	missingCaddy := len(scanRes.CaddyList) == 0
	missingPHP := len(scanRes.PHPList) == 0
	missingMariaDB := len(scanRes.MariaDBList) == 0

	// 若無缺失任何核心元件則直接返回
	if !missingCaddy && !missingPHP && !missingMariaDB {
		return
	}

	var msgBuilder strings.Builder
	msgBuilder.WriteString("WinCMP detected that you have not configured the required core dependencies:\n\n")

	if missingCaddy {
		msgBuilder.WriteString("  [Missing] Caddy   (Executable not found)\n")
	} else {
		msgBuilder.WriteString("  [Detected] Caddy   (Detected)\n")
	}

	if missingPHP {
		msgBuilder.WriteString("  [Missing] PHP     (Executable not found)\n")
	} else {
		msgBuilder.WriteString("  [Detected] PHP     (Detected)\n")
	}

	if missingMariaDB {
		msgBuilder.WriteString("  [Missing] MariaDB (Executable not found)\n")
	} else {
		msgBuilder.WriteString("  [Detected] MariaDB (Detected)\n")
	}

	msgBuilder.WriteString("\nWould you like to start the automatic download and configuration now?")

	lbl := widget.NewLabel(msgBuilder.String())
	lbl.Wrapping = fyne.TextWrapWord

	dialogContent := container.NewVScroll(lbl)
	dialogContent.SetMinSize(fyne.NewSize(450, 200))

	fyne.Do(func() {
		dialog.NewCustomConfirm("Dependency Missing", "Auto Download (Recommended)", "Configure Later", dialogContent, func(confirm bool) {
			if confirm {
				startAutoDownload(win, missingCaddy, missingPHP, missingMariaDB)
			}
		}, win).Show()
	})
}

// startAutoDownload 執行非同步下載與安裝核心元件
func startAutoDownload(win fyne.Window, missingCaddy, missingPHP, missingMariaDB bool) {
	type depSpec struct {
		name    string
		url     string
		destZip string
		destDir string
	}

	var specs []depSpec
	binDir := filepath.Join(baseDir, "bin")

	if missingCaddy {
		caddyVer := depCfg["caddy"].Version
		specs = append(specs, depSpec{
			name:    "Caddy v" + caddyVer,
			url:     depCfg["caddy"].URL,
			destZip: filepath.Join(binDir, "caddy_"+caddyVer+".zip"),
			destDir: filepath.Join(binDir, "caddy", "caddy-"+caddyVer),
		})
	}

	if missingPHP {
		php83Ver := depCfg["php83"].Version
		specs = append(specs, depSpec{
			name:    "PHP v" + php83Ver + " NTS",
			url:     depCfg["php83"].URL,
			destZip: filepath.Join(binDir, "php_"+php83Ver+".zip"),
			destDir: filepath.Join(binDir, "php", "php-"+php83Ver),
		})
	}

	if missingMariaDB {
		mariaDBVer := depCfg["mariadb"].Version
		specs = append(specs, depSpec{
			name:    "MariaDB v" + mariaDBVer,
			url:     depCfg["mariadb"].URL,
			destZip: filepath.Join(binDir, "mariadb_"+mariaDBVer+".zip"),
			destDir: filepath.Join(binDir, "mariadb"),
		})
	}

	if len(specs) == 0 {
		return
	}

	// 建立下載進度 UI
	titleLabel := widget.NewLabel("Dependency Downloader")
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	titleLabel.Alignment = fyne.TextAlignCenter

	statusLabel := widget.NewLabel("Preparing download environment...")
	statusLabel.Alignment = fyne.TextAlignCenter
	progressBar := widget.NewProgressBar()
	progressBar.SetValue(0)

	var d *widget.PopUp

	// 自訂按鈕元件
	bgBtn := widget.NewButton("Background", func() {
		d.Hide()
	})

	closeBtn := widget.NewButton("Close", func() {
		d.Hide()
	})
	closeBtn.Hide() // 初始隱藏

	buttonBox := container.NewHBox(layout.NewSpacer(), bgBtn, closeBtn, layout.NewSpacer())

	contentBox := container.NewVBox(
		titleLabel,
		widget.NewSeparator(),
		statusLabel,
		progressBar,
		buttonBox,
	)

	// 用透明矩形撐開最小尺寸，避免對話框縮成一小團
	bgRect := canvas.NewRectangle(color.Transparent)
	bgRect.SetMinSize(fyne.NewSize(380, 140))

	dialogContent := container.NewPadded(container.NewStack(bgRect, contentBox))

	d = widget.NewModalPopUp(dialogContent, win.Canvas())
	d.Show()

	go func() {
		hasError := false
		for i, spec := range specs {
			prefix := fmt.Sprintf("[%d/%d] Downloading %s...", i+1, len(specs), spec.name)
			addLog("system", fmt.Sprintf("Downloading core dependency: %s...", spec.name))

			// 執行檔案下載
			err := downloader.DownloadFile(spec.url, spec.destZip, func(current, total int64) {
				var percent float64
				if total > 0 {
					percent = float64(current) / float64(total)
				}
				currentMB := float64(current) / 1024 / 1024
				totalMB := float64(total) / 1024 / 1024

				fyne.Do(func() {
					statusLabel.SetText(fmt.Sprintf("%s\n%.2fMB / %.2fMB", prefix, currentMB, totalMB))
					progressBar.SetValue(percent)
				})
			})

			if err != nil {
				hasError = true
				fyne.Do(func() {
					statusLabel.SetText(fmt.Sprintf("Download failed for %s:\n%v", spec.name, err))
					bgBtn.Hide()
					closeBtn.Show()
					buttonBox.Refresh()
				})
				addErrorLog("system", fmt.Sprintf("Download failed for %s", spec.name), err)
				break
			}

			// 執行解壓縮
			fyne.Do(func() {
				statusLabel.SetText(fmt.Sprintf("[%d/%d] Extracting %s...", i+1, len(specs), spec.name))
				progressBar.SetValue(0.5)
			})
			addLog("system", fmt.Sprintf("Extracting core dependency: %s...", spec.name))

			err = downloader.Unzip(spec.destZip, spec.destDir)
			if err != nil {
				hasError = true
				fyne.Do(func() {
					statusLabel.SetText(fmt.Sprintf("Extraction failed for %s:\n%v", spec.name, err))
					bgBtn.Hide()
					closeBtn.Show()
					buttonBox.Refresh()
				})
				addErrorLog("system", fmt.Sprintf("Extraction failed for %s", spec.name), err)
				break
			}

			// 清理下載的 zip
			os.Remove(spec.destZip)

			// MariaDB 的目錄重命名處理
			if strings.HasPrefix(spec.name, "MariaDB v") {
				version := strings.TrimPrefix(spec.name, "MariaDB v")
				oldDir := filepath.Join(binDir, "mariadb", "mariadb-"+version+"-winx64")
				newDir := filepath.Join(binDir, "mariadb", "mariadb-"+version)
				if _, err := os.Stat(oldDir); err == nil {
					if _, err := os.Stat(newDir); os.IsNotExist(err) {
						if renameErr := os.Rename(oldDir, newDir); renameErr != nil {
							addErrorLog("system", "MariaDB directory rename failed", renameErr)
						}
					}
				}
			}
			addLog("system", fmt.Sprintf("Installed core dependency: %s", spec.name))
		}

		if !hasError {
			fyne.Do(func() {
				statusLabel.SetText("All missing dependencies have been downloaded and configured!")
				progressBar.SetValue(1.0)
				bgBtn.Hide()
				closeBtn.Show()
				buttonBox.Refresh()

				// 自動重新掃描並更新介面
				var scanErr error
				scanRes, scanErr = scanner.ScanBinDir(baseDir)
				if scanErr != nil {
					addErrorLog("system", "Rescan failed", scanErr)
				} else {
					addLog("system", "Rescan completed, reloading dashboard...")
					newDashboard := createDashboard(win, func() {})
					mainTabs.Items[0].Content = newDashboard
					mainTabs.Refresh()
				}
			})
		}
	}()
}

// compareVersions 比較兩個版本號字串大小 (如 v1 < v2 回傳 -1，v1 > v2 回傳 1，相等回傳 0)
func compareVersions(v1, v2 string) int {
	clean := func(v string) string {
		v = strings.TrimPrefix(v, "v")
		v = strings.Split(v, "-")[0]
		return v
	}
	v1 = clean(v1)
	v2 = clean(v2)

	p1 := strings.Split(v1, ".")
	p2 := strings.Split(v2, ".")

	for i := 0; i < len(p1) || i < len(p2); i++ {
		var n1, n2 int
		if i < len(p1) {
			n1, _ = strconv.Atoi(p1[i])
		}
		if i < len(p2) {
			n2, _ = strconv.Atoi(p2[i])
		}
		if n1 < n2 {
			return -1
		} else if n1 > n2 {
			return 1
		}
	}
	return 0
}

func getLocalCaddyVersion() string {
	if len(scanRes.CaddyList) > 0 {
		return scanRes.CaddyList[0].Version
	}
	return ""
}

func getLocalMariaDBVersion() string {
	if len(scanRes.MariaDBList) > 0 {
		return scanRes.MariaDBList[0].Version
	}
	return ""
}

func getLocalPHPVersion(majorMin string) string {
	for _, php := range scanRes.PHPList {
		if php.MajorMin == majorMin {
			return php.Version
		}
	}
	return ""
}

func getLocalComposerVersion() string {
	if len(scanRes.ComposerList) > 0 {
		return scanRes.ComposerList[0].Version
	}
	return ""
}

func getLocalHeidiSQLVersion() string {
	if len(scanRes.HeidiSQLList) > 0 {
		return scanRes.HeidiSQLList[0].Version
	}
	return ""
}

func getLocalNodeVersion() string {
	if len(scanRes.NodeList) > 0 {
		return scanRes.NodeList[0].Version
	}
	return ""
}

// depSpec 定義依賴元件的下載規格
type depSpec struct {
	name    string
	url     string
	destZip string
	destDir string
}

// depButtonTheme 自定義依賴管理按鈕主題，以不同顏色區分下載、更新與重裝
type depButtonTheme struct {
	action string // "Download", "Update", "Reinstall"
}

func (m *depButtonTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameForeground:
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}
	case theme.ColorNameButton:
		switch m.action {
		case "Download":
			// 藍色 (吸引點擊)
			return color.RGBA{R: 13, G: 110, B: 253, A: 255}
		case "Update":
			// 橘色 (警示有新版本)
			return color.RGBA{R: 253, G: 126, B: 20, A: 255}
		case "Reinstall":
			// 深灰色 (次要操作)
			return color.RGBA{R: 108, G: 117, B: 125, A: 255}
		}
	}
	return fyne.CurrentApp().Settings().Theme().Color(name, variant)
}

func (m *depButtonTheme) Font(style fyne.TextStyle) fyne.Resource {
	return fyne.CurrentApp().Settings().Theme().Font(style)
}

func (m *depButtonTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return fyne.CurrentApp().Settings().Theme().Icon(name)
}

func (m *depButtonTheme) Size(name fyne.ThemeSizeName) float32 {
	return fyne.CurrentApp().Settings().Theme().Size(name)
}

func createDependencyRow(win fyne.Window, d *dialog.Dialog, name string, localVer, recVer string, spec depSpec) fyne.CanvasObject {
	nameLabel := widget.NewLabel(name)
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	infoLabel := widget.NewLabel("")
	var btn *widget.Button
	var btnWrapper fyne.CanvasObject

	actionFn := func() {
		if d != nil && *d != nil {
			(*d).Hide()
		}
		startSingleDependencyDownload(win, spec)
	}

	if localVer == "" {
		infoLabel.SetText(fmt.Sprintf("Not Installed (Recommended: %s)", recVer))
		btn = widget.NewButton("Download", actionFn)
		btnWrapper = container.NewThemeOverride(btn, &depButtonTheme{action: "Download"})
	} else {
		cmp := compareVersions(localVer, recVer)
		if cmp < 0 {
			infoLabel.SetText(fmt.Sprintf("Installed: %s (Update available to %s)", localVer, recVer))
			btn = widget.NewButton("Update", actionFn)
			btnWrapper = container.NewThemeOverride(btn, &depButtonTheme{action: "Update"})
		} else {
			infoLabel.SetText(fmt.Sprintf("Installed: %s (Up to date)", localVer))
			btn = widget.NewButton("Reinstall", actionFn)
			btnWrapper = container.NewThemeOverride(btn, &depButtonTheme{action: "Reinstall"})
		}
	}

	return container.NewGridWithColumns(3, nameLabel, infoLabel, btnWrapper)
}

func showDependencyManager(win fyne.Window) {
	// 每次點開 Dependency Manager 時，重新從磁碟讀取 dependencies.json
	depCfgPath := filepath.Join(baseDir, "conf", "dependencies.json")
	if loaded, err := config.LoadDependencies(depCfgPath); err == nil {
		depCfg = loaded
		addLog("system", fmt.Sprintf("✓ 重新載入依賴設定檔成功，Caddy: %s, HeidiSQL: %s, Node: %s", depCfg["caddy"].Version, depCfg["heidisql"].Version, depCfg["node"].Version))
	} else {
		addErrorLog("system", "重新載入依賴設定檔失敗，將使用記憶體中現有配置", err)
	}

	binDir := filepath.Join(baseDir, "bin")

	caddyVer := depCfg["caddy"].Version
	caddyUrl := depCfg["caddy"].URL
	caddySpec := depSpec{
		name:    "Caddy v" + caddyVer,
		url:     caddyUrl,
		destZip: filepath.Join(binDir, "caddy_"+caddyVer+".zip"),
		destDir: filepath.Join(binDir, "caddy", "caddy-"+caddyVer),
	}

	mariaDBVer := depCfg["mariadb"].Version
	mariaDBUrl := depCfg["mariadb"].URL
	mariaDBSpec := depSpec{
		name:    "MariaDB v" + mariaDBVer,
		url:     mariaDBUrl,
		destZip: filepath.Join(binDir, "mariadb_"+mariaDBVer+".zip"),
		destDir: filepath.Join(binDir, "mariadb"),
	}

	php73Ver := depCfg["php73"].Version
	php73Url := depCfg["php73"].URL
	php73Spec := depSpec{
		name:    "PHP v" + php73Ver,
		url:     php73Url,
		destZip: filepath.Join(binDir, "php_"+php73Ver+".zip"),
		destDir: filepath.Join(binDir, "php", "php-"+php73Ver),
	}

	php82Ver := depCfg["php82"].Version
	php82Url := depCfg["php82"].URL
	php82Spec := depSpec{
		name:    "PHP v" + php82Ver,
		url:     php82Url,
		destZip: filepath.Join(binDir, "php_"+php82Ver+".zip"),
		destDir: filepath.Join(binDir, "php", "php-"+php82Ver),
	}

	php83Ver := depCfg["php83"].Version
	php83Url := depCfg["php83"].URL
	php83Spec := depSpec{
		name:    "PHP v" + php83Ver,
		url:     php83Url,
		destZip: filepath.Join(binDir, "php_"+php83Ver+".zip"),
		destDir: filepath.Join(binDir, "php", "php-"+php83Ver),
	}

	composerVer := depCfg["composer"].Version
	composerUrl := depCfg["composer"].URL
	composerSpec := depSpec{
		name:    "Composer v" + composerVer,
		url:     composerUrl,
		destZip: filepath.Join(binDir, "composer", "composer-"+composerVer, "composer.phar"),
		destDir: filepath.Join(binDir, "composer", "composer-"+composerVer),
	}

	heidiSQLVer := depCfg["heidisql"].Version
	heidiSQLUrl := depCfg["heidisql"].URL
	heidiSQLSpec := depSpec{
		name:    "HeidiSQL v" + heidiSQLVer,
		url:     heidiSQLUrl,
		destZip: filepath.Join(binDir, "heidisql_"+heidiSQLVer+".zip"),
		destDir: filepath.Join(binDir, "heidisql", "heidisql-"+heidiSQLVer),
	}

	nodeVer := depCfg["node"].Version
	nodeUrl := depCfg["node"].URL
	nodeSpec := depSpec{
		name:    "Node.js v" + nodeVer,
		url:     nodeUrl,
		destZip: filepath.Join(binDir, "node_"+nodeVer+".zip"),
		destDir: filepath.Join(binDir, "node"),
	}

	var d dialog.Dialog

	coreRows := container.NewVBox(
		createDependencyRow(win, &d, "Caddy Server", getLocalCaddyVersion(), caddyVer, caddySpec),
		widget.NewSeparator(),
		createDependencyRow(win, &d, "MariaDB Database", getLocalMariaDBVersion(), mariaDBVer, mariaDBSpec),
		widget.NewSeparator(),
		createDependencyRow(win, &d, "PHP 7.3 NTS", getLocalPHPVersion("7.3"), php73Ver, php73Spec),
		widget.NewSeparator(),
		createDependencyRow(win, &d, "PHP 8.2 NTS", getLocalPHPVersion("8.2"), php82Ver, php82Spec),
		widget.NewSeparator(),
		createDependencyRow(win, &d, "PHP 8.3 NTS", getLocalPHPVersion("8.3"), php83Ver, php83Spec),
	)
	coreCard := widget.NewCard("Core Dependencies", "Web server, database and PHP versions", coreRows)

	otherRows := container.NewVBox(
		createDependencyRow(win, &d, "Composer", getLocalComposerVersion(), composerVer, composerSpec),
		widget.NewSeparator(),
		createDependencyRow(win, &d, "HeidiSQL", getLocalHeidiSQLVersion(), heidiSQLVer, heidiSQLSpec),
		widget.NewSeparator(),
		createDependencyRow(win, &d, "Node.js LTS", getLocalNodeVersion(), nodeVer, nodeSpec),
	)
	otherCard := widget.NewCard("Other Dependencies", "Package managers, GUI tools and runtime systems", otherRows)

	fetchBtn := widget.NewButtonWithIcon("取得最新建議版本", theme.DownloadIcon(), func() {
		fetchLatestDependencies(win, d)
	})

	scrollContent := container.NewVScroll(container.NewVBox(fetchBtn, coreCard, otherCard))
	scrollContent.SetMinSize(fyne.NewSize(650, 420))

	d = dialog.NewCustom("Dependency Manager", "Close", scrollContent, win)
	d.Show()
}

func fetchLatestDependencies(win fyne.Window, d dialog.Dialog) {
	progress := dialog.NewProgressInfinite("取得最新版本", "正在從遠端獲取最新依賴資訊...", win)
	progress.Show()

	go func() {
		defer progress.Hide()

		url := "https://raw.githubusercontent.com/wukh1124/wincmp/main/conf/dependencies.json"
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			fyne.Do(func() {
				dialog.ShowError(fmt.Errorf("無法連線至伺服器: %w", err), win)
			})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fyne.Do(func() {
				dialog.ShowError(fmt.Errorf("伺服器回應錯誤: %s", resp.Status), win)
			})
			return
		}

		var newCfg config.DependencyConfig
		if err := json.NewDecoder(resp.Body).Decode(&newCfg); err != nil {
			fyne.Do(func() {
				dialog.ShowError(fmt.Errorf("解析資料失敗: %w", err), win)
			})
			return
		}

		// 簡單驗證資料完整性
		requiredKeys := []string{"caddy", "mariadb", "php73", "php82", "php83", "composer", "heidisql", "node"}
		for _, key := range requiredKeys {
			if _, ok := newCfg[key]; !ok {
				fyne.Do(func() {
					dialog.ShowError(fmt.Errorf("取得的設定檔格式不正確，缺少欄位: %s", key), win)
				})
				return
			}
		}

		// 儲存至本地
		depCfgPath := filepath.Join(baseDir, "conf", "dependencies.json")
		if err := config.SaveDependencies(depCfgPath, newCfg); err != nil {
			fyne.Do(func() {
				dialog.ShowError(fmt.Errorf("儲存設定檔失敗: %w", err), win)
			})
			return
		}

		// 更新記憶體中的依賴配置
		depCfg = newCfg

		fyne.Do(func() {
			dialog.ShowInformation("成功", "已成功更新最新建議依賴版本！", win)
			if d != nil {
				d.Hide()
			}
			showDependencyManager(win)
		})
	}()
}

func startSingleDependencyDownload(win fyne.Window, spec depSpec) {
	titleLabel := widget.NewLabel("Dependency Downloader")
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	titleLabel.Alignment = fyne.TextAlignCenter

	statusLabel := widget.NewLabel("Preparing to download...")
	statusLabel.Alignment = fyne.TextAlignCenter
	progressBar := widget.NewProgressBar()
	progressBar.SetValue(0)

	var d *widget.PopUp

	// 自訂按鈕元件
	bgBtn := widget.NewButton("Background", func() {
		d.Hide()
	})

	backBtn := widget.NewButton("Back", func() {
		d.Hide()
		showDependencyManager(win)
	})
	backBtn.Hide() // 初始隱藏

	closeBtn := widget.NewButton("Close", func() {
		d.Hide()
	})
	closeBtn.Hide() // 初始隱藏

	buttonBox := container.NewHBox(layout.NewSpacer(), bgBtn, backBtn, closeBtn, layout.NewSpacer())

	contentBox := container.NewVBox(
		titleLabel,
		widget.NewSeparator(),
		statusLabel,
		progressBar,
		buttonBox,
	)

	// 用透明矩形撐開最小尺寸，避免對話框縮成一小團
	bgRect := canvas.NewRectangle(color.Transparent)
	bgRect.SetMinSize(fyne.NewSize(380, 140))

	dialogContent := container.NewPadded(container.NewStack(bgRect, contentBox))

	d = widget.NewModalPopUp(dialogContent, win.Canvas())
	d.Show()

	go func() {
		hasError := false
		addLog("system", fmt.Sprintf("Starting download for: %s...", spec.name))

		err := downloader.DownloadFile(spec.url, spec.destZip, func(current, total int64) {
			var percent float64
			if total > 0 {
				percent = float64(current) / float64(total)
			}
			currentMB := float64(current) / 1024 / 1024
			totalMB := float64(total) / 1024 / 1024

			fyne.Do(func() {
				statusLabel.SetText(fmt.Sprintf("Downloading %s...\n%.2fMB / %.2fMB", spec.name, currentMB, totalMB))
				progressBar.SetValue(percent)
			})
		})

		if err != nil {
			hasError = true
			fyne.Do(func() {
				statusLabel.SetText(fmt.Sprintf("Download failed for %s:\n%v", spec.name, err))
				bgBtn.Hide()
				backBtn.Show()
				closeBtn.Show()
				buttonBox.Refresh()
			})
			addErrorLog("system", fmt.Sprintf("Download failed for %s", spec.name), err)
			return
		}

		binDir := filepath.Join(baseDir, "bin")
		if strings.HasSuffix(spec.destZip, ".zip") {
			fyne.Do(func() {
				statusLabel.SetText(fmt.Sprintf("Extracting %s...", spec.name))
				progressBar.SetValue(0.5)
			})
			addLog("system", fmt.Sprintf("Extracting: %s...", spec.name))

			err = downloader.Unzip(spec.destZip, spec.destDir)
			if err != nil {
				hasError = true
				fyne.Do(func() {
					statusLabel.SetText(fmt.Sprintf("Extraction failed for %s:\n%v", spec.name, err))
					bgBtn.Hide()
					backBtn.Show()
					closeBtn.Show()
					buttonBox.Refresh()
				})
				addErrorLog("system", fmt.Sprintf("Extraction failed for %s", spec.name), err)
				return
			}

			os.Remove(spec.destZip)

			if strings.HasPrefix(spec.name, "MariaDB v") {
				version := strings.TrimPrefix(spec.name, "MariaDB v")
				oldDir := filepath.Join(binDir, "mariadb", "mariadb-"+version+"-winx64")
				newDir := filepath.Join(binDir, "mariadb", "mariadb-"+version)
				if _, err := os.Stat(oldDir); err == nil {
					if _, err := os.Stat(newDir); os.IsNotExist(err) {
						if renameErr := os.Rename(oldDir, newDir); renameErr != nil {
							addErrorLog("system", "MariaDB directory rename failed", renameErr)
						}
					}
				}
			}

			if strings.HasPrefix(spec.name, "Node.js v") {
				version := strings.TrimPrefix(spec.name, "Node.js v")
				oldDir := filepath.Join(binDir, "node", "node-v"+version+"-win-x64")
				newDir := filepath.Join(binDir, "node", "node-"+version)
				if _, err := os.Stat(oldDir); err == nil {
					if _, err := os.Stat(newDir); os.IsNotExist(err) {
						if renameErr := os.Rename(oldDir, newDir); renameErr != nil {
							addErrorLog("system", "Node.js directory rename failed", renameErr)
						}
					}
				}
			}
		} else {
			if strings.Contains(spec.name, "Composer") {
				batPath := filepath.Join(spec.destDir, "composer.bat")
				batContent := `@php "%~dp0composer.phar" %*`
				if err := os.WriteFile(batPath, []byte(batContent), 0755); err != nil {
					addErrorLog("system", "Failed to create composer.bat", err)
				}
			}
		}

		if !hasError {
			fyne.Do(func() {
				statusLabel.SetText(fmt.Sprintf("%s has been installed and configured!", spec.name))
				progressBar.SetValue(1.0)
				bgBtn.Hide()
				backBtn.Show()
				closeBtn.Show()
				buttonBox.Refresh()

				var scanErr error
				scanRes, scanErr = scanner.ScanBinDir(baseDir)
				if scanErr != nil {
					addErrorLog("system", "Rescan failed", scanErr)
				} else {
					addLog("system", fmt.Sprintf("Installation of %s completed, reloading dashboard...", spec.name))
					newDashboard := createDashboard(win, func() {})
					mainTabs.Items[0].Content = newDashboard
					mainTabs.Refresh()
				}
			})
		}
	}()
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
				{Service: "Caddy", Port: 80},
				{Service: "Caddy", Port: 443},
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
			if err := regenerateCaddyAndReload(); err != nil {
				addErrorLog("caddy", "生成配置失敗", err)
				return
			}
			if err := procMgr.StartCaddy(info.Version, info.ExePath); err != nil {
				addErrorLog("caddy", "啟動 Caddy 失敗", err)
				return
			}
			pids := procMgr.GetPIDs("caddy")
			if len(pids) > 0 {
				statusLabel.SetText(fmt.Sprintf("Running (PID: %d)", pids[0]))
			}
			actionBtn.SetText("Stop")
			actionBtn.SetIcon(theme.CancelIcon())
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
			actionBtn.SetIcon(theme.MediaPlayIcon())
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
		actionBtn.SetIcon(theme.CancelIcon())
		reloadBtn.Enable()
		monitorUptime("caddy", uptimeData)
	} else {
		actionBtn.SetIcon(theme.MediaPlayIcon())
	}

	// 初始化主題包裝器 (使用閉包檢查按鈕文字)
	actionBtnWrapper := container.NewThemeOverride(actionBtn, &coloredButtonTheme{
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
func createMariaDBRow(win fyne.Window, info scanner.ServiceInfo) fyne.CanvasObject {
	statusLabel := widget.NewLabel("Stopped")
	uptimeData := binding.NewString()
	uptimeData.Set("")
	uptimeLabel := widget.NewLabelWithData(uptimeData)

	var actionBtn *widget.Button

	isExternal := appCfg.Global.MariaDBExternal
	serviceKey := process.MariaDBServiceKey(info.Version)
	serviceName := "MariaDB"
	versionLabel := info.Version

	if isExternal {
		serviceKey = process.MariaDBExternalServiceKey
		serviceName = appCfg.Global.MariaDBType
		versionLabel = "External"
	}

	actionBtn = widget.NewButton("Start", func() {
		if !procMgr.IsRunning(serviceKey) {
			checkPort := 3306
			if appCfg.Global.MariaDBPort > 0 {
				checkPort = appCfg.Global.MariaDBPort
			}
			blocked := port.CheckPorts([]port.PortInfo{
				{Service: "MariaDB", Port: checkPort},
			})
			if len(blocked) > 0 {
				for _, p := range blocked {
					addErrorLog("mariadb", fmt.Sprintf("通訊埠 %d 被佔用，無法啟動 %s", p.Port, serviceName), nil)
				}
				return
			}

			startMariaDBWithUI := func() {
				overlay := showCenterOverlay(win, "啟動資料庫中，請稍候...", color.White, 180)

				go func() {
					done, errCh := procMgr.StartMariaDBAsync(
						info.Version,
						isExternal,
						appCfg.Global.MariaDBBasedir,
						appCfg.Global.MariaDBDatadir,
						appCfg.Global.MariaDBType,
						appCfg.Global.MariaDBPort,
					)
					<-done
					fyne.Do(func() {
						hideCenterOverlay(win, overlay)
					})

					err := <-errCh
					if err != nil {
						addErrorLog("mariadb", "啟動 "+serviceName+" 失敗", err)
						return
					}
					pids := procMgr.GetPIDs(serviceKey)
					if len(pids) > 0 {
						fyne.Do(func() {
							statusLabel.SetText(fmt.Sprintf("Running (PID: %d)", pids[0]))
						})
					}
					fyne.Do(func() {
						actionBtn.SetText("Stop")
						actionBtn.SetIcon(theme.CancelIcon())
					})
					monitorUptime(serviceKey, uptimeData)
					saveLastServiceState()
				}()
			}

			if !isExternal {
				baseDir := procMgr.GetBaseDir()
				mysqlDBPath := filepath.Join(baseDir, "data", "mariadb", "mysql")
				needsInit := false
				if _, err := os.Stat(mysqlDBPath); os.IsNotExist(err) {
					needsInit = true
				}

				dataDir := filepath.Join(baseDir, "data", "mariadb")

				if needsInit {
					dialog.ShowConfirm(
						"MariaDB 初始化確認",
						fmt.Sprintf("MariaDB 資料庫即將進行初始化（約需 10-30 秒）。\n\n路徑: %s\n\n初始化將會清空上述路徑下的資料，確定要繼續嗎？", dataDir),
						func(confirmed bool) {
							if confirmed {
								startMariaDBWithUI()
							}
						},
						win,
					)
					return
				}
			}

			startMariaDBWithUI()
		} else {
			if err := procMgr.StopMariaDB(
				info.Version,
				isExternal,
				appCfg.Global.MariaDBBasedir,
				appCfg.Global.MariaDBType,
				appCfg.Global.MariaDBPort,
			); err != nil {
				addErrorLog("mariadb", "停止 "+serviceName+" 失敗", err)
				return
			}
			statusLabel.SetText("Stopped")
			actionBtn.SetText("Start")
			actionBtn.SetIcon(theme.MediaPlayIcon())
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
		actionBtn.SetIcon(theme.CancelIcon())
		monitorUptime(serviceKey, uptimeData)
	} else {
		actionBtn.SetIcon(theme.MediaPlayIcon())
	}

	actionBtnWrapper := container.NewThemeOverride(actionBtn, &coloredButtonTheme{
		isStop: func() bool { return actionBtn.Text == "Stop" },
	})
	originalCallback := actionBtn.OnTapped
	actionBtn.OnTapped = func() {
		originalCallback()
		actionBtn.Refresh()
	}

	portStr := "3306"
	if appCfg.Global.MariaDBPort > 0 {
		portStr = fmt.Sprintf("%d", appCfg.Global.MariaDBPort)
	}

	settingsBtn := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		showMariaDBSettingsDialog(win)
	})

	actionGroup := container.NewBorder(nil, nil, nil, settingsBtn, actionBtnWrapper)

	return container.NewGridWithColumns(6,
		widget.NewLabel(serviceName),
		widget.NewLabel(versionLabel),
		statusLabel,
		uptimeLabel,
		widget.NewLabel(portStr),
		actionGroup,
	)
}

// createMailpitRow 建立 Mailpit 服務列
func createMailpitRow(win fyne.Window, info scanner.ServiceInfo) fyne.CanvasObject {
	statusLabel := widget.NewLabel("Stopped")
	uptimeData := binding.NewString()
	uptimeData.Set("")
	uptimeLabel := widget.NewLabelWithData(uptimeData)

	var actionBtn *widget.Button

	smtpPort := 1025
	if appCfg.Global.MailpitSMTPPort > 0 {
		smtpPort = appCfg.Global.MailpitSMTPPort
	}
	httpPort := 8025
	if appCfg.Global.MailpitHTTPPort > 0 {
		httpPort = appCfg.Global.MailpitHTTPPort
	}

	portStr := fmt.Sprintf("%d, %d", smtpPort, httpPort)

	actionBtn = widget.NewButton("Start", func() {
		if !procMgr.IsRunning(process.MailpitServiceKey()) {
			blocked := port.CheckPorts([]port.PortInfo{
				{Service: "Mailpit SMTP", Port: smtpPort},
				{Service: "Mailpit HTTP", Port: httpPort},
			})
			if len(blocked) > 0 {
				for _, p := range blocked {
					addErrorLog("mailpit", fmt.Sprintf("通訊埠 %d 被佔用，無法啟動 Mailpit", p.Port), nil)
				}
				return
			}

			if err := procMgr.StartMailpit(info.Version, info.ExePath, smtpPort, httpPort, appCfg.Global.MailpitUseDB); err != nil {
				addErrorLog("mailpit", "啟動 Mailpit 失敗", err)
				return
			}
			pids := procMgr.GetPIDs(process.MailpitServiceKey())
			if len(pids) > 0 {
				statusLabel.SetText(fmt.Sprintf("Running (PID: %d)", pids[0]))
			}
			actionBtn.SetText("Stop")
			actionBtn.SetIcon(theme.CancelIcon())
			monitorUptime(process.MailpitServiceKey(), uptimeData)
			saveLastServiceState()
		} else {
			if err := procMgr.StopMailpit(); err != nil {
				addErrorLog("mailpit", "停止 Mailpit 失敗", err)
				return
			}
			statusLabel.SetText("Stopped")
			actionBtn.SetText("Start")
			actionBtn.SetIcon(theme.MediaPlayIcon())
			uptimeData.Set("")
			saveLastServiceState()
		}
	})

	if procMgr.IsRunning(process.MailpitServiceKey()) {
		pids := procMgr.GetPIDs(process.MailpitServiceKey())
		if len(pids) > 0 {
			statusLabel.SetText(fmt.Sprintf("Running (PID: %d)", pids[0]))
		}
		actionBtn.SetText("Stop")
		actionBtn.SetIcon(theme.CancelIcon())
		monitorUptime(process.MailpitServiceKey(), uptimeData)
	} else {
		actionBtn.SetIcon(theme.MediaPlayIcon())
	}

	actionBtnWrapper := container.NewThemeOverride(actionBtn, &coloredButtonTheme{
		isStop: func() bool { return actionBtn.Text == "Stop" },
	})
	originalCallback := actionBtn.OnTapped
	actionBtn.OnTapped = func() {
		originalCallback()
		actionBtn.Refresh()
	}

	settingsBtn := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		showMailpitSettingsDialog(win)
	})

	actionGroup := container.NewBorder(nil, nil, nil, settingsBtn, actionBtnWrapper)

	return container.NewGridWithColumns(6,
		widget.NewLabel("Mailpit"),
		widget.NewLabel(info.Version),
		statusLabel,
		uptimeLabel,
		widget.NewLabel(portStr),
		actionGroup,
	)
}

// showMailpitSettingsDialog 顯示 Mailpit 設定對話框
func showMailpitSettingsDialog(win fyne.Window) {
	smtpPortEntry := newScrollPassthroughEntry()
	smtpPortEntry.SetPlaceHolder("1025")
	if appCfg.Global.MailpitSMTPPort > 0 {
		smtpPortEntry.SetText(fmt.Sprintf("%d", appCfg.Global.MailpitSMTPPort))
	} else {
		smtpPortEntry.SetText("1025")
	}

	httpPortEntry := newScrollPassthroughEntry()
	httpPortEntry.SetPlaceHolder("8025")
	if appCfg.Global.MailpitHTTPPort > 0 {
		httpPortEntry.SetText(fmt.Sprintf("%d", appCfg.Global.MailpitHTTPPort))
	} else {
		httpPortEntry.SetText("8025")
	}

	useDBCheck := widget.NewCheck("持久化存儲 (Database)", nil)
	useDBCheck.SetChecked(appCfg.Global.MailpitUseDB)

	httpPort := 8025
	if appCfg.Global.MailpitHTTPPort > 0 {
		httpPort = appCfg.Global.MailpitHTTPPort
	}

	copyURLBtn := widget.NewButtonWithIcon("複製網址", theme.ContentCopyIcon(), func() {
		url := fmt.Sprintf("http://localhost:%d", httpPort)
		win.Clipboard().SetContent(url)
		addLog("mailpit", fmt.Sprintf("已複製 Mailpit 網址: %s", url))
	})

	smtpHint := canvas.NewText("SMTP 端口，用於接收郵件（預設 1025）", color.NRGBA{R: 150, G: 150, B: 150, A: 255})
	smtpHint.TextSize = 10
	httpHint := canvas.NewText("網頁管理介面端口（預設 8025）", color.NRGBA{R: 150, G: 150, B: 150, A: 255})
	httpHint.TextSize = 10
	dbHint := canvas.NewText("啟用後郵件將保存至 data/mailpit 目錄，重啟後不遺失", color.NRGBA{R: 150, G: 150, B: 150, A: 255})
	dbHint.TextSize = 10

	form := widget.NewForm(
		widget.NewFormItem("SMTP Port", container.NewVBox(smtpPortEntry, smtpHint)),
		widget.NewFormItem("HTTP Port", container.NewVBox(httpPortEntry, httpHint)),
		widget.NewFormItem("Web UI", copyURLBtn),
		widget.NewFormItem("Data Storage", container.NewVBox(useDBCheck, dbHint)),
	)

	d := dialog.NewCustomConfirm("Mailpit Settings", "Save", "Cancel", container.NewVScroll(form), func(save bool) {
		if !save {
			return
		}

		smtpPortVal, err := strconv.Atoi(smtpPortEntry.Text)
		if err != nil || smtpPortVal < 1 || smtpPortVal > 65535 {
			dialog.ShowError(fmt.Errorf("SMTP Port 必須是 1-65535 的數字"), win)
			return
		}

		httpPortVal, err := strconv.Atoi(httpPortEntry.Text)
		if err != nil || httpPortVal < 1 || httpPortVal > 65535 {
			dialog.ShowError(fmt.Errorf("HTTP Port 必須是 1-65535 的數字"), win)
			return
		}

		if smtpPortVal == httpPortVal {
			dialog.ShowError(fmt.Errorf("SMTP Port 和 HTTP Port 不能相同"), win)
			return
		}

		blocked := port.CheckPorts([]port.PortInfo{
			{Service: "Mailpit SMTP", Port: smtpPortVal},
			{Service: "Mailpit HTTP", Port: httpPortVal},
		})

		// 如果 Mailpit 正在運行，過濾掉自己佔用的端口
		if procMgr.IsRunning(process.MailpitServiceKey()) {
			currentSmtp := 1025
			currentHttp := 8025
			if appCfg.Global.MailpitSMTPPort > 0 {
				currentSmtp = appCfg.Global.MailpitSMTPPort
			}
			if appCfg.Global.MailpitHTTPPort > 0 {
				currentHttp = appCfg.Global.MailpitHTTPPort
			}
			filtered := make([]port.PortInfo, 0)
			for _, p := range blocked {
				if p.Port == currentSmtp || p.Port == currentHttp {
					continue
				}
				filtered = append(filtered, p)
			}
			blocked = filtered
		}
		if len(blocked) > 0 {
			for _, p := range blocked {
				dialog.ShowError(fmt.Errorf("Port %d 已被佔用", p.Port), win)
				return
			}
		}

		appCfg.Global.MailpitSMTPPort = smtpPortVal
		appCfg.Global.MailpitHTTPPort = httpPortVal
		appCfg.Global.MailpitUseDB = useDBCheck.Checked

		cfgPath := filepath.Join(baseDir, "conf", "wincmp.json")
		if err := appCfg.Save(cfgPath); err != nil {
			dialog.ShowError(fmt.Errorf("儲存設定失敗: %v", err), win)
			return
		}

		dbMode := "記憶體"
		if useDBCheck.Checked {
			dbMode = "持久化"
		}
		addLog("mailpit", fmt.Sprintf("Mailpit 設定已儲存: SMTP=%d, HTTP=%d, 存儲=%s", smtpPortVal, httpPortVal, dbMode))

		// 重新整理 Dashboard
		newDashboard := createDashboard(win, func() {})
		mainTabs.Items[0].Content = newDashboard
		mainTabs.Refresh()
	}, win)

	d.Resize(fyne.NewSize(440, 380))
	d.Show()
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
			a, _ := strconv.Atoi(options[i])
			b, _ := strconv.Atoi(options[j])
			return a < b
		})
	}

	processSelect := widget.NewSelect(options, nil)
	processSelect.SetSelected(fmt.Sprintf("%d", currentCount))
	processSelect.OnChanged = func(val string) {
		count := 3
		if v, err := strconv.Atoi(val); err == nil && v > 0 {
			count = v
		}

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
			monitorUptime(serviceKey, uptimeData)
			saveLastServiceState()
			refreshAllPHPStatus()
		} else {
			if err := procMgr.StopPHPCGI(info.Version); err != nil {
				addErrorLog("php", "停止 PHP-CGI 失敗", err)
				return
			}
			saveLastServiceState()
			refreshAllPHPStatus()
		}
	})

	if procMgr.IsRunning(serviceKey) {
		pids := procMgr.GetPIDs(serviceKey)
		statusLabel.SetText(fmt.Sprintf("Running (%d PIDs)", len(pids)))
		actionBtn.SetText("Stop")
		actionBtn.SetIcon(theme.CancelIcon())
		processSelect.Disable()
		monitorUptime(serviceKey, uptimeData)
	} else {
		actionBtn.SetIcon(theme.MediaPlayIcon())
	}

	actionBtnWrapper := container.NewThemeOverride(actionBtn, &coloredButtonTheme{
		isStop: func() bool { return actionBtn.Text == "Stop" },
	})
	originalCallback := actionBtn.OnTapped
	actionBtn.OnTapped = func() {
		originalCallback()
		actionBtn.Refresh()
	}

	if phpRowUI != nil {
		phpRowUI[info.MajorMin] = &phpRowComponents{
			StatusLabel:   statusLabel,
			ActionBtn:     actionBtn,
			ProcessSelect: processSelect,
			UptimeData:    uptimeData,
		}
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

// refreshAllPHPStatus 根據實際進程狀態刷新 Dashboard 上所有 PHP Row 的 UI
func refreshAllPHPStatus() {
	if phpRowUI == nil {
		return
	}

	for majorMin, ui := range phpRowUI {
		var info *scanner.PHPVersionInfo
		for i := range scanRes.PHPList {
			if scanRes.PHPList[i].MajorMin == majorMin {
				info = &scanRes.PHPList[i]
				break
			}
		}
		if info == nil {
			continue
		}

		serviceKey := process.PHPServiceKey(info.Version)
		if procMgr.IsRunning(serviceKey) {
			pids := procMgr.GetPIDs(serviceKey)
			ui.StatusLabel.SetText(fmt.Sprintf("Running (%d PIDs)", len(pids)))
			ui.ActionBtn.SetText("Stop")
			ui.ActionBtn.SetIcon(theme.CancelIcon())
			ui.ProcessSelect.Disable()
			monitorUptime(serviceKey, ui.UptimeData)
		} else {
			ui.StatusLabel.SetText("Stopped")
			ui.ActionBtn.SetText("Start")
			ui.ActionBtn.SetIcon(theme.MediaPlayIcon())
			ui.ProcessSelect.Enable()
			ui.UptimeData.Set("")
		}
	}
}

// ===== 2. 網頁專案管理 =====

// showProjectEditor 顯示專案編輯器
func showProjectEditor(win fyne.Window, proj *config.ProjectConfig, onSave func()) {
	// 保存原始值，用於判斷是否需要停止 Runtime
	origEnabled := proj.Enabled
	origType := proj.Type
	origRuntimePort := proj.RuntimePort

	// --- 基本設定 ---
	nameEntry := newScrollPassthroughEntry()
	nameEntry.SetText(proj.Name)

	// 使用多行輸入框解決捲軸擋住文字的問題
	// 減少內部滾動攔截：Wrapping=TextWrapOff 讓文字不自動換行（但仍可手動換行），
	// Scroll=ScrollVerticalOnly 保留 multiLineRows 高度計算（Ref: entryRenderer.MinSize），
	// 因為 ScrollNone 模式下 MinSize 不使用 multiLineRows，會導致只顯示一行。
	domainsEntry := widget.NewMultiLineEntry()
	domainsEntry.SetMinRowsVisible(3)
	domainsEntry.Wrapping = fyne.TextWrapOff
	domainsEntry.Scroll = fyne.ScrollVerticalOnly
	domainsEntry.SetText(strings.Join(proj.Domains, ", "))
	domainsEntry.PlaceHolder = "e.g. local-project.test, www.project.test"

	rootPathEntry := newScrollPassthroughEntry()
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

	phpVersions := []string{"None"} // 提示空值含義
	for _, p := range scanRes.PHPList {
		phpVersions = append(phpVersions, p.MajorMin)
	}
	phpSelect := widget.NewSelect(phpVersions, nil)
	if proj.PHPVersion == "" {
		phpSelect.SetSelected("None")
	} else {
		phpSelect.SetSelected(proj.PHPVersion)
	}

	basicForm := widget.NewForm(
		widget.NewFormItem("Project Name", nameEntry),
		widget.NewFormItem("Domains (CSV)", domainsEntry),
		widget.NewFormItem("Root path", container.NewBorder(nil, nil, nil, rootBrowse, rootPathEntry)),
		widget.NewFormItem("PHP Version", container.NewVBox(
			phpSelect,
			widget.NewLabelWithStyle("(Empty or 'None' means PHP is disabled for this project)", fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
		)),
	)

	// --- 狀態 ---
	enabledCheck := widget.NewCheck("Project Enabled", nil)
	enabledCheck.Checked = proj.Enabled

	// Project Type 下拉選單 (使用 Preset Labels)
	presetLabels := preset.GetPresetLabels()
	typeSelect := widget.NewSelect(presetLabels, nil)
	currentPreset := preset.GetPreset(proj.Type)
	typeSelect.SetSelected(currentPreset.Label)

	// 硬編碼最小寬度，確保最長的 "Python FastAPI" 顯示完整
	minW := canvas.NewRectangle(color.Transparent)
	minW.SetMinSize(fyne.NewSize(160, 0))

	runtimePortEntry := newScrollPassthroughEntry()
	p := preset.GetPreset(proj.Type)
	portVal := proj.RuntimePort
	if portVal == 0 {
		portVal = p.DefaultPort
		if portVal == 0 {
			portVal = 3000
		}
	}
	runtimePortEntry.SetText(fmt.Sprintf("%d", portVal))

	// Runtime 下拉選單
	runtimeLabels := preset.GetRuntimeLabelsForType(proj.Type)
	runtimeTypeSelect := widget.NewSelect(runtimeLabels, nil)
	currentRuntimeLabel := preset.GetRuntimeLabel(proj.RuntimeType)
	if proj.RuntimeType == "" || proj.RuntimeType == "none" {
		currentRuntimeLabel = preset.GetRuntimeLabel(p.DefaultRuntime)
	}
	foundRuntime := false
	for _, label := range runtimeLabels {
		if label == currentRuntimeLabel {
			runtimeTypeSelect.SetSelected(currentRuntimeLabel)
			foundRuntime = true
			break
		}
	}
	if !foundRuntime && len(runtimeLabels) > 0 {
		runtimeTypeSelect.SetSelected(runtimeLabels[0])
	}

	useBundledCheck := widget.NewCheck("Use bundled runtime", nil)
	useBundledCheck.Checked = proj.UseWinCMPBin
	useBundledNote := widget.NewLabelWithStyle("(使用 WinCMP 內建 bin/ 執行檔，未勾選則使用系統環境變數)", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})

	commandEntry := newScrollPassthroughEntry()
	commandEntry.SetText(proj.Command)
	commandEntry.PlaceHolder = "e.g. deno task dev --port %PORT% --host %HOST%"
	commandNote := widget.NewLabelWithStyle("支援佔位符: %PORT% %HOST% %PROJECT_DIR% %BIN_DIR%", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
	commandNote2 := widget.NewLabelWithStyle("注意: 若使用自訂指令，請勿在指令中包含引號，否則可能導致解析錯誤。", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})

	portFormItem := widget.NewFormItem("Port", runtimePortEntry)
	runtimeFormItem := widget.NewFormItem("Runtime", runtimeTypeSelect)
	bundledFormItem := widget.NewFormItem("Use bundled runtime", container.NewVBox(useBundledCheck, useBundledNote))
	commandFormItem := widget.NewFormItem("Start Command", container.NewVBox(commandEntry, commandNote, commandNote2))

	// 使用 dirty flag 追蹤命令是否被手動修改
	commandDirty := proj.CommandDirty

	// 進階設定區域的初始顯示
	isRuntimeType := preset.IsRuntimeProject(proj.Type)
	isCustom := proj.Type == preset.TypeCustom || proj.RuntimeType == preset.RuntimeCustom

	if !isRuntimeType {
		portFormItem.Widget.Hide()
		runtimeFormItem.Widget.Hide()
		bundledFormItem.Widget.Hide()
		commandFormItem.Widget.Hide()
	} else {
		runtimeFormItem.Widget.Show()
		bundledFormItem.Widget.Show()
		portFormItem.Widget.Show()
		if isCustom || commandDirty {
			commandFormItem.Widget.Show()
		} else {
			commandFormItem.Widget.Hide()
		}
	}

	// Python/Go/Custom 不支援 bundled runtime
	updateBundledForcedState := func(rt string) {
		forcedOff := rt == "python" || rt == "go_air" || rt == "go_run" || rt == "custom" || rt == "none"
		if forcedOff {
			useBundledCheck.SetChecked(false)
			useBundledCheck.Disable()
		} else {
			useBundledCheck.Enable()
		}
	}
	currentRT := proj.RuntimeType
	if currentRT == "" || currentRT == "auto" {
		currentRT = p.DefaultRuntime
	}
	updateBundledForcedState(currentRT)

	// Type 變更回調
	typeSelect.OnChanged = func(label string) {
		typeID := preset.GetPresetIDByLabel(label)
		newPreset := preset.GetPreset(typeID)
		isRuntime := newPreset.IsRuntimeProject

		if isRuntime {
			portFormItem.Widget.Show()
			runtimeFormItem.Widget.Show()
			bundledFormItem.Widget.Show()

			// 更新 Runtime 選項
			newRuntimeLabels := preset.GetRuntimeLabelsForType(typeID)
			runtimeTypeSelect.Options = newRuntimeLabels
			defaultRuntimeLabel := preset.GetRuntimeLabel(newPreset.DefaultRuntime)
			runtimeTypeSelect.SetSelected(defaultRuntimeLabel)

			// 更新 Port 預設值（只在用戶未手動修改時）
			if runtimePortEntry.Text == "0" || runtimePortEntry.Text == "" {
				defaultPort := newPreset.DefaultPort
				if defaultPort == 0 {
					defaultPort = 3000
				}
				runtimePortEntry.SetText(fmt.Sprintf("%d", defaultPort))
			}

			// 根據新的 Runtime 類型更新 bundled 狀態
			updateBundledForcedState(newPreset.DefaultRuntime)

			// 非自定義類型且未手動修改命令時，隱藏命令欄位
			if typeID != preset.TypeCustom && !commandDirty {
				commandFormItem.Widget.Hide()
			} else {
				commandFormItem.Widget.Show()
			}
		} else {
			portFormItem.Widget.Hide()
			runtimeFormItem.Widget.Hide()
			bundledFormItem.Widget.Hide()
			commandFormItem.Widget.Hide()
		}
	}

	// Runtime 變更回調
	runtimeTypeSelect.OnChanged = func(label string) {
		rt := preset.GetRuntimeIDByLabel(label)
		isCustomRT := rt == "custom"
		if isCustomRT {
			commandFormItem.Widget.Show()
		} else {
			// 非 Custom 時清空 Start Command，避免殘留指令影響 Runtime 啟動
			commandFormItem.Widget.Hide()
			commandEntry.SetText("")
			commandDirty = false
		}
		updateBundledForcedState(rt)
	}

	commandEntry.OnChanged = func(text string) {
		if text != "" {
			commandDirty = true
		} else {
			commandDirty = false
		}
	}

	// --- 進階設定 ---
	useSSLCheck := widget.NewCheck("Enable SSL", nil)
	useSSLCheck.Checked = proj.UseSSL

	sslCrtEntry := newScrollPassthroughEntry()
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

	sslKeyEntry := newScrollPassthroughEntry()
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
		openLocalPath(rootPathEntry.Text, true)
	})
	openCaddyfileBtn := widget.NewButtonWithIcon("Open Caddyfile", theme.DocumentIcon(), func() {
		openLocalPath(filepath.Join(baseDir, "conf", "sites", proj.Name+".caddy"), false)
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
		container.NewHBox(widget.NewLabel("Project Type: "), container.NewMax(minW, typeSelect)),
		widget.NewForm(portFormItem, runtimeFormItem, bundledFormItem, commandFormItem),
		widget.NewSeparator(),
		advTitle,
		advForm,
	)

	d := dialog.NewCustomConfirm("Edit Project", "Save", "Cancel", container.NewVScroll(content), func(save bool) {
		if save {
			proj.Name = config.SanitizeProjectName(nameEntry.Text)
			rawDomains := domainsEntry.Text
			rawDomains = strings.ReplaceAll(rawDomains, "\n", ",")
			doms := strings.Split(rawDomains, ",")
			var finalDomains []string
			for _, d := range doms {
				trimmed := strings.TrimSpace(d)
				if trimmed != "" {
					// 移除路徑部分 (e.g. example.test/dashboard -> example.test)
					if idx := strings.Index(trimmed, "/"); idx != -1 {
						trimmed = trimmed[:idx]
					}
					// 再次清理以防萬一
					trimmed = strings.TrimSpace(trimmed)
					if trimmed != "" {
						finalDomains = append(finalDomains, trimmed)
					}
				}
			}
			proj.Domains = finalDomains

			if phpSelect.Selected == "None" {
				proj.PHPVersion = ""
			} else {
				proj.PHPVersion = phpSelect.Selected
			}

			proj.RootPath = rootPathEntry.Text
			proj.UseSSL = useSSLCheck.Checked
			proj.SSLCrt = sslCrtEntry.Text
			proj.SSLKey = sslKeyEntry.Text
			proj.Enabled = enabledCheck.Checked

			// 保存 Project Type
			newTypeID := preset.GetPresetIDByLabel(typeSelect.Selected)
			proj.Type = newTypeID

			// 保存 Runtime Type
			if preset.IsRuntimeProject(newTypeID) {
				proj.RuntimeType = preset.GetRuntimeIDByLabel(runtimeTypeSelect.Selected)
				if v, err := strconv.Atoi(runtimePortEntry.Text); err == nil && v > 0 && v <= 65535 {
					proj.RuntimePort = v
				} else {
					proj.RuntimePort = 0
				}
			} else {
				proj.RuntimeType = "none"
				proj.RuntimePort = 0
				proj.RuntimeMode = ""
				proj.RuntimeVersion = ""
				proj.Command = ""
				proj.UseWinCMPBin = false
			}

			proj.UseWinCMPBin = useBundledCheck.Checked
			proj.Command = commandEntry.Text
			proj.CommandDirty = commandDirty

			// 如果不是 custom 且使用者沒手動改過命令，清空命令讓系統自動產生
			if newTypeID != preset.TypeCustom && !commandDirty {
				proj.Command = ""
			}

			// 檢查是否需要停止 Runtime（當 Enabled、Type、RuntimePort 有變更時）
			isRuntimeOrig := preset.IsRuntimeProject(origType)
			needStopRuntime := (origEnabled && !enabledCheck.Checked && isRuntimeOrig) ||
				(isRuntimeOrig && newTypeID != origType) ||
				(isRuntimeOrig && newTypeID == origType && origRuntimePort != proj.RuntimePort)

			if needStopRuntime {
				serviceKey := process.RuntimeServiceKey(proj.Name)
				isRunning := procMgr.IsRunning(serviceKey) || process.CheckRuntimeRunning(origRuntimePort)
				if isRunning {
					runtimeLabel := preset.GetFullTypeLabel(origType, proj.RuntimeType, len(scanRes.BunList) > 0)
					dialog.ShowConfirm("停止 Runtime",
						fmt.Sprintf("「%s」的設定變更會影響 %s 運行。\n是否要自動停止正在運行的 Runtime？", proj.Name, runtimeLabel),
						func(confirm bool) {
							if confirm {
								go func() {
									procMgr.StopRuntime(*proj)
									fyne.Do(func() { onSave() })
								}()
							}
						}, win)
					return
				}
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
	projectRect.SetMinSize(fyne.NewSize(150, 0))
	projectH := container.NewStack(widget.NewLabelWithStyle("Project", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), projectRect)

	availabilityRect := canvas.NewRectangle(color.Transparent)
	availabilityRect.SetMinSize(fyne.NewSize(120, 0))
	availabilityH := container.NewStack(widget.NewLabelWithStyle("Availability", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), availabilityRect)

	typeRect := canvas.NewRectangle(color.Transparent)
	typeRect.SetMinSize(fyne.NewSize(100, 0))
	typeH := container.NewStack(widget.NewLabelWithStyle("Type", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), typeRect)

	header := container.NewBorder(nil, nil,
		container.NewHBox(projectH, availabilityH, typeH),
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
			typeBox := container.NewStack()
			domainsBox := container.NewStack()
			pathBox := container.NewStack()

			leftHBox := container.NewHBox(projectBox, availabilityBox, typeBox)
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
			typeBox := leftHBox.Objects[2].(*fyne.Container)

			domainsBox := centerGrid.Objects[0].(*fyne.Container)
			pathBox := centerGrid.Objects[1].(*fyne.Container)

			// Project Name (Hover 顯示完整名稱)
			projectNameHover := ttwidget.NewLabel(proj.Name)
			projectNameHover.SetToolTip(proj.Name)
			projectNameHover.TextStyle = fyne.TextStyle{Bold: true}
			projectNameHover.Truncation = fyne.TextTruncateEllipsis
			nameRect := canvas.NewRectangle(color.Transparent)
			nameRect.SetMinSize(fyne.NewSize(150, 0))
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

			// Type
			typeText := preset.GetProjectTypeLabel(proj.Type)
			if proj.Type == "" {
				typeText = "Static"
			}
			if proj.Type == "" && proj.RuntimeType != "" && proj.RuntimeType != "none" {
				typeText = preset.GetRuntimeLabel(proj.RuntimeType)
			}

			typeLabel := canvas.NewText(typeText, theme.ForegroundColor())
			typeLabel.TextStyle = fyne.TextStyle{Bold: true}

			typeBoxRect := canvas.NewRectangle(color.Transparent)
			typeBoxRect.SetMinSize(fyne.NewSize(100, 0))
			typeBox.Objects = []fyne.CanvasObject{container.NewStack(typeBoxRect, container.NewHBox(typeLabel))}
			typeBox.Refresh()

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
					regenerateCaddyAndReload()
					appCfg.RefreshConfigExists(baseDir)
					list.Refresh()
					addLog("system", fmt.Sprintf("✅ 已更新專案: %s", proj.Name))
				})
			}
			btns.Objects[1].(*widget.Button).OnTapped = func() {
				dialog.ShowConfirm("刪除專案", fmt.Sprintf("確定要從清單移除 %s 嗎？\n(不會刪除實際檔案)", proj.Name), func(b bool) {
					if b {
						// 使用名稱而非索引定位，避免並行操作時 index 錯位
						for j := range appCfg.Projects {
							if appCfg.Projects[j].Name == proj.Name {
								appCfg.Projects = append(appCfg.Projects[:j], appCfg.Projects[j+1:]...)
								break
							}
						}
						appCfg.Save(filepath.Join(baseDir, "conf", "wincmp.json"))
						regenerateCaddyAndReload()
						appCfg.RefreshConfigExists(baseDir)
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

				detRes := preset.DetectProjectPreset(path)
				projectType := detRes.Type
				phpVersion := ""
				runtimePort := detRes.Port
				runtimeType := detRes.Runtime
				if projectType == preset.TypeLaravel {
					laravelRes := detect.DetectLaravel(path)
					mapping, fallback := loadLaravelPHPMapping()
					phpVersion = getRecommendedPHPVersion(laravelRes.Version, mapping, fallback)
				}

				newProj := config.ProjectConfig{
					Name:        name,
					Domains:     []string{GenerateValidDomain(name)},
					PHPVersion:  phpVersion,
					Type:        projectType,
					RuntimeType: runtimeType,
					RootPath:    path,
					UseSSL:      true,
					Enabled:     true,
					RuntimePort: runtimePort,
				}

				appCfg.Projects = append(appCfg.Projects, newProj)

				appCfg.Save(filepath.Join(baseDir, "conf", "wincmp.json"))
				regenerateCaddyAndReload()
				appCfg.RefreshConfigExists(baseDir)

				// 更新 UI
				list.Refresh()
				addLog("system", fmt.Sprintf("📌 已新增專案: %s，即將自動開啟編輯器", name))

				// 自動開啟編輯器 (稍微延遲以避開 Zenity Blocker 的競爭或干擾)
				time.AfterFunc(300*time.Millisecond, func() {
					fyne.Do(func() {
						// 重新獲取最新的指標，確保安全
						var latestProj *config.ProjectConfig
						for i := range appCfg.Projects {
							if appCfg.Projects[i].Name == name {
								latestProj = &appCfg.Projects[i]
								break
							}
						}

						if latestProj != nil {
							showProjectEditor(win, latestProj, func() {
								appCfg.Save(filepath.Join(baseDir, "conf", "wincmp.json"))
								regenerateCaddyAndReload()
								list.Refresh()
								addLog("system", fmt.Sprintf("✅ 已更新專案: %s", latestProj.Name))
							})
						}
					})
				})

				if projectType == preset.TypeLaravel {
					phpInfo := ""
					if phpVersion != "" {
						phpInfo = fmt.Sprintf(", 建議 PHP: %s", phpVersion)
					}
					addLog("system", fmt.Sprintf("  ↳ 偵測為 Laravel%s (Confidence: %d, Reasons: %s)", phpInfo, detRes.Confidence, strings.Join(detRes.Reasons, ", ")))
				} else if projectType != "" && projectType != preset.TypeStatic {
					addLog("system", fmt.Sprintf("  ↳ 偵測為 %s (Confidence: %d, Reasons: %s)", preset.GetProjectTypeLabel(projectType), detRes.Confidence, strings.Join(detRes.Reasons, ", ")))
				}
			},
			zenity.Title("Select Project Folder"),
		)
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
	scanBtnWrap := container.NewHBox(scanBtn, addBtn)

	topBar := container.NewBorder(nil, nil, nil, scanBtnWrap, title)
	content := container.NewBorder(container.NewVBox(topBar, headerContainer), nil, nil, nil, list)
	return content
}

// ===== 3. 簡易資料庫檢視器 (Database Explorer) =====

// getDBPool 取得或建立 MariaDB 連線池（全域共用，避免頻繁開關連線）
func getDBPool() (*sql.DB, error) {
	port := appCfg.Global.MariaDBPort
	if port <= 0 {
		port = 3306
	}
	user := appCfg.Global.MariaDBUser
	if user == "" {
		user = "root"
	}
	password := appCfg.Global.MariaDBPassword
	cfg := mysql.NewConfig()
	cfg.User = user
	cfg.Passwd = password
	cfg.Net = "tcp"
	cfg.Addr = fmt.Sprintf("127.0.0.1:%d", port)
	cfg.Timeout = 5 * time.Second
	dsn := cfg.FormatDSN()

	dbPoolMu.Lock()
	defer dbPoolMu.Unlock()

	if dbPool != nil && dbPoolDSN == dsn {
		return dbPool, nil
	}

	if dbPool != nil {
		dbPool.Close()
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(3)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(30 * time.Second)
	db.SetConnMaxIdleTime(10 * time.Second)

	dbPool = db
	dbPoolDSN = dsn
	return dbPool, nil
}

// closeDBPool 關閉 MariaDB 連線池
func closeDBPool() {
	dbPoolMu.Lock()
	defer dbPoolMu.Unlock()
	if dbPool != nil {
		dbPool.Close()
		dbPool = nil
		dbPoolDSN = ""
	}
}

func createDatabaseExplorerTab() (fyne.CanvasObject, func()) {
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

	// queryDatabases 查詢所有 Schema（使用連線池）
	queryDatabases := func() ([]string, error) {
		db, err := getDBPool()
		if err != nil {
			return nil, err
		}

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

	// queryTables 查詢指定 Schema 的資料表（使用連線池）
	queryTables := func(schema string) ([]string, error) {
		port := appCfg.Global.MariaDBPort
		if port <= 0 {
			port = 3306
		}
		user := appCfg.Global.MariaDBUser
		if user == "" {
			user = "root"
		}
		password := appCfg.Global.MariaDBPassword
		cfg := mysql.NewConfig()
		cfg.User = user
		cfg.Passwd = password
		cfg.Net = "tcp"
		cfg.Addr = fmt.Sprintf("127.0.0.1:%d", port)
		cfg.DBName = schema
		cfg.Timeout = 5 * time.Second
		dsn := cfg.FormatDSN()

		// queryTables 需要 DBName，與連線池 DSN 不同，因此使用短連線
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
		if appCfg.Global.MariaDBExternal {
			return procMgr.IsRunning(process.MariaDBExternalServiceKey)
		}
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
	var loadingIndicator *widget.ProgressBarInfinite
	var dbTabLock sync.Mutex

	// refreshUIWithLog 共用邏輯，logEnabled 控制是否輸出「MariaDB 未運行」的 log
	refreshUIWithLog := func(logEnabled bool) {
		if !isMariaDBRunning() {
			fyne.Do(func() {
				split.Hide()
				statusLabel.SetText("")
				if notRunningBox != nil {
					notRunningBox.Show()
				}
				loadingIndicator.Hide()
				isMainTabLoading.Store(false)
				if logEnabled {
					addLog("system", "DB Explorer: MariaDB 未運行")
				}
			})
			return
		}

		fyne.Do(func() {
			loadingIndicator.Show()
			statusLabel.SetText("連線中...")
			split.Hide()
			if notRunningBox != nil {
				notRunningBox.Hide()
			}
		})

		go func() {
			databases, err := queryDatabases()
			fyne.Do(func() {
				loadingIndicator.Hide()
				isMainTabLoading.Store(false)
				if err != nil {
					statusLabel.SetText(fmt.Sprintf("連線失敗: %v", err))
					addLog("system", fmt.Sprintf("DB Explorer: 連線失敗 - %v", err))
					notRunningMsg.SetText(fmt.Sprintf("⚠️ 無法連線到 MariaDB\n\n錯誤: %v\n\n請確認 MariaDB 已正常啟動並運行中。", err))
					if notRunningBox != nil {
						notRunningBox.Show()
					}
					return
				}

				split.Show()
				statusLabel.SetText("已連線")
				schemaListData.Set(databases)
				tableHeader.SetText("選擇左側的資料庫以檢視資料表")
				tableListData.Set([]string{})
				addLog("system", fmt.Sprintf("DB Explorer: 已載入 %d 個資料庫", len(databases)))
			})
		}()
	}

	refreshUI := func() {
		refreshUIWithLog(true)
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
		container.NewCenter(dashboardBtn),
	)

	// 使用 Stack 在連線提示和 split 之間切換
	contentStack := container.NewStack(split, container.NewCenter(notRunningBox))

	// 選擇 Schema 時查詢資料表（異步 + Loading）
	schemaList.OnSelected = func(id widget.ListItemID) {
		schemas, _ := schemaListData.Get()
		if id >= len(schemas) {
			return
		}
		selectedSchema := schemas[id]
		tableHeader.SetText(fmt.Sprintf("資料庫 '%s' 的資料表：", selectedSchema))
		tableListData.Set([]string{"載入中..."})

		go func() {
			tables, err := queryTables(selectedSchema)
			fyne.Do(func() {
				if err != nil {
					tableListData.Set([]string{fmt.Sprintf("查詢失敗: %v", err)})
					return
				}
				if len(tables) == 0 {
					tableListData.Set([]string{"（此資料庫沒有資料表）"})
				} else {
					tableListData.Set(tables)
				}
			})
		}()
	}

	// 頂部按鈕列
	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		notRunningMsg.SetText("請先至 Dashboard 頁面啟動 MariaDB 服務，\n再使用 Database Explorer。")
		dbTabLock.Lock()
		defer dbTabLock.Unlock()
		refreshUI()
	})

	heidiSQLBtn := widget.NewButtonWithIcon("Open in HeidiSQL", theme.ComputerIcon(), func() {
		if len(scanRes.HeidiSQLList) == 0 {
			addLog("system", "DB Explorer: 找不到 HeidiSQL 執行檔")
			return
		}
		heidiPath := scanRes.HeidiSQLList[0].ExePath
		port := appCfg.Global.MariaDBPort
		if port <= 0 {
			port = 3306
		}
		if port < 1 || port > 65535 {
			addLog("system", "DB Explorer: 無效的連接埠號")
			return
		}
		user := appCfg.Global.MariaDBUser
		if user == "" {
			user = "root"
		}
		if !validDbUserPattern.MatchString(user) {
			addLog("system", "DB Explorer: 使用者名稱含不合法字元")
			return
		}
		cmd := exec.Command(heidiPath, "-h=127.0.0.1", fmt.Sprintf("-P=%d", port), fmt.Sprintf("-u=%s", user))
		if err := cmd.Start(); err != nil {
			addErrorLog("system", "啟動 HeidiSQL 失敗", err)
			return
		}
		go cmd.Wait()
		addLog("system", fmt.Sprintf("DB Explorer: 已啟動 HeidiSQL (127.0.0.1:%d)", port))
	})

	toolbar := container.NewHBox(refreshBtn, heidiSQLBtn)
	topBar := container.NewBorder(nil, nil, nil, toolbar, title)

	// --- Loading 狀態控制 ---
	loadingIndicator = widget.NewProgressBarInfinite()
	loadingIndicator.Hide()

	safeRefresh := func() {
		dbTabLock.Lock()
		defer dbTabLock.Unlock()
		isMainTabLoading.Store(true)
		refreshUI()
	}

	// 初始化刷新：靜默更新 UI 狀態，不輸出「MariaDB 未運行」的 log（初始化時 DB 很可能還沒啟動）
	go func() {
		dbTabLock.Lock()
		isMainTabLoading.Store(true)
		refreshUIWithLog(false)
		dbTabLock.Unlock()
	}()

	return container.NewBorder(container.NewVBox(topBar, statusLabel, loadingIndicator), nil, nil, nil, contentStack), safeRefresh
}

var (
	themeCache     map[string]fyne.Theme
	themeCacheOnce sync.Once
)

func initThemeCache() {
	themeCache = map[string]fyne.Theme{
		"Light":     theme.LightTheme(),
		"Dark":      theme.DarkTheme(),
		"Dark Blue": createDarkTheme("blue"),
		"Dark Gray": createDarkTheme("gray"),
		"Twilight":  createDarkTheme("twilight"),
		"System":    theme.DefaultTheme(),
	}
}

func getTheme(name string) fyne.Theme {
	themeCacheOnce.Do(initThemeCache)
	if t, ok := themeCache[name]; ok {
		return t
	}
	return themeCache["System"]
}

func applyTheme(themeName string) {
	appCfg.Global.Theme = themeName
	myApp.Settings().SetTheme(getTheme(themeName))
}

// ===== 4. 全域設定 (Settings) =====

func getAppVersion() string {
	type config struct {
		Details struct {
			Version string
		}
	}
	var cfg config
	fyneAppPath := filepath.Join(baseDir, "FyneApp.toml")
	if data, err := os.ReadFile(fyneAppPath); err == nil {
		toml.Decode(string(data), &cfg)
	}
	if cfg.Details.Version == "" {
		return "v0.0.0"
	}
	return "v" + cfg.Details.Version
}

func createSettingsTab(win fyne.Window) fyne.CanvasObject {
	title := widget.NewLabelWithStyle("WinCMP Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	hint := canvas.NewText(" (Changes will be automatically saved immediately with debouncing)", color.NRGBA{R: 128, G: 128, B: 128, A: 255})
	hint.TextSize = 11
	version := canvas.NewText(getAppVersion(), color.NRGBA{R: 128, G: 128, B: 128, A: 255})
	version.TextSize = 11

	header := container.NewHBox(
		title,
		hint,
		layout.NewSpacer(),
		version,
		canvas.NewText("  ", color.NRGBA{R: 0, G: 0, B: 0, A: 0}),
	)

	// --- 1. Basic Settings 組件聲明 ---
	wwwDirEntry := newScrollPassthroughEntry()
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

	sslDirEntry := newScrollPassthroughEntry()
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

	autoUpdateHostsCheck := widget.NewCheck("Auto Update System Hosts File", nil)
	autoUpdateHostsCheck.Checked = appCfg.Global.AutoUpdateHosts

	// --- 3. Log Settings 組件聲明 ---
	maxLogEntry := newScrollPassthroughEntry()
	maxLogEntry.SetText(fmt.Sprintf("%d", appCfg.Global.MaxLogRetention))
	if appCfg.Global.MaxLogRetention == 0 {
		maxLogEntry.SetText("30") // 預設 30 天
	}

	maxLogLinesEntry := newScrollPassthroughEntry()
	maxLogLinesEntry.SetText(fmt.Sprintf("%d", appCfg.Global.MaxLogLines))
	if appCfg.Global.MaxLogLines == 0 {
		maxLogLinesEntry.SetText("500") // 預設 500 行
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
			if isQuitting.Load() {
				return
			}
			mu.Lock()
			appCfg.Global.DefaultWWW = wwwDirEntry.Text
			appCfg.Global.DefaultSSL = sslDirEntry.Text
			appCfg.Global.RestoreLastState = restoreLastStateCheck.Checked
			appCfg.Global.AutoUpdateHosts = autoUpdateHostsCheck.Checked
			appCfg.Global.MinimizeToTray = minToTrayCheck.Checked

			days, errDays := strconv.Atoi(maxLogEntry.Text)
			if errDays != nil || days < 0 {
				days = appCfg.Global.MaxLogRetention
			}
			lines, errLines := strconv.Atoi(maxLogLinesEntry.Text)
			if errLines != nil || lines < 0 {
				lines = appCfg.Global.MaxLogLines
			}
			appCfg.Global.MaxLogRetention = days
			appCfg.Global.MaxLogLines = lines

			cfgPath := filepath.Join(baseDir, "conf", "wincmp.json")
			if err := appCfg.Save(cfgPath); err != nil {
				addErrorLog("system", "自動儲存設定失敗", err)
			} else {
				addLog("system", fmt.Sprintf("⚙️ %s: [%v] ➔ [%v] (Auto Saved)", settingName, oldVal, newVal))
				cleanupOldLogs(days)
			}
			mu.Unlock()
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
	autoUpdateHostsCheck.OnChanged = func(b bool) {
		if b != appCfg.Global.AutoUpdateHosts {
			debouncedSave("Auto Update Hosts", appCfg.Global.AutoUpdateHosts, b)
		}
	}
	maxLogEntry.OnChanged = func(s string) {
		current, err := strconv.Atoi(s)
		if err != nil || current < 0 {
			current = appCfg.Global.MaxLogRetention
		}
		if current != appCfg.Global.MaxLogRetention {
			debouncedSave("Log Retention", appCfg.Global.MaxLogRetention, current)
		}
	}
	maxLogLinesEntry.OnChanged = func(s string) {
		current, err := strconv.Atoi(s)
		if err != nil || current < 0 {
			current = appCfg.Global.MaxLogLines
		}
		if current != appCfg.Global.MaxLogLines {
			debouncedSave("Max Log Lines", appCfg.Global.MaxLogLines, current)
		}
	}
	themeSelect.OnChanged = func(selected string) {
		if selected == appCfg.Global.Theme {
			return
		}
		oldTheme := appCfg.Global.Theme
		if oldTheme == "" {
			oldTheme = "System"
		}

		// 顯示 Loading Overlay，避免主題切換期間卡頓讓用戶誤以為當機
		overlay := showCenterOverlay(win, "正在切換主題...", color.NRGBA{R: 225, G: 225, B: 225, A: 255}, 180)

		// 延遲執行主題切換，讓 Overlay 先渲染出來
		time.AfterFunc(30*time.Millisecond, func() {
			fyne.Do(func() {
				applyTheme(selected)
				hideCenterOverlay(win, overlay)
				debouncedSave("Theme", oldTheme, selected)
			})
		})
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
	)
	sysGrid := container.NewGridWithColumns(2, autoForm, behForm)

	appearTitle := widget.NewLabelWithStyle("Appearance", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	appearForm := widget.NewForm(
		widget.NewFormItem("Theme", themeSelect),
	)

	logTitle := widget.NewLabelWithStyle("Log Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	maxLogLinesHint := canvas.NewText("(僅限制顯示行數，不影響日誌檔保存)", color.NRGBA{R: 128, G: 128, B: 128, A: 255})
	maxLogLinesHint.TextSize = 10
	logForm := widget.NewForm(
		widget.NewFormItem("Retention (Days)", maxLogEntry),
		widget.NewFormItem("Max Log Lines (Terminal)", container.NewVBox(maxLogLinesEntry, maxLogLinesHint)),
	)

	hostsBtn := widget.NewButtonWithIcon("hosts", theme.DocumentIcon(), func() {
		openLocalPath("C:\\Windows\\System32\\drivers\\etc\\hosts", true)
	})
	phpIniBtn := widget.NewButtonWithIcon("php.ini", theme.DocumentIcon(), func() {
		openLocalPath(filepath.Join(baseDir, "conf", "php", "php.ini"), false)
	})
	jsonConfigBtn := widget.NewButtonWithIcon("WinCMP Config", theme.DocumentIcon(), func() {
		openLocalPath(filepath.Join(baseDir, "conf", "wincmp.json"), false)
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

// showMariaDBSettingsDialog 顯示 MariaDB 設定對話框
func showMariaDBSettingsDialog(win fyne.Window) {
	runningModeHint := canvas.NewText("選擇「內建」使用 WinCMP 內建的 MariaDB，選擇「自訂」使用您電腦中現有的資料庫。", color.NRGBA{R: 150, G: 150, B: 150, A: 255})
	runningModeHint.TextSize = 10

	dbType := appCfg.Global.MariaDBType
	if dbType == "" {
		dbType = "MariaDB"
	}
	typeSelect := widget.NewSelect([]string{"MariaDB", "MySQL"}, nil)
	typeSelect.SetSelected(dbType)

	dbEngineHint := canvas.NewText("請選擇與您 Binary Path 中一致的資料庫類型 (MySQL 或 MariaDB)。", color.NRGBA{R: 150, G: 150, B: 150, A: 255})
	dbEngineHint.TextSize = 10

	binaryPathHint := canvas.NewText("請選擇包含 bin 資料夾的根目錄 (例如：...\\mysql-8.4.3-winx64)", color.NRGBA{R: 150, G: 150, B: 150, A: 255})
	binaryPathHint.TextSize = 10
	binaryPathEntry := newScrollPassthroughEntry()
	binaryPathEntry.SetText(appCfg.Global.MariaDBBasedir)
	binaryPathBrowse := widget.NewButtonWithIcon("Browse", theme.FolderOpenIcon(), func() {
		openZenitySelector(win, binaryPathEntry.Text, baseDir, true,
			func(path string) {
				binaryPathEntry.SetText(path)
				if path != "" {
					if _, err := os.Stat(filepath.Join(path, "bin", "mariadbd.exe")); err == nil {
						typeSelect.SetSelected("MariaDB")
					} else if _, err := os.Stat(filepath.Join(path, "bin", "mysqld.exe")); err == nil {
						typeSelect.SetSelected("MySQL")
					}
				}
			},
			zenity.Title("Select MariaDB/MySQL Binary Path"))
	})
	binaryPathFormItem := widget.NewFormItem("Binary Path", container.NewVBox(
		container.NewBorder(nil, nil, nil, binaryPathBrowse, binaryPathEntry),
		binaryPathHint,
	))

	dataPathHint := canvas.NewText("資料庫檔案存放位置 (例如：\\data\\mysql-8.4.3)，建議與程式目錄分開存放以利升級。", color.NRGBA{R: 150, G: 150, B: 150, A: 255})
	dataPathHint.TextSize = 10
	dataPathEntry := newScrollPassthroughEntry()
	dataPathEntry.SetText(appCfg.Global.MariaDBDatadir)
	dataPathBrowse := widget.NewButtonWithIcon("Browse", theme.FolderOpenIcon(), func() {
		openZenitySelector(win, dataPathEntry.Text, baseDir, true,
			func(path string) { dataPathEntry.SetText(path) },
			zenity.Title("Select MariaDB/MySQL Data Path"))
	})
	dataPathFormItem := widget.NewFormItem("Data Path", container.NewVBox(
		container.NewBorder(nil, nil, nil, dataPathBrowse, dataPathEntry),
		dataPathHint,
	))

	portEntry := newScrollPassthroughEntry()
	if appCfg.Global.MariaDBPort > 0 {
		portEntry.SetText(fmt.Sprintf("%d", appCfg.Global.MariaDBPort))
	} else {
		portEntry.SetText("")
	}
	portFormItem := widget.NewFormItem("Port", portEntry)

	runningModeGroup := widget.NewRadioGroup([]string{"內建 (使用預設 MariaDB 實例)", "自訂 (手動指定 Binary Path 與 Data Path)"}, func(selected string) {
		if selected == "自訂 (手動指定 Binary Path 與 Data Path)" {
			typeSelect.Enable()
			dbEngineHint.Show()
			binaryPathFormItem.Widget.Show()
			dataPathFormItem.Widget.Show()
		} else {
			typeSelect.Disable()
			dbEngineHint.Hide()
			binaryPathFormItem.Widget.Hide()
			dataPathFormItem.Widget.Hide()
		}
	})
	if appCfg.Global.MariaDBExternal {
		runningModeGroup.Selected = "自訂 (手動指定 Binary Path 與 Data Path)"
	} else {
		runningModeGroup.Selected = "內建 (使用預設 MariaDB 實例)"
	}

	form := widget.NewForm(
		widget.NewFormItem("Running Mode", container.NewVBox(container.NewHBox(runningModeGroup), runningModeHint)),
		widget.NewFormItem("DB Engine", container.NewVBox(typeSelect, dbEngineHint)),
		binaryPathFormItem,
		dataPathFormItem,
		portFormItem,
	)

	if !appCfg.Global.MariaDBExternal {
		typeSelect.Disable()
		dbEngineHint.Hide()
		binaryPathFormItem.Widget.Hide()
		dataPathFormItem.Widget.Hide()
	}

	d := dialog.NewCustomConfirm("Database Settings", "Save", "Cancel", container.NewVScroll(form), func(save bool) {
		if !save {
			return
		}

		isExt := runningModeGroup.Selected == "自訂 (手動指定 Binary Path 與 Data Path)"
		basedir := binaryPathEntry.Text
		datadir := dataPathEntry.Text
		dbTypeSel := typeSelect.Selected

		portStr := portEntry.Text
		portVal := 0
		if portStr != "" {
			var err error
			portVal, err = strconv.Atoi(portStr)
			if err != nil || portVal < 1 || portVal > 65535 {
				dialog.ShowError(fmt.Errorf("Port 必須是 1-65535 的數字"), win)
				return
			}
		}

		if portVal > 0 {
			blocked := port.CheckPorts([]port.PortInfo{
				{Service: "MariaDB", Port: portVal},
			})
			if len(blocked) > 0 {
				for _, p := range blocked {
					dialog.ShowError(fmt.Errorf("Port %d 已被佔用，無法使用", p.Port), win)
					return
				}
			}
		}

		if isExt {
			if basedir == "" {
				dialog.ShowError(fmt.Errorf("外部模式必須指定 Binary Path"), win)
				return
			}
			if datadir == "" {
				dialog.ShowError(fmt.Errorf("外部模式必須指定 Data Path"), win)
				return
			}
			if _, err := os.Stat(datadir); os.IsNotExist(err) {
				dialog.ShowError(fmt.Errorf("資料目錄不存在: %s\n\n請先初始化資料庫", datadir), win)
				return
			}
		}

		appCfg.Global.MariaDBExternal = isExt
		appCfg.Global.MariaDBType = dbTypeSel
		appCfg.Global.MariaDBBasedir = basedir
		appCfg.Global.MariaDBDatadir = datadir
		appCfg.Global.MariaDBPort = portVal

		cfgPath := filepath.Join(baseDir, "conf", "wincmp.json")
		if err := appCfg.Save(cfgPath); err != nil {
			dialog.ShowError(fmt.Errorf("儲存設定失敗: %v", err), win)
			return
		}

		if portVal > 0 {
			myIniPath := filepath.Join(baseDir, "conf", "my.ini")
			if content, err := os.ReadFile(myIniPath); err == nil {
				lines := strings.Split(string(content), "\n")
				for i, line := range lines {
					if strings.HasPrefix(strings.TrimSpace(line), "port=") {
						lines[i] = fmt.Sprintf("port=%d", portVal)
						break
					}
				}
				os.WriteFile(myIniPath, []byte(strings.Join(lines, "\n")), 0600)
			}
		}

		mode := "內建"
		if isExt {
			mode = "外部 " + dbTypeSel
		}
		portLabel := "3306 (預設)"
		if portVal > 0 {
			portLabel = fmt.Sprintf("%d", portVal)
		}
		addLog("system", fmt.Sprintf("MariaDB 設定已儲存: 模式=%s, Port=%s", mode, portLabel))

		newDashboard := createDashboard(win, func() {})
		mainTabs.Items[0].Content = newDashboard
		mainTabs.Refresh()
	}, win)

	d.Resize(fyne.NewSize(540, 440))
	d.Show()
}

// refreshSystemTray 根據設定更新系統匣選單 (目前維持選單始終存在以確保穩定性)
func refreshSystemTray(myApp fyne.App, myWindow fyne.Window) {
	if desk, ok := myApp.(desktop.App); ok {
		m := fyne.NewMenu("WinCMP",
			fyne.NewMenuItem("顯示 WinCMP", func() {
				myWindow.Show()
			}),
			fyne.NewMenuItem("完全退出 (Quit)", func() {
				saveAndQuit(myApp)
			}),
		)
		desk.SetSystemTrayMenu(m)

		go func() {
			time.Sleep(500 * time.Millisecond)
			systray.SetTooltip("WinCMP")
		}()
	}
}

// triggerHostsUpdate 檢查並更新系統 hosts 檔
func triggerHostsUpdate() {
	if !appCfg.Global.AutoUpdateHosts {
		return
	}

	// 收集所有專案的網域 (不論是否 Enabled，只要加入專案就應更新 Hosts 以便開發)
	var allDomains []string
	for _, proj := range appCfg.Projects {
		allDomains = append(allDomains, proj.Domains...)
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

	// 檢查是否有無效域名（含底線等非法字元）
	var invalidDomains []string
	var validMissing []string
	for _, d := range missing {
		if hosts.IsValidDomain(d) {
			validMissing = append(validMissing, d)
		} else {
			invalidDomains = append(invalidDomains, d)
		}
	}

	// 如果有無效域名，顯示警告
	if len(invalidDomains) > 0 {
		addLog("system", fmt.Sprintf("⚠️ 以下網域含非法字元(含底線)，已跳過: %v", invalidDomains))
	}

	if len(validMissing) == 0 {
		// 沒有有效域名需要更新，直接返回
		addErrorLog("system", "更新系統 Hosts 失敗", fmt.Errorf("所有域名均含非法字元，請手動新增至 hosts: %v", invalidDomains))
		return
	}

	// 2. 備份 Hosts
	backupPath, err := hosts.BackupHosts(baseDir)
	if err != nil {
		addErrorLog("system", "備份 Hosts 失敗 (將停止更新)", err)
		return
	}
	addLog("system", fmt.Sprintf("✅ 已備份現有 Hosts 到: %s", backupPath))

	// 3. 更新 Hosts（只寫入有效域名）
	err = hosts.UpdateHosts(validMissing)
	if err != nil {
		addErrorLog("system", "更新系統 Hosts 失敗", err)
		showHostsWriteFailedDialog(validMissing)
		return
	}

	addLog("system", fmt.Sprintf("🚀 已成功將 %d 個網域寫入系統 Hosts 檔", len(validMissing)))
}

// getMainWindow 獲取當前應用程式的主視窗
func getMainWindow() fyne.Window {
	wins := fyne.CurrentApp().Driver().AllWindows()
	if len(wins) > 0 {
		return wins[0]
	}
	return nil
}

// showHostsWriteFailedDialog 當更新 hosts 失敗時，彈出視窗提示用戶手動加入並提供開啟工具
func showHostsWriteFailedDialog(missingDomains []string) {
	fyne.Do(func() {
		win := getMainWindow()
		if win == nil {
			return
		}

		var sb strings.Builder
		for _, d := range missingDomains {
			sb.WriteString(fmt.Sprintf("127.0.0.1  %s\n", d))
		}
		hostsContent := sb.String()

		desc := widget.NewLabel("由於沒有管理員權限，無法自動更新系統 Hosts 檔案。\n請手動將以下內容新增到您的系統 Hosts 中：")
		
		richText := widget.NewRichText(
			&widget.TextSegment{
				Style: widget.RichTextStyleCodeBlock,
				Text:  hostsContent,
			},
		)

		copyBtn := widget.NewButtonWithIcon("複製內容", theme.ContentCopyIcon(), func() {
			win.Clipboard().SetContent(hostsContent)
			dialog.ShowInformation("成功", "已複製到剪貼簿", win)
		})

		openBtn := widget.NewButtonWithIcon("以管理員權限開啟 Hosts 檔案", theme.DocumentCreateIcon(), func() {
			cmd := exec.Command("powershell", "-Command", `Start-Process notepad.exe -ArgumentList "C:\Windows\System32\drivers\etc\hosts" -Verb runAs`)
			if err := cmd.Run(); err != nil {
				dialog.ShowError(fmt.Errorf("無法開啟 Hosts 檔案: %w", err), win)
			}
		})

		content := container.NewVBox(
			desc,
			container.NewGridWrap(fyne.NewSize(500, 150), container.NewScroll(richText)),
			container.NewHBox(copyBtn, openBtn),
		)

		d := dialog.NewCustom("Hosts 更新失敗", "關閉", content, win)
		d.Show()
	})
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
	rect := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: 100})
	return widget.NewSimpleRenderer(rect)
}

func (b *modalBlocker) Tapped(_ *fyne.PointEvent)          {}
func (b *modalBlocker) TappedSecondary(_ *fyne.PointEvent) {}

// ==== 居中 Loading Overlay ====

type loadingOverlay struct {
	widget.BaseWidget
	bg      *canvas.Rectangle
	label   *canvas.Text
	bgAlpha uint8
}

func newLoadingOverlay(text string, textColor color.Color, bgAlpha uint8) *loadingOverlay {
	o := &loadingOverlay{bgAlpha: bgAlpha}
	o.ExtendBaseWidget(o)

	o.bg = canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: bgAlpha})
	o.label = canvas.NewText(text, textColor)
	o.label.Alignment = fyne.TextAlignCenter
	o.label.TextSize = 15
	o.label.TextStyle = fyne.TextStyle{Bold: true}

	return o
}

func (o *loadingOverlay) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(o.bg, container.NewCenter(o.label)))
}

func (o *loadingOverlay) MinSize() fyne.Size {
	return fyne.NewSize(300, 100)
}

func showCenterOverlay(win fyne.Window, text string, textColor color.Color, bgAlpha uint8) *loadingOverlay {
	overlay := newLoadingOverlay(text, textColor, bgAlpha)
	// Fyne overlay 系統不會自動 Resize widget！必須手動設定為 canvas 大小
	overlay.Resize(win.Canvas().Size())
	win.Canvas().Overlays().Add(overlay)
	return overlay
}

func hideCenterOverlay(win fyne.Window, overlay *loadingOverlay) {
	win.Canvas().Overlays().Remove(overlay)
}
