//go:build windows

package singleinstance

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modkernel32       = windows.NewLazySystemDLL("kernel32.dll")
	moduser32         = windows.NewLazySystemDLL("user32.dll")
	procCreateMutexW  = modkernel32.NewProc("CreateMutexW")
	procFindWindowW   = moduser32.NewProc("FindWindowW")
	procShowWindow    = moduser32.NewProc("ShowWindow")
	procSetForeground = moduser32.NewProc("SetForegroundWindow")
	procIsIconic      = moduser32.NewProc("IsIconic")
)

var (
	mutexName  = "Global\\WinCMP_SingleInstance_Mutex"
	socketPath = filepath.Join(os.TempDir(), "wincmp_activation.sock")
)

var hMutex windows.Handle

// isOtherProcessRunning 遍歷系統進程，判斷是否有指定名稱的「其他進程」正在運行
func isOtherProcessRunning(exeName string) bool {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	err = windows.Process32First(snapshot, &entry)
	if err != nil {
		return false
	}

	myPID := uint32(os.Getpid())
	for {
		name := windows.UTF16ToString(entry.ExeFile[:])
		if strings.ToLower(name) == strings.ToLower(exeName) {
			if entry.ProcessID != myPID {
				return true
			}
		}
		err = windows.Process32Next(snapshot, &entry)
		if err != nil {
			break
		}
	}
	return false
}

// TryAcquire 嘗試取得唯一 Mutex
// 回傳 (isFirstInstance bool, err error)
func TryAcquire() (bool, error) {
	// 互斥防護：確保不與舊版 wincmp v1 同時運作
	if isOtherProcessRunning("wincmp.exe") || isOtherProcessRunning("wincmp-wails.exe") {
		return false, nil
	}

	name, err := windows.UTF16PtrFromString(mutexName)
	if err != nil {
		return false, err
	}

	handle, _, lastErr := procCreateMutexW.Call(
		0, // lpMutexAttributes (nil)
		1, // bInitialOwner = TRUE
		uintptr(unsafe.Pointer(name)),
	)

	hMutex = windows.Handle(handle)

	if lastErr == windows.ERROR_ALREADY_EXISTS {
		return false, nil
	}
	if handle == 0 {
		return false, fmt.Errorf("CreateMutex failed: %w", lastErr)
	}

	return true, nil
}

// Release 釋放 Mutex（程式結束時呼叫）
func Release() {
	if hMutex != 0 {
		windows.CloseHandle(hMutex)
		hMutex = 0
	}
	_ = os.Remove(socketPath)
}

// BringExistingToFront 透過管道通知現有實例顯示視窗
func BringExistingToFront() {
	// 嘗試連接管道並發送啟動訊號
	conn, err := net.DialTimeout("unix", socketPath, 1*time.Second)
	if err != nil {
		// 如果管道連接失敗，至少我們嘗試過了
		return
	}
	defer conn.Close()
	fmt.Fprint(conn, "ACTIVATE")
}

// ListenForActivation 在背景啟動管道監聽
// 當接收到新訊息時，執行 callback
func ListenForActivation(callback func()) {
	// 移除舊的管道檔（如果存在）
	_ = os.Remove(socketPath)

	go func() {
		l, err := net.Listen("unix", socketPath)
		if err != nil {
			return
		}
		defer l.Close()

		for {
			conn, err := l.Accept()
			if err != nil {
				continue
			}
			// 接收到任何連接即表示有新實例嘗試啟動
			callback()
			conn.Close()
		}
	}()
}

// ActivateWindow 嘗試將已存在的視窗帶到前景並還原最小化
func ActivateWindow(windowTitle string) {
	titlePtr, err := windows.UTF16PtrFromString(windowTitle)
	if err != nil {
		return
	}

	// FindWindow 找到視窗的 HWND (即便是自己)
	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(titlePtr)))
	if hwnd == 0 {
		return
	}

	// 如果最小化 (IsIconic)，則執行 SW_RESTORE (9) 還原視窗
	isIconic, _, _ := procIsIconic.Call(hwnd)
	if isIconic != 0 {
		procShowWindow.Call(hwnd, 9) // SW_RESTORE
	}

	// 強制置頂並獲取焦點
	procSetForeground.Call(hwnd)
}
