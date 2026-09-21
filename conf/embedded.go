package conf

import "embed"

// DependenciesJSON 儲存嵌入的 dependencies.json 原始位元組資料
//go:embed dependencies.json
var DependenciesJSON []byte

// DefaultConfFS 嵌入 conf 目錄下的官方預設設定檔與子目錄
//go:embed Caddyfile my.ini wincmp.json dependencies.json php/* snippets/* sites/.gitkeep ssl/.gitkeep
var DefaultConfFS embed.FS
