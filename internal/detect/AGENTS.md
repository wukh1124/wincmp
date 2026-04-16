# WinCMP - detect 目錄

OVERVIEW: 專案類型偵測器。透過信心分數制識別框架，回傳 Preset 類型與運行配置。

STRUCTURE:
```
detect/
├── framework.go    # 偵測介面定義與 DetectResult 結構
├── laravel.go      # Laravel 偵測 (信心分數制)
├── go.go           # Go 專案偵測
├── python.go       # Python 專案偵測
├── node.go         # Node.js 專案偵測
└── bun.go          # Bun 專案偵測
```

WHERE TO LOOK:
| Preset Type      | 信心分數 | 檢查檔案                     | 備註                    |
|------------------|----------|------------------------------|-------------------------|
| laravel          | 50+      | composer.json, .env          | 有 laravel 鍵         |
| next             | 40+      | package.json                 | 有 next 依賴          |
| nuxt             | 40+      | package.json                 | 有 nuxt 依賴          |
| astro            | 40+      | package.json                 | 有 astro 依賴         |
| vite             | 40+      | package.json, vite.config.*  |                         |
| pocketbase       | 40+      | 可執行 pb 二進位              |                         |
| python_django    | 40+      | manage.py, requirements.txt  | 有 django 依賴        |
| python_fastapi   | 40+      | requirements.txt             | 有 fastapi 依賴       |
| python_flask     | 40+      | requirements.txt             | 有 flask 依賴         |
| go_api           | 50+      | go.mod, .air.toml, main.go   | 有 http 路由跡象      |

DetectResult 結構:
```go
type DetectResult struct {
    Type       string   // Preset 類型常數
    Confidence int      // 信心分數 (40-100)
    Reasons    []string // 偵測依據說明
    Runtime    string   // 運行環境 (nodejs|bun|python|go|custom)
    Port       int      // 預設埠號
}
```

CONVENTIONS:
- 信心分數: Laravel/Go 50+, 其他框架 40+
- 優先度順序: Laravel(1) > Next(2) > Nuxt(3) > Astro(4) > Vite(5) > PocketBase(6) > Python(7) > GoAPI(8)
- 檔案檢查順序: 按優先度降序，符合即回傳
- Runtime 識別: 由 main 檔案內容或 package.json scripts 推斷

ANTI-PATTERNS:
- 禁止直接讀取整個檔案內容做比對，優先檢查檔案存在性
- 禁止回傳未定義的 Preset Type
- 避免低信心分數 (<40) 的猜測式偵測
- 禁止修改工作目錄或執行外部指令
