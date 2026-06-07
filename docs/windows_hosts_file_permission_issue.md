# Windows hosts file 寫入議題

## 問題背景

WinCMP 需要自動將專案網域寫入 Windows hosts 檔案，以便本機開發時使用自訂域名（如 `local-www.test`）。

**目標檔案**：`C:\Windows\System32\drivers\etc\hosts`

---

## 錯誤現象

### 第一階段：LockFileEx 失敗

```
❌ 更新系統 Hosts 失敗 (請嘗試以管理員權限執行 WinCMP): 無法鎖定 hosts 檔案: Access is denied.
```

程式碼使用 `LockFileEx` 進行獨佔鎖定時失敗，誤以為是權限問題。

### 第二階段：臨時檔案創建失敗

```
無法寫入 hosts 檔案 (可能需要管理員權限): open C:\Windows\System32\drivers\etc\hosts.tmp: Access is denied.
```

嘗試在 `etc\` 目錄下創建臨時檔案 `hosts.tmp` 來測試權限，但失敗。

---

## 排查過程

### 1. 檢查 hosts 檔案權限

```powershell
icacls "C:\Windows\System32\drivers\etc\hosts"
```

預設權限：
```
NT AUTHORITY\SYSTEM:(I)(F)        # 完全控制
BUILTIN\Administrators:(I)(F)     # 完全控制
BUILTIN\Users:(I)(RX)             # 只有讀取權限
```

### 2. 發現特殊權限

```powershell
Get-Acl 'C:\Windows\System32\drivers\etc\hosts' | Format-List
```

**關鍵發現**：hosts 檔案被授予了額外的寫入權限！

```
S-1-5-21-{domain-rid}-{user-rid}-5617 Allow  Write, ReadAndExecute, Synchronize
S-1-5-21-{domain-rid}-{user-rid}-5622 Allow  Write, ReadAndExecute, Synchronize
S-1-5-21-{domain-rid}-{user-rid}-5625 Allow  Write, ReadAndExecute, Synchronize
```

> 💡 這些是企業環境中的特定用戶/群組 SID，表示該環境已自訂 hosts 檔案權限。

### 3. 確認用戶所屬群組

```powershell
whoami /groups
```

用戶 `DOMAIN\username` 所屬群組包含：
```
S-1-5-21-{domain-rid}-{user-rid}-5625  # 與 hosts ACL 匹配！
```

**結論**：用戶**本來就有 hosts 檔案的寫入權限**！

### 4. 驗證直接寫入

```cmd
echo "test line" >> C:\Windows\System32\drivers\etc\hosts
```

✅ **成功**！

---

## 問題根源分析

### 問題 1：LockFileEx 失敗

**原因**：`LockFileEx` 需要特定的檔案共享模式，且可能與其他程式衝突。

**解決**：移除 `LockFileEx`，直接寫入。

### 問題 2：臨時檔案創建失敗

**原因**：用戶有 **hosts 檔案**的寫入權限，但沒有 **etc\ 目錄**的寫入權限。

| 操作 | 結果 | 原因 |
|------|------|------|
| 修改 hosts 檔案 | ✅ | 有檔案寫入權限 |
| 在 etc\ 創建新檔案 | ❌ | 沒有目錄寫入權限 |

**解決**：移除臨時檔案測試，直接開啟 hosts 檔案。

---

## 最終解決方案

### 簡化的 UpdateHosts 實作

```go
func UpdateHosts(domains []string) error {
    if len(domains) == 0 {
        return nil
    }

    safeDomains := sanitizeDomains(domains)
    if len(safeDomains) == 0 {
        return fmt.Errorf("所有域名均未通過安全性驗證，已拒絕寫入 hosts")
    }

    // 直接以追加模式開啟 hosts 檔案
    f, err := os.OpenFile(HostsFilePath, os.O_APPEND|os.O_WRONLY, 0644)
    if err != nil {
        return fmt.Errorf("無法開啟 hosts 檔案進行寫入 (可能需要管理員權限): %w", err)
    }
    defer f.Close()

    // 寫入時間戳和域名
    timestamp := time.Now().Format("2006-01-02 15:04:05")
    if _, err := f.WriteString(fmt.Sprintf("\n# Added by WinCMP at %s\n", timestamp)); err != nil {
        return err
    }

    for _, domain := range safeDomains {
        line := fmt.Sprintf("127.0.0.1  %s\n", domain)
        if _, err := f.WriteString(line); err != nil {
            return err
        }
    }

    return nil
}
```

### 關鍵改變

1. ❌ 移除 `LockFileEx` 鎖定機制
2. ❌ 移除臨時檔案權限測試
3. ✅ 直接用 `os.OpenFile` 以追加模式開啟 hosts

---

## 教訓總結

### ⚠️ 常見誤區

| 誤區 | 真相 |
|------|------|
| hosts 檔案需要管理員權限 | 不一定！企業環境可能已授予用戶寫入權限 |
| LockFileEx 可以保護寫入 | 對系統檔案可能失敗，且非必要 |
| 創建臨時檔案可以測試權限 | 只能測試目錄寫入權限，不能測試檔案寫入權限 |

### ✅ 最佳實踐

1. **先測試直接寫入**：用 `echo >> hosts` 或程式碼直接測試
2. **檢查實際權限**：用 `Get-Acl` 檢查 hosts 檔案的完整 ACL
3. **區分檔案與目錄權限**：修改檔案 ≠ 創建新檔案
4. **簡化實作**：不需要 LockFileEx，直接寫入即可

### 🔍 排查指令速查

```powershell
# 檢查 hosts 檔案權限
icacls "C:\Windows\System32\drivers\etc\hosts"

# 檢查完整 ACL（顯示 SID）
Get-Acl 'C:\Windows\System32\drivers\etc\hosts' | Format-List

# 檢查用戶所屬群組
whoami /groups

# 測試直接寫入
echo "test" >> C:\Windows\System32\drivers\etc\hosts
```

---

## 相關檔案

- `internal/hosts/hosts.go` - hosts 檔案寫入實作
- `main.go` - `triggerHostsUpdate()` 呼叫端

---

## 更新記錄

| 日期 | 變更 |
|------|------|
| 2026-04-16 | 初始版本，記錄 LockFileEx 失敗和臨時檔案權限問題 |
