# WinCMP 🚀

![Go Version](https://img.shields.io/badge/Go-1.26.2+-00ADD8?style=for-the-badge&logo=go)
![Wails Version](https://img.shields.io/badge/Wails-v2.12.0-red?style=for-the-badge&logo=wails)
![React Version](https://img.shields.io/badge/React-v18-blue?style=for-the-badge&logo=react)
![Platform](https://img.shields.io/badge/Platform-Windows_11-0078D6?style=for-the-badge&logo=windows)
![License](https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge)

**WinCMP** 是一個專為 Windows 設計的現代化、可攜式本機開發環境控制面板。
名稱取自 **Win**dows + **C**addy + **M**ariaDB + **P**HP。

受到 XAMPP 和 Laragon 的啟發，WinCMP 提供一個更輕量、**免安裝 (Portable)**、且 **不需要管理員權限** 的開發環境解決方案。基於 Go 語言核心與 Wails v2 框架打造，前端使用 React 18 技術棧，具備極佳的視覺美感、極低的資源佔用與極快的啟動速度。

---

## ✨ 核心特色

- 🪶 **輕量**：採用 Go 核心編譯，使用系統自帶的 Web 渲染引擎 (WebView2)，無須 Electron 依賴，啟動快速，資源佔用低。
- 🛡️ **免管理員權限**：無需系統管理員權限即可運行，不修改系統環境變數，不寫入登錄檔。*（註：自動修改 Windows `hosts` 檔為選用功能，啟用時需要 UAC 權限）*
- 🎨 **現代化 UI/UX**：內建深色/淺色模式，直覺的圖形化界面，即時監控服務狀態。
- 🔄 **PHP 多版本支援**：同時管理多個 PHP 版本，自動負載均衡提升效能。
- 🚀 **Runtime 多環境運行**：支援 Node.js、Bun、Python、Go (Air/Run)、Custom 等多種開發環境。
- 📂 **專案管理**：可視化管理 Laravel、Next.js、Nuxt、Astro、Vite、Python、Go 等專案，自動偵測框架並生成配置。
- 📜 **完全可攜**：整個環境集中在一個資料夾，可隨身攜帶，隨插即用。

---

## 📥 快速開始

### 系統需求

- **作業系統**：Windows 10 / Windows 11（64位元）
- **硬碟空間**：至少 500 MB 可用空間
- **記憶體**：建議 4 GB 以上

### 安裝步驟

1. **下載 WinCMP**
   - 從 Releases 頁面下載最新版本的 `wincmp.zip`（Light 版，僅包含本體程式）。
   - 解壓縮至您想要的位置（例如：`D:\wincmp`）。

2. **啟動 WinCMP 與依賴偵測**
   - 雙擊 `wincmp.exe` 啟動程式。
   - **依賴偵測提示**：啟動時，WinCMP 會自動掃描 `bin/` 目錄。如果缺少運行所需的核心元件（Caddy、PHP、MariaDB 等），程式會彈出**「依賴元件缺失提示」**。
   - 您可以直接在提示視窗中點擊**「自動下載與安裝」**，WinCMP 會自動從官方託管地址下載推薦的版本並配置好目錄；您也可以選擇手動準備元件。

3. **手動放置二進制檔案（備用 / 自訂版本）**
   如果您想使用自訂的版本，可以手動下載對應元件並解壓縮放置於以下目錄結構：
   ```
   wincmp/
   ├── bin/
   │   ├── caddy/          # Caddy 執行檔 (放置 caddy.exe)
   │   ├── mariadb/        # MariaDB 執行檔 (放置 mariadbd.exe 所在目錄)
   │   ├── php/            # PHP 執行檔（可放多個版本，如 php-8.3.28-nts-Win32...）
   │   │   ├── php-8.2/
   │   │   └── php-8.3/
   │   ├── node/           # Node.js 執行檔（可選）
   │   ├── bun/            # Bun 執行檔（可選）
   │   ├── composer/       # Composer 執行檔（可選）
   │   ├── heidisql/       # HeidiSQL 執行檔（可選）
   │   └── mailpit/        # Mailpit 執行檔（可選）
   ```


---

## 📁 目錄結構說明

```text
wincmp/
├── wincmp.exe               # 主程式
├── conf/                    # 配置文件
│   ├── ssl/                 # SSL 憑證
│   ├── snippets/            # Caddy 共用配置
│   ├── sites/               # 專案網站配置
│   ├── wincmp.json          # 程式設定檔
│   ├── Caddyfile            # Caddy 主配置
│   └── my.ini               # MariaDB 設定
├── bin/                     # 服務執行檔
│   ├── caddy/
│   ├── mariadb/
│   ├── php/
│   ├── node/                # Node.js（可選）
│   ├── bun/                 # Bun（可選）
│   ├── composer/            # Composer（可選）
│   ├── heidisql/            # HeidiSQL（可選）
│   └── mailpit/             # Mailpit（可選）
├── data/                    # 資料庫檔案
│   └── mariadb/
├── logs/                    # 執行日誌
└── www/                     # 預設網站根目錄
```

**注意**：請勿刪除 `conf/`、`data/`、`logs/` 目錄，這些目錄存放著您的設定與資料。

---

## 🚀 使用指南

### 啟動服務

1. 開啟 WinCMP 主程式
2. 在主介面選擇要啟動的服務：
   - **Caddy**：Web 伺服器（預設端口：80/443）
   - **MariaDB**：資料庫伺服器（預設端口：3306）
   - **PHP**：選擇要啟動的 PHP 版本
   - **Mailpit**：郵件測試服務（預設端口：8025/1025，可選）
3. 點擊「啟動」按鈕，服務狀態會即時更新

### 建立新專案

1. 在 WinCMP 主介面點擊「新增專案」
2. 設定專案名稱與根目錄路徑
3. 選擇專案類型（Laravel、Next.js、Nuxt、Astro、Vite、Python、Go API、PocketBase、Custom 等）
4. 設定網域名稱（可選，預設為 `local-{專案名}.test`）
5. 點擊「建立」，WinCMP 會自動偵測框架並生成配置

### 存取您的網站

- **本機存取**：`http://localhost`
- **指定專案**：根據您設定的域名存取
- **資料庫管理**：可透過 phpMyAdmin 或其他 MySQL 管理工具連線

### 系統匣功能

- 點擊右上角「最小化」可將程式縮小至系統匣
- 在系統匣圖示上按右鍵可快速啟動/停止服務
- 服務運行時間會顯示在介面上

---

## ⚙️ 進階設定

### 修改端口號

若預設端口被佔用，可於 `conf/wincmp.json` 中修改：
- Caddy HTTP 端口：預設 80
- Caddy HTTPS 端口：預設 443
- MariaDB 端口：預設 3306

### PHP 多版本配置

將不同版本的 PHP 放置於 `bin/php/` 下的獨立資料夾：
```
bin/php/
├── php-8.2.30/
├── php-8.3.28/
```
WinCMP 會自動識別所有可用的 PHP 版本，並僅保留每個 Minor 版本的最新 Patch 版本。

### Runtime 開發環境

WinCMP 的 Runtime Tab 支援多種開發環境：
- **Node.js / Bun**：放置於 `bin/node/` 或 `bin/bun/` 即可自動掃描
- **Python / Go**：使用系統 PATH 中的安裝，啟動時自動偵測版本
- **Custom**：自訂啟動指令，支援 `%PORT%`、`%HOST%`、`%PROJECT_DIR%`、`%BIN_DIR%` 佔位符

Runtime 支援 **Background**（背景執行，輸出至 Runtime Logs）和 **Terminal**（開啟獨立 CMD 視窗）兩種模式。

### SSL 憑證

預設使用 Caddy 自動產生的本地憑證。如需使用自訂憑證，請將檔案放置於 `conf/ssl/` 目錄。

---

## 🛠️ 常見問題

**Q: 啟動時顯示「找不到服務執行檔」？**  
A: 請確認已將 Caddy、MariaDB、PHP 放置於正確的 `bin/` 子目錄中。

**Q: 端口被佔用怎麼辦？**  
A: 在設定中修改端口號，或關閉佔用端口的其他程式（如其他 XAMPP/WAMP）。

**Q: 資料庫連線失敗？**  
A: 確認 MariaDB 服務已啟動，並檢查 `conf/my.ini` 中的配置。如使用外部 MariaDB/MySQL，請在 Settings 中啟用外部資料庫選項。

**Q: Runtime 啟動失敗？**
A: 若使用 Python/Go 類型，請確認已安裝至系統 PATH（`python -V`、`go version` 可執行）。若使用 Node.js/Bun 類型，請確認 `bin/node/` 或 `bin/bun/` 目錄中有對應的執行檔。

**Q: 如何備份資料庫？**  
A: `data/mariadb/` 目錄即為資料庫檔案所在，直接複製該目錄即可備份。

**Q: 可以攜帶到不同電腦使用嗎？**  
A: 可以！WinCMP 是完全可攜式的，只要複製整個資料夾到新電腦即可。

---

## 📄 授權條款

本項目基於 [MIT License](LICENSE) 授權。

---

## 💡 相關連結

- **Caddy**：https://caddyserver.com/
- **MariaDB**：https://mariadb.org/
- **PHP**：https://www.php.net/
- **Mailpit**：https://mailpit.axllent.org/
- **Bun**：https://bun.sh/
- **Node.js**：https://nodejs.org/

---

*感謝您使用 WinCMP！若有任何問題或建議，歡迎提交 Issue。*
