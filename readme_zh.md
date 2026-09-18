# WinCMP

![Go Version](https://img.shields.io/badge/Go-1.26.2+-00ADD8?style=for-the-badge&logo=go)
![Wails Version](https://img.shields.io/badge/Wails-v2.12.0-red?style=for-the-badge&logo=wails)
![React Version](https://img.shields.io/badge/React-v18-blue?style=for-the-badge&logo=react)
![Platform](https://img.shields.io/badge/Platform-Windows%2010%2F11-0078D6?style=for-the-badge&logo=windows)
![License](https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge)

**WinCMP** 是專為 Windows 設計的可攜式本地開發控制面板。
名稱來自 **Win**dows + **C**addy + **M**ariaDB + **P**HP。

受 XAMPP、Laragon 啟發，但更輕：免安裝、不污染系統 PATH，核心服務可在一般用戶權限下執行（Hosts 同步為選用，需 UAC）。以 Go + Wails v2 與 React 18 打造。

**下載：** [GitHub Releases](https://github.com/wukh1124/wincmp/releases/latest)

---

## 預覽

![WinCMP Dashboard](screenshot/sketch/dashboard.png)

---

## 特色

- **輕量單檔** — Go + Wails，使用系統 WebView2（非 Electron）。面板閒置記憶體約 150–250MB。
- **綠色可攜** — 遷移只需 `conf/wincmp.json` + `/bin`。
- **核心服務免 Admin** — Caddy、PHP-CGI、MariaDB、Mailpit、Redis 在用戶空間執行。*（Hosts 同步與部分依賴下載可能需 UAC。）*
- **現代化 UI** — 深色／淺色主題、即時服務狀態、專案互動終端。
- **PHP 多進程** — Caddy upstream 負載均衡；每版預設 3 個 FastCGI 進程（Dashboard 可調）。Port 規則 `3<主><次><序號>`。
- **框架 Preset** — 自動偵測 Laravel、Next.js、Nuxt、Astro、Vite、Django/FastAPI/Flask、PocketBase、Go API 等。
- **多 Runtime** — Node.js、Bun、Python、Go、Custom；Background / Terminal 雙模式。
- **內建終端** — ConPTY + xterm.js Drawer（PowerShell / CMD / Git Bash / WSL），支援 TAB 補齊。
- **依賴下載器** — 可下載 Caddy、PHP、MariaDB、Mailpit、Redis、Node.js、Bun、Composer、HeidiSQL 至 `/bin`。
- **資料庫工具** — 內建 DB Explorer，可一鍵以 HeidiSQL 開啟。
- **環境隔離** — 只對子進程注入 `PATH`，不改系統全域設定。
- **熱重載** — 專案設定寫入 `conf/sites/` 後 reload Caddy，面板本身不必重啟。

---

## 目錄結構

```text
wincmp/
├── main.go / app.go / bridge.go
├── downloader_bridge.go
├── conf/
│   ├── wincmp.json          # 主設定（專案 + 全域）
│   ├── Caddyfile
│   ├── my.ini
│   ├── dependencies.json    # 下載器版本目錄
│   ├── snippets/  sites/  ssl/
├── bin/                     # 服務執行檔（自備或下載）
│   ├── caddy/  mariadb/  php/
│   ├── mailpit/  redis/
│   ├── node/  bun/  composer/  heidisql/
├── data/mariadb/
├── logs/
├── www/
├── internal/                # config、scanner、process、detect、preset、hosts …
└── frontend/                # React + TypeScript
```

應用實際讀寫的設定路徑：**`conf/wincmp.json`**。

---

## 架構重點

**PHP Port** — `3<主版本><次版本><序號>`：
PHP 7.3 → `37300+`；PHP 8.2 → `38200+`。每版預設 3 個進程，可調整。

**設定生效** — UI 更新 `conf/wincmp.json` → 重寫 `conf/sites/{project}.caddy` → `caddy reload`。

**PATH 隔離** — 僅在子進程環境前置二進位目錄，不改寫系統 PATH。

---

## 開發

### 環境需求

- Go 1.26.2+
- [Wails v2](https://wails.io/docs/gettingstarted/installation/)
- MinGW-w64（WinLibs），可執行 `gcc -v`
- Node.js 18+

### 指令

```cmd
wails dev

go mod tidy
cd frontend && npm install && cd ..
wails build -clean
wails build -clean -ldflags "-s -w"
wails build -ldflags "-X main.AppVersion=v2.1.0"

node scripts/generate-release-json.js
```

終端用戶安裝說明見 [`packaging/wincmp/readme_zh.md`](packaging/wincmp/readme_zh.md)。

---

## Roadmap

### 已完成

- Wails + React UI、多分頁日誌、系統匣、還原上次服務
- MariaDB 掃描、DB Explorer + HeidiSQL、Hosts 同步與備份
- PHP 多進程、Runtime 多環境、框架 Preset
- 專案終端（ConPTY + xterm.js）、Mailpit、Redis + PHP Redis 擴充
- 依賴下載器（Caddy/PHP/MariaDB/Node/Bun/Composer/HeidiSQL/Mailpit/Redis）
- PHP OPcache 自動優化、專案拖曳排序

### 計畫中

- PHP 進程 Watchdog／自動恢復

---

## Credits

- [Go](https://go.dev/)、[Wails v2](https://wails.io/)、[React 18](https://react.dev/)
- [Windows ConPTY](https://learn.microsoft.com/windows/console/pseudoconsole)、[xterm.js](https://xtermjs.org/)
- 管理的服務：Caddy、MariaDB、PHP、Mailpit、Redis
- 主題靈感：[Open Design](https://github.com/nexu-io/open-design)

## 授權

[MIT](LICENSE)
