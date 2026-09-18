# WinCMP 自動化擷圖指南 (Automated Screenshot Guide)

本指南說明如何在 **release 流程中**產出官網展示用的多主題截圖。

---

## 簡介

擷圖腳本：
- `frontend/scripts/capture.cjs`（Playwright 核心）
- `scripts/capture_release_screenshots.ps1`（release 包裝：backup + 連線檢查 + 執行擷圖）

擷取 2 個主題 × 8 個頁面 = 16 張：
- **Carbon (暗色)** → `screenshot/dark/`
- **Sketch (手繪亮色)** → `screenshot/sketch/`

規格：`1264 x 729`，`deviceScaleFactor: 2`

擷圖期間腳本會**強制**下列偏好（不受開發機 `conf/wincmp.json` 或 localStorage 影響）：
- 字體大小：**S（small）**
- 介面語系：**en-US（英文）**
- 側邊欄：**鎖定展開**（避免 projects 頁自動收合）

並在結束時驗證：字體 S、英文語系、主題 id 是否正確。

---

## 前置準備

```powershell
cd frontend
npm install -D playwright
npx playwright install chromium
```

**必須**在專案根目錄啟動 Wails 開發伺服器：

```powershell
wails dev
# 目標: http://localhost:34115
```

---

## Release 標準用法（推薦）

```powershell
# 必須在專案根目錄執行（脚本會自動向上尋找 VERSION）
powershell -ExecutionPolicy Bypass -File .\scripts\capture_release_screenshots.ps1
```

腳本會：
1. 讀取 `VERSION`（目標發布版）
2. 從 `release_info.json` 或 git tag 取得**發布前舊版**版號
3. 將現有截圖備份到 `screenshot/backup/v{舊版}/`（此目錄已 gitignore）
4. 確認 `localhost:34115` 可連線；未啟動 `wails dev` 會直接中止並提示
5. 執行 `capture.cjs` 產出新圖，並輸出字體／語系／主題驗證結果

擷圖完成後請檢查品質，再將 `screenshot/` 一併 commit。  
官網部署時會自動把 `screenshot/*` 複製到 `website/screenshot/`。

---

## 僅執行擷圖（不 backup）

```powershell
cd frontend
npm run screenshot
# 或
node scripts/capture.cjs
```

---

## 技術細節

### 1. 強制擷圖偏好（字體 S、英文）
透過 `context.addInitScript()`：
- 寫入 `localStorage`：`wincmp-font-size=small`、`wincmp_sidebar_locked=true`
- 代理 `window.go.main.App.GetConfig`：強制 `font_size=small`、`language=en-US`，並關閉 onboarding 氣泡
- 代理 `SaveQuickSettings` / `SaveConfig` 為 no-op，避免把擷圖偏好寫回開發機 `conf/wincmp.json`
- Playwright context 設定 `locale: 'en-US'`，讓 `navigator.language` 回退也是英文
- 擷圖前後會以 `data-font-size`、UI 文案、側邊欄寬度再驗證一次，必要時點擊快速切換鈕補償

### 2. 側邊欄一致性
App 在切到 `projects` 分頁時若未鎖定側邊欄會自動收合。  
擷圖腳本會鎖定 `wincmp_sidebar_locked`，並在**每張擷圖前**檢查／展開側邊欄，確保官網截圖側邊欄一致展開。

### 3. 主題切換
- Cycle 順序：`carbon → cream → sketch → carbon`
- 腳本只產出 `carbon`（dark）與 `sketch`
- 切換後讀取 `data-theme` 驗證，最多嘗試 6 次
- 每輪主題結束後關閉依賴管理器，避免遮擋下一輪

### 4. 延遲設定
| 場景 | 延遲 | 原因 |
|------|------|------|
| 導覽切換 / 擷圖前穩定 | 400ms | 頁面與動畫 |
| Drawer / 依賴管理器 | 500–600ms | 展開／關閉動畫 |
| 專案終端 PTY | 1200ms | 終端啟動 |
| DB Explorer `information_schema` | 800ms | 資料表載入 |
| Resource Monitor | 2500ms | 避免 Loading 畫面 |
| 主題切換 | 700ms | React 重繪 |

### 5. 阻擋 Onboarding 氣泡
透過 GetConfig 代理強制：
- `wincmp_onboarding_shown`
- `wincmp_dep_onboarding_shown`
- `wincmp_sidebar_guide_shown`

### 6. DB Explorer 自動點開 information_schema
切換後尋找含 `information_schema` 的按鈕並點擊，等待 800ms。

### 7. 官網同步
- 目前使用圖：`screenshot/{dark,sketch}/`（需提交）
- 歷史備份：`screenshot/backup/vX.Y.Z/`（不提交）
- 部署：`deploy-website.yml` 的 `cp -r screenshot/* website/screenshot/`

---

## 產出頁面清單

| 檔名 | 內容 |
|------|------|
| `dashboard.png` | 儀表板服務卡片 |
| `add_new_project.png` | 新增專案 Drawer |
| `projects.png` | 專案列表（側邊欄展開） |
| `project_terminal.png` | 專案終端 |
| `db_explorer.png` | 資料庫瀏覽（information_schema） |
| `resource_monitor.png` | 資源監控 |
| `settings.png` | 系統設定（應顯示 English + Font Size Small） |
| `wincmp_dependencies.png` | 依賴庫管理彈窗 |

---

## 常見問題

**Q: 為什麼截圖是中文或字體很大？**  
A: 請確認已使用最新 `capture.cjs`。腳本會強制 `font_size=small`、`language=en-US`，並在結尾輸出驗證結果。

**Q: projects 頁側邊欄收起來了？**  
A: 新版腳本會鎖定側邊欄並在每張圖前展開；若仍收合，請檢查 `#btn-toggle-sidebar` 是否可見。

**Q: conf/wincmp.json 被改成英文 / sketch？**  
A: 腳本會攔截 `SaveQuickSettings`/`SaveConfig`。若仍被寫回，代表 Wails 在攔截後重綁了方法，請再執行一次擷圖（腳本會重套攔截），或手動把 conf 改回偏好值。
