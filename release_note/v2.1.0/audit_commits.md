# Release Audit Checklist: WinCMP v2.1.0

> **審核說明**：本檔案為發行前代碼變更之真實性、完整性核對底稿與歷史決策稽核檔案 (Audit Trail)。
> 完成核對後請一併提交至 Git 倉庫歸檔留存。

- **比對基準**：`v2.0.6..HEAD`
- **發行日期**：2026-09-18

---

## 1. 變更條目與代碼核對矩陣 (Audit Matrix)

### [Added]
- [x] **Redis 伺服器與 PHP Redis 擴充支援**
  - **實際影響**：依賴管理器新增 Redis 服務一鍵安裝與獨立啟停控制（預設 Port 6379）；安裝 PHP 時自動配置 `php_redis` 擴充模組並提示版本相容性。
  - **關聯 Commit**：`20bb6eb` feat(deps): add Redis server to dependency manager and auto-configure php_redis on PHP install
  - **核心變更檔案**：`internal/process/redis.go`, `frontend/src/components/DependencyManager.tsx`, `downloader_bridge.go`
  - **查驗指令**：`git show 20bb6eb --stat`

- [x] **專案拖曳排序與順序持久化**
  - **實際影響**：專案列表支援直覺拖曳自訂顯示排序，限定透過專屬拖曳手柄觸發防止誤觸，排序狀態自動保存至 `wincmp.json`。
  - **關聯 Commit**：`504927b` refactor(ui): refine logs console toolbar and restrict project drag handle, `f4de92e` feat: release v2.1.0 with redis support, project drag reordering...
  - **核心變更檔案**：`frontend/src/components/Projects.tsx`, `internal/config/config.go`
  - **查驗指令**：`git show 504927b --stat`

- [x] **PHP OPcache 與核心設定自動遷移**
  - **實際影響**：PHP 服務啟動時自動檢測並優化 OPcache 核心配置（含 JIT 與記憶體分配），並自動啟用常用延伸模組（curl, mbstring, openssl, pdo_mysql 等）。
  - **關聯 Commit**：`2e65a86` feat(php): auto-migrate opcache settings and enable recommended extensions
  - **核心變更檔案**：`internal/config/init.go`, `conf/php/php.ini`
  - **查驗指令**：`git show 2e65a86 --stat`

- [x] **依賴下載完成目錄捷徑**
  - **實際影響**：依賴下載完成後保留操作按鈕，並新增「開啟安裝目錄」捷徑，方便直接檢視安裝檔案。
  - **關聯 Commit**：`e13b71b` fix(deps): retain action buttons on download completed and add open install dir
  - **核心變更檔案**：`frontend/src/components/DependencyManager.tsx`
  - **查驗指令**：`git show e13b71b --stat`

- [x] **本地發布管理與預檢工具**
  - **實際影響**：引入標準化發布規範與自動化前檢腳本，確保發行品質與雙語更新日誌一致性。
  - **關聯 Commit**：`e5cda63` feat(release): add wincmp-release local skill and pre-flight validator
  - **核心變更檔案**：`.agents/skills/wincmp-release/`
  - **查驗指令**：`git show e5cda63 --stat`

### [Changed]
- [x] **日誌控制台精簡單行風格**
  - **實際影響**：日誌面板頂部工具欄重構為 VS Code 風格單行設計，整合服務分頁切換、清空與日誌級別過濾，大幅提升視覺對比度與操作流暢度。
  - **關聯 Commit**：`e372efc` style(terminal): refactor log console header to vscode single-line ghost style
  - **核心變更檔案**：`frontend/src/components/TerminalLogs.tsx`
  - **查驗指令**：`git show e372efc --stat`

- [x] **高解析度佈局與字型縮放優化**
  - **實際影響**：專案列表表頭改用 rem 比例縮放，避免高解析度螢幕下欄位折行或錯位；優化服務卡片與依賴管理器的留白間距。
  - **關聯 Commit**：`0d15e9e` style(projects): scale header font with rem, fix column wrapping and enlarge default window, `4f34412` style(projects): optimize table cell padding...
  - **核心變更檔案**：`frontend/src/components/Projects.tsx`, `frontend/src/style.css`
  - **查驗指令**：`git show 0d15e9e --stat`

- [x] **後台效能負載優化**
  - **實際影響**：移除未使用的儀表板狀態概覽區塊與無效的 Port 衝突輪詢機制，進一步降低背景閒置時的 CPU 佔用。
  - **關聯 Commit**：`b9d50db` refactor(dashboard): remove unused system status overview section and port conflict polling
  - **核心變更檔案**：`frontend/src/components/Dashboard.tsx`
  - **查驗指令**：`git show b9d50db --stat`

### [Fixed]
- [x] **PHP 設定自動優化觸發時機**
  - **實際影響**：修復 PHP ini 優化配置在初次啟動時未能即時套用的問題，確保每次啟動 PHP 時皆自動校驗。
  - **關聯 Commit**：`b7835aa` fix(config): trigger EnsurePHPIniOptimizations properly and check on PHP startup
  - **核心變更檔案**：`internal/process/php.go`, `internal/config/init.go`
  - **查驗指令**：`git show b7835aa --stat`

- [x] **依賴管理器按鈕懸停視覺**
  - **實際影響**：修復依賴下拉選單在特定主題下的 Hover 視覺反饋異常與 Dashboard 下拉箭頭遺失問題。
  - **關聯 Commit**：`05f406a` style(ui): fix dependency dropdown hover effect and terminal log tab contrast, `651b4cf` style(dashboard): restore dropdown arrow for php process selector
  - **核心變更檔案**：`frontend/src/style.css`, `frontend/src/components/Dashboard.tsx`
  - **查驗指令**：`git show 05f406a --stat`

- [x] **新手引導氣泡模糊問題**
  - **實際影響**：修復引導氣泡在特定背景濾鏡下邊緣模糊的視覺瑕疵。
  - **關聯 Commit**：`23105b2` style(ui): fix guide bubble blur and optimize core service card layout
  - **核心變更檔案**：`frontend/src/style.css`
  - **查驗指令**：`git show 23105b2 --stat`

---

## 2. 未納入發布說明的 Commit 查驗 (防遺漏清單)

| Commit | 類型 | 標題 | 未納入原因 |
| :--- | :--- | :--- | :--- |
| `c05d66d` | `merge` | Merge branch 'feature/version-2.1.0' into main | 分支合併提交，實際功能已分別納入上述條目 |
| `8b2223d` | `refactor` | refactor(deps): separate php extensions and enhance installation detection | 依賴結構重構，已整合於 Redis 與 PHP 擴充功能中 |
| `2073cde` | `style` | style(deps): update php redis not support version (ex. php 7.3) | 樣式與細微版本字串設定，已納入 Redis 支援條目 |
| `f72f24b` | `style` | style(ui): clarify dependency fetch wording and admin privilege badge | 介面文案細微調整 |

---

## 3. 發布前動作確認清單

- [x] 1. 條目代碼核對：上述矩陣條目均已通過代碼核對。
- [x] 2. 規範存活驗證：執行 `node .agents/skills/wincmp-release/scripts/validate_release.js` 通過。
- [ ] 3. 前端建置通過：`cd frontend && npm run build` 無型別或建置報錯。
- [ ] 4. 後端測試通過：`go test ./...` 全數通過。
- [ ] 5. 打包腳本執行：執行 `.\release.bat` 成功生成 Release 壓縮包與更新 notes。
