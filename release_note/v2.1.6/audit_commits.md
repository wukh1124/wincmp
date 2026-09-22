# Release Audit Checklist: WinCMP v2.1.6

> **審核說明**：本檔案為發行前代碼變更之真實性、完整性核對底稿。
> 完成核對後請一併提交至 Git 倉庫歸檔留存。

- **比對基準**：`v2.1.5..HEAD`
- **發行日期**：2026-09-23
- **發布分支**：`feature/version-v2.1.6`

---

## 1. 變更條目與代碼核對矩陣 (Audit Matrix)

### [Added]
- [x] **開源合規與第三方授權聲明**
  - **實際影響**：專案根目錄與發布範本新增 `THIRD-PARTY-NOTICES.md`，詳列受控服務（Caddy、MariaDB、Redis、PHP、Node 等）、Go 後端模組與前端 NPM 套件之授權條款 (SPDX) 及官方原始碼獲取途徑；雙語 README 增設商標免責與非背書聲明。
  - **關聯 Commit**：`294f943` `docs(compliance): draft v2.1.6 open-source compliance and legal governance plan`
  - **核心變更檔案**：`THIRD-PARTY-NOTICES.md`, `readme.md`, `readme_zh.md`, `packaging/wincmp/THIRD-PARTY-NOTICES.md`
  - **查驗指令**：`git show 294f943 -- THIRD-PARTY-NOTICES.md readme.md readme_zh.md`

- [x] **依賴庫管理卡片授權徽章與操作選單**
  - **實際影響**：各依賴服務名稱旁標註極簡 License Badge（例如 `GPL-2.0`、`Apache-2.0`、`BSD-3-Clause`），下拉選單整合「授權協議」標記與「官方原始碼 ↗」外部連結，點擊即以預設瀏覽器開啟官方倉庫。
  - **關聯 Commit**：`294f943`
  - **核心變更檔案**：`frontend/src/components/DependencyManager.tsx`, `frontend/src/i18n.ts`, `internal/i18n/i18n.go`
  - **查驗指令**：`git show 294f943 -- frontend/src/components/DependencyManager.tsx`

- [x] **依賴庫彈窗合規聲明對話框**
  - **實際影響**：彈窗底部新增「開源授權與商標聲明」連結，點擊開啟自適應主題之聲明對話框，展示 WinCMP 核心架構說明、商標免責宣告與受控服務總表，並全面支援前後端繁中與英文雙語切換。
  - **關聯 Commit**：`294f943`
  - **核心變更檔案**：`frontend/src/components/DependencyManager.tsx`, `frontend/src/i18n.ts`, `internal/i18n/i18n.go`
  - **查驗指令**：`git show 294f943 -- frontend/src/components/DependencyManager.tsx`

- [x] **獨立執行檔 (exe) 啟動自解壓合規保證**
  - **實際影響**：透過 Go 內嵌資源，當使用者僅下載單獨 exe 於全新目錄首次啟動時，自動安全檢查並釋放 `LICENSE`、`THIRD-PARTY-NOTICES.md` 與雙語 `readme`，確保單檔運行時檔案系統合規性完整，且具備冪等性不覆蓋使用者自訂檔案。
  - **關聯 Commit**：`294f943`
  - **核心變更檔案**：`docs_embedded.go`, `app.go`, `app_test.go`
  - **查驗指令**：`git show 294f943 -- docs_embedded.go app.go app_test.go`

### [Changed]
- [x] **依賴目錄配置與文件標準化**
  - **實際影響**：`conf/dependencies.json`、`internal/config/dependencies.go` 與 `scripts/check_deps.go` 擴充 `license`、`homepage` 與 `source_url` 欄位，並更新 `docs/dependencies.md` 依賴說明表格。
  - **關聯 Commit**：`294f943`
  - **核心變更檔案**：`conf/dependencies.json`, `internal/config/dependencies.go`, `scripts/check_deps.go`, `docs/dependencies.md`
  - **查驗指令**：`git show 294f943 -- conf/dependencies.json docs/dependencies.md`

- [x] **背景檢查更新冷卻防抖**
  - **實際影響**：依賴庫管理彈窗加入 60 秒冷卻時間機制，避免頻繁開關彈窗時無效重複向遠端發起網路請求。
  - **關聯 Commit**：`294f943`
  - **核心變更檔案**：`frontend/src/components/DependencyManager.tsx`
  - **查驗指令**：`git show 294f943 -- frontend/src/components/DependencyManager.tsx`

### [Fixed]
- [x] **聲明彈窗 Carbon 暗色主題穿透與按鈕對比度修復**
  - **實際影響**：修復 Carbon 暗色主題下聲明彈窗因背景透明度過高導致底層文字穿透的問題；修復「關閉」按鈕白底白字問題，並優化 Hover 視覺回饋。
  - **關聯 Commit**：`294f943`
  - **核心變更檔案**：`frontend/src/components/DependencyManager.tsx`
  - **查驗指令**：`git show 294f943 -- frontend/src/components/DependencyManager.tsx`

---

## 2. 未納入發布說明的 Commit 查驗

| Commit | 類型 | 標題 | 未納入原因 |
| :--- | :--- | :--- | :--- |
| `294f943` (部分文件) | `docs` | `docs/v2.1.6_plan.md` | 內部開發規劃文件，功能已拆入上方條目 |

---

## 3. 發布前動作確認清單
- [x] 1. 條目代碼核對通過
- [x] 2. `node .agents/skills/wincmp-release/scripts/validate_release.js` 通過
- [x] 3. `cd frontend && npm run build` 通過
- [x] 4. `go test ./...` 通過
- [x] 5. `.\release.bat` 成功生成 Release 壓縮包 (wincmp-v2.1.6-win-x64.zip)

