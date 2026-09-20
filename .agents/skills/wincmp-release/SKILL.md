---
name: wincmp-release
description: Release management, release_note validation, and release workflow automation for WinCMP (Wails v2 Windows App). Use when releasing new versions, bumping VERSION, drafting dual-language release notes, capturing screenshots, auditing release commits, running pre-flight verification, or executing automated packaging.
---

# WinCMP Release Manager: 桌面應用版本發布管理規範

本 Skill 專為 **WinCMP** (Wails v2 + Go + React / TypeScript) 桌面應用程式設計。

> **唯一內容來源**：對外更新說明以 `release_note/vX.Y.Z/release_notes.md` 與 `release_notes_zh.md` 為準。
> 根目錄不再維護 `CHANGELOG.md` / `CHANGELOG_zh.md`（歷史內容已歸檔於 `release_note/archives/`）。
> 內建更新器、GitHub Release body、官網 `release.json` 皆直接讀取 release_note。

---

## 0. 核心發布原則與架構

### 0.1 版本號流轉核心
版本號以專案根目錄 `VERSION`（格式 `x.y.z`）為單一信任來源：
- **Wails 編譯**：`-ldflags "-X main.AppVersion=v$Version"`
- **前端顯示**：React 透過 `GetAppVersion()`
- **App 更新檢查**：`internal/updater` 直接打 GitHub `releases/latest`
- **官網 release.json**：`scripts/generate-release-json.js` 讀 **VERSION** + `release_note/vX.Y.Z/release_notes*.md`（日期自 notes 擷取；fallback git tag）
- **發布日期**：寫在雙語 notes 內固定日期行（見 §2.1），**不再使用** `release_info.json`
- **打包發行**：`release.bat` / CI `release.ps1` 讀取 `VERSION` 產出 Zip / Exe
- **對外說明**：`release_note/vX.Y.Z/release_notes*.md`（人工/skill 撰寫）

### 0.2 標準 Release 流程（新）

```text
1. 使用本 Skill 撰寫 release_note/vX.Y.Z/release_notes.md + release_notes_zh.md
   （含發布日期行）並完成 audit_commits.md 內部稽核底稿
2. 更新 VERSION → 執行 pre-flight validator
   （validator 檢查 notes 日期行；不再檢查/寫入 release_info.json）
3. 本機啟動 wails dev（http://localhost:34115）
4. 執行 scripts/capture_release_screenshots.ps1
   - 以 git tag「發布前舊版」備份 screenshot/{dark,sketch} → screenshot/backup/v{prev}/
   - 重新擷取 16 張官網截圖
5. 執行 ./bat/release.ps1（建置；校驗 notes 已存在與日期行）
6. Git 發行（由開發者審核執行，見 §7）：
   - 在發行分支 commit（VERSION + release_note 等）
   - **線性進入 main**（rebase 後 fast-forward；禁止 merge commit）
   - 在 main 上 `git tag -a vX.Y.Z`
   - 先 `push origin main`，再 `push origin vX.Y.Z`
7. GitHub Actions：
   - Build and Release：check_deps → build exe/zip → 建立 GitHub Release
     （CI 讀 VERSION）
   - Deploy Website：generate-release-json（讀 VERSION + notes 日期行）+ 複製 screenshot 至 gh-pages
```

### 0.3 安全防護底線
- **嚴禁自動推送**：AI 助手**絕對不得**擅自執行 `git push`、`git merge` 或 `git tag`；僅可輸出建議指令供開發者審核
- **main 歷史**：GitHub 規則 **main 不得包含 merge commits**；合併發行內容必須用 rebase + `--ff-only`（或 cherry-pick），不可 `git merge feature/...`
- **Tag 位置**：tag 必須指向 **main 上已存在的發行 commit**（通常為 main HEAD），再 push tag 觸發 CI
- **禁止二進位提交**：打包產出至父目錄 `wincmp-release-only/`，嚴禁提交 `*.exe` / `*.zip`

---

## 1. 發行前變更分析 (Pre-release Audit)

```bash
git tag -l --sort=-v:refname | head -n 2
git log <last_tag>..HEAD --oneline
git diff --name-only <last_tag>..HEAD -- frontend/ internal/ app.go bridge.go
```

---

## 2. 唯一內容來源：Release Notes 雙語檔案

```text
wincmp/
├── VERSION                               # 單一版號定義 (x.y.z)
└── release_note/
    ├── archives/                         # 歷史 CHANGELOG 歸檔（唯讀）
    │   ├── CHANGELOG_en.md
    │   └── CHANGELOG_zh.md
    └── vX.Y.Z/
        ├── release_notes_zh.md           # 對外繁中發布說明（唯一來源；含發布日期）
        ├── release_notes.md              # 對外英文發布說明（唯一來源；含 Release date）
        └── audit_commits.md              # 內部稽核底稿
```

### 2.1 Release Notes 編寫規範

#### 檔案結構範本
```markdown
# WinCMP vX.Y.Z

發布日期：YYYY-MM-DD
<!-- 英文: Release date: YYYY-MM-DD -->

此版本為 WinCMP 帶來了新的功能、更新與修正。

## What's Changed

### Added
- **功能名稱**：動詞開頭說明使用者影響...

### Changed
- **...**

### Fixed
- **...**

## Getting Started
1. 下載 `wincmp-vX.Y.Z-win-x64.zip`。
2. 解壓縮至您系統中的任何資料夾。
3. 按兩下 `WinCMP_vX.Y.Z.exe` 啟動控制面板。
```

**日期行（必填）**：緊接主標題之後，單獨一行。
- 英文：`Release date: YYYY-MM-DD`
- 繁中：`發布日期：YYYY-MM-DD`
- 擷取由 `validate_release.js` / `generate-release-json.js` 以正規表示法完成，無需 `release_info.json`

#### 分類標籤白名單（嚴格限定以下 7 種）
- `### Added` / `### Changed` / `### Deprecated` / `### Removed`
- `### Fixed` / `### Security` / `### Dependencies`

#### 文風
- 中文：`- **粗體功能名稱**：動詞...`
- 英文：`- **Feature Name**: Verb...`
- **零 Commit 污染**：release notes 絕不包含 commit hash 或 PR 編號
- **拒絕官腔**：聚焦使用者與操作影響

---

### 2.2 檔案 B: `audit_commits.md`（內部稽核）

```markdown
# Release Audit Checklist: WinCMP vX.Y.Z

> **審核說明**：本檔案為發行前代碼變更之真實性、完整性核對底稿。

- **比對基準**：`<last_tag>..HEAD`
- **發行日期**：YYYY-MM-DD

## 1. 變更條目與代碼核對矩陣 (Audit Matrix)
### [Added]
- [ ] **條目簡稱**
  - **實際影響**：...
  - **關聯 Commit**：`<hash>` `<subject>`
  - **核心變更檔案**：`...`
  - **查驗指令**：`git show <hash> --stat`

## 2. 未納入發布說明的 Commit 查驗
| Commit | 類型 | 標題 | 未納入原因 |
| :--- | :--- | :--- | :--- |

## 3. 發布前動作確認清單
- [ ] 1. 條目代碼核對通過
- [ ] 2. `node .agents/skills/wincmp-release/scripts/validate_release.js` 通過
- [ ] 3. `cd frontend && npm run build` 通過
- [ ] 4. `go test ./...` 通過
- [ ] 5. `wails dev` 已啟動且 `scripts/capture_release_screenshots.ps1` 完成
- [ ] 6. `.\release.bat` 成功生成 Release 壓縮包
```

---

## 3. 發行前自動驗證 (Pre-flight Validation)

```powershell
node .agents/skills/wincmp-release/scripts/validate_release.js
# 或
powershell -ExecutionPolicy Bypass -File .agents/skills/wincmp-release/scripts/validate_release.ps1
```

驗證器將自動檢查：
1. `VERSION` 是否符合 `x.y.z`
2. `release_note/vX.Y.Z/release_notes.md` 與 `release_notes_zh.md` 是否存在且有內容
3. 雙語 notes 是否含發布日期行（`Release date:` / `發布日期：`）
4. `What's Changed` 分類標籤是否在白名單內
5. 是否已建立 `audit_commits.md`（警告）
6. Git 工作區狀態

---

## 4. 全套品質迴歸驗證

```bash
cd frontend && npm run build && cd ..
go test ./...
```

---

## 5. 截圖自動化（Release 必做）

```powershell
# 前置：另一個終端機已執行 wails dev；請在專案根目錄執行
powershell -ExecutionPolicy Bypass -File .\scripts\capture_release_screenshots.ps1
```

腳本行為：
1. 讀取 `VERSION` 目標版
2. 從 **git tag** 取得**發布前舊版**版號（不使用 release_info.json）
3. 將現有 `screenshot/dark`、`screenshot/sketch` 複製到 `screenshot/backup/v{舊版}/`
4. 確認 `http://localhost:34115` 可連線（否則結束並提示啟動 `wails dev`）
5. 執行 `frontend/scripts/capture.cjs` 產出新圖

`screenshot/backup/` 已在 `.gitignore`，不會進倉庫；目前使用的 `screenshot/{dark,sketch}/` 需隨 release commit 提交，官網部署時會複製到 `website/screenshot/`。

---

## 6. 執行自動化發行打包

```powershell
.\release.bat
```

此腳本將自動完成：
1. `wails build -clean -ldflags "-s -w -X main.AppVersion=v$Version"`
2. 組裝 `../wincmp-release-only/wincmp_v$Version/`
3. 清理 `.gitkeep` / `.example` / 日誌
4. 產生 `wincmp-v$Version-win-x64.zip` 與獨立 `WinCMP_v$Version.exe`
5. **校驗**（非產生）`release_note/v$Version/release_notes*.md` 已存在；缺失時才寫入 stub 並警告
6. 檢查 notes 是否含發布日期行；若仍存在舊的 `release_info.json` 會自動移除

---

## 7. Git 發行指令引導

完成打包與人工確認後，輸出指令供開發者審核執行（**Agent 嚴禁擅自 rebase / push / tag**）。

### 7.1 建議流程：線性進入 main（rebase + ff-only），再 tag

**GitHub 規則**：`main` **不得包含 merge commits**。  
禁止使用 `git merge feature/...` 直接合併（會產生 merge commit，觸發 protected branch 規則；即便帳號可 bypass，仍違反專案歷史約定）。

Tag 應指向「**已進入 main 的最終發行 commit**」。GitHub Actions、官網 `release.json` 與內建更新器皆以 tag / main 為準。

```bash
# 1) 在發行分支提交發布內容（依實際變更調整路徑）
#    notes 須含發布日期行；validator 會檢查
git add VERSION release_note/vX.Y.Z/ screenshot/ conf/dependencies.json scripts/ .github/ .agents/
# 若功能碼尚未提交，一併 add 對應 frontend/ internal/ 等路徑
git commit -m "chore(release): bump version to vX.Y.Z"

# 2) 更新 main
git checkout main
git pull origin main

# 3) 將發行分支 rebase 到 main 之上（改寫 feature 歷史為線性）
git checkout feature/version-X.Y.Z
git rebase main
# 若衝突：逐檔解決後 git add <file> && git rebase --continue

# 4) fast-forward main（不產生 merge commit）
git checkout main
git merge --ff-only feature/version-X.Y.Z

# 5) 確認 main 上 VERSION / release_note 已就緒，且 log 無 Merge commit
git show HEAD:VERSION
git ls-tree --name-only HEAD release_note/vX.Y.Z/
git log --oneline --merges -1   # 應無本次發行的 merge

# 6) 在 main 上打 annotated tag（必須與 VERSION 一致：2.1.1 → v2.1.1）
git tag -a vX.Y.Z -m "Release version X.Y.Z"

# 7) 先推 main，再推 tag（確保 CI checkout 的 main 已含發行檔案）
git push origin main
git push origin vX.Y.Z
```

### 7.2 精簡版（feature 已 commit、內容尚未進 main）

```bash
git checkout main && git pull origin main
git checkout feature/version-X.Y.Z && git rebase main
git checkout main && git merge --ff-only feature/version-X.Y.Z
git tag -a vX.Y.Z -m "Release version X.Y.Z"
git push origin main
git push origin vX.Y.Z
```

### 7.3 線性替代方案（不改寫 feature 歷史時）

若不想 rebase feature，可在 main 上 cherry-pick 發行 commit（同樣保持線性）：

```bash
git checkout main && git pull origin main
git cherry-pick <commit1> <commit2> ...   # 按時間序，通常到 chore(release) 為止
git tag -a vX.Y.Z -m "Release version X.Y.Z"
git push origin main && git push origin vX.Y.Z
```

### 7.4 注意事項
- **禁止** `git merge feature/...` 直接進 main；必須 rebase + `--ff-only` 或 cherry-pick
- tag 必須指向 **main 上的 commit**，且 **push tag** 後 CI 才會跑；只 push main 不會自動發版
- tag 格式：`v` + `VERSION` 內容（`v2.1.1`，不是 `version-2.1.1`）
- 若 remote 出現 `Bypassed rule violations` / `must not contain merge commits`：表示 main 被寫入 merge commit（規則被繞過），流程有誤，應改走線性流程
- 同一 tag 不可重複推送；重發需改版號，或先刪遠端 tag 後重來
- 不要提交：`*.exe`、`*.zip`、`wincmp-release-only/`、`screenshot/backup/`
- rebase 後若 main 另有未發布 commit，應重新確認 release notes 是否仍完整

---

## 8. 依賴設定 (`conf/dependencies.json`)

CI 的 `go run scripts/check_deps.go --check` 會下載每一項並比對 `sha256`。
發行前請確認：

```powershell
go run scripts/check_deps.go --check
# 需更新 hash 時
go run scripts/check_deps.go --update --force
```

注意：
- 每一項都必須有 `sha256`，否則 check 模式會失敗
- `cacert.pem` 上游會定期更新，SHA 失配屬預期，需更新版本標示與 hash
- 大檔（MariaDB 等）下載逾時已放寬至 600 秒
