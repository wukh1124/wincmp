package process

import (
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"
)

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	time.Sleep(30 * time.Second)
	os.Exit(0)
}

func startHelperCmd(t *testing.T) *exec.Cmd {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("取得測試執行檔路徑失敗: %v", err)
	}
	cmd := exec.Command(exe, "-test.run=TestHelperProcess")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("啟動測試子行程失敗: %v", err)
	}
	return cmd
}

type capturedErrLog struct {
	category string
	msg      string
	err      error
}

func newTestManager(t *testing.T) (*Manager, *[]capturedErrLog, *sync.Mutex) {
	t.Helper()
	var mu sync.Mutex
	var errLogs []capturedErrLog
	m := NewManager(t.TempDir(), func(category, msg string) {}, func(category, contextMsg string, err error) {
		mu.Lock()
		errLogs = append(errLogs, capturedErrLog{category: category, msg: contextMsg, err: err})
		mu.Unlock()
	})
	return m, &errLogs, &mu
}

func TestStopService_NoFalseAbnormalExitLog(t *testing.T) {
	m, errLogs, mu := newTestManager(t)
	cmd := startHelperCmd(t)
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	})

	const key = "caddy"
	state := m.register(key, "Caddy", "", []*exec.Cmd{cmd})

	waitDone := make(chan struct{})
	go func() {
		m.waitForExit(cmd, key, state, "caddy", "Caddy")
		close(waitDone)
	}()

	if err := m.stopService(key); err != nil {
		t.Fatalf("stopService 回傳非預期錯誤: %v", err)
	}

	select {
	case <-waitDone:
	case <-time.After(3 * time.Second):
		t.Fatal("等待 waitForExit 結束逾時")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(*errLogs) != 0 {
		t.Fatalf("主動停止不應產生錯誤日誌，實際收到: %+v", *errLogs)
	}
	if m.IsRunning(key) {
		t.Fatal("停止後服務不應仍標記為運行中")
	}
}

func TestWaitForExit_LogsErrorOnUnexpectedKill(t *testing.T) {
	m, errLogs, mu := newTestManager(t)
	cmd := startHelperCmd(t)
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	})

	const key = "caddy"
	state := m.register(key, "Caddy", "", []*exec.Cmd{cmd})

	waitDone := make(chan struct{})
	go func() {
		m.waitForExit(cmd, key, state, "caddy", "Caddy")
		close(waitDone)
	}()

	// 模擬外部異常終止（未經 stopService）
	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("終止測試子行程失敗: %v", err)
	}

	select {
	case <-waitDone:
	case <-time.After(3 * time.Second):
		t.Fatal("等待 waitForExit 結束逾時")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(*errLogs) != 1 {
		t.Fatalf("外部異常終止應記錄 1 筆錯誤日誌，實際收到: %+v", *errLogs)
	}
	if m.IsRunning(key) {
		t.Fatal("子行程異常退出後應自動註銷服務")
	}
}

func TestMonitorPHPCGIProcesses_ManualStopNoFalseErrors(t *testing.T) {
	m, errLogs, mu := newTestManager(t)
	const count = 4
	cmds := make([]*exec.Cmd, 0, count)
	ports := make([]int, 0, count)
	for i := 0; i < count; i++ {
		cmd := startHelperCmd(t)
		cmds = append(cmds, cmd)
		ports = append(ports, 38500+i)
	}
	t.Cleanup(func() {
		for _, c := range cmds {
			if c.Process != nil {
				_ = c.Process.Kill()
			}
		}
	})

	const key = "php-8.5.13"
	state := m.register(key, "PHP-CGI 8.5.13", "", cmds)
	m.monitorPHPCGIProcesses(key, "8.5.13", state, cmds, ports)

	if err := m.stopService(key); err != nil {
		t.Fatalf("stopService 回傳非預期錯誤: %v", err)
	}

	// 等待子行程完全被 OS 回收與監控 goroutine 返回
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(*errLogs) != 0 {
		t.Fatalf("手動停止 PHP-CGI 不應產生異常退出警告，實際收到: %+v", *errLogs)
	}
}

func TestMonitorPHPCGIProcesses_LogsErrorOnUnexpectedCrash(t *testing.T) {
	m, errLogs, mu := newTestManager(t)
	cmds := []*exec.Cmd{startHelperCmd(t), startHelperCmd(t)}
	ports := []int{38500, 38501}
	t.Cleanup(func() {
		for _, c := range cmds {
			if c.Process != nil {
				_ = c.Process.Kill()
			}
		}
	})

	const key = "php-8.5.13"
	state := m.register(key, "PHP-CGI 8.5.13", "", cmds)
	m.monitorPHPCGIProcesses(key, "8.5.13", state, cmds, ports)

	// 模擬其中一個 PHP-CGI 子行程意外崩潰
	if err := cmds[0].Process.Kill(); err != nil {
		t.Fatalf("終止第一個測試子行程失敗: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(*errLogs)
		mu.Unlock()
		if n >= 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	mu.Lock()
	n := len(*errLogs)
	mu.Unlock()
	if n != 1 {
		t.Fatalf("單一 PHP-CGI 意外崩潰應記錄 1 筆錯誤日誌，實際收到 %d 筆", n)
	}
	if !m.IsRunning(key) {
		t.Fatal("尚有其他 PHP-CGI 行程運行時應保持服務運行狀態")
	}

	// 手動停止剩餘行程不應再產生錯誤日誌
	if err := m.stopService(key); err != nil {
		t.Fatalf("stopService 回傳非預期錯誤: %v", err)
	}
	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	finalCount := len(*errLogs)
	mu.Unlock()
	if finalCount != 1 {
		t.Fatalf("後續手動停止剩餘行程不應再新增錯誤日誌，實際共 %d 筆", finalCount)
	}
}

func TestIsExpectedRunning_RejectsStaleStateAfterRestart(t *testing.T) {
	m, _, _ := newTestManager(t)
	const key = "php-8.5.13"

	oldState := m.register(key, "PHP-CGI 8.5.13", "", nil)
	m.unregister(key)
	newState := m.register(key, "PHP-CGI 8.5.13", "", nil)

	if m.isExpectedRunning(key, oldState) {
		t.Fatal("舊實例在重新啟動後不應再被視為有效運行實例")
	}
	if !m.isExpectedRunning(key, newState) {
		t.Fatal("新實例應被視為有效運行實例")
	}
}
