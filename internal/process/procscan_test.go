package process

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// TestRemoveAllWithRetry 驗證被獨佔鎖定的檔案會讓第一次刪除失敗，釋放後可成功。
func TestRemoveAllWithRetry(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "brotlicommon.dll")
	if err := os.WriteFile(file, []byte("locked"), 0644); err != nil {
		t.Fatalf("建立測試檔案失敗: %v", err)
	}

	pathPtr, err := windows.UTF16PtrFromString(file)
	if err != nil {
		t.Fatalf("轉換路徑失敗: %v", err)
	}
	// mode=0：不允許其他行程共享刪除，模擬 php-cgi 載入 DLL 的狀態
	handle, err := windows.CreateFile(pathPtr, windows.GENERIC_READ, 0, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatalf("鎖定測試檔案失敗: %v", err)
	}

	if err := RemoveAllWithRetry(dir, 2, 50*time.Millisecond); err == nil {
		windows.CloseHandle(handle)
		t.Fatal("檔案被鎖定時 RemoveAllWithRetry 應該回傳錯誤")
	}
	if _, statErr := os.Stat(file); statErr != nil {
		t.Fatalf("刪除失敗後原檔案應仍存在，實際: %v", statErr)
	}

	windows.CloseHandle(handle)
	if err := RemoveAllWithRetry(dir, 3, 50*time.Millisecond); err != nil {
		t.Fatalf("解除鎖定後應可刪除，實際: %v", err)
	}
	if _, statErr := os.Stat(dir); !os.IsNotExist(statErr) {
		t.Fatalf("目錄應已被移除，實際 stat: %v", statErr)
	}
}

// TestTerminateProcessesInDir 驗證能找出並終止指定目錄下的行程。
func TestTerminateProcessesInDir(t *testing.T) {
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		t.Skip("找不到 SystemRoot，略過")
	}
	src := filepath.Join(systemRoot, "System32", "ping.exe")
	if _, err := os.Stat(src); err != nil {
		t.Skipf("找不到 %s，略過: %v", src, err)
	}

	dir := t.TempDir()
	dst := filepath.Join(dir, "ping.exe")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Skipf("複製來源讀取失敗，略過: %v", err)
	}
	if err := os.WriteFile(dst, data, 0755); err != nil {
		t.Skipf("無法寫入測試執行檔，略過: %v", err)
	}

	cmd := exec.Command(dst, "-n", "30", "127.0.0.1")
	if err := cmd.Start(); err != nil {
		t.Skipf("無法啟動測試行程（可能受環境限制），略過: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	var found []ProcessInfo
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		found, err = FindProcessesInDir(dir, "ping.exe")
		if err != nil {
			t.Fatalf("FindProcessesInDir 失敗: %v", err)
		}
		if len(found) > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if len(found) == 0 {
		t.Fatalf("應能找到目錄下的 ping.exe (PID %d)", cmd.Process.Pid)
	}

	// 前綴比對不應誤傷：查詢同層但不同名稱的檔案不該有結果
	if others, err := FindProcessesInDir(dir, "php-cgi.exe"); err != nil {
		t.Fatalf("FindProcessesInDir 失敗: %v", err)
	} else if len(others) != 0 {
		t.Fatalf("不應比對到 php-cgi.exe，實際: %v", others)
	}

	killed, err := TerminateProcessesInDir(dir, 3*time.Second, "ping.exe")
	if err != nil {
		t.Fatalf("TerminateProcessesInDir 失敗: %v", err)
	}
	if len(killed) == 0 {
		t.Fatal("應終止至少一個行程")
	}
	if _, err := cmd.Process.Wait(); err != nil {
		t.Fatalf("行程結束後 Wait 不應報錯: %v", err)
	}
}
