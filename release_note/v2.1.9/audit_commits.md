# Release Audit Checklist: WinCMP v2.1.9

> **審核說明**：本檔案為發行前代碼變更之真實性、完整性核對底稿。

- **比對基準**：`v2.1.8..HEAD`（單一 commit `f8cd74c`）
- **發行日期**：2026-10-23

## 1. 變更條目與代碼核對矩陣 (Audit Matrix)

### [Security]
- [x] **改用 Go 1.27.3 編譯**
  - **實際影響**：`go.mod` 的 `go` 指令由 `1.26.2` 提升為 `1.27.3`。GitHub Actions 的 `actions/setup-go@v5` 以 `go-version-file: go.mod` 解析版本，因此 v2.1.9 起的執行檔由 Go 1.27.3 編譯，內含 `crypto/x509` 對 CVE-2026-111943（封鎖特定中繼 CA）的阻擋邏輯。
  - **關聯 Commit**：`f8cd74c` `build(go): require Go 1.27.3 to pick up CVE-2026-111943 fix`
  - **核心變更檔案**：`go.mod`
  - **查驗指令**：`git show f8cd74c -- go.mod`；`git show v2.1.8:go.mod`（應為 `go 1.26.2`，證明舊版未修補）

- [x] **工具鏈下限文件同步**
  - **實際影響**：README 徽章與 Prerequisites 段落由「1.26.2+，用 1.27.x 亦可」改為硬性最低 1.27.3，避免後續建置再用舊版工具鏈。
  - **關聯 Commit**：`f8cd74c`
  - **核心變更檔案**：`readme.md`、`readme_zh.md`
  - **查驗指令**：`git show f8cd74c -- readme.md readme_zh.md`

## 2. 未納入發布說明的 Commit 查驗

| Commit | 類型 | 標題 | 未納入原因 |
| :--- | :--- | :--- | :--- |
| （無） | — | `v2.1.8..HEAD` 僅有 `f8cd74c` 一筆 | 全部已列入上方矩陣 |

## 3. 發布前動作確認清單

- [x] 1. 條目代碼核對通過（`git show f8cd74c --stat` 僅 3 檔 5 行）
- [x] 2. `node .agents/skills/wincmp-release/scripts/validate_release.js` 通過
- [ ] 3. `cd frontend && npm run build` 通過（本版未動前端，建議於打包前仍跑一次）
- [x] 4. `go test ./...`：`go vet ./...` 與 `go mod tidy -diff` 皆 0 錯誤；測試僅 `TestOpenFolderAndShowInExplorerValidation` 失敗，原因為 `ShellExecute` 在受限環境被拒（`Access is denied`），與本版改動無關，需於開發者一般終端機複驗
- [ ] 5. 截圖：本版無 UI 變更，沿用 v2.1.8 的 `screenshot/`（未改動），可免跑 `node scripts/capture.js`
- [ ] 6. `.\release.bat` 成功生成 Release 壓縮包

## 4. 併發風險與後續驗證

- **建置產物驗證**：發布後執行 `go version -m build\bin\wincmp.exe`，輸出的 `go1.27.3` 為本次修補生效的鐵證；若仍顯示 `go1.26.2`，代表 CI 未吃到 `go.mod`，需立即檢查 workflow。
- **相依完整性**：本次未新增或升級任何 Go/Node 套件，`go.sum` 與 `frontend/package-lock.json` 均無變動，`scripts/check_deps.go --check` 結果不受影響。
- **使用者影響**：更新器以 `checksums.txt` 做 SHA-256 強制比對；因 `checksums.txt` 與 exe 同源，本次修補的實際價值在於切斷「被點名中繼 CA 可同時替換兩者」的路徑。
