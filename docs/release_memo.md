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

落地頁的「最新版本號」、「立即下載按鈕」以及「更新日誌」採用**靜態生成與動態讀取相結合**的機制，完全避免了 GitHub API 速率限制（Rate Limit）問題。

- **核心技術**：透過 `website/main.js` 直接以 AJAX/fetch 讀取落地頁同目錄底下的 `./release.json` 檔案。
- **生成邏輯**：
  1. 部署網站時，由 GitHub Action（或手動執行）啟動 `node scripts/generate-release-json.js`。
  2. 該腳本會讀取 `release_note/release_info.json` 的版本資訊與 `release_note/v[Version]/` 底下的 Markdown 更新日誌。
  3. 自動組合出最新版本的下載連結，並生成包含中英文日誌的 `website/release.json` 檔。
- **解析與渲染**：
  1. `main.js` 讀取 `release.json` 後，擷取並展示最新版本號與下載連結（指向 GitHub Release 的 Zip 檔）。
  2. 依據瀏覽器目前的語系切換設定，動態載入對應的 `changelog_zh` 或 `changelog_en`。
  3. 利用 `marked.js` 解析器將 Markdown 轉譯為 HTML 並渲染到網頁上。
- **容錯與備份**：
  如果 `./release.json` 讀取失敗，落地頁會顯示備份的 `v2.0.0` 靜態下載連結，並引導使用者前往 GitHub 倉庫直接查看 Release。

---

## 3. 標準新版本發布流程 (Release Checklist)

當開發完新功能，準備將 WinCMP 新版本發布上線時，推薦採用 **「PR 合併後手動打 Tag」** 的標準發布流程：

### 🟩 第一步：程式碼檢查與測試
- [ ] 執行 Go 後端單元測試，確保核心代碼無誤：
  ```powershell
  go test ./...
  ```
- [ ] 確認 `frontend/` 已成功編譯，無殘留的 Debug 狀態。

### 🟩 第二步：在本地端準備 Release 資訊
- [ ] 切換到 `develop` 分支。
- [ ] 修改專案根目錄的 `VERSION` 檔案，填入新的版本號（例如 `2.0.2`）。
- [ ] 更新 `packaging/wincmp/CHANGELOG_zh.md` 與 `CHANGELOG.md`，寫入本次版本的更新內容。
- [ ] 在專案根目錄下執行 Release 腳本（用以產生 Release Notes 與更新 `release_info.json`）：
  ```powershell
  ./bat/release.ps1
  ```
- [ ] 提交並推送 `develop` 分支：
  ```powershell
  git add .
  git commit -m "chore(release): prepare for v2.0.2"
  git push origin develop
  ```

### 🟩 第三步：PR 合併至 main 分支
- [ ] 前往 GitHub 建立 Pull Request：`develop` -> `main`。
- [ ] 通過 Review 後，將 PR 合併（Merge）至 `main`。

### 🟩 第四步：推送 Tag 觸發自動發布
- [ ] 在本地切換到 `main` 分支並拉取最新代碼：
  ```powershell
  git checkout main
  git pull origin main
  ```
- [ ] 打上對應版本的 Tag 並推送至 GitHub（注意與 `VERSION` 內容一致）：
  ```powershell
  git tag v2.0.2
  git push origin v2.0.2
  ```

### 🟩 第五步：自動化 CI/CD 與驗證
- [ ] 前往 GitHub **Actions** 頁面，確認 **Build and Release** 工作流已成功啟動並編譯完成。
- [ ] 工作流會自動建立對應版本的 GitHub Release，並自動上傳以下檔案：
  - `wincmp-v2.0.2-win-x64.zip`（完整發布包）
  - `WinCMP_v2.0.2.exe`（單獨執行檔，用於自動更新）
- [ ] Release 發布後，會觸發 **Deploy Website** 工作流，將最新版 `release.json` 與靜態資源部署至 GitHub Pages。
- [ ] 打開落地頁 `https://wukh1124.github.io/wincmp/`，確認最新版本號、下載按鈕連結、Changelog 內容皆已正確更新。
