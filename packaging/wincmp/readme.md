# WinCMP 🚀

![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=for-the-badge&logo=go)
![Fyne Version](https://img.shields.io/badge/Fyne-v2.7-blue?style=for-the-badge)
![Platform](https://img.shields.io/badge/Platform-Windows_11-0078D6?style=for-the-badge&logo=windows)
![License](https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge)

**WinCMP** 是一個專為 Windows 設計的現代化、可攜式本機開發環境控制面板。
名稱取自 **Win**dows + **C**addy + **M**ariaDB + **P**HP。

受到 XAMPP 和 Laragon 的啟發，WinCMP 提供一個更輕量、**免安裝 (Portable)**、且 **不需要管理員權限** 的 PHP 開發環境解決方案。

---

## ✨ 核心特色

- 🪶 **極致輕量**：採用 Go 語言開發，無須 Node.js 或 Electron，啟動快速，資源佔用低。
- 🛡️ **免管理員權限**：無需系統管理員權限即可運行，不修改系統環境變數，不寫入登錄檔。
- 🎨 **現代化 UI/UX**：內建深色/淺色模式，直覺的圖形化界面，即時監控服務狀態。
- 🔄 **PHP 多版本支援**：同時管理多個 PHP 版本，自動負載均衡提升效能。
- 📂 **專案管理**：可視化管理 Laravel、Vue 等專案，自動化設定 Caddy 配置。
- 📜 **完全可攜**：整個環境集中在一個資料夾，可隨身攜帶，隨插即用。

---

## 📥 快速開始

### 系統需求

- **作業系統**：Windows 10 / Windows 11（64位元）
- **硬碟空間**：至少 500 MB 可用空間
- **記憶體**：建議 4 GB 以上

### 安裝步驟

1. **下載 WinCMP**
   - 從 Releases 頁面下載最新版本的 `wincmp.zip`
   - 解壓縮至您想要的位置（例如：`D:\wincmp`）

2. **準備運行環境**
   WinCMP 需要以下元件才能運作：
   - **Caddy**：Web 伺服器
   - **MariaDB**：資料庫伺服器
   - **PHP**：PHP 運行環境（可選多版本）

3. **放置二進制檔案**
   將下載好的 Caddy、MariaDB、PHP 解壓縮並放置於對應目錄：
   ```
   wincmp/
   ├── bin/
   │   ├── caddy/          # Caddy 執行檔
   │   ├── mariadb/        # MariaDB 執行檔
   │   └── php/            # PHP 執行檔（可放多個版本）
   │       ├── php-7.4/
   │       ├── php-8.1/
   │       └── php-8.2/
   ```

4. **啟動 WinCMP**
   - 雙擊 `wincmp.exe` 啟動程式
   - 程式會自動掃描可用的服務版本

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
│   └── php/
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
3. 點擊「啟動」按鈕，服務狀態會即時更新

### 建立新專案

1. 在 WinCMP 主介面點擊「新增專案」
2. 設定專案名稱與根目錄路徑
3. 選擇要使用的 PHP 版本
4. 設定網域名稱（可選，預設為 `localhost`）
5. 點擊「建立」，WinCMP 會自動生成配置

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
├── php-7.4.33/
├── php-8.1.28/
└── php-8.2.19/
```
WinCMP 會自動識別所有可用的 PHP 版本。

### SSL 憑證

預設使用 Caddy 自動產生的本地憑證。如需使用自訂憑證，請將檔案放置於 `conf/ssl/` 目錄。

---

## 🛠️ 常見問題

**Q: 啟動時顯示「找不到服務執行檔」？**  
A: 請確認已將 Caddy、MariaDB、PHP 放置於正確的 `bin/` 子目錄中。

**Q: 端口被佔用怎麼辦？**  
A: 在設定中修改端口號，或關閉佔用端口的其他程式（如其他 XAMPP/WAMP）。

**Q: 資料庫連線失敗？**  
A: 確認 MariaDB 服務已啟動，並檢查 `conf/my.ini` 中的配置。

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

---

*感謝您使用 WinCMP！若有任何問題或建議，歡迎提交 Issue。*
