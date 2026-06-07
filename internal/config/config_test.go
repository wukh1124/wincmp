package config

import (
	"os"
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

func TestConfig_GetProjectPhysicalRoot(t *testing.T) {
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
	root1 := cfg.GetProjectPhysicalRoot(p1, "C:/app")
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
	root2 := cfg2.GetProjectPhysicalRoot(p2, "C:/app")
	expected2 := filepath.Clean("C:/app/www/proj2")
	if filepath.Clean(root2) != expected2 {
		t.Errorf("相對路徑解析錯誤: 預期 %s, 實際 %s", expected2, root2)
	}

	// 3. Laravel 專案不補上 public
	p3 := ProjectConfig{
		Name:     "proj3",
		RootPath: "C:/www/proj3",
		Type:     "laravel",
	}
	root3 := cfg.GetProjectPhysicalRoot(p3, "C:/app")
	expected3 := filepath.Clean("C:/www/proj3")
	if filepath.Clean(root3) != expected3 {
		t.Errorf("Laravel 專案物理路徑解析錯誤: 預期 %s, 實際 %s", expected3, root3)
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

func TestRestoreDefaultConf(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "wincmp-test-*")
	if err != nil {
		t.Fatalf("無法建立暫時目錄: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 1. 第一次執行，確認是否順利釋放檔案
	if err := RestoreDefaultConf(tempDir); err != nil {
		t.Fatalf("RestoreDefaultConf 失敗: %v", err)
	}

	caddyfilePath := filepath.Join(tempDir, "conf", "Caddyfile")
	if _, err := os.Stat(caddyfilePath); err != nil {
		t.Errorf("Caddyfile 釋放失敗: %v", err)
	}

	gitkeepPath := filepath.Join(tempDir, "conf", "sites", ".gitkeep")
	if _, err := os.Stat(gitkeepPath); err == nil {
		t.Error(".gitkeep 檔案不應該被釋放建立，但它居然存在了！")
	}

	// 2. 測試防覆蓋機制
	originalContent := "custom caddyfile configuration"
	if err := os.WriteFile(caddyfilePath, []byte(originalContent), 0644); err != nil {
		t.Fatalf("寫入自訂 Caddyfile 失敗: %v", err)
	}

	if err := RestoreDefaultConf(tempDir); err != nil {
		t.Fatalf("第二次 RestoreDefaultConf 失敗: %v", err)
	}

	data, err := os.ReadFile(caddyfilePath)
	if err != nil {
		t.Fatalf("讀取 Caddyfile 失敗: %v", err)
	}

	if string(data) != originalContent {
		t.Errorf("防覆蓋機制失效，檔案被重新覆蓋！預期為 %q, 實際為 %q", originalContent, string(data))
	}
}
