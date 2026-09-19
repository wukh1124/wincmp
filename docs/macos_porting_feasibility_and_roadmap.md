# WinCMP 跨平台支援 (macOS) 評估與實作計畫可行性報告

本文件詳細評估將 **WinCMP**（目前僅支援 Windows）擴展至 **macOS**（進而達成全平台/跨平台支援）之架構衝擊、各底層模組改動方案、實作難度與分階段路線圖。

---

## 1. 核心結論與總體可行性摘要

- **可行性評估**：**高度可行 (Highly Feasible)**。
- **架構優勢**：WinCMP 的 GUI 層採用 **Wails v2 (Go + WebKit/WebView2 + React)**，其架構原生即為跨平台設計；主要的平台綁定集中在 `internal/process`、`internal/terminal`、`internal/hosts`、`internal/singleinstance` 與 `internal/crypto` 等模組。
- **最大挑戰**：
  1. **PHP 二進位散佈 (Distribution)**：PHP 官方在 Unix/macOS 下**僅釋出源始碼，無官方預編譯 binary**。需透過 Homebrew 整合或採用 `static-php-cli` (spc) 進行預先靜態編譯封裝。
  2. **行程生命週期 (Process Lifecycle)**：Windows 依賴核心層級的 `Job Object` 來確保主程式崩潰時子行程必定自毀；macOS/POSIX 必須改採 `Process Group (PGID)` 與自定義退場保險機制。
  3. **macOS 權限與檔案路徑規範**：macOS 的 `.app` Bundle 為唯讀受簽名保護目錄，不能如 Windows 綠色版直接在同級目錄寫入 `data/`、`logs/` 與動態下載二進位檔，需重構路徑系統對接 `~/Library/Application Support/`。

---

## 2. 各底層模組架構差異與改動方案

### 2.1 行程管理與生命週期 (Process Management & Job Object)
- **難度等級**：★★★☆☆ (中等)
- **Windows 現狀**：
  - 使用 `internal/process/job.go` 封裝 Windows Job Object (`CreateJobObject`, `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`)。
  - 使用 `CreationFlags: 0x08000000` (`CREATE_NO_WINDOW`) 防止背景行程閃現命令提示字元黑窗。
  - 自動重啟使用 `0x01000000` (`CREATE_BREAKAWAY_FROM_JOB`)。
- **macOS 改動方案**：
  - **進程組 (Process Group)**：改用 POSIX 的 `Setpgid: true`：
    ```go
    cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
    ```
  - **殺死進程樹**：終止服務時，向進程組負數 PID 發送信號，確保所有派生孫進程同時終止：
    ```go
    syscall.Kill(-pgid, syscall.SIGTERM) // 優先優雅終止
    // 超時後 fallback 至 SIGKILL
    syscall.Kill(-pgid, syscall.SIGKILL)
    ```
  - **視窗標誌隔離**：macOS 無需 `CREATE_NO_WINDOW`（背景 `exec.Cmd` 預設就不會跳出終端視窗）。透過 Go Build Tags 拆分為 `manager_windows.go` 與 `manager_darwin.go`。

---

### 2.2 PHP FastCGI (多行程負載平衡)
- **難度等級**：★★★★☆ (需策略選擇)
- **Windows 現狀**：
  - 直接下載解壓 `windows.php.net` 官方打包的 x64 NTS 綠色版。
  - 啟動多個 `php-cgi.exe -b 127.0.0.1:38200` 等連續 Port，由 Caddy 做反向代理輪詢。
  - 擴充套件為 `.dll`，PATH 分隔符號為分號 `;`。
- **macOS 改動方案**：
  - **二進位提供策略 (雙軌制)**：
    - *軌道 A (強烈推薦，最佳開箱即用)*：採用 `static-php-cli` (spc) 為 Apple Silicon (arm64) 與 Intel (x86_64) 預編譯乾淨的獨立單一執行檔 `php-cgi`（內建 curl, openssl, mbstring, pdo_mysql, opcache 等常用擴充），放入發布版本或下載管理器。
    - *軌道 B (本機環境整合)*：自動偵測 Homebrew 安裝的 PHP（如 `/opt/homebrew/bin/php-cgi` 或 `/opt/homebrew/opt/php@8.2/bin/php-cgi`）。
  - **執行模式**：維持原有的 `php-cgi -b 127.0.0.1:port` 機制，與現有 `internal/scanner` 的 Port 計算公式與 Caddyfile 逆向代理機制 **100% 相容**，免去重寫 php-fpm pool 配置的負擔。
  - **擴充套件格式**：若有動態載入需求，副檔名需為 `.so`；PATH 分隔符改為冒號 `:`。

---

### 2.3 Web 伺服器 (Caddy)
- **難度等級**：★☆☆☆☆ (極低)
- **Windows 現狀**：
  - 執行 `caddy.exe run --config conf/Caddyfile --adapter caddyfile --watch`。
  - Windows 專屬問題：timberjack 在 log rotation 壓縮為 `.gz` 時，因 Windows 檔案鎖導致刪除舊原始檔失敗，特別設計了 `cleanupStaleRotatedLogs()`。
- **macOS 改動方案**：
  - Caddy 官方原生提供 macOS arm64 (`caddy_..._mac_arm64.tar.gz`) 與 amd64 二進位檔，解壓即可直接執行。
  - macOS (Unix 核心) 天生支援刪除開啟中的檔案 (Unlink open files)，完全不會發生 Windows 的檔案鎖崩潰問題，`cleanupStaleRotatedLogs()` 在 macOS 下可直接作為 no-op。
  - **注意 80 / 443 埠號**：macOS 預設非 root 使用者無法直接綁定低於 1024 的埠。解決方案：
    - 方案 1：本機開發時使用 Caddy 自訂高位埠 (如 8080/8443)，或使用 `internal/hosts` 配合反向代理。
    - 方案 2：初次執行時透過 `sudo` / macOS 授權對 caddy 二進位賦予權限，或使用 `pfctl` 進行本機轉發。

---

### 2.4 資料庫 (MariaDB / MySQL) 與 GUI 工具
- **難度等級**：★★★☆☆ (中等)
- **Windows 現狀**：
  - 下載 `mariadb-11.4.10-winx64.zip`，以 `mariadb-install-db.exe` 初始化 `data/mariadb`，執行 `mariadbd.exe`。
  - 捆綁 `heidisql.exe`（純 Windows Delphi 開發工具）。
- **macOS 改動方案**：
  - **MariaDB 二進位**：
    - MariaDB 官方有 macOS Tarball，亦可透過 Homebrew (`brew install mariadb`)。
    - 初始化腳本在 Unix 下為 `mariadb-install-db`，配置檔名慣例為 `my.cnf` 取代 `my.ini`。
  - **資料庫管理工具 (替換 HeidiSQL)**：
    - HeidiSQL 無法在 macOS 運行。
    - *首選方案*：全面推廣 WinCMP 前端既有的 **DBExplorer** (內建 Web 資料庫檢視器，全平台完全一致)。
    - *外部 GUI 方案*：偵測或引導啟動 macOS 上最受歡迎的開源免費客戶端：
      - **Sequel Ace** (`open -a "Sequel Ace"`)
      - **TablePlus** (`open -a "TablePlus"`)
      - 或通用的 URL Scheme 喚起：`open "mysql://root:password@127.0.0.1:3306"`。

---

### 2.5 快取服務 (Redis)
- **難度等級**：★☆☆☆☆ (極低)
- **Windows 現狀**：
  - 依賴停更多年的 `tporadowski/redis` 5.0.14.1 Windows 移植版。
  - 依賴 `redis.windows.conf`。
- **macOS 改動方案**：
  - **原生大幅升級**：Redis 原生為 Unix/Linux 打造，在 macOS 上可以直接運行最新的 Redis 7.x 甚至 8.x。
  - 二進位檔名為標準 `redis-server`，設定檔為標準 `redis.conf`。

---

### 2.6 專案執行環境 (Node.js, Bun, Python, Go)
- **難度等級**：★★☆☆☆ (低至中等)
- **Windows 現狀**：
  - `runtime.go` 內部硬編碼 `cmd.exe /c start "title" cmd.exe /k ...` 與 `cmd.exe /c chcp 65001 >nul && ...`。
  - 透過 `npm.cmd`、`bun.exe` 執行。
- **macOS 改動方案**：
  - Node.js / Bun：官方皆有 macOS arm64/x86_64 二進位檔 (`node`, `npm`, `npx`, `bun`)。
  - 指令包裝改用標準 POSIX Shell：
    ```go
    exec.Command("/bin/zsh", "-c", runtimeCmd)
    ```
    完全移除 Windows 特有的 `chcp 65001` 與 `cmd.exe`。
  - Python：macOS 預設指令為 `python3`，需相容此命名。

---

### 2.7 內建終端機 (ConPTY vs Unix PTY)
- **難度等級**：★★☆☆☆ (低至中等)
- **Windows 現狀**：
  - `internal/terminal/terminal.go` 使用 `//go:build windows` 並引用 `github.com/UserExistsError/conpty`。
  - Shell 解析強制為 `powershell.exe` / `cmd.exe`。
- **macOS 改動方案**：
  - 保留 Windows 版實作，新增 `terminal_darwin.go`。
  - 引入成熟的 Go Unix PTY 庫：`github.com/creack/pty`。
    ```go
    ptmx, err := pty.Start(cmd)
    ```
  - Shell 解析：讀取系統環境變數 `$SHELL`（macOS 預設為 `/bin/zsh`，回退至 `/bin/bash`）。
  - 對前端 xterm.js 的事件流（`terminal_output`, `terminal_exit`, `SendTerminalInput`, `ResizeTerminal`）完全相容，無縫接軌。

---

### 2.8 資源監控 (Resource Monitor - CPU & RAM)
- **難度等級**：★☆☆☆☆ (極低)
- **Windows 現狀**：
  - `internal/resource/monitor.go` 使用 `github.com/shirou/gopsutil/v3`。
  - 監控主程式 PID、子服務 PID 以及 WebView2 渲染進程。
- **macOS 改動方案**：
  - `gopsutil` 對 Darwin (macOS) 的 CPU 與 RAM (RSS) 原生完整支援。
  - 差異點：macOS 下 Wails 使用原生 WebKit (WKWebView)，其輔助進程由系統 `launchd` 託管；若需精確計算前端視窗記憶體，可調整過濾邏輯或由前端自行回報。

---

### 2.9 單一實例 (Single Instance)
- **難度等級**：★☆☆☆☆ (極低)
- **Windows 現狀**：
  - `singleinstance.go` 使用 Win32 API `CreateMutexW` 與 `FindWindowW`。
- **macOS 改動方案**：
  - macOS 的 `.app` Bundle 透過 Finder 或 Dock 啟動時，系統 LaunchServices 本身預設即為單實例模式。
  - 在 CLI 或獨立二進位層面，改用標準 Unix 檔案鎖 (`flock`) 或 Unix Domain Socket (`/tmp/wincmp_activation.sock`) 來防止重複執行並實現前台視窗喚醒。

---

### 2.10 資料加密保護 (Crypto / DPAPI)
- **難度等級**：★★☆☆☆ (低)
- **Windows 現狀**：
  - `dpapi_windows.go` 呼叫 Windows `crypt32.dll` 的 `CryptProtectData`。
- **macOS 改動方案**：
  - 抽象化介面為 `crypto_windows.go` 與 `crypto_darwin.go`。
  - macOS 實作方案：
    - *方案 A (標準)*：呼叫 macOS 原生 Keychain (`/usr/bin/security` 或 Go keychain 庫)。
    - *方案 B (輕量通用)*：以主機唯一 Machine-UUID + 本機 Salt 生成密鑰，採用 AES-256-GCM 進行加解密，維持原有 `ENC:...` 格式完全相容。

---

### 2.11 系統 Hosts 檔案與管理員權限
- **難度等級**：★★☆☆☆ (低)
- **Windows 現狀**：
  - 路徑硬編碼為 `C:\Windows\System32\drivers\etc\hosts`。
  - 呼叫 `shell32.dll` 的 `IsUserAnAdmin()` 檢查權限。
- **macOS 改動方案**：
  - 路徑改為 `/etc/hosts`。
  - 權限檢查改為 `os.Geteuid() == 0`。
  - **優雅提權機制**：macOS 應用程式平時以一般使用者執行，當需要修改 `/etc/hosts` 時，使用 AppleScript 觸發原生系統授權認證窗：
    ```go
    script := fmt.Sprintf("do shell script \"echo '%s' >> /etc/hosts\" with administrator privileges", line)
    cmd := exec.Command("osascript", "-e", script)
    ```
    使用者體驗平滑，免去強制要求以 sudo 啟動主程式的危險操作。

---

### 2.12 檔案系統與目錄架構規範
- **難度等級**：★★★☆☆ (中等)
- **關鍵問題**：
  - Windows 上可直接以綠色軟體 (Portable) 形式將 `bin/`, `conf/`, `data/`, `logs/` 置於同一目錄。
  - macOS 上所有 GUI 軟體均打包為 `/Applications/xxx.app`，該目錄受 Gatekeeper 程式碼簽名保護，**強制唯讀**，嚴禁在內部寫入資料。
- **目錄重構配置**：
  | 模組用途 | Windows 現狀 | macOS 規範路徑 (符合 XDG / Apple Guideline) |
  |---|---|---|
  | **主設定檔** | `conf/wincmp.json` | `~/Library/Application Support/wincmp/conf/wincmp.json` |
  | **資料庫儲存** | `data/mariadb/` | `~/Library/Application Support/wincmp/data/mariadb/` |
  | **日誌儲存** | `logs/` | `~/Library/Logs/wincmp/` 或 `~/Library/Application Support/wincmp/logs/` |
  | **內建二進位** | `bin/` | `~/Library/Application Support/wincmp/bin/` 或 App 內置資源 |
  | **預設網站根目錄** | `www/` | `~/Sites/wincmp/` 或 `~/Documents/wincmp/www/` |

---

## 3. 前端介面 (React / TypeScript) 改動清單

雖然前端 90% 以上為平台無關的純 Web 介面，但以下細節需進行環境感知 (Environment Adaptive) 調整：

1. **資料庫連線按鈕**：
   - 移除或將「Open in HeidiSQL」改為「在資料庫工具中開啟 (Open in DB Tool)」。
   - 在 macOS 下彈出支援 Sequel Ace、TablePlus 等說明，或引導使用者使用原生內建的 DBExplorer。
2. **終端機設定 (`Settings.tsx`)**：
   - 預設 Shell 下拉選單：
     - Windows: `PowerShell (powershell.exe)`, `Command Prompt (cmd.exe)`, `Git Bash`
     - macOS: `Zsh (/bin/zsh)`, `Bash (/bin/bash)`, `Fish (/opt/homebrew/bin/fish)`
3. **路徑選擇器與預設路徑展示**：
   - 專案根目錄範例路徑依據平台動態顯示（Windows 為 `C:\...`，macOS 為 `/Users/...`）。
4. **系統快捷鍵提示**：
   - 依據 `navigator.platform`，將 `Ctrl + ...` 替換為 `Cmd (⌘) + ...`。

---

## 4. 實作難度矩陣與分階段路線圖

```text
難度矩陣圖：
[低難度 ★]   Caddy, Redis, 資源監控, 單一實例 (flock)
[中難度 ★★]  終端機 PTY (creack/pty), Hosts 提權 (osascript), DPAPI 替代 (AES/Keychain)
[高難度 ★★★] 行程組管理 (Process Group), 目錄沙盒重構 (~/Library/Application Support)
[特難度 ★★★★] PHP / MariaDB 跨平台二進位打包與 Homebrew 聯動
```

### 建議的 4 階段實作里程碑：

- **階段一：架構解耦與條件編譯 (Refactor & Build Tags)**
  - 將 Windows 專屬 API 拆解至 `*_windows.go`。
  - 建立對應的 `*_darwin.go` 介面 Stub。
  - 完成目錄路徑抽象層（支援 Windows 可攜模式與 macOS Application Support 模式）。
  - 達成 `GOOS=darwin go build` 零錯誤編譯。

- **階段二：POSIX 進程組與終端機 PTY**
  - 引入 `github.com/creack/pty` 實作 macOS 終端管理器。
  - 實作基於 `syscall.Setpgid` 的進程群組管理與級聯中斷清理。
  - 串接 `osascript` 實現 macOS 下的 hosts 安全寫入。

- **階段三：依賴二進位與環境整合**
  - 實作 Caddy、Redis、Mailpit 的 macOS (arm64 / amd64) 下載支援。
  - 整合 `static-php-cli` 預編譯的獨立 PHP-CGI 二進位套件，並支援 Homebrew 偵測。
  - 整合 MariaDB macOS 支援與外部 GUI (Sequel Ace) 連動。

- **階段四：macOS 封裝、簽名與測試**
  - 配置 `build/darwin/Info.plist` 與高解析度 `.icns` 應用程式圖示。
  - 測試 Apple Silicon (M1/M2/M3/M4) 與 Intel Mac 的原生執行。
  - 產出 `.dmg` 打包發行流程。

---

## 5. 項目更名提案 (跨平台後的全新品牌命名)

> **命名考量重點**：
> 1. **擺脫 `cmp` 縮寫的誤導**：`cmp` 在 Unix 指令與開發者社群中是極為著名的 `compare`（檔案比對）指令縮寫（容易讓人誤以為是類似 Beyond Compare 或 WinMerge 的比對工具）。
> 2. **凸顯「支援 Node.js 與多語言指令執行」之核心 Feature**：WinCMP 本質已從單純的 PHP/Web 伺服器，演進為能直接管理與執行 Node.js、Bun、Python、Go 等多語言指令（Task/Process Runner）與終端管線的一站式本機平台。
> 3. **避免與現有知名開源專案重複**：確保在 GitHub 生態系中具備獨立性與高搜尋能見度。

以下提供 3 個專注於「服務棧 + 執行環境 (Runner / Panel)」且不撞名的開源社群風格提案：

### 提案一：`StackRun` (服務棧與指令執行器 - 最推薦)
- **命名構詞**：`Stack` (Web 服務棧: Caddy / PHP / MariaDB / Redis) + `Run` (Node.js / Bun / Python / Go 多指令執行)。
- **設計理念**：
  - 徹底捨棄容易被聯想為比對工具的 "CMP"，直球替換為代表強大執行力的 "Run"。
  - 精準表達核心能力：不僅為你託管伺服器棧（Stack），還能一鍵執行（Run）任何全端專案與自訂指令。
- **社群風格特點**：
  - 極短（僅 8 個字母）、發音俐落，CLI 指令極為自然（例如：`stackrun start`、`stackrun dev`）。
  - 在 GitHub 開源社群中乾淨無重量級衝突，現代感與實用性兼備。

### 提案二：`DevRunner` (開發者行程與指令運行中心)
- **命名構詞**：`Dev` (Developer / Development) + `Runner` (多語言行程執行器 / 指令管線)。
- **設計理念**：
  - 直球切入「支援 Node 或其他語言指令執行」這項核心 Feature。
  - 在現代開發者認知中，"Runner" 代表能夠在背景自動化拉起進程、綁定 Port、擷取終端輸出與管理生命週期的強大引擎。
- **社群風格特點**：
  - 形象從傳統的「靜態伺服器面板」升級為「全能指令與專案執行中心」。
  - 易於在社群中推廣，開源開發者一眼便能理解其作為 Local Runner 的強大能力。

### 提案三：`StackPanel` (全棧服務與開發控制面板)
- **命名構詞**：`Stack` (全棧環境) + `Panel` (標準控制面板 Control Panel)。
- **設計理念**：
  - WinCMP 的本意原為 "Windows Control Panel"，直接將容易混淆的縮寫 `CMP` 還原為開源社群最通用的正式名詞 `Panel`。
  - "Panel" 絕對不會被誤解為 Compare，且最符合視覺化 GUI 管理（整合服務狀態、資料庫、Terminal 與即時日誌）的使用者預期。
- **社群風格特點**：
  - 沉穩、專業、可信賴，具備老牌開源工具的紮實質感。
  - 對原有面板功能的表達最為精確。


