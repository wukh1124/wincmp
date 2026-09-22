package main

import (
	"embed"
	"os"
	"path/filepath"
)

// EmbeddedDocs 內嵌基礎開源授權與說明文檔
//
//go:embed LICENSE THIRD-PARTY-NOTICES.md packaging/wincmp/readme.md packaging/wincmp/readme_zh.md
var EmbeddedDocs embed.FS

// EnsureBaseDocumentation 檢查並在 baseDir 下安全釋放必要的合規與說明文檔（若檔案不存在才建立）。
// 此機制確保使用者僅下載單獨 exe 執行檔時，首次啟動能自動補齊完整的開源許可證與免責聲明。
func EnsureBaseDocumentation(baseDir string) {
	docs := []struct {
		embedPath  string
		targetName string
	}{
		{"LICENSE", "LICENSE"},
		{"THIRD-PARTY-NOTICES.md", "THIRD-PARTY-NOTICES.md"},
		{"packaging/wincmp/readme_zh.md", "readme_zh.md"},
		{"packaging/wincmp/readme.md", "readme.md"},
	}

	for _, doc := range docs {
		targetPath := filepath.Join(baseDir, doc.targetName)
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			if data, readErr := EmbeddedDocs.ReadFile(doc.embedPath); readErr == nil {
				_ = os.WriteFile(targetPath, data, 0644)
			}
		}
	}
}
