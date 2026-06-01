package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDependencies_NonExistent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "wincmp_test")
	if err != nil {
		t.Fatalf("無法建立暫存目錄: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testPath := filepath.Join(tempDir, "dependencies.json")

	// 測試載入不存在的設定檔，預期會自動建立並回傳預設值
	cfg, err := LoadDependencies(testPath)
	if err != nil {
		t.Fatalf("載入依賴設定檔失敗: %v", err)
	}

	if len(cfg) == 0 {
		t.Error("回傳的依賴設定檔內容為空")
	}

	// 驗證是否有特定預設值
	caddy, ok := cfg["caddy"]
	if !ok {
		t.Error("預設設定缺少 caddy 項")
	} else if caddy.Version != "2.7.6" {
		t.Errorf("預期 caddy 版本為 2.7.6，實際為 %s", caddy.Version)
	}

	// 驗證檔案是否真的寫入磁碟了
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Error("設定檔未成功寫入磁碟")
	}
}

func TestSaveAndLoadDependencies(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "wincmp_test")
	if err != nil {
		t.Fatalf("無法建立暫存目錄: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testPath := filepath.Join(tempDir, "dependencies.json")

	customCfg := DependencyConfig{
		"caddy": DependencyItem{
			Version: "2.8.0",
			URL:     "https://example.com/caddy.zip",
		},
	}

	// 儲存設定
	if err := SaveDependencies(testPath, customCfg); err != nil {
		t.Fatalf("儲存設定失敗: %v", err)
	}

	// 重新載入設定
	loaded, err := LoadDependencies(testPath)
	if err != nil {
		t.Fatalf("載入設定失敗: %v", err)
	}

	caddy, ok := loaded["caddy"]
	if !ok {
		t.Fatal("缺少 caddy 項")
	}

	if caddy.Version != "2.8.0" || caddy.URL != "https://example.com/caddy.zip" {
		t.Errorf("載入的資料與儲存的不符: %+v", caddy)
	}
}

func TestLoadDependencies_Corrupted(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "wincmp_test")
	if err != nil {
		t.Fatalf("無法建立暫存目錄: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testPath := filepath.Join(tempDir, "dependencies.json")
	if err := os.WriteFile(testPath, []byte("{invalid json"), 0644); err != nil {
		t.Fatalf("無法寫入測試檔案: %v", err)
	}

	_, err = LoadDependencies(testPath)
	if err == nil {
		t.Error("預期解析損壞的 JSON 會回傳錯誤，但卻沒有")
	}
}

func TestSaveDependencies_InvalidPath(t *testing.T) {
	err := SaveDependencies("", DefaultDependencies)
	if err == nil {
		t.Error("預期在非法路徑下儲存會回傳錯誤，但卻沒有")
	}
}

