# Changelog

## [1.2.0] - 2026-04-10

### Added
- 新增Runtime開發環境運行 (從只支援Node.js加上更多)

### Changed
- Node.js 改為 Runtime, Node.js Port 改為 Runtime Port, Node.js Projects 改為 Projects Runtime
- Node Version 改為 Runtime, 同時選項改為 Node.js, Bun, Python, Go+Air, Custom

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