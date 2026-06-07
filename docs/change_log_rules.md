# Changelog 格式規範和標準分類定義

**版本定義**

## [x.y.z] - yyyy-mm-dd

x 為 MAJOR、y 為 MINOR、z 為 PATCH，用來傳達程式碼變化含義。
- MAJOR (x)：不相容 API 修改，如移除指令、改變預設行為，強迫用戶調整使用方式。
- MINOR (y)：向下相容新功能，新增指令或參數，不破壞舊腳本；也包括宣告 Deprecated。
- PATCH (z)：向下相容 bug 修復，修正錯誤、效能優化，不加新功能。

**核心規則**：發布後不可修改版本內容；初版用 1.0.0，開發階段用 0.y.z。

**改動內容定義**

### Added
- 新增功能、指令或元件。

### Changed
- 既有功能修改、行為調整（非 bug fix）。

### Deprecated
- 宣告即將移除的功能（警告用戶）。

### Removed
- 完全刪除的功能。

### Fixed
- 修復 bug 或問題。

### Security
- 安全相關修復或強化。

### Dependencies
- 外部依賴變化。