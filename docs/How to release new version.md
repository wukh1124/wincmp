# WinCMP 新版本發布指南

Feature 開發完成後，走這份流程發版。  
**對外說明唯一來源**：`release_note/vX.Y.Z/release_notes.md` + `release_notes_zh.md`  
（根目錄不寫 CHANGELOG；歷史在 `release_note/archives/`）

---

## 0. 什麼時候啟動

- 功能已合入目標分支（建議 `main`）
- `go test ./...`、`cd frontend && npm run build` 無錯誤
- 依賴若有變動，已確認 `conf/dependencies.json` 可用

對 Agent 說：

```text
使用 wincmp-release skill，準備發布 vX.Y.Z。
```

Agent 會做：對照上一 tag 寫 release notes / audit、更新 `VERSION`、跑 pre-flight、協助截圖與打包。  
**Agent 不會自動 `git push` / `git tag`**，推送與打 tag 由你確認後執行。

---

## 1. Agent 任務清單（給 Agent 執行）

1. **變更分析**  
   `git tag -l --sort=-v:refname`  
   `git log <last_tag>..HEAD --oneline`  
   `git diff --name-only <last_tag>..HEAD -- frontend/ internal/ app.go bridge.go conf/`

2. **寫 Release Notes**（建立 `release_note/vX.Y.Z/`）  
   - `release_notes.md`（英文）  
   - `release_notes_zh.md`（繁中）  
   - `audit_commits.md`（內部對照 commit / 檔案）

3. **更新 `VERSION`** 為 `x.y.z`（不加 `v`）

4. **驗證**
   ```powershell
   node .agents/skills/wincmp-release/scripts/validate_release.js
   go run scripts/check_deps.go --check
   ```

5. **截圖**（需本機 `wails dev` 已啟動）
   ```powershell
   # 終端機 A
   wails dev
   # 終端機 B（專案根目錄）
   powershell -ExecutionPolicy Bypass -File .\scripts\capture_release_screenshots.ps1
   ```
   腳本會把舊圖備份到 `screenshot/backup/v{發布前舊版}/`，再產出 dark/sketch 各 8 張。

6. **打包**（本地驗證產物；正式 build 以 CI 為準）
   ```powershell
   .\release.bat
   ```
   確認 `release_note/release_info.json` 已更新，且 notes **不是** stub。

7. **輸出審核清單給你**（見下一節），等你確認後由你 commit / push / tag。

---

## 2. 人工審核（必做）

### 2.1 Release Notes
- [ ] 中英文內容對齊，功能點有寫到、沒寫錯
- [ ] 分類只用：`Added` / `Changed` / `Deprecated` / `Removed` / `Fixed` / `Security` / `Dependencies`
- [ ] 沒有 commit hash、PR 編號
- [ ] `Getting Started` 的 zip / exe 檔名與版號一致

### 2.2 Git Commit 內容
- [ ] `git status` / `git diff --cached` 只含該 release 應提交的檔
- [ ] 建議 staged：`VERSION`、`release_note/`、`screenshot/`、`conf/dependencies.json`（若依賴有改）、本次功能原始碼
- [ ] 不要提交：`*.exe` / `*.zip`、`wincmp-release-only/`、`screenshot/backup/`
- [ ] commit message 使用 `type(scope): description`

### 2.3 截圖
- [ ] `screenshot/dark/*`、`screenshot/sketch/*` 已是新 UI
- [ ] 無 loading 空白、氣泡遮擋、錯誤對話框入鏡

---

## 3. Git / GitHub 操作（由你執行）

```powershell
# 確認目前分支
git status -sb

# 提交發布內容（依實際變更調整路徑）
git add VERSION release_note/ screenshot/ conf/dependencies.json
# 若功能碼尚未提交，一併 add 對應 src / internal / frontend 路徑
git commit -m "chore(release): prepare for vX.Y.Z"

git push origin <branch>          # 通常為 main

# Tag 必須與 VERSION 一致，例如 VERSION=2.1.1 → v2.1.1
git tag vX.Y.Z
git push origin vX.Y.Z
```

**注意**
- tag 格式：`v` + `VERSION` 內容（`v2.1.1`，不是 `version-2.1.1`）
- 同一 tag 不可重複推送；要重發請改版號或刪除遠端 tag 後重來
- `develop` → `main` 的 PR 合併後，tag 打在 **main**

---

## 4. CI 流程

### 4.1 Build and Release  
`.github/workflows/release.yml`  
**觸發**：push `v*` tag，或 Actions 手動 `workflow_dispatch`

```text
Checkout
→ setup Go / Node / Wails
→ npm ci (frontend)
→ go run scripts/check_deps.go --check   # 依賴 URL + SHA256
→ ./bat/release.ps1                      # build + 組裝 + 更新 release_info
→ 讀取 release_note/vX.Y.Z/release_notes.md 當 Release body
→ softprops/action-gh-release
   上傳 ../wincmp-release-only/wincmp-vX.Y.Z-win-x64.zip
   上傳 ../wincmp-release-only/WinCMP_vX.Y.Z.exe
```

成功後 GitHub 應有：
- Release：`WinCMP vX.Y.Z`
- Assets：zip + 獨立 exe（自動更新用）

### 4.2 Deploy Website  
`.github/workflows/deploy-website.yml`  
**觸發**：
- push 到 `main` 且變更 `website/**` 或 `release_note/**`
- GitHub Release `published`
- `workflow_dispatch`

```text
Checkout
→ icon.svg → website/favicon.svg
→ cp screenshot/* → website/screenshot/
→ node scripts/generate-release-json.js  # 讀 release_info + release_notes
→ 推 gh-pages
```

官網：`https://wukh1124.github.io/wincmp/`  
（版本號、下載連結、changelog、截圖）

---

## 5. 發布後驗證清單

- [ ] Actions：Build and Release = success
- [ ] Actions：Deploy Website = success
- [ ] GitHub Release 有 zip + exe，body 是 release notes
- [ ] 下載連結可下載
- [ ] 官網 badge 版號正確
- [ ] 官網 changelog 中英文正確
- [ ] 官網截圖是新版
- [ ] App 內「版本更新」能讀到新版本（更新器打 GitHub / raw release notes）

---

## 6. 注意事項

| 項目 | 說明 |
|------|------|
| 內容唯一來源 | 只改 `release_note/vX.Y.Z/release_notes*.md`，不要另寫 CHANGELOG |
| 依賴校驗 | CI 必過 `check_deps --check`；缺 sha256 / cacert 過期會擋整個 release |
| 截圖 | 必須本機 `wails dev` + capture script；CI 不會自動擷圖 |
| 備份目錄 | `screenshot/backup/` 已 gitignore，不需 commit |
| 二進位 | 禁止 commit exe/zip；產物在 `wincmp-release-only/`（倉庫外） |
| push / tag | 僅人工執行；Agent 只產指令與檔案 |
| 同步順序 | 先 push 程式碼，再打 tag；避免 tag 指到舊 commit |

---

## 7. 錯誤應對

### 7.1 Build and Release 失敗在 `check_deps`

```powershell
go run scripts/check_deps.go --check
# 需重算 hash
go run scripts/check_deps.go --update --force
git add conf/dependencies.json
git commit -m "chore(deps): refresh dependency checksums"
git push origin main
```

- **cacert SHA 不匹配**：上游更新過，改 version 標示 + 新 sha256  
- **大檔逾時 / EOF**：`scripts/check_deps.go` 已有 600s + 重試；仍失敗再查網路或換鏡像  
- **tag 已存在但 Release 失敗**：修完後用 Actions 對該 workflow `Re-run`，或 `workflow_dispatch`；不必急著刪 tag（除非 tag 指錯 commit）

### 7.2 Release 建立了但 assets 缺 zip/exe

- 看 job log：`Run Release Script` / `Create GitHub Release`
- 常見：`release.ps1` 找不到 wails、7-Zip、`build/bin/wincmp.exe`
- 修好後 re-run workflow，讓 action 重新上傳 assets

### 7.3 官網版本號 / changelog 沒更新

1. 確認 `release_note/release_info.json` 的 `latest-version` 已是新值  
2. 確認 `release_note/vX.Y.Z/release_notes*.md` 已 push 到 main  
3. Actions → Deploy Website → Re-run  
4. 瀏覽器強制重新整理（CDN/快取）

### 7.4 官網截圖仍是舊圖

1. 確認 `screenshot/dark`、`screenshot/sketch` 已更新並 **commit + push**  
2. 確認 Deploy Website 有跑（會 `cp screenshot/*`）  
3. 需要時手動 `workflow_dispatch` 觸發部署

### 7.5 Validator 失敗

| 訊息 | 處理 |
|------|------|
| 找不到 release_notes | 由 Agent/skill 先寫 notes，或檢查目錄名 `vX.Y.Z` |
| 分類不在白名單 | 只用 §2.1 的 7 種 `###` 標題 |
| notes 過短 | 補齊 What's Changed 實質內容，勿留 stub 上線 |

### 7.6 截圖腳本失敗

| 訊息 | 處理 |
|------|------|
| 找不到 VERSION / 無法定位專案根目錄 | 在專案根目錄執行 `.\scripts\capture_release_screenshots.ps1` |
| Wails dev server is not reachable | 另開終端機先 `wails dev`，再重跑腳本 |
| Playwright 失敗 | `cd frontend` 後 `npx playwright install chromium` |

### 7.7 打錯 tag / 要撤回

```powershell
# 僅在尚未被他人依賴、且你確定要重來時執行
git tag -d vX.Y.Z
git push origin :refs/tags/vX.Y.Z
# 修正內容後重新打 tag 並 push
```

GitHub 上若已建立錯誤 Release，到 Releases 頁刪除該 Release 再重建。

---

## 8. 目錄速查

| 路徑 | 用途 |
|------|------|
| `VERSION` | 版號唯一來源 `x.y.z` |
| `release_note/vX.Y.Z/release_notes.md` | 英文對外說明 |
| `release_note/vX.Y.Z/release_notes_zh.md` | 繁中對外說明 |
| `release_note/vX.Y.Z/audit_commits.md` | 內部稽核 |
| `release_note/release_info.json` | 最新版號 / 日期（官網 + 更新器） |
| `release_note/archives/` | 歷史 CHANGELOG（只讀） |
| `screenshot/{dark,sketch}/` | 官網目前截圖（要 commit） |
| `screenshot/backup/v*/` | 舊圖備份（不 commit） |
| `conf/dependencies.json` | 依賴版本 + SHA256（CI 校驗） |
| `.agents/skills/wincmp-release/` | 發布 skill 與 validator |
| `.github/workflows/release.yml` | 打包 + GitHub Release |
| `.github/workflows/deploy-website.yml` | 官網部署 |

---

## 9. 一句話流程

```text
功能完成
→ 通知 Agent 走 wincmp-release skill
→ 審核 release notes + git diff
→ wails dev + 截圖腳本
→ release.bat / validator
→ 人工 git commit + push + tag
→ CI Build Release + Deploy Website
→ 驗證 GitHub Release 與官網
```
