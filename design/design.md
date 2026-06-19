# WinCMP v3 UI/UX 介面重建與設計規範 Specification

## 1. 專案願景與核心目標

本規範旨在將原有的 Go Fyne 桌面應用程式，重構為基於現代化 Web 技術棧的精美、高密度、極速反應的開發者桌面工具（媲美 Docker Desktop、TablePlus、Warp 及 Vercel Dashboard）。為了保證極佳的渲染流暢度與自定義外觀，WinCMP 棄用臃腫的第三方組件庫，完全採用原生 React 組件與高效的 HSL/CSS 變數主題系統。

### 技術棧 (Tech Stack)
* **核心框架**：[Wails v2](https://wails.io/) (Go + Web-frontend binding)
* **前端框架**：React 18 + TypeScript + [Vite](https://vite.dev/)
* **樣式系統**：Tailwind CSS v4 (配合 CSS 變數多主題切換)
* **狀態管理**：Wails Binding IPC + React State / Context + 全局 `CustomEvent` 事件通訊機制 (僅日誌系統採用自定義輕量級 `logStore`)
* **虛擬終端**：@xterm/xterm + @xterm/addon-fit (用於即時高性能日誌輸出)

---

## 2. 佈局架構與 UI 視覺草圖 (App Layout)

應用程式採用主側邊欄、頂部導航、主內容區以及可折疊/拉伸的底部終端日誌面板的經典三欄式佈局，並具備完美的響應式和高密度資訊展示。

```text
┌────────────────────────────────────────────────────────────────────────┐
│  WinCMP Local Dev Panel v2.0.4                      ● Go 核心已連線   │
├──────────────┬─────────────────────────────────────────────────────────┤
│ ❖ WinCMP     │  🔍 搜尋專案、網域或路徑... (Ctrl+K)                      │
│   Local Dev  ├─────────────────────────────────────────────────────────┤
│              │  [ Dashboard / 儀表板 ]                                 │
│ 🔘 儀表板    │  ┌──────────────────┐ ┌──────────────────┐             │
│ 📂 專案管理  │  │ 🖥️ CPU 佔用      │ │ 💾 RAM 記憶體    │             │
│ 🗄️ 資料庫瀏覽│  │ [████░░░░░░] 35% │ │ [██████░░░░] 60% │             │
│ ⚙️ 系統設定  │  └──────────────────┘ └──────────────────┘             │
│              │  ┌────────────────────────────────────────────────────┐ │
│              │  │ 服務健康狀態                                        │ │
│              │  │ 🟢 Caddy v2.7    Port: 80, 443    [ 重啟 ] [ 停止 ] │ │
│              │  │ 🟢 MariaDB 11.4  Port: 3306       [ 重啟 ] [ 停止 ] │ │
│              │  │ 🔴 PHP-CGI 8.2   Port: 38200      [ 啟動 ] [ 設定 ] │ │
│              │  └────────────────────────────────────────────────────┘ │
├──────────────┼─────────────────────────────────────────────────────────┤
│ 🌐 繁 🔘 CAR │  ▲ [收起 Logs 控制台]                                   │
│ CPU: 12.5%   ├─────────────────────────────────────────────────────────┤
│ [██░░░░░░]  │  $ [14:32:01] [INFO] Caddy process started on port 80   │
│ RAM: 512 MB  │  $ [14:32:02] [INFO] MariaDB database system is ready   │
│ [█████░░░]  │  $ [14:32:05] [ERROR] PHP-CGI execution failed (port)   │
└──────────────┴─────────────────────────────────────────────────────────┘
```

---

## 3. 設計語言與 Themes 規範 (Themes & Design Tokens)

為了提供別具一格的個性化開發體驗，WinCMP 支援三大完全獨立、別具特色的主題視覺系統，透過 HTML 屬性 `data-theme` 進行全域變數切換：

### 3.1 三大主題定義

1. **Carbon (單色極簡粗獷主義，默認)**
   * **特色**：深色背景搭配高對比單色文字，零陰影，純粹的代碼感。
   * **顏色**：主底色為 `#1f2228`，側邊欄為 `#181a1f`，強調色與焦點色均為純白色（`#ffffff`）。
   * **字型**：主要展示與等寬字型均使用 `GeistMono`。

2. **Cream (溫暖奶油色，淺色)**
   * **特色**：以 Anthropics 風格的暖色調奶油色為基底，乾淨、 approachable 且極富親和力。
   * **顏色**：底色為 `#faf9f7`，側邊欄為 `#f0ede8`，強調色為陶土紅 / 磚紅（`#c96442`），文字為暖炭黑色（`#1a1916`）。
   * **字型**：展示字型使用 `Söhne` 或 `Avenir Next`，等寬字型使用 `JetBrains Mono`。

3. **Sketch (手繪線框圖)**
   * **特色**：以手繪網格設計圖紙為背景，粗體 marker 標題字型，鉛筆描邊，並以黏貼式便簽紙（Sticky Notes）樣式呈現引導氣泡與警告卡片。
   * **顏色**：底色為網格紙底色 `#f0ebe0`，強調色為手繪藍色馬克筆色（`#2b6cb0`）。
   * **字型**：展示字型使用 `Comic Neue` 或 `Caveat`，等寬字型使用 `JetBrains Mono`。

### 3.2 字型與排版系統 (Typography)

* **主要字型 (Sans-serif)**：`-apple-system`, `BlinkMacSystemFont`, `"Segoe UI"`, `Roboto`, `system-ui`
* **等寬字型 (Monospace - 用於終端與日誌)**：`JetBrains Mono`, `GeistMono`, `Fira Code`, `SF Mono`, `Consolas`
* **字型大小切換 (Font Size Settings)**：
  支援 Small (16px)、Medium (18px) 與 Large (20px) 三檔切換，可在側邊欄底部快速變更並防抖儲存至後端設定檔中。

---

## 4. 導航與快速設定系統 (Navigation System)

### 4.1 左側導航欄 (Sidebar)
* **尺寸規範**：展開寬度 `256px`，折疊寬度 `64px`。
* **主要選單項目**：
  1. **儀表板 (Dashboard)**：全域服務管理與核心狀態。
  2. **專案管理 (Projects)**：本機開發網站目錄、Caddy 虛擬主機與 Hosts 同步。
  3. **資料庫瀏覽 (Database)**：HeidiSQL 工具快速呼叫與內建輕量 SQLite/MariaDB 數據網格檢視。
  4. **資源監控 (Resources)**：系統 CPU 與記憶體進程負載監測。
  5. **系統設定 (Settings)**：Hosts 編輯器與服務組態管理。
  6. **版本更新 (Update)**：軟體版本自動/手動更新檢測。
  7. **終端日誌 (Logs)**：全螢幕日誌控制台。
* **快速設定按鈕組 (Quick Settings Panel)**：
  側邊欄底部提供三個圓角圖示按鈕，分別對應「語系切換 (zh-TW / en-US)」、「主題切換 (Carbon / Cream / Sketch)」與「字型大小調整」。當首次使用時，上方會浮現「快速設定指南氣泡」進行引導。

### 4.2 頂部工具欄 (Topbar)
* **全域搜尋 (Ctrl + K)**：
  支援全域快速模糊搜尋，可直接過濾專案名稱、本機路徑或域名，點擊搜尋結果會跳轉至專案頁並高亮標識對應專案。
* **安全指示器 (Admin Badge)**：
  顯示「管理員模式」或「限制模式」。當檢測到非 UAC 管理員執行時給予警告提示，防止 Hosts 檔案寫入失敗。

---

## 5. 核心功能頁面規格 (Core Pages)

### 5.1 儀表板 (Dashboard)
* **系統概要 (Overview Cards)**：
  展示 CPU 與 RAM 的即時環狀進度條，以及當前執行中的服務數量。
* **服務管理卡片 (Service Cards)**：
  為 Caddy、MariaDB、PHP FastCGI 等提供卡片式控制台。支援一鍵 [啟動 / 停止 / 重啟]，動態展示連線 Port 與運行時間。

### 5.2 專案管理 (Projects)
* **Monorepo 與專案新增**：
  支援 Monorepo 專案選項，勾選後會自動聯動調整專案名稱與網域別名，並寫入設定檔中。
* **進階 Custom Command 運作環境**：
  支援每個專案的自訂執行指令 (Custom Command) 與對應的運行時環境 (Runtime) 設定，便於啟動前端 Vite、Node.js 服務或獨立的後端守護進程。
* **快速操作**：
  提供一鍵 [開啟本機資料夾]、[編輯專案組態]、[複製域名] 與 [在瀏覽器中開啟] 的功能連結。

### 5.3 依賴庫管理器 (Dependency Manager)
* **彈出式 Dialog 設計**：
  提供本機開發環境依賴的一鍵下載與解壓配置。
* **管理項目**：
  * 核心執行環境：Caddy Web 伺服器、MariaDB 資料庫。
  * PHP 執行環境：PHP 7.4 / 8.0 / 8.1 / 8.2 / 8.3 / 8.4 NTS 等多版本。
  * 開發輔助工具：Composer 包管理器、Node.js 運行環境、Mailpit 郵件測試伺服器、HeidiSQL 資料庫工具。
* **下載管道**：
  整合 Go 後端的 Downloader Pipeline，即時反饋「準備中 / 下載中 (顯示 MB/s 與進度條) / 解壓中 / 安裝成功 / 失敗重試」等狀態。

### 5.4 系統設定 (Settings)
* **分類模組**：
  * 一般設定：開機啟動、最小化托盤、WWW 根目錄配置。
  * Hosts 檔案編輯器：支援語法高亮的純文字 Hosts 編輯器，具備「未儲存防禦機制」，防止使用者切換頁面時遺失修改。
  * 服務組態：配置各版本 PHP 的啟動連接埠與 FastCGI 進程數量。

---

## 6. 即時日誌終端面板 (Terminal Logs Console)

採用折疊拉伸式設計，位於應用程式主界面底部，整合 xterm.js 實現流暢的高性能 stdout/stderr 渲染。

* **分頁過濾**：可切換檢視 Caddy、MariaDB、Mailpit、各 PHP 版本及各專案的獨立日誌串流。
* **特色功能**：
  * 支援日誌層級色彩著色（Cyan/Amber/Red/Purple）。
  * 提供「暫停日誌流」、「清除緩衝區」、「自動捲動開關」以及「匯出為 .log 檔案」的操作控制列。

---

## 7. 核心開發與通訊規範 (Implementation Rules)

為了維持程式碼庫的健壯性，所有開發人員與 Agent 必須嚴格遵守以下前後端通訊規範：

### 7.1 i18n 多國語言規範
* **核心原則**：前後端統一採用 **繁體中文 (zh-TW)** 作為翻譯字典的 Key，未命中翻譯時預設直接顯示 Key。
* **後端 Go 語法**：使用 `i18n.T("中文 Key")` 或是 `i18n.Tfmt("日誌: %s", value)`。
* **前端 React 語法**：使用 `useLanguage` hook 取出 `t` 函數，例如 `{t("中文 Key")}`。

### 7.2 禁用全域 `EventsOff`
* 在 Wails v2 中，`EventsOff("事件名")` 會註銷該事件下的**所有**全域監聽器，導致其他組件的監聽失效。
* **安全規範**：在組件解除掛載 (Unmount) 時，**必須**使用 `EventsOn` 呼叫所回傳的 `unsubscribe` 註銷函數，嚴禁直接使用 `EventsOff`。
```typescript
useEffect(() => {
  const unsubscribe = EventsOn('resource_usage', handleData);
  return () => {
    unsubscribe(); // 僅註銷此實例
  };
}, []);
```

### 7.3 全域自訂彈窗規範
* 為避免在 Windows 上出現帶有原生網頁標題的醜陋對話框，WinCMP **全面禁用** `window.alert()` 和 `window.confirm()`。
* **替代方法**：使用全域掛載的非同步自訂 React 彈窗：
  * Alert：`await (window as any).customAlert("提示訊息")`
  * Confirm：`const ok = await (window as any).customConfirm("確認執行嗎？")`

### 7.4 Windows 環境變數與路徑隔離
* **正斜線原則**：傳遞給 Caddyfile 或本機配置文件時，統一將路徑中的反斜線 `\` 替換為正斜線 `/`。
* **PATH 環境隔離**：啟動專案的 Custom Command 或 PHP-CGI 時，嚴禁修改系統全域的 `PATH`，必須動態 append 到進程的 `exec.Cmd.Env` 中。
