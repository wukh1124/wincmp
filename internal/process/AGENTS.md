# internal/process

## OVERVIEW
子行程生命週期管理中心。提供服務註冊、啟動、停止、監控，透過 Windows Job Object 確保崩潰時自動清理。

## STRUCTURE
```
manager.go  - Manager 核心與 ServiceState 結構
job.go      - Windows Job Object 初始化 (JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE)
caddy.go    - Caddy 服務控制
mariadb.go  - MariaDB 服務控制
php.go      - PHP-CGI 多行程管理 (每版本多 Port)
runtime.go  - Runtime (Node/Bun/Python/Go) 控制
```

## WHERE TO LOOK
| 查詢 | 位置 |
|------|------|
| 服務識別 Key 格式 | manager.go:36 註解 |
| 服務狀態結構 | manager.go:16-25 (ServiceState) |
| 註冊/取消註冊 | manager.go:154-177, 237-249 |
| 停止全部服務 | manager.go:251-265 (StopAll) |
| 建立指令 | manager.go:296-303 (createCommand) |
| 輸出導向 | manager.go:307-338 (pipeOutput) |
| PHP 多行程邏輯 | php.go:48-76 |
| PHP Port 計算 | scanner/php.go (不在此目錄) |
| Job Object 初始化 | job.go:19-57 |

## CONVENTIONS

### 服務識別 Key 格式
```
caddy              → "caddy"
mariadb-{version}  → "mariadb-11.4"
php-{version}      → "php-8.2.30"
runtime-{name}     → "runtime-node_abc123"
```

### 服務啟動模板
```go
const serviceKey = "servicename"

func (m *Manager) StartService(...) error {
    if m.IsRunning(serviceKey) {
        return fmt.Errorf("已在運行")
    }
    cmd := m.createCommand(exePath, args...)
    m.pipeOutput(cmd, "category", "Name")
    if err := cmd.Start(); err != nil {
        return err
    }
    m.register(serviceKey, name, exePath, []*exec.Cmd{cmd})
    go m.waitForExit(cmd, serviceKey, "category", "Name")
    return nil
}
```

### 日誌分類
- `system` - Manager 系統事件
- `caddy`  - Caddy 相關
- `mariadb`- MariaDB 相關
- `php`    - PHP-CGI 相關
- `runtime`- Runtime 相關

### Mutex 使用
```go
m.mu.Lock()
defer m.mu.Unlock()
// 僅存取 services map，不包含 I/O
```

### Context 生命週期
- `register()` 建立 `context.WithCancel()`
- `unregister()` 呼叫 `Cancel()`
- 用於偵測服務結束事件

## ANTI-PATTERNS
- ❌ 在 Lock 內執行 I/O (pipeOutput, Process.Kill)
- ❌ 直接存取 services map 而不加鎖
- ❌ 忽略 cmd.Start() 錯誤，未清理已啟動的子行程
- ❌ PHP 啟動失敗時未終止已啟動的程序
- ❌ 使用 fmt.Sprintf 組合錯誤而非 %w 包裝
