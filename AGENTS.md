# WinCMP - Agent 開發指南

本檔案為專案唯一 Agent 指令來源，提供在此程式碼庫開發與維護時所需的架構、規範與套件速查。

---

## 1. 開發與建置環境

### 前置需求

* **Go**：Go 1.26.2+（以 `go.mod` 為準）
* **Wails CLI**：Wails v2（目前依賴 `github.com/wailsapp/wails/v2 v2.12.0`）。未安裝時：`go install github.com/wailsapp/wails/v2/cmd/wails@latest`
* **C 編譯器**：MinGW-w64 (WinLibs)，請確認 `gcc -v` 可執行
* **Node.js**：Node.js 18+（前端開發與打包）

### 開發熱重載

```powershell
wails dev
```

### 建置編譯

```bash
go mod tidy
cd frontend; npm install; cd ..

# 開發/偵錯建置
wails build -debug

# 正式發布（無主控台，產出 wincmp.exe）
wails build -clean

# 壓縮並移除 symbols（動態讀取 VERSION，禁止寫死版號）
# PowerShell:
$ver = (Get-Content VERSION).Trim(); wails build -clean -ldflags "-s -w -X main.AppVersion=v$ver"

# 自動化完整發布打包（產生 zip 與獨立 exe 於 ../wincmp-release-only/）
.\release.bat
```

### 常用維護指令（統一採用 Node.js）

```bash
# 自動化截圖（需先執行 wails dev；自動備份舊圖至 screenshot/backup/）
node scripts/capture.js

# 截圖並直接同步至官網目錄
node scripts/capture.js --sync-website

# 本地官網資源同步（複製 icon、screenshot 並生成 website/release.json）
node scripts/sync-website.js

# 發行前驗證器
node .agents/skills/wincmp-release/scripts/validate_release.js
```

### 測試

* 全域 Go 測試：`go test ./...`
* 指定套件：`go test ./internal/config`、`go test ./internal/process` 等

---

## 2. 專案架構與目錄佈局

```text
wincmp/
├── main.go                 # 進入點：初始化與啟動 Wails
├── app.go                  # Wails 生命週期 (startup/shutdown)、日誌/監控推送
├── bridge.go               # Wails 與 Go 主 Binding API (前後端 RPC)
├── tray.go                 # 系統匣
├── downloader_bridge.go    # 下載管理器 Binding
├── conf/                   # 系統配置 (Caddyfile, wincmp.json, php/, my.ini 等)
├── bin/                    # 二進位套件 (Caddy, MariaDB, PHP-CGI, Mailpit, Node 等)
├── scripts/                # 專案工具腳本（統一使用 Node.js，如 capture.js, sync-website.js）
├── bat/                    # 發布批次檔 (release.ps1, release.bat) 與歷史舊腳本
├── website/                # 介紹官網（GitHub Pages 靜態網站）
├── internal/               # Go 後端核心（不含 GUI 邏輯）
│   ├── config/             # wincmp.json 讀寫、依賴設定、php.ini 優化
│   ├── crypto/             # DPAPI 等加密封裝
│   ├── detect/             # Laravel 信心分數與版本輔助偵測（非專案型別主偵測）
│   ├── downloader/         # 依賴下載
│   ├── hosts/              # 系統 Hosts 管理（同步與 UAC 備份）
│   ├── i18n/               # 後端多國語言字典
│   ├── port/               # 埠號工具
│   ├── preset/             # 專案型別 Preset 主表、指令生成、主偵測 DetectProjectPreset
│   ├── process/            # 子行程生命週期 (Job Object)：Caddy/MariaDB/PHP/Mailpit/Redis/Runtime
│   ├── resource/           # CPU & RAM 監控
│   ├── scanner/            # bin/ 掃描、PHP Port 計算 (calcPHPPortBase)
│   ├── singleinstance/     # 單一實例
│   ├── terminal/           # 專案終端 (conpty)
│   └── updater/            # 版本更新
├── frontend/               # React + TSX
│   └── src/
│       ├── wailsjs/        # Wails 自動產生的 Go 綁定（勿手動修改）
│       ├── components/     # Dashboard, Projects, DBExplorer, Settings 等
│       ├── stores/         # Zustand
│       ├── i18n.ts         # 前端翻譯字典
│       └── App.tsx         # 掛載 customAlert / customConfirm
└── legacy_fyne/            # 已歸檔舊 Fyne 程式碼（僅供移植參考，勿在此開發）
```

---

## 3. 前後端通訊與開發規範

### 3.1 Go 綁定方法

* `bridge.go` / `app.go` / `downloader_bridge.go` 中 `*App` 的 **PascalCase** 匯出方法，Wails 編譯時自動產生前端 SDK。
* 前端呼叫範例：

```tsx
import { GetConfig, StartCaddy } from '../wailsjs/go/main/App';

const config = await GetConfig();
```

### 3.2 背景事件推送

* **單向推送**：Go 端用 `runtime.EventsEmit` 推送（資源監控、終端日誌等）。
* **Go 端**：

```go
runtime.EventsEmit(a.ctx, "resource_usage", map[string]interface{}{
    "cpu":    cpuPercent,
    "memory": memoryMB,
})
```

* **前端接收**：

```tsx
import { EventsOn } from '../wailsjs/runtime/runtime';

useEffect(() => {
  // EventsOn 回傳 unsubscribe，僅註銷此實例
  const unsubscribe = EventsOn("resource_usage", (data) => {
    // handle data
  });
  return () => {
    unsubscribe();
  };
}, []);
```

* **致命規範：全面禁用全域 `EventsOff(eventName)`**
  * Wails v2 的 `EventsOff` 會註銷該事件名下**所有**監聽器。
  * 組件卸載或切換語言時若呼叫 `EventsOff`，會誤傷全域 `logStore` 等監聽，造成日誌/監控永久失效。
  * **必須**使用 `EventsOn` 回傳的 `unsubscribe`；嚴禁直接 `EventsOff`。

### 3.3 程式碼風格與命名

* **後端 Go**：`gofmt`；錯誤包裝 `fmt.Errorf("...: %w", err)`；`sync.Mutex` 務必 `defer mu.Unlock()`。
* **前端 TypeScript/React**：變數/函式 `camelCase`，元件/介面 `PascalCase`；精確型別；全域狀態用 Zustand，避免 Prop drilling。

---

## 4. 核心業務邏輯與注意事項

### 4.1 多國語言 (i18n)

* **核心**：前後端統一以 **繁體中文 (zh-TW)** 作為翻譯字典 Key；未命中時直接顯示 Key（優雅降級）。
* **後端**：所有給使用者看的字串、錯誤與日誌，**必須** `i18n.T` / `i18n.Tfmt` 包裹；新增 Key 後於 `internal/i18n/i18n.go` 的英文對照補齊。
* **前端**：UI 可見文字、placeholder、彈窗訊息，**必須** `useLanguage` 的 `t()`；新增 Key 後於 `frontend/src/i18n.ts` 英文對照補齊。

```go
i18n.T("釋放預設設定檔失敗")
i18n.Tfmt("已自動刪除過期日誌檔: %s", name)
```

```tsx
import { useLanguage } from '../i18n';

const { t } = useLanguage();
return (
  <button title={t("刪除專案")}>
    {t("快速新增首個專案")}
  </button>
);
```

* **既有字典**可能含歷史符號；**新增** UI 文案與註解一律不使用 Emoji。

### 4.2 Windows 路徑與環境變數隔離

* 傳給 Caddyfile 的本機路徑使用正斜線：`strings.ReplaceAll(path, "\\", "/")`。
* 啟動 PHP-CGI / Node 等時，**絕對不要**改系統全域 `PATH`；必須將對應 bin 目錄動態 append 到該進程的 `exec.Cmd.Env`。

### 4.3 系統服務端口

* **PHP Port 基數**：實作在 `internal/scanner/scanner.go` 的 `calcPHPPortBase()`。  
  公式：`30000 + major*1000 + minor*100`（PHP 8.2 → 38200 起連續埠）。  
  每版本預設啟動多個 `php-cgi` 進程（Port 從基數遞增）。
* **Hosts 同步**：寫入 `C:\Windows\System32\drivers\etc\hosts` 前必須檢查是否需新增，並在寫入前備份。

### 4.4 全域自訂彈出視窗 (Alert/Confirm)

* **禁止**原生 `window.alert()` / `window.confirm()` / `confirm()`（避免 Windows 上出現 `wails.localhost 說` 對話框）。
* **Alert**：`(window as any).customAlert("提示訊息")`（Promise 非同步）。
* **Confirm**：`await (window as any).customConfirm("確認訊息")`（確定 `true`，取消 `false`）。
* 亦有 `customAlertWithCheckbox`（見 `frontend/src/App.tsx` 掛載）。

### 4.5 官網圖標與 Lucide

* 靜態官網 `website/index.html` 使用 Lucide 時：v1.0.0 起**品牌圖標已移除**（如 `github`），`<i data-lucide="github">` 會失敗。
* **品牌圖標必須使用 Inline SVG**；非品牌圖標（`menu`、`settings` 等）可繼續用 Lucide。

### 4.6 專案偵測分工（必讀）

本專案有**兩條**偵測路徑，不要混用或只改其中一份卻以為改完：

| 路徑 | 位置 | 職責 |
|------|------|------|
| **主偵測** | `preset.DetectProjectPreset(rootDir)` | 回傳專案型別 `Type`、`Runtime`、預設 `Port` |
| **輔助偵測** | `detect.DetectLaravel(rootDir)` | Laravel 信心分數與版本；`bridge.go` 在型別已是 Laravel 時用來推薦 PHP 版本 |

* 新增/調整「專案型別、啟動指令、Runtime 選項」→ 改 `internal/preset`。
* 只調整「Laravel 判定分數或版本解析」→ 改 `internal/detect`。
* `detect.DetectResult` **不含** Runtime/Port，欄位為：`IsLaravel`、`Confidence`、`Reasons`、`Version`、`Type`。

### 4.7 腳本技術棧規範（全面採用 Node.js，嚴禁濫用 .ps1 / .bat）

* **唯一指定技術棧**：專案輔助工具、發布校驗、資料處理、自動截圖等維護腳本，**一律優先使用 Node.js**（放置於 `scripts/*.js`）。
* **禁止新增 .ps1 / .bat**：Windows PowerShell 容易受 `ExecutionPolicy`（腳本執行策略受限）阻礙，且終端編碼易生歧異；Node.js 跨平台相容性高、編碼預設 UTF-8。
* **現有腳本清查與狀態盤點**：
  * **已改為 Node.js**：自動化截圖全面由 `scripts/capture.js` 取代（`capture_release_screenshots.ps1` 僅保留作為相容轉發）；官網資源同步由 `scripts/sync-website.js` 負責。
  * **真實流程使用中（保留穩定）**：發布打包 `bat/release.ps1` 與捷徑 `release.bat`（CI `release.yml` 亦調用之，核心建置與壓縮邏輯穩定運作）。
  * **歷史閒置待觀察（列為待清理）**：`bat/start-*.bat`、`bat/stop-services.bat`、`bat/zip-wincmp.bat` 屬早期手動測試殘留，現行代碼與流程無任何引用，列為觀察名單，新開發嚴禁調用。

---

## 5. 套件深潛速查

### 5.1 `internal/process` — 子行程生命週期

* **Job Object**：`job.go` 以 `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` 綁定主程序，崩潰時子行程自動清理。
* **服務 Key 格式**（`Manager.services` map）：

```text
caddy               → "caddy"
mariadb-{version}   → "mariadb-11.4"
mariadb-external    → "mariadb-external"
php-{version}       → "php-8.2.30"
runtime_{projectID} → "runtime_my-project"
mailpit             → "mailpit"
redis               → "redis"
```

* **日誌分類**：`system` / `caddy` / `mariadb` / `php` / `runtime` / `mailpit` / `redis`。
* **Mutex**：`m.mu` 僅保護 `services` map 存取；**禁止**在 Lock 內做 I/O（`pipeOutput`、`Process.Kill` 等）。
* **Context**：`register()` 建立 `context.WithCancel`；`unregister()` 呼叫 `Cancel()`。
* **啟動模板慣例**：檢查 `IsRunning` → `createCommand` → `pipeOutput` → `cmd.Start()` → `register` → `go waitForExit`。失敗時必須清理已啟動的子行程（尤其 PHP 多進程啟動中途失敗）。
* **檔案**：`manager.go`（核心）、`job.go`、`caddy.go`、`mariadb.go`、`php.go`、`mailpit.go`、`redis.go`、`runtime.go`。
* **PHP Port 計算不在 process**，在 `internal/scanner`。

### 5.2 `internal/preset` — 專案型別與指令

* **Project Type 常數（14）**：  
  `static`、`php`、`laravel`、`next`、`nuxt`、`astro`、`vite`、`python`、`python_django`、`python_fastapi`、`python_flask`、`go_api`、`pocketbase`、`custom`
* **Runtime 常數（8）**：  
  `auto`、`none`、`node`、`bun`、`python`、`go_air`、`go_run`、`custom`
* **DetectPriority**：數字**越小**優先度越高（Laravel=1 … PHP=90，Static=100，Custom=999）。
* **Auto Runtime**：偵測 `bin/bun` 存在則 Bun，否則 Node；真正解析責任常在呼叫端（如 `bridge.go` 還會查系統 PATH 與內建 bin）。
* **指令佔位符**：`%PORT%`、`%HOST%`、`%PROJECT_DIR%`、`%BIN_DIR%`。
* **常用 API**：`GetPreset` / `GetAllPresets` / `ResolveRuntime` / `BuildStartCommand` / `BuildStartCommandWithExePath` / `DetectProjectPreset` / `IsPythonType` / `NormalizeProjectType` / `NormalizeRuntimeType`。
* **向後相容**：舊值 `"go"` → `go_api`；`"node"`/`"bun"` → `vite`（以 `Normalize*` 為準）。
* **禁止**直接改 `presets` map；透過查詢函式。不要假設 Runtime 已解析（`auto` 需呼叫端處理）。不要跳過 `Normalize*()`。

### 5.3 `internal/detect` — Laravel 輔助

* `DetectLaravel(root)`：目錄存在性加權計分（artisan、bootstrap/app.php 等）+ `composer.json` 的 laravel/framework / laravel/laravel；`Confidence >= 50` 視為 Laravel。
* `DetectFramework(root)`：讀 `package.json` dependencies，區分 `nuxt` / `next`（輔助，非主 Preset 表）。
* 勿在此套件「發明」 Preset 型別或 Runtime/Port。

---

## 6. 注意事項與禁止行為

* **嚴禁**提交編譯產物與執行時垃圾：`*.exe`、`frontend/dist/`、可忽略的 `*.log` 等（確認 `.gitignore`）。
* **腳本規範**：盡量禁止使用或新增 `.ps1` / `.bat` 腳本，日常工具與維護腳本一律以 Node.js 實作。
* **禁止**在非必要時順手修改其他無關程式碼的註解。
* **避免過度工程**：Minimal Diff，不提前建置過度複雜的抽象層。
* **代碼隔離**：`legacy_fyne/` 僅供參考，新 GUI 功能一律在 `frontend/` 以 React 實作。
* **前端事件**：只用 `EventsOn` 的 unsubscribe，不用 `EventsOff`。
* **前端彈窗**：只用 `customAlert` / `customConfirm`，不用原生 alert/confirm。
* **UI 文案**：走 i18n；**新增**文案與註解不使用 Emoji。
* **錯誤**：Go 端用 `%w` 包裝；鎖內不做 I/O。

---

> 本檔為專案唯一 AGENTS 指令；子目錄不再另放 AGENTS.md，避免規範分叉與過時。
