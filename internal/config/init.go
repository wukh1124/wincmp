package config

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"wincmp/conf"
)

// RestoreDefaultConf 遞迴將 embedded 內的預設設定檔釋放到 baseDir/conf 中。
// 採用安全機制：若檔案已存在則不覆蓋，保留已自訂的設定。
func RestoreDefaultConf(baseDir string) error {
	err := fs.WalkDir(conf.DefaultConfFS, ".", func(path string, d fs.DirEntry, err error) error {
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
		data, err := fs.ReadFile(conf.DefaultConfFS, path)
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

// IsPHPIniRedisExtensionLine 判斷 php.ini 單行是否為 redis 擴充設定
// 支援 extension=redis、extension = redis、extension=php_redis.dll 等寫法
func IsPHPIniRedisExtensionLine(line string) (active bool, commented bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false, false
	}
	if strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "#") {
		rest := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(trimmed, ";"), "#"))
		return false, isPHPIniRedisExtValue(rest)
	}
	return isPHPIniRedisExtValue(trimmed), false
}

func isPHPIniRedisExtValue(s string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), " ", ""))
	if !strings.HasPrefix(normalized, "extension=") {
		return false
	}
	value := strings.TrimPrefix(normalized, "extension=")
	value = strings.TrimSuffix(value, ".dll")
	return value == "redis" || value == "php_redis"
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
		active, commented := IsPHPIniRedisExtensionLine(line)
		if active {
			hasActiveRedis = true
			break
		}
		if commented {
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

// ResolveEffectivePHPIni 依據指定 PHP 版本的 ext 目錄實際 DLL 狀況，回傳可直接使用的 php.ini 路徑。
// 主要處理跨版本相容差異：
//   - gd ↔ gd2（PHP 7.3/7.4 為 php_gd2.dll，PHP 8.0+ 為 php_gd.dll）
//   - zip（PHP 7.3 為內建模組無 php_zip.dll，PHP 7.4+ 才需載入 DLL）
//   - redis（未安裝 php_redis.dll 時省略，避免啟動與 CLI 噴 Warning）
// 若當前版本完全相容主設定檔，直接回傳原 conf/php/php.ini 路徑，不額外產生檔案。
func ResolveEffectivePHPIni(baseDir, phpDir string) string {
	baseIniPath := filepath.Join(baseDir, "conf", "php", "php.ini")
	data, err := os.ReadFile(baseIniPath)
	if err != nil {
		return baseIniPath
	}

	extDir := filepath.Join(phpDir, "ext")
	if info, err := os.Stat(extDir); err != nil || !info.IsDir() {
		return baseIniPath
	}

	hasDLL := func(name string) bool {
		_, err := os.Stat(filepath.Join(extDir, name))
		return err == nil
	}

	hasGD := hasDLL("php_gd.dll")
	hasGD2 := hasDLL("php_gd2.dll")
	hasZip := hasDLL("php_zip.dll")
	hasRedis := hasDLL("php_redis.dll")

	content := string(data)
	lineEnding := "\n"
	if strings.Contains(content, "\r\n") {
		lineEnding = "\r\n"
	}
	rawLines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	changed := false

	for i, line := range rawLines {
		extName, ok := parseActivePHPExtensionName(line)
		if !ok {
			continue
		}
		switch extName {
		case "gd", "php_gd":
			if !hasGD {
				if hasGD2 {
					rawLines[i] = "extension=gd2"
				} else {
					rawLines[i] = "; [WinCMP] omitted missing gd"
				}
				changed = true
			}
		case "gd2", "php_gd2":
			if !hasGD2 {
				if hasGD {
					rawLines[i] = "extension=gd"
				} else {
					rawLines[i] = "; [WinCMP] omitted missing gd2"
				}
				changed = true
			}
		case "zip", "php_zip":
			if !hasZip {
				rawLines[i] = "; [WinCMP] omitted zip (built-in or missing)"
				changed = true
			}
		case "redis", "php_redis":
			if !hasRedis {
				rawLines[i] = "; [WinCMP] omitted missing redis"
				changed = true
			}
		}
	}

	if !changed {
		return baseIniPath
	}

	runtimeDir := filepath.Join(baseDir, "conf", "php", ".runtime")
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		return baseIniPath
	}

	versionKey := filepath.Base(filepath.Clean(phpDir))
	if versionKey == "" || versionKey == "." {
		versionKey = "default"
	}
	targetPath := filepath.Join(runtimeDir, versionKey+".ini")
	rendered := strings.Join(rawLines, lineEnding)

	if existing, err := os.ReadFile(targetPath); err == nil && string(existing) == rendered {
		return targetPath
	}
	if err := os.WriteFile(targetPath, []byte(rendered), 0644); err != nil {
		return baseIniPath
	}
	return targetPath
}

// PreparePHPCLIRuntimeDir 為指定 PHP 版本建立含獨立 php.ini（含 extension_dir 與 CA 路徑）的目錄，
// 供 PHPRC 環境變數使用，確保 Composer 衍生的 @php 子行程（如 @php artisan ...）也能完整載入擴充與憑證。
func PreparePHPCLIRuntimeDir(baseDir, phpDir string) string {
	effIni := ResolveEffectivePHPIni(baseDir, phpDir)
	data, err := os.ReadFile(effIni)
	if err != nil {
		return ""
	}

	versionKey := filepath.Base(filepath.Clean(phpDir))
	if versionKey == "" || versionKey == "." {
		versionKey = "default"
	}
	cliDir := filepath.Join(baseDir, "conf", "php", ".runtime", versionKey)
	if err := os.MkdirAll(cliDir, 0755); err != nil {
		return ""
	}

	extDir := filepath.ToSlash(filepath.Join(phpDir, "ext"))
	cacertPath := filepath.Join(baseDir, "conf", "ssl", "cacert.pem")
	var extra strings.Builder
	extra.WriteString("\n; === WinCMP CLI Runtime Overrides ===\n")
	extra.WriteString("extension_dir = \"" + extDir + "\"\n")
	if _, err := os.Stat(cacertPath); err == nil {
		caSlash := filepath.ToSlash(cacertPath)
		extra.WriteString("openssl.cafile = \"" + caSlash + "\"\n")
		extra.WriteString("curl.cainfo = \"" + caSlash + "\"\n")
	}

	rendered := string(data) + extra.String()
	targetIni := filepath.Join(cliDir, "php.ini")
	if existing, err := os.ReadFile(targetIni); err != nil || string(existing) != rendered {
		if err := os.WriteFile(targetIni, []byte(rendered), 0644); err != nil {
			return ""
		}
	}
	return cliDir
}

func parseActivePHPExtensionName(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "#") {
		return "", false
	}
	// 剝除行尾註解（例：extension=exif      ; 備註）
	if idx := strings.IndexAny(trimmed, ";#"); idx >= 0 {
		trimmed = strings.TrimSpace(trimmed[:idx])
	}
	normalized := strings.ToLower(strings.ReplaceAll(trimmed, " ", ""))
	if !strings.HasPrefix(normalized, "extension=") {
		return "", false
	}
	val := strings.TrimPrefix(normalized, "extension=")
	val = strings.TrimSuffix(val, ".dll")
	return val, val != ""
}
