package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsurePHPIniRedisExtension(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "wincmp_php_test_*")
	if err != nil {
		t.Fatalf("建立臨時目錄失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	confPhpDir := filepath.Join(tempDir, "conf", "php")
	if err := os.MkdirAll(confPhpDir, 0755); err != nil {
		t.Fatalf("建立 conf/php 失敗: %v", err)
	}
	phpIniPath := filepath.Join(confPhpDir, "php.ini")

	// 1. 測試情境：原本無 extension=redis，應自動追加並建立 .bak 備份
	initialContent := "[PHP]\nmemory_limit = 512M\n"
	if err := os.WriteFile(phpIniPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("寫入初始 php.ini 失敗: %v", err)
	}

	if err := EnsurePHPIniRedisExtension(tempDir); err != nil {
		t.Fatalf("EnsurePHPIniRedisExtension 執行失敗: %v", err)
	}

	updated, err := os.ReadFile(phpIniPath)
	if err != nil {
		t.Fatalf("讀取更新後 php.ini 失敗: %v", err)
	}
	if !strings.Contains(string(updated), "extension=redis") {
		t.Errorf("預期應包含 extension=redis，但內容為:\n%s", string(updated))
	}

	bakPath := filepath.Join(confPhpDir, "php.ini.bak")
	if _, err := os.Stat(bakPath); err != nil {
		t.Errorf("預期應生成備份檔 php.ini.bak: %v", err)
	}

	// 2. 測試情境：原本有被註解的 ;extension=redis，應取消註解
	commentedContent := "[PHP]\n;extension=redis\nmemory_limit = 512M\n"
	if err := os.WriteFile(phpIniPath, []byte(commentedContent), 0644); err != nil {
		t.Fatalf("寫入註解 php.ini 失敗: %v", err)
	}

	if err := EnsurePHPIniRedisExtension(tempDir); err != nil {
		t.Fatalf("EnsurePHPIniRedisExtension 第二次執行失敗: %v", err)
	}

	uncommented, err := os.ReadFile(phpIniPath)
	if err != nil {
		t.Fatalf("讀取解除註解後 php.ini 失敗: %v", err)
	}
	if !strings.Contains(string(uncommented), "\nextension=redis\n") {
		t.Errorf("預期應解除註解為 extension=redis，但內容為:\n%s", string(uncommented))
	}

	// 3. 測試情境：原本已包含未註解的 extension=redis，不應重複追加
	beforeStat, _ := os.Stat(phpIniPath)
	if err := EnsurePHPIniRedisExtension(tempDir); err != nil {
		t.Fatalf("EnsurePHPIniRedisExtension 第三次執行失敗: %v", err)
	}
	afterStat, _ := os.Stat(phpIniPath)
	if beforeStat.ModTime() != afterStat.ModTime() {
		t.Errorf("原本已啟動時不應再修改檔案")
	}
}

func TestIsPHPIniRedisExtensionLine(t *testing.T) {
	cases := []struct {
		line      string
		active    bool
		commented bool
	}{
		{"extension=redis", true, false},
		{"extension = redis", true, false},
		{"extension=php_redis.dll", true, false},
		{"extension=php_redis", true, false},
		{"  extension=redis  ", true, false},
		{";extension=redis", false, true},
		{"  ; extension = redis", false, true},
		{"#extension=redis", false, true},
		{"extension=mysqli", false, false},
		{"", false, false},
		{"[PHP]", false, false},
	}
	for _, tc := range cases {
		active, commented := IsPHPIniRedisExtensionLine(tc.line)
		if active != tc.active || commented != tc.commented {
			t.Errorf("IsPHPIniRedisExtensionLine(%q) = (%v, %v), want (%v, %v)",
				tc.line, active, commented, tc.active, tc.commented)
		}
	}
}

func TestResolveEffectivePHPIniCrossVersion(t *testing.T) {
	base := t.TempDir()
	confPhpDir := filepath.Join(base, "conf", "php")
	if err := os.MkdirAll(confPhpDir, 0755); err != nil {
		t.Fatalf("建立 conf/php 失敗: %v", err)
	}
	baseIni := filepath.Join(confPhpDir, "php.ini")
	iniContent := "extension=curl\nextension=gd\nextension=zip\nextension=redis\n"
	if err := os.WriteFile(baseIni, []byte(iniContent), 0644); err != nil {
		t.Fatalf("寫入 php.ini 失敗: %v", err)
	}

	// 1. 模擬 PHP 7.3：只有 php_gd2.dll，無 php_zip.dll（內建）也無 php_redis.dll
	php73Dir := filepath.Join(base, "bin", "php", "php-7.3.33")
	php73Ext := filepath.Join(php73Dir, "ext")
	if err := os.MkdirAll(php73Ext, 0755); err != nil {
		t.Fatalf("建立 php-7.3 ext 失敗: %v", err)
	}
	_ = os.WriteFile(filepath.Join(php73Ext, "php_gd2.dll"), []byte{}, 0644)

	got73 := ResolveEffectivePHPIni(base, php73Dir)
	if got73 == baseIni {
		t.Fatalf("PHP 7.3 應產生調整後的版本專屬 ini，不應直接回傳 %s", baseIni)
	}
	raw73, err := os.ReadFile(got73)
	if err != nil {
		t.Fatalf("讀取 PHP 7.3 ini 失敗: %v", err)
	}
	str73 := string(raw73)
	if !strings.Contains(str73, "extension=gd2") {
		t.Errorf("PHP 7.3 應將 extension=gd 轉為 extension=gd2，實際:\n%s", str73)
	}
	if strings.Contains(str73, "\nextension=zip") || strings.Contains(str73, "\nextension=redis") {
		t.Errorf("PHP 7.3 應註解掉缺失的 zip/redis，實際:\n%s", str73)
	}

	// 2. 模擬 PHP 8.2（齊全）：php_gd.dll + php_zip.dll + php_redis.dll，應直接復用原 php.ini
	php82Dir := filepath.Join(base, "bin", "php", "php-8.2.36")
	php82Ext := filepath.Join(php82Dir, "ext")
	if err := os.MkdirAll(php82Ext, 0755); err != nil {
		t.Fatalf("建立 php-8.2 ext 失敗: %v", err)
	}
	for _, dll := range []string{"php_gd.dll", "php_zip.dll", "php_redis.dll"} {
		_ = os.WriteFile(filepath.Join(php82Ext, dll), []byte{}, 0644)
	}
	if got82 := ResolveEffectivePHPIni(base, php82Dir); got82 != baseIni {
		t.Errorf("PHP 8.2 齊全時應直接復用 %s，實際回傳 %s", baseIni, got82)
	}

	// 3. 驗證 PreparePHPCLIRuntimeDir 會產生包含 extension_dir 與 openssl.cafile 的獨立 php.ini
	sslDir := filepath.Join(base, "conf", "ssl")
	_ = os.MkdirAll(sslDir, 0755)
	_ = os.WriteFile(filepath.Join(sslDir, "cacert.pem"), []byte("CA"), 0644)

	cliDir := PreparePHPCLIRuntimeDir(base, php73Dir)
	if cliDir == "" {
		t.Fatalf("PreparePHPCLIRuntimeDir 不應回傳空字串")
	}
	cliIniRaw, err := os.ReadFile(filepath.Join(cliDir, "php.ini"))
	if err != nil {
		t.Fatalf("讀取 CLI php.ini 失敗: %v", err)
	}
	cliIni := string(cliIniRaw)
	if !strings.Contains(cliIni, "extension=gd2") || !strings.Contains(cliIni, "extension_dir =") || !strings.Contains(cliIni, "openssl.cafile =") {
		t.Errorf("CLI php.ini 內容缺少預期設定:\n%s", cliIni)
	}
}
