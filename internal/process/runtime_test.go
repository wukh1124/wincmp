package process

import (
	"strings"
	"testing"
)

func TestGetPortOccupiedProcess_SecurityProtection(t *testing.T) {
	// 1. 測試小於等於 1024 端口
	_, err := GetPortOccupiedProcess(80)
	if err == nil {
		t.Error("預期端口 80 應該被安全保護拒絕，但沒有返回 error")
	} else if !strings.Contains(err.Error(), "系統安全保護") {
		t.Errorf("錯誤訊息不符: %v", err)
	}

	_, err = GetPortOccupiedProcess(1024)
	if err == nil {
		t.Error("預期端口 1024 應該被安全保護拒絕，但沒有返回 error")
	}

	// 2. 測試 3389 端口
	_, err = GetPortOccupiedProcess(3389)
	if err == nil {
		t.Error("預期端口 3389 應該被安全保護拒絕，但沒有返回 error")
	}

	// 3. 測試大於 1024 且非 3389 的安全端口，應該正常進行，不會被安全保護攔截
	_, err = GetPortOccupiedProcess(3000)
	if err != nil && strings.Contains(err.Error(), "系統安全保護") {
		t.Errorf("預期端口 3000 不會觸發安全保護，但被攔截了: %v", err)
	}
}

func TestKillProcessByPort_SecurityProtection(t *testing.T) {
	// 1. 測試小於等於 1024 端口
	err := KillProcessByPort(443)
	if err == nil {
		t.Error("預期關閉端口 443 應該被安全保護拒絕，但沒有返回 error")
	} else if !strings.Contains(err.Error(), "系統安全保護") {
		t.Errorf("錯誤訊息不符: %v", err)
	}

	// 2. 測試 3389 端口
	err = KillProcessByPort(3389)
	if err == nil {
		t.Error("預期關閉端口 3389 應該被安全保護拒絕，但沒有返回 error")
	}
}

func TestSanitizeRuntimeCommand_StartPrefixBlocking(t *testing.T) {
	validCmds := []string{
		"npm run dev",
		"bun run dev",
		"python app.py",
		"go run main.go",
		`"C:\Program Files\nodejs\npm.cmd" run dev`,
	}
	for _, cmd := range validCmds {
		if _, err := sanitizeRuntimeCommand(cmd); err != nil {
			t.Errorf("合法指令 %s 應通過驗證，但被攔截: %v", cmd, err)
		}
	}

	blockedCmds := []string{
		"start npm run dev",
		"START npm run dev",
		"  start vite",
		"start",
	}
	for _, cmd := range blockedCmds {
		if _, err := sanitizeRuntimeCommand(cmd); err == nil {
			t.Errorf("指令 %s 應該被攔截，但通過了驗證", cmd)
		} else if !strings.Contains(err.Error(), "start") {
			t.Errorf("指令 %s 錯誤訊息不符合預期: %v", cmd, err)
		}
	}
}

func TestSanitizeRuntimeCommand_ShellMetacharBlocking(t *testing.T) {
	blockedCmds := []string{
		"npm run dev & calc",
		"npm run dev | more",
		"npm run dev; whoami",
		"npm run dev > out.txt",
		"npm run dev < in.txt",
		"npm run dev $env",
		"npm run dev `whoami`",
		"npm run dev (echo hi)",
		`npm run "dev&calc"`,
		`"C:\path\app.exe" run & calc`,
		`npm run dev"`, // 不成對引號
	}
	for _, cmd := range blockedCmds {
		if _, err := sanitizeRuntimeCommand(cmd); err == nil {
			t.Errorf("危險指令 %s 應被攔截，但通過了驗證", cmd)
		}
	}
}

func TestSanitizeRuntimeCommand_NulRedirectAllowed(t *testing.T) {
	// 系統內部會組出 chcp 65001 >nul，自訂指令若含 >nul 亦應放行
	cmds := []string{
		"chcp 65001 >nul && npm run dev", // 注意：此含 && 仍應被拒
		"npm run dev >nul",
	}
	if _, err := sanitizeRuntimeCommand(cmds[1]); err != nil {
		t.Errorf(">nul 應被允許: %v", err)
	}
	if _, err := sanitizeRuntimeCommand(cmds[0]); err == nil {
		t.Error("含 && 的指令即使有 >nul 也應被攔截")
	}
}

func TestSanitizeRuntimeCommand_DetachedLaunchBlocking(t *testing.T) {
	blockedCmds := []string{
		"cmd /c start npm run dev",
		"cmd.exe /c start npm run dev",
		"cmd /k start vite",
		`cmd /c "start npm run dev"`,
		"powershell -Command Start-Process npm -ArgumentList run,dev",
		"powershell.exe Start-Process bun",
		"pwsh -c Start-Process npm",
	}
	for _, cmd := range blockedCmds {
		if _, err := sanitizeRuntimeCommand(cmd); err == nil {
			t.Errorf("脫鉤指令 %s 應被攔截，但通過了驗證", cmd)
		}
	}
}

func TestFirstCommandToken(t *testing.T) {
	cases := map[string]string{
		"npm run dev": "npm",
		`"C:\Program Files\nodejs\npm.cmd" run dev`: `C:\Program Files\nodejs\npm.cmd`,
		`  bun run dev`: "bun",
		``:               "",
	}
	for input, want := range cases {
		if got := firstCommandToken(input); got != want {
			t.Errorf("firstCommandToken(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestIsPortAvailable_BindSemantics(t *testing.T) {
	// 未被佔用的高位端口應可用；此測試不保證環境絕對空閒，僅驗證函式可呼叫且回傳合理型別
	// 使用極高埠降低與既有服務衝突機率
	available := IsPortAvailable(59999)
	if !available && IsPortAvailable(59999) && FindPIDByPort(59999) == 0 {
		t.Log("port 59999 bind 失敗但無 LISTEN PID，屬環境特殊狀態")
	}
}
