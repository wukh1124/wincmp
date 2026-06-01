package config

import (
	"path/filepath"
	"testing"
)

func TestSanitizeProjectName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Empty string", "", "project"},
		{"Standard name", "my-project", "my-project"},
		{"Forbidden characters", "my:project/test*name", "my-project-test-name"},
		{"Shell symbols", "my&project;test|name", "my-project-test-name"},
		{"Spaces and tabs", "my project\tname", "my-project-name"},
		{"Multiple consecutive dashes", "my--project---name", "my-project-name"},
		{"Trailing/leading dashes", "-my-project-", "my-project"},
		{"All symbols to empty fallback", "<>:\"/\\|?*", "project"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeProjectName(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeProjectName(%q) = %q; 預期 %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConfig_GetProjectRoot(t *testing.T) {
	cfg := &WincmpConfig{
		Global: GlobalConfig{
			DefaultWWW: "C:/www",
		},
	}

	// 1. 自訂路徑
	p1 := ProjectConfig{
		Name:     "proj1",
		RootPath: "D:/custom/path",
	}
	root1 := cfg.GetProjectRoot(p1, "C:/app")
	if root1 != "D:/custom/path" {
		t.Errorf("預期為 D:/custom/path, 實際為 %s", root1)
	}

	// 2. 預設路徑 (相對路徑)
	cfg2 := &WincmpConfig{
		Global: GlobalConfig{
			DefaultWWW: "www",
		},
	}
	p2 := ProjectConfig{
		Name: "proj2",
	}
	root2 := cfg2.GetProjectRoot(p2, "C:/app")
	expected2 := filepath.Clean("C:/app/www/proj2")
	if filepath.Clean(root2) != expected2 {
		t.Errorf("相對路徑解析錯誤: 預期 %s, 實際 %s", expected2, root2)
	}

	// 3. Laravel 專案自動補上 public
	p3 := ProjectConfig{
		Name:     "proj3",
		RootPath: "C:/www/proj3",
		Type:     "laravel",
	}
	root3 := cfg.GetProjectRoot(p3, "C:/app")
	expected3 := filepath.Clean("C:/www/proj3/public")
	if filepath.Clean(root3) != expected3 {
		t.Errorf("Laravel 專案路徑解析錯誤: 預期 %s, 實際 %s", expected3, root3)
	}
}

func TestConfig_MigrateLegacyNodeFields(t *testing.T) {
	cfg := &WincmpConfig{
		Projects: []ProjectConfig{
			{
				Name:        "legacy-proj",
				NodePort:    8080,
				NodeMode:    "Terminal",
				NodeVersion: "v18.0.0",
				UseEnvBin:   true,
				Type:        "go",
			},
		},
	}

	migrateLegacyNodeFields(cfg)

	p := cfg.Projects[0]
	if p.RuntimePort != 8080 {
		t.Errorf("RuntimePort 遷移錯誤: 實際為 %d", p.RuntimePort)
	}
	if p.RuntimeMode != "Terminal" {
		t.Errorf("RuntimeMode 遷移錯誤: 實際為 %s", p.RuntimeMode)
	}
	if p.RuntimeVersion != "v18.0.0" {
		t.Errorf("RuntimeVersion 遷移錯誤: 實際為 %s", p.RuntimeVersion)
	}
	if !p.UseWinCMPBin {
		t.Error("UseWinCMPBin 應繼承自 UseEnvBin")
	}
	if p.Type != "go_api" {
		t.Errorf("Type 'go' 應遷移為 'go_api': 實際為 %s", p.Type)
	}
	if p.NodePort != 0 || p.NodeMode != "" || p.NodeVersion != "" || p.UseEnvBin {
		t.Error("遷移後應清除 legacy 欄位")
	}
}
