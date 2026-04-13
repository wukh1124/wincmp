package detect

import "path/filepath"

// DetectBun 偵測 Bun 專案 (bun.lockb / bunfig.toml)
func DetectBun(root string) DetectResult {
	score := 0
	reasons := make([]string, 0)

	if exists(filepath.Join(root, "bun.lockb")) {
		score += 50
		reasons = append(reasons, "found bun.lockb")
	}

	if exists(filepath.Join(root, "bunfig.toml")) {
		score += 30
		reasons = append(reasons, "found bunfig.toml")
	}

	// 如果有 package.json 且 bun.lockb 存在，額外加分
	if exists(filepath.Join(root, "package.json")) && exists(filepath.Join(root, "bun.lockb")) {
		score += 20
		reasons = append(reasons, "found package.json with bun.lockb")
	}

	return DetectResult{
		IsLaravel:  false,
		Confidence: score,
		Reasons:    reasons,
		Type:       "bun",
	}
}
