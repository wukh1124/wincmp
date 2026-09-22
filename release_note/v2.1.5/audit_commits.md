# Release Audit Checklist: WinCMP v2.1.5

> **審核說明**：本檔案為發行前代碼變更之真實性、完整性核對底稿。
> 完成核對後請一併提交至 Git 倉庫歸檔留存。

- **比對基準**：`v2.1.4..HEAD`
- **發行日期**：2026-09-22
- **發布分支**：`feature/version-v2.1.5`

---

## 1. 變更條目與代碼核對矩陣 (Audit Matrix)

### [Added]
- [x] **終端日誌未讀行數提示**
  - **實際影響**：各分類分頁標籤與運行專案 Hover Tooltip 增強顯示未讀日誌行數（如：`Caddy - 共 4 行 (未讀 2 行)`），點選切換分頁或專案時即時清除標記。
  - **關聯 Commit**：`860e5a0` `feat(logs): show unread count in log tab tooltip and polish dependency update UX`
  - **核心變更檔案**：`frontend/src/components/TerminalLogs.tsx`, `frontend/src/i18n.ts`, `internal/i18n/i18n.go`
  - **查驗指令**：`git show 860e5a0 -- frontend/src/components/TerminalLogs.tsx`

- [x] **版本更新即時檢查與手動刷新**
  - **實際影響**：版本更新頁面新增「檢查更新」按鈕並提供平滑旋轉動畫，進入頁面時若逾 60 秒自動連線取得最新發布資訊，支援穿透快取以避免跳版更新時需升級兩次。
  - **關聯變更**：`frontend/src/components/VersionUpdate.tsx`, `bridge.go`, `internal/updater/updater.go`, `frontend/src/i18n.ts`, `internal/i18n/i18n.go`
  - **查驗指令**：`git diff -- frontend/src/components/VersionUpdate.tsx bridge.go internal/updater/updater.go`

### [Changed]
- [x] **依賴管理更新互動體驗最佳化**
  - **實際影響**：開啟依賴庫視窗時改採背景靜默自動檢查，手動點擊「檢查更新」保證至少 500ms 平滑動畫過渡；移除阻斷式提示彈窗，並在設定實質更新時以平滑淡入淡出切換畫面。
  - **關聯 Commit**：`860e5a0` `feat(logs): show unread count in log tab tooltip and polish dependency update UX`
  - **核心變更檔案**：`frontend/src/components/DependencyManager.tsx`
  - **查驗指令**：`git show 860e5a0 -- frontend/src/components/DependencyManager.tsx`

### [Fixed]
- [x] **Windows 檔案總管路徑與定位異常**
  - **實際影響**：修復在檔案總管中開啟檔案時，因參數逸出包裹引號導致退回「我的文件」目錄的問題；改採原生命令列傳遞，且當目標日誌尚未生成時安全降級開啟所在目錄。
  - **關聯 Commit**：`c006544` `fix(logs): resolve explorer fallback to documents on open containing folder`
  - **核心變更檔案**：`app.go`, `app_test.go`, `bridge.go`
  - **查驗指令**：`git show c006544 --stat`

- [x] **版本更新呼吸燈啟動閃爍消除**
  - **實際影響**：啟動新版本時預設保持平靜，待背景檢查確認有新版後才點亮紅點；啟動與更新重啟時自動校驗並清理過期更新標記，徹底解決剛升級完呼吸燈閃現又自動消失的突兀體驗。
  - **關聯變更**：`frontend/src/App.tsx`, `app.go`, `bridge.go`
  - **查驗指令**：`git diff -- frontend/src/App.tsx app.go bridge.go`

---

## 2. 未納入發布說明的 Commit 查驗

| Commit | 類型 | 標題 | 未納入原因 |
| :--- | :--- | :--- | :--- |
| `860e5a0` (部分文件) | `docs` | `docs/v2.1.5_plan.md` | 開發與規劃文件，功能已拆入上方條目 |

---

## 3. 發布前動作確認清單
- [x] 1. 條目代碼核對通過
- [x] 2. `node .agents/skills/wincmp-release/scripts/validate_release.js` 通過
- [x] 3. `cd frontend && npm run build` 通過
- [x] 4. `go test ./...` 通過
- [ ] 5. `wails dev` 已啟動且 `node scripts/capture.js` 完成
- [ ] 6. `.\release.bat` 成功生成 Release 壓縮包
