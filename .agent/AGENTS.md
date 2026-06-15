# WinCMP - Agent 開發指南 (Wails v2.0.0+)

本檔案提供 Agent 程式碼助手在此程式碼庫中進行開發與維護時所需的一切架構資訊與規範。

---

## 1. 開發與建置環境 (Wails v2)

### 前置需求
* **Go**：Go 1.26.2+
* **Wails CLI**：請確認系統已安裝 Wails v2。若未安裝，可使用 `go install github.com/wailsapp/wails/v2/cmd/wails@latest` 安裝。
* **C 編譯器**：MinGW-w64 (WinLibs) ── 用於 Wails 內部/底層對 Windows API 依賴的編譯，請確保 `gcc -v` 可正常執行。
* **Node.js**：Node.js 18+ (用於前端開發與打包)。

### 開發熱重載指令 (Hot Reload)
```cmd
# 啟動 Wails 開發模式 (Go 後端與 React 前端同步熱重載)
wails dev
```

### 建置編譯指令
```cmd
# 後端/前端依賴整理
go mod tidy
cd frontend && npm install && cd ..

# 開發/偵錯建置 (含除錯控制台與調試工具)
wails build -debug

# 正式發布編譯 (無視窗主控台，編譯後產出 wincmp.exe)
wails build -clean

# 壓縮並移除 symbols 的正式發布 (適用於體積優化)
wails build -clean -ldflags "-s -w"
```

### 測試指令
* 執行 Go 後端單元測試：`go test ./...`
* 執行指定後端套件測試：`go test ./internal/config`

---

## 2. 專案架構與目錄佈局

```text
wincmp/
├── main.go               # 應用程式進入點：初始化與啟動 Wails
├── app.go                # Wails 生命週期管理器 (startup, shutdown) 及核心日誌/監控推送
├── bridge.go             # Wails 與 Go 端的主 Binding API (前後端 RPC 接口)
├── downloader_bridge.go  # Wails 下載管理器 Binding 接口
├── conf/                 # 系統配置目錄 (Caddyfile, wincmp.json, my.ini 等)
├── bin/                  # 二進位套件目錄 (Caddy, MariaDB, PHP-CGI, Mailpit, Node 等)
├── internal/             # Go 後端核心套件 (不包含 GUI 邏輯)
│   ├── config/           # wincmp.json 設定檔讀寫
│   ├── scanner/          # bin/ 目錄掃描器與 Port 計算
│   ├── process/          # 子進程生命週期與 Caddy/MariaDB/PHP/Mailpit 啟停管理器
│   ├── hosts/            # 系統 Hosts 檔案管理 (同步與 UAC 備份)
│   ├── i18n/             # 本地化多國語言字典
│   └── resource/         # 硬體 CPU & RAM 監控
├── frontend/             # 前端 React + TSX 專案
│   ├── src/
│   │   ├── wailsjs/      # Wails 自動產生的 Go 函式綁定與 JS 映射檔 (勿手動修改)
│   │   ├── components/   # 前端 React 元件 (Dashboard, Projects, DBExplorer 等)
│   │   └── stores/       # Zustand 狀態管理器
│   └── tailwind.config.js# Tailwind 樣式變數設定
└── legacy_fyne/          # 已歸檔的舊 Go Fyne 程式碼 (僅供功能移植參考，勿在此修改)
```

---

## 3. 前後端通訊與開發規範

### 3.1 Go 綁定方法 (Go to JS binding)
* 凡是在 `bridge.go` 或 `app.go` 中，屬於 `*App` 結構體的 **PascalCase（首字母大寫）** 匯出方法，Wails 都會在編譯時自動產生前端的 JS 呼叫 SDK。
* 前端 React 調用範例：
  ```tsx
  import { GetConfig, StartCaddy } from '../wailsjs/go/main/App';

  // 非同步調用
  const config = await GetConfig();
  ```

### 3.2 背景事件推送 (Events Mechanism)
* **單向推送**：Go 端使用 `runtime.EventsEmit` 向前端推送事件，例如資源佔用監控與終端即時日誌。
* **Go 端推送範例**：
  ```go
  // 推送即時資源佔用
  runtime.EventsEmit(a.ctx, "resource_usage", map[string]interface{}{
      "cpu":    cpuPercent,
      "memory": memoryMB,
  })
  ```
* **前端 React 接收範例**：
  ```tsx
  import { EventsOn } from '../wailsjs/runtime/runtime';

  useEffect(() => {
    // EventsOn 會回傳一個用於註銷該次監聽的函數 (unsubscribe)
    const unsubscribe = EventsOn("resource_usage", (data) => {
      console.log("CPU:", data.cpu, "RAM:", data.memory);
    });
    return () => {
      unsubscribe(); // 僅註銷此實例，絕不影響全域其他同名監聽器
    };
  }, []);
  ```
* **⚠️ 致命 Bug 防範：全面禁用全域 `EventsOff(eventName)`**
  * 在 Wails v2 中，直接調用 `EventsOff("事件名")` 會**註銷該事件名下登記的所有全域監聽器**。
  * 若在 React 組件卸載（Unmount）或切換語言重載組件時使用 `EventsOff`，會把其他組件中（如全域 `logStore`）的同名事件監聽徹底註銷，造成日誌與監控功能永久失效。
  * **規範**：前端組件清理時，**必須**使用 `EventsOn` 呼叫所回傳的 `unsubscribe` 註銷函數，嚴禁直接使用 `EventsOff`。

### 3.3 程式碼風格與命名慣例
* **後端 Go**：
  * 遵循 Go 標準規範（Go fmt 格式化）。
  * 錯誤包裝：使用 `fmt.Errorf("錯誤描述: %w", err)`。
  * 資源解鎖：使用 `sync.Mutex` 且務必搭配 `defer mu.Unlock()`。
* **前端 TypeScript/React**：
  * 變數與函式使用 `camelCase`，元件與介面使用 `PascalCase`。
  * 精確定義 Prop 與 State 的 TypeScript 類型。
  * 使用 Zustand 管理全域狀態，避免 React Prop drilling。

---

## 4. 核心業務邏輯與注意事項

### 4.1 多國語言 (i18n) 本地化
* **核心規範**：前後端統一採用 **繁體中文 (zh-TW)** 作為翻譯字典的 Key，未命中翻譯時預設直接顯示 Key 實現優雅降級。
* **後端 Go 規範**：
  * 所有呈現給使用者的字串、錯誤訊息與日誌，**必須**使用 `i18n.T` 或是 `i18n.Tfmt` 包裹。
  * **Go 端語法**：
    ```go
    i18n.T("釋放預設設定檔失敗")
    i18n.Tfmt("ℹ️ 已自動刪除過期日誌檔: %s", name)
    ```
  * **字典維護**：在程式碼中新增 `i18n.T("中文 Key")` 後，務必於 `internal/i18n/i18n.go` 的 `enTranslations` 字典中補上英文對照。
* **前端 React 規範**：
  * 所有 UI 上的可見文字、Placeholder 提示字串與全域彈窗訊息，**必須**使用 `useLanguage` hook 取出 `t` 函數進行包裹。
  * **React 端語法**：
    ```tsx
    import { useLanguage } from '../i18n';
    
    const { t } = useLanguage();
    return (
      <button title={t("刪除專案")}>
        {t("快速新增首個專案")}
      </button>
    );
    ```
  * **字典維護**：在前端程式碼中新增 `t("中文 Key")` 後，務必於 `frontend/src/i18n.ts` 的 `enTranslations` 字典中補上英文對照。

### 4.2 Windows 路徑與環境變數隔離
* **正斜線統一**：傳遞給 Caddyfile 的本機路徑，請使用 `strings.ReplaceAll(path, "\\", "/")`，避免 Caddy 讀取反斜線時發生轉義錯誤。
* **動態 Env 注入**：啟動 PHP-CGI 或 Node 專案時，**絕對不要**修改系統全域的 `PATH`。必須將對應的二進位 bin 目錄動態 append 注入到啟動進程的 `exec.Cmd.Env` 中。

### 4.3 系統服務端口規範
* **PHP 連接埠**：由 `calcPHPPortBase()` 計算（預設公式：`3` + `主版本` + `次版本` + `00` 起始，例如 PHP 8.2 為 `38200` 開始的連續端口），每版本預設啟動 3 個 `php-cgi` 進程。
* **網域 Hosts 同步**：本機自訂網域同步寫入 `C:\Windows\System32\drivers\etc\hosts` 前，必須檢測是否有新增，並在寫入前進行 Hosts 檔案備份。

### 4.4 全域自訂彈出視窗規範 (Alert/Confirm)
* **禁止原生 WebView 視窗**：為避免在 Windows 上出現帶有 `wails.localhost 說` 標題的原生對話框影響美觀，本專案**全面禁止**直接使用瀏覽器原生的 `window.alert()`、`window.confirm()` 以及 `confirm()`。
* **自訂方法呼叫**：
  * **Alert 提示**：必須使用全域掛載的 `(window as any).customAlert("提示訊息")`，這會以 Promise 非同步彈出自訂的 React 彈出視窗。
  * **Confirm 確認**：必須使用 `await (window as any).customConfirm("確認訊息")` 進行非同步等待，並依據回傳的 `boolean` (確定為 `true`，取消為 `false`) 執行後續邏輯。

### 4.5 官網圖標與 Lucide 踩坑限制
* **品牌圖標缺失問題**：在靜態官網 `website/index.html` 中引入 Lucide CDN（例如 `unpkg.com/lucide`）時，由於 Lucide 官方在 v1.0.0 起移成了所有品牌類型的圖標（如 `github`、`slack`、`facebook` 等），導致使用 `<i data-lucide="github"></i>` 會造成圖標無法渲染，並在控制台拋出 `icon name was not found` 錯誤。
* **解決方案**：對於所有的品牌類型圖標（特別是 GitHub），**必須使用 Inline SVG** 代替 `<i data-lucide="...">`。其他非品牌圖標（如 `menu`、`settings` 等）可繼續正常使用 Lucide。

---

## 5. 注意事項與禁止行為

* **嚴禁**提交任何編譯後的二進位檔案（如 `*.exe`、`*.log` 或 `frontend/dist/` 目錄，請確保已被納入 `.gitignore`）。
* **禁止**在非必要時順手修改其他無關程式碼的註解。
* **避免過度工程**：追求 Minimal Diff，不提前建置過度複雜的抽象層。
* **注意代碼隔離**：舊的 Fyne 程式碼已封存在 `legacy_fyne/` 目錄中，不要在該目錄中進行任何新功能開發；所有新 GUI 功能應直接在 `frontend/` 中使用 React 與 CSS 實作。

---

> 最後更新：2026-06-07
