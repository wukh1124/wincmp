# Changelog

## [2.0.1] 2026-06-08

### Added
- **自動化 SSL CA 憑證配置**：新增 PHP 在本地環境缺少憑證時，自動下載並配置 `cacert.pem` 的功能，解決 PHP 本地 SSL 請求失敗的問題。
- **自動更新與版本檢查**：新增背景每 6 小時定時檢查 GitHub 最新發布版本、左側 Sidebar 新增獨立「版本更新」分頁與新版本紅點提醒 (Badge)，並實作 Windows 系統下無數位簽章 `.exe` 的一鍵熱替換與重啟清理功能。
- **發布腳本優化**：在 `release.ps1` 腳本中，新增輸出單個獨立的 `WinCMP_v*.exe`，方便一鍵更新時能直接下載，免除解壓 zip 的步驟。

### Changed
- **依賴優化**：徹底移除已廢棄的 `fyne.io/fyne/v2` 主框架依賴與舊版 Fyne GUI 資源監控死代碼，進一步精簡專案體積與打包速度。
- **Wails 建置修復**：重新將 Wails 建置模版檔（如 `icon.ico` 與 `manifest` 檔案）納入 Git 追蹤，修正因 `.gitignore` 設定不當導致 `wails build` 時找不到 Windows 圖示的打包錯誤。

## [2.0.0] 2026-06-07

### Added
- **全面重構 GUI 核心**：遷移至 **Wails v2** + **React 18** + **TypeScript**，大幅提升效能並降低資源佔用。
- **深色專業級設計**：採用現代化 Dark Professional 介面，使用 Tailwind CSS 與 Zustand 狀態管理。
- **全新功能面板**：包括高效能即時日誌終端、基於 TanStack Table 的專案與資料庫瀏覽器。

### Removed
- **移除舊版 GUI**：完全移除舊有的 Go Fyne 程式碼（已將舊程式碼歸檔至 `legacy_fyne/` 目錄中）。

## [1.2.6] 2026-06-03

### Added
- 在「Settings」中新增「Display Language」設定，支援語系切換 繁體中文 (`zh-TW`) 與英文 (`en-US`)。
- 在修改語言的提示對話框中，新增「自動重啟 WinCMP」功能按鈕。

### Changed
- 調整 `conf/dependencies.json` 中的依賴配置，改為只允許最新的 Caddy 和 MariaDB 版本啟動

## [1.2.5] 2026-06-02

### Added
- 新增核心依賴自動下載與解壓縮功能（支援 Caddy, MariaDB, PHP 7.3/8.2/8.3, Composer, HeidiSQL, Node.js 等）並提供進度 UI
- 新增啟動時核心依賴完整性檢測與警告對話框
- 新增 `conf/dependencies.json` 設定檔，將依賴版本與下載網址移出代碼統一管理
- 新增「取得最新建議版本 (Fetch)」功能，支援從遠端 GitHub 動態更新依賴配置
- 新增系統 Hosts 更新失敗時的引導對話框，提供一鍵複製 Hosts 規則與管理員權限（UAC）啟動記事本編輯之功能

### Changed
- 優化依賴管理器 UI 佈局，區分「下載」與「重新安裝」按鈕顏色並調整垂直間距

### Fixed
- 修正自動下載 MariaDB 與 Node.js 後的目錄命名格式，並自動生成 `composer.bat`



## [1.2.4] - 2026-04-20

### Fixed
- 修正無法打開 Edit Project 下 Open Project Directory 和 Settings 下的 hosts 的問題

## [1.2.3] - 2026-04-16

### Fixed
- 修復帶底線 Domains 導致 Caddy 設定檔 fallback 到錯誤網域的問題（Caddyfile 現在直接使用用戶輸入的網域，不做安全過濾 fallback）
- 增強 Hosts 更新失敗時的錯誤訊息，明確列出含非法字元的網域，通知用戶需手動新增至 hosts

## [1.2.2] - 2026-04-16

### Fixed
- 修復 Windows Hosts 檔案寫入問題
- 修復 Terminal Logs 分頁索引錯誤（Mailpit/PHP/Runtime Tab 索引映射不正確）
- 修復應用程式啟動時，Terminal Logs 自動跳到 Runtime 分頁的問題（新增初始化鎖定機制）
- 修復 Runtime Log 有新內容時，自動切換分頁無效的問題（需條件滿足才觸發切換）
- 修復分頁切換後，Log 內容未自動滾動到最新的問題（將滾動移至分頁切換後執行）

### Changed
- Terminal Logs 分頁自動滾動優化：分頁切換時在目標分頁執行滾動到底部

## [1.2.1] - 2026-04-16

### Added
- 新增 Mailpit 郵件測試服務整合 (Dashboard 新增 Mailpit 服務啟停與設定對話框)
- Terminal Logs 新增 Mailpit 分頁
- Runtime 支援系統 PATH 回退 (當 `bin/` 中沒有對應執行檔時，自動偵測系統 PATH 中的 Node.js/Bun)

### Changed
- Go 版本升級至 1.26.2
- Terminal Logs 分頁重新排序為 System / Caddy / MariaDB / Mailpit / PHP / Runtime

### Fixed
- 修復 Entry 元件阻擋上層 VScroll 滾輪事件的問題
- 修復專案名稱包含特殊字元時檔名異常 (特殊字元自動替換為連字號)
- 修復 Caddy Timberjack 在 Windows 上停止後殘留的過期日誌未清理
- 修復非 Custom Runtime 未清除啟動指令與 MariaDB 狀態標籤殘留
- 修復 UseWinCMPBin=false 時 Windows 反斜線路徑被誤判為 Shell 注入字元

### Dependencies
- Mailpit 1.29.6

## [1.2.0] - 2026-04-13

### Added
- 新增 Runtime 開發環境運行 (從只支援 Node.js 加上 Bun, Python, Go, Custom)
- Runtime 分頁新增專案 Log 篩選按鈕，可快速切換到對應專案的 Terminal Logs
- Domain 欄位新增一鍵複製連結按鈕
- 專案類型自動偵測 (Preset 系統)，支援 Laravel, Next.js, Nuxt, Astro, Vite, Python (Django/FastAPI/Flask), PocketBase, Go API 等
- 舊版 Node.js 專案自動遷移至新 Runtime 架構
- System Tray 系統匣新增懸停文字

### Changed
- Node.js 改為 Runtime, Node.js Port 改為 Runtime Port, Node.js Projects 改為 Projects Runtime
- Node Version 改為 Runtime, 同時選項改為 Auto, Node.js, Bun, Python, Go Air, Go Run, Custom
- 改用 RSS (WorkingSetSize) 顯示 WinCMP 佔用的 RAM
- 底部資訊欄的 Monitor 區域懸停顯示 Tooltip 的 Stack Total 和服務明細資訊
- MariaDB 設定可使用外部 MariaDB/MySQL, 自訂路徑和端口
- Runtime 啟動模式改為 Background / Terminal 雙模式選擇
- Terminal Logs 日誌限制最佳化 (500 行或 200KB 字符)
- 頁面切換和連續點擊 Tab 效能優化 (防抖機制)

### Fixed
- 修復頁面卡頓和效能缺陷 (Projects 減少 OS Stat 調用, DB Explorer 和 Node.js 異步載入)
- 修復 Settings 的 MaxLogRetention 能自動刪除過期的記錄檔
- 修復 Terminal Logs 暗色模式下日誌文字對比度不足

### Security
- 檢視報告詳見 `doc/audit_report_v1.2.0.md`

### Dependencies
- Bun 1.3.11

---

## [1.1.3] - 2026-04-09

### Fixed
- System Tray 系統匣新增懸停文字

---

## [1.1.2] - 2026-04-02

### Changed
- 改用 RSS (WorkingSetSize) 顯示 WinCMP 佔用的 RAM, 反映程式實際佔用的總物理記憶體 (和 Windows Task Manager 顯示仍有差異)
- 底部資訊欄的 Monitor 區域懸停顯示自訂 Tooltip 的 Stack Total 和 服務明細資訊 (如 Caddy, MariaDB, PHP-CGI, Node.js)

---

## [1.1.1] - 2026-03-30

### Added
- 底部資訊欄加上 Monitor, 顯示 WinCMP 佔用的 CPU 和 RAM

### Changed
- 新增 MariaDB 設定, 可使用外部 MariaDB/MySQL, 自訂路徑和端口

### Fixed
- 新增 Terminal Logs 日誌限制 (500 行或 200KB 字符)
- 修復頁面卡頓和效能缺陷 (Projects 減少OS Stat調用和使用預計算函式, DB Explorer和Node.js異步載入, 移除非必要延遲, 快速連續點擊 Tab 會被忽略, 必須等當前 Tab 載入完成)
- 修復 Settings 的 MaxLogRetention 能自動刪除過期的 `error-*.log`, `node-*.log`, `wincmp-*.log` 記錄檔

---

## [1.1.0] - 2026-03-26

### Added
- Node.js 項目支持啟動/反向代理
- Terminal Logs 新增打開log檔按鈕

### Changed
- 啟動 Caddy 時對 PHP 版本的提示
- 對 Laravel 項目 PHP 版本判斷
- 對 Node 項目判斷
- MariaDB 初始化提示
- wincmp.json 設定名 auto_start 改為 restore_last_state
- Terminal Logs 暗色模式下日誌文字改用亮灰色

### Dependencies
- Composer 1.10.10 / 2.9.3
- Node 24.14.1

---

## [1.0.0] - 2026-03-23

### Added
- **WinCMP** 可攜式 Windows 開發面板核心框架
- Caddy 伺服器一鍵啟停與熱重載支援
- MariaDB 資料庫管理介面（連線測試、備份）
- PHP 多版本負載平衡（7.3/8.2/8.3）
- 專案快速建立與環境隔離工具

### Dependencies
- Caddy 2.11.1
- Heidisql 12.16
- MariaDB 11.4.10
- PHP 7.3.33 / 8.2.30 / 8.3.28