package i18n

import (
	"fmt"
	"sync"
)

var (
	currentLang = "zh-TW"
	mu          sync.RWMutex
)

// SetLanguage 設定目前的顯示語言 (如: "zh-TW", "en-US")
func SetLanguage(lang string) {
	mu.Lock()
	defer mu.Unlock()
	currentLang = lang
}

// T 翻譯函式，如果目前語言為英文，則查表替換。支援簡單的字串替換。
func T(text string) string {
	mu.RLock()
	lang := currentLang
	mu.RUnlock()

	if lang == "en-US" {
		if val, ok := enTranslations[text]; ok {
			return val
		}

		// 簡單處理包含變數的常用模式
		// 如果沒有完全匹配，嘗試部分匹配或特定前綴/後綴處理，
		// 但最安全的是直接要求將變數作為 format 參數處理。
		// 這裡先保留原樣，未命中字典則回傳原字串
	}
	return text
}

// Tfmt 提供帶有 fmt.Sprintf 功能的翻譯
func Tfmt(format string, a ...interface{}) string {
	translatedFormat := T(format)
	return fmt.Sprintf(translatedFormat, a...)
}

// enTranslations 存放繁體中文對應至英文的字典
var enTranslations = map[string]string{
	// 系統共用與設定
	"設定":    "Settings",
	"提示":    "Notice",
	"錯誤":    "Error",
	"警告":    "Warning",
	"儲存":    "Save",
	"取消":    "Cancel",
	"確定":    "OK",
	"是":     "Yes",
	"否":     "No",
	"語言":    "Language",
	"請選擇語言": "Select Language",
	"語言設定已變更，需重新啟動 WinCMP 以完全套用。": "Language changed. Please restart WinCMP to fully apply.",
	"語言設定已變更，需重新啟動 WinCMP 以完全套用。是否立即重啟？": "Language changed. Restart WinCMP to apply changes now?",
	"立即重啟": "Restart Now",
	"稍後重啟": "Restart Later",
	"關閉":   "Close",
	"重新啟動": "Restart",
	"顯示語言": "Display Language",

	// 主選單與服務
	"儀表板": "Dashboard",
	"專案":  "Projects",
	"依賴":  "Dependencies",
	"系統":  "System",
	"說明":  "Help",

	// MariaDB Explorer
	"連線中...":   "Connecting...",
	"已連線":      "Connected",
	"連線失敗: %v": "Connection failed: %v",
	"請先至 Dashboard 頁面啟動 MariaDB 服務，\n再使用 Database Explorer。": "Please start MariaDB service on Dashboard first,\nthen use Database Explorer.",
	"⚠️ 無法連線到 MariaDB\n\n錯誤: %v\n\n請確認 MariaDB 已正常啟動並運行中。":   "⚠️ Cannot connect to MariaDB\n\nError: %v\n\nPlease ensure MariaDB is running.",
	"已載入 %d 個資料庫":    "Loaded %d databases",
	"選擇左側的資料庫以檢視資料表": "Select a database on the left to view tables",
	"資料庫 '%s' 的資料表：": "Tables in '%s':",
	"載入中...":         "Loading...",
	"查詢失敗: %v":       "Query failed: %v",
	"（此資料庫沒有資料表）":    "(No tables in this database)",
	"MariaDB 尚未啟動":   "MariaDB is not running",
	"前往 Dashboard":   "Go to Dashboard",

	// Settings
	"自動儲存設定失敗":             "Auto save failed",
	"正在切換主題...":            "Changing theme...",
	"Port 必須是 1-65535 的數字": "Port must be a number between 1-65535",
	"Port %d 已被佔用，無法使用":    "Port %d is already in use",
	"儲存設定失敗: %v":           "Failed to save settings: %v",
	"無法取得執行檔路徑: %v":     "Failed to get executable path: %v",
	"自動重啟失敗":               "Auto restart failed",
	"自動重啟失敗: %v":           "Auto restart failed: %v",

	// ui_runtime.go
	"切換 Terminal Logs 至 Runtime (%s)": "Switch Terminal Logs to Runtime (%s)",
	"複製失敗":                            "Copy failed",
	"無效的 Domain，無法複製連結":               "Invalid Domain, cannot copy link",
	"❌ 複製連結失敗 [%s]: 無效的 Domain":       "❌ Failed to copy link [%s]: Invalid Domain",
	"✅ 已複製連結 [%s]: %s%s":              "✅ Link copied [%s]: %s%s",
	"ℹ️ [%s] 偵測到 %s":                  "ℹ️ [%s] Detected %s",
	"[%s] 沒有可用的 %s 版本，請至 bin/ 檢查":     "[%s] No available version of %s, please check bin/",
	"[%s] 啟動失敗當前端口 %d 不可用":            "[%s] Start failed, current port %d is already in use",
	"啟動失敗":                            "Start failed",
	"當前端口 %d 不可用":                     "Current port %d is already in use",

	// Hosts updates & Admin rights
	"由於沒有管理員權限，無法自動更新系統 Hosts 檔案。\n請手動將以下內容新增到您的系統 Hosts 中：": "Cannot update system Hosts file due to lack of administrator privileges.\nPlease manually add the following content to your system Hosts file:",
	"複製內容":              "Copy Content",
	"成功":                "Success",
	"已複製到剪貼簿":           "Copied to clipboard",
	"以管理員權限開啟 Hosts 檔案": "Open Hosts File as Administrator",
	"Hosts 更新失敗":        "Hosts update failed",
	"更新系統 Hosts 失敗":     "Failed to update system Hosts",
	"🚀 已成功將 %d 個網域寫入系統 Hosts 檔": "Successfully wrote %d domains to system Hosts file",
	"備份 Hosts 失敗 (將停止更新)":       "Failed to backup Hosts (update stopped)",
	"✅ 已備份現有 Hosts 到: %s":       "Backup existing Hosts to: %s",

	// Dashboard & Project actions
	"啟動資料庫中，請稍候...": "Starting database, please wait...",
	"刪除專案":          "Delete Project",
	"確定要從清單移除 %s 嗎？\n(不會刪除實際檔案)":      "Are you sure you want to remove %s from the list?\n(This will not delete the actual files)",
	"尚未設定預設 WWW 目錄，請至 Settings 頁面設定。": "Default WWW directory is not configured. Please configure it in Settings.",
	"確定要自動掃描預設目錄嗎？\n\n路徑：%s\n\n系統將會嘗試將此目錄下的所有「子資料夾」加入為網頁專案清單。\n(已存在的專案將不會重複加入)": "Are you sure you want to scan the default directory?\n\nPath: %s\n\nThe system will try to add all subfolders under this directory as web projects.\n(Existing projects will not be duplicated)",
	"掃描確認":               "Scan Confirmation",
	"🔍 手動掃描預設目錄完成":       "🔍 Manual scan of default directory completed",
	"📌 已自動掃描並加入 %d 個新專案": "📌 Automatically scanned and added %d new projects",

	// MariaDB Initialization
	"MariaDB 初始化確認": "MariaDB Initialization Confirmation",
	"MariaDB 資料庫即將進行初始化（約需 10-30 秒）。\n\n路徑: %s\n\n初始化將會清空上述路徑下的資料，確定要繼續嗎？": "MariaDB database is about to be initialized (takes about 10-30 seconds).\n\nPath: %s\n\nInitialization will clear the data in the above path. Are you sure you want to continue?",

	// Dependecy Manager
	"手動獲取最新依賴":         "Fetch Latest Dependencies",
	"正在從遠端獲取最新依賴資訊...": "Fetching latest dependency info from remote...",
	"獲取成功":             "Fetch Success",
	"最新依賴資訊已成功下載並更新！":  "Latest dependency info has been downloaded and updated!",
	"無法獲取最新依賴資訊，請檢查網路連線或稍後再試。":         "Unable to fetch latest dependency info. Please check your network connection or try again later.",
	"無法獲取最新依賴資訊：\n%v\n\n請檢查網路連線或稍後再試。": "Unable to fetch latest dependency info:\n%v\n\nPlease check your network connection or try again later.",
	"手動獲取時，無法儲存下載 of 依賴建議版本":           "Failed to save downloaded dependency configuration.",
	"依賴設定檔格式不正確，缺少必要欄位":                "Incorrect dependency config format, missing required fields",
	"⚠️ 手動獲取的依賴設定檔格式不正確，缺少必要欄位":        "⚠️ Incorrect dependency config format, missing required fields",
	"無法解析下載的依賴設定檔":                     "Failed to parse downloaded dependency config",
	"伺服器回應錯誤狀態碼: %d":                   "Server returned error status code: %d",
	"⚠️ 手動獲取依賴建議版本伺服器回應錯誤狀態碼: %d":      "⚠️ Server returned error status code during dependency fetch: %d",
	"手動連線伺服器取得最新依賴資訊失敗，請檢查網路連線":        "Failed to connect to server for dependency info, please check your network connection",

	// System Tray & Other
	"顯示 WinCMP":        "Show WinCMP",
	"完全退出 (Quit)":      "Quit WinCMP",
	"正在儲存狀態與關閉所有服務...": "Saving state and stopping all services...",
	"⚠️ 發現 %d 個重複項目名稱 [%s]，Runtime Log 可能出現混淆，建議修改項目名稱以區分": "⚠️ Found %d duplicate project names [%s]. Runtime logs may conflict, rename is recommended",
	"自動啟動上次執行的服務: Caddy":                    "Auto-starting last run service: Caddy",
	"自動啟動上次執行的服務: MariaDB":                  "Auto-starting last run service: MariaDB",
	"自動啟動 MariaDB 失敗":                       "Failed to auto-start MariaDB",
	"自動啟動上次執行的服務: Mailpit":                  "Auto-starting last run service: Mailpit",
	"自動啟動 Mailpit 失敗":                       "Failed to auto-start Mailpit",
	"自動啟動上次執行的服務: PHP-CGI %s":               "Auto-starting last run PHP-CGI %s",
	"WinCMP 資源監控\n\n載入中...":                 "WinCMP Resource Monitor\n\nLoading...",
	"⚠️ 專案 %s: 憑證遺失，將使用自動 TLS":              "⚠️ Project %s: Certificate missing, using auto TLS",
	"⚠️ 專案 %s: 金鑰遺失，將使用自動 TLS":              "⚠️ Project %s: Key missing, using auto TLS",
	"無法掃描 www 目錄":                           "Failed to scan www directory",
	"寫入 Caddy 設定檔 %s 失敗: %w":                "Failed to write Caddy config %s: %w",
	"Reload Caddy 失敗":                       "Reload Caddy failed",
	"✅ Caddy 設定已重新載入":                       "✅ Caddy configuration reloaded",
	"⚠️ 專案 %s 需要 PHP %s，但 PHP %s 尚未啟動！":     "⚠️ Project %s requires PHP %s, but PHP %s is not running!",
	"💡 請在 Dashboard 的 PHP FastCGI 區塊啟動對應版本": "💡 Please start the corresponding version in the PHP FastCGI section on Dashboard",
	"以下專案需要啟動 PHP-CGI 才能正常運作：\n\n":          "The following projects require PHP-CGI to run:\n\n",
	"✅ 已啟動":        "✅ Running",
	"❌ 未啟動":        "❌ Stopped",
	"\n即將啟動：":      "\nStarting soon:",
	"啟動 PHP %s 失敗": "Failed to start PHP %s",
	"✅ PHP %s 已啟動": "✅ PHP %s started",
	"啟動 ":          "Start ",
	" 失敗":          " failed",
	"停止 ":          "Stop ",

	// Startup & Scanner Logs
	"正在初始化 WinCMP...":                "Initializing WinCMP...",
	"專案根目錄: %s":                      "Project root: %s",
	"掃描 ./bin 目錄中的服務版本...":           "Scanning ./bin directory for service versions...",
	"掃描服務版本失敗":                       "Failed to scan service versions",
	"  ✓ 找到 Caddy 版本: [%s]":          "  ✓ Found Caddy version(s): [%s]",
	"  ✗ 未找到 Caddy":                  "  ✗ Caddy not found",
	"  ✓ 找到 Composer 版本: [%s]":       "  ✓ Found Composer version(s): [%s]",
	"  ✗ 未找到 Composer":               "  ✗ Composer not found",
	"  ✓ 找到 HeidiSQL 版本: [%s]":       "  ✓ Found HeidiSQL version(s): [%s]",
	"  ✗ 未找到 HeidiSQL":               "  ✗ HeidiSQL not found",
	"  ✓ 找到 Mailpit 版本: [%s]":        "  ✓ Found Mailpit version(s): [%s]",
	"  ✗ 未找到 Mailpit":                "  ✗ Mailpit not found",
	"  ✓ 找到 MariaDB 版本: [%s]":        "  ✓ Found MariaDB version(s): [%s]",
	"  ✗ 未找到 MariaDB":                "  ✗ MariaDB not found",
	"  ✓ 找到 Node 版本: [%s]":           "  ✓ Found Node version(s): [%s]",
	"  ✗ 未找到 Node":                   "  ✗ Node not found",
	"  ✓ 找到 PHP 版本: [%s]":            "  ✓ Found PHP version(s): [%s]",
	"  ℹ 略過舊 Patch 版本 (僅保留最新): [%s]": "  ℹ Skipped older patch version(s) (retained latest only): [%s]",
	"  ✗ 未找到 PHP":                    "  ✗ PHP not found",
	"無法載入設定檔":                        "Failed to load configuration",
	"  ✓ 設定檔已載入 (%d 個專案)":            "  ✓ Configuration loaded (%d project(s))",
	"無法載入依賴設定檔，將使用預設設定":        "Failed to load dependency configuration, default configuration will be used",
	"  ✓ 依賴設定檔已載入":               "  ✓ Dependency configuration loaded",

	// Settings hints
	"(系統 Hosts)": "(System Hosts)",
	"(PHP 全域設定)": "(PHP global config)",
	"(核心設定)":     "(Core settings)",
	"(僅限制顯示行數，不影響日誌檔保存)": "(Only limits displayed lines, doesn't affect log files)",

	"路徑無效: %s": "Invalid path: %s",
	"路徑不在允許的目錄內: %s": "Path is not within the allowed directory: %s",
	"不允許開啟可執行檔: %s": "Opening executable files is not allowed: %s",
	"開啟失敗: %s": "Failed to open: %s",
	"找不到日誌檔: %s": "Log file not found: %s",
	"啟動 %s 失敗": "Failed to start %s",
	"停止 %s 失敗": "Failed to stop %s",
	"  ↳ 自動遷移專案 [%s] 的框架類型為: %s": "  ↳ Automatically migrated project [%s] framework type to: %s",
	"  ↳ 自動遷移專案 [%s] 的類型為: %s": "  ↳ Automatically migrated project [%s] type to: %s",
	"  ↳ %s: 偵測為 Laravel (Confidence: %d, Reasons: %s)": "  ↳ %s: Detected Laravel (Confidence: %d, Reasons: %s)",
	"  ↳ %s: 偵測為 %s (Runtime: %s, Confidence: %d, Reasons: %s)": "  ↳ %s: Detected %s (Runtime: %s, Confidence: %d, Reasons: %s)",
	"專案 %s: 憑證路徑不安全 (%v)，使用自動 TLS": "Project %s: Certificate path insecure (%v), using auto TLS",
	"專案 %s: 金鑰路徑不安全 (%v)，使用自動 TLS": "Project %s: Key path insecure (%v), using auto TLS",
	"專案 %s: 憑證遺失，使用自動 TLS": "Project %s: Certificate missing, using auto TLS",
	"專案 %s: 金鑰遺失，使用自動 TLS": "Project %s: Key missing, using auto TLS",
	"✓ 自動下載並更新最新依賴建議版本成功 (Auto Download 流程)": "✓ Automatically downloaded and updated latest recommended dependency versions (Auto Download process)",
	"⚠️ 自動下載核心依賴時，無法取得遠端最新配置，將使用本地快取配置": "⚠️ Unable to get remote latest configuration when auto downloading core dependencies, local cached configuration will be used",
	"✓ 重新載入依賴設定檔成功，Caddy: %s, HeidiSQL: %s, Node: %s": "✓ Successfully reloaded dependency configuration, Caddy: %s, HeidiSQL: %s, Node: %s",
	"重新載入依賴設定檔失敗，將使用記憶體中現有配置": "Failed to reload dependency configuration, current configuration in memory will be used",
	"✓ 手動下載並更新最新依賴建議版本成功": "✓ Successfully downloaded and updated latest recommended dependency versions manually",
	"手動獲取時，無法儲存下載的依賴建議版本": "Failed to save downloaded dependency configuration during manual fetch",
	"✓ 背景下載並更新最新依賴建議版本成功": "✓ Successfully downloaded and updated latest recommended dependency versions in background",
	"無法儲存背景下載的依賴建議版本": "Failed to save background downloaded dependency configuration",
	"⚠️ 背景下載的依賴設定檔格式不正確，缺少必要欄位": "⚠️ Incorrect background downloaded dependency configuration format, missing required fields",
	"無法解析背景下載的依賴設定檔": "Failed to parse background downloaded dependency configuration",
	"⚠️ 背景下載依賴建議版本伺服器回應錯誤狀態碼: %d": "⚠️ Server returned error status code during background dependency fetch: %d",
	"背景連線伺服器取得最新依賴資訊失敗，將使用本地快取配置": "Failed to connect to server in background, local cached configuration will be used",
	"✓ 依賴管理器面板已自動刷新最新建議版本": "✓ Dependency manager panel automatically refreshed with latest recommended versions",
	"已複製 Mailpit 網址: %s": "Mailpit URL copied: %s",
	"Mailpit 設定已儲存: SMTP=%d, HTTP=%d, 存儲=%s": "Mailpit settings saved: SMTP=%d, HTTP=%d, Storage=%s",
	"PHP %s 進程數變更為 %d (重啟後生效)": "PHP %s process count changed to %d (takes effect after restart)",
	"PHP 進程配置已更變，請重載 (Reload) 或重啟 Caddy 以套用新端口": "PHP process configuration changed. Please reload or restart Caddy to apply new ports",
	"📋 已複製網址到剪貼簿: %s": "📋 URL copied to clipboard: %s",
	"✅ 已更新專案: %s": "✅ Project updated: %s",
	"✅ 已移除專案: %s": "✅ Project removed: %s",
	"📌 已新增專案: %s，即將自動開啟編輯器": "📌 Project added: %s, opening editor...",
	"  ↳ 偵測為 Laravel%s (Confidence: %d, Reasons: %s)": "  ↳ Detected Laravel%s (Confidence: %d, Reasons: %s)",
	"  ↳ 偵測為 %s (Confidence: %d, Reasons: %s)": "  ↳ Detected as %s (Confidence: %d, Reasons: %s)",
	"DB Explorer: 連線失敗 - %v": "DB Explorer: Connection failed - %v",
	"DB Explorer: 已啟動 HeidiSQL (127.0.0.1:%d)": "DB Explorer: HeidiSQL started (127.0.0.1:%d)",
	"⚙️ %s: [%v] ➔ [%v] (Auto Saved)": "⚙️ %s: [%v] ➔ [%v] (Auto Saved)",
	"🔍 偵測到 %d 個網域不在系統 Hosts 中: %s": "🔍 Detected %d domains not in system Hosts: %s",
	"⚠️ 以下網域含非法字元(含底線)，已跳過: %v": "⚠️ The following domains contain invalid characters (including underscores) and have been skipped: %v",
	"所有域名均含非法字元，請手動新增至 hosts: %v": "All domains contain invalid characters, please add to hosts manually: %v",
	"無法儲存服務狀態至設定檔": "Failed to save service state to configuration file",
	"DB Explorer: MariaDB 未運行": "DB Explorer: MariaDB is not running",
	"DB Explorer: 找不到 HeidiSQL 執行檔": "DB Explorer: HeidiSQL executable not found",
	"DB Explorer: 無效的連接埠號": "DB Explorer: Invalid port number",
	"DB Explorer: 使用者名稱含不合法字元": "DB Explorer: Username contains invalid characters",
	"檢查 Hosts 失敗": "Failed to check Hosts",

	// Database Settings
	"Database Settings": "Database Settings",
	"選擇「內建」使用 WinCMP 內建的 MariaDB，選擇「自訂」使用您電腦中現有的資料庫。": "Select 'Built-in' to use the default MariaDB instance, or 'Custom' to manually specify local paths.",
	"請選擇與您 Binary Path 中一致的資料庫類型 (MySQL 或 MariaDB)。": "Please select the database type (MySQL or MariaDB) that matches your Binary Path.",
	"請選擇包含 bin 資料夾的根目錄 (例如：...\\mysql-8.4.3-winx64)": "Please select the root directory containing the 'bin' folder (e.g., ...\\mysql-8.4.3-winx64)",
	"Select MariaDB/MySQL Binary Path": "Select MariaDB/MySQL Binary Path",
	"資料庫檔案存放位置 (例如：\\data\\mysql-8.4.3)，建議與程式目錄分開存放以利升級。": "Database data storage location (e.g., \\data\\mysql-8.4.3). Keeping it separate from the app directory is recommended for easier updates.",
	"Select MariaDB/MySQL Data Path": "Select MariaDB/MySQL Data Path",
	"內建 (使用預設 MariaDB 實例)": "Built-in (use default MariaDB instance)",
	"自訂 (手動指定 Binary Path 與 Data Path)": "Custom (manually specify Binary Path and Data Path)",
	"外部模式必須指定 Binary Path": "Binary Path must be specified in Custom mode",
	"外部模式必須指定 Data Path": "Data Path must be specified in Custom mode",
	"資料目錄不存在: %s\n\n請先初始化資料庫": "Data directory does not exist: %s\n\nPlease initialize the database first",
	"內建": "Built-in",
	"外部 ": "Custom ",
	"3306 (預設)": "3306 (default)",
	"MariaDB 設定已儲存: 模式=%s, Port=%s": "MariaDB settings saved: Mode=%s, Port=%s",
	"瀏覽": "Browse",

	// Mailpit Settings
	"Mailpit Settings": "Mailpit Settings",
	"持久化存儲 (Database)": "Persistent Storage (Database)",
	"複製網址": "Copy URL",
	"SMTP 端口，用於接收郵件（預設 1025）": "SMTP Port, used for receiving emails (default 1025)",
	"網頁管理介面端口（預設 8025）": "Web UI port (default 8025)",
	"啟用後郵件將保存至 data/mailpit 目錄，重啟後不遺失": "When enabled, emails will be saved to the data/mailpit directory and will not be lost after restart",
	"SMTP Port 必須是 1-65535 的數字": "SMTP Port must be a number between 1-65535",
	"HTTP Port 必須是 1-65535 的數字": "HTTP Port must be a number between 1-65535",
	"SMTP Port 和 HTTP Port 不能相同": "SMTP Port and HTTP Port cannot be the same",
	"Port %d 已被佔用": "Port %d is already in use",
	"記憶體": "Memory",
	"持久化": "Database",

	// Resource Monitor
	"WinCMP 資源監控": "WinCMP Resource Monitor",
	"主程式 RAM:   %s": "Main App RAM:   %s",
	"主程式 CPU:   %s": "Main App CPU:   %s",
	"目前沒有啟動中的子服務": "No running child services currently",
	"── Stack Total 明細 ──": "── Stack Total Breakdown ──",

	// Project Edit / Run Config
	"停止 Runtime": "Stop Runtime",
	"「%s」的設定變更會影響 %s 運行。\n是否要自動停止正在運行的 Runtime？": "Changes to '%s' configuration will affect the running %s.\nWould you like to automatically stop the running Runtime?",
	"取得最新版本": "Fetch Latest Version",
}
