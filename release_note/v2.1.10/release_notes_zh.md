# WinCMP v2.1.10

發布日期：2026-11-20

本版本將執行檔改為固定檔名以維持 Windows 工具列釘選，並更新內建相依套件目錄版本。

## What's Changed

### Changed
- **固定執行檔名稱為 `WinCMP.exe`**：發行壓縮包內的執行檔改為固定名稱 `WinCMP.exe`（不再使用 `WinCMP_vX.Y.Z.exe`），應用程式內自動更新覆蓋後也會維持 `WinCMP.exe`，避免每次升級後 Windows 工作列釘選與桌面捷徑失效。

### Security
- **改用 Go 1.27.4 編譯**：修補 Go 標準庫的 `net/http` 伺服器當機漏洞（GO-2026-6859，`MaxBytesReader` 標頭競態）。WinCMP 內建更新器、依賴下載與內嵌 Caddy 反向代理皆會觸及該路徑，因此 `go.mod` 的工具鏈下限同步提高為 1.27.4；本機與 GitHub Actions 產出的執行檔都由此版本編譯。

### Dependencies
- **更新內建相依套件目錄**：同步更新 `conf/dependencies.json` 與 `docs/dependencies.md` 的版本與 SHA-256 校驗碼：
  - Caddy：`2.11.7` -> `2.11.9`
  - MariaDB：`11.4.13` -> `11.4.14`
  - PHP：`8.2.35` -> `8.2.36`、`8.3.36` -> `8.3.37`、`8.4.27` -> `8.4.28`、`8.5.12` -> `8.5.13`
  - Mailpit：`1.31.6` -> `1.34.0`
  - Node.js：`24.22.1` -> `24.22.3`
  - Composer：`2.10.3` -> `2.11.2`
  - HeidiSQL：`12.21` -> `12.22`
  - CA 憑證包 (`cacert`)：`2026-10-20` -> `2026-11-05`

## Getting Started
1. 下載 `wincmp-v2.1.10-win-x64.zip`。
2. 解壓縮至您系統中的任何資料夾。
3. 按兩下 `WinCMP.exe` 啟動控制面板。
4. 從 v2.1.9 升級：更新器會先以 SHA-256 比對 `checksums.txt`，通過後以固定檔名 `WinCMP.exe` 覆蓋現有執行檔。
