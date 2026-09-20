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
	// 1. 合法正常指令
	validCmds := []string{
		"npm run dev",
		"bun run dev",
		"python app.py",
		"go run main.go",
	}
	for _, cmd := range validCmds {
		if _, err := sanitizeRuntimeCommand(cmd); err != nil {
			t.Errorf("合法指令 %s 應通過驗證，但被攔截: %v", cmd, err)
		}
	}

	// 2. 包含 start 前綴的脫鉤指令
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

