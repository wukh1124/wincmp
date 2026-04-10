# 新增自定義開發環境運行

**Runtime取代Node**
設計可支援使用不用Runtime運行開發環境(如Bun, Python, Go Air, 或其他自定義Command),同時可自定端口傳參 (而非現在只支援Node.js)

## 新增bun依賴
位置: \wincmp\bin\bun\bun-1.3.11\bun.exe
需掃瞄bun, 只拿取最新的版本運作

## 其他依賴
Python 和 Go (Air)不放在bin內, 需要用戶自行加到Windows環境變數
用戶在嘗試在Runtime啟動Python和Go(Air)時, 需要先用version測試一下(如python -V 和 go version), 如果發現用戶沒有安裝或加入Windows環境變數需要跳窗+logs提醒, 如已安裝就在logs記錄version拿到的版本號, 方便有問題時偵錯

## Web Projects
在Web Projects -> Edit Project 視窗 -> Avaliability & Type 下
1. Project Type 由現在的只支援None(Static), Laravel, Node.js 現在需要新增: 加上 Bun, Python, Go (Air), Custom
2. 新增 Use Env Bin (輸入方式為checkbox), 只在當Project Type是Node.js, Bun, Python, Go (Air), Custom顯示, 同時 Python, Go(Air) 和 Custom 強制勾上(變灰色用戶無法修改).顯示時需附上小斜字說明, 此欄位 影響 wincmp.json中projects下的use_env_bin參數

## Terminal Logs
Node.log改為Runtime.log, 資源監控中的Node (project_name)改為 Runtime (project_name)
node.log日誌記錄檔名改為runtime.log
Node分頁改為Runtime

## Runtime 類型
1. Node.js（使用 node / npm 執行）
2. Bun（使用 bun 執行，npm 相容）
3. Python（使用 python / python -m http.server）
4. Go (Air)（使用 air 進行熱重載）
5. Custom（自訂指令，如 Deno, Rust 等）

## Command
1. npm run dev -- -p 3001
2. bun run dev -- -p 3001 -H 0.0.0.0
3. python -m http.server 8080 --bind 0.0.0.0
4. air -- serve --http="0.0.0.0:8090"
5. 自訂指令

## wincmp.json
- projects內 新增 command, 例如 "command": "bun run dev -- -p 3001 -H 0.0.0.0" 要根據 port 來決定port位, host預設開放0.0.0.0
- projects內 新增 use_env_bin 


"type" = "node" 或 "laravel"
"node_port": 3005 需要為 "runtime_port": 3005
如果Project Type 是 None (Static) 那 type 和 node_port(或runtime_port) 鍵會消失

**先檢查一次內容是否有node_port, 如有就把node_port轉為runtime_port**

### 運行方式
1. 目前Terminal運行方式
```
innerCmd := "chcp 65001 >nul && " + exePath + " run dev -- -p " + strconv.Itoa(project.NodePort)
		cmd := exec.Command("cmd.exe", "/c", "start", "WinCMP Node: "+project.Name, "cmd.exe", "/k", innerCmd)
```
2. 目前Backgroud運行方式
```
cmd := exec.Command("cmd.exe", "/c", "chcp 65001 >nul && "+exePath+" run dev -- -p "+strconv.Itoa(project.NodePort))
```
上述需要改為可根據wincmp.json的command來執行不用Runtime運行指令, node_port可改為RuntimePort