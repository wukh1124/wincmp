# 🚀 WinCMP - 極簡 Windows 開發環境

WinCMP (**Win**dows + **C**addy + **M**ariaDB + **P**HP) 是一個基於 Go 與 Wails (React + TypeScript) 開發的「綠色免安裝」開發環境控制面板。旨在提供比 XAMPP 更現代、更輕量、且完全可控的 Windows 本機開發體驗。

---

### 📦 核心堆疊 (Core Stack)
- **Caddy 2**: 現代化的高效能 Web 伺服器，自帶 HTTPS 與自動化配置。
- **MariaDB**: 穩定可靠的關聯式資料庫，專為 Windows 環境優化。
- **PHP**: 多版本共存與切換支援，滿足不同專案的依賴需求。
- **Runtime**: 支援 Node.js、Bun、Python、Go (Air/Run)、Custom 等多種開發環境運行。

### ✨ 項目亮點 (Key Highlights)
- **🕹️ 直觀管理**：整合式儀表板，即時監控服務狀態與進程資源 (PID)。
- **🌐 多專案並行**：支援多種 Runtime (Node.js, Bun, Python, Go, Custom) 的開發環境並行運行。
- **🏗️ Runtime 系統**：支援 Background / Terminal 雙模式啟動，動態注入環境變數，確保環境絕對隔離。
- **🛡️ 零依賴設計**：完全綠色化，不污染系統環境變數，不寫入登錄檔。
- **🔍 Preset 自動偵測**：自動識別 Laravel, Next.js, Nuxt, Astro, Vite, Python (Django/FastAPI/Flask), PocketBase, Go API 等框架，自動配置啟動指令與端口。
- **📝 全面記錄**：標籤化分類日誌系統 (System, Caddy, DB, PHP, Runtime)，支援按專案篩選，排錯快如閃電。

---
> [!TIP]
> **為什麼選擇 WinCMP？** 因為開發環境應該是「解壓即用，關閉即走」的，而不應該成為系統的負擔。
