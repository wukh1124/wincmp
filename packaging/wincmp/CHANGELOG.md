# Changelog

## [1.1.1] - 2026-03-27

### Changed
- 新增 MariaDB 設定, 可使用外部 MariaDB/MySQL, 自訂路徑和端口

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