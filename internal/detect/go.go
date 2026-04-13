package detect

import "path/filepath"

// DetectGo 偵測 Go (Air) 專案 (go.mod + .air.toml)
func DetectGo(root string) DetectResult {
	score := 0
	reasons := make([]string, 0)

	if exists(filepath.Join(root, "go.mod")) {
		score += 40
		reasons = append(reasons, "found go.mod")
	}

	if exists(filepath.Join(root, ".air.toml")) {
		score += 40
		reasons = append(reasons, "found .air.toml")
	}

	// 有 go.mod 但沒有 .air.toml 也給予低分，因為可能是純 Go 專案
	if exists(filepath.Join(root, "go.mod")) && !exists(filepath.Join(root, ".air.toml")) {
		score += 10
		reasons = append(reasons, "found go.mod (no .air.toml)")
	}

	return DetectResult{
		IsLaravel:  false,
		Confidence: score,
		Reasons:    reasons,
		Type:       "go",
	}
}
