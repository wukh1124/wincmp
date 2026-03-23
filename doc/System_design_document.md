# WinCMP System Design Document

目的： 建立專案成員間的開發共識，描述系統架構。

> **版本**: v1.0  
> **最後更新**: 2026-03-17  
> **維護者**: WinCMP 開發團隊

---

## 目錄

1. [系統概觀](#1-系統概觀)
2. [技術棧與依賴](#2-技術棧與依賴)
3. [目錄結構與職責](#3-目錄結構與職責)
4. [核心模組設計](#4-核心模組設計)
5. [請求流程與資料流](#5-請求流程與資料流)
6. [PHP-CGI 進程池機制](#6-php-cgi-進程池機制)
7. [Caddy 配置生成機制](#7-caddy-配置生成機制)
8. [已知架構問題：Self-Referencing Deadlock](#8-已知架構問題self-referencing-deadlock)
9. [緩解方案評估](#9-緩解方案評估)
10. [配置檔案規格](#10-配置檔案規格)
11. [安全與隔離設計](#11-安全與隔離設計)

---

## 1. 系統概觀

### 1.1 產品定位

WinCMP (**Win**dows + **C**addy + **M**ariaDB + **P**HP) 是一個專為 Windows 11 設計的 **可攜式 (Portable)**、**免管理員權限** 的本機開發環境控制面板。

### 1.2 設計原則

| 原則             | 說明                                                                       |
| ---------------- | -------------------------------------------------------------------------- |
| **可攜性**       | 免安裝、不修改系統環境變數、不寫入登錄檔（除非使用者主動啟用開機啟動）     |
| **零管理員權限** | 所有服務以普通使用者身份啟動，僅 Hosts 檔更新需要提升權限                  |
| **隔離性**       | 子進程的 `PATH` 環境變數透過動態注入，確保不同版本的 PHP 及其 DLL 互不干擾 |
| **簡單至上**     | 追求最少改動量，避免過度工程                                               |

[🔼 回目錄](#目錄)


### 1.3 核心場景

```
使用者啟動 WinCMP
    → 掃描 bin/ 目錄偵測已安裝的服務版本
    → 載入 conf/wincmp.json 設定
    → (選配) 根據上次服務狀態自動啟動 Caddy / MariaDB / PHP-CGI
    → 使用者透過 GUI 管理專案、啟停服務、瀏覽日誌
    → 關閉時儲存服務狀態並釋放所有子進程
```

---

## 2. 技術棧與依賴

### 2.1 編譯環境

| 組件                | 版本    | 用途                                |
| ------------------- | ------- | ----------------------------------- |
| Go                  | 1.25.7+ | 主程式語言                          |
| Fyne                | v2.7.3  | 跨平台 GUI 框架 (基於 Cgo + OpenGL) |
| WinLibs (MinGW-w64) | Latest  | C 編譯器 (Fyne 所需的 Cgo 依賴)     |

### 2.2 核心第三方套件

| 套件                                | 用途                             |
| ----------------------------------- | -------------------------------- |
| `fyne.io/fyne/v2`                   | GUI 框架 (含 System Tray 支援)   |
| `fyne.io/systray`                   | Windows 系統匣整合               |
| `github.com/ncruces/zenity`         | 原生檔案選擇器對話框             |
| `github.com/dweymouth/fyne-tooltip` | 為 Fyne Widget 添加 Tooltip 支援 |
| `gopkg.in/natefinsh/lumberjack.v2`  | 日誌滾動與保留期限管理           |

### 2.3 管理的外部服務

| 服務        | 類型                              | 通訊協議                            |
| ----------- | --------------------------------- | ----------------------------------- |
| **Caddy**   | HTTP/3、HTTP/2、HTTP/1.1 反向代理 | 管理 API: `localhost:2019`          |
| **MariaDB** | 關聯式資料庫                      | TCP: `localhost:3306`               |
| **PHP-CGI** | FastCGI 處理引擎                  | FastCGI over TCP: `127.0.0.1:3xxxx` |

[🔼 回目錄](#目錄)


---

## 3. 目錄結構與職責

```text
wincmp/
├── wincmp.exe               # Go 編譯產物：主程式 (含 GUI + 進程管理)
├── main.go                  # 應用程式進入點：GUI 建構、事件綁定、配置生成
│
├── internal/                # 核心業務邏輯 (不對外暴露)
│   ├── config/              # JSON 設定檔的讀寫與資料結構定義
│   │   └── config.go        #   WincmpConfig, GlobalConfig, ProjectConfig
│   ├── scanner/             # bin/ 目錄掃描器：偵測已安裝服務版本與 Port 計算
│   │   └── scanner.go       #   ScanBinDir(), PHPVersionInfo, ServiceInfo
│   ├── process/             # 子進程生命週期管理器
│   │   ├── manager.go       #   Manager 核心：register/unregister/StopAll
│   │   ├── caddy.go         #   StartCaddy / StopCaddy / ReloadCaddy
│   │   ├── mariadb.go       #   StartMariaDB / StopMariaDB
│   │   └── php.go           #   StartPHPCGI / StopPHPCGI (多進程)
│   ├── detect/              # 專案類型偵測器
│   │   └── laravel.go       #   DetectLaravel(): 信心分數制判定
│   └── hosts/               # Windows Hosts 檔管理
│       └── hosts.go         #   CheckHosts / UpdateHosts / BackupHosts
│
├── conf/                    # 配置文件中心
│   ├── wincmp.json          # ★ 核心設定檔 (全域 + 專案列表)
│   ├── Caddyfile            # Caddy 進入點 (import snippets & sites)
│   ├── my.ini               # MariaDB 啟動配置
│   ├── snippets/            # Caddy 共用配置片段
│   │   ├── common.caddy     #   共用 headers、日誌、IP 白名單
│   │   └── php-upstream.caddy # ★ 自動生成的 PHP 負載均衡定義
│   ├── sites/               # ★ 自動生成的專案站點配置
│   │   ├── bs_api.caddy
│   │   └── ...
│   └── ssl/                 # SSL 憑證 (*.crt, *.key)
│
├── bin/                     # 服務二進制檔 (使用者自備或下載器取得)
│   ├── caddy/               # caddy-x.y.z/caddy.exe
│   ├── mariadb/             # mariadb-x.y.z/bin/mariadbd.exe
│   └── php/                 # php-x.y.z/php-cgi.exe
│
├── data/                    # 持久化資料
│   ├── mariadb/             # MariaDB Data 目錄
│   └── backup/hosts/        # Hosts 備份檔
│
├── logs/                    # 日誌輸出 (依日期分檔)
│   ├── wincmp-YYYY-MM-DD.log  # 應用程式日誌
│   ├── error-YYYY-MM-DD.log   # 獨立錯誤日誌
│   ├── caddy.log              # Caddy 自身日誌 (由 Caddy 管理)
│   └── access.log             # HTTP 存取日誌 (由 Caddy 管理)
│
├── www/                     # 預設網頁專案根目錄
└── bat/                     # 備份用啟動腳本 (調試參考)
```

---

## 4. 核心模組設計

### 4.1 模組依賴關係圖

```
┌──────────────────────────────────────────────────────┐
│                     main.go                          │
│  (GUI 建構 · 事件處理 · Caddyfile 生成 · Hosts 觸發) │
└──────────┬──────────┬──────────┬──────────┬──────────┘
           │          │          │          │
    ┌──────▼──────┐   │   ┌──────▼──────┐   │
    │   config    │   │   │   scanner   │   │
    │ (設定讀寫)  │   │   │ (版本偵測)  │   │
    └─────────────┘   │   └─────────────┘   │
               ┌──────▼──────┐       ┌──────▼──────┐
               │   process   │       │   detect    │
               │ (進程管理)  │       │ (專案偵測)  │
               └─────────────┘       └─────────────┘
                      │
               ┌──────▼──────┐
               │    hosts    │
               │ (Hosts 管理)│
               └─────────────┘
```

### 4.2 `internal/scanner` — 版本掃描器

**職責**：啟動時掃描 `bin/` 目錄，自動偵測所有已安裝的服務版本。

**關鍵資料結構**：

```go
// ServiceInfo — Caddy / MariaDB 共用
type ServiceInfo struct {
    Name    string // "caddy" | "mariadb"
    Version string // "2.11.1" | "11.4.10"
    ExePath string // 完整執行檔路徑
}

// PHPVersionInfo — PHP 專用 (含 Port 配置)
type PHPVersionInfo struct {
    Version   string // "8.2.30"
    ExePath   string // php-cgi.exe 的完整路徑
    MajorMin  string // "8.2" (用於 Port 計算與設定 key)
    PortBase  int    // Port 基數 (如 38200)
    PortCount int    // 進程數量 (預設 3，可設定 3/10/20/.../100)
}
```

**PHP 版本去重規則**：同一個 Minor Version (如 8.2) 若存在多個 Patch 版本，只保留最新的，舊的記錄到 `SkippedPHP` 並輸出 log。

### 4.3 `internal/process` — 進程管理器

**職責**：管理所有子進程的完整生命週期 (啟動 → 監控 → 停止)。

**核心設計**：

```go
type Manager struct {
    mu       sync.Mutex
    services map[string]*ServiceState // key 格式: "caddy" | "mariadb-11.4" | "php-8.2.30"
    baseDir  string
    logFn    LogFunc
    errLogFn ErrorLogFunc
}

type ServiceState struct {
    Name      string
    Running   bool
    ExePath   string
    Commands  []*exec.Cmd  // PHP 可能有多個子進程
    PIDs      []int
    StartTime time.Time
    Ctx       context.Context    // 用於通知 Uptime 監控 goroutine
    Cancel    context.CancelFunc
}
```

**服務啟動流程**：

| 服務        | 啟動方式                                                   | 特殊處理                                               |
| ----------- | ---------------------------------------------------------- | ------------------------------------------------------ |
| **Caddy**   | `caddy run --config Caddyfile --adapter caddyfile --watch` | 支援 `caddy reload` 熱重載                             |
| **MariaDB** | `mariadbd.exe --defaults-file=conf/my.ini --console`       | 停止時先嘗試 `mariadb-admin shutdown`，失敗才強制 Kill |
| **PHP-CGI** | 每個 Port 啟動一個 `php-cgi.exe -b 127.0.0.1:<port>`       | 動態注入 PATH 環境變數確保 DLL 依賴                    |

### 4.4 `internal/config` — 配置管理

**職責**：管理 `conf/wincmp.json` 的序列化/反序列化，並提供路徑推導輔助函式。

**路徑推導邏輯**：

- `GetProjectRoot()`: 若有自訂 RootPath 則使用，否則拼接 `DefaultWWW + ProjectName`。若 Type 為 `laravel`，自動附加 `/public`。
- `GetSSLCertPath()` / `GetSSLKeyPath()`: 若有自訂路徑則使用，否則以 `DefaultSSL + 第一個 Domain + .crt/.key` 推導。

### 4.5 `internal/detect` — 專案類型偵測

**職責**：透過**信心分數制**判定專案是否為 Laravel。

**計分規則** (總分 ≥ 50 判定為 Laravel)：

| 檢查項目                               | 分數 |
| -------------------------------------- | ---- |
| `composer.json` 含 `laravel/framework` | +35  |
| `artisan` 檔案存在                     | +30  |
| `bootstrap/app.php`                    | +25  |
| `public/index.php`                     | +20  |
| `routes/web.php`                       | +10  |
| `config/app.php`                       | +10  |
| `app/Http`                             | +8   |
| `resources/views`                      | +6   |
| `database/migrations`                  | +6   |
| `storage`                              | +5   |

### 4.6 `main.go` — GUI 與協調器

**職責**：應用程式進入點，負責 GUI 建構、事件處理、配置文件生成、服務狀態協調。

**GUI 結構** (`Fyne v2.7`)：

```
┌─────────────────────────────────────────────┐
│ WinCMP Control Panel                        │
├─────────┬───────────────────────────────────┤
│ 側邊選單 │                                   │
│ ───────  │   上方功能區 (65%)                │
│ Dashboard│   ┌───────────────────────────┐   │
│ Projects │   │ Dashboard / Projects /    │   │
│ DB Explor│   │ DB Explorer / Settings    │   │
│ Settings │   └───────────────────────────┘   │
│          ├───────────────────────────────────┤
│          │   下方 Log 區 (35%)               │
│          │   ┌─────────────────────────────┐ │
│          │   │ [System][Caddy][MariaDB][PHP]│ │
│          │   │ Terminal Logs                │ │
│          │   └─────────────────────────────┘ │
└──────────┴───────────────────────────────────┘
```

---

## 5. 請求流程與資料流

### 5.1 HTTP 請求處理流程 (正常情況)

```
瀏覽器
  │
  ▼ (HTTPS / HTTP)
┌───────────┐
│   Caddy   │ ─── 監聽 Port 80, 443
│ (Reverse  │     ↓ 根據 Caddyfile 的 Domain 匹配
│  Proxy)   │
└─────┬─────┘
      │ FastCGI over TCP
      ▼
┌───────────┐
│ PHP-CGI   │ ─── 多進程池 (如 38200~38207)
│ (FastCGI  │     Caddy 以 Round-Robin 分配
│  Workers) │
└─────┬─────┘
      │ (若 PHP 需要查詢 DB)
      ▼
┌───────────┐
│ MariaDB   │ ─── Port 3306
└───────────┘
```

### 5.2 Caddy → PHP-CGI 的 FastCGI 分流機制

Caddy 使用 `php_fastcgi` 指令搭配多個 upstream 地址做負載均衡：

```caddyfile
# conf/snippets/php-upstream.caddy (自動生成)
(php82) {
    php_fastcgi 127.0.0.1:38200 127.0.0.1:38201 127.0.0.1:38202 ... 127.0.0.1:38207
}
```

```caddyfile
# conf/sites/bs_api.caddy (自動生成)
local-api.domain.xyz {
    tls C:/.../ssl/domain.xyz.chained.crt C:/.../ssl/domain.xyz.key
    import common_dev
    root * C:/.../bs_api/public
    import php82          # ← 引用上方的 snippet
    import static_site
}
```

### 5.3 服務啟停事件流

```
使用者點擊 "Start" (Dashboard)
    │
    ├─→ checkSSLCerts()           # 驗證所有啟用專案的 SSL 檔案存在
    ├─→ generateCaddyfiles()      # 1. generatePHPUpstream() → php-upstream.caddy
    │                             # 2. 遍歷 Projects → sites/*.caddy
    ├─→ procMgr.StartCaddy()      # 啟動 Caddy 子進程
    ├─→ triggerHostsUpdate()       # 檢查 & 更新 Windows Hosts 檔
    └─→ monitorUptime()           # 背景 goroutine 每秒更新計時

使用者點擊 "Stop"
    │
    ├─→ procMgr.StopCaddy()       # Kill 子進程
    └─→ unregister()              # 清理狀態 + Cancel context
```

---

## 6. PHP-CGI 進程池機制

### 6.1 Port 命名規則

WinCMP 採用 **`3<主版本><次版本><序號00~99>`** 的規則分配 Port：

| PHP 版本 | 進程數   | Port 範圍                | 計算公式                          |
| -------- | -------- | ------------------------ | --------------------------------- |
| 7.3      | 3 (預設) | 37300, 37301, 37302      | `37000 + 7*100 + 3*10 + [00..02]` |
| 8.2      | 8 (設定) | 38200, 38201, ..., 38207 | `38000 + 8*100 + 2*10 + [00..07]` |

**PHP-CGI 端口範圍分配備忘**

> PHP-FPM 的預設端口通常是 9000
> 37xxx - 38xxx 這個區段並沒有被廣泛使用的知名服務佔用。
> 若擔心與其他註冊軟體衝突，可選用 49152 以上。這是 IANA 定義的「私有/動態端口」範圍，不會有官方預留的服務

### 6.2 進程管理特性

- **每版本可配置 3~100 個進程**，透過 Settings → `wincmp.json` 中的 `php.processes` (per Minor Version) 管理。
- **環境隔離**：啟動 `php-cgi.exe` 時動態將 PHP 所在目錄注入 `PATH`，確保 DLL 依賴 (如 `libssl.dll`) 正確載入。
- **部分進程退出監控**：若個別 php-cgi 進程異常退出，僅記錄 log，不自動 unregister 整個服務 (其他進程可能仍在運行)。

### 6.3 目前限制

> ⚠️ 當某個 php-cgi 進程崩潰時，Caddy 仍會將請求分配到該 Port，導致請求失敗。目前缺乏進程自動重啟 (Auto-Restart) 與健康檢查 (Health Check) 機制。

### 6.4 進程自動重啟與健康檢查方案研究

> 📋 **研究結論**，尚未實作，供決策參考。

#### 方向 A：WinCMP 層 — Go Goroutine Watchdog（進程自動重啟）

**原理**：目前 `internal/process/php.go` 中的 goroutine 已在 `cmd.Wait()` 後記錄 log，只需在此基礎上 **加入重啟邏輯**即可。

**關鍵注意事項**：

- `php-cgi.exe` 預設在處理 **500 個請求** 後會正常自行退出（非崩潰），這是 `PHP_FCGI_MAX_REQUESTS` 環境變數的行為。Watchdog 必須同時處理「正常退出」與「崩潰退出」兩種情況。
- 可設定 `PHP_FCGI_MAX_REQUESTS=0` 停用此限制，但有記憶體洩漏風險（長時間運行的生產環境不建議，本地開發可接受）。

**概念性程式碼**（尚未實作）：

```go
// 在啟動每個 php-cgi 進程的 goroutine 中
go func(cmd *exec.Cmd, port int) {
    err := cmd.Wait()
    if !isIntentionallyStopped { // 判斷是否為使用者主動停止
        log.Printf("⚠️ PHP-CGI port %d 異常退出，3 秒後自動重啟...", port)
        time.Sleep(3 * time.Second)
        restartPHPCGI(port) // 重新啟動同一 Port 的進程
    }
}(cmd, port)
```

**設計決策點**：
| 問題 | 選項 |
|------|------|
| 重啟冷卻時間 | 固定 3s / 指數退避 (3s → 10s → 30s) |
| 重啟上限 | 無限制 / 每分鐘最多 N 次 |
| 重啟後是否通知 UI | 靜默 / 在 Log 頁面顯示 badge |

---

#### 方向 B：Caddy 層 — php_fastcgi Health Check（健康檢查）

**原理**：Caddy 的 `php_fastcgi` 繼承自 `reverse_proxy`，支援 **主動健康檢查 (Active Health Check)**，可定期探測各 upstream 端口，自動將不健康的端口移出輪換池。

**Active Health Check 配置範例**（需在 Caddyfile 模板中加入）：

```caddyfile
(php82) {
    php_fastcgi 127.0.0.1:38200 127.0.0.1:38201 ... 127.0.0.1:38207 {
        # 主動健康檢查：FastCGI 層用 TCP 連線探測
        health_uri     /health-check.php   # 需要一個固定存在的 PHP 檔案
        health_interval 5s                 # 每 5 秒探測一次
        health_timeout  2s                 # 探測超時
        health_status   2xx                # 回應 2xx 視為健康
        health_fails    2                  # 連續失敗 2 次才標記不健康
        health_passes   1                  # 1 次成功即恢復
    }
}
```

**Passive Health Check**（預設即存在，無需額外配置）：

```caddyfile
(php82) {
    php_fastcgi 127.0.0.1:38200 ... {
        # Caddy 預設：若請求失敗，短暫移除該 upstream 並換下一個
        # 以下可調整被動健康檢查的行為
        fail_duration      10s   # 被標記不健康後，暫時移除持續 10 秒
        max_fails          3     # 最多允許連續 3 次失敗才觸發
        unhealthy_latency  2s    # 超過此延遲也算「不健康」
    }
}
```

**Active Health Check 的前提限制**：

- 需要在每個 PHP 專案的 `public/` 下放置一個 `/health-check.php`（或利用 Laravel 的 `/health` route），Caddy 才能做 HTTP 探測。
- Caddy 的 `php_fastcgi` 健康檢查實質上是用 **HTTP 請求** 探測 FastCGI 端點，因此需要有一個可回應的 PHP 頁面，對於「空崩潰的 Port」（TCP 連線直接拒絕）則靠 **Passive Check** 偵測。

---

#### 兩個方向的比較

| 面向                     | 方向 A (Watchdog)      | 方向 B (Caddy Health Check)                  |
| ------------------------ | ---------------------- | -------------------------------------------- |
| 解決「崩潰後自動恢復」   | ✅ 直接重啟進程        | ❌ 只是不分配流量，進程仍死亡                |
| 解決「崩潰期間不丟請求」 | ❌ 重啟期間端口仍無效  | ✅ Passive Check 可改分配其他端口            |
| 實작 位置                | WinCMP Go 程式碼       | Caddyfile 模板（`generatePHPUpstream` 函數） |
| 需要 PHP 端配合          | 否                     | Active Check 需要存在 `/health` 端點         |
| 複雜度                   | 中（需處理重啟狀態機） | 低（Passive）/ 中（Active）                  |

> 💡 **建議組合策略**：**A + B Passive**。先讓 Caddy Passive Check 在最短時間內避開死亡端口（防止請求失敗），同時由 Watchdog 在背景重啟崩潰的進程（恢復完整容量）。Active Check 可選配，取決於是否願意維護 `/health-check.php`。

## 7. Caddy 配置生成機制

### 7.1 生成流程

```
main.go generateCaddyfiles()
  │
  ├─→ generatePHPUpstream()
  │     讀取 scanRes.PHPList + appCfg.Global.PHP.Processes
  │     輸出: conf/snippets/php-upstream.caddy
  │       (php82) { php_fastcgi 127.0.0.1:38200 ... }
  │
  ├─→ 清除 conf/sites/*.caddy 舊檔案
  │
  └─→ 遍歷 appCfg.Projects (僅 Enabled == true)
        對每個專案生成: conf/sites/<name>.caddy
          - Domain 設定
          - TLS 憑證路徑 (若啟用 SSL)
          - import common_dev
          - root * <project_root>
          - import php<ver>
          - import static_site
```

### 7.2 Caddyfile 引入結構

```
conf/Caddyfile (入口)
├── import snippets/*.caddy
│   ├── common.caddy        (headers, log, IP 白名單)
│   └── php-upstream.caddy  (PHP FastCGI 負載均衡定義)
│
└── import sites/*.caddy
    ├── bs_api.caddy
    ├── cms_global.caddy
    └── pg_api_global.caddy
```

### 7.3 熱重載

- Caddy 以 `--watch` 模式啟動，可自動偵測 Caddyfile 變更。
- GUI 提供 Reload 按鈕，呼叫 `caddy reload --config Caddyfile --adapter caddyfile`。

---

## 8. 已知架構問題：Self-Referencing Deadlock

> [!WARNING]
> 本節僅為摘要。詳細的故障排除過程與證據，請參閱獨立報告：[Docs_PHP-CGI_Deadlock_Analysis.md](Docs_PHP-CGI_Deadlock_Analysis.md)。

### 8.1 問題描述


當 Laravel 應用程式存在**自引用請求** (Self-Referencing Request) 的場景時 — 即 PHP Request A 在處理過程中透過 cURL 對**同一個 Caddy** 發起另一個 PHP Request B — 會產生 **php-cgi 進程池耗盡**的死鎖問題。

### 8.2 重現步驟

以 `bs_api` 專案的 JS API Tester 為例：

1. 瀏覽器向 `Caddy` 發送 `/test/devtools/js_api_tester` 的 proxy 請求
2. `php-cgi-A` 處理此 proxy 請求，內部透過 cURL 向同一個 Caddy 發起 `/broker/auth/token/verify`
3. Caddy 需分配 `php-cgi-B` 來處理 verify 請求
4. **每次 Send 因此同時佔用 2 個 php-cgi 進程**

### 8.3 時間線分析

```
                    Caddy 進程池 (8 個 php-cgi)
                    ┌─┬─┬─┬─┬─┬─┬─┬─┐
 Send 1:  使用 2 個  │A│B│ │ │ │ │ │ │  可用: 6
 Send 2:  使用 2 個  │A│B│A│B│ │ │ │ │  可用: 4
 Send 3:  使用 2 個  │A│B│A│B│A│B│ │ │  可用: 2
 Send 4:  使用 2 個  │A│B│A│B│A│B│A│B│  可用: 0  ← 理論極限
                    └─┴─┴─┴─┴─┴─┴─┴─┘

 但實際因 Windows TCP TIME_WAIT 回收延遲 + Caddy FastCGI 連線池回收時差：
 Send 7~8: 新請求的「proxy」佔走最後可用進程
           → proxy 內部 cURL 需要另一個進程 → 無可用進程
           → 死鎖！cURL 等待 php-cgi 釋放，但 php-cgi 卡在等 cURL 完成
           → 30 秒後 cURL 超時 → 所有進程釋放 → 再點一次就成功
```

### 8.4 關鍵證據

| 指標                                     | 值       | 說明                            |
| ---------------------------------------- | -------- | ------------------------------- |
| `/broker/auth/token/verify` PHP 執行時間 | 138ms    | PHP 本身處理很快                |
| Caddy 報告的該請求總耗時                 | 30.56 秒 | 差了 30.4 秒 = 在排隊等 php-cgi |
| cURL Timeout                             | 30 秒    | proxy 端 cURL 超時              |

### 8.5 根本原因

```
瀏覽器 ─(HTTPS)─▶ Caddy ─(FastCGI)─▶ php-cgi-A [處理 proxy，佔用中]
                                         │
                                         └── cURL ─(HTTPS)─▶ Caddy ─(FastCGI)─▶ php-cgi-B
                                                                                    ↑
                                                                       需要空閒的 php-cgi 進程！
```

**這是一個結構性問題**：應用層 (Laravel) 的請求路由設計，導致同一個 FastCGI 進程池需要處理「外部請求」與「內部自引用請求」，形成資源競爭。

---

## 9. 緩解方案評估

> 以下按「實作成本」與「效果」排序，建議優先評估前三項。

### 方案 A：增加 PHP-CGI 進程數 (治標：簡單)

**原理**：加大進程池，延後耗盡的臨界點。

**操作**：在 WinCMP Settings 將 PHP 8.2 的進程數從 8 調高到 15~20。

| 優點                           | 缺點                       |
| ------------------------------ | -------------------------- |
| 零程式碼修改，設定即生效       | 只延後而非根除死鎖         |
| 設定 UI 已支援 3/10/20/.../100 | 記憶體佔用隨進程數線性增長 |
|                                | 每個 php-cgi 約佔 15-30MB  |

**結論**：作為**短期暫解**可行，但不解根因。可調高進程數以延後耗盡臨界點。

---

### 方案 B：Laravel 側改用直接函式呼叫取代 cURL 自引用 (治本：應用層)

**原理**：避免「PHP → cURL → Caddy → PHP」的自引用迴路，改為「PHP → 直接呼叫內部 Service 函式」。

```php
// ❌ 原本的做法 (會佔用 2 個 php-cgi)
$response = Http::post('https://local-api.domain.xyz/broker/auth/token/verify', [...]);

// ✅ 改為直接呼叫內部 Service (只佔 1 個)
$result = app(TokenService::class)->verify($token);
```

| 優點                            | 缺點                               |
| ------------------------------- | ---------------------------------- |
| 完全消除自引用死鎖              | 需修改 Laravel 應用程式碼          |
| 效能提升 (省去 cURL + TLS 開銷) | 僅適用於「呼叫自身 API」的場景     |
| 零記憶體開銷增加                | 需要 API 端點有對應的 Service 類別 |

**結論**：**最佳方案**，但需由 Laravel 開發者配合修改。對於 JS API Tester 這類用途，可考慮直接走內部呼叫。

---

### 方案 C：為自引用請求設置獨立的 PHP-CGI Pool (治本：WinCMP 層)

**原理**：建立兩組 php-cgi 進程池 — 一組處理外部請求、一組專門處理內部自引用請求，避免資源競爭。

```
                    外部 Pool (Port 38200~38207)
瀏覽器 → Caddy  ──→ php-cgi (外) ─── cURL ───→ Caddy ──→ php-cgi (內)
                                                          ↑
                                               內部 Pool (Port 38250~38255)
```

**Caddy 配置範例**：

```caddyfile
# 外部流量用的 snippet
(php82) {
    php_fastcgi 127.0.0.1:38200 127.0.0.1:38201 ... 127.0.0.1:38207
}

# 內部自引用用的 snippet (透過不同的 Domain 或 Path 匹配)
(php82_internal) {
    php_fastcgi 127.0.0.1:38250 127.0.0.1:38251 ... 127.0.0.1:38255
}
```

| 優點                    | 缺點                                  |
| ----------------------- | ------------------------------------- |
| 根除死鎖問題            | 需修改 WinCMP 的進程啟動邏輯          |
| 不需修改 Laravel 程式碼 | 記憶體佔用增加                        |
| 對應用層透明            | 需要定義「哪些請求走內部 Pool」的規則 |

**結論**：如果 Laravel 側無法修改，這是最佳的 WinCMP 層解決方案，但複雜度較高。

---

### 方案 D：啟用 Caddy 的 FastCGI 連線逾時配置 (治標：改善體驗)

**原理**：縮短 FastCGI 請求等待超時時間，讓失敗更快發生、更快釋放資源。

```caddyfile
(php82) {
    php_fastcgi 127.0.0.1:38200 ... {
        dial_timeout  5s   # 連線超時：5 秒
        read_timeout  60s  # 讀取超時
        write_timeout 10s  # 寫入超時
    }
}
```

| 優點                       | 缺點                              |
| -------------------------- | --------------------------------- |
| 降低死鎖時使用者的等待時間 | 不解決根因                        |
| 只需修改 Caddy 配置模板    | 過短的 timeout 可能影響正常慢查詢 |

**結論**：作為輔助手段，搭配方案 A 使用。

---

### 方案 E：引入 PHP-FPM 取代 PHP-CGI (長期方案)

**原理**：PHP-FPM 內建完善的進程池管理、自動回收、健康檢查機制。

| 優點                              | 缺點                            |
| --------------------------------- | ------------------------------- |
| 業界標準方案                      | PHP-FPM 原生不支援 Windows      |
| 自動管理進程數 (dynamic/ondemand) | 需要 WSL 或其他解決方案         |
| 支援 `pm.max_children` 精確控制   | 偏離 WinCMP「免安裝」的設計原則 |

**結論**：不適合 WinCMP 的「純 Windows 可攜式」定位，**暫不建議**。

---

### 方案建議總結

| 優先順序 | 方案  | 層級           | 效果     | 實作成本            |
| -------- | ----- | -------------- | -------- | ------------------- |
| ⭐⭐⭐   | **B** | Laravel 應用層 | 根治     | 中 (需修改 Laravel) |
| ⭐⭐     | **A** | WinCMP 設定    | 暫解     | 低 (調高進程數上限) |
| ⭐⭐     | **D** | Caddy 配置     | 改善體驗 | 低 (修改模板)       |
| ⭐       | **C** | WinCMP 架構    | 根治     | 高 (雙 Pool 機制)   |
| ❌       | **E** | 系統架構       | 根治     | 極高 (偏離產品定位) |

---

## 10. 配置檔案規格

### 10.1 `conf/wincmp.json` 完整結構

```jsonc
{
  "global": {
    // ===== 路徑設定 =====
    "default_www": "www", // 預設網頁專案根目錄
    "default_ssl": "conf/ssl", // 預設 SSL 憑證目錄
    "log_file": "", // (保留欄位)

    // ===== 系統行為 =====
    "auto_start": false, // 啟動時自動還原上次服務狀態
    "minimize_to_tray": false, // 關閉視窗時縮小到系統匣
    "run_on_boot": false, // Windows 開機自動啟動
    "auto_update_hosts": true, // 自動更新 Windows Hosts 檔

    // ===== 服務狀態快照 =====
    "last_service_state": {
      "caddy": true,
      "mariadb": true,
      "php": {
        "8.2.30": true, // key = 完整版本號
        "7.3.33": false,
      },
    },

    // ===== 日誌設定 =====
    "log_level": "All",
    "log_to_console": true, // 同步輸出日誌到 Console
    "max_log_retention": 3, // 日誌保留天數

    // ===== PHP 進程設定 =====
    "php": {
      "processes_per_version": 3, // 預設每版本進程數
      "processes": {
        // 個別版本覆寫 (key = Minor Version)
        "8.2": 8,
        "7.3": 3,
      },
      "base_port_mapping": null, // (保留欄位：未來可自訂 Port 基數)
    },
  },
  "projects": [
    {
      "id": "", // (保留欄位)
      "name": "bs_api", // 專案名稱 (也作為 sites/*.caddy 檔名)
      "domains": ["local-api.domain.xyz"],
      "type": "laravel", // "" | "laravel"
      "php_version": "8.2", // Minor Version (對應 php-upstream.caddy)
      "root_path": "C:\\path\\to\\bs_api", // 專案根目錄 (Laravel 自動加 /public)
      "ssl_crt": "C:\\path\\to\\*.crt", // 自訂 SSL 證書路徑
      "ssl_key": "C:\\path\\to\\*.key", // 自訂 SSL 金鑰路徑
      "use_ssl": true,
      "enabled": true,
    },
  ],
}
```

### 10.2 設定自動儲存機制

Settings 頁面的所有變更採用 **Debounce 模式** (800ms 延遲) 自動儲存：

- 每次 UI 控件內容變更時，重設 800ms 計時器
- 計時器到期後，一次性收集所有最新值，寫入 `wincmp.json`
- 變更前後會對比新舊值，僅在實際變更時才觸發儲存並輸出日誌

---

## 11. 安全與隔離設計

### 11.1 環境變數動態注入

```go
// internal/process/php.go - StartPHPCGI()
phpDir := filepath.Dir(phpInfo.ExePath) // 取得 php-cgi.exe 所在目錄
env := append(os.Environ(),
    fmt.Sprintf("PATH=%s;%s", phpDir, os.Getenv("PATH")),
)
cmd.Env = env
```

**目的**：確保 php-cgi 能找到同目錄下的 DLL (如 `libcrypto.dll`, `libssl.dll`)，且不污染系統全域 `PATH`。

### 11.2 Caddy 安全設定

```caddyfile
{
    auto_https off       # 停用自動 HTTPS (本地開發不需要)
    admin localhost:2019  # 管理 API 僅綁定 localhost
    persist_config off    # 不持久化運行時配置
}
```

### 11.3 IP 白名單 (可選)

```caddyfile
(common-ip-allow) {
    @blockedIps not remote_ip 10.150.0.0/24 10.60.0.0/24 127.0.0.1
    handle @blockedIps {
        respond "Forbidden" 403
    }
}
```

### 11.4 Hosts 檔安全

- 每次修改前**自動備份**到 `data/backup/hosts/hosts_<timestamp>`
- 寫入時僅使用 `Append` 模式，不刪除已有內容
- 新增的記錄帶有 `# Added by WinCMP at <timestamp>` 標記

---

> **附註**：本文件將隨專案迭代持續更新。對於任何架構變更，請先在此文件中記錄設計思路，再進行實作。
