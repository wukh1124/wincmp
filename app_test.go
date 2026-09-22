package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"


	"gopkg.in/natefinch/lumberjack.v2"
	"wincmp/internal/config"
)

func TestCleanExpiredLogs(t *testing.T) {
	// 1. 建立測試用的臨時目錄
	tempDir, err := os.MkdirTemp("", "wincmp_log_test")
	if err != nil {
		t.Fatalf("無法建立臨時目錄: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logDir := filepath.Join(tempDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("無法建立 logs 目錄: %v", err)
	}

	// 2. 準備測試檔案名稱
	now := time.Now()
	expiredDateStr := now.AddDate(0, 0, -10).Format("2006-01-02") // 10 天前 (過期)
	validDateStr := now.AddDate(0, 0, -2).Format("2006-01-02")    // 2 天前 (未過期)

	testFiles := []struct {
		name      string
		shouldKeep bool
	}{
		// 系統日誌 (過期/未過期)
		{fmt.Sprintf("wincmp-caddy-%s.log", expiredDateStr), false},
		{fmt.Sprintf("wincmp-caddy-%s.log", validDateStr), true},
		// 錯誤日誌 (過期/未過期)
		{fmt.Sprintf("error-%s.log", expiredDateStr), false},
		{fmt.Sprintf("error-%s.log", validDateStr), true},
		// 專案日誌 (過期/未過期)
		{fmt.Sprintf("runtime-astro-sample-%s.log", expiredDateStr), false},
		{fmt.Sprintf("runtime-astro-sample-%s.log", validDateStr), true},
		// 一般日誌檔 (不符日期格式)
		{"access.log", true},
		{"caddy.log", true},
		// 其他檔案
		{"readme.txt", true},
		{"wincmp-invalid-date.log", true},
	}

	for _, tf := range testFiles {
		filePath := filepath.Join(logDir, tf.name)
		if err := os.WriteFile(filePath, []byte("test log content"), 0644); err != nil {
			t.Fatalf("無法建立測試檔案 %s: %v", tf.name, err)
		}
	}

	// 3. 建立 App 實例並執行 cleanExpiredLogs
	app := &App{
		baseDir: tempDir,
		appCfg: &config.WincmpConfig{
			Global: config.GlobalConfig{
				MaxLogRetention: 7, // 保存 7 天
			},
		},
		runtimeLogWriters: make(map[string]*lumberjack.Logger),
	}

	app.cleanExpiredLogs()

	// 4. 驗證檔案是否被正確刪除或保留
	for _, tf := range testFiles {
		filePath := filepath.Join(logDir, tf.name)
		_, err := os.Stat(filePath)
		exists := err == nil

		if tf.shouldKeep && !exists {
			t.Errorf("檔案 %s 應該要被保留，但它被刪除了", tf.name)
		}
		if !tf.shouldKeep && exists {
			t.Errorf("檔案 %s 已經過期，但它沒有被刪除", tf.name)
		}
	}
}

func TestNewAppBaseDir(t *testing.T) {
	app := NewApp()
	if app.baseDir == "" {
		t.Errorf("NewApp() 的 baseDir 不應為空")
	}
}

func TestOpenFolderAndShowInExplorerValidation(t *testing.T) {
	app := NewApp()

	// 1. 測試空路徑
	if err := app.OpenFolder(""); err == nil {
		t.Errorf("OpenFolder(\"\") 應該要報錯")
	}
	if err := app.ShowInExplorer(""); err == nil {
		t.Errorf("ShowInExplorer(\"\") 應該要報錯")
	}

	// 2. 測試完全不存在的路徑
	nonExistent := filepath.Join(os.TempDir(), "non_existent_dir_123456789", "file.log")
	if err := app.OpenFolder(nonExistent); err == nil {
		t.Errorf("OpenFolder 不存在的目錄應該報錯")
	}
	if err := app.ShowInExplorer(nonExistent); err == nil {
		t.Errorf("ShowInExplorer 不存在的父目錄檔案應該報錯")
	}

	// 3. 測試目標檔案不存在但父目錄存在時，ShowInExplorer 能否安全降級到父目錄
	tempDir, err := os.MkdirTemp("", "wincmp_show_test_*")
	if err != nil {
		t.Fatalf("無法建立臨時目錄: %v", err)
	}
	defer os.RemoveAll(tempDir)

	nonExistentFileInValidDir := filepath.Join(tempDir, "missing.log")
	// 該檔案不存在，但父目錄存在，ShowInExplorer 應能降級開啟父目錄（透過 ShellExecute 不報錯）
	if err := app.ShowInExplorer(nonExistentFileInValidDir); err != nil {
		t.Errorf("ShowInExplorer 對於父目錄存在的缺失檔案應安全降級，但回傳錯誤: %v", err)
	}
}



