# Release Audit Checklist: WinCMP v2.1.7

> **審核說明**：本檔案為發行前代碼變更之真實性、完整性核對底稿。

- **比對基準**：`v2.1.6..HEAD`
- **發行日期**：2026-10-22
- **變更性質**：安全應變（PHP CVE-2026-109618 / CVE-2026-109619）

## 1. 變更條目與代碼核對矩陣 (Audit Matrix)

### [Security]
- [x] **PHP 重大漏洞修補（8.5.12 / 8.4.27 / 8.3.36 / 8.2.35）**
  - **實際影響**：內建 PHP 依賴目錄指向修補版；使用者重新安裝即可取得修補二進位。
  - **關聯 Commit**：`f8b1a4e` `fix(deps): bump bundled PHP to patched releases and add PHP 8.5`
  - **核心變更檔案**：`conf/dependencies.json`
  - **查驗指令**：`go run scripts/check_deps.go --check --fast`（19/19 通過）、SHA-256 與官方 `sha256sum.txt` 逐項相符

### [Added]
- [x] **PHP 8.5 執行環境**
  - **實際影響**：依賴庫新增 `php85` 與 `php_redis_85`，掃描器與前端以既有通用邏輯辨識。
  - **關聯 Commit**：`f8b1a4e` `fix(deps): bump bundled PHP to patched releases and add PHP 8.5`、`0676082` `fix(deps): derive php_redis target version from key suffix`
  - **核心變更檔案**：`conf/dependencies.json`、`downloader_bridge.go`
  - **查驗指令**：`git show f8b1a4e -- conf/dependencies.json`
- [x] **PHP 生命週期標示（EOL / 即將停止支援）**
  - **實際影響**：`ScanResult.PHPLifecycle` 帶出官方 EOL 日期；Dashboard 與依賴庫顯示徽章與 tooltip。
  - **關聯 Commit**：`c96b774` `feat(php): surface EOL status and clean up superseded patch runtimes`
  - **核心變更檔案**：`internal/phpver/phpver.go`、`internal/scanner/scanner.go`、`frontend/src/phpLifecycle.ts`、`frontend/src/components/PhpLifecycleBadge.tsx`
  - **查驗指令**：`go test ./internal/phpver/ -v`
- [x] **舊版執行環境偵測與清理**
  - **實際影響**：曝露 `SkippedPHP`，同分支有較新 patch 時提供清理按鈕；後端 `RemoveStalePHPPatch` 先重掃、再以白名單與路徑邊界檢查把關。
  - **關聯 Commit**：`c96b774`
  - **核心變更檔案**：`downloader_bridge.go`、`frontend/src/components/DependencyManager.tsx`
  - **查驗指令**：`git show c96b774 -- downloader_bridge.go`

### [Changed]
- [x] **依賴庫列表精簡與動效邊界**
  - **實際影響**：移除 `DependencyManager` 列表標題旁的授權 Badge；`Dashboard` 的 `handleScan` 區分主動點擊與背景回呼，背景刷新不再觸發掃描動效；`AGENTS.md` 補上對應 UI/UX 原則。
  - **關聯 Commit**：`8c41481` `refactor(ui): streamline dependency list titles and constrain rescan animation`
  - **核心變更檔案**：`frontend/src/components/DependencyManager.tsx`、`frontend/src/components/Dashboard.tsx`、`AGENTS.md`
  - **查驗指令**：`git show 8c41481 --stat`

### [Dependencies]
- [x] **Wails 2.17.0 / go-webview2 1.0.23**
  - **實際影響**：CLI 內嵌的 x/tools 已能解析 Go 1.27 型別資訊（原 v2.12.0 於 `go mod tidy` 與 bindings 產生階段會出現 `package ... without types`）；go-webview2 v1.0.23 修正 `PutShouldDetectMonitorScaleChanges` 的 COM 參數傳遞。
  - **關聯 Commit**：`39b14c7` `chore(deps): upgrade wails to v2.17.0 and go-webview2 to v1.0.23`
  - **核心變更檔案**：`go.mod`、`go.sum`
  - **查驗指令**：`go build ./...`、`go test ./...`
- [x] **預設依賴版本更新（Caddy / Node / MariaDB / Mailpit）**
  - **實際影響**：依賴庫預設下載版本提升；Node.js 24.22.1 為上游安全更新。
  - **關聯 Commit**：`f8b1a4e`
  - **核心變更檔案**：`conf/dependencies.json`、`docs/dependencies.md`
  - **查驗指令**：`git show f8b1a4e -- conf/dependencies.json`
- [x] **CA 憑證改釘不可變快照**
  - **實際影響**：`cacert` 改指向 `cacert-2026-10-20.pem`，避免上游刷新滾動檔造成校驗失敗；下載後仍存為 `conf/ssl/cacert.pem`。
  - **關聯 Commit**：`f8b1a4e`
  - **核心變更檔案**：`conf/dependencies.json`
  - **查驗指令**：`git show f8b1a4e -- conf/dependencies.json`

### [Fixed]
- [x] **PHP 同分支版本比較修正**
  - **實際影響**：`8.4.9` 不再被誤判為比 `8.4.10` 新；去重與清單排序皆改數值比較。
  - **關聯 Commit**：`a49bb23` `fix(scanner): compare PHP versions numerically to pick newest patch`
  - **核心變更檔案**：`internal/scanner/scanner.go`、`internal/scanner/scanner_test.go`
  - **查驗指令**：`go test ./internal/scanner/ -run "TestScanBinDirPHPVersionOrdering|TestCompareVersions" -v`
- [x] **php_redis 擴充安裝目標修正**
  - **實際影響**：未知後綴不再產生空字串前綴，避免把 ABI 不符的 DLL 複製進所有 PHP 目錄。
  - **關聯 Commit**：`0676082`
  - **核心變更檔案**：`downloader_bridge.go`、`internal/i18n/i18n.go`
  - **查驗指令**：`git show 0676082 -- downloader_bridge.go`
- [x] **依賴連結歸檔備援**
  - **實際影響**：`releases/` 連結失效時自動改用 `releases/archives/`；判定改以 `php.net` 網域通用，直連 `downloads.php.net` 亦適用。
  - **關聯 Commit**：`52e99a0` `fix(ci): optimize dependency validation and resolve php archive 404s`、`56043a4` `fix(downloader): generalize php archives fallback to php.net hosts`
  - **核心變更檔案**：`internal/downloader/downloader.go`、`scripts/check_deps.go`
  - **查驗指令**：`git show 56043a4 -- internal/downloader/downloader.go`

## 2. 未納入發布說明的 Commit 查驗

| Commit | 類型 | 標題 | 未納入原因 |
| :--- | :--- | :--- | :--- |
| `49d1a24` | docs | docs: sync dependency versions, fix bundled PHP path layout and add v2.1.7 plan | 純文件同步，無使用者可見行為變更 |
| `d18429d` | docs | docs: reorganize plans under docs/plan and archive legacy docs | 純目錄整理，不影響執行 |
| `c6b38f2` | docs | docs(plan): condense v2.1.7 plan into done, remaining and out-of-scope | 內部計畫文件 |
| `edd7472` | chore | chore(release): bump version to v2.1.7 | 版號提升，內容已由 `VERSION` 與 release notes 呈現 |
| `254509a` | build | fix(git): pin go.mod and go.sum to LF line endings | 換行正規化，無行為變更 |

## 3. 發布前動作確認清單

- [x] 1. 條目代碼核對通過
- [x] 2. `go test ./...` 通過（`TestOpenFolderAndShowInExplorerValidation` 於受沙箱限制的環境會因無法啟動 `explorer.exe` 失敗，非程式缺陷）
- [x] 3. `go run scripts/check_deps.go --check --fast` 通過（19/19）
- [x] 4. `node .agents/skills/wincmp-release/scripts/validate_release.js` 通過（分類白名單與日期行驗證成功）
- [x] 5. `cd frontend && npm run build` 通過（已於 `task run-build` 的 Compiling frontend 階段驗證）
- [x] 6. 本次 UI 無重大變更，不重新擷取截圖（沿用既有 `screenshot/`）
- [ ] 7. `.\release.bat` 成功生成 Release 壓縮包
