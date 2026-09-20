# Release Audit Checklist: WinCMP v2.1.2

> **審核說明**：本檔案為發行前代碼變更之真實性、完整性核對底稿。
> 完成核對後請一併提交至 Git 倉庫歸檔留存。

- **比對基準**：`v2.1.1..HEAD`
- **發行日期**：2026-09-20
- **發布分支**：`feature/version-v2.1.2`

---

## 1. 變更條目與代碼核對矩陣 (Audit Matrix)

### [Added]
- [x] **內建 Redis 快取瀏覽器**
  - **實際影響**：DBExplorer 新增 Redis 分頁，支援 DB 0–15、SCAN 分頁、鍵值詳情、DEL 與 FLUSHDB；Redis 未啟動時提供啟動引導。
  - **關聯 Commit**：`84c15d7` `feat(core): implement v2.1.2 features with redis explorer and port release guarantee`
  - **核心變更檔案**：`internal/redisexplorer/explorer.go`, `bridge.go`, `frontend/src/components/DBExplorer.tsx`
  - **查驗指令**：`git show 84c15d7 --stat`

- [x] **PHP Redis 擴充自動啟用與狀態徽章**
  - **實際影響**：啟動流程自動平滑遷移 `php.ini` 的 `extension=redis`（含 `.bak` 備份與取消註解）；儀表板 PHP 卡片顯示 Redis 狀態並可一鍵修復；依賴管理僅在 DLL + ini 皆有效時顯示已就緒。
  - **關聯 Commit**：`84c15d7` `feat(core): implement v2.1.2 features with redis explorer and port release guarantee`
  - **核心變更檔案**：`internal/config/init.go`, `internal/config/init_test.go`, `bridge.go`, `frontend/src/components/Dashboard.tsx`, `frontend/src/components/DependencyManager.tsx`
  - **查驗指令**：`git show 84c15d7 --stat`

- [x] **Redis 自訂連接埠**
  - **實際影響**：設定頁可配置 Redis 執行 Port（預設 6379），影響服務啟停與內建瀏覽器連線；資料庫瀏覽器顯示依設定埠號連線。
  - **關聯 Commit**：`2b640ef` `feat(redis): support custom port for redis in setting, optimize redis display in database explorer`
  - **核心變更檔案**：`frontend/src/components/Settings.tsx`, `frontend/src/components/DBExplorer.tsx`, `internal/redisexplorer/explorer.go`, `frontend/src/i18n.ts`
  - **查驗指令**：`git show 2b640ef --stat`

- [x] **側邊欄收合簡化**
  - **實際影響**：移除鎖定狀態與頁籤切換自動收合；收合狀態持久化於 `localStorage`（`wincmp_sidebar_collapsed`）。
  - **關聯 Commit**：`84c15d7` `feat(core): implement v2.1.2 features with redis explorer and port release guarantee`
  - **核心變更檔案**：`frontend/src/App.tsx`, `frontend/src/i18n.ts`
  - **查驗指令**：`git show 84c15d7 -- frontend/src/App.tsx`

### [Changed]
- [x] **儀表板 PHP 卡片版面**
  - **實際影響**：PHP FastCGI 卡片改為頂部版本＋狀態、下方埠口與 Redis 徽章的版面，資訊層級更清楚。
  - **關聯 Commit**：`0131e95` `style(dashboard): update php cards style`
  - **核心變更檔案**：`frontend/src/components/Dashboard.tsx`
  - **查驗指令**：`git show 0131e95 --stat`

- [x] **依賴管理 Redis 狀態判定**
  - **實際影響**：修正假陽性，僅 DLL 存在且 `php.ini` 有效啟用時才顯示已就緒。
  - **關聯 Commit**：`84c15d7` `feat(core): implement v2.1.2 features with redis explorer and port release guarantee`
  - **核心變更檔案**：`frontend/src/components/DependencyManager.tsx`
  - **查驗指令**：`git show 84c15d7 -- frontend/src/components/DependencyManager.tsx`

### [Fixed]
- [x] **Runtime 啟動指令安全防護**
  - **實際影響**：攔截 `start` 前綴與 `cmd start`／`Start-Process` 脫鉤指令，避免脫離 Job Object 監控；提示改用原生前台指令。
  - **關聯 Commit**：`84c15d7` `feat(core): implement v2.1.2 features with redis explorer and port release guarantee`；`72a0c51` `fix(core): harden runtime command safety and port release guarantees`
  - **核心變更檔案**：`internal/process/runtime.go`, `internal/process/runtime_test.go`, `bridge.go`
  - **查驗指令**：`git show 72a0c51 --stat`

- [x] **Port 釋放保證**
  - **實際影響**：`KillProcessByPort` 後同步輪詢 TCP 釋放（約 150ms × 最多 3 秒）；`StartRuntime` 啟動前檢查埠號佔用，降低 `EADDRINUSE` 與跳號。
  - **關聯 Commit**：`84c15d7` `feat(core): implement v2.1.2 features with redis explorer and port release guarantee`；`72a0c51` `fix(core): harden runtime command safety and port release guarantees`
  - **核心變更檔案**：`internal/process/runtime.go`, `internal/scanner/scanner.go`, `internal/config/init.go`, `frontend/src/components/Dashboard.tsx`
  - **查驗指令**：`git show 72a0c51 --stat`

### [Dependencies]
- [x] **Go Redis 用戶端**
  - **實際影響**：後端引入 `github.com/redis/go-redis/v9`，支撐內建 Redis 瀏覽器。
  - **關聯 Commit**：`84c15d7` `feat(core): implement v2.1.2 features with redis explorer and port release guarantee`
  - **核心變更檔案**：`go.mod`, `go.sum`
  - **查驗指令**：`git show 84c15d7 -- go.mod go.sum`

---

## 2. 未納入發布說明的 Commit 查驗

| Commit | 類型 | 標題 | 未納入原因 |
| :--- | :--- | :--- | :--- |
| `1ace114` | `docs` | docs(release): require linear main history and ff-only release flow | 發布流程文件與 skill 調整，非終端使用者變更 |
| `a803737` | `docs` | docs(roadmap): add macOS porting feasibility evaluation and naming proposals | 內部可行性評估文件 |
| `0a6ac65` | `docs` | docs(plan): v2.1.2 plan | 開發規格文件，功能已拆入上方 Added/Fixed 條目 |
| `0131e95` 部分 | `feat` | 產出 Wails 綁定與 models 更新 | 自動產生程式碼，服務於 Redis / PHP 狀態 API，非獨立功能 |

---

## 3. 發布前動作確認清單

- [ ] 1. 條目代碼核對通過（開發者覆核）
- [ ] 2. `node .agents/skills/wincmp-release/scripts/validate_release.js` 通過
- [ ] 3. `cd frontend && npm run build` 通過
- [ ] 4. `go test ./...` 通過
- [ ] 5. `wails dev` 已啟動且 `scripts/capture_release_screenshots.ps1` 完成（開發者手動）
- [ ] 6. `.\release.bat` 成功生成 Release 壓縮包（開發者手動）

---

## 4. 建議 Git 發行指令（僅供審核，Agent 不執行 push/tag）

比對基準 tag：`v2.1.1`；目標版號：`v2.1.2`；目前分支：`feature/version-v2.1.2`。

```bash
# 1) 在發行分支提交發布內容
git add VERSION release_note/v2.1.2/
git commit -m "chore(release): bump version to v2.1.2"

# 2) 更新 main
git checkout main
git pull origin main

# 3) 線性 rebase 發行分支
git checkout feature/version-v2.1.2
git rebase main

# 4) fast-forward main（禁止 merge commit）
git checkout main
git merge --ff-only feature/version-v2.1.2

# 5) 確認
git show HEAD:VERSION
git ls-tree --name-only HEAD release_note/v2.1.2/
git log --oneline --merges -1

# 6) annotated tag 必須指向 main 上的 commit
git tag -a v2.1.2 -m "Release version 2.1.2"

# 7) 先推 main，再推 tag
git push origin main
git push origin v2.1.2
```

> 截圖（`screenshot/{dark,sketch}`）與 `release.bat` 打包完成後，若有變更，請一併納入步驟 1 的 commit。
