//go:build windows
package terminal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveWorkDir_InvalidPreferredFallsBack(t *testing.T) {
	base, err := os.MkdirTemp("", "wincmp-term-base-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(base)

	www := filepath.Join(base, "www")
	if err := os.MkdirAll(www, 0o755); err != nil {
		t.Fatal(err)
	}

	missing := filepath.Join(base, "not-exist-project")
	got, err := ResolveWorkDir(missing, base)
	if err != nil {
		t.Fatalf("ResolveWorkDir should fallback, got err: %v", err)
	}
	if got != www {
		// On Windows Abs may normalize separators
		if filepath.Clean(got) != filepath.Clean(www) {
			t.Fatalf("expected fallback www dir, got %s (want %s)", got, www)
		}
	}
}

func TestResolveWorkDir_ValidPreferred(t *testing.T) {
	base, err := os.MkdirTemp("", "wincmp-term-proj-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(base)

	proj := filepath.Join(base, "site")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := ResolveWorkDir(proj, base)
	if err != nil {
		t.Fatalf("ResolveWorkDir failed: %v", err)
	}
	if filepath.Clean(got) != filepath.Clean(proj) {
		t.Fatalf("expected project dir %s, got %s", proj, got)
	}
}

func TestResolveShell_Default(t *testing.T) {
	got, err := ResolveShell("")
	if err != nil {
		t.Fatalf("ResolveShell empty failed: %v", err)
	}
	if !strings.Contains(strings.ToLower(got), "powershell") && !strings.Contains(strings.ToLower(got), "cmd") {
		t.Fatalf("unexpected shell path: %s", got)
	}
}

func TestStartTerminal_InvalidCwdDoesNotPanic(t *testing.T) {
	mgr := NewManager()
	defer mgr.StopAll()

	noop := func(data string) {}
	noopExit := func() {}

	// 模擬專案路徑不存在：StartTerminal 應自動回退，不應回傳 directory name is invalid
	id, err := mgr.StartTerminal(
		"missing-project",
		"cmd.exe",
		filepath.Join(os.TempDir(), "wincmp-missing-cwd-should-fallback"),
		80,
		24,
		noop,
		noopExit,
	)
	if err != nil {
		t.Fatalf("StartTerminal with invalid cwd should fallback, got: %v", err)
	}
	if id == "" {
		t.Fatal("empty session id")
	}
}
