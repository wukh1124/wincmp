# WinCMP 版本發布與落地頁部署備忘錄 (Release Memo)

本文件記載 WinCMP 新版本上線時的標準發布流程，以及 GitHub Actions、GitHub Pages 落地頁與動態 Changelog 機制的設定說明。

---

## 1. 落地頁架構與部署機制

WinCMP v2 落地頁程式碼完全放置於 `website/` 目錄中，並與專案內部的開發/架構文檔 (`docs/`) 進行物理隔離。

### 1.1 GitHub Actions 自動化部署
我們使用 `.github/workflows/deploy-website.yml` 來處理落地頁的部署。
- **觸發時機**：
  - 當有變更推送到 `main` 分支的 `website/` 資料夾時。
  - 當有新的 GitHub Release 發布時（確保最新版本資訊在靜態編譯時可能需要更新，或手動重啟觸發）。
- **執行動作**：
  1. 檢出 (Checkout) 專案。
  2. 自動將根目錄的最新 `icon.svg` 複製為 `website/favicon.svg`。
  3. 自動將根目錄下的最新 `screenshot/` 資料夾完整複製到 `website/screenshot/` 中，以確保落地頁在線上能正確載入最新截圖。
  4. 使用 `peaceiris/actions-gh-pages` 動作，將 `website/` 資料夾的內容推送到專案的 `gh-pages` 部署分支。

### 1.2 GitHub Pages 設定步驟
為了讓落地頁順利在線上開啟，必須在 GitHub 倉庫進行以下一次性設定：
1. 前往 GitHub 專案倉庫，點選 **Settings** -> **Pages**。
2. 在 **Build and deployment** 下的 **Source** 選擇 **Deploy from a branch**。
3. 在 **Branch** 選單中，選擇 **`gh-pages`** 且目錄選擇 **`/ (root)`**。
4. 點擊 **Save**。
5. 稍等數分鐘，網站即可在 `https://<你的 GitHub 帳號>.github.io/wincmp/` 正常訪問。

---

## 2. 動態 Changelog 與版本下載機制

落地頁的「最新版本號」、「立即下載按鈕」以及「更新日誌」採用**客戶端動態串接機制**，不需要手動更新 HTML 檔案！

- **核心技術**：透過 `website/main.js` 呼叫 GitHub 的官方公開 API：
  `https://api.github.com/repos/wukh1124/wincmp/releases/latest`
- **解析邏輯**：
  1. 擷取 Release 資訊中的 `tag_name`（如 `v2.0.0`），動態更新首頁的最新版本 Badge。
  2. 掃描 Release 的 `assets`，若發現檔名以 `.exe` 結尾的二進位檔案，則將「立即下載」按鈕連結指向該 `.exe` 的下載網址；若無，則退回 GitHub Release 頁面網址。
  3. 讀取 Release 的 `body` (即發布時寫的 Markdown 更新日誌)，並利用 `marked.js` 解析器將其轉譯為 HTML，動態渲染至更新日誌容器內。
- **Rate Limit 防護 (快取)**：
  為避免大量流量訪問時觸發 GitHub API 的速率限制（未驗證的 API 每小時上限為 60 次），`main.js` 內建了 **15 分鐘的 LocalStorage 快取機制**。如果在 15 分鐘內重複造訪網頁，將直接從用戶瀏覽器本機快取讀取版本資料。
- **容錯備份**：
  如果 API 存取失敗且瀏覽器無任何舊快取，落地頁會顯示備份的 `v2.0.0` 靜態下載連結，並提示使用者點選連結直接去 GitHub 倉庫查看 Release 歷史。

---

## 3. 標準新版本發布流程 (Release Checklist)

當開發完新功能，準備將 WinCMP 新版本發布上線時，請按照以下步驟執行：

### 🟩 第一步：程式碼檢查與測試
- [ ] 執行 Go 後端單元測試，確保核心代碼無誤：
  ```powershell
  go test ./...
  ```
- [ ] 確認 `frontend/` 已成功編譯，無殘留的 Debug 模式設定。
- [ ] 確認 Hosts 檔案修改提權功能文案準確。

### 🟩 第二步：更新專案版本號
- [ ] 修改專案根目錄的 `VERSION` 檔案，填入新的版本號（例如 `2.0.1`）。

### 🟩 第三步：編譯正式發布版二進位檔案
- [ ] 使用 Wails CLI 編譯適用於生產環境的無主控台、壓縮 symbols 之二進位程式：
  ```powershell
  wails build -clean -ldflags "-s -w -X main.AppVersion=v2.0.1"
  ```
  *(註：`-X main.AppVersion=v...` 可依專案實際變數注入需要決定是否添加)*

### 🟩 第四步：建立 Git Tag 並推送至 GitHub
- [ ] 在本地端建立對應版本的 Git Tag：
  ```powershell
  git tag v2.0.1
  git push origin v2.0.1
  ```

### 🟩 第五步：在 GitHub 上建立正式 Release
- [ ] 前往 GitHub 專案的 **Releases** 頁面，點擊 **Draft a new release**。
- [ ] 選擇剛剛推送的 Tag `v2.0.1`。
- [ ] 在標題輸入 `WinCMP v2.0.1 - [主題]`（例如：`WinCMP v2.0.1 - 終端效能優化與 Bug 修復`）。
- [ ] **撰寫更新日誌 (Changelog)**：在內容框中使用標準 Markdown 格式撰寫本次更新的細節（例如 `### ✨ 新增功能`、`### 🐛 Bug 修復`）。
- [ ] **上傳二進位檔**：將剛剛編譯完成的 `build/bin/wincmp.exe` 檔案拖曳上傳至 Release 的 Assets 中（請確保檔名以 `.exe` 結尾）。
- [ ] 點擊 **Publish release**。

### 🟩 第六步：確認落地頁自動更新
- [ ] 發布 Release 後，GitHub Actions 會自動執行部署（或因網頁讀取 API，用戶訪問網頁時會即時抓取）。
- [ ] 打開 `https://wukh1124.github.io/wincmp/`，確認最新版本號是否已變更為 `v2.0.1`，下載連結是否正確指向新版 `.exe`，且 Changelog 是否正確渲染。
