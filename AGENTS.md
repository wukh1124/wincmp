# WinCMP - Agent 開發指南

本檔案為專案唯一 Agent 指令來源，收錄核心規範、架構邊界與易踩地雷。一般環境安裝與使用者說明請參閱 `readme.md`。

---

## 1. 核心開發約束與指令

* **版本號規範**：發布或編譯版本動態讀取 `VERSION` 檔，**禁止寫死版號**。
  * 範例：`$ver = (Get-Content VERSION).Trim(); wails build -clean -ldflags "-s -w -X main.AppVersion=v$ver"`
* **測試驗證**：代碼變更後務必執行測試：`go test ./...`（或指定套件如 `go test ./internal/process`）。
* **清理原則**：**嚴禁**提交編譯產物與暫存檔（`*.exe`、`frontend/dist/`、`*.log` 等）。
* **腳本技術棧規範**：專案工具、發布驗證與輔助腳本**一律使用 Node.js**（置於 `scripts/*.js`），**嚴禁新增或調用 .ps1 / .bat**（避免受 Windows ExecutionPolicy 阻礙與編碼歧異）。

---

## 2. 模組架構與目錄邊界

```text
wincmp/
├── main.go, app.go, bridge.go   # Wails 生命週期、事件推送與 Go-TS 核心 RPC 綁定
├── conf/                        # 預設設定檔 (wincmp.json, Caddyfile, my.ini 等)
├── bin/                         # 服務與相依二進位套件目錄 (隔離環境)
├── scripts/                     # 維護與自動化工具 (統一 Node.js，如 capture.js, sync-website.js)
├── internal/                    # Go 後端核心邏輯 (純業務與系統底層，不含 GUI)
│   ├── config/                  # 設定檔讀寫 (conf/wincmp.json)
│   ├── detect/                  # 輔助偵測 (僅限 Laravel 信心分數與版本評估)
│   ├── hosts/                   # 系統 Hosts 管理 (寫入前必須備份)
│   ├── i18n/                    # 後端繁中字典與英文對照
│   ├── preset/                  # 專案型別 Preset 主表、啟動指令生成、主偵測 (DetectProjectPreset)
│   ├── process/                 # 子行程生命週期 (Job Object 綁定清理、日誌管線)
│   └── scanner/                 # bin/ 掃描與 PHP Port 基數計算 (calcPHPPortBase)
└── frontend/                    # React 18 + TSX + Zustand (所有新 GUI 功能在此實作)
```

---

## 3. 前後端通訊與介面規範 (地雷警戒區)

### 3.1 事件推送與致命禁令

* **單向推送**：Go 端呼叫 `runtime.EventsEmit(a.ctx, eventName, data)`。
* **前端監聽**：**必須**使用 `EventsOn` 回傳的 `unsubscribe` 進行註銷。
  ```tsx
  useEffect(() => {
    const unsubscribe = EventsOn("resource_usage", (data) => { /* 處理資料 */ });
    return () => unsubscribe();
  }, []);
  ```
* **【致命規範】全面禁用全域 `EventsOff(eventName)`**：
  Wails v2 的 `EventsOff` 會註銷該事件名下**所有**監聽器（例如誤傷全域 `logStore`），造成日誌與監控永久失效。

### 3.2 彈出視窗規範

* **禁止**原生 `window.alert()` / `window.confirm()`（防出現 `wails.localhost 說` 對話框）。
* **統一使用全域自訂視窗**：
  * Alert：`await (window as any).customAlert("提示訊息")`
  * Confirm：`const ok = await (window as any).customConfirm("確認訊息")`

### 3.3 多國語言 (i18n) 與 UI 規範

* **Key 規範**：前後端統一以 **繁體中文 (zh-TW)** 作為翻譯字典 Key（未命中時直接回退顯示 Key）。
* **雙向補齊**：新增任何可見文字/錯誤訊息，必須同步於 `internal/i18n/i18n.go` 與 `frontend/src/i18n.ts` 補齊英文翻譯。
* **禁 Emoji**：全站 UI、彈窗與註解**一律不使用 Emoji**，圖標請用套件圖示或 SVG。
* **官網品牌圖標**：`website/index.html` 的品牌圖標（如 GitHub）請使用 Inline SVG（Lucide v1.0.0 已移除品牌圖標）。

---

## 4. 系統底層與業務陷阱

### 4.1 環境變數與路徑隔離

* 傳遞給 Caddyfile 的 Windows 路徑一律將反斜線轉為正斜線：`strings.ReplaceAll(path, "\\", "/")`。
* 啟動 PHP-CGI、Node、Python 等進程時，**嚴禁修改系統全域 PATH**；一律將對應 `bin/` 目錄動態注入至 `exec.Cmd.Env`。

### 4.2 專案偵測職責劃分

本專案有兩條獨立的偵測路徑，切勿混淆：
1. **主偵測** (`internal/preset/preset.go` → `DetectProjectPreset`)：
   * 負責判斷專案型別、Runtime、預設 Port 及啟動指令。新增專案型別與指令調整一律改此處。
   * 操作 Preset 時必須使用查詢函式與 `Normalize*()`，切勿直接修改 map。
2. **輔助偵測** (`internal/detect/laravel.go` → `DetectLaravel`)：
   * 僅在型別確定或需判斷 Laravel 時提供信心評分與推薦 PHP 版本。
   * **切勿**在此套件新增 Preset 型別或定義 Runtime/Port。

### 4.3 子行程管理 (`internal/process`)

* **Mutex 邊界**：`m.mu` 僅用於保護 `services` map 存取；**嚴禁在鎖內執行任何 I/O 操作**（如 `pipeOutput`、`Process.Kill`、等待進程等）。
* **啟動清理**：啟動多進程服務中途若有任一失敗，必須主動清理已啟動之子進程。
* **PHP Port 計算**：公式為 `30000 + major*1000 + minor*100`，計算邏輯位於 `internal/scanner`，不在 `process`。
* **錯誤處理**：Go 端錯誤包裝統一使用 `fmt.Errorf("...: %w", err)`。

---

## 5. 安全防護與供應鏈防禦規範

所有涉及網路下載、執行檔替換、外部輸入或子進程啟動的新功能與重構，必須嚴格遵守以下安全規範：

### 5.1 遠端下載與白名單約束
* **強制 HTTPS 與官方網域**：嚴禁自非信任來源下載檔案。
  * 依賴下載 (`internal/downloader`)：僅允許 `downloader.TrustedHostSuffixes` 官方白名單（`github.com`, `curl.se`, `mariadb.org`, `nodejs.org`, `php.net` 等）。
  * 主程式更新 (`internal/updater`)：僅允許官方 GitHub Release 位址（前綴為 `https://github.com/wukh1124/wincmp/releases/download/`）。

### 5.2 完整性與格式雙重校驗 (嚴禁略過)
* **強制 SHA-256**：下載後解壓或執行前，**必須**透過 `downloader.CalculateSHA256` 進行雜湊比對。**嚴禁**允許空 Hash 略過校驗；比對不符必須立即刪除臨時檔並中斷流程。
* **Windows PE 格式檢驗**：覆蓋執行檔前，必須調用 `ValidateExecutablePE` 檢查檔案大小門檻及前 2 位元組 `MZ` 標頭，防止 404 HTML 或錯誤文字檔損毀主程式。
* **發布資產不可變與 Checksums**：Release 發布必須隨附 `checksums.txt`，且 GitHub 倉庫需維持開啟 `Enable release immutability`。

### 5.3 命令注入與權限防護
* **Runtime 命令淨化**：所有專案自訂指令必須經 `SanitizeRuntimeCommand` 檢查，防範 Shell 惡意字元注入與 `cmd start` / `Start-Process` 脫鉤繞過。
* **路徑防遍歷**：解壓縮必須嚴格防範 Zip Slip，禁止包含 `..` 的相對路徑穿越。

---

## 6. Agent 交付前自檢清單 (Checklist)

每次完成代碼變更交付前，請確認：
- [ ] 無新產生之編譯產物或垃圾檔（`*.exe`, `dist/`, 臨時日誌）留存於 git 暫存區。
- [ ] 未使用全域 `EventsOff`；前端事件監聽已正確調用 `unsubscribe`。
- [ ] 未使用原生 `alert()` / `confirm()`；彈窗皆走 `customAlert` / `customConfirm`。
- [ ] 新增字串皆透過 i18n 機制，繁中為 Key 且英譯已補齊；無新增 Emoji。
- [ ] 未更動系統全域 PATH，路徑正斜線轉換正確。
- [ ] 涉及 `internal/preset` 或 `internal/detect` 時職責邊界清楚，無混用。
- [ ] 涉及檔案下載或更新時，落實官方 URL 白名單、強制 SHA-256 比對與 PE 標頭驗證。
- [ ] 執行 `go test ./...` 零錯誤。

