# Release Audit Checklist: WinCMP v2.1.4

> **審核說明**：本檔案為發行前代碼變更之真實性、完整性核對底稿。
> 完成核對後請一併提交至 Git 倉庫歸檔留存。

- **比對基準**：`v2.1.3..HEAD`
- **發行日期**：2026-09-22
- **發布分支**：`feature/version-v2.1.4`

---

## 1. 變更條目與代碼核對矩陣 (Audit Matrix)

### [Dependencies]
- [x] **PHP 7.4 與 Redis 擴充支援**
  - **實際影響**：依賴庫加入 PHP 7.4.33 與 PECL Redis 5.3.7，支援下載解壓自動移至 ext 目錄，並配置 Caddy upstream 與 Laravel 推薦版本。
  - **關聯 Commit**：`9e8f6df` `feat(deps): add support for PHP 7.4 NTS and PECL Redis extension`
  - **核心變更檔案**：`conf/dependencies.json`, `downloader_bridge.go`, `internal/config/default_conf/dependencies.json`, `conf/wincmp.json.example`, `conf/snippets/php-upstream.caddy.example`, `docs/dependencies.md`
  - **查驗指令**：`git show 9e8f6df --stat`

### [Security]
- [x] **軟體更新與依賴下載防禦**
  - **實際影響**：依賴下載與主程式更新強制實施官方網域白名單、SHA-256 雜湊與 PE 標頭驗證；更新失敗自動回滾舊版執行檔並清理暫存檔，下載異常時提供來源連結與手動排查導引。
  - **關聯 Commit**：
    - `0f5e153` `feat(security): enforce mandatory sha256 validation and official domain whitelist for dependencies`
    - `4b66696` `feat(security): implement release url whitelist, sha256 verification and pe validation for updater`
    - `8562b3e` `fix(updater): add automatic rollback on launch failure, cleanup on error and manual fallback guidance`
    - `78d039e` `feat(downloader): enrich security check error messages with dependency url and diagnostic guidance`
  - **核心變更檔案**：`internal/downloader/downloader.go`, `internal/updater/updater.go`, `downloader_bridge.go`, `frontend/src/i18n.ts`, `internal/i18n/i18n.go`
  - **查驗指令**：`git show 0f5e153 4b66696 8562b3e 78d039e --stat`

### [Added]
- [x] **在檔案總管中顯示日誌**
  - **實際影響**：日誌右鍵選單新增「在檔案總管中顯示」，呼叫 Explorer 開啟該檔案所在資料夾並選取。
  - **關聯 Commit**：`cf35ed1` `fix(logs): resolve cmd console flicker on open file and add reveal in explorer action`
  - **核心變更檔案**：`bridge.go` (`OpenPathInExplorer`), `frontend/src/components/TerminalLogs.tsx`, `internal/i18n/i18n.go`, `frontend/src/i18n.ts`
  - **查驗指令**：`git show cf35ed1 --stat`

- [x] **日誌分頁行數統計提示**
  - **實際影響**：分頁標籤滑鼠懸停顯示服務名稱與緩衝日誌行數（如：`Caddy - 共 4 行`）。
  - **關聯 Commit**：`711934a` `feat(ui): refine DB explorer layout, add terminal log counters, and auto-check dependencies`
  - **核心變更檔案**：`frontend/src/components/TerminalLogs.tsx`
  - **查驗指令**：`git show 711934a -- frontend/src/components/TerminalLogs.tsx`

- [x] **依賴管理自動檢查與手動複製**
  - **實際影響**：進入依賴管理頁面時自動背景檢查更新（含冷卻防抖），並於按鈕組新增「複製下載連結」以利手動下載與排查。
  - **關聯 Commit**：
    - `711934a` `feat(ui): refine DB explorer layout, add terminal log counters, and auto-check dependencies`
    - `e28652a` `feat(dependency): unify split buttons with copy link and support dynamic i18n diagnostics`
  - **核心變更檔案**：`frontend/src/components/DependencyManager.tsx`, `frontend/src/i18n.ts`, `internal/i18n/i18n.go`
  - **查驗指令**：`git show 711934a e28652a -- frontend/src/components/DependencyManager.tsx`

### [Changed]
- [x] **資料庫瀏覽器佈局最佳化**
  - **實際影響**：切換器移至頂部標題列、連線資訊改 Hover Tooltip、刷新/HeidiSQL/Redis 搜尋/DB 選擇器就近歸位、標題列高度統一 48px，並強化重新整理按鈕回饋。
  - **關聯 Commit**：
    - `711934a` `feat(ui): refine DB explorer layout, add terminal log counters, and auto-check dependencies`
    - `3e0f6c8` `feat(ui): enhance refresh button visual feedback and tab sync behavior`
  - **核心變更檔案**：`frontend/src/components/DBExplorer.tsx`, `frontend/src/components/Dashboard.tsx`
  - **查驗指令**：`git show 711934a 3e0f6c8 --stat`

- [x] **系統設定輔助說明完善**
  - **實際影響**：精簡設定項目開關標題，並補齊下方輔助說明文案與中英文翻譯字典。
  - **關聯 Commit**：`b3f6fbf` `feat(settings): update settings system title and desc text`
  - **核心變更檔案**：`frontend/src/components/Settings.tsx`, `frontend/src/i18n.ts`, `internal/i18n/i18n.go`
  - **查驗指令**：`git show b3f6fbf --stat`

### [Fixed]
- [x] **儀表板頁面切換閃爍**
  - **實際影響**：利用模組快取與自適應骨架屏消除切換頁面時控制按鈕與載入狀態的短暫閃爍。
  - **關聯 Commit**：`5454a6c` `fix(dashboard): eliminate button flicker on navigation with module cache and adaptive skeleton`
  - **核心變更檔案**：`frontend/src/components/Dashboard.tsx`
  - **查驗指令**：`git show 5454a6c --stat`

- [x] **Windows 檔案鎖定與程序警告**
  - **實際影響**：增加依賴目錄重新命名重試機制以應對 Windows 檔案暫態鎖定造成的存取被拒；抑制停止 runtime 服務時產生的偽陽性存取被拒與 taskkill 128 警告日誌。
  - **關聯 Commit**：
    - `c424f00` `fix(downloader): add retry mechanism for directory renaming to prevent transient access denied on windows`
    - `1d55fc1` `fix(process): suppress false-positive access denied and taskkill 128 warnings when stopping runtime`
  - **核心變更檔案**：`internal/downloader/downloader.go`, `internal/process/manager.go`
  - **查驗指令**：`git show c424f00 1d55fc1 --stat`

- [x] **全新安裝 Caddy 設定缺失**
  - **實際影響**：全新解壓縮執行時自動釋出內嵌 snippets，避免 Caddy 因找不到 common.caddy 或 php-upstream.caddy 而啟動失敗。
  - **關聯 Commit**：`1b526e1` `fix(config): embed default caddy snippets and update ignore rules`
  - **核心變更檔案**：`internal/config/default_conf/snippets/*`, `internal/config/config_test.go`
  - **查驗指令**：`git show 1b526e1 --stat`

- [x] **日誌與選單顯示問題**
  - **實際影響**：解決開啟日誌命令提示字元黑窗閃爍與特定路徑呼叫失敗；修正 Carbon 暗色主題下右鍵選單背景透明度與文字穿透。
  - **關聯 Commit**：
    - `cf35ed1` `fix(logs): resolve cmd console flicker on open file and add reveal in explorer action`
    - `711934a` `feat(ui): refine DB explorer layout, add terminal log counters, and auto-check dependencies`
  - **核心變更檔案**：`bridge.go`, `frontend/src/components/TerminalLogs.tsx`, `frontend/src/themes.css`
  - **查驗指令**：`git show cf35ed1 711934a --stat`

---

## 2. 未納入發布說明的 Commit 查驗

| Commit | 類型 | 標題 | 未納入原因 |
| :--- | :--- | :--- | :--- |
| `076bb8b` | `docs` | add v2.1.4 plan | 開發與規劃文件，功能已拆入上方條目 |
| `693acb1` | `chore` | bump version to v2.1.4 | 初版版號標記與版本元數據提交 |
| `c4a159b` | `refactor` | migrate screenshot and website sync to node.js and update guidelines | 內部維護腳本遷移至 Node.js，無對外使用者操作影響 |
| `e330d3a` | `docs` | streamline AGENTS.md instructions and focus on core constraints | 內部 Agent 開發規範調整，無代碼或運行時影響 |
| `41a53dd` | `refactor` | consolidate single source of truth in conf and isolate dev runtime | 內部設定檔真相源整合與 dev runtime 隔離，行為保持向後相容 |
| `4a33a9a` | `fix` | isolate wails dev runtime to build/dev to prevent release clean | 內部開發環境執行檔輸出目錄隔離，不影響發行版本與使用者 |

---

## 3. 發布前動作確認清單
- [x] 1. 條目代碼核對通過
- [x] 2. `node .agents/skills/wincmp-release/scripts/validate_release.js` 通過
- [x] 3. `cd frontend && npm run build` 通過
- [x] 4. `go test ./...` 通過
- [x] 5. `wails dev` 已啟動且 `node scripts/capture.js` 完成
- [x] 6. `.\release.bat` 成功生成 Release 壓縮包
