# TODO List

## Issue 1. 新增帶底線 Domains 問題

1. Web Project -> Add Project 
2. 新增一個帶底線的項目, 如CMS_Global
3. 由於 Domains 檢查中禁止了底線, 預設 Domains 會是 local-cms-global.test 
4. 但如果用戶手動改了 Domains 為有底線, 如 local-cms.cms_global.com
5. 點擊 Save 後, Termainal Logs會顯示: 更新系統 Hosts 失敗
6. 此時項目Caddy檔, (如cms_global.caddy) 會 Fallback 為一個奇怪的 Domain (如local-cms_global.test, 既不是預設的local-cms-global.test, 也不是用戶輸入的local-cms.cms_global.com) .而是這個 Domain 也沒有被加進 Hosts 內, 因此項目會處於無法訪問的狀態

### 解決方法: caddy檔內的 domains 要和 Projects 內用戶輸入和顯示的 Domains 要保持一致, Hosts檔可以提示寫入失敗, 通知用戶手動更新Hosts即可

### 修復摘要
Bug 根源
1. generateCaddyfiles() 中的 fallback 邏輯有問題：當用戶輸入的 domains 含底線時，會被 validDomainPattern 驗證過濾掉，然後 fallback 到 "local-" + proj.Name + ".test"
2. 這個 fallback 值並非用戶輸入，也非預設值，導致 Caddyfile 和 UI 顯示不一致