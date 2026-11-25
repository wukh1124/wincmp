# PHP 專案 Composer 控制台與指令歷史持久化計畫

主題：**PHP 專案專屬 Composer 簡化控制台（自製終端介面）＋ 指令執行歷史持久化與滾動分頁載入**。於專案管理操作選單提供快速入口，自動綁定專案 PHP 版本並支援多版本 Composer 切換，內建 Laravel 常用指令與參數提示，並持久化記錄每次執行的時間、耗時、狀態與輸出以利偵錯。

---

## 1. 背景與可行性評估

### 1.1 單一最新版 Composer 與多版本 PHP 相容性

* **結論**：單一最新版 Composer（目前 `conf/dependencies.json` 為 **v2.11.2**）即可完整覆蓋 WinCMP 所有內建 PHP 版本（**PHP 7.3 ～ PHP 8.5**）。
* **原因**：Composer 2.3+ 至 2.11.x 官方最低需求皆為 **PHP 7.2.5**，且完整支援至 PHP 8.5。執行時透過專案對應的 `php.exe` 驅動 `composer.phar`，會自動以該 PHP 版本與擴充環境解析依賴。
* **例外**：僅維護 PHP 7.2.4 以下（PHP 5.6 / 7.0 / 7.1）的極舊系統才需手動放置 Composer 2.2 LTS；介面提供多版本偵測切換即可兼顧此情境。

### 1.2 指令歷史持久化與滾動分頁評估

* **合理性**：高度實用。Laravel 開發頻繁執行 `install`、`update`、`dump-autoload -o`、`audit`、`show` 等指令，保留時間戳、耗時、PHP/Composer 版本與輸出摘要可快速回溯問題。
* **儲存位置與格式選型**：
  * **不採用** `conf/wincmp.json`（避免設定檔膨脹與高頻寫入競爭）與 `localStorage`（5MB 上限且不易備份）。
  * **採用檔案持久化**：按專案獨立儲存於 `data/composer_history/<project_name>.jsonl`（**JSONL / NDJSON**，每行一筆 JSON，並於 `.gitignore` 新增 `/data/composer_history/`）。
* **上限與效能控管**：
  * **磁碟保存上限**：每專案預設最多保留 **50 筆**（`MaxHistoryPerProject = 50`），寫入超過上限時自動淘汰最舊紀錄。
  * **單筆輸出頭尾截斷上限**：每筆 `output` 最多保留 **32 KB**（**前 8 KB + 後 24 KB**，於換行與 UTF-8 字元邊界安全切分，避免切斷中文字或 ANSI 控制碼）；中間插入省略提示並於 Schema 記錄 `original_bytes` 與 `original_lines`，確保開頭環境資訊與結尾錯誤堆疊皆完整保留。
  * **超時與異常失敗防禦**：採用「**總執行超時（600s）＋ 無輸出閒置超時（180s）**」雙層保護，超時或手動停止時終止整個子行程樹（含 `git` / `artisan` 衍生進程），且無論 `success` / `error` / `stopped` / `timeout` 一律持久化紀錄方便事後偵錯。
  * **前端顯示與分頁策略**：每次開啟控制台預設載入最新 **5 筆**（`offset=0, limit=5`）；使用者**向上滾動至頂部**（或點擊頂部「載入更早記錄」按鈕）時，再向後端分批讀取更早紀錄（每次 5 ～ 10 筆）並維持捲動位置不跳動。
* **終端呈現選型**：
  * 採用**自製結構化 ANSI 彩色終端輸出區**（將 ANSI 色碼安全解析為 React 節點，不用 `dangerouslySetInnerHTML`），取代單一 `@xterm/xterm` 畫布，以便每筆歷史各自呈現時間戳、耗時、版本標籤、退出碼，並流暢支援向上滾動分頁與一鍵重跑。

---

## 2. UI / UX 設計（ASCII 示意）

### 2.1 操作欄下拉選單入口（`frontend/src/components/Projects.tsx`）

僅針對 `proj.type === 'laravel' || proj.type === 'php'` 的專案顯示 `Composer` 選項：

```text
┌──────────────────────┐
│ [>_] 開啟專案終端     │
│ [Pk] Composer        │  <-- 僅 PHP / Laravel 專案顯示
│ [Fd] 開啟專案資料夾   │
│ [Ed] 編輯專案設定     │
├──────────────────────┤
│ [Tr] 刪除專案         │
└──────────────────────┘
```

### 2.2 Composer 控制台面板（`frontend/src/components/ComposerModal.tsx`）

```text
+------------------------------------------------------------------------+
| [Pk] Composer 控制台  |  cms-global                 [清除歷史]     [X] |
| C:\Users\tony.wu\laragon\www\cms_global                                |
+------------------------------------------------------------------------+
| 環境列:  [PHP 8.2.36 (跟隨專案)]   [Composer: v2.11.2 v]    [指令速查] |
+------------------------------------------------------------------------+
| (可收合：Laravel 常用指令 & 參數速查列，點擊帶入或附加參數)                |
|  指令: [install] [update] [dump-autoload -o] [show] [audit] [outdated] |
|  參數: [--no-dev] [--ignore-platform-reqs] [--dry-run] [-W]            |
+------------------------------------------------------------------------+
|                    [ ^ 向上滾動載入更早記錄 (還有 12 筆) ]                |
|                                                                        |
|  2026-11-25 10:15:02 | PHP 8.2.36 | Composer 2.11.2 | 1.4s | Exit 0    |
|  $ composer audit                                                      |
|  No security vulnerability advisories found.                           |
|  --------------------------------------------------------------------  |
|  2026-11-25 10:28:40 | PHP 8.2.36 | Composer 2.11.2 | 3.8s | Exit 0    |
|  $ composer dump-autoload -o                                           |
|  Generating optimized autoload files                                   |
|  Generated optimized autoload files containing 6214 classes            |
|                                                                        |
+------------------------------------------------------------------------+
|  composer > [ 輸入指令，如 install / show / require laravel/sanctum  ] |
|  (輸入即時篩選指令與中文用法提示，支援 上/下鍵 翻查歷史)    [執行] [停止] |
+------------------------------------------------------------------------+
```

---

## 3. 資料結構與後端設計

### 3.1 歷史紀錄 Schema（`data/composer_history/<project>.jsonl`）

```json
{
  "id": "1764037720123-a1b2",
  "project": "cms-global",
  "command": "dump-autoload -o",
  "args": ["dump-autoload", "-o"],
  "php_version": "8.2.36",
  "composer_version": "2.11.2",
  "started_at": "2026-11-25T10:28:40+08:00",
  "finished_at": "2026-11-25T10:28:44+08:00",
  "duration_ms": 3812,
  "exit_code": 0,
  "status": "success",
  "error_reason": "",
  "output": "Generating optimized autoload files...",
  "truncated": false,
  "original_bytes": 128,
  "original_lines": 3
}
```

欄位補充說明：
* `status`：`success`（Exit 0）｜`error`（非 0 退出或啟動失敗）｜`stopped`（使用者手動停止）｜`timeout`（總時長或無輸出超時）。
* `error_reason`：記錄失敗或逾時的精簡原因（例如 `idle_timeout_180s`、`hard_timeout_600s`、啟動錯誤訊息），正常成功時為空字串。
* `truncated` / `original_bytes` / `original_lines`：當輸出超過 32 KB 時 `truncated: true`，`output` 保留前 8 KB 與後 24 KB，中間插入如 `... [已省略 142,850 bytes / 1,840 行，原始總計 175,618 bytes / 2,210 行] ...` 之分隔提示。

### 3.2 長輸出截斷與逾時／失敗處理規格

1. **輸出串流與頭尾緩衝（Head + Tail Buffer）**：
   * 執行階段後端以固定容量之「頭部緩衝（8 KB）＋ 尾部環形緩衝（24 KB）」同步累計 `original_bytes` 與 `original_lines`，即使遇到 `composer install -vvv` 輸出數十 MB 也不會導致記憶體飆升。
   * 切分時對齊 UTF-8 字元與換行邊界（若尾部開頭殘留半截 ANSI 轉義序列，則回退至下一個換行或補上重置碼 `\x1b[0m`），確保前端顯示不破版。
   * 前端即時串流顯示亦設單次畫面行數上限（例如最多即時保留 1,500 行，超出自動淘汰最舊顯示行），防止大量輸出拖垮渲染。
2. **雙層逾時機制（Hard Timeout + Idle Timeout）**：
   * **總執行超時（Hard Timeout）**：預設 **600 秒（10 分鐘）**，防止大型依賴更新被過短超時誤殺，同時兜底避免進程無限掛起。
   * **無輸出閒置超時（Idle Timeout）**：預設 **180 秒（3 分鐘）** 內 stdout/stderr 完全無任何位元組輸出即判定卡死（例如網路斷線、私有套件認證等待互動、post-script 死鎖），自動觸發終止並於輸出尾端附加可讀診斷提示。
   * 同步於子行程環境注入 `COMPOSER_PROCESS_TIMEOUT=600` 與 `COMPOSER_NO_INTERACTION=1`，避免 Composer 內部預設 300 秒超時與外層策略不一致。
3. **失敗與中斷處理**：
   * **行程樹完整終止**：逾時或手動點擊「停止」時，透過 Windows Job Object 關閉整個行程樹（連同 `git.exe`、`unzip`、`php artisan` 等衍生子進程一併結束），不留孤兒進程。
   * **異常必留紀錄**：無論啟動失敗、非 0 退出、手動中止或超時，皆將當下已收錄的輸出、耗時與 `error_reason` 寫入歷史檔，前端以不同色標（綠/紅/灰/橘）區分狀態並提供一鍵「重試執行」。

### 3.3 Go 後端模組與 RPC 介面

建議新增 `internal/composer/` 套件封裝核心邏輯，並於 `composer_bridge.go`（或 `bridge.go`）暴露給 Wails 前端：

* `RunComposerCommand(projName string, composerVer string, args []string) (string, error)`
  * 回傳 `runID`，非同步執行並透過 `composer_output`、`composer_exit` 事件推送串流。
  * 內建 Hard Timeout（600s）與 Idle Timeout（180s），結束後自動計算 `duration_ms`、`exit_code`、`status`（`success` / `error` / `stopped` / `timeout`）、`error_reason`、`original_bytes`、`original_lines` 並寫入歷史檔。
* `StopComposerCommand(runID string) error`
  * 主動終止執行中的 Composer 子行程並標記狀態為 `stopped`。
* `GetComposerHistory(projName string, offset int, limit int) (*ComposerHistoryPage, error)`
  * 由新到舊分頁讀取（預設 `limit = 5`），回傳 `{ Items, Total, HasMore }`。
* `ClearComposerHistory(projName string) error`
  * 手動清除指定專案的持久化歷史檔案。

### 3.4 安全防護規範（遵循 `AGENTS.md` §5）

1. **零 Shell 拼接防命令注入**：
   * 絕不呼叫 `cmd /C` 或 PowerShell 拼接指令字串。
   * 直接以 `exec.Command(phpExe, append([]string{"-d", "memory_limit=-1", composerPhar, "--ansi"}, args...)...)` 啟動，阻斷 `&&`、`|`、`;`、`>` 等元字元注入。
2. **路徑防遍歷與來源白名單**：
   * 專案名稱僅允許 `[a-zA-Z0-9_-]`，組裝歷史檔案路徑後驗證位於 `data/composer_history/` 目錄內。
   * `php.exe` 與 `composer.phar` 必須由 `scanner` 掃描結果解析，限定於專案 `bin/` 目錄下。
   * 工作目錄鎖定為專案物理根目錄。
3. **環境隔離與子行程清理**：
   * 僅對該次 `exec.Cmd.Env` 動態前置注入專案對應 PHP 目錄，不修改系統全域 PATH；設定 `COMPOSER_NO_INTERACTION=1`。
   * 子行程納入清理管理，避免面板關閉或中途取消後殘留孤兒行程。

### 3.5 Laravel 導向內建指令提示清單

| 分類 | 指令範例 | 說明 |
|------|----------|------|
| 依賴安裝 | `install` | 依 `composer.lock` 精準安裝套件 |
| 正式部署 | `install --no-dev --optimize-autoloader` | 排除開發套件並最佳化自動載入 |
| 依賴更新 | `update` / `update <pkg> -W` | 更新全部或指定套件（`-W` 含連帶依賴） |
| 套件增刪 | `require <pkg>` / `require --dev <pkg>` / `remove <pkg>` | 新增或移除套件 |
| 自動載入 | `dump-autoload -o` | 重建最佳化 classmap（解決 Laravel class not found） |
| 盤點檢視 | `show` / `show --tree` / `outdated` | 檢視已安裝套件、依賴樹與可更新項目 |
| 安全稽核 | `audit` | 掃描已安裝套件之已知安全漏洞（CVE） |
| 衝突追蹤 | `why <pkg>` / `why-not <pkg> <ver>` | 查詢套件為何被安裝或無法升級 |
| 驗證維護 | `validate` / `run-script --list` / `clear-cache` | 檢查 `composer.json`、列出腳本、清除快取 |

常用參數快速切換：`--no-dev`、`--ignore-platform-reqs`、`--dry-run`、`--prefer-dist`、`-W`、`--no-scripts`。

---

## 4. 待辦任務 (TODO)

### 4.1 掃描器與後端核心

- [ ] **增強 Composer 掃描器 (`internal/scanner/scanner.go`)**：
  - 掃描 `bin/composer/composer-*` 時同時支援辨識 `composer.phar` 與 `composer.bat`（補齊對應單元測試）。
- [ ] **建立 Composer 執行與歷史持久化套件 (`internal/composer/`)**：
  - 實作無 Shell 拼接的安全執行器（含即時 stdout/stderr 串流、Job Object 行程樹終止、Hard Timeout 600s + Idle Timeout 180s、環境變數隔離與 `COMPOSER_NO_INTERACTION=1` / `COMPOSER_PROCESS_TIMEOUT=600`）。
  - 實作串流 Head（8 KB）+ Tail（24 KB）安全緩衝（對齊 UTF-8 / 換行 / ANSI 重置，累計 `original_bytes` 與 `original_lines`）。
  - 實作 `data/composer_history/<project>.jsonl` 原子讀寫、單專案 50 筆上限淘汰、分頁查詢與手動清除。
  - 補齊單元測試（路徑防遍歷、頭尾截斷與 UTF-8 邊界、原始字數統計、Hard/Idle Timeout 觸發、上限淘汰、分頁邊界）。
- [ ] **新增 Wails RPC 橋接 (`composer_bridge.go`) 與 `.gitignore`**：
  - 暴露 `RunComposerCommand`、`StopComposerCommand`、`GetComposerHistory`、`ClearComposerHistory` 並推送 `composer_output` / `composer_exit` 事件。
  - 於 `.gitignore` 新增 `/data/composer_history/`。

### 4.2 前端介面與互動

- [ ] **建立 Composer 控制台元件 (`frontend/src/components/ComposerModal.tsx`)**：
  - 上方環境列：顯示跟隨專案的 PHP 版本、可切換的 Composer 版本選單（偵測多版本時啟用下拉）與「清除歷史」按鈕（透過 `customConfirm` 確認）。
  - 結構化 ANSI 彩色輸出區：開啟時預設顯示最新 5 筆執行紀錄（含時間、耗時、版本、Exit Code），支援**向上滾動自動分頁載入**更早紀錄並保持捲動位置不跳動。
  - 簡化指令輸入列：固定 `composer >` 前綴，支援即時指令／參數篩選提示（聚焦 Laravel 常用指令）、快捷參數切換與上／下鍵指令歷史。
  - 嚴格使用 `EventsOn` 回傳的 `unsubscribe` 清理事件監聽，不使用全域 `EventsOff`。
- [ ] **串接專案列表與多國語言 (`frontend/src/components/Projects.tsx`, `frontend/src/i18n.ts`, `internal/i18n/i18n.go`)**：
  - 於 `...` 操作選單針對 `laravel` / `php` 專案加入 `Composer` 入口。
  - 補齊所有新增介面與錯誤訊息之中英文 i18n 字典（全站無 Emoji）。

### 4.3 驗證與測試

- [ ] **自動化驗證**：執行 `npx tsc --noEmit` 與 `go test ./...` 確認零錯誤，並確認無編譯產物留存於暫存區。
- [ ] **手動流程驗證**：測試多版本 PHP 切換、執行 `show` / `audit` / `dump-autoload -o`、重開控制台載入最新 5 筆、向上滾動載入舊紀錄、中途停止與手動清除歷史。
