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
5. 執行 `capture.cjs` 產出新圖

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

### 1. 阻擋 Onboarding 氣泡
透過 `context.addInitScript()` 在載入前代理 `window.go.main.App.GetConfig`，強制：
- `wincmp_onboarding_shown`
- `wincmp_dep_onboarding_shown`
- `wincmp_sidebar_guide_shown`

### 2. DB Explorer 自動點開 information_schema
切換後尋找含 `information_schema` 的按鈕並點擊，等待 800ms。

### 3. Resource Monitor 延遲
等待 2500ms，避免擷到 Loading 畫面。

### 4. 遮罩 / 彈窗清理
- 新增專案 Drawer、專案終端、依賴管理器會在流程中自動關閉
- 每輪主題結束後關閉依賴管理器，避免影響下一輪

### 5. 官網同步
- 目前使用圖：`screenshot/{dark,sketch}/`（需提交）
- 歷史備份：`screenshot/backup/vX.Y.Z/`（不提交）
- 部署：`deploy-website.yml` 的 `cp -r screenshot/* website/screenshot/`
