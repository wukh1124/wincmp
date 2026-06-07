# WinCMP 項目結構與技術棧

> **版本**: v2.0  
> **最後更新**: 2026-06-03  
> **維護者**: WinCMP 開發團隊

---

## 目錄

1. [項目概述](#1-項目概述)
2. [技術棧](#2-技術棧)
3. [項目結構](#3-項目結構)
4. [核心模組說明](#4-核心模組說明)
5. [前後端通訊機制](#5-前後端通訊機制)
6. [依賴關係圖](#6-依賴關係圖)
7. [GUI 佈局架構](#7-gui-佈局架構)

---

## 1. 項目概述

**WinCMP** (**Win**dows + **C**addy + **M**ariaDB + **P**HP) 是一個專為 Windows 設計的**可攜式 (Portable)**、**免管理員權限**的本機開發環境控制面板。

從 v2.0 開始，WinCMP 的 GUI 核心已從 Go Fyne 框架全面重構為基於現代化 Web 技術棧的 **Wails v2 + React 18**。這帶來了更精美的專業 UI、更流暢的動畫交互、以及更強大的日誌渲染能力。

### 1.1 設計原則

| 原則 | 說明 |
|------|------|
| **可攜性** | 免安裝、不修改系統環境變數、不寫入登錄檔 |
| **零管理員權限** | 所有服務以普通使用者身份啟動，僅 Hosts 檔更新與自動下載依賴時可能需要 UAC 權限 |
| **隔離性** | 子進程的 `PATH` 環境變數透過動態注入，確保不同版本的 PHP/Runtime 及其 DLL 互不干擾 |
| **極致效能** | Go 後端進行高效能進程管理，前端使用 Zustand 做精確的狀態渲染，搭配 xterm.js 處理高性能日誌流 |

---

## 2. 技術棧

### 2.1 後端編譯環境 (Go Core)

| 組件 | 版本 | 用途 |
|------|------|------|
| **Go** | 1.26.2+ | 主程式語言，處理核心系統與進程生命週期 |
| **Wails** | v2.12.0 | 跨平台桌面應用框架 (Go + Web-frontend binding) |
| **WinLibs (MinGW-w64)**| Latest | C/C++ 編譯器 (Wails 底層對 Windows API 依賴的編譯) |

### 2.2 前端編譯環境 (Web-Frontend)

| 組件 | 版本 | 用途 |
|------|------|------|
| **React** | 18+ | 前端 UI 渲染框架 |
| **TypeScript** | 5.x | 前端型別安全與邏輯開發 |
| **Vite** | Latest | 開發熱重載與前端靜態打包工具 |
| **Node.js** | 18+ | 前端套件依賴與打包編譯環境 |

### 2.3 核心第三方與前端套件

| 套件 (後端 Go) | 版本 | 用途 |
|------|------|------|
| `github.com/wailsapp/wails/v2` | v2.12.0 | 前後端橋接 API 與視窗生命週期管理 |
| `github.com/shirou/gopsutil/v3` | v3.24.5 | 監控系統硬體 (CPU %、RAM MB 使用率) |
| `github.com/nicksnyder/go-i18n/v2`| v2.5.1 | 多國語言 (Traditional Chinese / English) 本地化套件 |
| `gopkg.in/natefinch/lumberjack.v2`| v2.2.1 | 日誌滾動與保留期限管理 |
| `github.com/go-sql-driver/mysql` | v1.9.3 | MariaDB / MySQL 資料庫驅動 |
| `golang.org/x/sys` | v0.42.0 | 系統層級操作 (進程生命週期與 Hosts 管理) |

| 套件 (前端 React) | 用途 |
|------|------|
| `Zustand` | 全域狀態管理 (避免 React Prop drilling 與非必要渲染) |
| `Tailwind CSS` | 樣式系統，實作自訂 HSL 設計標記的 Dark Professional 主題 |
| `shadcn/ui` (Radix) | 高品質無障礙前端組件庫 |
| `xterm.js` | 高性能虛擬終端渲染，用於即時系統/服務日誌流輸出 |
| `Lucide React` | UI 圖示庫 |

### 2.4 管理的外部服務

| 服務 | 類型 | 通訊協議 |
|------|------|----------|
| **Caddy** | Web 伺服器 (HTTP/3 反向代理) | 管理 API: `localhost:2019` |
| **MariaDB** | 關聯式資料庫 | TCP: `localhost:3306` |
| **PHP-CGI** | FastCGI 處理引擎 | FastCGI over TCP: `127.0.0.1:3xxxx` |
| **Mailpit** | 郵件測試與捕獲服務 | SMTP: `1025`, HTTP GUI: `8025` |

---

## 3. 項目結構

```
wincmp/
├── main.go                  # 應用程式進入點：初始化與啟動 Wails 視窗
├── app.go                   # Wails 生命週期管理器 (startup, shutdown) 及核心監控推送
├── bridge.go                # Wails 與 Go 端的主 Binding API (前後端 RPC 接口)
├── downloader_bridge.go     # Wails 下載管理器 Binding 接口
├── wincmp.json              # WinCMP 全域與專案配置檔案
│
├── internal/                 # 核心業務邏輯 (不對外暴露，且不包含 GUI 邏輯)
│   ├── config/               # JSON 設定檔的讀寫與資料結構定義
│   │   ├── config.go         #   WincmpConfig, GlobalConfig, ProjectConfig
│   │   └── dependencies.go   #   自動下載依賴版本配置與 Fetch 功能
│   ├── scanner/              # bin/ 目錄掃描器：偵測已安裝服務版本與 Port 計算
│   │   └── scanner.go        #   ScanBinDir(), PHPVersionInfo, ServiceInfo
│   ├── process/              # 子進程生命週期管理器
│   │   ├── manager.go        #   Manager 核心：register/unregister/StopAll
│   │   ├── caddy.go          #   StartCaddy / StopCaddy / ReloadCaddy
│   │   ├── mariadb.go        #   StartMariaDB / StopMariaDB
│   │   ├── php.go            #   StartPHPCGI / StopPHPCGI (多進程負載均衡)
│   │   ├── mailpit.go        #   StartMailpit / StopMailpit
│   │   └── runtime.go        #   StartRuntime (Node.js/Bun/Python/Go/Custom 進程啟停)
│   ├── detect/               # 專案類型自動偵測器
│   │   └── laravel.go        #   DetectLaravel(): 信心分數制判定
│   ├── preset/               # 專案類型 Preset 系統 (框架偵測與啟動指令範本)
│   ├── hosts/                # Windows Hosts 檔管理
│   │   └── hosts.go         #   CheckHosts / UpdateHosts / BackupHosts
│   ├── port/                 # Port 佔用檢測工具
│   ├── i18n/                 # 本地化翻譯系統
│   │   └── i18n.go          #   i18n.T() 翻譯接口與字典維護
│   └── resource/             # 硬體資源監控 (CPU/RAM/Stack)
│
├── frontend/                # 前端 React + TSX 專案 (Wails GUI 面板)
│   ├── src/
│   │   ├── wailsjs/         # Wails 自動產生的 Go 函式綁定與 JS 映射檔 (勿手動修改)
│   │   ├── components/      # 共享的通用 UI 元件 (Button, Input, Drawer 等)
│   │   ├── layouts/         # 頁面主體佈局 (Sidebar, Topbar, AppShell)
│   │   ├── pages/           # 各分頁組件 (Dashboard, Projects, DBExplorer, Settings)
│   │   ├── stores/          # Zustand 狀態管理器 (serviceStore, projectStore 等)
│   │   └── main.tsx         # 前端啟動進入點
│   ├── tailwind.config.js   # Tailwind CSS 變數與 HSL 設計標記設定
│   └── package.json         # 前端 npm 依賴定義
│
├── conf/                     # 配置文件中心
│   ├── dependencies.json    # ★ 自動下載的依賴版本與下載網址配置
│   ├── Caddyfile            # Caddy 進入點 (import snippets & sites)
│   ├── my.ini               # MariaDB 啟動配置
│   ├── snippets/            # Caddy 共用配置片段
│   │   ├── common.caddy     #   共用 headers、日誌、IP 白名單
│   │   └── php-upstream.caddy # ★ 自動生成的 PHP 負載均衡定義
│   ├── sites/               # ★ 自動生成的專案站點配置
│   └── ssl/                 # SSL 憑證 (*.crt, *.key)
│
├── bin/                     # 服務二進制檔 (使用者自備或透過 UI 下載器取得)
│   ├── caddy/               # caddy-x.y.z/caddy.exe
│   ├── mariadb/             # mariadb-x.y.z/bin/mariadbd.exe
│   └── php/                 # php-x.y.z/php-cgi.exe
│
├── data/                    # 持久化資料
│   ├── mariadb/             # MariaDB 預設 Data 目錄
│   └── backup/hosts/        # Hosts 備份檔
│
├── logs/                    # 日誌輸出 (依日期分檔)
├── www/                     # 預設網頁專案根目錄
├── legacy_fyne/             # 已歸檔的舊 Go Fyne 程式碼 (歸檔參考，不參與編譯)
└── go.mod / go.sum         # Go 模組依賴管理
```

### 3.1 目錄職責對照表

| 目錄/檔案 | 職責 |
|-----------|------|
| `main.go` | 初始化 Wails 應用程式設定，啟動視窗並加載前端網頁 |
| `app.go` | 接管 Wails 視窗生命週期 (Startup / DomReady / Shutdown)，管理背景協程並發送系統監控數據到前端 |
| `bridge.go` | 實現主要綁定的 Go-RPC 方法，前端可直接非同步呼叫此處的 API 方法 |
| `downloader_bridge.go`| 下載依賴管理器的 RPC 方法與進度回傳 |
| `frontend/` | UI 的 React 網頁應用程式，與 Go 核心進行完全的 IPC 資料通訊 |
| `internal/process` | 管理所有子進程 (Caddy/MariaDB/PHP-CGI/Mailpit/Runtime) 的完整生命週期 |
| `internal/i18n` | 處理 GUI 字串多國語言翻譯 |
| `conf/dependencies.json`| 外部依賴設定，讓下載與版本比對與程式邏輯解耦 |

---

## 4. 核心模組說明

### 4.1 `internal/scanner` — 版本掃描器

掃描 `bin/` 目錄，自動偵測已安裝服務版本並導出，PHP 連接埠計算公式如下。

### 4.2 `internal/process` — 進程生命週期管理

進程管理器是 WinCMP 的後端核心，管理 Caddy、MariaDB、PHP-CGI、Mailpit 以及以特定開發環境 (Runtime) 運行的子專案 (如 Node.js 專案)。

* **PHP 進程**：每個版本預設啟動 3 個（或自定義數量）的 `php-cgi` 進程。進程管理器以 Slice 儲存，以便在關閉時逐一 Kill。
* **Runtime 進程**：支援以 Background (不顯示視窗，日誌串流引導至 Runtime Logs) 或 Terminal (開啟 Windows cmd 視窗) 模式啟動。

### 4.3 `internal/port` — Port 計算與分配

**PHP Port 命名規則**：採用 **`3<主版本><次版本><序號00~99>`** 的規則分配 Port，確保不同 PHP 版本的 FastCGI 進程能夠並行不衝突。

| PHP 版本 | 進程數 | Port 範圍 | 計算公式 |
|----------|--------|-----------|----------|
| 7.3 | 3 (預設) | 37300, 37301, 37302 | `37000 + 7*100 + 3*10 + [00..02]` |
| 8.2 | 8 (設定) | 38200~38207 | `38000 + 8*100 + 2*10 + [00..07]` |

---

## 5. 前後端通訊機制

Wails v2 提供了兩種前後端通訊機制，WinCMP 充分利用了這兩者以達到極佳的流暢度：

### 5.1 Go-to-JS Binding (前後端非同步 RPC)
* 凡是在 `bridge.go` 中，屬於 `*Bridge` 結構體的首字母大寫方法（如 `GetConfig()`, `SaveConfig()` 等），Wails 在建置時會自動生成對應的前端 TypeScript SDK。
* 前端只需透過 `import { GetConfig } from "../wailsjs/go/main/Bridge"` 即可發起非同步 Promise 呼叫。

### 5.2 背景事件推送 (Events Mechanism)
* 用於單向即時資料推送（如系統 CPU/RAM 資源佔用率、各服務實時日誌）。
* **Go 端推送**：
  使用 `runtime.EventsEmit(ctx, "event_name", data)`。
* **React 端接收**：
  在 `useEffect` 中使用 `EventsOn("event_name", (data) => {})` 監聽，並更新 Zustand Store，藉此推動 UI 的渲染。

---

## 6. 依賴關係圖

```
┌────────────────────────────────────────────────────────┐
│                      frontend (UI)                     │
│    React Components  ──>  Zustand Stores (State)       │
└───────────────────────────┬────────────────────────────┘
                            │ (Wails RPC / IPC Events)
┌───────────────────────────▼────────────────────────────┐
│                    main.go / app.go                    │
│   (Wails Window Lifecycle & IPC Event Dispatcher)      │
└────────────┬──────────────┬──────────────┬─────────────┘
             │              │              │
       ┌─────▼─────┐  ┌─────▼─────┐  ┌─────▼─────┐
       │ bridge.go │  │ downloader│  │  i18n.go  │
       │ (RPC API) │  │_bridge.go │  │(多國語言) │
       └─────┬─────┘  └─────┬─────┘  └───────────┘
             │              │
       ┌─────▼───────────────────────▼─────┐
       │          internal/process         │
       │       (子進程生命週期管理器)      │
       └─────┬──────────────┬──────────────┘
             │              │
       ┌─────▼─────┐  ┌─────▼─────┐
       │  Caddy    │  │  MariaDB  │
       │ (Web Srv) │  │ (Database)│
       └───────────┘  └───────────┘
```

---

## 7. GUI 佈局架構

重構為 Wails React 網頁視窗後的現代化三欄式高密度介面：

```
┌────────────────────────────────────────────────────────────────────────┐
│  WinCMP Local Dev Panel v2.0.0                      ● Go 核心已連線   │
├──────────────┬─────────────────────────────────────────────────────────┤
│ ❖ WinCMP     │  🔍 搜尋專案或設定... (Ctrl+K)                          │
│   Local Dev  ├─────────────────────────────────────────────────────────┤
│              │  [ Dashboard / 儀表板 ]                                 │
│ 🔘 儀表板    │  ┌──────────────────┐ ┌──────────────────┐             │
│ 📂 專案管理  │  │ 🖥️ CPU 佔用      │ │ 💾 RAM 記憶體    │             │
│ 🗄️ 資料庫瀏覽│  │ [████░░░░░░] 35% │ │ [██████░░░░] 60% │             │
│ ⚙️ 系統設定  │  └──────────────────┘ └──────────────────┘             │
│              │  ┌────────────────────────────────────────────────────┐ │
│              │  │ 服務健康狀態                                        │ │
│              │  │ 🟢 Caddy v2.11   Port: 80, 443    [ 重啟 ] [ 停止 ] │ │
│              │  │ 🟢 MariaDB 11.4  Port: 3306       [ 重啟 ] [ 停止 ] │ │
│              │  │ 🔴 PHP-CGI 8.2   Port: 38200      [ 啟動 ] [ 設定 ] │ │
│              │  └────────────────────────────────────────────────────┘ │
├──────────────┼─────────────────────────────────────────────────────────┤
│ 📊 系統監控  │  ▲ [收起 Logs 控制台]                                   │
│ CPU: 12.5%   ├─────────────────────────────────────────────────────────┤
│ [██░░░░░░]  │  $ [14:32:01] [INFO] Caddy process started on port 80   │
│ RAM: 512 MB  │  $ [14:32:02] [INFO] MariaDB database system is ready   │
│ [█████░░░]  │  $ [14:32:05] [ERROR] PHP-CGI execution failed (port)   │
└──────────────┴─────────────────────────────────────────────────────────┘
```

---

> **附註**：本文件將隨專案迭代持續更新。如有架構變更，請先更新此文件再進行實作。
