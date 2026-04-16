package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"wincmp/internal/crypto"
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
	RestoreLastState bool             `json:"restore_last_state"`
	MinimizeToTray   bool             `json:"minimize_to_tray"`
	RunOnBoot        bool             `json:"run_on_boot"`
	Theme            string           `json:"theme"` // 主題設定: "light", "dark", "system" (預設)
	LastServiceState LastServiceState `json:"last_service_state,omitempty"`

	// 日誌設定
	LogLevel        string `json:"log_level"`
	LogToConsole    bool   `json:"log_to_console"`
	MaxLogRetention int    `json:"max_log_retention"` // 天數
	MaxLogLines     int    `json:"max_log_lines"`     // UI 顯示行數限制

	AutoUpdateHosts bool `json:"auto_update_hosts"` // 自動更新 Hosts

	PHP PHPSettings `json:"php,omitempty"`

	MariaDBExternal bool   `json:"mariadb_external"`
	MariaDBBasedir  string `json:"mariadb_basedir"`
	MariaDBDatadir  string `json:"mariadb_datadir"`
	MariaDBType     string `json:"mariadb_type"`
	MariaDBPort     int    `json:"mariadb_port"`
	MariaDBUser     string `json:"mariadb_user"`
	MariaDBPassword string `json:"mariadb_password"`

	MailpitSMTPPort int  `json:"mailpit_smtp_port,omitempty"` // SMTP 端口 (預設 1025)
	MailpitHTTPPort int  `json:"mailpit_http_port,omitempty"` // 網頁端口 (預設 8025)
	MailpitUseDB    bool `json:"mailpit_use_db,omitempty"`    // 是否使用 database 持久化存儲
}

// LastServiceState 記錄各個服務上次關閉時的狀態
type LastServiceState struct {
	Caddy   bool            `json:"caddy"`
	MariaDB bool            `json:"mariadb"`
	Mailpit bool            `json:"mailpit"`
	PHP     map[string]bool `json:"php"` // key 為 PHP 版本號 (例: "8.2.30"), value 為啟動狀態
}

// PHPSettings 控制 PHP 進程數量與基礎 Port
type PHPSettings struct {
	ProcessesPerVersion int            `json:"processes_per_version"` // 預設值 (例如 3)
	Processes           map[string]int `json:"processes"`             // 個別版本設定 (key 為 Minor Version, 如 "8.2")
}

// ProjectConfig 單一專案設定
type ProjectConfig struct {
	ID             string   `json:"id,omitempty"`
	Name           string   `json:"name"`
	Domains        []string `json:"domains"`
	Type           string   `json:"type,omitempty"`         // 專案類型: "static", "laravel", "next", "nuxt", "astro", "vite", "python", "python_django", "python_fastapi", "python_flask", "go_api", "pocketbase", "custom"
	RuntimeType    string   `json:"runtime_type,omitempty"` // 執行器: "auto", "none", "node", "bun", "python", "go_air", "go_run", "custom"
	PHPVersion     string   `json:"php_version"`
	RootPath       string   `json:"root_path"`
	SSLCrt         string   `json:"ssl_crt"`
	SSLKey         string   `json:"ssl_key"`
	UseSSL         bool     `json:"use_ssl"`
	Enabled        bool     `json:"enabled"`
	RuntimePort    int      `json:"runtime_port,omitempty"`
	RuntimeMode    string   `json:"runtime_mode,omitempty"`    // "Background" 或 "Terminal"
	RuntimeVersion string   `json:"runtime_version,omitempty"` // "24.14.1" 等
	Command        string   `json:"command,omitempty"`         // 自定義啟動指令 (Custom 類型或手動覆寫)
	CommandDirty   bool     `json:"command_dirty,omitempty"`   // 使用者是否手動修改過 Command
	UseWinCMPBin   bool     `json:"use_wincmp_bin,omitempty"`  // 是否使用 WinCMP 內建執行檔 (bundled runtime)

	// ConfigExists 快取：Caddy 設定檔是否存在（避免每次渲染都 os.Stat）
	ConfigExists bool `json:"-"`

	// Deprecated: 向後相容
	UseEnvBin   bool   `json:"use_env_bin,omitempty"`
	NodePort    int    `json:"node_port,omitempty"`
	NodeMode    string `json:"node_mode,omitempty"`
	NodeVersion string `json:"node_version,omitempty"`
}

// migrateLegacyNodeFields 將舊版欄位遷移至新模型
func migrateLegacyNodeFields(cfg *WincmpConfig) {
	for i := range cfg.Projects {
		p := &cfg.Projects[i]
		// 1. 舊版 Port/Mode/Version 欄位遷移
		if p.RuntimePort == 0 && p.NodePort > 0 {
			p.RuntimePort = p.NodePort
		}
		if p.RuntimeMode == "" && p.NodeMode != "" {
			p.RuntimeMode = p.NodeMode
		}
		if p.RuntimeVersion == "" && p.NodeVersion != "" {
			p.RuntimeVersion = p.NodeVersion
		}
		if !p.UseWinCMPBin && p.UseEnvBin {
			p.UseWinCMPBin = p.UseEnvBin
		}

		// 2. 舊版 Type 語義遷移
		if p.RuntimeType == "" {
			switch p.Type {
			case "node", "bun":
				p.RuntimeType = p.Type
			case "python":
				p.RuntimeType = "python"
			case "go":
				p.RuntimeType = "go_air"
			case "custom":
				p.RuntimeType = "custom"
			case "laravel":
				p.RuntimeType = "none"
			}
		}

		// 3. 新版 Type 遷移 (舊版 "go" → "go_api")
		switch p.Type {
		case "go":
			p.Type = "go_api"
		case "node", "bun":
			// 舊版 node/bun 類型：如果有框架偵測結果會在 main.go 處理
			// 否則 fallback 為 vite
			if p.RuntimeType == p.Type {
				p.Type = "vite"
			}
		}

		// 4. 新版 RuntimeType 遷移
		switch p.RuntimeType {
		case "go":
			p.RuntimeType = "go_air"
		}

		// 5. 靜態類型 / Laravel 不需要 Runtime
		if p.Type == "" || p.Type == "static" || p.Type == "laravel" {
			p.RuntimeType = "none"
			p.RuntimePort = 0
		}

		// 清理舊欄位
		p.NodePort = 0
		p.NodeMode = ""
		p.NodeVersion = ""
		p.UseEnvBin = false
	}
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

	// 向後相容：舊版 node_port/node_mode/node_version → 新版 runtime_*
	migrateLegacyNodeFields(&cfg)

	// 載入後解密 MariaDB 密碼（向後相容：明文密碼直接回傳）
	if crypto.IsEncrypted(cfg.Global.MariaDBPassword) {
		dec, err := crypto.Decrypt(cfg.Global.MariaDBPassword)
		if err != nil {
			// 解密失敗時保留原文，避免阻斷正常使用
			_ = err
		} else {
			cfg.Global.MariaDBPassword = dec
		}
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
		os.MkdirAll(backupDir, 0700)

		bakPath := filepath.Join(backupDir, filepath.Base(path)+".bak")
		content, readErr := os.ReadFile(path)
		if readErr == nil {
			os.WriteFile(bakPath, content, 0600)
		}
	}

	// 2. 序列化新設定 (清理已遷移的舊欄位)
	for i := range c.Projects {
		c.Projects[i].NodePort = 0
		c.Projects[i].NodeMode = ""
		c.Projects[i].NodeVersion = ""
		c.Projects[i].UseEnvBin = false
	}

	// 儲存前加密 MariaDB 密碼
	originalPassword := c.Global.MariaDBPassword
	if originalPassword != "" && !crypto.IsEncrypted(originalPassword) {
		enc, err := crypto.Encrypt(originalPassword)
		if err != nil {
			// 加密失敗時仍以明文儲存，記錄錯誤但不中斷
			_ = err
		} else {
			c.Global.MariaDBPassword = enc
		}
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("無法序列化設定: %w", err)
	}

	// 儲存後還原明文密碼（避免記憶體中的 cfg 被改為密文）
	c.Global.MariaDBPassword = originalPassword

	// 確保目錄存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("無法建立目錄 %s: %w", dir, err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
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

// RefreshConfigExists 預計算所有專案的 Caddy 設定檔是否存在
// 避免每次渲染 List 都重複 os.Stat()，提升 UI 效能
func (c *WincmpConfig) RefreshConfigExists(baseDir string) {
	sitesDir := filepath.Join(baseDir, "conf", "sites")
	for i := range c.Projects {
		caddyConfigPath := filepath.Join(sitesDir, c.Projects[i].Name+".caddy")
		if _, err := os.Stat(caddyConfigPath); err == nil {
			c.Projects[i].ConfigExists = true
		} else {
			c.Projects[i].ConfigExists = false
		}
	}
}

// SanitizeProjectName 清理專案名稱，移除 Windows 檔名禁用字元和 Shell 危險字元
// 將特殊字元替換為連字號，確保名稱可用於檔案路徑
func SanitizeProjectName(name string) string {
	if name == "" {
		return "project"
	}

	// Windows 檔名禁用字元: < > : " / \ | ? *
	// Shell 危險字元: & | ; < > $ " ! ( ) { } [ ] ' `
	// 額外清理: 空白字元
	specialChars := []string{
		"<", ">", ":", "\"", "/", "\\", "|", "?", "*",
		"&", "|", ";", "$", "!", "(", ")", "{", "}", "[", "]", "'", "`",
	}

	result := name
	for _, char := range specialChars {
		result = strings.ReplaceAll(result, char, "-")
	}

	// 空白字元也替換為連字號
	result = strings.ReplaceAll(result, " ", "-")
	result = strings.ReplaceAll(result, "\t", "-")

	// 連續連字號合併為單個
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}

	// 去除首尾連字號
	result = strings.Trim(result, "-")

	// 如果清理後為空，回傳預設值
	if result == "" {
		return "project"
	}

	return result
}
