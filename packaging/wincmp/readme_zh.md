# WinCMP

![Platform](https://img.shields.io/badge/Platform-Windows%2010%2F11-0078D6?style=for-the-badge&logo=windows)
![License](https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge)

Windows 可攜式本地開發控制面板：**Caddy + MariaDB + PHP + Mailpit + Redis**，並支援專案 Runtime 與內建終端。免安裝，核心服務可在一般用戶權限下執行。

原始碼與 Issue：[github.com/wukh1124/wincmp](https://github.com/wukh1124/wincmp)

---

## 特色

- 一鍵啟停 Caddy、MariaDB、PHP-CGI、Mailpit、Redis
- PHP 多版本負載均衡（每版預設 3 個 FastCGI 進程，可調整）
- 專案 Preset（Laravel、Next.js、Nuxt、Astro、Vite、Django/FastAPI/Flask 等）
- Runtime：Node.js / Bun / Python / Go / Custom（Background 或 Terminal）
- 內建終端 Drawer、DB Explorer（HeidiSQL）
- 依賴一鍵下載至 `/bin`
- 完全可攜 — 帶走 `conf/` + `bin/` + `data/` 即可

---

## 快速開始

### 系統需求

- Windows 10 / 11（64 位元）
- 約 500MB 可用空間（若安裝多項服務／版本，建議 1GB+）
- 建議 4GB+ 記憶體

### 安裝

1. 至 [Releases](https://github.com/wukh1124/wincmp/releases/latest) 下載最新 **`wincmp-v*-win-x64.zip`**。
2. 解壓縮到任意目錄（例如 `D:\wincmp`）。
3. 執行 **`WinCMP_vX.Y.Z.exe`**。
4. 若缺少核心元件，使用程式內**依賴下載器**（建議），或自行放到 `bin/`。

### 手動 `bin/` 結構（可選）

```text
wincmp/bin/
├── caddy/          # caddy.exe
├── mariadb/        # 版本目錄內的 mariadbd.exe
├── php/            # php-x.x.x-nts-.../php-cgi.exe（可多版本）
├── mailpit/
├── redis/
├── node/  bun/  composer/  heidisql/   # 可選
```

請勿刪除 `conf/`、`data/`、`logs/`——設定與 MariaDB 資料都在這裡。

---

## 使用方式

### 服務

Dashboard 服務：**Caddy**（80/443）、**MariaDB**（3306）、**PHP**（選版本）、**Mailpit**（1025 / 8025）、**Redis**（6379）。

### 新增專案

1. 點 **新增專案**
2. 設定名稱與根目錄
3. 選擇專案類型 / Preset
4. 可自訂網域（預設 `local-{專案名}.test`)
5. **建立** — 自動產生 Caddy 設定

### 存取

- 本機：`http://localhost` 或專案網域
- 資料庫：面板 **Database** 頁，或 **Open in HeidiSQL**

### 系統匣

可在設定中開啟「關閉時最小化至系統匣」；匣圖示選單可快速啟停服務。

---

## 設定說明

- **連接埠** — 於 Settings 或 `conf/wincmp.json` 修改（Caddy 80/443、MariaDB 3306）。
- **PHP 版本** — `bin/php/` 下每版一個資料夾（如 `php-8.3.33-nts-...`）。
- **Runtime** — Node/Bun 掃描 `bin/`；Python/Go 使用系統 PATH；Custom 支援 `%PORT%`、`%HOST%`、`%PROJECT_DIR%`、`%BIN_DIR%`。
- **SSL** — 預設 Caddy 本地 CA；自訂憑證放 `conf/ssl/`。

---

## 常見問題

**找不到執行檔**  
用依賴下載器安裝 Caddy / MariaDB / PHP，或自行放到 `bin/`。

**連接埠被佔用**  
在設定改 Port，或關閉佔用程式（XAMPP/WAMP/IIS 等）。

**資料庫連線失敗**  
確認 MariaDB 已啟動，並檢查 `conf/my.ini`。如需外部 MySQL/MariaDB，可在 Settings 設定。

**Runtime 啟動失敗**  
Python/Go 請確認 PATH 可執行（`python -V`、`go version`）。Node/Bun 請確認在 `bin/node/` 或 `bin/bun/`。

**如何備份資料庫**  
複製 `data/mariadb/`。

**可以搬到別台電腦嗎**  
可以。複製整個資料夾，或至少 `conf/` + `bin/` + `data/`。

---

## 相關連結

- [Caddy](https://caddyserver.com/)
- [MariaDB](https://mariadb.org/)
- [PHP](https://www.php.net/)
- [Mailpit](https://mailpit.axllent.org/)
- [Redis](https://redis.io/)
- [Node.js](https://nodejs.org/) / [Bun](https://bun.sh/)

## 授權

[MIT](LICENSE)
