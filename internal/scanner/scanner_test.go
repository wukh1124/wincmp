package scanner

import (
	"os"
	"path/filepath"
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
