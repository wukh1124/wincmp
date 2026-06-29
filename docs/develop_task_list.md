# 📅 WinCMP 開發任務清單 (Development Roadmap)

本文件追踪 WinCMP 的開發進度與未來規劃。任務按功能領域分類，並標註實作優先級。

---

## ✅ 已完成項目 (Milestones)

### UX與效能優化 (UX & Performance)
- [x] **WebUI 預設配置與啟動閃爍優化**
  - 調整預設配置為 `Sketch`（線稿）主題、`大 (large)` 字型大小，並支持 Windows 系統語言自動適應。
  - 解決 WebView2 啟動時長達數秒的深灰色背景與主題切換造成的 UI 閃爍。啟用 Wails 的 `StartHidden` 屬性，在前端 React 完全載入且套用主題後再顯式呼叫後端顯示視窗，並附帶 5 秒安全超時兜底顯示。
- [x] **系統托盤 (System Tray) 穩定性加強**
  - 解決應用程式長時間在背景掛載（一星期以上）時可能導致 Windows 系統匣卡死打不開的隱患。
  - 當應用程式最小化至托盤（隱藏狀態）時，**自動凍結/暫停**資源監控的 Loop，達到背景掛機 0% CPU 消耗，徹底防範控制代碼（Handles）洩漏與 Windows Explorer 訊息隊列阻塞。

---

## 🛠️ 待處理任務 (To-Do)

### 開發工具鏈整合 (Dev Tools)
- [ ] **內建 Composer 支援** (⭐)
  - 目標：內建 `composer.phar` 並與當前 PHP 環境綁定，實現免安裝開發。



---

## 🚀 階段性實作建議 (Action Plan)



---
> [!NOTE]
> 難度說明：⭐ (小時級) | ⭐⭐ (天級) | ⭐⭐⭐ (週級)
