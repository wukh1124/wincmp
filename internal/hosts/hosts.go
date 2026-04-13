package hosts

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	modkernel32      = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = modkernel32.NewProc("LockFileEx")
	procUnlockFileEx = modkernel32.NewProc("UnlockFileEx")
)

const (
	HostsFilePath = `C:\Windows\System32\drivers\etc\hosts`

	lockfileExclusiveLock = 0x00000002
)

// lockFile 使用 Windows LockFileEx 對檔案進行獨佔鎖定
func lockFile(f *os.File) error {
	var overlapped syscall.Overlapped
	ret, _, err := procLockFileEx.Call(
		f.Fd(),
		uintptr(lockfileExclusiveLock),
		0,
		0xFFFFFFFF, 0xFFFFFFFF,
		uintptr(unsafe.Pointer(&overlapped)),
	)
	if ret == 0 {
		return err
	}
	return nil
}

// unlockFile 解除檔案鎖定
func unlockFile(f *os.File) error {
	var overlapped syscall.Overlapped
	ret, _, err := procUnlockFileEx.Call(
		f.Fd(),
		0,
		0xFFFFFFFF, 0xFFFFFFFF,
		uintptr(unsafe.Pointer(&overlapped)),
	)
	if ret == 0 {
		return err
	}
	return nil
}

// validDomainPattern 用於驗證域名是否只含合法字元（防止 hosts 注入攻擊）
var validDomainPattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?)*$`)

// BackupHosts 將目前的 hosts 備份至指定的備份目錄
func BackupHosts(baseDir string) (string, error) {
	backupDir := filepath.Join(baseDir, "data", "backup", "hosts")
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return "", fmt.Errorf("無法建立備份目錄: %w", err)
	}

	timestamp := time.Now().Format("20060102150405")
	backupFileName := fmt.Sprintf("hosts_%s", timestamp)
	backupPath := filepath.Join(backupDir, backupFileName)

	if err := copyFile(HostsFilePath, backupPath); err != nil {
		return "", fmt.Errorf("備份 hosts 失敗: %w", err)
	}

	return backupPath, nil
}

// CheckHosts 檢查哪些網域不在 hosts 中，返回缺失的網域列表
func CheckHosts(domains []string) ([]string, error) {
	if len(domains) == 0 {
		return nil, nil
	}

	existingDomains, err := readExistingDomains()
	if err != nil {
		return nil, err
	}

	var missing []string
	for _, domain := range domains {
		found := false
		for _, existing := range existingDomains {
			if strings.EqualFold(domain, existing) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, domain)
		}
	}

	return missing, nil
}

// sanitizeDomains 驗證並過濾域名列表，只保留合法域名
func sanitizeDomains(domains []string) []string {
	var safe []string
	for _, domain := range domains {
		if validDomainPattern.MatchString(domain) {
			safe = append(safe, domain)
		}
	}
	return safe
}

// UpdateHosts 將缺失的網域寫入 hosts 檔（需要 UAC 權限）
// 使用 Windows file locking (LockFileEx) 確保寫入原子性
func UpdateHosts(domains []string) error {
	if len(domains) == 0 {
		return nil
	}

	safeDomains := sanitizeDomains(domains)
	if len(safeDomains) == 0 {
		return fmt.Errorf("所有域名均未通過安全性驗證，已拒絕寫入 hosts")
	}

	f, err := os.OpenFile(HostsFilePath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("無法開啟 hosts 檔案進行寫入 (可能需要管理員權限): %w", err)
	}
	defer f.Close()

	// Windows file locking: 獨佔鎖定防止同時寫入
	if err := lockFile(f); err != nil {
		return fmt.Errorf("無法鎖定 hosts 檔案: %w", err)
	}
	defer unlockFile(f)

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	if _, err := f.WriteString(fmt.Sprintf("\n# Added by WinCMP at %s\n", timestamp)); err != nil {
		return err
	}

	for _, domain := range safeDomains {
		line := fmt.Sprintf("127.0.0.1  %s\n", domain)
		if _, err := f.WriteString(line); err != nil {
			return err
		}
	}

	return nil
}

// readExistingDomains 讀取目前 hosts 中已存在的網域
func readExistingDomains() ([]string, error) {
	file, err := os.Open(HostsFilePath)
	if err != nil {
		return nil, fmt.Errorf("無法讀取 hosts 檔案: %w", err)
	}
	defer file.Close()

	var domains []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 2 {
			for i := 1; i < len(fields); i++ {
				domains = append(domains, fields[i])
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return domains, nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
