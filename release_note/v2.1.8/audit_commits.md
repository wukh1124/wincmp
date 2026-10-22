# Release Audit Checklist: WinCMP v2.1.8

> **審核說明**：本檔案為發行前代碼變更之真實性、完整性核對底稿。

- **比對基準**：`v2.1.7..HEAD`
- **發行日期**：2026-10-22

## 1. 變更條目與代碼核對矩陣 (Audit Matrix)

### [Fixed]

- [x] **舊版 PHP 清理不再被檔案鎖擋死**
  - **實際影響**：`RemoveStalePHPPatch` 在 `os.RemoveAll` 前先停服務、終止目錄下殘留的 `php-cgi`，並以 3 次 / 200ms 重試；錯誤訊息會帶佔用行程數。
  - **關聯 Commit**：`288bb3b` `fix(downloader): terminate lingering php-cgi before removing stale PHP dir`
  - **核心變更檔案**：`downloader_bridge.go`、`internal/process/procscan.go`、`internal/i18n/i18n.go`
  - **查驗指令**：`git show 288bb3b --stat`

- [x] **升級時立即停掉被取代的版本**
  - **實際影響**：`runDependencyDownloadPipeline` 安裝完成重掃後呼叫 `reconcilePHPRestoreState`，停止 `SkippedPHP` 版本並清除其自動啟動紀錄。
  - **關聯 Commit**：`39d94a6` `fix(php): prune restore state and stop superseded runtimes after upgrade`
  - **核心變更檔案**：`app.go`、`downloader_bridge.go`
  - **查驗指令**：`git show 39d94a6 --stat`

- [x] **還原狀態自動校正**
  - **實際影響**：啟動掃描後執行 `reconcilePHPRestoreState`，移除不在 `PHPList` 的 `last_service_state.php` 條目；修正幽靈行程被反覆自動啟動。
  - **關聯 Commit**：`39d94a6` `fix(php): prune restore state and stop superseded runtimes after upgrade`
  - **核心變更檔案**：`app.go`、`app_test.go`
  - **查驗指令**：`git show 39d94a6 -- app.go app_test.go`

### [Changed]

- [x] **建置入口行為調整（Taskfile）**
  - **實際影響**：建置前偵測執行中的 `WinCMP` 直接失敗、`clean` 後驗證 `build/bin` 已清空、移除 `sh:` 版號依賴、新增 `task doctor`。
  - **關聯 Commit**：`1f2f16a` `chore(tooling): inline Taskfile commands and define script policy tiers`
  - **核心變更檔案**：`Taskfile.yml`、`AGENTS.md`
  - **查驗指令**：`git show 1f2f16a -- Taskfile.yml`

## 2. 未納入發布說明的 Commit 查驗

| Commit | 類型 | 標題 | 未納入原因 |
| :--- | :--- | :--- | :--- |
| `ef8ef89` | docs | docs: pin Wails requirement and clarify Go minimum | 僅開發文件（Wails 版本、Go 最低版本、`task doctor`、legacy 標註），不影響使用者行為 |

## 3. 發布前動作確認清單

- [x] 1. 條目代碼核對通過
- [ ] 2. `node .agents/skills/wincmp-release/scripts/validate_release.js` 通過
- [ ] 3. `cd frontend && npm run build` 通過
- [ ] 4. `go test ./...` 通過（目前僅 `TestOpenFolderAndShowInExplorerValidation` 因環境缺 `ShellExecute` 失敗，待修）
- [ ] 5. `wails dev` 已啟動且截圖完成
- [ ] 6. `.\release.bat` 成功生成 Release 壓縮包
