# 📅 WinCMP 開發任務清單 (Development Roadmap)

本文件追踪 WinCMP 的開發進度與未來規劃。任務按功能領域分類，並標註實作優先級。

---

## ✅ 已完成項目 (Milestones)

- [x] **Web Projects UI 優化** (2026-03-18)
  - 新增 `Json Config` 與 `Open Project Directory` 等快捷按鈕。
  - 支援快速複製項目網址與開啟 Caddyfile。
- [x] **Laravel 專案自動偵測** (2026-03-16)
  - 基於信心分數制自動判定 Laravel 結構並導向 `public/`。
- [x] **程式通訊埠 (Port) 檢查**
  - 啟動前自動檢查競爭狀態，減少啟動失敗風險。
- [x] **Hosts 本地網域自動管理**
  - 獲取 UAC 權限後自動同步 Caddy Domains 至系統 Hosts。
- [x] **本地 SSL 憑證整合**
  - 支援自定義 SSL 路徑與基本的 HTTPS 執行環境。
- [x] **深色/淺色模式切換**
  - 整合 Fyne 主題系統，支援 UI 視覺風格切換。

---

## 🛠️ 待處理任務 (To-Do)

### 1. 核心系統整合 (System & Core)
- [ ] **Windows 開機啟動** (⭐)
  - 目標：寫入 `HKCU\...\Run` 實現背景自動啟動與服務自動還原。
- [ ] **Windows 環境變數一鍵設定** (⭐⭐)
  - 目標：自動將內部 PHP/Caddy 路徑加入系統 `Path`，方便 CLI 使用。
- [ ] **服務執行檔自動下載器** (⭐⭐⭐)
  - 目標：內建版本管理介面，自動從官方源下載並解壓縮服務。

### 2. 開發工具鏈整合 (Dev Tools)
- [ ] **內建 Composer 支援** (⭐)
  - 目標：內建 `composer.phar` 並與當前 PHP 環境綁定，實現免安裝開發。
- [ ] **DB Explorer 進階功能** (⭐⭐⭐)
  - 目標：預覽 MariaDB 表結構，並提供「一鍵開啟 HeidiSQL」功能。

### 3. 抗性與穩定性優化 (Stability)
- [ ] **PHP 進程 Watchdog** (⭐⭐)
  - 目標：監控 php-cgi 崩潰並自動重啟，搭配 Caddy 健康檢查。

---

## 🚀 階段性實作建議 (Action Plan)

| 階段 | 重點目標 | 關鍵任務 |
| :--- | :--- | :--- |
| **Phase 1** | **工具鏈補完** | 內建 Composer 支援、環境變數設定 |
| **Phase 2** | **自動化與部署** | 開機啟動、服務自動下載器 |
| **Phase 3** | **開發者體驗** | DB Explorer 加強、HeidiSQL 深度整合 |

---
> [!NOTE]
> 難度說明：⭐ (小時級) | ⭐⭐ (天級) | ⭐⭐⭐ (週級)
