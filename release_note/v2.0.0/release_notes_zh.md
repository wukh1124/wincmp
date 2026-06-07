# WinCMP v2.0.0
此版本為 WinCMP 帶來了新的功能、更新與修正。

## What's Changed

### Added
- **全面重構 GUI 核心**：遷移至 **Wails v2** + **React 18** + **TypeScript**，大幅提升效能並降低資源佔用。
- **深色專業級設計**：採用現代化 Dark Professional 介面，使用 Tailwind CSS 與 Zustand 狀態管理。
- **全新功能面板**：包括高效能即時日誌終端、基於 TanStack Table 的專案與資料庫瀏覽器。

### Removed
- **移除舊版 GUI**：完全移除舊有的 Go Fyne 程式碼（已將舊程式碼歸檔至 `legacy_fyne/` 目錄中）。

## Getting Started
1. 下載 `wincmp-v2.0.0-win-x64.zip`。
2. 解壓縮至您系統中的任何資料夾。
3. 按兩下 `WinCMP_v2.0.0.exe` 啟動控制面板。