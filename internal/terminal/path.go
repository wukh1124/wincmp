package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ResolveWorkDir 解析可安全用於 ConPTY 建立 console 的工作目錄。
// Windows CreateProcess 在目錄不存在時會回傳 "The directory name is invalid"，
// 因此啟動前必須確認 cwd 真實存在。
func ResolveWorkDir(preferred string, baseDir string) (string, error) {
	candidates := make([]string, 0, 6)

	if p := strings.TrimSpace(preferred); p != "" {
		candidates = append(candidates, p)
	}
	if b := strings.TrimSpace(baseDir); b != "" {
		candidates = append(candidates,
			filepath.Join(b, "www"),
			b,
		)
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidates = append(candidates, home)
	}
	if tmp := os.TempDir(); tmp != "" {
		candidates = append(candidates, tmp)
	}

	tried := make([]string, 0, len(candidates))
	for _, dir := range candidates {
		if dir == "" {
			continue
		}
		abs, err := filepath.Abs(dir)
		if err != nil {
			continue
		}
		info, err := os.Stat(abs)
		if err != nil || !info.IsDir() {
			tried = append(tried, abs)
			continue
		}
		return abs, nil
	}

	return "", fmt.Errorf("找不到可用的終端工作目錄 (嘗試: %s)", strings.Join(tried, "; "))
}

// ResolveShell 解析終端 shell 的絕對路徑；找不到設定值時回退到 powershell / cmd。
func ResolveShell(shell string) (string, error) {
	shell = strings.TrimSpace(shell)
	if shell == "" {
		shell = "powershell.exe"
	}

	if filepath.IsAbs(shell) {
		if _, err := os.Stat(shell); err == nil {
			return shell, nil
		}
	} else if p, err := exec.LookPath(shell); err == nil && p != "" {
		return p, nil
	}

	fallbacks := []string{
		`C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`,
		`C:\Windows\System32\cmd.exe`,
	}
	for _, f := range fallbacks {
		if _, err := os.Stat(f); err == nil {
			return f, nil
		}
	}

	return "", fmt.Errorf("找不到終端 Shell: %s", shell)
}
