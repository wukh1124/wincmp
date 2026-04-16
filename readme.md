# WinCMP 🚀

![Go Version](https://img.shields.io/badge/Go-1.26.2+-00ADD8?style=for-the-badge&logo=go)
![Fyne Version](https://img.shields.io/badge/Fyne-v2.7.3-blue?style=for-the-badge)
![Platform](https://img.shields.io/badge/Platform-Windows_11-0078D6?style=for-the-badge&logo=windows)
![License](https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge)

**WinCMP** 是一個專為 Windows 設計的現代化、可攜式本機開發環境控制面板。
名稱取自 **Win**dows + **C**addy + **M**ariaDB + **P**HP。

受到 XAMPP 和 Laragon 的啟發，WinCMP 旨在提供一個更輕量、**免安裝 (Portable)**、且 **不需要管理員權限** 的開發解決方案。透過 Go 語言與 Fyne GUI 框架打造，具備極低的資源佔用與極快的啟動速度。

---

## ✨ 核心特色

- 🪶 **極致輕量**：採用 Go 靜態編譯，無須 Electron 依賴。
- 🛡️ **免管理員權限**：完全支援在受限環境下運作，不修改系統環境變數，不寫入登錄檔。
- 🎨 **現代化 UI/UX**：內建深色/淺色模式，提供流暢的側邊導覽與即時狀態監控。
- 🔄 **PHP 多進程負載均衡**：利用 Caddy 的 Upstream 機制，每個 PHP 版本啟動多個 FastCGI 進程進行分流。
- 📂 **自動化專案管理**：可視化管理 Laravel、Next.js、Nuxt、Astro、Vite、Python、Go 等專案，自動偵測框架並生成配置。
- 🚀 **Runtime 多環境運行**：支援 Node.js、Bun、Python、Go (Air/Run)、Custom 等多種開發環境，可選 Background 或 Terminal 模式啟動。
- 📜 **隔離環境 (Isolation)**：啟動子進程時動態注入 `PATH`，確保 PHP 及其擴展運行在正確的 binaries 環境中。

---

## 📁 項目架構與目錄規限

為了達成「隨插即用」的特性，WinCMP 嚴格遵守以下目錄結構：

```text
wincmp/
├── wincmp.exe               # Go 編譯的主程式
├── conf/                    # 配置文件中心
│   ├── ssl/                 # SSL 憑證 (crt/key)
│   ├── snippets/            # Caddy 共用配置片段
│   ├── sites/               # 動態生成的專案 Caddyfile
│   ├── wincmp.json          # WinCMP 全域與專案設定 (UI 資料來源)
│   ├── Caddyfile            # Caddy 進入點 (Import snippets & sites)
│   └── my.ini               # MariaDB 啟動設定
├── bin/                     # 二進制執行檔目錄 (自備或自動掃描)
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
├── internal/                # 核心代碼邏輯
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
├── ui_runtime.go            # Runtime Tab UI 定義
├── bundled_icon.go         # 應用圖示資源
└── bat/                     # 備份用的啟動腳本 (測試參考)
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

### 1. 系統需求
- [Go 1.26.2+](https://go.dev/dl/)
- C 語言編譯器 (用於 Fyne Cgo 依賴)

### 2. 免管理員權限編譯 (使用 WinLibs)
若無權限安裝 MSYS2：
1. 下載 [WinLibs MinGW-w64](https://winlibs.com/) Zip 版。
2. 解壓縮並將其中的 `bin/` 加入**使用者環境變數 (User PATH)**。
3. 確認 `gcc -v` 可執行。

### 3. 編譯指令
```cmd
# 初始化依賴
go mod tidy

# 一般編譯
go build -v -o wincmp.exe .

# 正式發布（無 CMD 視窗）
go build -v -o wincmp.exe -ldflags "-H windowsgui" .

# 發布 + 縮小體積
go build -ldflags "-H windowsgui -s -w" -o wincmp.exe .

# 使用 Fyne 打包 (包含圖示與資源)
fyne package -release

# 如打包失敗, 先嘗試清Cache
go clean -cache
```

---

## 🗺️ 開發路線圖 (Roadmap)

### ✅ 已完成 (Completed)
- [x] 現代化 UI 原型與專案管理界面。
- [x] 多分頁系統日誌與旋轉日誌機制。
- [x] MariaDB 掃描與資料庫檢視器。
- [x] Caddy 多進程 PHP 負載均衡邏輯。
- [x] **Windows 系統匣 (System Tray)** 最小化支援。
- [x] **自動啟動上次關閉時的服務** (狀態記錄於 `wincmp.json`)。
- [x] **服務運行時間計時** (Caddy, MariaDB, PHP 獨立統計)。
- [x] **Laravel 專案自動偵測** (信心分數制，自動導向 `public/`)。
- [x] **Port 佔用檢查** (啟動前自動檢測，減少競爭狀態)。
- [x] **Hosts 本地網域自動管理** (UAC 權限提升後自動同步)。
- [x] **深色/淺色模式切換** (整合 Fyne 主題系統)。
- [x] **Runtime 多環境運行** (Node.js, Bun, Python, Go Air/Run, Custom)。
- [x] **Preset 自動偵測** (Next.js, Nuxt, Astro, Vite, Django, FastAPI, Flask, PocketBase, Go API)。
- [x] **Runtime 雙模式啟動** (Background / Terminal)。
- [x] **舊版 Node.js 專案自動遷移** (node_port → runtime_port 等)。
- [x] **Mailpit 郵件測試服務整合** (Dashboard 啟停管理與設定對話框)。

### ⏳ 計畫中 (Planned)
> **💡 關於詳細的開發規劃、技術分析與實作順序，請參閱完整的 [開發任務清單 (Develop Task List)](doc/develop_task_list.md)。**

- **系統深度整合**：Windows 開機自動啟動 (`HKCU\Run`)、一鍵設定 Windows 系統路徑環境變數 (`Path`)。
- **開發工具鏈**：內建 Composer 支援 (免安裝 `composer.phar`)、PHP 進程 Watchdog 自動重啟。
- **進階服務管理**：整合 HeidiSQL 預覽與快速連線、服務執行檔 (Caddy/PHP/MariaDB) 多版本自動下載器。

## 📄 授權條款

本項目基於 [MIT License](https://opensource.org/license/mit/) 授權。
歡迎提交 PR 或 Issue 與我交流！