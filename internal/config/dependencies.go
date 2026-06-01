package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DependencyItem 定義單一依賴項目的版本與下載網址
type DependencyItem struct {
	Version string `json:"version"`
	URL     string `json:"url"`
}

// DependencyConfig 對應 dependencies.json 的結構，儲存所有依賴項
type DependencyConfig map[string]DependencyItem

// DefaultDependencies 預設的依賴版本與下載網址
var DefaultDependencies = DependencyConfig{
	"caddy": {
		Version: "2.7.6",
		URL:     "https://github.com/caddyserver/caddy/releases/download/v2.7.6/caddy_2.7.6_windows_amd64.zip",
	},
	"mariadb": {
		Version: "11.4.2",
		URL:     "https://archive.mariadb.org/mariadb-11.4.2/winx64-packages/mariadb-11.4.2-winx64.zip",
	},
	"php73": {
		Version: "7.3.33",
		URL:     "https://windows.php.net/downloads/releases/archives/php-7.3.33-nts-Win32-VC15-x64.zip",
	},
	"php82": {
		Version: "8.2.30",
		URL:     "https://windows.php.net/downloads/releases/archives/php-8.2.30-nts-Win32-vs16-x64.zip",
	},
	"php83": {
		Version: "8.3.28",
		URL:     "https://windows.php.net/downloads/releases/archives/php-8.3.28-nts-Win32-vs16-x64.zip",
	},
	"composer": {
		Version: "2.7.7",
		URL:     "https://github.com/composer/composer/releases/download/2.7.7/composer.phar",
	},
	"heidisql": {
		Version: "12.17",
		URL:     "https://github.com/HeidiSQL/HeidiSQL/releases/download/12.17/HeidiSQL_12.17_64_Portable.zip",
	},
	"node": {
		Version: "20.15.0",
		URL:     "https://nodejs.org/dist/v20.15.0/node-v20.15.0-win-x64.zip",
	},
}

// LoadDependencies 從指定路徑載入 dependencies.json
func LoadDependencies(path string) (DependencyConfig, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// 檔案不存在，建立預設檔
		if err := SaveDependencies(path, DefaultDependencies); err != nil {
			return nil, fmt.Errorf("無法建立預設依賴設定檔: %w", err)
		}
		return DefaultDependencies, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("無法讀取依賴設定檔 %s: %w", path, err)
	}

	var cfg DependencyConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("無法解析依賴設定檔 %s: %w", path, err)
	}

	return cfg, nil
}

// SaveDependencies 將依賴設定寫回 JSON 檔案
func SaveDependencies(path string, cfg DependencyConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("無法建立目錄 %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("無法序列化依賴設定: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("無法寫入依賴設定檔 %s: %w", path, err)
	}

	return nil
}
