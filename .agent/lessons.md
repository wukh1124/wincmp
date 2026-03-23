# MariaDB 啟動失敗與初始化處理

## 問題描述
MariaDB 在首次啟動時，若 `datadir`（例如 `data/mariadb`）是空的，會因為找不到系統表（`mysql.db`, `mysql.user` 等）而導致 `Fatal error`。

## 解決方案
在 `StartMariaDB` 邏輯中加入檢索機制：
1. 檢查 `data/mariadb/mysql` 是否存在。
2. 若不存在，先執行 `mariadb-install-db.exe --datadir=...` 進行初始化。
3. 初始化完成後再啟動 `mariadbd.exe`。

## 經驗教訓
- Windows 版本的 MariaDB 免安裝版在首次運行前必須手動或程式化執行初始化命令。
- 在 Go 中啟動服務時，應優先檢查資料完整性。
