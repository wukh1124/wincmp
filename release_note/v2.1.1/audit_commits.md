# Release Audit Checklist: WinCMP v2.1.1

> **審核說明**：本檔案為發行前代碼變更之真實性、完整性核對底稿。
> 完成核對後請一併提交至 Git 倉庫歸檔留存。

- **比對基準**：`v2.1.0..HEAD`
- **發行日期**：2026-09-19
- **發布分支**：`feature/version-2.1.1`

---

## 1. 變更條目與代碼核對矩陣 (Audit Matrix)

### [Fixed]
- [x] **終端啟動工作目錄回退**
  - **實際影響**：專案路徑遺失或被移動時，終端啟動不再因無效目錄失敗；自動依序回退到可用路徑，並解析有效的 Shell 絕對路徑，讓專案終端仍可正常開啟。
  - **關聯 Commit**：`6aaa461` `fix(terminal): fallback working directory when project path is missing`
  - **核心變更檔案**：`bridge.go`, `internal/terminal/path.go`, `internal/terminal/path_test.go`, `internal/terminal/terminal.go`
  - **查驗指令**：`git show 6aaa461 --stat`

- [x] **資料庫瀏覽器選取樣式**
  - **實際影響**：資料庫清單選中項目不再使用深色實心填色，改為淺色底與 accent 文字；sketch 主題亦調整為不透明淺藍紙色，避免格線穿透並維持可讀性。
  - **關聯 Commit**：`18011d8` `fix(db-explorer): stop selected schema item from using solid accent fill`
  - **核心變更檔案**：`frontend/src/components/DBExplorer.tsx`, `frontend/src/style.css`
  - **查驗指令**：`git show 18011d8 --stat`

### [Dependencies]
- [x] **預設依賴版本更新**
  - **實際影響**：更新 Caddy、Composer、HeidiSQL、Mailpit、Node.js、PHP 8.2／8.3／8.4 與 php_redis 等預設依賴版本，並補齊 SHA-256 校驗值，提升依賴下載與驗證可靠性。
  - **關聯 Commit**：`d806800` `chore(release): adopt release_note as single source and fix dependency validation`
  - **核心變更檔案**：`conf/dependencies.json`, `scripts/check_deps.go`
  - **查驗指令**：`git show d806800 --stat`
  - **備註**：同 commit 亦調整 release 流程（release_note 單一來源、release.ps1、截圖腳本）；對外僅保留依賴版本與校驗值部分。

---

## 2. 未納入發布說明的 Commit 查驗

| Commit | 類型 | 標題 | 未納入原因 |
| :--- | :--- | :--- | :--- |
| `d806800` | `chore` | chore(release): adopt release_note as single source and fix dependency validation | 發布流程內部調整；依賴更新已獨立列入 Dependencies |
| `0deda81` | `fix` | fix(screenshot): force font size S and English UI during release captures | 發行截圖工具內部調整，非終端使用者功能 |
| `42d39ca` | `chore` | chore(screenshot): update new screenshot for v2.1.0 | 截圖資產更新，非功能變更 |
| `a090d2e` | `docs` | docs(website): rewrite comparison as current-version tech specs and align feature copy | 官網文案 |
| `fc6ecc0` | `docs` | docs(readme): refresh readmes for v2.1.0 and keep packaging guide accurate | 專案文件 |
| `76eabd8` | `docs` | docs: align dependencies.md with conf/dependencies.json and remove updated docs | 專案文件 |
| `82b94e6` | `docs` | docs(agents): consolidate agent guide into root AGENTS.md | 開發者指引 |

---

## 3. 發布前動作確認清單

- [x] 1. 條目代碼核對通過
- [x] 2. `node .agents/skills/wincmp-release/scripts/validate_release.js` 通過
- [x] 3. `cd frontend && npm run build` 通過
- [x] 4. `go test ./...` 通過
- [ ] 5. `wails dev` 已啟動且 `scripts/capture_release_screenshots.ps1` 完成（本次由開發者手動執行）
- [ ] 6. `.\release.bat` 成功生成 Release 壓縮包（本次由開發者手動執行）
