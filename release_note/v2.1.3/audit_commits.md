# Release Audit Checklist: WinCMP v2.1.3

> **審核說明**：本檔案為發行前代碼變更之真實性、完整性核對底稿。
> 完成核對後請一併提交至 Git 倉庫歸檔留存。

- **比對基準**：`v2.1.2..HEAD`
- **發行日期**：2026-09-20
- **發布分支**：`feature/version-v2.1.3`

---

## 1. 變更條目與代碼核對矩陣 (Audit Matrix)

### [Added]
- [x] **終端日誌右鍵選單**
  - **實際影響**：日誌區支援複製／複製此行／全選／開啟當天分類日誌檔；無檔案時停用並顯示路徑提示。
  - **關聯 Commit**：`4cf5bf2` `feat(core): v2.1.3 backend APIs for updater, redis explorer, and log files`；`734bd74` `feat(terminal): runtime project dropdown and log context menu`
  - **核心變更檔案**：`bridge.go`（`GetCategoryLogFilePath` / `OpenCategoryLogFile` / `OpenPathInExplorer`）, `frontend/src/components/TerminalLogs.tsx`
  - **查驗指令**：`git show 4cf5bf2 --stat`；`git show 734bd74 --stat`

- [x] **Runtime 專案日誌選單提示**
  - **實際影響**：自訂專案下拉，其他專案有新 log 時顯示圓點。
  - **關聯 Commit**：`734bd74` `feat(terminal): runtime project dropdown and log context menu`
  - **核心變更檔案**：`frontend/src/components/TerminalLogs.tsx`
  - **查驗指令**：`git show 734bd74 -- frontend/src/components/TerminalLogs.tsx`

### [Changed]
- [x] **資料庫瀏覽器工具列與 Redis 版面**
  - **實際影響**：按鈕對齊、Redis 隱藏 HeidiSQL、連線/DB 移至分頁旁、搜尋移入 Keys 列、標題列高度固定。
  - **關聯 Commit**：`25f3230` `refactor(redis): restyle database explorer toolbar and key panel`
  - **核心變更檔案**：`frontend/src/components/DBExplorer.tsx`
  - **查驗指令**：`git show 25f3230 --stat`

- [x] **終端日誌 Node / Custom**
  - **實際影響**：分頁標籤由 Node / Bun 改為 Node / Custom。
  - **關聯 Commit**：`734bd74` `feat(terminal): runtime project dropdown and log context menu`
  - **核心變更檔案**：`frontend/src/components/TerminalLogs.tsx`, `frontend/src/i18n.ts`
  - **查驗指令**：`git show 734bd74 -- frontend/src/components/TerminalLogs.tsx`

- [x] **設定文案與依賴檢查刷新本機**
  - **實際影響**：自動檢查文案調整；依賴「檢查更新」會清掃描快取並重新掃描本機環境。
  - **關聯 Commit**：`0f82eed` `feat(ui): update-check labels, badge sync, and dependency rescan`；`4cf5bf2`（`ScanServices` 清快取）
  - **核心變更檔案**：`frontend/src/components/Settings.tsx`, `frontend/src/components/DependencyManager.tsx`, `bridge.go`
  - **查驗指令**：`git show 0f82eed --stat`；`git show 4cf5bf2 -- bridge.go`

### [Removed]
- [x] **Redis 清空 DB / 刪除鍵值**
  - **實際影響**：移除 FLUSHDB 與單鍵 DEL 之 UI 與 Go API，降低誤操作風險。
  - **關聯 Commit**：`4cf5bf2` `feat(core): v2.1.3 backend APIs...`；`25f3230` `refactor(redis): restyle database explorer toolbar and key panel`
  - **核心變更檔案**：`internal/redisexplorer/explorer.go`, `bridge.go`, `frontend/src/components/DBExplorer.tsx`, `frontend/wailsjs/go/main/App.*`
  - **查驗指令**：`git show 4cf5bf2 -- internal/redisexplorer/explorer.go bridge.go`

### [Fixed]
- [x] **Redis 鍵名搜尋**
  - **實際影響**：自動補 `*` 與多頁 SCAN，避免搜尋無結果的假陰性。
  - **關聯 Commit**：`25f3230` `refactor(redis): restyle database explorer toolbar and key panel`
  - **核心變更檔案**：`frontend/src/components/DBExplorer.tsx`
  - **查驗指令**：`git show 25f3230 -- frontend/src/components/DBExplorer.tsx`

- [x] **版本更新提示與紅點持久化**
  - **實際影響**：啟動必查、預設 6 小時間隔、`has_update_available` 寫入設定並同步側邊欄呼吸燈。
  - **關聯 Commit**：`4cf5bf2` `feat(core): v2.1.3 backend APIs...`；`0f82eed` `feat(ui): update-check labels, badge sync, and dependency rescan`
  - **核心變更檔案**：`app.go`, `bridge.go`, `internal/config/config.go`, `frontend/src/App.tsx`, `frontend/src/components/VersionUpdate.tsx`
  - **查驗指令**：`git show 4cf5bf2 -- app.go internal/config/config.go`

- [x] **Redis 未啟動時重新整理圖示轉動**
  - **實際影響**：連線探測不再視為載入中，未啟動時圖示不轉。
  - **關聯 Commit**：`25f3230` `refactor(redis): restyle database explorer toolbar and key panel`
  - **核心變更檔案**：`frontend/src/components/DBExplorer.tsx`
  - **查驗指令**：`git show 25f3230 -- frontend/src/components/DBExplorer.tsx`

---

## 2. 未納入發布說明的 Commit 查驗

| Commit | 類型 | 標題 | 未納入原因 |
| :--- | :--- | :--- | :--- |
| `bb2228c` | `refactor(release)!` | drop release_info.json in favor of VERSION and notes date lines | 發布流程與 validator 調整，非終端使用者功能 |
| `334fd88` | `docs` | add v2.1.3 plan and remove outdated docs | 開發規格文件，功能已拆入上方條目 |

---

## 3. 發布前動作確認清單
- [x] 1. 條目代碼核對通過
- [x] 2. `node .agents/skills/wincmp-release/scripts/validate_release.js` 通過
- [x] 3. `cd frontend && npm run build` 通過
- [x] 4. `go test ./...` 通過
- [x] 5. `wails dev` 已啟動且 `scripts/capture_release_screenshots.ps1` 完成
- [x] 6. `.\release.bat` 成功生成 Release 壓縮包
