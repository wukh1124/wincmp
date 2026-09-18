# 依賴元件說明

WinCMP 依賴的二進位檔案目錄定義於 [`conf/dependencies.json`](../conf/dependencies.json)（版本、下載 URL、SHA-256）。  
程式內建**依賴下載器**會依該檔下載並解壓到 `bin/`；本文件說明預設目錄、手動擺放方式與注意事項。

---

## 預設目錄（與 dependencies.json 對應）

下列版本以目前 `conf/dependencies.json` 為準；更新目錄後請同步調整該 JSON。

| Key | 元件 | 目前版本 | 安裝位置（相對於 `bin/`） |
|-----|------|----------|---------------------------|
| `caddy` | Caddy | 2.11.4 | `caddy/caddy-2.11.4/caddy.exe` |
| `mariadb` | MariaDB | 11.4.10 | `mariadb/mariadb-11.4.10/bin/mariadbd.exe` |
| `php73` | PHP 7.3 | 7.3.33 | `php/php-<zip 資料夾名>/php-cgi.exe` |
| `php82` | PHP 8.2 | 8.2.33 | 同上（例如 `php/php-8.2.33-nts-Win32-vs16-x64/`） |
| `php83` | PHP 8.3 | 8.3.33 | 同上 |
| `php84` | PHP 8.4 | 8.4.25 | 同上（vs17） |
| `php_redis_82` 等 | PHP Redis 擴充 | 6.3.0 | 解壓後的 `php_redis.dll` → `php/php-*/ext/` |
| `redis` | Redis | 5.0.14.1 | `redis/redis-5.0.14.1/redis-server.exe` |
| `mailpit` | Mailpit | 1.31.1 | `mailpit/mailpit-1.31.1/mailpit.exe` |
| `node` | Node.js | 24.21.0 | `node/node-24.21.0/npm.cmd` |
| `composer` | Composer | 2.10.3 | `composer/composer-2.10.3/composer.phar`（並產生 `composer.bat`） |
| `heidisql` | HeidiSQL | 12.21 | `heidisql/heidisql-12.21/heidisql.exe` |
| `cacert` | CA 憑證包 | 2026.08.13 | 由下載流程使用（非 `bin/` 服務） |

### 目錄範例

```text
wincmp/bin/
├── caddy/caddy-2.11.4/caddy.exe
├── mariadb/mariadb-11.4.10/bin/mariadbd.exe
├── php/
│   ├── php-8.3.33-nts-Win32-vs16-x64/
│   │   ├── php-cgi.exe
│   │   └── ext/php_redis.dll
│   └── php-8.2.33-nts-Win32-vs16-x64/
├── redis/redis-5.0.14.1/redis-server.exe
├── mailpit/mailpit-1.31.1/mailpit.exe
├── node/node-24.21.0/npm.cmd
├── composer/composer-2.10.3/composer.phar
└── heidisql/heidisql-12.21/heidisql.exe
```

### 相容掃描

掃描器（`internal/scanner`）也接受部分較寬的擺法：

- Caddy：`bin/caddy/caddy.exe`（無版本資料夾）
- Redis：`bin/redis/redis-server.exe`（無版本資料夾）

建議仍使用「版本資料夾」格式，便於多版本並存與下載器管理。

### 非 bin 元件

| 元件 | 說明 |
|------|------|
| **Bun** | 不在 `dependencies.json`。若要當 Runtime，自行放到 `bin/bun/bun-x.x.x/bun.exe`。 |
| **Python / Go** | 使用系統 PATH，不需放 `bin/`。 |

---

## 手動下載建議

若不用內建下載器，可至官方來源取得 **Windows x64** 版本，再放到上表路徑。

| 元件 | 來源 | 建議 |
|------|------|------|
| PHP | [windows.php.net](https://windows.php.net/downloads/releases/) | **NTS + x64**；FastCGI 使用 `php-cgi.exe`。舊版見 [Archives](https://windows.php.net/downloads/releases/archives/) |
| Caddy | [caddyserver.com](https://caddyserver.com/download) / [GitHub](https://github.com/caddyserver/caddy/releases) | Windows amd64 |
| MariaDB | [MariaDB](https://mariadb.org/download/) | **ZIP 免安裝版**（攜帶式）；勿用系統 MSI 裝到全局路徑 |
| Redis | [tporadowski/redis](https://github.com/tporadowski/redis/releases)（Win 移植版） | 目錄內需有 `redis-server.exe` |
| PHP Redis | [PECL redis Windows](https://pecl.php.net/package/redis) | DLL 版本須對應 PHP 主次版本與 TS/NTS |
| Mailpit | [axllent/mailpit](https://github.com/axllent/mailpit/releases) | Windows amd64 |
| Node.js | [nodejs.org](https://nodejs.org/dist/) | ZIP x64；目錄內需有 `npm.cmd` |
| Composer | [getcomposer.org](https://getcomposer.org/download/) | 下載 **`composer.phar`** 放進版本資料夾；**不要**用 Composer-Setup.exe 做系統全域安裝 |
| HeidiSQL | [heidisql.com](https://www.heidisql.com/download.php) | Portable 64-bit |
| Bun | [bun.sh](https://bun.sh/downloads) | Windows x64 zip |

### 手動放置注意

1. 路徑深度需與掃描規則一致（見上表）。
2. PHP 資料夾名稱需以 `php-` 開頭，並含可辨識的版本資訊。
3. Composer 需在 `composer-<版本>/` 內，且掃描會找 `composer.bat`（下載器會自動產生；手動放置時請一併提供 bat 或包一層）。
4. 更新 `conf/dependencies.json` 時，請同時更新 `version`、`url`、`sha256`，避免下載器校驗失敗。

---

## 相關文件

- 使用者安裝說明：[`packaging/wincmp/readme_zh.md`](../packaging/wincmp/readme_zh.md)
- Composer 指令細節：[`docs/composer_command.md`](composer_command.md)
