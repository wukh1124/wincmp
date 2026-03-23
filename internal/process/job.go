package process

import (
	"fmt"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	globalJobHandle windows.Handle
	initJobOnce     sync.Once
	initJobErr      error
)

// initJobObject 初始化一個 Windows Job Object，並設定當 Job 關閉時自動結束其下的所有處理程序。
// 然後將當前程序 (wincmp) 綁定到該 Job Object 中，這樣所有未來由 wincmp 建立的子程序都會繼承此行為。
func initJobObject() error {
	initJobOnce.Do(func() {
		job, err := windows.CreateJobObject(nil, nil)
		if err != nil {
			initJobErr = fmt.Errorf("CreateJobObject failed: %v", err)
			return
		}

		info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
			BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
				LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
			},
		}

		_, err = windows.SetInformationJobObject(
			job,
			windows.JobObjectExtendedLimitInformation,
			uintptr(unsafe.Pointer(&info)),
			uint32(unsafe.Sizeof(info)),
		)
		if err != nil {
			windows.CloseHandle(job)
			initJobErr = fmt.Errorf("SetInformationJobObject failed: %v", err)
			return
		}

		// 將目前應用程式的處理程序分配到這個 Job Object
		// 從 Windows 8 開始支援巢狀 Job Object，所以這通常是安全的。
		err = windows.AssignProcessToJobObject(job, windows.CurrentProcess())
		if err != nil {
			windows.CloseHandle(job)
			initJobErr = fmt.Errorf("AssignProcessToJobObject failed: %v", err)
			return
		}

		// 儲存在全域變數中以確保 Handle 不會被意外回收關閉
		globalJobHandle = job
	})
	return initJobErr
}
