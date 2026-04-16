# Changelog

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