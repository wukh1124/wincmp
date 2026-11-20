# Release Audit Checklist: WinCMP v2.1.10

> **審核說明**：本檔案為發行前代碼變更之真實性、完整性核對底稿。

- **比對基準**：`v2.1.9..HEAD`（4 筆 commits：`3a7adc0`、`5bfebf1`、`3ed0b25`、`0e51bfc`）
- **發行日期**：2026-11-20

## 1. 變更條目與代碼核對矩陣 (Audit Matrix)

### [Changed]
- [x] **固定執行檔名稱為 `WinCMP.exe`**
  - **實際影響**：`bat/release.ps1` 將壓縮包內執行檔改為固定 `WinCMP.exe`（父目錄獨立自動更新檔仍保留 `WinCMP_v$Version.exe`）；`internal/updater/updater.go` 新增 `FixedExecutableName = "WinCMP.exe"` 與 `ReplaceExecutable`，更新落地一律使用 `WinCMP.exe`，維持 Windows 工作列釘選與捷徑有效，並於 `updater_test.go` 補齊測試。
  - **關聯 Commit**：`0e51bfc` `feat(updater): use fixed WinCMP.exe filename on update to preserve taskbar pin`
  - **核心變更檔案**：`internal/updater/updater.go`、`internal/updater/updater_test.go`、`bat/release.ps1`、`packaging/wincmp/readme.md`、`packaging/wincmp/readme_zh.md`、`.agents/skills/wincmp-release/SKILL.md`
  - **查驗指令**：`git show 0e51bfc --stat`

### [Dependencies]
- [x] **更新內建相依套件目錄**
  - **實際影響**：更新 `conf/dependencies.json` 內 Caddy (`2.11.9`)、MariaDB (`11.4.14`)、PHP (`8.2.36`、`8.3.37`、`8.4.28`、`8.5.13`)、Mailpit (`1.34.0`)、Node.js (`24.22.3`)、Composer (`2.11.2`)、HeidiSQL (`12.22`) 與 `cacert` (`2026-11-05`) 之版本、下載網址與 SHA-256 校驗碼，並同步 `docs/dependencies.md`。
  - **關聯 Commit**：`3ed0b25` `chore(deps): update bundled dependencies catalog and sync docs`
  - **核心變更檔案**：`conf/dependencies.json`、`docs/dependencies.md`、`.agents/skills/wincmp-dependencies-update/SKILL.md`
  - **查驗指令**：`git show 3ed0b25 -- docs/dependencies.md`

### [Security]
- [x] **改用 Go 1.27.4 編譯**
  - **實際影響**：`go.mod` 的 `go` 指令由 `1.27.3` 提升為 `1.27.4`，修補 GO-2026-6859（`net/http` 的 `MaxBytesReader` 標頭競態導致伺服器 crash）。CI 的 `actions/setup-go@v7` 以 `go-version-file: go.mod` 解析版本，因此 v2.1.10 起的執行檔由 Go 1.27.4 編譯；README 徽章與環境需求段落同步改為硬性最低 1.27.4。
  - **觸發來源**：v2.1.10 tag 的 Build and Release run 由 `govulncheck` 在 `Scan for Known Vulnerabilities` 步驟阻斷（`net/http@go1.27.3`，修復於 `go1.27.4`）。
  - **核心變更檔案**：`go.mod`、`readme.md`、`readme_zh.md`、`release_note/v2.1.10/release_notes*.md`
  - **查驗指令**：`go version -m build\bin\wincmp.exe`（應顯示 `go1.27.4`）

## 2. 未納入發布說明的 Commit 查驗

| Commit | 類型 | 標題 | 未納入原因 |
| :--- | :--- | :--- | :--- |
| `3a7adc0` | `ci` | `ci(actions): move to node24 action runtimes and gate releases with govulncheck` | 屬 GitHub Actions CI/CD 建置與安全掃描流程強化，非使用者端功能變更 |
| `5bfebf1` | `docs` | `docs(website): state Go 1.27 as the app architecture version` | 屬官網靜態頁面架構版本文字更新，非桌面應用本身變更 |

## 3. 發布前動作確認清單

- [x] 1. 條目代碼核對通過
- [x] 2. `node .agents/skills/wincmp-release/scripts/validate_release.js` 通過
- [x] 3. `cd frontend && npm run build` 通過
- [x] 4. `go test ./...` 通過（僅 `TestOpenFolderAndShowInExplorerValidation` 在沙盒環境因 `ShellExecute` 被拒而失敗，非代碼問題）
- [ ] 5. 截圖：本版無 UI 變更，沿用 v2.1.9 的 `screenshot/`（未改動），可免重跑 `node scripts/capture.js`
- [ ] 6. `.\release.bat` 成功生成 Release 壓縮包
