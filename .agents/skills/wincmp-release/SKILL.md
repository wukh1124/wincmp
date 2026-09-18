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
- **更新檢查**：比對官網/`release_note/release_info.json` 的 `latest-version`
- **打包發行**：`release.bat` 讀取 `VERSION` 產出 Zip / Exe
- **對外說明**：`release_note/vX.Y.Z/release_notes*.md`（人工/skill 撰寫）

### 0.2 標準 Release 流程（新）

```text
1. 使用本 Skill 撰寫 release_note/vX.Y.Z/release_notes.md + release_notes_zh.md
   並完成 audit_commits.md 內部稽核底稿
2. 更新 VERSION，執行 pre-flight validator
3. 本機啟動 wails dev（http://localhost:34115）
4. 執行 scripts/capture_release_screenshots.ps1
   - 以「發布前舊版」備份 screenshot/{dark,sketch} → screenshot/backup/v{prev}/
   - 重新擷取 16 張官網截圖
5. 執行 ./bat/release.ps1（建置 + 更新 release_info.json；校驗 release_notes 已存在）
6. git commit + push + tag vX.Y.Z
7. GitHub Actions：
   - Build and Release：check_deps → build exe/zip → 建立 GitHub Release
   - Deploy Website：generate-release-json + 複製 screenshot 至 gh-pages
```

### 0.3 安全防護底線
- **嚴禁自動推送**：AI 助手**絕對不得**擅自執行 `git push` 或 `git tag`
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
    ├── release_info.json                 # latest-version, release-date
    ├── archives/                         # 歷史 CHANGELOG 歸檔（唯讀）
    │   ├── CHANGELOG_en.md
    │   └── CHANGELOG_zh.md
    └── vX.Y.Z/
        ├── release_notes_zh.md           # 對外繁中發布說明（唯一來源）
        ├── release_notes.md              # 對外英文發布說明（唯一來源）
        └── audit_commits.md              # 內部稽核底稿
```

### 2.1 Release Notes 編寫規範

#### 檔案結構範本
```markdown
# WinCMP vX.Y.Z

此版本為 WinCMP 帶來了新的功能、更新與修正。
<!-- 英文: This release introduces new features, updates, and fixes to WinCMP. -->

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
3. `What's Changed` 分類標籤是否在白名單內
4. 是否已建立 `audit_commits.md`（警告）
5. Git 工作區狀態

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
2. 從 `release_info.json` / git tag 取得**發布前舊版**版號
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
6. 更新 `release_note/release_info.json`

---

## 7. Git 發行指令引導

完成打包與人工確認後，輸出指令供開發者審核執行（**Agent 嚴禁擅自 push/tag**）：

```bash
git add VERSION release_note/ screenshot/ conf/dependencies.json scripts/ .github/ .agents/
git commit -m "chore(release): bump version to vX.Y.Z"
git tag -a vX.Y.Z -m "Release version X.Y.Z"
git push origin <current_branch>
git push origin vX.Y.Z
```

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
