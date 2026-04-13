# 新增自定義開發環境運行

## 已發佈版本: 1.2.0
- 新增 Runtime 開發環境運行 (從只支援 Node.js 加上更多 Runtime)
- Node.js 改為 Runtime, Node.js Port 改為 Runtime Port, Node.js Projects 改為 Runtime
- Node Version 改為 Runtime, 同時選項改為 Auto, Node.js, Bun, Python, Go Air, Go Run, Custom
### Dependencies
- Bun 1.3.11

**Runtime取代Node**
設計可支援使用不同 Runtime 運行開發環境 (如 Bun, Python, Go Air/Run, 或其他自定義 Command), 同時可自定端口傳參 (而非現在只支援 Node.js)

## 新增 Bun 依賴
位置: `\wincmp\bin\bun\bun-1.3.11\bun.exe`
掃描器會自動掃描 `bin/bun/` 目錄，拿取最新的版本運作

## 其他依賴
Python 和 Go (Air/Run) 不放在 bin 內, 需要用戶自行加到 Windows 環境變數
用戶在嘗試在 Runtime 啟動 Python 和 Go (Air) 時, 需要先用 version 測試一下 (如 `python -V` 和 `go version`), 如果發現用戶沒有安裝或加入 Windows 環境變數需要跳窗 + logs 提醒, 如已安裝就在 logs 記錄 version 拿到的版本號, 方便有問題時偵錯

## Web Projects (Preset 系統)
在 Web Projects -> Edit Project 視窗 -> Availability & Type 下
1. Project Type 由現在的只支援 None (Static), Laravel, Node.js 改為支援 13 種 Preset 類型:
   - Static Site, Laravel, Next.js, Nuxt, Astro, Vite App
   - Python App, Python Django, Python FastAPI, Python Flask
   - Go API, PocketBase, Custom
2. 新增 Use WinCMP Bin (輸入方式為 checkbox), 只在當 Project Type 是 Node.js/Bun 類型時顯示。Python, Go 和 Custom 類型強制使用系統 PATH (不顯示此選項)。顯示時需附上小斜字說明, 此欄位影響 wincmp.json 中 projects 下的 use_wincmp_bin 參數

## Custom 選項的 Wincmp.json 佔位符
WinCMP 一套「官方支援」占位符

- %PORT% → 目前專案 Runtime 端口
- %HOST% → 預設 0.0.0.0 或使用者自訂
- %PROJECT_DIR% → 專案根目錄
- %BIN_DIR% → wincmp/bin 路徑

例如使用者可以寫：

- `deno task dev --port %PORT% --host %HOST%`
- `python app.py --port %PORT%`
- `my-custom-dev.exe --root "%PROJECT_DIR%" --port %PORT%`

> WinCMP 只要在 Go 裡做簡單 strings.ReplaceAll(cmd, "%PORT%", port) 這種，就能達到你要的效果。
> **⚠️ 陷阱提醒 (關於 Windows `cmd.exe` 引號解析)：**
>
> 由於 WinCMP 背後是使用 `cmd.exe /c` 執行命令，若你在設定指令時使用引號包圍參數（例如 `go run ... --http="0.0.0.0:%PORT%"`），**這些雙引號在某些情況會被原封不動傳給你的應用程式**（例如變成 `"0.0.0.0:8090"`），導致像 Go 語言等程式在解析端口時出錯 (`unknown port ..."`\)。
> - **IP / Port** 等不含空格的參數，**絕對不要加雙引號**，請直接寫：`--http=0.0.0.0:%PORT%`。
> - 只有在**路徑含有空格**時（如 `%PROJECT_DIR%`），才使用雙引號包圍。

## Terminal Logs
Node.log 改為 Runtime.log, 資源監控中的 Node (project_name) 改為 Runtime (project_name)
node.log 日誌記錄檔名改為 runtime.log
Node 分頁改為 Runtime

## Runtime 類型
1. **Auto**（自動選擇：有 Bun 用 Bun，否則用 Node.js）
2. **Node.js**（使用 node / npm 執行）
3. **Bun**（使用 bun 執行，npm 相容）
4. **Python**（使用 python，支援 Django / FastAPI / Flask / Generic HTTP Server）
5. **Go Air**（使用 air 進行熱重載）
6. **Go Run**（使用 go run 直接執行，適用 PocketBase 等）
7. **Custom**（自訂指令，如 Deno, Rust 等）

```
1. Nuxt (3 & 4) 常用開發指令
Nuxt 使用 nuxi 作為 CLI 工具，參數通常以 -- 開頭。
環境	基本啟動	指定 Host & Port
npm	npm run dev	npm run dev -- --host 0.0.0.0 --port 8080
bun	bun run dev	bun run dev --host 0.0.0.0 --port 8080
bun (直接調用 nuxi)	bun nuxi dev	bun nuxi dev --host 0.0.0.0 --port 8080
注意： npm 在傳遞參數時，必須先加一個 --（例如 npm run dev -- [參數]），否則參數會被 npm 自己吃掉，而不會傳給 Nuxt。Bun 則不需要，可以直接接在後面。
2. Next.js 常用開發指令
Next.js 的 CLI 參數稍有不同，Host 通常使用 -H，Port 使用 -p。
環境	基本啟動	指定 Host & Port
npm	npm run dev	npm run dev -- -H 0.0.0.0 -p 8080
bun	bun run dev	bun run dev -H 0.0.0.0 -p 8080
bun (直接調用 next)	bun next dev	bun next dev -H 0.0.0.0 -p 8080
```

## Preset 預設指令模板
| Project Type | Runtime | Default Port | Command Template |
|---|---|---|---|
| Next.js | Bun | 3000 | `bun run dev -- --port %PORT%` |
| Next.js | Node.js | 3000 | `npm run dev -- --port %PORT%` |
| Nuxt | Bun | 3000 | `bun run dev -- --port %PORT% --host 0.0.0.0` |
| Nuxt | Node.js | 3000 | `npm run dev -- --port %PORT% --host 0.0.0.0` |
| Astro | Bun | 4321 | `bun run dev -- --host 0.0.0.0 --port %PORT%` |
| Astro | Node.js | 4321 | `npm run dev -- --host 0.0.0.0 --port %PORT%` |
| Vite App | Bun | 5173 | `bun run dev -- --host 0.0.0.0 --port %PORT%` |
| Vite App | Node.js | 5173 | `npm run dev -- --host 0.0.0.0 --port %PORT%` |
| Python App | Python | 8000 | `python -m http.server %PORT% --bind 0.0.0.0` |
| Python Django | Python | 8000 | `python manage.py runserver 0.0.0.0:%PORT%` |
| Python FastAPI | Python | 8000 | `python -m uvicorn main:app --host 0.0.0.0 --port %PORT% --reload` |
| Python Flask | Python | 5000 | `python -m flask run --host=0.0.0.0 --port=%PORT%` |
| Go API | Go Air | 8080 | `air` |
| Go API | Go Run | 8080 | `go run main.go` |
| PocketBase | Go Run | 8090 | `go run main.go serve --http=0.0.0.0:%PORT%` |

## wincmp.json
- projects 內新增 `command`, 例如 `"command": "bun run dev -- --host %HOST% --port %PORT%"` 要根據 port 來決定 port 位, host 預設開放 0.0.0.0
- projects 內新增 `use_wincmp_bin`
- projects 內新增 `runtime_mode` (Background / Terminal)
- projects 內新增 `runtime_version` (如 "24.14.1")
- projects 內新增 `command_dirty` (使用者是否手動修改過 Command)

```
"type" = "next" 或 "laravel" 或 "python_django" 等
"runtime_type": "node" 或 "bun" 或 "python" 或 "go_air" 或 "go_run" 或 "custom" 或 "auto"
"runtime_port": 3005 (原 node_port 已遷移, 若發現舊值會自動轉換)
"runtime_mode": "Background" 或 "Terminal"
"command": "bun run dev -- --host %HOST% --port %PORT%"
"use_wincmp_bin": true
```

如果 Project Type 是 None (Static) 那 type 和 runtime_port 鍵會消失，runtime_type 會設為 "none"

**載入時自動遷移：檢查一次內容是否有 node_port, 如有就把 node_port 轉為 runtime_port, use_env_bin 轉為 use_wincmp_bin**

### 運行方式
1. Terminal 運行方式
```
cmd := exec.Command("cmd.exe", "/c", "start", "WinCMP Runtime: "+project.Name, "cmd.exe", "/k", innerCmd)
```
2. Background 運行方式
```
cmd := exec.Command("cmd.exe", "/c", "chcp 65001 >nul && "+runtimeCmd)
```
上述已改為根據 wincmp.json 的 command 和 Preset 模板來執行不同 Runtime 運行指令, node_port 已改為 RuntimePort

### 舊版專案自動遷移
- `node_port` → `runtime_port`
- `node_mode` → `runtime_mode`
- `node_version` → `runtime_version`
- `use_env_bin` → `use_wincmp_bin`
- 舊版 type `"go"` → `"go_api"`
- 舊版 type `"node"` / `"bun"` → `"vite"` (fallback)
- 舊版 runtime_type `"go"` → `"go_air"`
