package hosts

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	HostsFilePath = `C:\Windows\System32\drivers\etc\hosts`
)

// BackupHosts 將目前的 hosts 備份至指定的備份目錄
func BackupHosts(baseDir string) (string, error) {
	backupDir := filepath.Join(baseDir, "data", "backup", "hosts")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
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

// UpdateHosts 將缺失的網域寫入 hosts 檔（需要 UAC 權限）
func UpdateHosts(domains []string) error {
	if len(domains) == 0 {
		return nil
	}

	f, err := os.OpenFile(HostsFilePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("無法開啟 hosts 檔案進行寫入 (可能需要管理員權限): %w", err)
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	if _, err := f.WriteString(fmt.Sprintf("\n# Added by WinCMP at %s\n", timestamp)); err != nil {
		return err
	}

	for _, domain := range domains {
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
		// 忽略註解與空行
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// hosts 格式通常為: IP Domain1 [Domain2 ...]
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			// fields[0] 是 IP, 後面都是 domain
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
