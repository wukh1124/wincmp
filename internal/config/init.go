package config

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// DefaultConfFS 嵌入 default_conf 目錄下的所有設定檔與子目錄
//go:embed all:default_conf/*
var DefaultConfFS embed.FS

// RestoreDefaultConf 遞迴將 embedded 內的 default_conf 釋放到 baseDir/conf 中。
// 採用安全機制：若檔案已存在則不覆蓋，保留已自訂的設定。
func RestoreDefaultConf(baseDir string) error {
	// embed.FS 會保留前綴 "default_conf"，使用 Sub 取得無前綴的子檔案系統
	subFS, err := fs.Sub(DefaultConfFS, "default_conf")
	if err != nil {
		return fmt.Errorf("無法取得子檔案系統 default_conf: %w", err)
	}

	return fs.WalkDir(subFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil
		}

		targetPath := filepath.Join(baseDir, "conf", path)

		if d.IsDir() {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("無法建立目錄 %s: %w", targetPath, err)
			}
			return nil
		}

		// 🌟 跳過 .gitkeep 佔位檔，運行時不需要建立
		if filepath.Base(path) == ".gitkeep" {
			return nil
		}

		// 🌟 安全機制：若檔案已存在，則跳過，避免覆蓋自訂設定
		if _, err := os.Stat(targetPath); err == nil {
			return nil
		}

		// 讀取嵌入的內容
		data, err := fs.ReadFile(subFS, path)
		if err != nil {
			return fmt.Errorf("無法讀取嵌入檔案 %s: %w", path, err)
		}

		// 確保父目錄存在
		parentDir := filepath.Dir(targetPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("無法建立父目錄 %s: %w", parentDir, err)
		}

		// 寫入目標檔案
		if err := os.WriteFile(targetPath, data, 0644); err != nil {
			return fmt.Errorf("無法寫入檔案 %s: %w", targetPath, err)
		}

		return nil
	})
}
