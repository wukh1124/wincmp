# WinCMP v1.2.0 專案審計報告

> 審計日期：2026-04-12  
> 審計範圍：安全性、效能、BUG 風險  
> 審計版本：v1.2.0 (pre-release)  
> 最後更新：2026-04-12（P3 修復完成）

---

## 一、安全性問題

### 🔴 S-01：Runtime 啟動指令命令注入風險 ✅ 已修復
- **檔案**: `internal/process/runtime.go:234-272`
- **嚴重度**: Critical → ✅ 已修復
- **描述**: `StartRuntime` 將 `project.RootPath`、`project.Command` 及使用者輸入透過 `replacePlaceholders` 直接嵌入 `cmd.exe /c` 命令字串。若 `project.Command`（自訂啟動指令）包含 `& | > <` 等 shell 中繼字元，可在 cmd.exe 執行任意命令。
- **修復內容**: 新增 `sanitizeRuntimeCommand()` 函式，使用 `shellMetacharPattern` 正則偵測 shell 中繼字元（`& | ; < > $ \` " \ ! ( ) { } [ ]`），在 `StartRuntime` 中呼叫驗證。例外允許 `>nul`（Windows 靜默重導）。驗證失敗直接回傳錯誤，拒絕執行。

### 🔴 S-02：域名注入至系統 Hosts 檔案 ✅ 已修復
- **檔案**: `internal/hosts/hosts.go:63-88`
- **嚴重度**: Critical → ✅ 已修復
- **描述**: `UpdateHosts` 將 `domains` 列表直接寫入 Windows hosts 檔案。若域名含換行符號或空格，可注入任意 hosts 規則。
- **修復內容**: 新增 `validDomainPattern` 正則（`^[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?)*$`）和 `sanitizeDomains()` 函式。`UpdateHosts()` 在寫入前過濾不合法域名，全部不合法則回傳錯誤。
- **同步修復**: `main.go` 的 `generateCaddyfiles()` 中域名寫入也加入 `validDomainPattern` 驗證。

### 🔴 S-03：Caddy 設定檔路徑注入（SSL 憑證路徑） ✅ 已修復
- **檔案**: `main.go:1569-1593`（`generateCaddyfiles`）
- **嚴重度**: Critical → ✅ 已修復
- **描述**: `proj.SSLCrt` 和 `proj.SSLKey` 來自使用者輸入的路徑，直接嵌入 Caddy 設定檔的 `tls` 指令，未做路徑清理或驗證。惡意路徑如 `../../etc/passwd` 可能導致路徑遍歷。
- **修復內容**: 新增 `sanitizePath()` 和 `validateCaddyPath()` 函式，使用 `filepath.Clean` 清理路徑後檢查是否包含 `..`。`generateCaddyfiles()` 中 SSL 憑證/金鑰路徑在嵌入前驗證，不合法路徑降級為 `tls internal` 並記錄錯誤 log。

### 🟠 S-04：HeidiSQL 啟動參數注入 ✅ 已修復
- **檔案**: `main.go:3225`
- **嚴重度**: High → ✅ 已修復
- **描述**: `exec.Command(heidiPath, ...)` 傳入使用者可設定的 `port` 和 `user` 變數，若 `MariaDBUser` 被惡意竄改可注入 HeidiSQL 參數。
- **修復內容**: 新增 `validDbUserPattern` 正則（`^[a-zA-Z0-9_\-]+$`）驗證使用者名稱；加入 Port 範圍驗證 (1-65535)。驗證失敗記錄日誌並拒絕啟動 HeidiSQL。

### 🟠 S-05：資料庫密碼明文儲存 ✅ 已修復
- **檔案**: `internal/config/config.go:45`; `internal/crypto/dpapi_windows.go`
- **嚴重度**: High → ✅ 已修復
- **描述**: MariaDB 密碼 `MariaDBPassword` 以明文儲存於 `wincmp.json`，且在 DSN 字串中直接拼接，可能出現在錯誤訊息或日誌中。
- **修復內容**: 新增 `internal/crypto/dpapi_windows.go` 模組，使用 Windows DPAPI (`CryptProtectData`/`CryptUnprotectData`) 加密/解密密碼。加密後密碼以 `ENC:` 前綴 + base64 格式儲存於 JSON，`Load()` 時自動解密，`Save()` 時自動加密（記憶體中保持明文以供 DSN 使用）。向後相容：無 `ENC:` 前綴的密碼視為明文直接回傳。

### 🟠 S-06：日誌與設定檔權限過寬 ✅ 已修復
- **檔案**: `main.go`; `internal/config/config.go`; `internal/hosts/hosts.go`; `internal/process/mariadb.go`
- **嚴重度**: High (跨平台考量) / Low (Windows) → ✅ 已修復
- **描述**: 多處使用 `os.MkdirAll(..., 0755)` 和 `os.WriteFile(..., 0644)`。在 Windows 上影響有限，但跨平台場景下可能使設定檔和日誌被其他使用者讀取。
- **修復內容**: 全域搜尋並替換：`os.MkdirAll(..., 0755)` → `0700`；`os.WriteFile(..., 0644)` → `0600`。涵蓋 `initLogWriters`、`getRuntimeLogWriter`、`generatePHPUpstream`、`generateCaddyfiles`、`config.Save` 備份/寫入、`BackupHosts`、MariaDB 資料目錄建立等所有位置。

### 🟡 S-07：`openLocalPath` 的 ShellExecute 濫用風險 ✅ 已修復
- **檔案**: `main.go:634-656`
- **嚴重度**: Medium → ✅ 已修復
- **描述**: `windows.ShellExecute(0, "open", ...)` 用 absPath 開啟檔案，若路徑被操控可開啟任意程式。
- **修復內容**: 新增兩項安全檢查：(1) 路徑白名單驗證，確保 `absPath` 在 `baseDir` 下（使用 `filepath.Clean` + `strings.HasPrefix`）；(2) 可執行檔封鎖，透過 `blockedExecExts` 集合攔截 `.exe/.bat/.cmd/.ps1/.vbs/.msi/.com/.scr` 等副檔名。驗證失敗記錄 `addErrorLog` 並拒絕執行。

### 🟡 S-08：`cleanupOldLogs` 潛在目錄刪除 ✅ 已修復
- **檔案**: `main.go:208-232`
- **嚴重度**: Medium → ✅ 已修復
- **描述**: `os.RemoveAll(f)` 在掃描 logs 目錄時，若目錄符合 pattern，會整個刪除。若有人建立了如 `runtime-evil` 目錄，會被誤刪。
- **修復內容**: (1) 將 `runtime-*` pattern 改為 `runtime-*.log`，精確匹配日誌檔案而非任意目錄；(2) 移除 `IsDir()` 分支中的 `os.RemoveAll(f)`，改為 `continue` 跳過目錄，確保只刪除符合條件的常規檔案。

### 🟡 S-09：Hosts 檔案寫入缺乏檔案鎖定 ✅ 已修復
- **檔案**: `internal/hosts/hosts.go`
- **嚴重度**: Medium → ✅ 已修復
- **描述**: `UpdateHosts` 以 `O_APPEND|O_WRONLY` 開啟 hosts 檔案但無 file locking，多人同寫可能造成資料損壞。
- **修復內容**: 新增 Windows `LockFileEx`/`UnlockFileEx` API 呼叫（透過 `syscall.NewLazyDLL`），在 `UpdateHosts` 中寫入前取得獨佔鎖定（`LOCKFILE_EXCLUSIVE_LOCK`），寫入完成後解鎖。確保多程序同時寫入時資料完整性。

---

## 二、效能問題

### 🔴 E-01：錯誤日誌每次建立新的 Lumberjack Logger ✅ 已修復
- **檔案**: `main.go:542-577`（`addErrorLog`）
- **嚴重度**: High（效能/資源洩漏）→ ✅ 已修復
- **描述**: 每次 `addErrorLog` 都 new 一個 `lumberjack.Logger` 並 `defer Close()`，導致頻繁的檔案開關與資源浪費。在高頻錯誤場景下嚴重影響效能。
- **修復內容**: 改用 `sync.Map` 快取 error logger，以日期字串（`YYYY-MM-DD`）為 key 做 `LoadOrStore` 單例管理。同一日共用同一個 `lumberjack.Logger` 實例，消除重複建立與 `Close()` 的資源洩漏。新增全域變數 `errorLogCache sync.Map`。

### 🟠 E-02：Log Binding 每次追加皆進行全量字串操作 ✅ 已修復
- **檔案**: `main.go:296-341`（`appendToLogBinding`）
- **嚴重度**: High → ✅ 已修復
- **描述**: 每次呼叫都 `Get()` → 截斷 → `Set()` 全部日誌內容。當日誌量大時，每次追加操作整個字串，時間複雜度 O(n)。多個 goroutine 觸發增加 UI 卡頓風險。
- **修復內容**: 新增 `logRingBuffer` 環形緩衝區結構體，以 `[]string` 切片逐行儲存日誌，取代全量字串的 Get/Set 操作。`append()` 操作僅追加切片並在超限時裁剪，時間複雜度降至 O(1) 分攤。新增全域 `logBuffers` map（`map[binding.String]*logRingBuffer`）管理每個日誌綁定對應的緩衝區。`startLogBufferSync()` 啟動 50ms 間隔的同步 goroutine，週期性地將 dirty 緩衝區透過 `strings.Join` 合併後 `Set()` 到 binding，大幅減少 UI 更新頻率。`stopLogBufferSync()` 於退出時停止同步 goroutine。

### 🟠 E-03：每 4 秒執行 `netstat` 查詢 PID ✅ 已修復
- **檔案**: `internal/process/runtime.go:529-564`
- **嚴重度**: High → ✅ 已修復
- **描述**: 每 4 秒透過 `netstat -ano | findstr :PORT` shell 命令查詢 PID，每個 Runtime 服務各有一個 goroutine。多服務同時運行時產生大量子程序。
- **修復內容**: 改用 `gopsutil/v3/net` 的 `net.Connections("tcp")` API 直接查詢 TCP 連線，避免每次建立 shell 子程序。找不到 LISTEN 狀態連線或 gopsutil 失敗時，降級回原本的 `netstat` 方式（`findPIDByPortFallback`），確保向後相容。新增 `net "github.com/shirou/gopsutil/v3/net"` import。

### 🟠 E-04：DB Explorer 每次查詢都建立新連線 ✅ 已修復
- **檔案**: `main.go:3000-3093`
- **嚴重度**: High → ✅ 已修復
- **描述**: `queryDatabases` 和 `queryTables` 每次呼叫都 `sql.Open` → `defer db.Close()`，未使用連線池。頻繁開關連線造成資料庫連線壓力。
- **修復內容**: 新增全域 `dbPool *sql.DB` 連線池變數（`dbPool`, `dbPoolMu`, `dbPoolDSN`）。新增 `getDBPool()` 函式，以 `dbPoolDSN` 比對判斷 DSN 是否變更，若變更則重建連線池（支援使用者修改 MariaDB 設定後自動重連）。連線池設定：`MaxOpenConns=3`, `MaxIdleConns=1`, `ConnMaxLifetime=30s`, `ConnMaxIdleTime=10s`。`queryDatabases` 改為使用 `getDBPool()` 取得連線池，不再每次 `sql.Open`/`db.Close`。`queryTables` 因需指定 DBName（與連線池 DSN 不同），仍使用短連線但加上註解說明。新增 `closeDBPool()` 函式，在 `saveAndQuit()` 退出時呼叫，確保連線池正確關閉。

### 🟡 E-05：`addLog` 中 UI 分頁切換的 throttle 機制存在鎖競爭 ✅ 已修復
- **檔案**: `main.go`
- **嚴重度**: Medium → ✅ 已修復
- **描述**: `tabSwitchMu.Lock()` 在高頻日誌輸出時會阻塞 goroutine，`tabTimer` 的 `Stop()` 和重建也可能造成延遲累積。
- **修復內容**: 將 `tabSwitchMu`/`lastSwitch`/`tabTimer` 的 mutex+timer 方案全面替換為 channel-based 非同步節流。新增 `tabSwitchReq` 結構體和 `tabSwitchCh`/`tabSwitchDone` channel。`addLog` 中改用 `select` 非阻塞發送切換請求到 channel。新增 `tabSwitchWorker()` goroutine 消費 channel 並以 500ms debounce timer 控制實際切換頻率，徹底消除鎖競爭。

### 🟡 E-06：Resource Monitor 每秒遍歷所有子程序 PID ✅ 已修復
- **檔案**: `internal/resource/monitor.go`
- **嚴重度**: Medium → ✅ 已修復
- **描述**: `fetchResourceData` 每秒呼叫 `process.NewProcess()` 取得每個 PID 的記憶體資訊，子程序數量多時造成 CPU 佔用。
- **修復內容**: 在 `Monitor` 結構體中新增 `cachedPIDs`、`cachedProcs`、`lastPIDUpdate` 快取欄位。`fetchResourceData` 中比對當前 PID 列表與快取，僅在列表變化或超過 5 秒時重新建立 `process.Process` 物件，大幅減少 `NewProcess()` 呼叫次數。

### 🟡 E-07：`createRuntimeTab` 每次刷新都重新掃描 bin 目錄 ✅ 已修復
- **檔案**: `ui_runtime.go`; `internal/scanner/scanner.go`
- **嚴重度**: Medium → ✅ 已修復（與 E-08 合併修復）
- **描述**: 每次進入及刷新 Runtime 分頁都呼叫 `scanner.ScanBinDir` 進行磁碟 I/O 掃描。
- **修復內容**: 見 E-08 修復內容。

### 🟡 E-08：`ScanBinDir` 未快取結果 ✅ 已修復
- **檔案**: `internal/scanner/scanner.go`
- **嚴重度**: Medium → ✅ 已修復
- **描述**: 每次呼叫都完整遍歷 `bin/` 目錄結構，多處同時呼叫可重複掃描。
- **修復內容**: 新增全域快取機制 `scanCache`/`scanCacheTime`/`scanCacheMu`，TTL 為 2 秒。`ScanBinDir()` 先檢查快取是否有效（2 秒內），有效則直接回傳快取結果。無效時呼叫 `scanBinDirInternal()` 執行實際掃描後更新快取。引用新增 `sync` 和 `time` 套件。

---

## 三、程式錯誤風險

### 🔴 B-01：多個全域變數的資料競爭 (Race Condition) ✅ 已修復
- **檔案**: `main.go:371,482`；`ui_runtime.go:499`
- **嚴重度**: Critical → ✅ 已修復
- **描述**: 以下全域變數在多個 goroutine 中無保護地讀寫：
  - `activeRuntimeProject`（第 371, 482 行讀寫）
  - `runtimeLogBindings` map（第 366, 519-523 行讀寫）
  - `appCfg.Projects` 切片（UI 回呼、goroutine）
  - `scanRes`（ui_runtime.go 第 499 行 goroutine 寫入）
- **修復內容**: 新增 `runtimeLogMu sync.RWMutex` 保護 `activeRuntimeProject` 和 `runtimeLogBindings`：
  - `addLog` 中讀取 `runtimeLogBindings` 和 `activeRuntimeProject` 改用 `runtimeLogMu.RLock()`
  - `switchRuntimeLog` 写入 `activeRuntimeProject` 和讀取 `runtimeLogBindings` 改用 `runtimeLogMu.Lock()`
  - `ensureRuntimeLogBinding` 寫入 `runtimeLogBindings` 改用 `runtimeLogMu.Lock()`
  - 日誌按鈕回呼讀取 `activeRuntimeProject` 改用 `runtimeLogMu.RLock()`

### 🔴 B-02：`tabTimer` 競爭條件 ✅ 已確認安全
- **檔案**: `main.go:392-404` 及 `main.go:447-461`
- **嚴重度**: Critical → ✅ 經審查確認已受保護
- **描述**: `tabTimer` 是全域 `*time.Timer`，在 `addLog` 中讀取和重設但未受 `tabSwitchMu` 完整保護。
- **修復內容**: 經重新審查，所有 `tabTimer` 的 Stop/建立操作都在 `tabSwitchMu.Lock()` 鎖區內，`AfterFunc` 回呼內也正確使用 `tabSwitchMu.Lock()`。原審計報告中「鎖區外」的判斷有誤，實際上已在鎖區內。無需額外修改。

### 🟠 B-03：PHP-CGI 單一程序退出未正確處理 ✅ 已修復
- **檔案**: `internal/process/php.go:81-92`
- **嚴重度**: High → ✅ 已修復
- **描述**: 單一 PHP-CGI 程序異常退出時僅記錄 log，不更新 PID 列表或 Running 狀態。若部分程序退出，服務仍標記為 Running 但 PID 列表已過時。
- **修復內容**: 在每個 `cmd.Wait()` 的 goroutine 回呼中：
  1. 使用 `atomic.AddInt32` 計數已退出的程序數量，判斷剩餘運行程序數
  2. 若仍有程序運行：呼叫新增的 `Manager.RemovePID()` 從 PID 列表移除已退出程序的 PID，並記錄剩餘數量的日誌
  3. 若所有程序都已退出：執行 `unregister` 將服務標記為停止
  - 同時新增 `Manager.RemovePID()` 方法（`manager.go`），以 mutex 保護的原子 PID 列表移除操作

### 🟠 B-04：`pipeOutput` / `pipeRuntimeOutput` 忽略 Pipe 錯誤 ✅ 已修復
- **檔案**: `internal/process/manager.go:291`；`internal/process/runtime.go:433`
- **嚴重度**: High → ✅ 已修復
- **描述**: `cmd.StdoutPipe()` 和 `cmd.StderrPipe()` 的錯誤都用 `_` 忽略。若 Pipe 建立失敗，goroutine 收到 nil Reader 只會 return，完全不會有任何錯誤提示。
- **修復內容**: 將 `_` 改為接收錯誤回傳值，若 Pipe 建立失敗則呼叫 `errorLog()` 記錄錯誤，避免 Pipeline 靜默失敗。修改 `pipeOutput()` 和 `pipeRuntimeOutput()` 兩處。

### 🟠 B-05：MySQL DSN 密碼特殊字元問題 ✅ 已修復
- **檔案**: `main.go:2982,3019`
- **嚴重度**: Medium-High → ✅ 已修復
- **描述**: `fmt.Sprintf("%s:%s@tcp(...)", user, password)` 若密碼含 `@`、`:` 或 `/` 等字元，會破壞 DSN 格式導致連線失敗。
- **修復內容**: 改用 `go-sql-driver/mysql` 的 `mysql.NewConfig()` 結構體構建 DSN，透過 `cfg.FormatDSN()` 正確轉義密碼中的特殊字元。新增 `mysql` import alias。`queryDatabases` 和 `queryTables` 兩處皆已修改。

### 🟠 B-06：`Monitor.Stop()` 可能無法停止 goroutine ✅ 已修復
- **檔案**: `internal/resource/monitor.go:238-244`
- **嚴重度**: High → ✅ 已修復
- **描述**: `Stop()` 用非阻塞 `select` 發送停止訊號，若 goroutine 正在 `fyne.Do()` 中阻塞，`default` 分支會丟棄訊號，導致 goroutine 永遠不會結束。
- **修復內容**: 將 `done chan struct{}` 改為 `context.CancelFunc`（`cancel context.CancelFunc`）。`NewAppResourceMonitor()` 中不再建立 channel；`Start()` 方法使用 `context.WithCancel(context.Background())` 建立 context 並儲存 cancel 函數；`Stop()` 方法呼叫 `cancel()` 發送取消訊號，確保 goroutine 一定能收到。import 新增 `context`。

### 🟠 B-07：`dynamicTooltipLabel` Context 洩漏 ✅ 已修復
- **檔案**: `main.go:131-174`
- **嚴重度**: High → ✅ 已修復
- **描述**: 每次 `MouseIn` 事件都建立新的 `context.WithCancel`，若快速反覆 hover，舊 context 不會被 cancel，導致 goroutine 洩漏。
- **修復內容**: 在 `MouseIn` 方法最前面加入 `if l.cancel != nil { l.cancel() }` 先取消舊 context，確保快速反覆 hover 時舊 goroutine 能被回收，避免洩漏。

### 🟡 B-08：`getAppVersion` 使用相對路徑 ✅ 已修復
- **檔案**: `main.go`
- **嚴重度**: Medium → ✅ 已修復
- **描述**: `os.ReadFile("FyneApp.toml")` 使用相對路徑，若工作目錄不在專案根目錄會讀取失敗。
- **修復內容**: 改用 `filepath.Join(baseDir, "FyneApp.toml")` 組合絕對路徑，確保任何工作目錄下都能正確讀取版本資訊。

### 🟡 B-09：`GetSSLKeyPath` 路徑計算錯誤 ✅ 已修復
- **檔案**: `internal/config/config.go:262`
- **嚴重度**: Medium → ✅ 已修復
- **描述**: `GetSSLKeyPath` 第 262 行 `sslDir = filepath.Join(sslDir, baseDir)`，引數順序相反。對比 `GetSSLCertPath`（第 247 行）是正確的 `filepath.Join(baseDir, sslDir)`。這導致當 SSL 目錄為相對路徑時，金鑰路徑計算錯誤。
- **修復內容**: 將 `filepath.Join(sslDir, baseDir)` 改為 `filepath.Join(baseDir, sslDir)`。

### 🟡 B-10：`generateCaddyfiles` 忽略寫入錯誤 ✅ 已修復
- **檔案**: `main.go:1619`
- **嚴重度**: Medium → ✅ 已修復
- **描述**: `os.WriteFile(caddyPath, ...)` 未檢查錯誤，若磁碟空間不足或權限問題，Caddy 設定檔寫入失敗但不會回報。
- **修復內容**: 改為 `if err := os.WriteFile(...); err != nil { return fmt.Errorf(...) }`，回傳錯誤給呼叫方。

### 🟡 B-11：`saveLastServiceState` 非同步安全問題 ✅ 已修復
- **檔案**: `main.go`
- **嚴重度**: Medium → ✅ 已修復
- **描述**: 讀取 `scanRes` 和呼叫 `procMgr.IsRunning` 時未加鎖，若程式退出過程中其他 goroutine 仍在修改狀態，可能讀取到不一致的資料。
- **修復內容**: 新增全域 `saveStateMu sync.Mutex`，`saveLastServiceState()` 在函式開頭加鎖、結尾解鎖，確保退出過程中不會與其他 goroutine 並行存取。

### 🟡 B-12：`isProcessFinished` 字串比對不可靠 ✅ 已修復
- **檔案**: `internal/process/manager.go`
- **嚴重度**: Medium → ✅ 已修復
- **描述**: 以 `err.Error()` 字串比對判斷程序是否已結束（`"os: process already finished"` 等），依賴未匯出的標準庫錯誤訊息。
- **修復內容**: 改用 `errors.Is(err, os.ErrProcessDone)` 作為主要判斷（Go 1.20+），保留字串比對作為回相容回退。新增 `errors` 和 `os` import。

### 🟡 B-13：Settings 儲存 Debounce Timer 競爭 ✅ 已修復
- **檔案**: `main.go`
- **嚴重度**: Medium → ✅ 已修復
- **描述**: `debouncedSave` 中的 `saveTimer` 由閉包外部的 `*time.Timer` 控制，用 `mu sync.Mutex` 保護。若兩個設定項幾乎同時變更，可能的 lock-unlock 順序問題。
- **修復內容**: 在 `time.AfterFunc` 回調內部也加入 `mu.Lock()/Unlock()` 保護，確保設定寫入操作與 timer 重設不會並行。同時將 `fmt.Sscanf` 整數解析改為 `strconv.Atoi` 並驗證範圍（與 B-14 合併修復）。

### 🟡 B-14：`fmt.Sscanf` 整數解析不安全 ✅ 已修復
- **檔案**: `main.go` 多處；`internal/scanner/scanner.go`
- **嚴重度**: Medium → ✅ 已修復
- **描述**: 多處使用 `fmt.Sscanf(val, "%d", &count)` 未檢查回傳值，也不處理溢位。
- **修復內容**: 全面替換為 `strconv.Atoi()` 並檢查錯誤回傳值。對 Port 號欄位加入 1-65535 範圍驗證；對 Settings UI 的數值輸入加入錯誤處理，解析失敗時保留原值。涉及 PHP 進程數排序/選擇、Runtime Port、Log 設定、MariaDB Port、`calcPHPPortBase` 等所有位置。

### 🟡 B-15：MariaDB 自動啟動可能清空資料目錄 ✅ 已修復
- **檔案**: `internal/process/mariadb.go`
- **嚴重度**: Medium → ✅ 已修復
- **描述**: 若偵測到 `mysql/db` 目錄不存在，會 `os.RemoveAll(dataDir)` 清空整個 data 目錄再重建。自動啟動流程沒有確認步驟。
- **修復內容**: 新增 `safeCleanDataDir()` 函式，只刪除 MariaDB 初始化已知的暫存檔和子目錄（`mysql`、`performance_schema`、`test`、`ibdata1`、`ib_logfile*`、`aria_log*`、`auto.cnf` 等），而非 `RemoveAll` 整個目錄。刪除後再 `MkdirAll` 確保目錄存在。

### 🟢 B-16：刪除專案的閉包捕獲索引問題 ✅ 已修復
- **檔案**: `main.go`
- **嚴重度**: Low → ✅ 已修復
- **描述**: 刪除專案回呼使用 `appCfg.Projects[:i]` 和 `appCfg.Projects[i+1:]`，若同時新增專案可能造成 index out of range。
- **修復內容**: 改用專案名稱定位刪除目標，遍歷 `appCfg.Projects` 找到匹配名稱的索引後再執行切片移除，避免與並行操作衝突。

### 🟢 B-17：`openZenitySelector` 中 `fyne.DoAndWait` 可能死鎖 ✅ 已修復
- **檔案**: `main.go`
- **嚴重度**: Low → ✅ 已修復
- **描述**: `fyne.DoAndWait` 在 goroutine 中呼叫，若 Fyne 主迴圈被阻塞（如 modal blocker），此呼叫可能永遠無法完成。
- **修復內容**: 將 `fyne.DoAndWait` 改為 `fyne.Do` 配合 channel 等待回調完成。新增 `resultCh := make(chan struct{})` channel，callback 完成後 close channel。使用 `select` 配合 5 秒超時保護，避免永遠阻塞。

---

## 四、架構/設計建議

| 編號 | 建議 | 優先級 |
|------|------|--------|
| A-01 | 全域變數過多（`sysLog`, `caddyLog`, `procMgr`, `scanRes`, `appCfg` 等）應重構為依賴注入或 Application Context 物件 | 中 |
| A-02 | `main.go` 過長（3796 行），應拆分為 `ui_dashboard.go`, `ui_projects.go`, `ui_settings.go`, `ui_database.go` 等 | 中 |
| A-03 | `addErrorLog` 中重複建立 Lumberjack 實例，應改為全域單例或使用結構化日誌庫 | 低 → ✅ P1 已修復 |
| A-04 | 資料庫連線應抽象為 Repository 層，並使用連線池模式 | 中 → ✅ P2 已修復（`getDBPool()` 連線池） |
| A-05 | SSL 憑證路徑驗證和 Caddy 設定產生應有單元測試保護 | 高 |

---

## 五、優先修復建議

### ✅ P0 已完成
| 編號 | 問題 | 修復狀態 | 修復內容 |
|------|------|----------|----------|
| S-01 | Runtime 命令注入 | ✅ 已修復 | 新增 `sanitizeRuntimeCommand()` 驗證 shell 中繼字元 |
| S-02 | 域名注入 Hosts | ✅ 已修復 | 新增 `validDomainPattern` 正則和 `sanitizeDomains()` 驗證 |
| S-03 | Caddy 路徑注入 | ✅ 已修復 | 新增 `validateCaddyPath()` 路徑遍歷驗證；域名也同步驗證 |
| B-01 | Race Condition | ✅ 已修復 | 新增 `runtimeLogMu sync.RWMutex` 保護併行讀寫 |
| B-02 | tabTimer 競爭 | ✅ 已確認安全 | 經審查所有操作已在 `tabSwitchMu` 鎖區內 |

### ✅ 額外修復（P0 過程中順便完成）
| 編號 | 問題 | 修復狀態 | 修復內容 |
|------|------|----------|----------|
| B-09 | `GetSSLKeyPath` 參數反轉 | ✅ 已修復 | `filepath.Join(sslDir, baseDir)` → `filepath.Join(baseDir, sslDir)` |
| B-10 | `generateCaddyfiles` 寫入忽略錯誤 | ✅ 已修復 | `os.WriteFile` 改為檢查錯誤並回傳 |

### ✅ P1 已完成
| 編號 | 問題 | 修復狀態 | 修復內容 |
|------|------|----------|----------|
| B-04 | Pipe 錯誤被忽略 | ✅ 已修復 | `StdoutPipe()`/`StderrPipe()` 改為接收錯誤並呼叫 `errorLog()` 記錄；消除靜默失敗 |
| B-05 | DSN 密碼特殊字元 | ✅ 已修復 | 改用 `mysql.NewConfig()` + `cfg.FormatDSN()` 構建 DSN，正確轉義密碼特殊字元 |
| E-01 | `addErrorLog` Lumberjack 重複建立 | ✅ 已修復 | 改用 `sync.Map` 按日期快取 `lumberjack.Logger` 實例（`LoadOrStore` 單例），移除 `defer Close()` 避免資源洩漏 |
| B-07 | Tooltip Context 洩漏 | ✅ 已修復 | `MouseIn` 開頭加入 `if l.cancel != nil { l.cancel() }` 先取消舊 context |
| B-03 | PHP-CGI PID 列表過時 | ✅ 已修復 | 使用 `atomic.AddInt32` 計數退出程序；單一退出呼叫 `RemovePID()` 更新列表；全部退出則 `unregister` |
| B-06 | Monitor.Stop goroutine 洩漏 | ✅ 已修復 | `done chan struct{}` → `cancel context.CancelFunc`；`Stop()` 呼叫 `cancel()` 確保 goroutine 收到取消訊號 |

### ✅ P2 已完成
| 編號 | 問題 | 修復狀態 | 修復內容 |
|------|------|----------|----------|
| E-02 | Log Binding 全量字串操作 | ✅ 已修復 | 新增 `logRingBuffer` 環形緩衝區，以 `[]string` 逐行儲存取代全量 `Get()/Set()`；每 50ms 週期同步至 UI binding（`startLogBufferSync`），大幅降低更新頻率 |
| E-03 | netstat shell 命令 | ✅ 已修復 | 改用 `gopsutil/v3/net` 的 `net.Connections("tcp")` API 直接查詢 PID；gopsutil 失敗時降級回 `netstat`（`findPIDByPortFallback`） |
| E-04 | DB 連線池 | ✅ 已修復 | 新增 `getDBPool()` 全域連線池管理（`dbPool *sql.DB`）；DSN 變更時自動重建；`queryDatabases` 改用連線池；退出時 `closeDBPool()` 關閉（MaxOpenConns=3, ConnMaxLifetime=30s） |
| S-04 | HeidiSQL 參數注入 | ✅ 已修復 | 新增 `validDbUserPattern` 正則驗證使用者名稱；加入 Port 範圍驗證 (1-65535)；驗證失敗記錄日誌並拒絕啟動 |
| S-07 | openLocalPath 白名單 | ✅ 已修復 | 新增路徑白名單驗證（`filepath.Clean` + `strings.HasPrefix` 確保在 `baseDir` 內）；新增 `blockedExecExts` 可執行檔副檔名封鎖（`.exe/.bat/.cmd/.ps1/.vbs/.msi/.com/.scr`） |
| S-08 | cleanupOldLogs 目錄刪除 | ✅ 已修復 | `runtime-*` pattern 改為 `runtime-*.log` 精確匹配；移除 `IsDir()` 分支中的 `os.RemoveAll(f)`，改為 `continue` 跳過目錄 |

### ✅ P3 已完成
| 編號 | 問題 | 修復狀態 | 修復內容 |
|------|------|----------|----------|
| S-05 | 資料庫密碼明文儲存 | ✅ 已修復 | 新增 `internal/crypto/dpapi_windows.go`，使用 Windows DPAPI 加解密密碼；`config.Save()` 加密後儲存（`ENC:` 前綴），`config.Load()` 解密還原；向後相容明文密碼 |
| S-06 | 日誌/設定檔權限過寬 | ✅ 已修復 | 全域替換 `os.MkdirAll(..., 0755)` → `0700`、`os.WriteFile(..., 0644)` → `0600`，涵蓋 logs、conf、backup、hosts 備份等所有位置 |
| S-09 | Hosts 檔案缺乏鎖定 | ✅ 已修復 | 新增 Windows `LockFileEx`/`UnlockFileEx` API 呼叫，`UpdateHosts()` 寫入前取得獨佔鎖定，防止並行寫入損壞 |
| E-05 | addLog 分頁 throttle 鎖競爭 | ✅ 已修復 | 改用 channel-based 非同步節流：`tabSwitchCh` channel + `tabSwitchWorker()` goroutine + 500ms debounce timer，取代 mutex+timer 方案 |
| E-06 | Resource Monitor PID 遍歷 | ✅ 已修復 | `Monitor` 新增 `cachedPIDs`/`cachedProcs`/`lastPIDUpdate` 快取欄位；僅在 PID 列表變化或超過 5 秒時重建 `process.Process` 物件 |
| E-07 | createRuntimeTab 重複掃描 | ✅ 已修復 | 與 E-08 合併修復：`ScanBinDir` 新增 2 秒 TTL 快取 |
| E-08 | ScanBinDir 未快取 | ✅ 已修復 | 新增 `scanCache`/`scanCacheTime`/`scanCacheMu` 全域快取；2 秒內重複呼叫直接回傳快取結果，超過 TTL 才重新掃描 |
| B-08 | getAppVersion 相對路徑 | ✅ 已修復 | `os.ReadFile("FyneApp.toml")` → `os.ReadFile(filepath.Join(baseDir, "FyneApp.toml"))` |
| B-11 | saveLastServiceState 非同步安全 | ✅ 已修復 | 新增 `saveStateMu sync.Mutex`，`saveLastServiceState()` 加鎖保護 |
| B-12 | isProcessFinished 字串比對 | ✅ 已修復 | 改用 `errors.Is(err, os.ErrProcessDone)` 為主要判斷，保留字串比對作回相容回退 |
| B-13 | Settings Debounce Timer 競爭 | ✅ 已修復 | `time.AfterFunc` 回調內部加入 `mu.Lock()/Unlock()` 保護；同時修復 B-14（`fmt.Sscanf` → `strconv.Atoi`） |
| B-14 | fmt.Sscanf 整數解析不安全 | ✅ 已修復 | 全部替換為 `strconv.Atoi()` 並檢查錯誤；Port 加入 1-65535 範圍驗證；Settings 數值解析失敗時保留原值 |
| B-15 | MariaDB 自動啟動清空資料 | ✅ 已修復 | 新增 `safeCleanDataDir()` 只刪除已知 MariaDB 暫存檔（mysql、performance_schema、ib_logfile* 等），取代 `RemoveAll(dataDir)` |
| B-16 | 刪除專案閉包索引問題 | ✅ 已修復 | 改用專案名稱遍歷定位刪除目標，避免索引與並行操作衝突 |
| B-17 | openZenitySelector 死鎖 | ✅ 已修復 | `fyne.DoAndWait` → `fyne.Do` + channel 等待 + 5 秒超時保護，避免 Fyne 主迴圈阻塞時永遠等待 |

---

*報告更新完成。P0、P1、P2、P3 項目全部已修復，`go build` 通過。*