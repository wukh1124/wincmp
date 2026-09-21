# Release Audit Checklist: WinCMP v2.1.4

> **審核說明**：本檔案為發行前代碼變更之真實性、完整性核對底稿。
> 完成核對後請一併提交至 Git 倉庫歸檔留存。

- **比對基準**：`v2.1.3..HEAD`
- **發行日期**：2026-09-21
- **發布分支**：`feature/version-v2.1.4`

---

## 1. 變更條目與代碼核對矩陣 (Audit Matrix)

### [Dependencies]
- [x] **PHP 7.4 與 Redis 擴充支援**
  - **實際影響**：依賴庫加入 PHP 7.4.33 與 PECL Redis 5.3.7，支援下載解壓自動移至 ext 目錄，並配置 Caddy upstream 與 Laravel 推薦版本。
  - **關聯 Commit**：`9e8f6df` `feat(deps): add support for PHP 7.4 NTS and PECL Redis extension`
  - **核心變更檔案**：`conf/dependencies.json`, `downloader_bridge.go`, `internal/config/default_conf/dependencies.json`, `conf/wincmp.json.example`, `conf/snippets/php-upstream.caddy.example`, `docs/dependencies.md`
  - **查驗指令**：`git show 9e8f6df --stat`

### [Added]
- [x] **在檔案總管中顯示日誌**
  - **實際影響**：日誌右鍵選單新增「在檔案總管中顯示」，呼叫 Explorer 開啟該檔案所在資料夾並選取。
  - **關聯 Commit**：`cf35ed1` `fix(logs): resolve cmd console flicker on open file and add reveal in explorer action`
  - **核心變更檔案**：`bridge.go` (`OpenPathInExplorer`), `frontend/src/components/TerminalLogs.tsx`, `internal/i18n/i18n.go`, `frontend/src/i18n.ts`
  - **查驗指令**：`git show cf35ed1 --stat`

- [x] **日誌分頁行數統計提示**
  - **實際影響**：分頁標籤滑鼠懸停顯示服務名稱與緩衝日誌行數（如：Caddy - 共 4 行）。
  - **關聯 Commit**：`711934a` `feat(ui): refine DB explorer layout, add terminal log counters, and auto-check dependencies`
  - **核心變更檔案**：`frontend/src/components/TerminalLogs.tsx`
  - **查驗指令**：`git show 711934a -- frontend/src/components/TerminalLogs.tsx`

- [x] **依賴管理自動檢查**
  - **實際影響**：進入依賴管理頁面時自動觸發背景檢查，並具備 5 分鐘冷卻防抖。
  - **關聯 Commit**：`711934a` `feat(ui): refine DB explorer layout, add terminal log counters, and auto-check dependencies`
  - **核心變更檔案**：`frontend/src/components/DependencyManager.tsx`
  - **查驗指令**：`git show 711934a -- frontend/src/components/DependencyManager.tsx`

### [Changed]
- [x] **資料庫瀏覽器佈局最佳化**
  - **實際影響**：切換器移至頂部標題列、連線資訊改 Hover Tooltip、刷新/HeidiSQL/Redis 搜尋/DB 選擇器就近歸位、標題列高度統一 48px。
  - **關聯 Commit**：`711934a` `feat(ui): refine DB explorer layout, add terminal log counters, and auto-check dependencies`
  - **核心變更檔案**：`frontend/src/components/DBExplorer.tsx`
  - **查驗指令**：`git show 711934a -- frontend/src/components/DBExplorer.tsx`

- [x] **系統設定輔助說明完善**
  - **實際影響**：精簡設定項目開關標題，並補齊下方輔助說明文案與中英文翻譯字典。
  - **關聯 Commit**：`b3f6fbf` `feat(settings): update settings system title and desc text`
  - **核心變更檔案**：`frontend/src/components/Settings.tsx`, `frontend/src/i18n.ts`, `internal/i18n/i18n.go`
  - **查驗指令**：`git show b3f6fbf --stat`

### [Fixed]
- [x] **全新安裝 Caddy 設定缺失**
  - **實際影響**：全新解壓縮執行時自動釋出內嵌 snippets，避免 Caddy 因找不到 common.caddy 或 php-upstream.caddy 而啟動失敗。
  - **關聯 Commit**：`1b526e1` `fix(config): embed default caddy snippets and update ignore rules`
  - **核心變更檔案**：`internal/config/default_conf/snippets/*`, `internal/config/config_test.go`
  - **查驗指令**：`git show 1b526e1 --stat`

- [x] **日誌開啟命令列視窗閃爍與呼叫失敗**
  - **實際影響**：以 HideWindow / 直接調用系統關聯程式解決 cmd 黑窗閃爍，並支援包含空格或引號之特殊路徑。
  - **關聯 Commit**：`cf35ed1` `fix(logs): resolve cmd console flicker on open file and add reveal in explorer action`
  - **核心變更檔案**：`bridge.go`, `frontend/src/components/TerminalLogs.tsx`
  - **查驗指令**：`git show cf35ed1 -- bridge.go`

- [x] **暗色主題右鍵選單穿透**
  - **實際影響**：Carbon 主題下右鍵選單與下拉選單增加實體背景色與陰影，改善對比與文字穿透。
  - **關聯 Commit**：`711934a` `feat(ui): refine DB explorer layout, add terminal log counters, and auto-check dependencies`
  - **核心變更檔案**：`frontend/src/themes.css`
  - **查驗指令**：`git show 711934a -- frontend/src/themes.css`

---

## 2. 未納入發布說明的 Commit 查驗

| Commit | 類型 | 標題 | 未納入原因 |
| :--- | :--- | :--- | :--- |
| `076bb8b` | `docs` | add v2.1.4 plan | 開發與規劃文件，功能已拆入上方條目 |

---

## 3. 發布前動作確認清單
- [x] 1. 條目代碼核對通過
- [x] 2. `node .agents/skills/wincmp-release/scripts/validate_release.js` 通過
- [x] 3. `cd frontend && npm run build` 通過
- [x] 4. `go test ./...` 通過
- [x] 5. `wails dev` 已啟動且 `scripts/capture_release_screenshots.ps1` 完成
- [x] 6. `.\release.bat` 成功生成 Release 壓縮包
