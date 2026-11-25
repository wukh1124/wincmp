---
name: wincmp-dependencies-update
description: Check, audit, and update WinCMP bundled binary dependencies in conf/dependencies.json (PHP, Caddy, MariaDB, Node.js, Composer, HeidiSQL, Mailpit, Redis, PECL php_redis, cacert). Use when checking for security releases, bumping dependency versions, refreshing SHA-256 hashes, or syncing docs/dependencies.md.
---

# WinCMP Dependencies Update: 依賴目錄檢查與更新規範

本 Skill 用於定期檢查、稽核與更新 WinCMP 的外部二進位依賴目錄 `conf/dependencies.json`，確保各元件維持安全版本、下載連結有效且 SHA-256 校驗完整。

---

## 0. 核心原則與安全底線

- **單一來源**：`conf/dependencies.json` 會透過 `//go:embed` 嵌入執行檔，且 WinCMP 內建下載器的「獲取最新」功能會直接拉取 `main` 分支的 `conf/dependencies.json`。
- **來源白名單**：所有 `url` 必須使用官方 HTTPS 信任網域（`github.com`、`downloads.php.net`、`windows.php.net`、`archive.mariadb.org`、`nodejs.org`、`curl.se`），嚴禁使用第三方鏡像或非 HTTPS 連結。
- **強制 SHA-256**：每一項依賴皆須具備 64 碼十六進位 `sha256`；嚴禁留空、略過校驗或手動估算。
- **同分支升級原則**：例行更新僅限同主/次分支的 security / patch / minor 升級（例如 `php82` 8.2.35 → 8.2.36、`mariadb` 11.4.x、`node` 24.x）。新增大版本（如 `php86`）或更換 LTS 主線需同步調整掃描器、下載器與前端 UI，須先與開發者確認。
- **嚴禁自動推送**：AI 助手不得擅自執行 `git commit` 或 `git push`；完成修改與驗證後，提供 commit message 草稿供開發者審核。
- **推送順序提醒**：發版前若有依賴安全更新，必須先將 `conf/dependencies.json` 推送至 `main` 再執行發版，避免使用者點擊「獲取最新」時被覆蓋回舊版。

---

## 1. 依賴項目與上游檢查策略

`conf/dependencies.json` 目前包含 20 個項目，檢查時依下列策略對照上游官方發布：

| 群組 | Keys | 上游來源與檢查重點 | 檔名 / 編譯器規則 |
| :--- | :--- | :--- | :--- |
| **PHP 活躍分支** | `php82`, `php83`, `php84`, `php85`, `php86` | `php.net/releases/` 與 `downloads.php.net/~windows/releases/`（優先關注 Security Release） | `php-<ver>-nts-Win32-<vc>-x64.zip`；`8.2`/`8.3` 用 `vs16`，`8.4`/`8.5` 用 `vs17`，`8.6` 用 `vs18` |
| **PHP EOL 分支** | `php73`, `php74` | 已停止維護，固定指向 `windows.php.net/downloads/releases/archives/` | 僅確認連結存活，不主動更動版本 |
| **PHP Redis 擴充** | `php_redis_74`, `php_redis_82`..`85` | `github.com/phpredis/phpredis/releases` 與 `downloads.php.net/~windows/pecl/releases/redis/` | `7.4` 鎖定 `5.3.7` (`vc15`)；`8.2`..`8.5` 跟隨 `6.x`（`8.2`/`8.3` 用 `vs16`，`8.4`/`8.5` 用 `vs17`） |
| **GitHub Releases** | `caddy`, `composer`, `heidisql`, `mailpit`, `redis` | 查詢各 `source_url` 對應之 GitHub Releases | `caddy`: `caddy_<ver>_windows_amd64.zip`<br>`composer`: `composer.phar`<br>`heidisql`: `HeidiSQL_<ver>_64_Portable.zip`<br>`mailpit`: `mailpit-windows-amd64.zip`<br>`redis`: `Redis-x64-<ver>.zip` (`tporadowski/redis`) |
| **MariaDB** | `mariadb` | `archive.mariadb.org`（鎖定 `11.4.x` LTS 分支） | `mariadb-<ver>/winx64-packages/mariadb-<ver>-winx64.zip` |
| **Node.js** | `node` | `nodejs.org/dist/`（鎖定 `24.x` LTS 分支） | `v<ver>/node-v<ver>-win-x64.zip` |
| **CA 憑證包** | `cacert` | `curl.se/ca/` | 使用固定日期檔 `cacert-YYYY-MM-DD.pem`（禁止使用無日期的 `cacert.pem`，避免上游輪替造成 SHA-256 失配） |

---

## 2. 標準檢查與更新流程

### Step 1：快速健康檢查（檢查現有連結與雜湊格式）

```powershell
go run scripts/check_deps.go --check --fast
```

- 若有 PHP 舊版連結出現「原連結已歸檔或變更」，代表上游已發布新 patch 並將舊檔移至 `releases/archives/`，應優先檢查新版。

### Step 2：上游版本比對與回報（先確認再改）

檢查指定元件（或全部元件）的上游最新穩定版與安全公告，向開發者輸出版本差異表，待確認後再動手修改：

```markdown
| Key | 目前版本 | 上游最新版 | 安全更新 / 備註 | 建議動作 |
| :--- | :--- | :--- | :--- | :--- |
| `php82` | 8.2.35 | 8.2.36 | Security release | 更新 |
```

### Step 3：更新 `conf/dependencies.json`

針對已確認要更新的項目：
1. 更新 `version` 與 `url`（保留既有 `license`、`homepage`、`source_url`，除非官方變更）。
2. 將**本次有變動項目**的 `sha256` 清空為 `""`（未變動項目保留原 `sha256`，讓下一步只下載異動項以節省時間與頻寬）。

### Step 4：自動下載並回填 SHA-256 與全量校驗

```powershell
# 僅下載 sha256 為空的異動項目，計算雜湊並寫回 conf/dependencies.json
go run scripts/check_deps.go --update

# 驗證所有項目連結存活與 SHA-256 格式（快速）
go run scripts/check_deps.go --check --fast
```

若為發版前或需重新核對全部檔案完整性，執行全量校驗（會下載全部依賴比對 SHA-256，大檔逾時上限為 600 秒）：

```powershell
go run scripts/check_deps.go --check
```

若網路不穩，可加上 `--concurrency 2` 降低併發數。

### Step 5：同步文件 `docs/dependencies.md`

檢查並同步更新 `docs/dependencies.md` 中的：
1. 「預設目錄（與 dependencies.json 對應）」表格內的「目前版本」與「安裝位置」欄位。
2. 「目錄範例」區塊中的版本號。

### Step 6：執行測試驗證

```powershell
go test ./...
```

確認 `internal/config` 能正確載入嵌入的 `conf/dependencies.json`，且所有單元測試通過。

---

## 3. 常見陷阱與排查

- **PHP 新版尚未同步至 Windows 下載站**：`php.net` 發布原始碼公告後，Windows 二進位（`downloads.php.net/~windows/releases/`）有時會延遲數小時至一日上架。若 `check_deps.go` 回報 404，請先確認 Windows 建置是否已上架，切勿提前改版號。
- **PECL `php_redis` 版本相容**：更新 PHP patch 版本（如 8.2.35 → 8.2.36）通常不影響同一 ABI 的 `php_redis` DLL，但仍建議順帶確認 `phpredis` 是否有新版或安全修復。
- **JSON 欄位順序與格式**：`scripts/check_deps.go --update` 會以 Go `json.MarshalIndent`（2 空白縮排）重寫 `conf/dependencies.json`，請用 `git diff conf/dependencies.json` 確認僅有預期項目變更。

---

## 4. 交付前自檢清單與 Commit 範本

交付前請逐項確認：
- [ ] `conf/dependencies.json` 所有更新項目皆來自官方 HTTPS 白名單網域。
- [ ] `go run scripts/check_deps.go --check --fast` 全部通過（無缺漏或非法 SHA-256）。
- [ ] `git diff conf/dependencies.json` 僅包含本次確認更新的項目，無誤改其他條目。
- [ ] `docs/dependencies.md` 已同步更新版本與路徑範例。
- [ ] `go test ./...` 執行零錯誤。

### 建議 Commit Message 草稿

- 單一或少數元件安全更新：
  ```text
  chore(deps): update php82 to 8.2.36 and refresh sha256
  ```
- 多項例行依賴更新：
  ```text
  chore(deps): update bundled dependencies catalog and sync docs
  ```
