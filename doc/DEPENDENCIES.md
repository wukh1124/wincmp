# 核心元件下載清單 (Core Components Download)

本文件提供 WinCMP 運行所需的核心二進位檔案下載連結與建議。

## 📥 元件清單

### 1. PHP (Windows 版)
- **連結**: [PHP Releases Archives](https://windows.php.net/downloads/releases/archives/)
- **建議**: 
    - 建議下載 **Non-Thread Safe (NTS)** 版本（若配合 FastCGI 使用）。
    - 確保下載 **x64** 版本以符合現代環境。

### 2. Caddy Server
- **連結**: [Caddy Download](https://caddyserver.com/download)
- **建議**: 
    - 選擇平台為 `Windows` 且架構為 `amd64`。
    - 若需額外插件（如 Cloudflare DNS），請在官網自定義編譯。

### 3. MariaDB
- **連結**: [MariaDB Download](https://mariadb.org/download/)
- **建議**: 
    - 建議使用 **MSI 安裝包** 以便於初始化，或使用 **ZIP** 免安裝版進行攜帶式配置。

### 4. Composer
- **連結**: [Composer Download](https://getcomposer.org/download/)
- **建議**: 
    - Windows 環境建議直接執行 `Composer-Setup.exe` 進行全域安裝。

### 5. HeidiSQL (資料庫管理)
- **連結**: [HeidiSQL Download](https://www.heidisql.com/download.php)
- **建議**: 
    - 輕量級的資料庫管理工具，建議下載安裝版或免安裝版備用。

---

## 📂 建議目錄結構 (Directory Structure)

下載後的二進位檔案應放置於專案根目錄的 `bin/` 資料夾下，並依照以下結構組織：

```text
wincmp/
└── bin/
    ├── caddy/
    │   └── caddy-2.11.1/
    │       └── caddy.exe
    ├── mariadb/
    │   └── mariadb-11.4.10-winx64/
    │       └── bin/
    │           └── mariadbd.exe
    └── php/
        ├── php-8.2.30-nts-Win32-vs16-x64/
        │   └── php-cgi.exe
        └── php-7.4.33-nts-Win32-vs16-x64/
            └── php-cgi.exe
```

> [!IMPORTANT]
> WinCMP 會自動掃描 `bin/` 目錄下的執行檔。請確保路徑深度與上述結構一致，以便掃描器正確認識版本號。
