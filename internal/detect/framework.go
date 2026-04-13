package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// DetectFramework 從 package.json 偵測前端框架 (next / nuxt)
func DetectFramework(root string) string {
	pkgJson := filepath.Join(root, "package.json")
	if !exists(pkgJson) {
		return ""
	}

	b, err := os.ReadFile(pkgJson)
	if err != nil {
		return ""
	}

	var obj struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return ""
	}

	// 優先判定 Nuxt (因為它通常有更具體的啟動參數)
	if _, ok := obj.Dependencies["nuxt"]; ok {
		return "nuxt"
	}
	if _, ok := obj.Dependencies["next"]; ok {
		return "next"
	}

	return ""
}
