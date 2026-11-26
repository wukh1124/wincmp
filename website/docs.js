document.addEventListener('DOMContentLoaded', () => {
    // ==========================================================================
    // 1. UI 雙語字典（與首頁共用 localStorage: wincmp_lang / wincmp_theme）
    // ==========================================================================
    const uiStrings = {
        en: {
            docs_page_title: "WinCMP Documentation - Usage & Technical Reference",
            nav_home: "Home",
            nav_docs: "Docs",
            nav_changelog: "Changelog",
            docs_sidebar_title: "DOCUMENTATION",
            docs_mobile_menu: "Menu",
            docs_prev: "Previous",
            docs_next: "Next",
            group_basics: "Basics & Layout",
            group_services: "Services & Tools",
            group_runtime: "Runtime & Ops",
        },
        zh: {
            docs_page_title: "WinCMP 使用文檔 - 操作與技術參考",
            nav_home: "首頁",
            nav_docs: "使用文檔",
            nav_changelog: "更新日誌",
            docs_sidebar_title: "使用文檔",
            docs_mobile_menu: "目錄",
            docs_prev: "上一篇",
            docs_next: "下一篇",
            group_basics: "基礎與目錄",
            group_services: "服務與工具",
            group_runtime: "運行時與維運",
        }
    };

    // ==========================================================================
    // 2. 精選 9 篇開源技術文檔（不寫死依賴版號，降低發版維護成本）
    // ==========================================================================
    const docSections = [
        {
            id: "getting-started",
            group: "group_basics",
            title: { zh: "快速上手與目錄結構", en: "Quick Start & Layout" },
            content: {
                zh: `# 快速上手與目錄結構

WinCMP 是單檔免安裝的 Windows 本機開發控制面板。解壓縮後直接執行 \`WinCMP.exe\`，程式會在同層目錄自動建立運作所需資料夾。

## 目錄職責

\`\`\`text
WinCMP/
├── WinCMP.exe      # 主程式（Go + Wails，透過系統 WebView2 渲染）
├── bin/            # 所有服務與工具的獨立二進位檔案
├── conf/           # 軟體設定檔（wincmp.json）、Caddyfile、php.ini、my.ini
├── data/           # MariaDB 資料庫實體檔案（datadir）與執行期資料
└── logs/           # 各服務與主程式輸出日誌
\`\`\`

## 首次啟動流程

1. 開啟 **依賴下載器**（Dashboard 右上角）下載所需元件（如 Caddy、PHP、MariaDB 等）。
2. 到 **Projects** 新增專案並指定根目錄；WinCMP 會自動偵測專案框架與建議埠口。
3. 在 **Dashboard** 啟動對應核心服務，或於專案列啟動 Runtime。

> 所有核心服務皆限制在 \`WinCMP/\` 目錄內運作，不會寫入 Windows 登錄檔，也不會修改系統全域 \`PATH\`。`,
                en: `# Quick Start & Directory Layout

WinCMP is a portable single-executable local dev panel for Windows. Extract and run \`WinCMP.exe\`; working folders are created automatically beside the binary.

## Directory Layout

\`\`\`text
WinCMP/
├── WinCMP.exe      # Main app (Go + Wails, rendered via Windows WebView2)
├── bin/            # Isolated binaries for services and runtimes
├── conf/           # wincmp.json, Caddyfile, conf/sites/, php.ini, my.ini
├── data/           # MariaDB data directory (datadir) and runtime state
└── logs/           # Service and application logs
\`\`\`

## First-Run Flow

1. Open **Dependency Manager** (top-right on Dashboard) to download components such as Caddy, PHP, or MariaDB.
2. Add a project in **Projects** and pick its root directory; WinCMP auto-detects the framework preset and default port.
3. Start core services from **Dashboard**, or launch app runtimes from the project list.

> Core services stay self-contained inside \`WinCMP/\` without touching the Windows Registry or global \`PATH\`.`
            }
        },
        {
            id: "bin-structure",
            group: "group_basics",
            title: { zh: "手動加入依賴與 bin/ 規範", en: "Manual bin/ Dependencies" },
            content: {
                zh: `# 手動加入依賴與 \`bin/\` 命名規範

除了使用內建下載器，也可以直接把官方釋出的 Windows 可攜版（ZIP / PHAR）解壓放進 \`bin/\`。WinCMP 會依固定目錄前綴與執行檔路徑自動掃描。

## 掃描目錄與命名格式

| 元件 | 必要路徑格式 | 版本擷取規則 |
| :--- | :--- | :--- |
| **Caddy** | \`bin/caddy/caddy-<version>/caddy.exe\` | 擷取 \`caddy-\` 後字串（亦相容舊版 \`bin/caddy/caddy.exe\`） |
| **PHP** | \`bin/php/php-<version>/php-cgi.exe\` | 需為 NTS 版本；主次版本以 \`major.minor\` 分組 |
| **MariaDB** | \`bin/mariadb/mariadb-<version>/bin/mariadbd.exe\` | 若資料夾尾端含 \`-winx64\` 會自動正規化移除 |
| **Composer** | \`bin/composer/composer-<version>/composer.phar\` | 亦支援 \`composer.bat\`，優先讀取 \`composer.phar\` |
| **Node.js** | \`bin/node/node-<version>/npm.cmd\` | 解壓官方 win-x64 ZIP 後保留 \`node-<version>\` 命名 |
| **Bun** | \`bin/bun/bun-<version>/bun.exe\` | 內建下載器未收錄，需手動放置或使用系統全域 \`bun\` |
| **HeidiSQL** | \`bin/heidisql/heidisql-<version>/heidisql.exe\` | 可攜版解壓後放入對應版本目錄 |
| **Mailpit** | \`bin/mailpit/mailpit-<version>/mailpit.exe\` | 單一執行檔放入版本目錄即可 |
| **Redis** | \`bin/redis/redis-<version>/redis-server.exe\` | 亦相容直接放於 \`bin/redis/redis-server.exe\` |

## 多版本掃描規則（重要）

- **單一活躍版本服務**：\`Caddy\`、\`MariaDB\`、\`Mailpit\`、\`Redis\` 若同時存在多個版本資料夾，掃描器會依語意化版本排序，**僅保留最新版本**以避免埠口與服務衝突。
- **PHP 同分支去重**：同一主次版本（例如同時存在多個 \`8.2.x\` patch）只會啟用最新的 patch 版本，其餘舊 patch 會略過。
- **PHP Redis 擴充**：若要為特定 PHP 版本啟用 Redis，請將對應版本的 \`php_redis.dll\` 放入 \`bin/php/php-<version>/ext/\`，並在 \`conf/php/php.ini\` 啟用 \`extension=redis\`（或直接於面板一鍵啟用）。

> 手動放入檔案後，重新進入 Dashboard 或點擊重新掃描即可生效（掃描快取 TTL 為 2 秒）。`,
                en: `# Manual Dependencies & \`bin/\` Naming Rules

Besides the built-in downloader, you can manually drop official Windows portable archives into \`bin/\`. WinCMP discovers binaries by fixed folder prefixes and executable paths.

## Expected Directory Layout

| Component | Required Path Pattern | Notes |
| :--- | :--- | :--- |
| **Caddy** | \`bin/caddy/caddy-<version>/caddy.exe\` | Also supports legacy \`bin/caddy/caddy.exe\` |
| **PHP** | \`bin/php/php-<version>/php-cgi.exe\` | Use NTS builds; grouped by \`major.minor\` |
| **MariaDB** | \`bin/mariadb/mariadb-<version>/bin/mariadbd.exe\` | Trailing \`-winx64\` folder suffix is auto-trimmed |
| **Composer** | \`bin/composer/composer-<version>/composer.phar\` | Also accepts \`composer.bat\` (\`composer.phar\` preferred) |
| **Node.js** | \`bin/node/node-<version>/npm.cmd\` | Extract official win-x64 ZIP into \`node-<version>\` |
| **Bun** | \`bin/bun/bun-<version>/bun.exe\` | Not in built-in downloader; add manually or use global \`bun\` |
| **HeidiSQL** | \`bin/heidisql/heidisql-<version>/heidisql.exe\` | Portable build |
| **Mailpit** | \`bin/mailpit/mailpit-<version>/mailpit.exe\` | Single binary inside version folder |
| **Redis** | \`bin/redis/redis-<version>/redis-server.exe\` | Also supports \`bin/redis/redis-server.exe\` |

## Multi-Version Resolution Rules

- **Single active version**: For \`Caddy\`, \`MariaDB\`, \`Mailpit\`, and \`Redis\`, if multiple folders exist, WinCMP sorts by semantic version and **keeps only the newest** to prevent port conflicts.
- **PHP per-branch deduplication**: Multiple patches under the same \`major.minor\` branch (e.g. multiple \`8.2.x\`) keep only the newest patch; older patches are skipped.
- **PHP Redis extension**: Place matching \`php_redis.dll\` into \`bin/php/php-<version>/ext/\` and enable \`extension=redis\` in \`conf/php/php.ini\`.

> After placing files manually, revisit Dashboard or trigger a rescan (scan cache TTL is 2 seconds).`
            }
        },
        {
            id: "permissions-isolation",
            group: "group_basics",
            title: { zh: "權限模型與環境隔離", en: "Privileges & Isolation" },
            content: {
                zh: `# 權限模型與環境隔離

## 為什麼核心服務免管理員權限？

- Caddy、PHP-CGI、MariaDB、Mailpit、Redis 與各專案 Runtime 皆在**一般使用者權限（Standard Rights）**下啟動。
- 所有子行程透過 Windows **Job Object** 綁定主程式生命週期：關閉 WinCMP 時會連同子服務一併清理，避免殘留孤兒行程。

## 何時會需要 UAC 提權？

- **同步 Windows Hosts**：寫入 \`C:\\Windows\\System32\\drivers\\etc\\hosts\` 受系統保護，執行同步與自動備份前會請求 UAC 授權。
- **安裝系統憑證或特定操作**：若需信任本機自簽 CA 憑證至系統根憑證存放區，會由系統跳出權限確認。

## 動態環境變數隔離

WinCMP **不會修改 Windows 全域系統環境變數**。啟動子行程（如 Node.js、PHP、Composer）時，僅針對該單一子行程動態前置注入對應的 \`bin/\` 路徑到 \`PATH\`（或設定 \`PHPRC\`），確保多版本並存且不污染本機原有環境。`,
                en: `# Privilege Model & Environment Isolation

## Running Core Services Without Admin Rights

- Caddy, PHP-CGI, MariaDB, Mailpit, Redis, and project runtimes launch under **Standard User** privileges.
- Child processes are bound to WinCMP via Windows **Job Objects**, so closing WinCMP cleanly terminates spawned services without leaving orphan processes.

## When UAC Elevation Is Needed

- **Windows Hosts Sync**: Modifying \`C:\\Windows\\System32\\drivers\\etc\\hosts\` requires UAC elevation. WinCMP creates a backup before writing.
- **Root CA Trust**: Installing Caddy's local CA certificate into the Windows trust store prompts for elevation.

## Scoped Environment Isolation

WinCMP **never mutates system-wide PATH**. When spawning a child process (Node.js, PHP, Composer, etc.), it injects the required \`bin/\` directory into that process's local \`PATH\` (and \`PHPRC\` where applicable), keeping your global environment untouched.`
            }
        },
        {
            id: "caddy-sites",
            group: "group_services",
            title: { zh: "Caddy 站台與本機 HTTPS", en: "Caddy Sites & Local HTTPS" },
            content: {
                zh: `# Caddy 站台與本機 HTTPS

WinCMP 使用 Caddy 作為統一反向代理與靜態檔案伺服器。

## 站台配置與熱重載

- 每個啟用中的專案會自動在 \`conf/sites/\` 生成獨立站台設定，由主 \`conf/Caddyfile\` 引入。
- 編輯專案設定、切換啟用狀態或調整 PHP 版本後，WinCMP 會自動重寫設定並於背景執行 Caddy 熱重載，無需手動重啟服務。

## 兩種託管模式

1. **靜態 / PHP 託管**（\`Static HTML\`、\`純 PHP\`、\`Laravel PHP\`）：直接由 Caddy 設定 \`root * <project_root>\`，PHP 專案則自動轉發至對應 PHP 版本的 FastCGI upstream。
2. **Runtime 反向代理**（Node / Bun / Python / Go / Custom）：Caddy 自動產生 \`reverse_proxy localhost:<runtime_port>\`，讓本機開發伺服器直接享有自訂域名與 HTTPS。

## 本機 HTTPS 憑證

- 專案預設啟用 SSL：若未指定自訂 \`.crt\` / \`.key\` 路徑（或檔案不存在），會自動使用 Caddy 內建 \`tls internal\` 簽發本機憑證。
- 若需客製化規則，可在專案設定或操作中直接開啟該專案的 Caddyfile 檢視。`,
                en: `# Caddy Sites & Local HTTPS

WinCMP uses Caddy as its unified reverse proxy and static file server.

## Site Generation & Hot Reload

- Each enabled project generates a dedicated site config under \`conf/sites/\`, imported by the main \`conf/Caddyfile\`.
- Saving project changes, toggling enabled state, or switching PHP versions automatically rewrites configs and hot-reloads Caddy in the background.

## Hosting Modes

1. **Static / PHP Hosting** (\`Static HTML\`, \`PHP Site\`, \`Laravel PHP\`): Caddy serves \`root * <project_root>\` directly and forwards PHP requests to the selected PHP version's FastCGI upstream.
2. **Runtime Reverse Proxy** (Node / Bun / Python / Go / Custom): Caddy generates \`reverse_proxy localhost:<runtime_port>\`, giving local dev servers custom domains and HTTPS.

## Local HTTPS

- SSL is enabled by default. If custom \`.crt\` / \`.key\` files are not provided, Caddy falls back to \`tls internal\` for automatic local certificates.`
            }
        },
        {
            id: "php-composer",
            group: "group_services",
            title: { zh: "PHP 多版本與內建 Composer", en: "Multi-PHP & Built-in Composer" },
            content: {
                zh: `# PHP 多版本並行與內建 Composer

## PHP FastCGI 埠口規則

每個 PHP 主次版本（\`major.minor\`）可同時啟動，埠口依固定公式分配，互不衝突：

\`\`\`text
Port Base = 30000 + major * 1000 + minor * 100
\`\`\`

- 例如 PHP \`8.2\` 使用 \`38200\`、\`38201\`、\`38202\`；PHP \`7.4\` 使用 \`37400\`–\`37402\`。
- 每個版本預設啟動 **3 個**獨立 \`php-cgi.exe\` 進程（可於設定調整），由 Caddy 進行 upstream 負載均衡，避免單進程阻塞。

## 共用 \`php.ini\` 與跨版本相容

- 所有 PHP 版本共用 \`conf/php/php.ini\`。
- 啟動各版本 PHP 或執行 Composer 時，WinCMP 會自動檢查該版本 \`ext/\` 目錄，動態過濾不存在的擴充模組，避免切換版本時因缺少特定 DLL 而報錯。

## 內建 Composer 控制台（入口位置）

> **注意**：Composer 入口只會在專案類型為 **純 PHP** 或 **Laravel PHP** 時出現。

- **開啟方式**：前往 **Projects** → 點擊該 PHP / Laravel 專案列最右側的 **\`...\`（更多操作）** → 選擇 **Composer**。
- **運作特性**：
  - 自動綁定該專案指定的 PHP 版本與對應 \`PHPRC\`，不需手動切換全域 PHP。
  - 使用 \`bin/composer/composer-<version>/composer.phar\` 執行指令，支援常用指令提示與歷史紀錄查詢。`,
                en: `# Multi-Version PHP & Built-in Composer

## Deterministic PHP FastCGI Ports

Multiple \`major.minor\` PHP branches can run side by side without port collisions:

\`\`\`text
Port Base = 30000 + major * 1000 + minor * 100
\`\`\`

- Example: PHP \`8.2\` binds \`38200\`–\`38202\`; PHP \`7.4\` binds \`37400\`–\`37402\`.
- Each version spawns **3** independent \`php-cgi.exe\` workers by default (configurable), load-balanced by Caddy to avoid single-process blocking.

## Shared \`php.ini\` Compatibility

- All PHP versions share \`conf/php/php.ini\`.
- Before launching PHP or Composer, WinCMP checks the target version's \`ext/\` directory and dynamically filters out unavailable extension DLLs so switching versions stays error-free.

## Built-in Composer Console (Where to Find It)

> **Note**: The Composer menu item only appears for **PHP Site** and **Laravel PHP** projects.

- **How to open**: Go to **Projects** → click the **\`...\`** action menu on the right of a PHP / Laravel row → choose **Composer**.
- **Behavior**:
  - Automatically binds the project's configured PHP binary and \`PHPRC\`.
  - Runs \`bin/composer/composer-<version>/composer.phar\` with quick command hints and searchable command history.`
            }
        },
        {
            id: "db-mailpit",
            group: "group_services",
            title: { zh: "MariaDB、Redis 與 Mailpit", en: "MariaDB, Redis & Mailpit" },
            content: {
                zh: `# MariaDB、Redis 與 Mailpit

## 核心服務與預設埠口

| 服務 | 預設埠口 | 用途與說明 |
| :--- | :--- | :--- |
| **MariaDB** | \`3306\` | 關聯式資料庫（設定檔位於 \`conf/my.ini\`，資料位於 \`data/\`） |
| **Redis** | \`6379\` | 記憶體快取與佇列服務 |
| **Mailpit** | SMTP \`1025\` / Web \`8025\` | 本機郵件攔截測試伺服器，啟動後可於瀏覽器開啟 Web UI 檢視信件 |

## 內建 Database Explorer

- 前往側邊欄 **Database** 分頁可切換 **MariaDB (MySQL)** 與 **Redis (NoSQL)**。
- **MariaDB 模式**：需先啟動 MariaDB 服務，可快速瀏覽 Schema 與資料表清單；需要完整 SQL 編輯與匯入匯出時，點擊右上角 **Open in HeidiSQL** 直接帶入連線資訊開啟。
- **Redis 模式**：支援依 DB 編號切換、萬用字元搜尋、以 \`:\` 分隔的 Key Tree 樹狀檢視，以及 JSON 格式化預覽。

## 外接既有 MariaDB

若本機已有其他 MariaDB / MySQL 實例，可在 **Settings** 啟用外接模式並指定埠口、帳號與路徑，即可沿用既有的 Database Explorer 與 HeidiSQL 整合。`,
                en: `# MariaDB, Redis & Mailpit

## Core Services & Default Ports

| Service | Default Port | Purpose |
| :--- | :--- | :--- |
| **MariaDB** | \`3306\` | Relational database (config in \`conf/my.ini\`, data in \`data/\`) |
| **Redis** | \`6379\` | In-memory cache and queue store |
| **Mailpit** | SMTP \`1025\` / Web \`8025\` | Local SMTP catcher with Web UI for inspecting outgoing emails |

## Built-in Database Explorer

- Open **Database** in the sidebar to switch between **MariaDB (MySQL)** and **Redis (NoSQL)**.
- **MariaDB tab**: Browse schemas and tables once MariaDB is running; click **Open in HeidiSQL** to launch the full GUI client with current connection settings.
- **Redis tab**: Switch DB index, filter keys with wildcards, browse colon-delimited keys as a tree, and inspect formatted JSON values.

## External MariaDB Support

If you already run a local MariaDB/MySQL instance, enable external mode in **Settings** to connect Database Explorer and HeidiSQL to your existing server.`
            }
        },
        {
            id: "builtin-terminal",
            group: "group_services",
            title: { zh: "內建專案終端（ConPTY）", en: "Built-in Terminal (ConPTY)" },
            content: {
                zh: `# 內建專案終端（ConPTY Drawer）

WinCMP 內建基於 Windows **ConPTY** 與 **xterm.js** 的互動式終端抽屜，無需切出視窗即可在專案根目錄操作指令。

## 開啟方式

前往 **Projects** → 點擊任意專案列右側的 **\`...\`（更多操作）** → 選擇 **開啟專案終端（Open Project Terminal）**。

## 功能特性

- **自動定位目錄**：開啟時工作目錄直接位於該專案的 \`root_path\`。
- **多 Shell 支援**：可在 **Settings** 切換預設 Shell，支援 \`PowerShell\`、\`CMD\`、\`Git Bash\` 與 \`WSL\`。
- **完整互動能力**：支援 \`Ctrl+C\` 中斷、方向鍵歷史指令、\`TAB\` 自動補齊與 ANSI 彩色輸出。
- **一鍵重啟**：終端右下角提供 \`Restart\` 按鈕，環境異常時可快速重建 PTY 工作階段。`,
                en: `# Built-in Project Terminal (ConPTY Drawer)

WinCMP includes an interactive terminal drawer powered by Windows **ConPTY** and **xterm.js**.

## How to Open

Go to **Projects** → click the **\`...\`** action menu on any project row → choose **Open Project Terminal**.

## Capabilities

- **Project-Scoped Working Directory**: Opens directly at the project's \`root_path\`.
- **Multiple Shells**: Choose between \`PowerShell\`, \`CMD\`, \`Git Bash\`, and \`WSL\` in **Settings**.
- **Full Interactive PTY**: Supports \`Ctrl+C\`, command history, \`TAB\` completion, and ANSI colors.
- **Quick Restart**: Use the bottom-right \`Restart\` button to reset the PTY session anytime.`
            }
        },
        {
            id: "runtime-presets",
            group: "group_runtime",
            title: { zh: "專案 Runtime 與預設指令", en: "Runtime Presets & Commands" },
            content: {
                zh: `# 專案 Runtime 與預設啟動指令

非靜態 / PHP 專案會由 WinCMP 管理其開發伺服器行程，並透過 Caddy 反向代理至指定 \`%PORT%\`。

## 預設 Preset 指令對照表

| 專案類型 | 預設 Port | 預設啟動指令模板 |
| :--- | :--- | :--- |
| **Next.js** | \`3000\` | \`npm run dev -- --port %PORT%\` / \`bun run dev -- --port %PORT%\` |
| **Nuxt** | \`3000\` | \`npm run dev -- --port %PORT% --host 0.0.0.0\` |
| **Astro** | \`4321\` | \`npm run dev -- --host 0.0.0.0 --port %PORT%\` |
| **Vite React/Vue** | \`5173\` | \`npm run dev -- --port %PORT%\` |
| **Python FastAPI** | \`8000\` | \`python -m uvicorn main:app --host 0.0.0.0 --port %PORT% --reload\` |
| **Python Django** | \`8000\` | \`python manage.py runserver 0.0.0.0:%PORT%\` |
| **Python Flask** | \`5000\` | \`python -m flask run --host=0.0.0.0 --port=%PORT%\` |
| **Python HTTP** | \`8000\` | \`python -m http.server %PORT% --bind 0.0.0.0\` |
| **Go Web API** | \`8080\` | \`air\`（Go Air）或 \`go run main.go\`（Go Run） |
| **PocketBase** | \`8090\` | \`go run main.go serve --http=0.0.0.0:%PORT%\` |
| **Custom** | \`3000\` | 由使用者自訂指令（支援 \`%PORT%\` 佔位符） |

## 運行細節說明

- **佔位符**：指令中可使用 \`%PORT%\`（自動替換為專案設定的執行埠口）。
- **Background vs Terminal**：
  - \`Background\`：靜默於背景執行，日誌匯入 WinCMP 底部終端面板。
  - \`Terminal\`：另開獨立終端機視窗，適合需要即時互動輸入的開發伺服器。
- **使用 WinCMP 內建執行檔（Bundled Runtime）**：針對 Node.js / Bun 專案，可切換使用 \`bin/\` 內的隔離版本或系統全域環境；若系統已安裝全域 Node/Bun，新增專案時預設不強制勾選內建版本。
- **自訂啟動指令安全限制**：為防範指令注入，啟動指令禁止包含 shell 中繼字元（如 \`&\`、\`|\`、\`;\`）或脫鉤指令（\`cmd /c start\`、\`Start-Process\`）；若執行本機腳本，僅允許位於該專案根目錄內。`,
                en: `# Project Runtimes & Default Commands

Non-static/PHP projects run their own dev server managed by WinCMP and reverse-proxied by Caddy to \`%PORT%\`.

## Default Preset Commands

| Project Type | Default Port | Default Command Template |
| :--- | :--- | :--- |
| **Next.js** | \`3000\` | \`npm run dev -- --port %PORT%\` / \`bun run dev -- --port %PORT%\` |
| **Nuxt** | \`3000\` | \`npm run dev -- --port %PORT% --host 0.0.0.0\` |
| **Astro** | \`4321\` | \`npm run dev -- --host 0.0.0.0 --port %PORT%\` |
| **Vite React/Vue** | \`5173\` | \`npm run dev -- --port %PORT%\` |
| **Python FastAPI** | \`8000\` | \`python -m uvicorn main:app --host 0.0.0.0 --port %PORT% --reload\` |
| **Python Django** | \`8000\` | \`python manage.py runserver 0.0.0.0:%PORT%\` |
| **Python Flask** | \`5000\` | \`python -m flask run --host=0.0.0.0 --port=%PORT%\` |
| **Python HTTP** | \`8000\` | \`python -m http.server %PORT% --bind 0.0.0.0\` |
| **Go Web API** | \`8080\` | \`air\` (Go Air) or \`go run main.go\` (Go Run) |
| **PocketBase** | \`8090\` | \`go run main.go serve --http=0.0.0.0:%PORT%\` |
| **Custom** | \`3000\` | User-defined command (supports \`%PORT%\`) |

## Execution Details

- **Placeholders**: \`%PORT%\` is automatically replaced with the project's configured runtime port.
- **Background vs Terminal**: \`Background\` streams output to WinCMP's log panel; \`Terminal\` opens a separate console window for interactive dev servers.
- **Bundled vs Global Runtime**: Node.js/Bun projects can toggle between \`bin/\` bundled binaries and system global binaries.
- **Command Validation**: Shell metacharacters (\`&\`, \`|\`, \`;\`) and detached launchers (\`cmd /c start\`, \`Start-Process\`) are rejected; local scripts must stay inside the project root.`
            }
        },
        {
            id: "portability-troubleshooting",
            group: "group_runtime",
            title: { zh: "可攜遷移與常見排查", en: "Portability & Troubleshooting" },
            content: {
                zh: `# 可攜遷移與常見排查

## 整包搬家與備份

將 WinCMP 遷移至另一台 Windows 電腦或不同磁碟槽時，只需複製以下項目：

1. \`WinCMP.exe\`（或重新下載最新版本）
2. \`conf/wincmp.json\`（以及自訂過的 \`conf/php/php.ini\`、\`conf/my.ini\`）
3. \`bin/\` 目錄（已下載的各服務二進位檔）
4. 若需保留 MariaDB 資料庫內容，請一併複製 \`data/\` 目錄（建議先停止 MariaDB 再複製）。

## 埠口佔用與安全保護

- 啟動專案 Runtime 前，WinCMP 會自動偵測目標埠口是否被其他進程佔用，並提示是否結束該進程。
- **安全保護限制**：埠口 \`<= 1024\`（系統保留埠）與 \`3389\`（遠端桌面）受安全機制保護，面板**拒絕強制終止**佔用這些埠口的進程。

## 常見問題速查

- **儀表板顯示缺少元件或未啟動**：頂部提醒列會依嚴重度提示缺少 Caddy、缺少專案所需 PHP 版本或服務未啟動，直接點擊提醒列上的按鈕即可下載或啟動。
- **手動停止服務或重啟更新**：所有服務皆有正常停止旗標，不會殘留孤兒進程；若從舊版一鍵更新重啟，先前開啟中的服務狀態會自動恢復。`,
                en: `# Portability & Troubleshooting

## Moving or Backing Up WinCMP

To migrate WinCMP to another Windows machine or drive, copy:

1. \`WinCMP.exe\` (or download the latest release)
2. \`conf/wincmp.json\` (plus any customized \`conf/php/php.ini\` or \`conf/my.ini\`)
3. \`bin/\` directory (downloaded service binaries)
4. \`data/\` directory if you want to keep MariaDB databases (stop MariaDB before copying).

## Port Conflict Detection & Safety Guard

- Before starting a project runtime, WinCMP checks if the target port is occupied and offers to terminate the conflicting process.
- **Protected Ports**: Ports \`<= 1024\` and \`3389\` (RDP) are protected; WinCMP refuses to force-kill processes bound to those ports.

## Quick Checks

- **Dashboard Alerts**: When enabled projects miss Caddy, a required PHP version, or have stopped services, the top alert banner provides a direct Download or Start action.
- **Clean Restarts**: Built-in self-updates wait for existing services to release ports and automatically restore previously running services.`
            }
        }
    ];

    const groupOrder = ["group_basics", "group_services", "group_runtime"];

    // ==========================================================================
    // 3. 主題、語系與導覽狀態管理
    // ==========================================================================
    let currentLang = localStorage.getItem('wincmp_lang') ||
        ((navigator.language || 'en').toLowerCase().startsWith('zh') ? 'zh' : 'en');
    let currentSectionId = getSectionIdFromHash();

    const menuEl = document.getElementById('docs-menu');
    const articleEl = document.getElementById('docs-article');
    const mobileCurrentEl = document.getElementById('docs-mobile-current');
    const prevBtn = document.getElementById('docs-prev-btn');
    const prevTitle = document.getElementById('docs-prev-title');
    const nextBtn = document.getElementById('docs-next-btn');
    const nextTitle = document.getElementById('docs-next-title');
    const sidebarEl = document.getElementById('docs-sidebar');
    const backdropEl = document.getElementById('docs-backdrop');

    function getSectionIdFromHash() {
        const hash = (window.location.hash || '').replace(/^#/, '').trim();
        if (hash && docSections.some(s => s.id === hash)) {
            return hash;
        }
        return docSections[0].id;
    }

    function applyTheme(theme) {
        if (theme === 'sketch') {
            document.documentElement.setAttribute('data-theme', 'sketch');
        } else {
            document.documentElement.removeAttribute('data-theme');
        }
        localStorage.setItem('wincmp_theme', theme);
    }

    function applyUIStrings() {
        const dict = uiStrings[currentLang] || uiStrings.en;
        document.querySelectorAll('[data-i18n]').forEach(el => {
            const key = el.getAttribute('data-i18n');
            if (dict[key]) {
                if (el.tagName === 'TITLE') {
                    document.title = dict[key];
                } else {
                    el.textContent = dict[key];
                }
            }
        });
        document.documentElement.lang = currentLang === 'zh' ? 'zh-TW' : 'en';
    }

    function renderSidebar() {
        const dict = uiStrings[currentLang] || uiStrings.en;
        let html = '';
        let globalIdx = 1;

        groupOrder.forEach(groupKey => {
            const items = docSections.filter(s => s.group === groupKey);
            if (!items.length) return;
            html += `<div class="docs-group">`;
            html += `<div class="docs-group-title">${dict[groupKey] || groupKey}</div>`;
            items.forEach(sec => {
                const isActive = sec.id === currentSectionId;
                const label = sec.title[currentLang] || sec.title.en;
                const num = String(globalIdx++).padStart(2, '0');
                html += `<a href="#${sec.id}" class="docs-menu-item${isActive ? ' active' : ''}" data-section-id="${sec.id}">`;
                html += `<span class="docs-menu-num">${num}</span>`;
                html += `<span>${label}</span>`;
                html += `</a>`;
            });
            html += `</div>`;
        });

        menuEl.innerHTML = html;
        menuEl.querySelectorAll('.docs-menu-item').forEach(link => {
            link.addEventListener('click', (e) => {
                e.preventDefault();
                const id = link.getAttribute('data-section-id');
                selectSection(id, true);
                closeSidebar();
            });
        });
    }

    function renderArticle() {
        const idx = docSections.findIndex(s => s.id === currentSectionId);
        const sec = docSections[idx >= 0 ? idx : 0];
        const md = sec.content[currentLang] || sec.content.en;

        if (window.marked) {
            articleEl.innerHTML = window.marked.parse(md);
            // 幫表格包上橫向捲動容器，避免窄螢幕撐爆版面
            articleEl.querySelectorAll('table').forEach(table => {
                const wrapper = document.createElement('div');
                wrapper.className = 'docs-table-wrap';
                table.parentNode.insertBefore(wrapper, table);
                wrapper.appendChild(table);
            });
        } else {
            articleEl.textContent = md;
        }

        if (mobileCurrentEl) {
            mobileCurrentEl.textContent = sec.title[currentLang] || sec.title.en;
        }

        // 更新上一篇 / 下一篇
        if (idx > 0) {
            const prevSec = docSections[idx - 1];
            prevBtn.href = `#${prevSec.id}`;
            prevTitle.textContent = prevSec.title[currentLang] || prevSec.title.en;
            prevBtn.classList.remove('hidden');
            prevBtn.onclick = (e) => { e.preventDefault(); selectSection(prevSec.id, true); };
        } else {
            prevBtn.classList.add('hidden');
            prevBtn.onclick = null;
        }

        if (idx < docSections.length - 1) {
            const nextSec = docSections[idx + 1];
            nextBtn.href = `#${nextSec.id}`;
            nextTitle.textContent = nextSec.title[currentLang] || nextSec.title.en;
            nextBtn.classList.remove('hidden');
            nextBtn.onclick = (e) => { e.preventDefault(); selectSection(nextSec.id, true); };
        } else {
            nextBtn.classList.add('hidden');
            nextBtn.onclick = null;
        }
    }

    function selectSection(id, updateHash = false) {
        if (!docSections.some(s => s.id === id)) return;
        currentSectionId = id;
        if (updateHash && window.location.hash !== `#${id}`) {
            history.pushState(null, '', `#${id}`);
        }
        renderSidebar();
        renderArticle();
        window.scrollTo({ top: 0, behavior: 'smooth' });
    }

    function openSidebar() {
        sidebarEl.classList.add('open');
        backdropEl.classList.add('open');
        document.body.classList.add('menu-open');
    }

    function closeSidebar() {
        sidebarEl.classList.remove('open');
        backdropEl.classList.remove('open');
        document.body.classList.remove('menu-open');
    }

    // 綁定事件
    const savedTheme = localStorage.getItem('wincmp_theme') || 'sketch';
    applyTheme(savedTheme);

    const themeBtn = document.getElementById('theme-switch-btn');
    if (themeBtn) {
        themeBtn.addEventListener('click', () => {
            const isSketch = document.documentElement.getAttribute('data-theme') === 'sketch';
            applyTheme(isSketch ? 'dark' : 'sketch');
        });
    }

    const langBtn = document.getElementById('lang-switch-btn');
    if (langBtn) {
        langBtn.addEventListener('click', () => {
            currentLang = currentLang === 'en' ? 'zh' : 'en';
            localStorage.setItem('wincmp_lang', currentLang);
            applyUIStrings();
            renderSidebar();
            renderArticle();
        });
    }

    const sidebarToggle = document.getElementById('docs-sidebar-toggle');
    const mobileMenuBtn = document.getElementById('docs-mobile-menu-btn');
    const sidebarClose = document.getElementById('docs-sidebar-close');
    if (sidebarToggle) sidebarToggle.addEventListener('click', openSidebar);
    if (mobileMenuBtn) mobileMenuBtn.addEventListener('click', openSidebar);
    if (sidebarClose) sidebarClose.addEventListener('click', closeSidebar);
    if (backdropEl) backdropEl.addEventListener('click', closeSidebar);

    window.addEventListener('hashchange', () => {
        const nextId = getSectionIdFromHash();
        if (nextId !== currentSectionId) {
            selectSection(nextId, false);
        }
    });

    // 初始化渲染
    applyUIStrings();
    renderSidebar();
    renderArticle();
    if (window.lucide) {
        window.lucide.createIcons();
    }
});
