//go:build windows
package terminal

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestTerminalSession(t *testing.T) {
	mgr := NewManager()
	defer mgr.StopAll()

	// 1. 測試建立與啟動終端 (使用 cmd.exe 作為測試對象，啟動最快且穩定)
	var outputWg sync.WaitGroup
	outputWg.Add(1)

	var outputBytes []string
	var mu sync.Mutex
	var once sync.Once

	onOutput := func(data string) {
		mu.Lock()
		outputBytes = append(outputBytes, data)
		// 如果在輸出中找到了我們預期的 echo 結果，就釋放 WaitGroup (只呼叫一次 Done)
		combined := strings.Join(outputBytes, "")
		if strings.Contains(combined, "test_wincmp_pty") {
			once.Do(func() {
				outputWg.Done()
			})
		}
		mu.Unlock()
	}

	exitCalled := make(chan struct{}, 1)
	onExit := func() {
		select {
		case exitCalled <- struct{}{}:
		default:
		}
	}

	// 啟動 cmd.exe
	sessionID, err := mgr.StartTerminal(
		"test_project",
		"cmd.exe",
		".",
		80,
		24,
		onOutput,
		onExit,
	)
	if err != nil {
		t.Fatalf("StartTerminal 失敗: %v", err)
	}

	if sessionID == "" {
		t.Fatal("StartTerminal 回傳了空白的 sessionID")
	}

	t.Logf("成功啟動終端，Session ID: %s", sessionID)

	// 2. 測試寫入指令 (echo 指令)
	// cmd.exe 需要 \r\n 作為換行符號執行
	err = mgr.Write(sessionID, "echo test_wincmp_pty\r\n")
	if err != nil {
		t.Fatalf("寫入終端失敗: %v", err)
	}

	// 等待輸出捕獲 (加上 5 秒超時)
	done := make(chan struct{})
	go func() {
		outputWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Log("成功在終端輸出中捕獲到 echo 指令結果")
	case <-time.After(5 * time.Second):
		mu.Lock()
		combined := strings.Join(outputBytes, "")
		mu.Unlock()
		t.Fatalf("捕獲輸出超時！當前接收到的輸出為: %q", combined)
	}

	// 3. 測試調整視窗大小 (Resize)
	err = mgr.Resize(sessionID, 100, 30)
	if err != nil {
		t.Errorf("Resize 失敗: %v", err)
	}
	t.Log("成功調整終端視窗尺寸")

	// 4. 測試停止終端會話與 onExit 觸發
	mgr.Stop(sessionID)

	select {
	case <-exitCalled:
		t.Log("成功觸發 onExit 回呼函數")
	case <-time.After(3 * time.Second):
		t.Fatal("停止終端後，onExit 沒有在 3 秒內被觸發")
	}
}

func TestStopAllSessions(t *testing.T) {
	mgr := NewManager()

	noop := func(data string) {}
	noopExit := func() {}

	// 啟動兩個會話
	id1, err := mgr.StartTerminal("p1", "cmd.exe", ".", 80, 24, noop, noopExit)
	if err != nil {
		t.Fatalf("啟動會話 1 失敗: %v", err)
	}

	id2, err := mgr.StartTerminal("p2", "cmd.exe", ".", 80, 24, noop, noopExit)
	if err != nil {
		t.Fatalf("啟動會話 2 失敗: %v", err)
	}

	mgr.mu.Lock()
	countBefore := len(mgr.sessions)
	mgr.mu.Unlock()

	if countBefore != 2 {
		t.Errorf("預期有 2 個會話，實際上有 %d 個", countBefore)
	}

	// 測試 StopAll
	mgr.StopAll()

	// 稍微等待 goroutine 清理
	time.Sleep(100 * time.Millisecond)

	mgr.mu.Lock()
	countAfter := len(mgr.sessions)
	mgr.mu.Unlock()

	if countAfter != 0 {
		t.Errorf("StopAll 後預期有 0 個會話，實際上有 %d 個", countAfter)
	}

	// 再次寫入已關閉會話應該要回傳錯誤
	err = mgr.Write(id1, "ls\n")
	if err == nil {
		t.Error("寫入已關閉的會話預期會回傳錯誤，但回傳了 nil")
	}

	err = mgr.Resize(id2, 80, 24)
	if err == nil {
		t.Error("Resize 已關閉的會話預期會回傳錯誤，但回傳了 nil")
	}
}
