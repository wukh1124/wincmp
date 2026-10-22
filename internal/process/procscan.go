package process

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ProcessInfo 描述一個執行中的行程與其映像檔路徑
type ProcessInfo struct {
	PID       uint32
	ImagePath string
}

// FindProcessesInDir 回傳映像檔位於 dir 之下、且檔名符合 names 的行程。
// names 為空時比對該目錄下所有執行檔；比對不區分大小寫，且只接受 dir 的子路徑，
// 避免用前綴比對時把 php-8.3.3 誤認成 php-8.3.33。
func FindProcessesInDir(dir string, names ...string) ([]ProcessInfo, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("解析目錄失敗: %w", err)
	}
	dirPrefix := strings.ToLower(strings.TrimRight(absDir, `\`) + `\`)

	want := make(map[string]struct{}, len(names))
	for _, name := range names {
		want[strings.ToLower(name)] = struct{}{}
	}

	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("建立行程快照失敗: %w", err)
	}
	defer windows.CloseHandle(snapshot)

	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	if err := windows.Process32First(snapshot, &entry); err != nil {
		return nil, fmt.Errorf("列舉行程失敗: %w", err)
	}

	selfPID := uint32(os.Getpid())
	var found []ProcessInfo
	for {
		pid := entry.ProcessID
		if pid != 0 && pid != selfPID {
			if imagePath, ok := processImagePath(pid); ok {
				if strings.HasPrefix(strings.ToLower(imagePath), dirPrefix) {
					if len(want) == 0 {
						found = append(found, ProcessInfo{PID: pid, ImagePath: imagePath})
					} else if _, ok := want[strings.ToLower(filepath.Base(imagePath))]; ok {
						found = append(found, ProcessInfo{PID: pid, ImagePath: imagePath})
					}
				}
			}
		}

		if err := windows.Process32Next(snapshot, &entry); err != nil {
			break
		}
	}
	return found, nil
}

// TerminateProcessesInDir 終止映像檔位於 dir 下、檔名符合 names 的行程，並等待其結束。
// 回傳實際送出終止的 PID，供呼叫端記錄或組錯誤訊息。
func TerminateProcessesInDir(dir string, wait time.Duration, names ...string) ([]uint32, error) {
	procs, err := FindProcessesInDir(dir, names...)
	if err != nil {
		return nil, err
	}

	waitMillis := uint32(wait / time.Millisecond)
	var killed []uint32
	for _, proc := range procs {
		handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE|windows.SYNCHRONIZE, false, proc.PID)
		if err != nil {
			// 行程可能在列舉後自行結束，或權限不足（例如系統服務），略過即可
			continue
		}
		if err := windows.TerminateProcess(handle, 1); err == nil {
			killed = append(killed, proc.PID)
			// 等行程真的結束，檔案鎖才會釋放
			_, _ = windows.WaitForSingleObject(handle, waitMillis)
		}
		windows.CloseHandle(handle)
	}
	return killed, nil
}

// RemoveAllWithRetry 以固定間隔重試 os.RemoveAll。
// Windows 上行程終止與檔案鎖釋放之間有短暫時間差，單次刪除容易誤報 Access is denied。
func RemoveAllWithRetry(dir string, attempts int, interval time.Duration) error {
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		if lastErr = os.RemoveAll(dir); lastErr == nil {
			return nil
		}
		if i < attempts-1 {
			time.Sleep(interval)
		}
	}
	return lastErr
}

// processImagePath 取得指定 PID 的完整映像檔路徑
func processImagePath(pid uint32) (string, bool) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", false
	}
	defer windows.CloseHandle(handle)

	buf := make([]uint16, windows.MAX_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(handle, 0, &buf[0], &size); err != nil {
		return "", false
	}
	return windows.UTF16ToString(buf[:size]), true
}
