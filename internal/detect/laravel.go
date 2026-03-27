package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type DetectResult struct {
	IsLaravel  bool
	Confidence int
	Reasons    []string
	Version    string
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func hasLaravelComposer(root string) (bool, string) {
	f := filepath.Join(root, "composer.json")
	b, err := os.ReadFile(f)
	if err != nil {
		return false, ""
	}

	var obj struct {
		Require    map[string]string `json:"require"`
		RequireDev map[string]string `json:"require-dev"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return false, ""
	}

	if ver, ok := obj.Require["laravel/framework"]; ok {
		return true, parseLaravelVersion(ver)
	}
	if ver, ok := obj.Require["laravel/laravel"]; ok {
		return true, parseLaravelVersion(ver)
	}
	if _, ok := obj.RequireDev["laravel/pint"]; ok {
		return true, ""
	}

	return false, ""
}

func parseLaravelVersion(constraint string) string {
	constraint = strings.TrimSpace(constraint)

	patterns := []struct {
		regex   *regexp.Regexp
		extract func([]string) string
	}{
		{regexp.MustCompile(`^\^(\d+)`), func(m []string) string { return m[1] + ".x" }},
		{regexp.MustCompile(`^(\d+)\.\d+\.\d+`), func(m []string) string { return m[1] + ".x" }},
		{regexp.MustCompile(`^v?(\d+)`), func(m []string) string { return m[1] + ".x" }},
	}

	for _, p := range patterns {
		if m := p.regex.FindStringSubmatch(constraint); m != nil {
			major, _ := strconv.Atoi(m[1])
			if major > 0 {
				return p.extract(m)
			}
		}
	}

	return ""
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

	var laravelVersion string
	if hasLaravel, ver := hasLaravelComposer(root); hasLaravel {
		score += 35
		reasons = append(reasons, "composer.json requires laravel")
		laravelVersion = ver
	}

	return DetectResult{
		IsLaravel:  score >= 50,
		Confidence: score,
		Reasons:    reasons,
		Version:    laravelVersion,
	}
}
