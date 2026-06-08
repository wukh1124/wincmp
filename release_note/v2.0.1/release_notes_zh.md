# WinCMP v2.0.1
此版本為 WinCMP 帶來了新的功能、更新與修正。

## What's Changed

### Added
- **自動化 SSL CA 憑證配置**：新增 PHP 在本地環境缺少憑證時，自動下載並配置 `cacert.pem` 的功能，解決 PHP 本地 SSL 請求失敗的問題。

### Changed
- **依賴優化**：徹底移除已廢棄的 `fyne.io/fyne/v2` 主框架依賴與舊版 Fyne GUI 資源監控死代碼，進一步精簡專案體積與打包速度。
- **Wails 建置修復**：重新將 Wails 建置模版檔（如 `icon.ico` 與 `manifest` 檔案）納入 Git 追蹤，修正因 `.gitignore` 設定不當導致 `wails build` 時找不到 Windows 圖示的打包錯誤。

## Getting Started
1. 下載 `wincmp-v2.0.1-win-x64.zip`。
2. 解壓縮至您系統中的任何資料夾。
3. 按兩下 `WinCMP_v2.0.1.exe` 啟動控制面板。