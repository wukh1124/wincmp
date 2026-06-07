package preset

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProjectType 定義專案類型的常數
const (
	TypeStatic        = "static"
	TypeLaravel       = "laravel"
	TypeNext          = "next"
	TypeNuxt          = "nuxt"
	TypeAstro         = "astro"
	TypeVite          = "vite"
	TypePython        = "python"
	TypePythonDjango  = "python_django"
	TypePythonFastAPI = "python_fastapi"
	TypePythonFlask   = "python_flask"
	TypeGoAPI         = "go_api"
	TypePocketBase    = "pocketbase"
	TypeCustom        = "custom"
	TypePHP           = "php"
)

// RuntimeType 定義執行器類型的常數
const (
	RuntimeAuto   = "auto"
	RuntimeNone   = "none"
	RuntimeNode   = "node"
	RuntimeBun    = "bun"
	RuntimePython = "python"
	RuntimeGoAir  = "go_air"
	RuntimeGoRun  = "go_run"
	RuntimeCustom = "custom"
)

// Preset 定義單一專案類型的完整預設配置
type Preset struct {
	ID                    string
	Label                 string
	DefaultPort           int
	DefaultRuntime        string
	RuntimeOptions        []string
	CommandTemplates      map[string]string // key = runtime type
	SupportsBundled       bool
	IsRuntimeProject      bool
	DetectPriority        int  // 數字越小優先度越高
	CommandDirtyProtected bool // 是否保護使用者手動修改的命令
}

// presets 是集中管理的預設配置表
var presets = map[string]Preset{
	TypeStatic: {
		ID:               TypeStatic,
		Label:            "Static Site",
		DefaultPort:      0,
		DefaultRuntime:   RuntimeNone,
		RuntimeOptions:   []string{RuntimeNone},
		IsRuntimeProject: false,
		DetectPriority:   100,
	},
	TypePHP: {
		ID:               TypePHP,
		Label:            "PHP Site",
		DefaultPort:      0,
		DefaultRuntime:   RuntimeNone,
		RuntimeOptions:   []string{RuntimeNone},
		IsRuntimeProject: false,
		DetectPriority:   90,
	},
	TypeLaravel: {
		ID:               TypeLaravel,
		Label:            "Laravel",
		DefaultPort:      0,
		DefaultRuntime:   RuntimeNone,
		RuntimeOptions:   []string{RuntimeNone},
		IsRuntimeProject: false,
		DetectPriority:   1,
	},
	TypeNext: {
		ID:             TypeNext,
		Label:          "Next.js",
		DefaultPort:    3000,
		DefaultRuntime: RuntimeAuto,
		RuntimeOptions: []string{RuntimeAuto, RuntimeBun, RuntimeNode, RuntimeCustom},
		CommandTemplates: map[string]string{
			RuntimeBun:  "bun run dev -- --port %PORT%",
			RuntimeNode: "npm run dev -- --port %PORT%",
		},
		SupportsBundled:  true,
		IsRuntimeProject: true,
		DetectPriority:   2,
	},
	TypeNuxt: {
		ID:             TypeNuxt,
		Label:          "Nuxt",
		DefaultPort:    3000,
		DefaultRuntime: RuntimeAuto,
		RuntimeOptions: []string{RuntimeAuto, RuntimeBun, RuntimeNode, RuntimeCustom},
		CommandTemplates: map[string]string{
			RuntimeBun:  "bun run dev -- --port %PORT% --host 0.0.0.0",
			RuntimeNode: "npm run dev -- --port %PORT% --host 0.0.0.0",
		},
		SupportsBundled:  true,
		IsRuntimeProject: true,
		DetectPriority:   3,
	},
	TypeAstro: {
		ID:             TypeAstro,
		Label:          "Astro",
		DefaultPort:    4321,
		DefaultRuntime: RuntimeAuto,
		RuntimeOptions: []string{RuntimeAuto, RuntimeBun, RuntimeNode, RuntimeCustom},
		CommandTemplates: map[string]string{
			RuntimeBun:  "bun run dev -- --host 0.0.0.0 --port %PORT%",
			RuntimeNode: "npm run dev -- --host 0.0.0.0 --port %PORT%",
		},
		SupportsBundled:  true,
		IsRuntimeProject: true,
		DetectPriority:   4,
	},
	TypeVite: {
		ID:             TypeVite,
		Label:          "Vite App",
		DefaultPort:    5173,
		DefaultRuntime: RuntimeAuto,
		RuntimeOptions: []string{RuntimeAuto, RuntimeBun, RuntimeNode, RuntimeCustom},
		CommandTemplates: map[string]string{
			RuntimeBun:  "bun run dev -- --host 0.0.0.0 --port %PORT%",
			RuntimeNode: "npm run dev -- --host 0.0.0.0 --port %PORT%",
		},
		SupportsBundled:  true,
		IsRuntimeProject: true,
		DetectPriority:   5,
	},
	TypePython: {
		ID:             TypePython,
		Label:          "Python App",
		DefaultPort:    8000,
		DefaultRuntime: RuntimePython,
		RuntimeOptions: []string{RuntimePython, RuntimeCustom},
		CommandTemplates: map[string]string{
			RuntimePython: "python -m http.server %PORT% --bind 0.0.0.0",
		},
		IsRuntimeProject: true,
		DetectPriority:   7,
	},
	TypePythonDjango: {
		ID:             TypePythonDjango,
		Label:          "Python Django",
		DefaultPort:    8000,
		DefaultRuntime: RuntimePython,
		RuntimeOptions: []string{RuntimePython, RuntimeCustom},
		CommandTemplates: map[string]string{
			RuntimePython: "python manage.py runserver 0.0.0.0:%PORT%",
		},
		IsRuntimeProject: true,
		DetectPriority:   7,
	},
	TypePythonFastAPI: {
		ID:             TypePythonFastAPI,
		Label:          "Python FastAPI",
		DefaultPort:    8000,
		DefaultRuntime: RuntimePython,
		RuntimeOptions: []string{RuntimePython, RuntimeCustom},
		CommandTemplates: map[string]string{
			RuntimePython: "python -m uvicorn main:app --host 0.0.0.0 --port %PORT% --reload",
		},
		IsRuntimeProject: true,
		DetectPriority:   7,
	},
	TypePythonFlask: {
		ID:             TypePythonFlask,
		Label:          "Python Flask",
		DefaultPort:    5000,
		DefaultRuntime: RuntimePython,
		RuntimeOptions: []string{RuntimePython, RuntimeCustom},
		CommandTemplates: map[string]string{
			RuntimePython: "python -m flask run --host=0.0.0.0 --port=%PORT%",
		},
		IsRuntimeProject: true,
		DetectPriority:   7,
	},
	TypeGoAPI: {
		ID:             TypeGoAPI,
		Label:          "Go API",
		DefaultPort:    8080,
		DefaultRuntime: RuntimeGoAir,
		RuntimeOptions: []string{RuntimeGoAir, RuntimeGoRun, RuntimeCustom},
		CommandTemplates: map[string]string{
			RuntimeGoAir: "air",
			RuntimeGoRun: "go run main.go",
		},
		IsRuntimeProject: true,
		DetectPriority:   8,
	},
	TypePocketBase: {
		ID:             TypePocketBase,
		Label:          "PocketBase",
		DefaultPort:    8090,
		DefaultRuntime: RuntimeGoRun,
		RuntimeOptions: []string{RuntimeGoRun, RuntimeCustom},
		CommandTemplates: map[string]string{
			RuntimeGoRun: "go run main.go serve --http=0.0.0.0:%PORT%",
		},
		IsRuntimeProject: true,
		DetectPriority:   6,
	},
	TypeCustom: {
		ID:               TypeCustom,
		Label:            "Custom",
		DefaultPort:      3000,
		DefaultRuntime:   RuntimeCustom,
		RuntimeOptions:   []string{RuntimeCustom},
		CommandTemplates: map[string]string{},
		IsRuntimeProject: true,
		DetectPriority:   999,
	},
}

// GetPreset 取得指定類型的 Preset
func GetPreset(typeID string) Preset {
	if p, ok := presets[typeID]; ok {
		return p
	}
	return presets[TypeStatic]
}

// GetAllPresets 按優先順序回傳所有 Preset
func GetAllPresets() []Preset {
	result := make([]Preset, 0, len(presets))
	for _, p := range presets {
		result = append(result, p)
	}
	sortByPriority(result)
	return result
}

// GetPresetLabels 取得所有 Preset ID → Label 的對應（用於 UI 下拉選單）
func GetPresetLabels() []string {
	sorted := GetAllPresets()
	labels := make([]string, 0, len(sorted))
	for _, p := range sorted {
		labels = append(labels, p.Label)
	}
	return labels
}

// GetPresetIDByLabel 根據 Label 反查 Preset ID
func GetPresetIDByLabel(label string) string {
	for id, p := range presets {
		if p.Label == label {
			return id
		}
	}
	return TypeStatic
}

// IsPythonType 判斷是否為 Python 類型（含子類型）
func IsPythonType(typeID string) bool {
	return typeID == TypePython || typeID == TypePythonDjango || typeID == TypePythonFastAPI || typeID == TypePythonFlask
}

// IsRuntimeProject 判斷專案類型是否為 Runtime 類型（需要顯示在 Runtime Tab）
func IsRuntimeProject(typeID string) bool {
	p := GetPreset(typeID)
	return p.IsRuntimeProject
}

// ResolveRuntime 將 "auto" 解析為實際的 Runtime
// 檢查 bin/ 目錄是否有 bun，優先回傳 bun，否則回傳 node
func ResolveRuntime(typeID string, hasBun bool) string {
	p := GetPreset(typeID)
	switch p.DefaultRuntime {
	case RuntimeAuto:
		if hasBun {
			return RuntimeBun
		}
		return RuntimeNode
	default:
		return p.DefaultRuntime
	}
}

// ResolveRuntimeFromProject 根據專案設定解析實際 Runtime
func ResolveRuntimeFromProject(typeID, runtimeType string, hasBun bool) string {
	if runtimeType == RuntimeAuto {
		return ResolveRuntime(typeID, hasBun)
	}
	if runtimeType == RuntimeNone || runtimeType == "" {
		return RuntimeNone
	}
	return runtimeType
}

// BuildStartCommand 根據 Preset、Runtime 與 Port 組合產生啟動指令
func BuildStartCommand(typeID, runtimeType string, port int, customCmd string) string {
	p := GetPreset(typeID)

	// Custom 類型優先使用自訂指令
	if typeID == TypeCustom || runtimeType == RuntimeCustom {
		if customCmd != "" {
			return replacePlaceholders(customCmd, port, "0.0.0.0", "", "")
		}
		return ""
	}

	// Python 類型：根據子類型使用 CommandTemplate
	if IsPythonType(typeID) {
		tmpl, ok := p.CommandTemplates[RuntimePython]
		if !ok {
			tmpl = "python -m http.server %PORT% --bind 0.0.0.0"
		}
		if port == 0 {
			port = p.DefaultPort
			if port == 0 {
				port = 8000
			}
		}
		return replacePlaceholders(tmpl, port, "0.0.0.0", "", "")
	}

	// Go 類型
	if typeID == TypeGoAPI {
		return buildGoCommand(runtimeType, port)
	}
	if typeID == TypePocketBase {
		return buildPocketBaseCommand(runtimeType, port)
	}

	// Node/Bun 類型 (Next/Nuxt/Astro/Vite)
	resolved := runtimeType
	if resolved == RuntimeAuto {
		resolved = RuntimeNode // 這裡先回傳 node 版本，實際 resolve 由呼叫端處理
	}

	tmpl, ok := p.CommandTemplates[resolved]
	if !ok {
		tmpl, ok = p.CommandTemplates[RuntimeNode]
	}
	if !ok {
		return ""
	}

	if port == 0 {
		port = p.DefaultPort
		if port == 0 {
			port = 3000
		}
	}

	return replacePlaceholders(tmpl, port, "0.0.0.0", "", "")
}

// BuildStartCommandWithExePath 根據 Preset、Runtime 組合產生帶 exePath 前綴的啟動指令
// 用於實際 Runtime 啟動流程
func BuildStartCommandWithExePath(typeID, runtimeType string, port int, customCmd string, exePath string) string {
	p := GetPreset(typeID)

	// Custom 或 Python/Go 類型使用 BuildStartCommand
	if typeID == TypeCustom || runtimeType == RuntimeCustom {
		cmd := BuildStartCommand(typeID, runtimeType, port, customCmd)
		return cmd
	}

	if IsPythonType(typeID) || typeID == TypeGoAPI || typeID == TypePocketBase {
		return BuildStartCommand(typeID, runtimeType, port, customCmd)
	}

	// Node/Bun 類型：使用 exePath 取代 npm/bun 前綴
	resolved := runtimeType
	if resolved == RuntimeAuto {
		// 根據 exePath 判斷是 npm 還是 bun
		if strings.Contains(strings.ToLower(exePath), "bun") {
			resolved = RuntimeBun
		} else {
			resolved = RuntimeNode
		}
	}

	tmpl, ok := p.CommandTemplates[resolved]
	if !ok {
		tmpl, ok = p.CommandTemplates[RuntimeNode]
	}
	if !ok {
		return ""
	}

	if port == 0 {
		port = p.DefaultPort
		if port == 0 {
			port = 3000
		}
	}

	// 將 exePath 替換進模板中的 npm/bun 指令
	cmdStr := replacePlaceholders(tmpl, port, "0.0.0.0", "", "")

	// 替換指令前綴為完整 exePath
	isNPM := strings.Contains(strings.ToLower(exePath), "npm")
	if isNPM {
		cmdStr = exePath + cmdStr[strings.Index(cmdStr, " run"):]
	} else {
		cmdStr = exePath + cmdStr[strings.Index(cmdStr, " run"):]
	}

	return cmdStr
}

// buildPythonCommand 根據專案結構偵測 Python 框架並回傳啟動指令
func buildPythonCommand(port int) string {
	return fmt.Sprintf("python -m http.server %d --bind 0.0.0.0", port)
}

// BuildPythonCommandFromRoot 根據專案目錄偵測 Python 框架並回傳適合的啟動指令
func BuildPythonCommandFromRoot(rootDir string, port int) string {
	if port == 0 {
		port = 8000
	}

	// 偵測 Django
	if exists(filepath.Join(rootDir, "manage.py")) {
		return fmt.Sprintf("python manage.py runserver 0.0.0.0:%d", port)
	}

	// 偵測 FastAPI
	if hasDependency(filepath.Join(rootDir, "requirements.txt"), "fastapi") ||
		hasDependency(filepath.Join(rootDir, "pyproject.toml"), "fastapi") {
		// 嘗試找主要進入點
		for _, candidate := range []string{"main.py", "app.py", "server.py"} {
			if exists(filepath.Join(rootDir, candidate)) {
				moduleName := strings.TrimSuffix(candidate, ".py")
				return fmt.Sprintf("python -m uvicorn %s:app --host 0.0.0.0 --port %d --reload", moduleName, port)
			}
		}
		return fmt.Sprintf("python -m uvicorn main:app --host 0.0.0.0 --port %d --reload", port)
	}

	// 偵測 Flask
	if hasDependency(filepath.Join(rootDir, "requirements.txt"), "flask") ||
		hasDependency(filepath.Join(rootDir, "pyproject.toml"), "flask") {
		return fmt.Sprintf("python -m flask run --host=0.0.0.0 --port=%d", port)
	}

	// Fallback: 簡易 HTTP 伺服器
	return fmt.Sprintf("python -m http.server %d --bind 0.0.0.0", port)
}

// buildGoCommand 根據 runtime 類型回傳 Go API 啟動指令
func buildGoCommand(runtimeType string, port int) string {
	if port == 0 {
		port = 8080
	}
	switch runtimeType {
	case RuntimeGoRun:
		return "go run main.go"
	case RuntimeGoAir:
		return "air"
	default:
		return "air"
	}
}

// buildPocketBaseCommand 根據 runtime 回傳 PocketBase 啟動指令
func buildPocketBaseCommand(runtimeType string, port int) string {
	if port == 0 {
		port = 8090
	}
	switch runtimeType {
	case RuntimeGoRun:
		return fmt.Sprintf("go run main.go serve --http=0.0.0.0:%d", port)
	default:
		return fmt.Sprintf("go run main.go serve --http=0.0.0.0:%d", port)
	}
}

// replacePlaceholders 替換佔位符
func replacePlaceholders(cmd string, port int, host, projectDir, binDir string) string {
	cmd = strings.ReplaceAll(cmd, "%PORT%", fmt.Sprintf("%d", port))
	cmd = strings.ReplaceAll(cmd, "%HOST%", host)
	cmd = strings.ReplaceAll(cmd, "%PROJECT_DIR%", projectDir)
	cmd = strings.ReplaceAll(cmd, "%BIN_DIR%", binDir)
	return cmd
}

// GetProjectTypeLabel 取得專案類型的顯示名稱
func GetProjectTypeLabel(typeID string) string {
	p := GetPreset(typeID)
	return p.Label
}

// GetRuntimeLabel 取得 Runtime 類型的顯示名稱
func GetRuntimeLabel(runtime string) string {
	switch runtime {
	case RuntimeAuto:
		return "Auto"
	case RuntimeNone:
		return "None"
	case RuntimeNode:
		return "Node.js"
	case RuntimeBun:
		return "Bun"
	case RuntimePython:
		return "Python"
	case RuntimeGoAir:
		return "Go Air"
	case RuntimeGoRun:
		return "Go Run"
	case RuntimeCustom:
		return "Custom"
	default:
		return runtime
	}
}

// GetRuntimeLabelByID 根據 Runtime ID 回傳 Label（用於 UI 下拉選單）
func GetRuntimeIDByLabel(label string) string {
	switch label {
	case "Auto":
		return RuntimeAuto
	case "None":
		return RuntimeNone
	case "Node.js":
		return RuntimeNode
	case "Bun":
		return RuntimeBun
	case "Python":
		return RuntimePython
	case "Go Air":
		return RuntimeGoAir
	case "Go Run":
		return RuntimeGoRun
	case "Custom":
		return RuntimeCustom
	default:
		return label
	}
}

// GetRuntimeLabelsForType 取得指定專案類型可用的 Runtime Label 清單
func GetRuntimeLabelsForType(typeID string) []string {
	p := GetPreset(typeID)
	labels := make([]string, 0, len(p.RuntimeOptions))
	for _, rt := range p.RuntimeOptions {
		labels = append(labels, GetRuntimeLabel(rt))
	}
	return labels
}

// GetDefaultPort 取得指定類型的預設 Port
func GetDefaultPort(typeID string) int {
	return GetPreset(typeID).DefaultPort
}

// GetFullTypeLabel 取得 "Framework (Runtime)" 格式的顯示標籤
func GetFullTypeLabel(typeID, runtimeType string, hasBun bool) string {
	p := GetPreset(typeID)
	if !p.IsRuntimeProject {
		return p.Label
	}

	resolved := ResolveRuntimeFromProject(typeID, runtimeType, hasBun)
	if resolved == RuntimeNone || resolved == "" {
		return p.Label
	}

	// 對於 Python 類型，不顯示 (Python)，因為 Python 只會用 Python 來 Runtime
	if resolved == RuntimePython {
		return p.Label
	}

	return fmt.Sprintf("%s (%s)", p.Label, GetRuntimeLabel(resolved))
}

// NormalizeProjectType 標準化專案類型 ID（處理舊值映射）
func NormalizeProjectType(typeID string) string {
	switch typeID {
	case "go":
		return TypeGoAPI
	case "node", "bun":
		return TypeVite // 舊的 node/bun fallback 到 Vite
	default:
		return typeID
	}
}

// NormalizeRuntimeType 標準化 Runtime 類型 ID（處理舊值映射）
func NormalizeRuntimeType(runtimeType string) string {
	switch runtimeType {
	case "go":
		return RuntimeGoAir
	default:
		return runtimeType
	}
}

// DetectResult 偵測結果
type DetectResult struct {
	Type       string
	Confidence int
	Reasons    []string
	Runtime    string // 建議的 Runtime
	Port       int    // 建議的 Port
}

// DetectProjectPreset 從專案目錄偵測適合的 Preset
func DetectProjectPreset(root string) DetectResult {
	results := []DetectResult{}

	// 1. Laravel
	if res := detectLaravelPreset(root); res.Confidence >= 50 {
		results = append(results, res)
	}

	// 2. Next.js
	if res := detectNextPreset(root); res.Confidence >= 40 {
		results = append(results, res)
	}

	// 3. Nuxt
	if res := detectNuxtPreset(root); res.Confidence >= 40 {
		results = append(results, res)
	}

	// 4. Astro
	if res := detectAstroPreset(root); res.Confidence >= 40 {
		results = append(results, res)
	}

	// 5. Vite
	if res := detectVitePreset(root); res.Confidence >= 40 {
		results = append(results, res)
	}

	// 6. PocketBase
	if res := detectPocketBasePreset(root); res.Confidence >= 40 {
		results = append(results, res)
	}

	// 7. Go API
	if res := detectGoAPIPreset(root); res.Confidence >= 50 {
		results = append(results, res)
	}

	// 8. Python (Django/FastAPI/Flask/Generic)
	if res := detectPythonPreset(root); res.Confidence >= 40 {
		results = append(results, res)
	}

	// 根據優先度排序，回傳最高信賴度的結果
	if len(results) == 0 {
		return DetectResult{
			Type:       TypeStatic,
			Confidence: 100,
			Reasons:    []string{"no runtime detected"},
			Runtime:    RuntimeNone,
			Port:       0,
		}
	}

	// 先按 Preset 優先度排序，同優先度按 Confidence 排序
	best := results[0]
	for _, r := range results[1:] {
		bestPriority := GetPreset(r.Type).DetectPriority
		currentPriority := GetPreset(best.Type).DetectPriority
		if bestPriority < currentPriority || (bestPriority == currentPriority && r.Confidence > best.Confidence) {
			best = r
		}
	}

	return best
}

// detectLaravelPreset 偵測 Laravel
func detectLaravelPreset(root string) DetectResult {
	score := 0
	reasons := []string{}

	checks := []struct {
		rel    string
		weight int
		reason string
	}{
		{"artisan", 30, "found artisan"},
		{"bootstrap/app.php", 25, "found bootstrap/app.php"},
		{"public/index.php", 20, "found public/index.php"},
		{"routes/web.php", 10, "found routes/web.php"},
		{"config/app.php", 10, "found config/app.php"},
	}

	for _, c := range checks {
		if exists(filepath.Join(root, c.rel)) {
			score += c.weight
			reasons = append(reasons, c.reason)
		}
	}

	// 檢查 composer.json 的 laravel 依賴
	pkgFile := filepath.Join(root, "composer.json")
	if b, err := os.ReadFile(pkgFile); err == nil {
		var obj struct {
			Require map[string]string `json:"require"`
		}
		if err := json.Unmarshal(b, &obj); err == nil {
			if _, ok := obj.Require["laravel/framework"]; ok {
				score += 40
				reasons = append(reasons, "composer.json requires laravel/framework")
			}
		}
	}

	return DetectResult{
		Type:       TypeLaravel,
		Confidence: score,
		Reasons:    reasons,
		Runtime:    RuntimeNone,
		Port:       0,
	}
}

// detectNextPreset 偵測 Next.js
func detectNextPreset(root string) DetectResult {
	pkgFile := filepath.Join(root, "package.json")
	score := 0
	reasons := []string{}

	if b, err := os.ReadFile(pkgFile); err == nil {
		var obj struct {
			Dependencies    map[string]string `json:"dependencies"`
			DevDependencies map[string]string `json:"devDependencies"`
		}
		if err := json.Unmarshal(b, &obj); err == nil {
			if _, ok := obj.Dependencies["next"]; ok {
				score += 80
				reasons = append(reasons, "found next in dependencies")
			}
		}
	}

	if exists(filepath.Join(root, "next.config.js")) || exists(filepath.Join(root, "next.config.mjs")) || exists(filepath.Join(root, "next.config.ts")) {
		score += 20
		reasons = append(reasons, "found next.config.*")
	}

	return DetectResult{
		Type:       TypeNext,
		Confidence: score,
		Reasons:    reasons,
		Runtime:    RuntimeAuto,
		Port:       3000,
	}
}

// detectNuxtPreset 偵測 Nuxt
func detectNuxtPreset(root string) DetectResult {
	pkgFile := filepath.Join(root, "package.json")
	score := 0
	reasons := []string{}

	if b, err := os.ReadFile(pkgFile); err == nil {
		var obj struct {
			Dependencies    map[string]string `json:"dependencies"`
			DevDependencies map[string]string `json:"devDependencies"`
		}
		if err := json.Unmarshal(b, &obj); err == nil {
			if _, ok := obj.Dependencies["nuxt"]; ok {
				score += 80
				reasons = append(reasons, "found nuxt in dependencies")
			}
		}
	}

	if exists(filepath.Join(root, "nuxt.config.ts")) || exists(filepath.Join(root, "nuxt.config.js")) {
		score += 20
		reasons = append(reasons, "found nuxt.config.*")
	}

	return DetectResult{
		Type:       TypeNuxt,
		Confidence: score,
		Reasons:    reasons,
		Runtime:    RuntimeAuto,
		Port:       3000,
	}
}

// detectAstroPreset 偵測 Astro
func detectAstroPreset(root string) DetectResult {
	pkgFile := filepath.Join(root, "package.json")
	score := 0
	reasons := []string{}

	if b, err := os.ReadFile(pkgFile); err == nil {
		var obj struct {
			Dependencies    map[string]string `json:"dependencies"`
			DevDependencies map[string]string `json:"devDependencies"`
		}
		if err := json.Unmarshal(b, &obj); err == nil {
			if _, ok := obj.Dependencies["astro"]; ok {
				score += 70
				reasons = append(reasons, "found astro in dependencies")
			}
			if _, ok := obj.DevDependencies["astro"]; ok {
				score += 70
				reasons = append(reasons, "found astro in devDependencies")
			}
		}
	}

	if exists(filepath.Join(root, "astro.config.mjs")) || exists(filepath.Join(root, "astro.config.ts")) {
		score += 20
		reasons = append(reasons, "found astro.config.*")
	}

	return DetectResult{
		Type:       TypeAstro,
		Confidence: score,
		Reasons:    reasons,
		Runtime:    RuntimeAuto,
		Port:       4321,
	}
}

// detectVitePreset 偵測 Vite (不含 Next/Nuxt/Astro)
func detectVitePreset(root string) DetectResult {
	pkgFile := filepath.Join(root, "package.json")
	score := 0
	reasons := []string{}

	if b, err := os.ReadFile(pkgFile); err == nil {
		var obj struct {
			Dependencies    map[string]string `json:"dependencies"`
			DevDependencies map[string]string `json:"devDependencies"`
		}
		if err := json.Unmarshal(b, &obj); err == nil {
			// 排除已經是 Next/Nuxt/Astro 的情況
			if _, ok := obj.Dependencies["next"]; ok {
				return DetectResult{Type: TypeVite, Confidence: 0}
			}
			if _, ok := obj.Dependencies["nuxt"]; ok {
				return DetectResult{Type: TypeVite, Confidence: 0}
			}
			if _, ok := obj.Dependencies["astro"]; ok {
				return DetectResult{Type: TypeVite, Confidence: 0}
			}
			if _, ok := obj.DevDependencies["astro"]; ok {
				return DetectResult{Type: TypeVite, Confidence: 0}
			}

			if _, ok := obj.DevDependencies["vite"]; ok {
				score += 60
				reasons = append(reasons, "found vite in devDependencies")
			}
		}
	}

	if exists(filepath.Join(root, "vite.config.ts")) || exists(filepath.Join(root, "vite.config.js")) {
		score += 30
		reasons = append(reasons, "found vite.config.*")
	}

	// 如果有 bun.lockb 表示偏好 Bun
	runtime := RuntimeAuto
	if exists(filepath.Join(root, "bun.lockb")) {
		runtime = RuntimeAuto // auto 會自動 resolve
	}

	return DetectResult{
		Type:       TypeVite,
		Confidence: score,
		Reasons:    reasons,
		Runtime:    runtime,
		Port:       5173,
	}
}

// detectPocketBasePreset 偵測 PocketBase
func detectPocketBasePreset(root string) DetectResult {
	score := 0
	reasons := []string{}

	goModFile := filepath.Join(root, "go.mod")
	if b, err := os.ReadFile(goModFile); err == nil {
		content := string(b)
		if strings.Contains(content, "pocketbase") {
			score += 80
			reasons = append(reasons, "found pocketbase in go.mod")
		}
	}

	mainGoFile := filepath.Join(root, "main.go")
	if b, err := os.ReadFile(mainGoFile); err == nil {
		content := string(b)
		if strings.Contains(content, "pocketbase") {
			score += 40
			reasons = append(reasons, "found pocketbase in main.go")
		}
	}

	return DetectResult{
		Type:       TypePocketBase,
		Confidence: score,
		Reasons:    reasons,
		Runtime:    RuntimeGoRun,
		Port:       8090,
	}
}

// detectGoAPIPreset 偵測 Go API (排除 PocketBase)
func detectGoAPIPreset(root string) DetectResult {
	score := 0
	reasons := []string{}

	goModFile := filepath.Join(root, "go.mod")
	if exists(goModFile) {
		score += 50
		reasons = append(reasons, "found go.mod")

		// 排除 PocketBase
		if b, err := os.ReadFile(goModFile); err == nil {
			if strings.Contains(string(b), "pocketbase") {
				return DetectResult{Type: TypeGoAPI, Confidence: 0}
			}
		}
	}

	if exists(filepath.Join(root, ".air.toml")) {
		score += 40
		reasons = append(reasons, "found .air.toml")
	}

	return DetectResult{
		Type:       TypeGoAPI,
		Confidence: score,
		Reasons:    reasons,
		Runtime:    RuntimeGoAir,
		Port:       8080,
	}
}

// detectPythonPreset 偵測 Python 專案，根據依賴區分 Django / FastAPI / Flask / Generic
func detectPythonPreset(root string) DetectResult {
	score := 0
	reasons := []string{}

	// 偵測 Django
	if exists(filepath.Join(root, "manage.py")) {
		score += 60
		reasons = append(reasons, "found manage.py")
	}
	if hasDependency(filepath.Join(root, "requirements.txt"), "django") ||
		hasDependency(filepath.Join(root, "pyproject.toml"), "django") {
		score += 40
		reasons = append(reasons, "found django in dependencies")
	}
	if score >= 60 {
		return DetectResult{
			Type:       TypePythonDjango,
			Confidence: score,
			Reasons:    reasons,
			Runtime:    RuntimePython,
			Port:       8000,
		}
	}

	// 偵測 FastAPI：先依賴檔案，再原始碼
	fastapiScore := 0
	fastapiReasons := []string{}
	if hasDependency(filepath.Join(root, "requirements.txt"), "fastapi") {
		fastapiScore += 80
		fastapiReasons = append(fastapiReasons, "found fastapi in requirements.txt")
	}
	if hasDependency(filepath.Join(root, "pyproject.toml"), "fastapi") {
		fastapiScore += 60
		fastapiReasons = append(fastapiReasons, "found fastapi in pyproject.toml")
	}
	if fastapiScore == 0 {
		if srcScore, srcReason := scanPythonSourceForFramework(root, "fastapi"); srcScore > 0 {
			fastapiScore = srcScore
			fastapiReasons = append(fastapiReasons, srcReason)
		}
	}
	if fastapiScore > 0 {
		return DetectResult{
			Type:       TypePythonFastAPI,
			Confidence: fastapiScore,
			Reasons:    fastapiReasons,
			Runtime:    RuntimePython,
			Port:       8000,
		}
	}

	// 偵測 Flask：先依賴檔案，再原始碼
	flaskScore := 0
	flaskReasons := []string{}
	if hasDependency(filepath.Join(root, "requirements.txt"), "flask") {
		flaskScore += 80
		flaskReasons = append(flaskReasons, "found flask in requirements.txt")
	}
	if hasDependency(filepath.Join(root, "pyproject.toml"), "flask") {
		flaskScore += 60
		flaskReasons = append(flaskReasons, "found flask in pyproject.toml")
	}
	if flaskScore == 0 {
		if srcScore, srcReason := scanPythonSourceForFramework(root, "flask"); srcScore > 0 {
			flaskScore = srcScore
			flaskReasons = append(flaskReasons, srcReason)
		}
	}
	if flaskScore > 0 {
		return DetectResult{
			Type:       TypePythonFlask,
			Confidence: flaskScore,
			Reasons:    flaskReasons,
			Runtime:    RuntimePython,
			Port:       5000,
		}
	}

	// Fallback: 通用 Python
	if exists(filepath.Join(root, "requirements.txt")) {
		score += 25
		reasons = append(reasons, "found requirements.txt")
	}
	if exists(filepath.Join(root, "pyproject.toml")) {
		score += 15
		reasons = append(reasons, "found pyproject.toml")
	}
	if exists(filepath.Join(root, "app.py")) {
		score += 10
		reasons = append(reasons, "found app.py")
	}

	if score == 0 {
		return DetectResult{
			Type:       TypePython,
			Confidence: 0,
			Reasons:    []string{},
			Runtime:    RuntimePython,
			Port:       8000,
		}
	}

	return DetectResult{
		Type:       TypePython,
		Confidence: score,
		Reasons:    reasons,
		Runtime:    RuntimePython,
		Port:       8000,
	}
}

// helper functions

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func hasDependency(depFile string, depName string) bool {
	b, err := os.ReadFile(depFile)
	if err != nil {
		return false
	}
	return strings.Contains(string(b), depName)
}

// scanPythonSourceForFramework 掃描 Python 原始碼檔案，檢查是否含有指定框架特徵
// 回傳 (score, reason)
func scanPythonSourceForFramework(root string, framework string) (int, string) {
	candidates := []string{"main.py", "app.py", "server.py", "application.py", "api.py", "views.py", "routes.py"}

	patterns := map[string][]string{
		"fastapi": {"from fastapi import", "import fastapi", "FastAPI()", "fastapi.FastAPI"},
		"flask":   {"from flask import", "import flask", "Flask(__name__)", "flask.Flask"},
	}

	ptns, ok := patterns[framework]
	if !ok {
		return 0, ""
	}

	for _, candidate := range candidates {
		filePath := filepath.Join(root, candidate)
		if !exists(filePath) {
			continue
		}
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		text := string(content)
		for _, pattern := range ptns {
			if strings.Contains(text, pattern) {
				return 60, "found " + framework + " in " + candidate
			}
		}
	}

	return 0, ""
}

func sortByPriority(presets []Preset) {
	for i := 0; i < len(presets); i++ {
		for j := i + 1; j < len(presets); j++ {
			if presets[i].DetectPriority > presets[j].DetectPriority {
				presets[i], presets[j] = presets[j], presets[i]
			}
		}
	}
}
