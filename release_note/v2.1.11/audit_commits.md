# Release Audit Checklist: WinCMP v2.1.11

> **審核說明**：本檔案為發行前代碼變更之真實性、完整性核對底稿。

- **比對基準**：`v2.1.10..HEAD`
- **發行日期**：2026-11-25

## 1. 變更條目與代碼核對矩陣 (Audit Matrix)

### [Added]
- [x] **PHP 專案內建 Composer 控制台**
  - **實際影響**：PHP / Laravel 專案可於操作選單開啟 Composer 控制台，自動對應專案 PHP 版本執行指令並保存歷史紀錄。
  - **關聯 Commit**：`4c0fa76` `feat(composer): add PHP project Composer console`
  - **核心變更檔案**：`composer_bridge.go`, `internal/composer/runner.go`, `internal/composer/history.go`, `internal/composer/buffer.go`, `frontend/src/components/ComposerModal.tsx`, `frontend/src/components/Projects.tsx`
  - **查驗指令**：`git show 4c0fa76 --stat`
- [x] **儀表板依優先度顯示單一警告橫幅**
  - **實際影響**：依序偵測 Caddy 未安裝、啟用中專案所需 PHP 未安裝、有啟用中專案時 Caddy 未啟動、啟用中專案所需 PHP 未啟動，同時僅顯示一條並支援一鍵下載或啟動。
  - **關聯 Commit**：`760d9da` `feat(dashboard): prioritize php missing/stopped warnings and remove duplicate icon`
  - **核心變更檔案**：`frontend/src/components/Dashboard.tsx`, `frontend/src/i18n.ts`, `internal/i18n/i18n.go`
  - **查驗指令**：`git show 760d9da --stat`
- [x] **開放 Python HTTP 專案類型並修正 Python 啟動指令分派**
  - **實際影響**：前端選單開放 `Python HTTP`（`"python"`，預設埠口 `8000`），且 `buildRuntimeCommand` 改為依所選 Preset 產生對應啟動指令，不再於啟動時重新掃描目錄覆蓋使用者選擇。
  - **關聯 Commit**：`f86590c` `feat(preset): expose Python HTTP preset and respect selected Python runtime command`
  - **核心變更檔案**：`frontend/src/components/Projects.tsx`, `internal/preset/preset.go`, `internal/process/runtime.go`, `internal/process/runtime_test.go`
  - **查驗指令**：`git show f86590c --stat`

### [Changed]
- [x] **專案管理操作列改為下拉選單**
  - **實際影響**：操作欄收折為下拉選單並使用 Portal 定位。
  - **關聯 Commit**：`f953e70` `refactor(projects): collapse row action buttons into dropdown menu`
  - **核心變更檔案**：`frontend/src/components/Projects.tsx`
  - **查驗指令**：`git show f953e70 --stat`
- [x] **服務啟動與還原狀態並行同步**
  - **實際影響**：加速啟動還原並透過 `GetServicesStateSnapshot` / `services_status_changed` 與前端快取消除按鈕閃爍。
  - **關聯 Commit**：`af99ebf` `perf(startup): eliminate start button flash with parallel restore and state push`
  - **核心變更檔案**：`app.go`, `bridge.go`, `frontend/src/components/Dashboard.tsx`, `frontend/src/components/Projects.tsx`, `internal/scanner/scanner.go`
  - **查驗指令**：`git show af99ebf --stat`

### [Fixed]
- [x] **修復自動更新重啟時的競態與狀態遺失**
  - **實際影響**：確保 `--restart` 交接等待舊進程與子服務釋放埠口、單實例鎖不互卡、設定檔原子寫入且保留 `last_service_state`。
  - **關聯 Commit**：`d1864f5` `fix(updater): prevent port race and state loss across self-update restarts`
  - **核心變更檔案**：`main.go`, `app.go`, `bridge.go`, `internal/config/config.go`, `internal/process/procscan.go`, `internal/singleinstance/singleinstance.go`, `frontend/src/components/Dashboard.tsx`
  - **查驗指令**：`git show d1864f5 --stat`
- [x] **防止手動停止服務時誤報異常結束日誌**
  - **實際影響**：手動停止各服務與專案 Runtime 不再誤記異常退出日誌。
  - **關聯 Commit**：`cb8b2c4` `fix(process): prevent false abnormal exit logs during manual service stop`
  - **核心變更檔案**：`internal/process/manager.go`, `internal/process/php.go`, `internal/process/caddy.go`, `internal/process/mariadb.go`, `internal/process/mailpit.go`, `internal/process/redis.go`, `internal/process/runtime.go`
  - **查驗指令**：`git show cb8b2c4 --stat`
- [x] **動態過濾不相容的 PHP 擴充模組與修正 Composer 進度條輸出**
  - **實際影響**：跨 PHP 版本共用 `php.ini` 不會因缺少特定擴充 DLL 失敗；Composer 進度條不重複洗版且子腳本繼承 `PHPRC`。
  - **關聯 Commit**：`69745d4` `fix(php): resolve version-compatible php.ini extensions dynamically`, `b550b8f` `fix(composer): normalize progress bar overwrites and propagate PHPRC to scripts`
  - **核心變更檔案**：`internal/config/init.go`, `internal/composer/runner.go`, `internal/process/php.go`, `frontend/src/components/ComposerModal.tsx`
  - **查驗指令**：`git show 69745d4 b550b8f --stat`
- [x] **移除儀表板警告重複圖示並調淺 Sketch 警告底色**
  - **實際影響**：警告標題僅留單一圖示；Sketch 主題 `--status-warn-bg` 調淺為 `#fff8d6`。
  - **關聯 Commit**：`760d9da` `feat(dashboard): prioritize php missing/stopped warnings and remove duplicate icon`
  - **核心變更檔案**：`frontend/src/components/Dashboard.tsx`, `frontend/src/themes.css`
  - **查驗指令**：`git show 760d9da -- frontend/src/themes.css frontend/src/components/Dashboard.tsx`

### [Dependencies]
- [x] **新增 PHP 8.6.0 (VS18) 至依賴目錄**
  - **實際影響**：依賴管理器支援一鍵下載與校驗 PHP 8.6.0。
  - **關聯 Commit**：`d12c3fd` `feat(deps): add PHP 8.6.0 (vs18) to dependency catalog`
  - **核心變更檔案**：`conf/dependencies.json`, `docs/dependencies.md`, `internal/phpver/phpver.go`, `internal/phpver/phpver_test.go`, `.agents/skills/wincmp-dependencies-update/SKILL.md`
  - **查驗指令**：`git show d12c3fd --stat`

## 2. 未納入發布說明的 Commit 查驗
| Commit | 類型 | 標題 | 未納入原因 |
| :--- | :--- | :--- | :--- |
| （無） | - | - | `v2.1.10..HEAD` 全部 10 筆 Commit 皆已核對並納入發布說明 |

## 3. 發布前動作確認清單
- [x] 1. 條目代碼核對通過
- [x] 2. `node .agents/skills/wincmp-release/scripts/validate_release.js` 通過
- [ ] 3. `cd frontend && npm run build` 通過
- [x] 4. `go test ./...` 通過
- [ ] 5. `wails dev` 已啟動且 `node scripts/capture.js --sync-website` 完成
- [ ] 6. `.\release.bat` 成功生成 Release 壓縮包
