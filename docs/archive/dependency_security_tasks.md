# 依賴自動下載安全加固計畫與任務清單

此文件記錄了針對 WinCMP 自動下載功能的安全性加固計畫。主要目的是防止中間人攻擊 (MITM)、DNS 污染、設定檔劫持，以及惡意軟體在傳輸或發布端對依賴二進位檔案進行調包。

---

## 📋 任務清單 (Task List)

### 1. 實作 SHA-256 檔案完整性校驗
- [x] **擴充 `dependencies.json` 結構**
  - 在 `internal/config/dependencies.go` 中的 `DependencyItem` 結構體新增 `SHA256` 欄位。
  - 更新本機嵌入的 `conf/dependencies.json`，手動或自動填入目前支援版本（Caddy, PHP, MariaDB, Node.js 等）的官方 SHA-256 值。
- [x] **實作 SHA-256 計算邏輯**
  - 在 `internal/downloader/downloader.go` 中新增輔助函式，用於計算指定本機檔案的 SHA-256 值。
- [x] **在解壓前進行校驗**
  - 在 `downloader_bridge.go` 的 `runDependencyDownloadPipeline` 流程中，於下載完成後、調用 `downloader.Unzip` 解壓之前，計算下載得到的 `.zip` / `.phar` 的 SHA-256 值。
  - 將計算出的值與設定檔中的預期值進行比對，若不一致，則刪除下載檔案、拒絕解壓，並向前端回報校驗失敗錯誤。

### 2. 實作遠端設定檔數位簽章驗證 (防 DNS 污染與網址竄改)
- [ ] **金鑰對生成與金鑰嵌入**
  - 生成一對非對稱加密金鑰（如 Ed25519 或 RSA）。
  - 將公鑰 (Public Key) 以硬編碼 (Hardcode) 的方式嵌入至 Go 後端代碼中。
- [ ] **簽章驗證流程實作**
  - 在 `FetchRemoteDependencies` 時，除了下載 `dependencies.json` 外，一併從伺服器請求下載簽章檔 `dependencies.json.sig`。
  - 在解析與儲存遠端設定檔前，使用內建的公鑰驗證下載的 `dependencies.json` 與 `dependencies.json.sig` 是否相符。
  - 若驗證失敗，放棄使用該遠端設定檔，並回退 (Fallback) 使用本地嵌入的預設設定檔。

### 3. 下載網域白名單過濾
- [ ] **定義安全網域白名單**
  - 在 Go 後端定義一個只允許下載依賴的官方安全網域白名單（例如：`github.com`, `windows.php.net`, `nodejs.org`, `archive.mariadb.org`, `curl.se` 等）。
- [ ] **下載前網域檢查**
  - 在 `DownloadDependency` 執行下載前，先解析依賴項 URL 的 Host，驗證其是否在白名單內。
  - 若 URL 的網域不在白名單中，則拒絕下載，防止被重導向至惡意未知網域。

---

## 🛠️ 自動化檢測與計算腳本用法

專案提供了一個自動化校驗與更新工具 [check_deps.go](/scripts/check_deps.go)，用於維護 `conf/dependencies.json` 中所有外部依賴下載連結的有效性與 SHA-256 雜湊值。

### 執行方式
在專案根目錄下，開啟終端機（如 PowerShell 或 Cmd）執行：

#### 1. 唯讀校驗模式 (適用於 CI/CD / GitHub Actions)
檢查所有依賴項的下載連結是否可用，並重新下載校驗其 SHA-256 是否與程式配置一致。若有任何出錯或雜湊不匹配，將會輸出錯誤日誌並以 `exit 1` 結束（此模式已被整合進 GitHub Actions 流程中）：
```powershell
go run scripts/check_deps.go --check
```

#### 2. 自動更新模式 (適用於依賴版本升級時)
自動檢查並下載所有**未配置 SHA-256 值**的依賴項目，計算其雜湊值後寫回更新至 `conf/dependencies.json`。已配置過的項目預設會自動跳過，以節省時間與頻寬。
```powershell
go run scripts/check_deps.go --update
```

#### 3. 強制更新模式
如果需要強制重新下載所有項目並重新計算 SHA-256，可以加上 `--force` 參數：
```powershell
go run scripts/check_deps.go --update --force
```
