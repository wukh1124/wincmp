package scanner

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestScanBinDirPHPVersionOrdering 驗證同分支存在多個 patch 目錄時，保留數值上較新的版本。
// 回歸：舊實作以字串比較版本，會把 "8.4.9" 誤判為比 "8.4.10" 新。
func TestScanBinDirPHPVersionOrdering(t *testing.T) {
	base := t.TempDir()
	for _, v := range []string{"8.4.9", "8.4.10"} {
		dir := filepath.Join(base, "bin", "php", "php-"+v)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("建立 PHP 目錄失敗: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "php-cgi.exe"), []byte{}, 0644); err != nil {
			t.Fatalf("建立 php-cgi.exe 失敗: %v", err)
		}
	}

	// 走內部函式以繞過 5 分鐘掃描快取，確保測試互不污染
	res, err := scanBinDirInternal(base)
	if err != nil {
		t.Fatalf("掃描 bin 目錄失敗: %v", err)
	}
	if len(res.PHPList) != 1 {
		t.Fatalf("同分支應只保留一個版本，實際保留 %d 個", len(res.PHPList))
	}
	if got := res.PHPList[0].Version; got != "8.4.10" {
		t.Errorf("應保留較新的 8.4.10，實際保留 %s", got)
	}
	if len(res.SkippedPHP) != 1 || res.SkippedPHP[0] != "8.4.9" {
		t.Errorf("應記錄被略過的 8.4.9，實際為 %v", res.SkippedPHP)
	}
}

// TestCompareVersions 驗證版本比較為數值分段，而非字串比較
func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a    string
		b    string
		want int
	}{
		{"8.4.9", "8.4.10", -1},
		{"8.5.12", "8.4.27", 1},
		{"8.3.36", "8.3.36", 0},
		{"v8.5.12", "8.5.12", 0},
	}
	for _, c := range cases {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Errorf("compareVersions(%q, %q) = %d, 預期 %d", c.a, c.b, got, c.want)
		}
	}
}

// TestScanBinDirConcurrentAndInvalidate 驗證併發呼叫 ScanBinDir 安全且 InvalidateScanCache 可立即刷新結果
func TestScanBinDirConcurrentAndInvalidate(t *testing.T) {
	base := t.TempDir()
	caddyDir := filepath.Join(base, "bin", "caddy", "caddy-2.9.0")
	if err := os.MkdirAll(caddyDir, 0755); err != nil {
		t.Fatalf("建立 Caddy 目錄失敗: %v", err)
	}
	if err := os.WriteFile(filepath.Join(caddyDir, "caddy.exe"), []byte{}, 0644); err != nil {
		t.Fatalf("建立 caddy.exe 失敗: %v", err)
	}

	InvalidateScanCache()
	defer InvalidateScanCache()

	const workers = 6
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	versions := make(chan string, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := ScanBinDir(base)
			if err != nil {
				errs <- err
				return
			}
			if len(res.CaddyList) != 1 {
				errs <- os.ErrNotExist
				return
			}
			versions <- res.CaddyList[0].Version
		}()
	}
	wg.Wait()
	close(errs)
	close(versions)

	for err := range errs {
		t.Fatalf("併發 ScanBinDir 失敗: %v", err)
	}
	for v := range versions {
		if v != "2.9.0" {
			t.Errorf("預期掃描到 2.9.0，實際為 %s", v)
		}
	}

	// 新增一個 PHP 版本並失效快取後，應立即掃描到新版本
	phpDir := filepath.Join(base, "bin", "php", "php-8.4.11")
	if err := os.MkdirAll(phpDir, 0755); err != nil {
		t.Fatalf("建立 PHP 目錄失敗: %v", err)
	}
	if err := os.WriteFile(filepath.Join(phpDir, "php-cgi.exe"), []byte{}, 0644); err != nil {
		t.Fatalf("建立 php-cgi.exe 失敗: %v", err)
	}

	InvalidateScanCache()
	res, err := ScanBinDir(base)
	if err != nil {
		t.Fatalf("失效快取後 ScanBinDir 失敗: %v", err)
	}
	if len(res.PHPList) != 1 || res.PHPList[0].Version != "8.4.11" {
		t.Errorf("預期失效快取後掃描到 PHP 8.4.11，實際為 %+v", res.PHPList)
	}
}

// TestScanBinDirComposerPharAndBat 驗證同時支援 composer.phar 與 composer.bat 的多版本掃描與降序排序
func TestScanBinDirComposerPharAndBat(t *testing.T) {
	base := t.TempDir()
	pharDir := filepath.Join(base, "bin", "composer", "composer-2.11.2")
	batDir := filepath.Join(base, "bin", "composer", "composer-2.2.25")
	if err := os.MkdirAll(pharDir, 0755); err != nil {
		t.Fatalf("建立 composer-2.11.2 目錄失敗: %v", err)
	}
	if err := os.MkdirAll(batDir, 0755); err != nil {
		t.Fatalf("建立 composer-2.2.25 目錄失敗: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pharDir, "composer.phar"), []byte{}, 0644); err != nil {
		t.Fatalf("建立 composer.phar 失敗: %v", err)
	}
	if err := os.WriteFile(filepath.Join(batDir, "composer.bat"), []byte{}, 0644); err != nil {
		t.Fatalf("建立 composer.bat 失敗: %v", err)
	}

	res, err := scanBinDirInternal(base)
	if err != nil {
		t.Fatalf("掃描 Composer 失敗: %v", err)
	}
	if len(res.ComposerList) != 2 {
		t.Fatalf("預期掃描到 2 個 Composer 版本，實際為 %d", len(res.ComposerList))
	}
	if res.ComposerList[0].Version != "2.11.2" || res.ComposerList[1].Version != "2.2.25" {
		t.Errorf("預期依版本由新到舊排序 [2.11.2, 2.2.25]，實際為 %+v", res.ComposerList)
	}
}
