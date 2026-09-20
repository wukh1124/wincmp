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
