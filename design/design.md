# WinCMP v3 UI/UX 介面重建與設計規範 Specification

## 1. 專案願景與核心目標

本規範旨在將原有的 Go Fyne 桌面應用程式，重構為基於現代化 Web 技術棧的精美、高密度、極速反應的開發者桌面工具（媲美 Docker Desktop、TablePlus、Warp 及 Vercel Dashboard）。

### 技術棧 (Tech Stack)
* **核心框架**：[Wails v2](https://wails.io/) (Go + Web-frontend binding)
* **前端框架**：React 18 + TypeScript + [Vite](https://vite.dev/)
* **樣式系統**：Tailwind CSS (自訂 HSL 設計標記)
* **組件庫**：[shadcn/ui](https://ui.shadcn.com/) (Radix UI) + [Lucide Icons](https://lucide.dev/)
* **狀態管理**：[Zustand](https://github.com/pmndrs/zustand)
* **表單與驗證**：React Hook Form + Zod
* **表格與數據**：[TanStack Table (React Table)](https://tanstack.com/table)
* **動效與過渡**：Framer Motion
* **虛擬終端**：xterm.js (用於即時高性能日誌輸出)

---

## 2. 佈局架構與 UI 視覺草圖 (App Layout)

應用程式採用主側邊欄、頂部導航、主內容區以及可折疊/拉伸的底部終端日誌面板的經典三欄式佈局，並具備完美的響應式和高密度資訊展示。

```text
┌────────────────────────────────────────────────────────────────────────┐
│  WinCMP Local Dev Panel v2.0.0                      ● Go 核心已連線   │
├──────────────┬─────────────────────────────────────────────────────────┤
│ ❖ WinCMP     │  🔍 搜尋專案或設定... (Ctrl+K)                          │
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
│ 📊 系統監控  │  ▲ [收起 Logs 控制台]                                   │
│ CPU: 12.5%   ├─────────────────────────────────────────────────────────┤
│ [██░░░░░░]  │  $ [14:32:01] [INFO] Caddy process started on port 80   │
│ RAM: 512 MB  │  $ [14:32:02] [INFO] MariaDB database system is ready   │
│ [█████░░░]  │  $ [14:32:05] [ERROR] PHP-CGI execution failed (port)   │
└──────────────┴─────────────────────────────────────────────────────────┘
```

---

## 3. 設計語言與 Tokens 規範 (Design Tokens)

為營造極具質感且低干擾的沉浸式開發體驗，UI 採用 **Dark Professional (深色專業級)** 主題，主色調搭配精緻微漸層與流暢的動態過渡。

### 3.1 顏色系統 (Color System - HSL mapped)

| Token 名稱 | 預設 HEX 值 | HSL 變數定義 | 應用場景 |
| :--- | :--- | :--- | :--- |
| **Primary** | `#3B82F6` | `hsl(217.2, 91.2%, 59.8%)` | 主品牌色、活動狀態按鈕、高亮線條 |
| **Success** | `#10B981` | `hsl(142.1, 70.6%, 45.3%)` | 服務正常運行 (Running)、狀態燈 |
| **Warning** | `#F59E0B` | `hsl(37.9, 90.2%, 50.2%)` | 服務重啟中、警告訊息 |
| **Danger** | `#EF4444` | `hsl(0, 84.2%, 60.2%)` | 服務停止 (Stopped)、錯誤日誌、危險操作 |
| **Background** | `#09090B` | `hsl(240, 10%, 3.9%)` | 應用程式最底層底色 |
| **Sidebar Bg** | `#0C0C0E` | `hsl(240, 10%, 5.5%)` | 左側導航底色，與主內容區拉開層次 |
| **Card / Surface**| `#18181B` | `hsl(240, 5.9%, 10%)` | 卡片元件、區塊容器、彈出視窗 |
| **Input / Field**| `#27272A` | `hsl(240, 5.9%, 15%)` | 輸入框背景、下拉選單背景 |
| **Border** | `#27272A` | `hsl(240, 5.9%, 15%)` | 分割線、表框、描邊 |
| **Text Primary** | `#F4F4F5` | `hsl(240, 5.9%, 96.1%)` | 主要標題、主文字 |
| **Text Secondary**| `#A1A1AA` | `hsl(240, 5%, 64.9%)` | 次要說明、副標題、屬性標籤 |

### 3.2 字型與排版系統 (Typography)

* **主要字型 (Sans-serif)**：`Inter`, `-apple-system`, `BlinkMacSystemFont`, `"Segoe UI"`, `Roboto`, `system-ui`
* **等寬字型 (Monospace - 用於終端與日誌)**：`JetBrains Mono`, `Fira Code`, `SF Mono`, `Consolas`
* **字級規範**：
  * `12px (Caption)`：次要屬性、狀態標記、表格首標。
  * `14px (Body)`：預設內文字型、按鈕標題。
  * `16px (Body Large)`：重點訊息、輸入框文字、卡片標題。
  * `20px (Section Title)`：區塊小標題、抽屜面板標題。
  * `24px / 28px (Page Title)`：頁面主標題。

### 3.3 動效與微互動 (Micro-interactions)

* **過渡效果 (Transitions)**：
  * 按鈕與選單 Hover：`transition-all duration-150 ease-in-out`
  * 折疊面板/側邊欄收合：`transition-all duration-300 cubic-bezier(0.4, 0, 0.2, 1)`
* **狀態切換動畫 (Animations)**：
  * 運行中指示燈：`animate-ping` 配合擴散陰影特效。
  * 加載中狀態：`animate-spin` 精細旋轉菊花。

---

## 4. 側邊欄與頂部導航 (Navigation System)

### 4.1 左側導航欄 (Sidebar)
* **尺寸規範**：展開寬度 `256px (w-64)`，折疊寬度 `72px (w-18)`。
* **主要選單項目**：
  1. **儀表板 (Dashboard)**：全域服務管理與硬體狀態。
  2. **專案管理 (Projects)**：本機開發網站目錄與 Caddyfile 自動同步。
  3. **資料庫瀏覽 (Database)**：整合 HeidiSQL 或內建 SQLite/MariaDB 輕量瀏覽。
  4. **系統設定 (Settings)**：Hosts、PHP-CGI、SSL 證書等設定。
* **底部組件 (System Resources)**：
  * 整合 CPU 及 RAM 的即時資源佔用進度條 (Progress bar)，直觀展示 WinCMP Core 對系統的負載。
  * 滑鼠懸停時顯示詳細進程數與記憶體堆疊資訊。

### 4.2 頂部工具欄 (Topbar)
* **搜尋輸入框**：
  * 支援全域快速搜尋，整合命令面板 (Command Palette)。
  * 快速鍵 `Ctrl + K` 喚醒全域模糊搜尋，可直接執行服務重啟、切換分頁、新增專案等快捷動作。
* **右側指示器**：
  * **Go 核心連線狀態**：使用綠色呼吸燈代表 Wails 與 Go 後端 IPC 通訊健全。
  * **版本資訊**：顯示當前 WinCMP 的軟體版本 (如 `v2.0.0`)。

---

## 5. 核心功能頁面規格 (Core Pages)

### 5.1 儀表板 (Dashboard)
* **系統概要卡片 (Overview Cards)**：
  * **硬體使用率**：即時 CPU、RAM 使用百分比（透過波形圖或圓形環狀圖呈現）。
  * **服務狀況數**：運行中服務 / 總服務數。
  * **活躍專案數**：已啟用的 Caddy 虛擬主機網站數。
* **服務管理卡片 (Service Cards - 高整合設計)**：
  * 代替原有的簡陋表格，為每個系統級服務（Caddy, MariaDB, Mailpit, PHP FastCGI）建立獨立卡片。
  * **卡片內容**：
    * 狀態徽章 (🟢 運行中 / 🔴 已停止) 與服務名稱、版本。
    * 監聽連接埠 (Ports) 與運行時間 (Uptime)。
    * **操作控制列**：[啟動/停止] (動態切換按鈕，停止時顯示 Danger 紅色，啟動時顯示 Success 綠色)、[重啟]、[檢視日誌]、[組態設定]。

### 5.2 專案管理 (Projects)
* **過濾工具列 (Toolbar)**：
  * 關鍵字搜尋、專案類型過濾 (Laravel / ThinkPHP / 靜態 HTML 等)、PHP 版本過濾。
  * **「掃描 WWW」按鈕**：一鍵掃描配置的網站根目錄並自動新增專案。
  * **「新增專案」按鈕**：打開新增專案抽屜。
* **專案表格 (Data Grid)**：
  * 使用 TanStack Table 實作的無縫數據表格。
  * **顯示欄位**：專案名稱、啟用狀態 (Switch 開關)、框架類型、PHP 版本、綁定域名 (點擊直接在瀏覽器開啟網站)、佔用 Port、更新時間、操作。
  * **快捷操作列**：
    * 📂 開啟本機資料夾。
    * 📝 編輯配置（右側抽屜喚醒）。
    * 🔗 複製域名。
    * 🗑️ 刪除。

### 5.3 專案編輯抽屜 (Project Drawer)
* 放棄傳統的阻擋式 Modal 彈出視窗，全面改用右側滑出式抽屜 (Slide-over Drawer) 以維持視覺的連貫性。
* **分頁配置項**：
  1. **基本資訊**：專案名稱、本機目錄路徑（支援系統資料夾選取器）。
  2. **域名與 SSL**：域名綁定清單（支援一鍵生成自簽署本地 SSL 憑證）。
  3. **運行環境**：選擇要繫結的 PHP 版本（如 PHP 8.2 / 8.3 / 7.4），以及運作模式。
  4. **進階設定**：自訂 Caddyfile 片段、部署/編譯指令。

### 5.4 資料庫瀏覽 (DB Explorer)
* 針對本地資料庫開發的極簡整合，外觀致敬 TablePlus。
* **功能板塊**：
  * **左側導航**：資料庫列表 (Database List) 與資料表樹狀圖 (Table List)。
  * **頂部工具列**：重新整理、匯入 SQL、匯出資料、**「開啟 HeidiSQL」快捷按鈕**（一鍵呼叫外部 HeidiSQL 工具進行高級操作）。
  * **主體數據檢視器**：網格化呈現資料表內容，支持分頁、過濾、排序與列寬自由拖曳。

### 5.5 系統設定 (Settings)
* 採用分類導航與卡片式表單相結合的設定中心。
* **分類模組**：
  * **一般設定**：開機啟動、最小化到系統托盤、WWW 根目錄配置。
  * **Hosts 檔案管理**：內建 Hosts 編輯器，支援語法高亮，並提供 Hosts 檔案同步與系統權限授權指示。
  * **服務組態管理**：
    * PHP FastCGI：配置各個 PHP 版本的啟動端口、進程數量與路徑。
    * Caddy：SSL 憑證路徑設定。
    * MariaDB：連接埠與 Root 密碼設定。
  * **外觀與主題**：主題切換（深色/淺色/系統同步）、強調色自訂（藍色/翠綠/紫羅蘭/活力橘）。

---

## 6. 即時日誌終端面板 (Terminal Logs Console)

底部的即時日誌區是本工具的核心效能與排錯重點，參照 Warp 終端的高性能渲染設計。

* **折疊控制**：可在底部自由拉伸高度，或點擊折疊條一鍵完全收合，保留最大主要工作區。
* **高性能終端**：整合 xterm.js，流暢渲染來自 Go 端發送的系統與服務 stdout/stderr 串流。
* **日誌分頁切換**：
  * [ 系統日誌 ] [ Caddy 日誌 ] [ MariaDB 日誌 ] [ Mailpit 日誌 ] [ 各 PHP 版本日誌 ] [ 專案日誌 ]
* **日誌特色功能**：
  * **日誌層級色彩著色**：
    * `INFO`：青色 (Cyan)
    * `WARN`：琥珀黃 (Amber)
    * `ERROR`：鮮紅色 (Red)
    * `DEBUG`：紫色 (Purple)
  * **全域搜尋與過濾**：在當前終端日誌中模糊搜尋關鍵字。
  * **操作欄**：[ 自動捲動開關 ] [ 暫停日誌流 ] [ 清除日誌 ] [ 匯出當前日誌為 .log ]。

---

## 7. 狀態管理 (Zustand Stores Structure)

採用分離且輕量的 Zustand 狀態庫，以確保各模組獨立更新、不導致非必要的前端重新渲染。

* **`serviceStore`**：管理 Caddy, MariaDB, Mailpit 和 PHP 的運行狀態、連接埠、CPU/記憶體消耗、以及 Start/Stop/Restart 的非同步狀態。
* **`projectStore`**：管理本機專案列表、掃描目錄任務、新增/修改/刪除專案的 API 綁定。
* **`databaseStore`**：管理本地資料庫、資料表結構快取、HeidiSQL 的呼叫狀態。
* **`settingsStore`**：管理一般設定、Hosts 檔案同步狀態、強調色與語系設定（i18n 多國語言狀態）。
* **`terminalStore`**：管理日誌面板的緩衝區數據、當前焦點分頁、過濾條件與捲動條鎖定。

---

## 8. 專案目錄結構規範 (Folder Structure)

前端 React 代碼將遵循清晰的 Feature-Based 架構，便於日後維護與擴展。

```text
frontend/
├── src/
│   ├── assets/             # 靜態資源 (圖示、標誌、全局圖片)
│   ├── wailsjs/            # Wails 自動產生的 Go 綁定代碼 (由 Wails CLI 生成)
│   ├── app/                # 全局應用配置 (主題提供者、全域 css)
│   ├── components/         # 跨頁面共享的通用 UI 組件 (Button, Input, Table, Drawer)
│   ├── layouts/            # 頁面佈局組件 (AppShell, Sidebar, Topbar)
│   ├── pages/              # 分頁主要元件
│   │   ├── Dashboard/      # 儀表板頁面與相關 ServiceCard 子元件
│   │   ├── Projects/       # 專案列表、工具列與 ProjectDrawer
│   │   ├── DBExplorer/     # 資料庫瀏覽器與數據網格
│   │   └── Settings/       # 設定中心與 Hosts 編輯器
│   ├── stores/             # Zustand Stores 定義 (service, project, etc.)
│   ├── hooks/              # 全局自訂 React Hooks
│   ├── types/              # TypeScript 類型定義
│   └── utils/              # 通用輔助工具函式
├── tailwind.config.js      # Tailwind CSS 變數與 HSL 設計標記設定
└── package.json            # 前端依賴定義
```

---

## 9. 軟體驗收標準 (Acceptance Criteria)

1. **功能完整度**：重構後必須完全繼承原 Go Fyne 版的所有本機服務控制與專案掃描機制，功能絕不遺漏。
2. **極致視覺美感**：界面外觀、漸層、陰影、字型需具備高品質現代開發者工具標準，無低質感網頁感。
3. **極低延遲反應**：前頁切換、抽屜拉出、對話框彈出等動畫需保持 60fps 順暢度，日誌控制台在大量 log 輸出時不得阻礙 UI 主執行緒。
4. **外觀一致性**：深色模式為第一優先（Dark Mode First），搭配自訂色彩主題在不同顯示器下保持高度清晰。
5. **健壯的異常處理**：在遇到系統權限不足（例如 hosts 寫入失敗、Port 被佔用、PHP 未能正常啟動）時，需在 UI 頂部給予優雅的 Toast 或 Dialog 錯誤提示，而非默默失敗。
