# WinCMP - Agent 開發指南

本檔案提供 Agent 程式碼助手在此程式碼庫中進行開發時所需的一切資訊。

---

## 1. 建置 / 測試指令

### 前置需求
- Go 1.25+
- MinGW-w64 (WinLibs) — 用於 Fyne Cgo 依賴的 C 編譯器
- 確認 `gcc -v` 可正常執行

### 依賴管理
```cmd
go mod tidy
```

### 建置指令
```cmd
# 開發建置
go build -v -o wincmp.exe .

# 正式發布（無 CMD 視窗）
go build -v -o wincmp.exe -ldflags "-H windowsgui" .

# 正式發布 + 縮小體積
go build -ldflags "-H windowsgui -s -w" -o wincmp.exe .

# 使用 Fyne 打包（包含圖示與資源）
fyne package -release
```

### 執行單一檔案或套件
```cmd
# 執行指定套件
go run ./internal/config

# 建置指定套件
go build -o test.exe ./internal/config
```

### 測試
- 目前此專案**尚無測試檔案**
- 若要新增測試，請將測試檔案命名為 `*_test.go`，放在被測試程式碼的同目錄
- 執行所有測試：`go test ./...`
- 執行單一套件測試：`go test ./internal/config`

---

## 2. 程式碼風格規範

### 語言與註解
- **主體語言**：繁體中文（zh-TW），所有註解與說明文件使用正體中文
- **程式碼本身**：英文識別符（遵循 Go 慣例）

### 套件結構
```
main.go                 # 應用程式進入點：GUI 建構、事件處理、配置生成
internal/
├── config/            # JSON 設定檔的讀寫與資料結構定義
├── scanner/           # bin/ 目錄掃描器：偵測已安裝服務版本與 Port 計算
├── process/           # 子行程生命週期管理器
│   ├── manager.go    # Manager 核心：register/unregister/StopAll
│   ├── caddy.go      # Caddy 服務：StartCaddy / StopCaddy / ReloadCaddy
│   ├── mariadb.go    # MariaDB 服務：StartMariaDB / StopMariaDB
│   └── php.go        # PHP-CGI 服務：StartPHPCGI / StopPHPCGI（多行程）
├── detect/            # 專案類型偵測器（Laravel）
└── hosts/            # Windows Hosts 檔管理
```

### 命名慣例
| 元素 | 慣例 | 範例 |
|------|------|------|
| 匯出的型別 | PascalCase | `WincmpConfig`, `ServiceInfo` |
| 匯出的函式 | PascalCase | `Load()`, `StartCaddy()` |
| 未匯出（私用） | camelCase | `baseDir`, `serviceKey` |
| JSON struct tag | snake_case | `json:"default_www"` |
| 常數 | camelCase | `caddyServiceKey` |
| 套件名稱 | 小寫 | `config`, `scanner` |

### Import 組織順序
標準庫放最前，第三方套件次之，內部套件最後：

```go
import (
    "fmt"
    "os"
    "path/filepath"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"

    "wincmp/internal/config"
    "wincmp/internal/process"
)
```

### 錯誤處理
```go
// 使用 fmt.Errorf 搭配 %w 包裝錯誤上下文
return nil, fmt.Errorf("無法讀取設定檔 %s: %w", path, err)

// 盡早檢查錯誤
if err != nil {
    return fmt.Errorf("操作失敗: %w", err)
}

// 非關鍵錯誤可記錄後繼續執行
if err != nil {
    m.errorLog("category", "描述", err)
    return // 或優雅地處理
}
```

### 結構體定義範例
```go
type ServiceInfo struct {
    Name    string `json:"name"`
    Version string `json:"version"`
    ExePath string `json:"exe_path"`
}

type PHPVersionInfo struct {
    Version   string `json:"version"`
    ExePath   string `json:"exe_path"`
    MajorMin  string `json:"major_min"`
    PortBase  int    `json:"port_base"`
    PortCount int    `json:"port_count"`
}
```

### 並行處理模式
```go
// 使用 sync.Mutex 保護共享狀態
type Manager struct {
    mu       sync.Mutex
    services map[string]*ServiceState
}

// 務必使用 defer 確保解鎖
func (m *Manager) IsRunning(serviceKey string) bool {
    m.mu.Lock()
    defer m.mu.Unlock()
    // ...
}

// 使用 context 處理可取消的作業
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
```

### 檔案權限
- 目錄：`0755`
- 檔案：`0644`

### 路徑處理（Windows 相容）
- 使用 `path/filepath` 確保跨平台相容性
- 使用 `filepath.Join()` 而非字串串接
- 必要时使用 `os.PathSeparator`

### 日誌記錄模式
```go
// 依分類記錄日誌，含時間戳
func (m *Manager) log(category string, format string, args ...interface{}) {
    if m.logFn != nil {
        m.logFn(category, fmt.Sprintf(format, args...))
    }
}

// 使用 Emoji 前綴表示狀態
m.log("php", "🚀 啟動 PHP-CGI %s...", version)
m.log("php", "✅ PHP-CGI %s 已啟動", version)
m.log("php", "🛑 停止 PHP-CGI %s...", version)
```

---

## 3. 架構設計原則

### 核心設計決策
- **可攜性**：不改變系統 PATH，不寫入登錄檔
- **隔離性**：每個 PHP 版本啟動時動態注入各自的 PATH
- **最小改動**：追求最少 diff，避免過度工程

### 服務識別鍵格式
| 服務 | 格式 | 範例 |
|------|------|------|
| Caddy | 直接用字面值 | `"caddy"` |
| MariaDB | `mariadb-` + 版本 | `"mariadb-11.4"` |
| PHP | `php-` + 版本 | `"php-8.2.30"` |

### PHP Port 配置規則
```
3<主版本><次版本>00  （例：PHP 8.2 → 38200 起）
```
- Port 基數由 `calcPHPPortBase()` 計算
- 每版本預設啟動 3 個行程，可設定 3/10/20/.../100

### JSON 設定檔位置
- 主設定檔：`conf/wincmp.json`
- 自動產生：`conf/sites/*.caddy`、`conf/snippets/php-upstream.caddy`

---

## 4. Fyne GUI 開發要點

### 主題覆寫模式
```go
// 自訂主題以覆寫特定顏色
type coloredButtonTheme struct {
    fyne.Theme
    isStop func() bool
}

func (m *coloredButtonTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
    // 覆寫特定顏色邏輯
}
```

### Fyne 執行緒安全
```go
// UI 更新必須在主執行緒執行
fyne.Do(func() {
    widget.Refresh()
})
```

### 資料綁定模式
```go
uptimeData := binding.NewString()
uptimeLabel := widget.NewLabelWithData(uptimeData)

// 從任何 goroutine 更新
uptimeData.Set("12:34:56")
```

---

## 5. 常見任務範例

### 新增一項服務
1. 在 `process/manager.go` 新增服務鍵常數
2. 建立新檔案：`process/<service>.go`
3. 實作 `Start<Service>()`、`Stop<Service>()`
4. 在 `main.go` 的 `createDashboard()` 中加入 UI 列

### 新增設定欄位
1. 在 `internal/config/config.go` 的結應結構中新增欄位
2. 更新 JSON struct tag
3. 在 `main.go` 的預設設定建立處初始化

### 修改 Caddy 配置生成
- 編輯 `main.go` 中的 `generateCaddyfiles()` 或 `generatePHPUpstream()`
- 輸出至 `conf/sites/` 與 `conf/snippets/`

---

## 6. 注意事項

### 禁止事項
- **嚴禁**修改系統 PATH 或登錄檔
- **嚴禁**提交 `wincmp.exe` 或其他二進位檔案
- **避免**使用 `git add .`，請指定具體檔案
- **避免**新增無意義註解（除非解釋非直覺邏輯）
- **避免**提前優化
- **避免**修改正常運作的程式碼（無明顯效益時）

### 建置環境提示
- 若無管理員權限安裝 MSYS2，可使用 [WinLibs MinGW-w64](https://winlibs.com/) Zip 版
- 將解壓縮後的 `bin/` 加入**使用者環境變數 (User PATH)**

---

## 7. 參考文件

| 文件 | 用途 |
|------|------|
| `System_design_document.md` | 系統架構與設計決策 |
| `Develop_Task_List.md` | 開發任務清單 |
| `readme.md` | 專案概述與建置說明 |
| `.agent/lessons.md` | 過往經驗與教訓 |

---

> 最後更新：2026-03-18
