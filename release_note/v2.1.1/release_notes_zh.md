# WinCMP v2.1.1
此版本為 WinCMP 帶來了新的功能、更新與修正。

## What's Changed

### Fixed
- **終端啟動工作目錄回退**：專案路徑遺失或被移動時，終端啟動不再因無效目錄失敗；自動依序回退到可用路徑，並解析有效的 Shell 絕對路徑，讓專案終端仍可正常開啟。
- **資料庫瀏覽器選取樣式**：資料庫清單選中項目不再使用深色實心填色，改為淺色底與 accent 文字；sketch 主題亦調整為不透明淺藍紙色，避免格線穿透並維持可讀性。

### Dependencies
- **預設依賴版本更新**：更新 Caddy、Composer、HeidiSQL、Mailpit、Node.js、PHP 8.2／8.3／8.4 與 php_redis 等預設依賴版本，並補齊 SHA-256 校驗值，提升依賴下載與驗證可靠性。

## Getting Started
1. 下載 `wincmp-v2.1.1-win-x64.zip`。
2. 解壓縮至您系統中的任何資料夾。
3. 按兩下 `WinCMP_v2.1.1.exe` 啟動控制面板。
