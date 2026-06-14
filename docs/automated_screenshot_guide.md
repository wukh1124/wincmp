# 📸 WinCMP 自動化擷圖指南 (Automated Screenshot Guide)

本指南說明如何使用自動化擷圖腳本，一鍵產出官網與展示所需要的多主題、多語系精美截圖。

---

## 📖 簡介

為了免去手動擷圖的繁瑣流程，我們在 `frontend/scripts/capture.cjs` 實作了基於 **Playwright** 的自動化擷圖腳本。
它會自動啟動無頭瀏覽器（Headless Browser）並模擬使用者操作，擷取以下兩個主題的 8 個主要頁面（共 16 張截圖）：
- **Carbon (暗色主題)**：儲存於專案根目錄的 `screenshot/dark/`
- **Sketch (手繪亮色主題)**：儲存於專案根目錄的 `screenshot/sketch/`

### 🎨 擷取規格
- **解析度**：`1264 x 729` px (哥哥指定的理想官網展示比例)
- **輸出品質**：雙倍像素清晰度 (`deviceScaleFactor: 2`)，確保在 Retina 螢幕上依然銳利不模糊。

---

## 🛠️ 前置準備與安裝

自動擷圖腳本依賴於 **Playwright (Chromium)**。請在執行前確認已安裝相關套件。

### 1. 安裝依賴
在 `frontend/` 目錄下執行以下指令安裝依賴庫：
```powershell
cd frontend
npm install -D playwright
```

*如果本地尚未安裝過 Playwright 的瀏覽器核心，請接著執行：*
```powershell
npx playwright install chromium
```

### 2. 確認 Wails 開發伺服器運行
自動擷圖腳本需要連線到 Wails 內建的本地開發伺服器（預設網址為 `http://localhost:34115`）。
請確保您已在專案根目錄執行了開發模式：
```powershell
wails dev
```

---

## 🚀 執行步驟

當 `wails dev` 成功運作且前端可以正常瀏覽後，打開另一個終端機，執行以下指令：

```powershell
cd frontend
npm run screenshot
```
或者也可以手動透過 `node` 執行該腳本：
```powershell
node scripts/capture.cjs
```

執行後，腳本會輸出連線狀態，並依序切換主題、點擊選單，自動將截圖儲存至：
- `wincmp/screenshot/dark/`
- `wincmp/screenshot/sketch/`

---

## ⚠️ 重要注意事項與技術細節

為了保證擷圖畫面乾淨、美觀且數據載入完整，腳本中實作了以下防護與操作機制：

### 1. 徹底阻擋 Onboarding 導引氣泡
* **問題**：React 在初始化時若發現 `localStorage` 中無標記，會自動彈出「Dependency Management Guide」等引導氣泡，這會遮擋儀表板的按鈕。
* **解決方案**：腳本在瀏覽器建立 `context` 後，利用 `context.addInitScript()` 在 **DOM/JS 載入前** 預先將 `wincmp_onboarding_shown` 與 `wincmp_dep_onboarding_shown` 設定為 `'true'`，從源頭徹底阻擋氣泡的渲染。

### 2. 資料庫瀏覽器 (DB Explorer) 自動展示 Schema
* **機制**：為了讓 `db_explorer.png` 的截圖內容更加豐富，腳本在切換到資料庫瀏覽器後，會自動在左側資料庫列表中尋找包含 `information_schema` 的按鈕並點擊，並等待 `800ms` 的資料表載入與渲染動畫。

### 3. 資源監視器 (Resource Monitor) 載入延遲
* **機制**：系統資源占用數據與圖表的繪製需要時間載入。腳本在切換到 `resource_monitor` 後，會強制等待 `2500ms` 以確保 "Loading..." 動畫完全消失並繪出數據，避免擷取到空白的載入畫面。

### 4. 遮罩與彈窗自動關閉
* **機制**：Playwright 在點擊側邊欄導覽按鈕時，如果畫面上殘留有 Drawer 抽屜（如新增專案）或 Dialog 彈窗（如依賴庫管理器），會因為半透明遮罩層的阻擋而導致點擊超時。
* **腳本處理**：
  - 在進入 `projects` 頁面擷圖前，會自動點擊關閉「新增專案」的抽屜。
  - 在進入 `db_explorer` 頁面前，會自動點擊關閉「專案終端」的面板。
  - 在每輪主題擷圖結束後，會自動關閉「依賴庫管理器」彈窗，避免影響下一輪主題擷圖。

### 5. 官網截圖同步
* 產出的截圖會儲存在專案根目錄的 `screenshot/` 下，若要將最新截圖同步至官網，請在 release 時將它們複製到 `website/screenshot/` 目錄。
