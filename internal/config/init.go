package config

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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

	err = fs.WalkDir(subFS, ".", func(path string, d fs.DirEntry, err error) error {
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
	if err != nil {
		return err
	}

	// 自動檢測並平滑升級既有 php.ini 的 OPcache 配置
	if err := EnsurePHPIniOptimizations(baseDir); err != nil {
		return err
	}

	// 自動檢測並平滑升級既有 php.ini 的 Redis 擴充配置
	return EnsurePHPIniRedisExtension(baseDir)
}

// EnsurePHPIniOptimizations 檢查現有的 conf/php/php.ini 是否已配置 OPcache。
// 若使用者是舊版本升級且 php.ini 缺少 OPcache 配置，會自動建立 .bak 備份並於末尾安全追加優化設定，不影響原使用者任何自訂項目。
func EnsurePHPIniOptimizations(baseDir string) error {
	phpIniPath := filepath.Join(baseDir, "conf", "php", "php.ini")
	data, err := os.ReadFile(phpIniPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("讀取 php.ini 失敗: %w", err)
	}

	content := string(data)

	// 檢查是否已存在 opcache 配置
	if strings.Contains(content, "zend_extension=opcache") || strings.Contains(content, "[opcache]") {
		return nil
	}

	// 建立安全備份
	bakPath := filepath.Join(baseDir, "conf", "php", "php.ini.bak")
	_ = os.WriteFile(bakPath, data, 0644)

	// 構造平滑追加的 OPcache 配置
	opcacheSnippet := `
; === OPcache 效能極速優化 (大幅縮短 Laravel 頁面渲染時間) ===
zend_extension=opcache
[opcache]
opcache.enable = 1
opcache.enable_cli = 0
opcache.memory_consumption = 128
opcache.interned_strings_buffer = 16
opcache.max_accelerated_files = 10000
opcache.validate_timestamps = 1
opcache.revalidate_freq = 0
opcache.save_comments = 1
`

	if !strings.HasSuffix(content, "\n") {
		opcacheSnippet = "\n" + opcacheSnippet
	}

	newContent := content + opcacheSnippet
	if err := os.WriteFile(phpIniPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("追加 OPcache 至 php.ini 失敗: %w", err)
	}

	return nil
}

// EnsurePHPIniRedisExtension 檢查現有的 conf/php/php.ini 是否已配置 extension=redis。
// 若為舊版本使用者且 php.ini 缺少該擴充或被註解，會自動建立 .bak 備份並平滑啟用，消除假陽性脫鉤狀況。
func EnsurePHPIniRedisExtension(baseDir string) error {
	phpIniPath := filepath.Join(baseDir, "conf", "php", "php.ini")
	data, err := os.ReadFile(phpIniPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("讀取 php.ini 失敗: %w", err)
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	hasActiveRedis := false
	hasCommentedRedis := false
	commentedIdx := -1

	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "extension=redis" || strings.HasPrefix(trimmed, "extension=redis ") {
			hasActiveRedis = true
			break
		}
		if trimmed == ";extension=redis" || strings.HasPrefix(trimmed, ";extension=redis") {
			hasCommentedRedis = true
			commentedIdx = idx
		}
	}

	if hasActiveRedis {
		return nil
	}

	// 建立安全備份
	bakPath := filepath.Join(baseDir, "conf", "php", "php.ini.bak")
	_ = os.WriteFile(bakPath, data, 0644)

	var newContent string
	if hasCommentedRedis && commentedIdx >= 0 {
		// 取消註解
		lines[commentedIdx] = "extension=redis"
		newContent = strings.Join(lines, "\n")
	} else {
		// 安全追加於檔案末尾
		redisSnippet := `
; === Redis 快取模組 (WinCMP 自動平滑追加) ===
extension=redis
`
		if !strings.HasSuffix(content, "\n") {
			redisSnippet = "\n" + redisSnippet
		}
		newContent = content + redisSnippet
	}

	if err := os.WriteFile(phpIniPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("啟用 php.ini redis 擴充失敗: %w", err)
	}

	return nil
}
