package detect

import "path/filepath"

// DetectPython 偵測 Python 專案 (requirements.txt / app.py / manage.py)
func DetectPython(root string) DetectResult {
	score := 0
	reasons := make([]string, 0)

	if exists(filepath.Join(root, "requirements.txt")) {
		score += 40
		reasons = append(reasons, "found requirements.txt")
	}

	if exists(filepath.Join(root, "app.py")) {
		score += 30
		reasons = append(reasons, "found app.py")
	}

	if exists(filepath.Join(root, "manage.py")) {
		score += 30
		reasons = append(reasons, "found manage.py")
	}

	if exists(filepath.Join(root, "pyproject.toml")) {
		score += 20
		reasons = append(reasons, "found pyproject.toml")
	}

	if exists(filepath.Join(root, "Pipfile")) {
		score += 20
		reasons = append(reasons, "found Pipfile")
	}

	return DetectResult{
		IsLaravel:  false,
		Confidence: score,
		Reasons:    reasons,
		Type:       "python",
	}
}
