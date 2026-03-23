package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WincmpConfig 是 conf/wincmp.json 的頂層結構
type WincmpConfig struct {
	Global   GlobalConfig    `json:"global"`
	Projects []ProjectConfig `json:"projects"`
}

// GlobalConfig 全域設定
type GlobalConfig struct {
	DefaultWWW string `json:"default_www"`
	DefaultSSL string `json:"default_ssl"`
	LogFile    string `json:"log_file"`

	// 系統設定
	RestoreLastState  bool             `json:"restore_last_state"`
	MinimizeToTray    bool             `json:"minimize_to_tray"`
	RunOnBoot        bool             `json:"run_on_boot"`
	Theme            string           `json:"theme"` // 主題設定: "light", "dark", "system" (預設)
	LastServiceState LastServiceState `json:"last_service_state,omitempty"`

	// 日誌設定
	LogLevel        string `json:"log_level"`
	LogToConsole    bool   `json:"log_to_console"`
	MaxLogRetention int    `json:"max_log_retention"` // 天數

	AutoUpdateHosts bool `json:"auto_update_hosts"` // 自動更新 Hosts

	PHP PHPSettings `json:"php,omitempty"`
}

// LastServiceState 記錄各個服務上次關閉時的狀態
type LastServiceState struct {
	Caddy   bool            `json:"caddy"`
	MariaDB bool            `json:"mariadb"`
	PHP     map[string]bool `json:"php"` // key 為 PHP 版本號 (例: "8.2.30"), value 為啟動狀態
}

// PHPSettings 控制 PHP 進程數量與基礎 Port
type PHPSettings struct {
	ProcessesPerVersion int            `json:"processes_per_version"` // 預設值 (例如 3)
	Processes           map[string]int `json:"processes"`             // 個別版本設定 (key 為 Minor Version, 如 "8.2")
	BasePortMapping     map[string]int `json:"base_port_mapping"`
}

// ProjectConfig 單一專案設定
type ProjectConfig struct {
	ID         string   `json:"id,omitempty"`
	Name       string   `json:"name"`
	Domains    []string `json:"domains"`
	Type       string   `json:"type,omitempty"`
	PHPVersion string   `json:"php_version"`
	RootPath   string   `json:"root_path"`
	SSLCrt     string   `json:"ssl_crt"`
	SSLKey     string   `json:"ssl_key"`
	UseSSL     bool     `json:"use_ssl"`
	Enabled    bool     `json:"enabled"`
}

// Load 從指定路徑載入 wincmp.json 設定檔
func Load(path string) (*WincmpConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("無法讀取設定檔 %s: %w", path, err)
	}

	var cfg WincmpConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("無法解析設定檔 %s: %w", path, err)
	}

	return &cfg, nil
}

// Save 將設定寫回 JSON 檔案，並在寫入前建立備份
func (c *WincmpConfig) Save(path string) error {
	// 1. 如果舊檔案存在，建立備份到 data/backup
	if _, err := os.Stat(path); err == nil {
		// 推導 data/backup 路徑 (假設 path 是 .../conf/wincmp.json)
		base := filepath.Dir(filepath.Dir(path))
		backupDir := filepath.Join(base, "data", "backup")
		os.MkdirAll(backupDir, 0755)

		bakPath := filepath.Join(backupDir, filepath.Base(path)+".bak")
		content, readErr := os.ReadFile(path)
		if readErr == nil {
			os.WriteFile(bakPath, content, 0644)
		}
	}

	// 2. 序列化新設定
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("無法序列化設定: %w", err)
	}

	// 確保目錄存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("無法建立目錄 %s: %w", dir, err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("無法寫入設定檔 %s: %w", path, err)
	}

	return nil
}

// GetProjectRoot 取得專案的根目錄。若有自訂路徑則使用，否則使用預設 www 路徑。
func (c *WincmpConfig) GetProjectRoot(project ProjectConfig, baseDir string) string {
	root := project.RootPath
	if root == "" {
		wwwDir := c.Global.DefaultWWW
		if !filepath.IsAbs(wwwDir) {
			wwwDir = filepath.Join(baseDir, wwwDir)
		}
		root = filepath.Join(wwwDir, project.Name)
	}

	if project.Type == "laravel" {
		cleanedBase := filepath.Base(filepath.Clean(root))
		if cleanedBase != "public" {
			root = filepath.Join(root, "public")
		}
	}
	return root
}

// GetSSLCertPath 取得 SSL 憑證路徑
func (c *WincmpConfig) GetSSLCertPath(project ProjectConfig, baseDir string) string {
	if project.SSLCrt != "" {
		return project.SSLCrt
	}
	// 預設使用 domain 的第一個來推導
	if len(project.Domains) == 0 {
		return ""
	}
	sslDir := c.Global.DefaultSSL
	if !filepath.IsAbs(sslDir) {
		sslDir = filepath.Join(baseDir, sslDir)
	}
	return filepath.Join(sslDir, project.Domains[0]+".crt")
}

// GetSSLKeyPath 取得 SSL 金鑰路徑
func (c *WincmpConfig) GetSSLKeyPath(project ProjectConfig, baseDir string) string {
	if project.SSLKey != "" {
		return project.SSLKey
	}
	if len(project.Domains) == 0 {
		return ""
	}
	sslDir := c.Global.DefaultSSL
	if !filepath.IsAbs(sslDir) {
		sslDir = filepath.Join(baseDir, sslDir)
	}
	return filepath.Join(sslDir, project.Domains[0]+".key")
}
