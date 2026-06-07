# WinCMP 🚀

![Go Version](https://img.shields.io/badge/Go-1.26.2+-00ADD8?style=for-the-badge&logo=go)
![Wails Version](https://img.shields.io/badge/Wails-v2.12.0-red?style=for-the-badge&logo=wails)
![React Version](https://img.shields.io/badge/React-v18-blue?style=for-the-badge&logo=react)
![Platform](https://img.shields.io/badge/Platform-Windows_11-0078D6?style=for-the-badge&logo=windows)
![License](https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge)

**WinCMP** 是一個專為 Windows 設計的現代化、可攜式本機開發環境控制面板。
名稱取自 **Win**dows + **C**addy + **M**ariaDB + **P**HP。

受到 XAMPP 和 Laragon 的啟發，WinCMP 旨在提供一個更輕量、**免安裝 (Portable)**，且**基本不需要管理員權限**（僅在寫入 Hosts 檔案時需要提權）的開發解決方案。透過 Go 語言核心與 Wails v2 框架打造，前端使用 React 18 技術棧，具備極佳的視覺美感、極低的資源佔用與極快的啟動速度。

---

## ✨ 核心特色

- 🪶 **極致輕量**：採用 Go + Wails 靜態編譯，使用系統自帶的 Web 渲染引擎 (WebView2)，無須 Electron 依賴，體積小巧。
- 🛡️ **核心服務免管理員權限**：完全支援在受限環境下運作，不修改系統環境變數，不寫入登錄檔。*（註：自動修改 Windows `hosts` 檔以設定自訂本地網域為選用功能，啟用時需要 UAC 管理員提權）*。
- 🎨 **現代化 UI/UX**：全新設計的 Dark Professional (深色專業級) 介面，支援流暢的側邊導覽、即時狀態監控與微互動。
- 🔄 **PHP 多進程負載均衡**：利用 Caddy 的 Upstream 機制，每個 PHP 版本啟動多個 FastCGI 進程進行分流。
- 📂 **自動化專案管理**：可視化管理 Laravel、Next.js、Nuxt、Astro、Vite、Python、Go 等專案，自動偵測框架並生成配置。
- 🚀 **Runtime 多環境運行**：支援 Node.js、Bun、Python、Go (Air/Run)、Custom 等多種開發環境，可選 Background 或 Terminal 模式啟動。
- 💻 **內建專案互動終端**：結合 Windows ConPTY 與 `xterm.js` 實作 Drawer 側邊抽屜終端，預設路徑為專案根目錄，支援 PowerShell、CMD、Git Bash、WSL 切換與互動指令。
- 📜 **隔離環境 (Isolation)**：啟動子進程時動態注入 `PATH`，確保 PHP 及其擴展運行在正確的 binaries 環境中。

---

## 📁 項目架構與目錄規限

為了達成「隨插即用」的特性，WinCMP 嚴格遵守以下目錄結構：

```text
wincmp/
├── main.go                  # 應用程式進入點：初始化與啟動 Wails
├── app.go                   # Wails 生命週期管理器 (startup, shutdown) 及核心日誌/監控推送
├── bridge.go                # Wails 與 Go 端的主 Binding API (前後端 RPC 接口)
├── downloader_bridge.go     # Wails 下載管理器 Binding 接口
├── wincmp.json              # 專案與全域設定檔
├── conf/                    # 配置文件中心
│   ├── ssl/                 # SSL 憑證 (crt/key)
│   ├── snippets/            # Caddy 共用配置片段
│   ├── sites/               # 動態生成的專案 Caddyfile
│   ├── Caddyfile            # Caddy 進入點 (Import snippets & sites)
│   └── my.ini               # MariaDB 啟動設定
├── bin/                     # 二進制執行檔目錄 (自備或自動下載)
│   ├── caddy/               # caddy-x.xx.x/caddy.exe
│   ├── mariadb/             # mariadb-x.x.x/bin/mariadbd.exe
│   ├── php/                 # php-x.x.x/php-cgi.exe
│   ├── node/                # node-x.x.x/npm.cmd
│   ├── bun/                 # bun-x.x.x/bun.exe
│   ├── composer/            # composer-x.x.x/composer.bat
│   ├── heidisql/            # heidisql-x.xx/heidisql.exe
│   └── mailpit/             # mailpit-x.xx.x/mailpit.exe
├── data/                    # 資料存儲區
│   └── mariadb/             # MariaDB 預設 Data 目錄
├── logs/                    # 服務執行日誌 (依日期分類)
├── www/                     # 預設網頁專案根目錄
├── internal/                # 核心代碼邏輯 (不包含 GUI 邏輯)
│   ├── config/              # JSON 設定讀寫
│   ├── scanner/             # Bin 目錄動態版本掃描
│   ├── process/             # 子進程生命週期管理 (Manager)
│   ├── detect/              # Laravel 專案偵測 (信心分數制)
│   ├── preset/              # 專案類型 Preset 系統 (框架偵測/指令模板)
│   ├── hosts/               # Windows Hosts 檔管理
│   ├── port/                # Port 佔用檢測
│   ├── resource/            # 資源監控 (CPU/RAM/Stack)
│   ├── crypto/              # MariaDB 密碼加密
│   └── singleinstance/      # 單實例鎖 + 視窗帶到前景
└── frontend/                # 前端 React + TSX 專案
    ├── src/                 # 前端源碼 (Dashboard, Projects, DBExplorer 等)
    └── tailwind.config.js   # Tailwind 樣式變數設定
```

---

## 🛠️ 技術深度與運作邏輯

### 1. PHP 進程管理與 Port 映射
WinCMP 採用 **3-版本-序號** 的規則來分配服務端口，確保不同版本的 PHP 可以同時並行且互不干擾：
- **命名規則**：`3<主版本><次版本><序號00-99>`
  - PHP 7.3 → `37300`, `37301`, `37302`
  - PHP 8.2 → `38200`, `38201`, `38202`
- **負載均衡**：每開啟一個版本，預設啟動 3 個 `php-cgi` 進程，並在 Caddyfile 中定義 `php_fastcgi 127.0.0.1:38200 127.0.0.1:38201 ...` 實現自動分流。

### 2. Caddy 動態配置生成
當用戶在 UI 調整專案設定時：
1. 更新 `conf/wincmp.json`。
2. Go 程式重寫 `conf/sites/{project}.caddy`。
3. 執行 `caddy reload` 實現零停機更新。

### 3. 環境變數動態注入
為避免修改系統全域 PATH，WinCMP 在透過 `os/exec` 啟動子程序（如 PHP）時，會將對應的二進制目錄動態加入 `cmd.Env`，確保子程序能找到正確的 DLL 或相依組件。

---

## 🚀 開發與編譯環境

### 1. 前置需求
- [Go 1.26.2+](https://go.dev/dl/)
- [Wails CLI](https://wails.io/zh-Hans/docs/gettingstarted/installation/)：請確認系統已安裝 Wails v2。若未安裝，可使用 `go install github.com/wailsapp/wails/v2/cmd/wails@latest` 安裝。
- C 編譯器：MinGW-w64 (WinLibs) ── 用於 Wails 內部/底層對 Windows API 依賴的編譯，請確保 `gcc -v` 可正常執行。
- [Node.js](https://nodejs.org/)：Node.js 18+ (用於前端開發與打包)。

### 2. 開發熱重載指令 (Hot Reload)
```cmd
# 啟動 Wails 開發模式 (Go 後端與 React 前端同步熱重載)
wails dev
```

### 3. 建置編譯指令
```cmd
# 後端/前端依賴整理
go mod tidy
cd frontend && npm install && cd ..

# 開發/偵錯建置 (含除錯控制台與調試工具)
wails build -debug

# 正式發布編譯 (無視窗主控台，編譯後產出 wincmp.exe)
wails build -clean

# 壓縮並移除 symbols 的正式發布 (適用於極致體積優化)
wails build -clean -ldflags "-s -w"

# 自動化打包建置：透過 Go 的 -ldflags 動態注入版本號 (例如 v3.1.0)
wails build -ldflags "-X main.AppVersion=v3.1.0"
```

---

## 🗺️ 開發路線圖 (Roadmap)

### ✅ 已完成 (Completed)
- [x] 現代化 UI 原型與專案管理界面 (重構為 Wails + React 18)。
- [x] 多分頁系統日誌與旋轉日誌機制。
- [x] MariaDB 掃描與資料庫檢視器。
- [x] Caddy 多進程 PHP 負載均衡邏輯。
- [x] **Windows 系統匣 (System Tray)** 最小化支援。
- [x] **自動啟動上次關閉時的服務** (狀態記錄於 `wincmp.json`)。
- [x] **服務運行時間計時** (Caddy, MariaDB, PHP 獨立統計)。
- [x] **Laravel 專案自動偵測** (信心分數制，自動導向 `public/`)。
- [x] **Port 佔用檢查** (啟動前自動檢測，減少競爭狀態)。
- [x] **Hosts 本地網域自動管理** (UAC 權限提升後自動同步)。
- [x] **深色/淺色模式切換** (結合 Tailwind CSS)。
- [x] **Runtime 多環境運行** (Node.js, Bun, Python, Go Air/Run, Custom)。
- [x] **Preset 自動偵測** (Next.js, Nuxt, Astro, Vite, Django, FastAPI, Flask, PocketBase, Go API)。
- [x] **Runtime 雙模式啟動** (Background / Terminal)。
- [x] **舊版 Node.js 專案自動遷移** (node_port → runtime_port 等)。
- [x] **Mailpit 郵件測試服務整合** (Dashboard 啟停管理與設定對話框)。
- [x] **專案內建互動式終端** (結合 Windows ConPTY 與 `xterm.js`，實作點擊滑出 Drawer 互動終端，支援多種 Shell 設定)。

### ⏳ 計畫中 (Planned)
> **💡 關於詳細的開發規劃、技術分析與實作順序，請參閱完整的 [開發任務清單 (Develop Task List)](doc/develop_task_list.md)。**

- **系統深度整合**：Windows 開機自動啟動 (`HKCU\Run`)、一鍵設定 Windows 系統路徑環境變數 (`Path`)。
- **開發工具鏈**：內建 Composer 支援 (免安裝 `composer.phar`)、PHP 進程 Watchdog 自動重啟。
- **進階服務管理**：整合 HeidiSQL 預覽與快速連線、服務執行檔 (Caddy/PHP/MariaDB) 多版本自動下載器。

## 📄 授權條款

本項目基於 [MIT License](https://opensource.org/license/mit/) 授權。
歡迎提交 PR 或 Issue 與我交流！