package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func DetectNode(root string) DetectResult {
	score := 0
	reasons := make([]string, 0)

	// 1. 檢查 package.json (絕對必要)
	pkgJson := filepath.Join(root, "package.json")
	if exists(pkgJson) {
		score += 40
		reasons = append(reasons, "found package.json")

		// 進一步檢查內容
		if b, err := os.ReadFile(pkgJson); err == nil {
			var obj struct {
				Dependencies    map[string]string `json:"dependencies"`
				DevDependencies map[string]string `json:"devDependencies"`
				Scripts         map[string]string `json:"scripts"`
			}
			if err := json.Unmarshal(b, &obj); err == nil {
				// 檢查 React
				if _, ok := obj.Dependencies["react"]; ok {
					score += 20
					reasons = append(reasons, "found react in dependencies")
				}
				// 檢查 Vue (順便)
				if _, ok := obj.Dependencies["vue"]; ok {
					score += 15
					reasons = append(reasons, "found vue in dependencies")
				}
				// 檢查 Next.js
				if _, ok := obj.Dependencies["next"]; ok {
					score += 20
					reasons = append(reasons, "found next.js in dependencies")
				}
				// 檢查 Vite
				if _, ok := obj.DevDependencies["vite"]; ok {
					score += 10
					reasons = append(reasons, "found vite in devDependencies")
				}
			}
		}
	}

	// 2. 檢查 node_modules
	if exists(filepath.Join(root, "node_modules")) {
		score += 10
		reasons = append(reasons, "found node_modules")
	}

	// 3. 檢查常見的設定檔
	checks := []struct {
		rel    string
		weight int
		reason string
	}{
		{"tsconfig.json", 5, "found tsconfig.json"},
		{"vite.config.ts", 5, "found vite.config.ts"},
		{"vite.config.js", 5, "found vite.config.js"},
		{"next.config.js", 5, "found next.config.js"},
		{"webpack.config.js", 5, "found webpack.config.js"},
		{"tailwind.config.js", 2, "found tailwind.config.js"},
	}

	for _, c := range checks {
		if exists(filepath.Join(root, c.rel)) {
			score += c.weight
			reasons = append(reasons, c.reason)
		}
	}

	return DetectResult{
		IsLaravel:  false,
		Confidence: score,
		Reasons:    reasons,
		Type:       "node",
	}
}

// 為了讓 DetectNode 也能在 detect package 內使用 DetectResult
// DetectResult 已在 laravel.go 定義，含 Type 欄位可標記偵測類型
