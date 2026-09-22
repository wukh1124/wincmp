# WinCMP v2.1.6

發布日期：2026-09-23

此版本為 WinCMP 帶來了完整的開源合規體系、第三方授權與商標免責聲明，並強化了依賴庫管理的透明度與穩定性。

## What's Changed

### Added
- **開源合規與第三方授權聲明**：專案根目錄與發布範本新增 `THIRD-PARTY-NOTICES.md`，詳列受控服務（Caddy、MariaDB、Redis、PHP、Node 等）、Go 後端模組與前端 NPM 套件之授權條款 (SPDX) 及官方原始碼獲取途徑；雙語 README 增設商標免責與非背書聲明。
- **依賴庫管理卡片授權徽章與操作選單**：各依賴服務名稱旁標註極簡 License Badge（例如 `GPL-2.0`、`Apache-2.0`、`BSD-3-Clause`），下拉選單整合「授權協議」標記與「官方原始碼 ↗」外部連結，點擊即以預設瀏覽器開啟官方倉庫。
- **依賴庫彈窗合規聲明對話框**：彈窗底部新增「開源授權與商標聲明」連結，點擊開啟自適應主題之聲明對話框，展示 WinCMP 核心架構說明、商標免責宣告與受控服務總表，並全面支援前後端繁中與英文雙語切換。
- **獨立執行檔 (exe) 啟動自解壓合規保證**：透過 Go 內嵌資源，當使用者僅下載單獨 exe 於全新目錄首次啟動時，自動安全檢查並釋放 `LICENSE`、`THIRD-PARTY-NOTICES.md` 與雙語 `readme`，確保單檔運行時檔案系統合規性完整，且具備冪等性不覆蓋使用者自訂檔案。

### Changed
- **依賴目錄配置與文件標準化**：`conf/dependencies.json`、`internal/config/dependencies.go` 與 `scripts/check_deps.go` 擴充 `license`、`homepage` 與 `source_url` 欄位，並更新 `docs/dependencies.md` 依賴說明表格。
- **背景檢查更新冷卻防抖**：依賴庫管理彈窗加入 60 秒冷卻時間機制，避免頻繁開關彈窗時無效重複向遠端發起網路請求。

### Fixed
- **聲明彈窗 Carbon 暗色主題穿透與按鈕對比度修復**：修復 Carbon 暗色主題下聲明彈窗因背景透明度過高導致底層文字穿透的問題；修復「關閉」按鈕白底白字問題，並優化 Hover 視覺回饋。

## Getting Started
1. 下載 `wincmp-v2.1.6-win-x64.zip`。
2. 解壓縮至您系統中的任何資料夾。
3. 按兩下 `WinCMP_v2.1.6.exe` 啟動控制面板。
