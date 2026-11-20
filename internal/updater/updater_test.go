package updater_test

import (
	"os"
	"path/filepath"
	"testing"
	"wincmp/internal/updater"
)

func TestValidateReleaseURL(t *testing.T) {
	testCases := []struct {
		name      string
		url       string
		expectErr bool
	}{
		{
			name:      "官方合法 Release exe 下載連結",
			url:       "https://github.com/wukh1124/wincmp/releases/download/v2.1.4/WinCMP_v2.1.4.exe",
			expectErr: false,
		},
		{
			name:      "官方合法 Release zip 下載連結",
			url:       "https://github.com/wukh1124/wincmp/releases/download/v2.1.4/wincmp-v2.1.4-win-x64.zip",
			expectErr: false,
		},
		{
			name:      "第三方偽造域名",
			url:       "https://evil.com/releases/download/v2.1.4/WinCMP_v2.1.4.exe",
			expectErr: true,
		},
		{
			name:      "其他非官方 GitHub 倉庫",
			url:       "https://github.com/attacker/wincmp/releases/download/v2.1.4/WinCMP_v2.1.4.exe",
			expectErr: true,
		},
		{
			name:      "非 HTTPS 協議",
			url:       "http://github.com/wukh1124/wincmp/releases/download/v2.1.4/WinCMP_v2.1.4.exe",
			expectErr: true,
		},
		{
			name:      "空網址",
			url:       "",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := updater.ValidateReleaseURL(tc.url)
			if tc.expectErr && err == nil {
				t.Errorf("預期應報錯但未報錯: %s (%s)", tc.name, tc.url)
			}
			if !tc.expectErr && err != nil {
				t.Errorf("預期通過但報錯: %s (%s): %v", tc.name, tc.url, err)
			}
		})
	}
}

func TestValidateExecutablePE(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wincmp-updater-test-*")
	if err != nil {
		t.Fatalf("建立臨時目錄失敗: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 1. 合法 PE：前兩字節為 "MZ"，且大於 1MB
	validExePath := filepath.Join(tmpDir, "valid.exe")
	validData := make([]byte, 1024*1024+50) // 1MB + 50 bytes
	validData[0] = 'M'
	validData[1] = 'Z'
	if err := os.WriteFile(validExePath, validData, 0755); err != nil {
		t.Fatalf("寫入有效檔案失敗: %v", err)
	}

	if err := updater.ValidateExecutablePE(validExePath); err != nil {
		t.Errorf("有效 PE 檔驗證失敗: %v", err)
	}

	// 2. 缺少 MZ 標頭的偽造檔 (如 HTML 錯誤頁)
	invalidHeaderPath := filepath.Join(tmpDir, "invalid_header.exe")
	invalidHeaderData := make([]byte, 1024*1024+50)
	copy(invalidHeaderData, []byte("<!DOCTYPE html><html>"))
	if err := os.WriteFile(invalidHeaderPath, invalidHeaderData, 0755); err != nil {
		t.Fatalf("寫入無效標頭檔案失敗: %v", err)
	}

	if err := updater.ValidateExecutablePE(invalidHeaderPath); err == nil {
		t.Errorf("無效標頭檔案預期應報錯但未報錯")
	}

	// 3. 檔案過小 (例如 404 文字錯誤)
	tooSmallPath := filepath.Join(tmpDir, "too_small.exe")
	tooSmallData := []byte("MZ error 404 not found")
	if err := os.WriteFile(tooSmallPath, tooSmallData, 0755); err != nil {
		t.Fatalf("寫入過小檔案失敗: %v", err)
	}

	if err := updater.ValidateExecutablePE(tooSmallPath); err == nil {
		t.Errorf("過小檔案預期應報錯但未報錯")
	}
}

func TestParseChecksumsContent(t *testing.T) {
	rawChecksums := `
# Official WinCMP Release Checksums
1708333f79e274c7697285afe6d592ab39314e0b131e9ec6bea08ad27df62ebf  WinCMP_v2.1.4.exe
fb7c76f0804321ee373daa49145f2056d2d88f321b614130adeb05a1644ea003 *wincmp-v2.1.4-win-x64.zip
invalid-line
`
	res := updater.ParseChecksumsContent(rawChecksums)

	expectedExeHash := "1708333f79e274c7697285afe6d592ab39314e0b131e9ec6bea08ad27df62ebf"
	expectedZipHash := "fb7c76f0804321ee373daa49145f2056d2d88f321b614130adeb05a1644ea003"

	if res["wincmp_v2.1.4.exe"] != expectedExeHash {
		t.Errorf("解析 exe hash 不符! 預期: %s, 實際: %s", expectedExeHash, res["wincmp_v2.1.4.exe"])
	}

	if res["wincmp-v2.1.4-win-x64.zip"] != expectedZipHash {
		t.Errorf("解析 zip hash 不符! 預期: %s, 實際: %s", expectedZipHash, res["wincmp-v2.1.4-win-x64.zip"])
	}

	if len(res) != 2 {
		t.Errorf("解析數量不符! 預期 2 個，實際得到: %d", len(res))
	}
}

func TestRollbackUpdateWithPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wincmp-rollback-test-*")
	if err != nil {
		t.Fatalf("建立臨時目錄失敗: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	execPath := filepath.Join(tmpDir, "wincmp.exe")
	oldPath := execPath + ".old"
	newExePath := filepath.Join(tmpDir, "wincmp_new.exe")

	// 模擬場景：舊版已被重命名為 .old，新版 exe 存在
	oldContent := []byte("old-binary-content")
	newContent := []byte("new-corrupted-binary")
	if err := os.WriteFile(oldPath, oldContent, 0755); err != nil {
		t.Fatalf("寫入 old 檔案失敗: %v", err)
	}
	if err := os.WriteFile(newExePath, newContent, 0755); err != nil {
		t.Fatalf("寫入 new 檔案失敗: %v", err)
	}

	// 執行回滾
	if err := updater.RollbackUpdateWithPath(execPath, newExePath); err != nil {
		t.Fatalf("回滾執行失敗: %v", err)
	}

	// 驗證 1：新版 exe 應被刪除
	if _, err := os.Stat(newExePath); !os.IsNotExist(err) {
		t.Errorf("預期新版 exe 應被刪除，但仍存在")
	}

	// 驗證 2：原本的 .old 應被還原為 wincmp.exe
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("預期 .old 應被還原，但仍存在")
	}

	restoredContent, err := os.ReadFile(execPath)
	if err != nil {
		t.Fatalf("無法讀取還原後的執行檔: %v", err)
	}
	if string(restoredContent) != string(oldContent) {
		t.Errorf("還原後內容不符! 預期: %s, 實際: %s", string(oldContent), string(restoredContent))
	}
}

func TestReplaceExecutableFixedName(t *testing.T) {
	t.Run("固定檔名原地升級", func(t *testing.T) {
		tmpDir := t.TempDir()
		execPath := filepath.Join(tmpDir, updater.FixedExecutableName)
		tempExePath := filepath.Join(tmpDir, "wincmp_new.exe")

		if err := os.WriteFile(execPath, []byte("old-v1"), 0755); err != nil {
			t.Fatalf("寫入舊執行檔失敗: %v", err)
		}
		if err := os.WriteFile(tempExePath, []byte("new-v2"), 0755); err != nil {
			t.Fatalf("寫入新執行檔失敗: %v", err)
		}

		newExePath, err := updater.ReplaceExecutable(execPath, tempExePath)
		if err != nil {
			t.Fatalf("替換執行檔失敗: %v", err)
		}
		if newExePath != execPath {
			t.Errorf("預期固定落地路徑為 %s，實際為 %s", execPath, newExePath)
		}

		gotNew, err := os.ReadFile(newExePath)
		if err != nil || string(gotNew) != "new-v2" {
			t.Errorf("新執行檔內容不符或讀取失敗: %q, err=%v", string(gotNew), err)
		}
		gotOld, err := os.ReadFile(execPath + ".old")
		if err != nil || string(gotOld) != "old-v1" {
			t.Errorf("舊備份檔內容不符或讀取失敗: %q, err=%v", string(gotOld), err)
		}
	})

	t.Run("舊版帶版號檔名過渡收斂為 WinCMP.exe", func(t *testing.T) {
		tmpDir := t.TempDir()
		execPath := filepath.Join(tmpDir, "WinCMP_v2.1.9.exe")
		tempExePath := filepath.Join(tmpDir, "wincmp_new.exe")

		if err := os.WriteFile(execPath, []byte("old-versioned"), 0755); err != nil {
			t.Fatalf("寫入舊執行檔失敗: %v", err)
		}
		if err := os.WriteFile(tempExePath, []byte("new-fixed"), 0755); err != nil {
			t.Fatalf("寫入新執行檔失敗: %v", err)
		}

		newExePath, err := updater.ReplaceExecutable(execPath, tempExePath)
		if err != nil {
			t.Fatalf("替換執行檔失敗: %v", err)
		}
		expectedPath := filepath.Join(tmpDir, updater.FixedExecutableName)
		if newExePath != expectedPath {
			t.Errorf("預期收斂為 %s，實際為 %s", expectedPath, newExePath)
		}
		if _, err := os.Stat(execPath + ".old"); err != nil {
			t.Errorf("預期舊版備份檔 %s 存在: %v", execPath+".old", err)
		}
	})

	t.Run("固定檔名啟動失敗時回滾", func(t *testing.T) {
		tmpDir := t.TempDir()
		execPath := filepath.Join(tmpDir, updater.FixedExecutableName)
		oldPath := execPath + ".old"

		if err := os.WriteFile(oldPath, []byte("old-good"), 0755); err != nil {
			t.Fatalf("寫入 old 檔失敗: %v", err)
		}
		if err := os.WriteFile(execPath, []byte("new-broken"), 0755); err != nil {
			t.Fatalf("寫入新檔失敗: %v", err)
		}

		if err := updater.RollbackUpdateWithPath(execPath, execPath); err != nil {
			t.Fatalf("同名回滾失敗: %v", err)
		}
		got, err := os.ReadFile(execPath)
		if err != nil || string(got) != "old-good" {
			t.Errorf("回滾後內容不符: %q, err=%v", string(got), err)
		}
		if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
			t.Errorf("預期 .old 檔案已還原並移除")
		}
	})
}

