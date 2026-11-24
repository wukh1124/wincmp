//go:build windows

package singleinstance

import (
	"path/filepath"
	"testing"
	"time"
)

func TestIsWinCMPExecutableInDir(t *testing.T) {
	dir := filepath.Join(`C:\Tools`, "WinCMP")

	tests := []struct {
		name      string
		imagePath string
		want      bool
	}{
		{"Fixed WinCMP.exe", filepath.Join(dir, "WinCMP.exe"), true},
		{"Case-insensitive WinCMP.exe", filepath.Join(`c:\tools\wincmp`, "wincmp.EXE"), true},
		{"Legacy versioned WinCMP", filepath.Join(dir, "WinCMP_v2.1.9.exe"), true},
		{"Renamed old backup", filepath.Join(dir, "WinCMP_v2.1.9.exe.old"), true},
		{"Fixed old backup", filepath.Join(dir, "WinCMP.exe.old"), true},
		{"Subdirectory php-cgi", filepath.Join(dir, "bin", "php", "php-7.3.33", "php-cgi.exe"), false},
		{"Subdirectory WinCMP", filepath.Join(dir, "sub", "WinCMP.exe"), false},
		{"Different directory", filepath.Join(`C:\Other`, "WinCMP.exe"), false},
		{"Unrelated exe in same dir", filepath.Join(dir, "other.exe"), false},
		{"Empty path", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isWinCMPExecutableInDir(tc.imagePath, dir); got != tc.want {
				t.Errorf("isWinCMPExecutableInDir(%q, %q) = %v, want %v", tc.imagePath, dir, got, tc.want)
			}
		})
	}
}

func TestWaitForSiblingWinCMPExit_NoSiblingReturnsImmediately(t *testing.T) {
	dir := t.TempDir()
	start := time.Now()
	WaitForSiblingWinCMPExit(dir, 2*time.Second)
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Errorf("無兄弟進程時應立即返回，實際耗時 %v", elapsed)
	}
}
