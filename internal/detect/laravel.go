package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type DetectResult struct {
	IsLaravel  bool
	Confidence int
	Reasons    []string
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func hasLaravelComposer(root string) bool {
	f := filepath.Join(root, "composer.json")
	b, err := os.ReadFile(f)
	if err != nil {
		return false
	}

	var obj struct {
		Require    map[string]string `json:"require"`
		RequireDev map[string]string `json:"require-dev"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return false
	}

	if _, ok := obj.Require["laravel/framework"]; ok {
		return true
	}
	if _, ok := obj.Require["laravel/laravel"]; ok {
		return true
	}
	if _, ok := obj.RequireDev["laravel/pint"]; ok {
		return true
	}

	return false
}

func DetectLaravel(root string) DetectResult {
	score := 0
	reasons := make([]string, 0)

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
		{"app/Http", 8, "found app/Http"},
		{"resources/views", 6, "found resources/views"},
		{"database/migrations", 6, "found database/migrations"},
		{"storage", 5, "found storage"},
	}

	for _, c := range checks {
		if exists(filepath.Join(root, c.rel)) {
			score += c.weight
			reasons = append(reasons, c.reason)
		}
	}

	if hasLaravelComposer(root) {
		score += 35
		reasons = append(reasons, "composer.json requires laravel")
	}

	return DetectResult{
		IsLaravel:  score >= 50,
		Confidence: score,
		Reasons:    reasons,
	}
}
