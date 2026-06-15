package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"wincmp/conf"
)

// DependencyItem 定義單一依賴項目的版本與下載網址
type DependencyItem struct {
	Version string `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256,omitempty"`
}

// DependencyConfig 對應 dependencies.json 的結構，儲存所有依賴項
type DependencyConfig map[string]DependencyItem

// DefaultDependencies 預設的依賴版本與下載網址，由 conf.DependenciesJSON 動態解析初始化
var DefaultDependencies DependencyConfig

func init() {
	DefaultDependencies = make(DependencyConfig)
	if err := json.Unmarshal(conf.DependenciesJSON, &DefaultDependencies); err != nil {
		panic(fmt.Sprintf("無法解析嵌入的 dependencies.json: %v", err))
	}
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
