package downloader_test

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"wincmp/internal/downloader"
)

func TestCalculateSHA256(t *testing.T) {
	// 建立臨時測試檔案
	tmpDir, err := os.MkdirTemp("", "wincmp-test-*")
	if err != nil {
		t.Fatalf("無法建立臨時目錄: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	content := []byte("hello-wincmp-sha256-test-content")
	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatalf("無法寫入測試檔案: %v", err)
	}

	// 計算預期的 SHA-256
	h := sha256.New()
	h.Write(content)
	expectedSHA := fmt.Sprintf("%x", h.Sum(nil))

	// 測試 CalculateSHA256
	actualSHA, err := downloader.CalculateSHA256(tmpFile)
	if err != nil {
		t.Fatalf("CalculateSHA256 失敗: %v", err)
	}

	if !strings.EqualFold(actualSHA, expectedSHA) {
		t.Errorf("SHA-256 不匹配! 預期: %s, 實際: %s", expectedSHA, actualSHA)
	}
}

func TestDownloadAndVerifySHA256(t *testing.T) {
	// 1. 建立 mock http server
	testContent := "dependency-mock-file-content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(testContent))
	}))
	defer server.Close()

	// 2. 建立臨時測試目錄與檔案路徑
	tmpDir, err := os.MkdirTemp("", "wincmp-test-*")
	if err != nil {
		t.Fatalf("無法建立臨時目錄: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	destPath := filepath.Join(tmpDir, "downloaded_dep.zip")

	// 3. 計算 mock content 的預期 SHA-256
	h := sha256.New()
	h.Write([]byte(testContent))
	correctSHA := fmt.Sprintf("%x", h.Sum(nil))
	wrongSHA := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

	// 4. 測試情況一：正確下載與校驗
	err = downloader.DownloadFile(server.URL, destPath, nil)
	if err != nil {
		t.Fatalf("下載失敗: %v", err)
	}

	// 校驗正確的 SHA-256
	shaVal, err := downloader.CalculateSHA256(destPath)
	if err != nil {
		t.Fatalf("計算下載檔案 SHA-256 失敗: %v", err)
	}
	if !strings.EqualFold(shaVal, correctSHA) {
		t.Errorf("下載檔案的 SHA-256 與預期不符! 預期: %s, 實際: %s", correctSHA, shaVal)
	}

	// 5. 測試情況二：校驗錯誤雜湊值
	if strings.EqualFold(shaVal, wrongSHA) {
		t.Errorf("模擬的錯誤 SHA-256 意外與正確雜湊值匹配")
	}
}
