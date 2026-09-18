---
name: wincmp-release
description: Release management, changelog validation, and release workflow automation for WinCMP (Wails v2 Windows App). Use when releasing new versions, bumping VERSION, drafting dual-language changelogs, auditing release commits, running pre-flight verification, or executing automated packaging.
---

# WinCMP Release Manager: 桌面應用版本發布管理規範

本 Skill 專為 **WinCMP** (Wails v2 + Go + React / TypeScript) 桌面應用程式設計，借鏡標準化發布體系，結合雙語更新日誌、雙檔協同發行機制與自動化打包流水線，確保對外發行說明極致專業純淨、內部提交 100% 精準可溯源。

> [!NOTE]
> **零專案污染與本地化**：本 Skill 存放於專案本地目錄 `.agents/skills/wincmp-release/`，隨 Git 倉庫同步管理，無需額外安裝全域依賴即可直接調用。

---

## 0. 核心發布原則與架構

### 0.1 版本號流轉核心
WinCMP 的版本號以專案根目錄的 `VERSION` 檔案（格式如 `2.0.6`，純數字三段式）為單一信任來源（Single Source of Truth）：
- **Wails 編譯**：透過 `-ldflags "-X main.AppVersion=v$Version"` 將版號注入 Go 端 `main.AppVersion`。
- **前端顯示**：React 介面透過 `GetAppVersion()` 向後端獲取版號。
- **更新檢查**：自動更新器與官網比對 `release_note/release_info.json` 之 `latest-version`。
- **打包發行**：`release.bat` 讀取 `VERSION` 產出對應版號的目錄與 Zip 壓縮包。

### 0.2 安全防護底線
- **嚴禁自動推送**：AI 助手**絕對不得**在未經指示或自動腳本中擅自執行 `git push` 或 `git tag`，所有發布指令均需輸出給開發者手動審核執行。
- **禁止二進位提交**：打包產出之 `*.exe`、`*.zip` 均輸出至父目錄 `wincmp-release-only/`，嚴禁將二進位檔提交進 Git 倉庫。

---

## 1. 發行前變更分析 (Pre-release Audit)

在起草文件與提升版號前，必須先分析自上一 Tag 以來的實際變更：

```bash
# 1. 查看上一個發行 Tag
git tag -l --sort=-v:refname | head -n 2

# 2. 檢視該區間內的所有 Commit
git log <last_tag>..HEAD --oneline

# 3. 分區檢視核心異動模組
git diff --name-only <last_tag>..HEAD -- frontend/ internal/ app.go bridge.go
```

### 變更維度歸類提示
- **前端異動 (`frontend/`)**：UI 介面調整、Zustand 狀態變更、i18n 翻譯字典、主題樣式。
- **後端核心 (`internal/`, `app.go`, `bridge.go`)**：系統進程管理、Caddyfile 生成、連接埠佔用偵測、資源監控。
- **設定與封裝 (`conf/`, `packaging/`, `bat/`)**：預設設定檔、打包範本、自動化腳本。

---

## 2. 雙語 Changelog 與雙檔協同機制

WinCMP 採雙軌發布設計：保持對外更新日誌乾淨高雅，同時在內部留存完整的代碼核對軌跡。

```text
wincmp/
├── VERSION                               # 單一版號定義 (x.y.z)
├── CHANGELOG_zh.md                       # [繁體中文 Changelog]
├── CHANGELOG.md                          # [英文 Changelog]
└── release_note/
    ├── release_info.json                 # 版本更新資訊 (latest-version, release-date)
    └── vX.Y.Z/
        ├── release_notes_zh.md           # [檔案 A-1] 對外繁中發布說明
        ├── release_notes.md              # [檔案 A-2] 對外英文發布說明
        └── audit_commits.md              # [檔案 B] 內部稽核底稿 (Audit Trail)
```

---

### 2.1 雙語 Changelog 編寫規範 (`CHANGELOG_zh.md` & `CHANGELOG.md`)

#### 版本標題格式
```markdown
## [2.0.7] - YYYY-MM-DD
<!-- 或 -->
## [2.0.7] YYYY-MM-DD
```

#### 分類標籤白名單（嚴格限定以下 7 種）
- `### Added`：全新功能、新增設定項或 UI 控制模組。
- `### Changed`：現有功能修改、預設行為調整、UI/UX 優化。
- `### Deprecated`：即將棄用的功能或設定提示。
- `### Removed`：已完全移除的功能或檔案。
- `### Fixed`：Bug 修復、競爭條件排除、錯誤防護。
- `### Security`：安全防護提升、權限與防篡改增強。
- `### Dependencies`：Go 模組、NPM 套件或外部依賴升級。

#### 文風與寫作準則
- **俐落開頭**：中文以「**粗體功能名稱**：動詞...」開頭，英文以「**Feature Name**: Verb...」開頭。
  - *中文範例*：`- **連接埠佔用偵測**：啟動專案時若目標 Port 被佔用，自動提示並支援安全終止佔用進程。`
  - *英文範例*：`- **Port Occupancy Detection**: Automatically detects conflicting processes and allows safe termination via UI.`
- **拒絕官腔與虛詞**：嚴禁「本次更新為您帶來了...」、「...之功能」等公文官腔。
- **聚焦使用者與操作影響**：說明實際解決了什麼痛點或提供了什麼操作，避免流水帳堆砌底層私有函式名。
- **零 Commit 污染**：Changelog 與 Release Notes 內**絕不包含** commit hash 或內部 PR 編號。

---

### 2.2 檔案 B: `release_note/vX.Y.Z/audit_commits.md` (內部稽核檔案)

供發布者核對代碼變更、避免遺漏重要功能，並歸檔留存於倉庫中作為審核底稿。

#### 稽核檔範本結構
```markdown
# Release Audit Checklist: WinCMP vX.Y.Z

> **審核說明**：本檔案為發行前代碼變更之真實性、完整性核對底稿與歷史決策稽核檔案 (Audit Trail)。
> 完成核對後請一併提交至 Git 倉庫歸檔留存。

- **比對基準**：`<last_tag>..HEAD`
- **發行日期**：YYYY-MM-DD

---

## 1. 變更條目與代碼核對矩陣 (Audit Matrix)

### [Added / Changed / Fixed / ...]
- [ ] **條目簡稱**
  - **實際影響**：1～2 句摘要實際功能或修復內容
  - **關聯 Commit**：`<hash>` `<commit subject>`
  - **核心變更檔案**：`frontend/src/...` 或 `internal/...`
  - **查驗指令**：`git show <hash> --stat`

---

## 2. 未納入發布說明的 Commit 查驗 (防遺漏清單)

| Commit | 類型 | 標題 | 未納入原因 |
| :--- | :--- | :--- | :--- |
| `<hash>` | `chore` | `...` | 開發環境微調或內部註解整理 |
| `<hash>` | `style` | `...` | 程式碼格式化 |

---

## 3. 發布前動作確認清單

- [ ] 1. 條目代碼核對：上述矩陣條目均已通過代碼核對。
- [ ] 2. 規範存活驗證：執行 `node .agents/skills/wincmp-release/scripts/validate_release.js` 通過。
- [ ] 3. 前端建置通過：`cd frontend && npm run build` 無型別或建置報錯。
- [ ] 4. 後端測試通過：`go test ./...` 全數通過。
- [ ] 5. 打包腳本執行：執行 `.\release.bat` 成功生成 Release 壓縮包與更新 notes。
```

---

## 3. 發行前自動驗證 (Pre-flight Validation)

在專案根目錄下執行本 Skill 隨附的驗證腳本：

```powershell
# 跨平台 Node.js 執行方式：
node .agents/skills/wincmp-release/scripts/validate_release.js

# 或透過 PowerShell 包裝腳本執行：
powershell -ExecutionPolicy Bypass -File .agents/skills/wincmp-release/scripts/validate_release.ps1
```

驗證器將自動檢查：
1. `VERSION` 檔案是否存在且符合 `x.y.z` 語意化版本格式。
2. `CHANGELOG_zh.md` 與 `CHANGELOG.md` 是否均已包含對應當前版號的區塊。
3. 更新條目之分類標題是否完全在 7 大白名單內。
4. 是否已建立 `release_note/vX.Y.Z/audit_commits.md` 內部稽核檔。
5. 檢查當前 Git 工作區狀態。

---

## 4. 全套品質迴歸驗證

在執行打包前，必須確保前端與後端建置雙綠燈：

```bash
# 1. 前端 TypeScript 與 Vite 打包檢查
cd frontend
npm run build
cd ..

# 2. Go 後端單元測試
go test ./...
```

---

## 5. 執行自動化發行打包

WinCMP 配備專用發行精靈 `release.bat`：

```powershell
# 執行專案打包精靈 (會自動調用 bat\release.ps1)
.\release.bat
```

此腳本將自動完成：
1. 以 `wails build -clean -ldflags "-s -w -X main.AppVersion=v$Version"` 編譯生產級二進位檔。
2. 在父目錄 `../wincmp-release-only/` 組裝發行目錄 `wincmp_v$Version/`。
3. 清理 `.gitkeep`、`.example`、日誌與測試資料目錄。
4. 使用 7-Zip 或 PowerShell 封裝產生 `wincmp-v$Version-win-x64.zip`。
5. 從雙語 CHANGELOG 自動提取內容，更新 `release_note/v$Version/release_notes.md` 與 `release_notes_zh.md`。
6. 更新 `release_note/release_info.json` 的最新版本號與發行日期。

---

## 6. Git 發行指令引導

打包完成且人工確認無誤後，主動輸出標準 Git 指令供開發者審核執行（**Agent 嚴禁擅自執行 git push 或 git tag**）：

```bash
# 1. 提交版號、更新日誌與發行說明
git add VERSION CHANGELOG.md CHANGELOG_zh.md release_note/
git commit -m "chore(release): bump version to vX.Y.Z"

# 2. 建立附註標籤 (Annotated Tag)
git tag -a vX.Y.Z -m "Release version X.Y.Z"

# 3. 推送程式碼與 Tag 至遠端倉庫 (由開發者確認執行)
git push origin <current_branch>
git push origin vX.Y.Z
```
