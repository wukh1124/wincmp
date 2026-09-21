package config

import (
	"os"
	"path/filepath"
	"strings"
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

	for _, checkFile := range []string{"conf/wincmp.json", "conf/dependencies.json", "conf/my.ini"} {
		if _, err := os.Stat(filepath.Join(tempDir, checkFile)); err != nil {
			t.Errorf("%s 釋放失敗: %v", checkFile, err)
		}
	}

	commonCaddyPath := filepath.Join(tempDir, "conf", "snippets", "common.caddy")
	commonCaddyData, err := os.ReadFile(commonCaddyPath)
	if err != nil {
		t.Errorf("snippets/common.caddy 釋放失敗: %v", err)
	} else if !strings.Contains(string(commonCaddyData), "(common_dev)") {
		t.Errorf("snippets/common.caddy 內容缺少 (common_dev) 片段")
	}

	upstreamCaddyPath := filepath.Join(tempDir, "conf", "snippets", "php-upstream.caddy")
	if _, err := os.Stat(upstreamCaddyPath); err != nil {
		t.Errorf("snippets/php-upstream.caddy 釋放失敗: %v", err)
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

	customCommonContent := "# custom common snippet"
	if err := os.WriteFile(commonCaddyPath, []byte(customCommonContent), 0644); err != nil {
		t.Fatalf("寫入自訂 common.caddy 失敗: %v", err)
	}

	// 模擬舊版本 php.ini（無 OPcache）
	phpIniPath := filepath.Join(tempDir, "conf", "php", "php.ini")
	oldPHPIni := "[PHP]\nmemory_limit = 256M\n"
	if err := os.WriteFile(phpIniPath, []byte(oldPHPIni), 0644); err != nil {
		t.Fatalf("模擬寫入舊版 php.ini 失敗: %v", err)
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

	commonData, err := os.ReadFile(commonCaddyPath)
	if err != nil {
		t.Fatalf("讀取 common.caddy 失敗: %v", err)
	}
	if string(commonData) != customCommonContent {
		t.Errorf("common.caddy 防覆蓋機制失效！預期為 %q, 實際為 %q", customCommonContent, string(commonData))
	}

	// 驗證 RestoreDefaultConf 是否確實觸發了 EnsurePHPIniOptimizations
	phpData, err := os.ReadFile(phpIniPath)
	if err != nil {
		t.Fatalf("讀取 php.ini 失敗: %v", err)
	}
	if !strings.Contains(string(phpData), "zend_extension=opcache") {
		t.Errorf("RestoreDefaultConf 未能成功觸發 EnsurePHPIniOptimizations，缺少 opcache 設定: %s", string(phpData))
	}
}

func TestConfig_LoadSaveCustomCommand(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "wincmp-config-test-*")
	if err != nil {
		t.Fatalf("無法建立暫時目錄: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "wincmp.json")
	cfg := &WincmpConfig{
		Projects: []ProjectConfig{
			{
				Name:          "test-custom-cmd-proj",
				Domains:       []string{"test.local"},
				Type:          "vite",
				RuntimeType:   "custom",
				Command:       "npm run start -- --port %PORT%",
				CustomCommand: "npm run start -- --port %PORT%",
			},
		},
	}

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("儲存設定失敗: %v", err)
	}

	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("載入設定失敗: %v", err)
	}

	if len(loaded.Projects) != 1 {
		t.Fatalf("載入的專案數量錯誤: %d", len(loaded.Projects))
	}

	p := loaded.Projects[0]
	if p.CustomCommand != "npm run start -- --port %PORT%" {
		t.Errorf("CustomCommand 欄位反序列化錯誤: 預期 'npm run start -- --port %%PORT%%', 實際 '%s'", p.CustomCommand)
	}
}

func TestEnsurePHPIniOptimizations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "wincmp-phpini-test-*")
	if err != nil {
		t.Fatalf("無法建立暫時目錄: %v", err)
	}
	defer os.RemoveAll(tempDir)

	phpDir := filepath.Join(tempDir, "conf", "php")
	if err := os.MkdirAll(phpDir, 0755); err != nil {
		t.Fatalf("建立目錄失敗: %v", err)
	}

	phpIniPath := filepath.Join(phpDir, "php.ini")
	bakPath := filepath.Join(phpDir, "php.ini.bak")

	// 1. 檔案不存在的情境
	if err := EnsurePHPIniOptimizations(tempDir); err != nil {
		t.Errorf("php.ini 不存在時應安全返回 nil, 錯誤: %v", err)
	}

	// 2. 既有使用者已自訂 extension=zip 但缺少 opcache
	originalContent := "upload_max_filesize = 500M\nextension=zip\n"
	if err := os.WriteFile(phpIniPath, []byte(originalContent), 0644); err != nil {
		t.Fatalf("寫入自訂 php.ini 失敗: %v", err)
	}

	if err := EnsurePHPIniOptimizations(tempDir); err != nil {
		t.Fatalf("EnsurePHPIniOptimizations 執行失敗: %v", err)
	}

	// 驗證 .bak 檔案被建立，內容與原檔案相同
	bakData, err := os.ReadFile(bakPath)
	if err != nil {
		t.Fatalf("預期產生 php.ini.bak 但讀取失敗: %v", err)
	}
	if string(bakData) != originalContent {
		t.Errorf("備份內容與原檔不符: 預期 %q, 實際 %q", originalContent, string(bakData))
	}

	// 驗證原檔內容：既保留原有自訂，又追加了 OPcache
	newIniData, err := os.ReadFile(phpIniPath)
	if err != nil {
		t.Fatalf("讀取更新後的 php.ini 失敗: %v", err)
	}
	newIniStr := string(newIniData)
	if !strings.Contains(newIniStr, "extension=zip") {
		t.Errorf("原有自訂設定被遺失！實際內容: %s", newIniStr)
	}
	if !strings.Contains(newIniStr, "zend_extension=opcache") || !strings.Contains(newIniStr, "[opcache]") {
		t.Errorf("缺少 OPcache 設定區塊！實際內容: %s", newIniStr)
	}

	// 3. 再次執行時（已包含 opcache），不應再次追加或更動
	modTimeBefore, _ := os.Stat(phpIniPath)
	if err := EnsurePHPIniOptimizations(tempDir); err != nil {
		t.Fatalf("第二次執行失敗: %v", err)
	}
	modTimeAfter, _ := os.Stat(phpIniPath)
	if modTimeBefore.ModTime() != modTimeAfter.ModTime() {
		t.Errorf("已有 opcache 設定時不應再次修改檔案")
	}
}

