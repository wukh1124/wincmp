package hosts

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"wincmp/internal/i18n"
)

const (
	HostsFilePath = `C:\Windows\System32\drivers\etc\hosts`
)

// validDomainPattern 用於驗證域名是否只含合法字元（允許底線，以防止 hosts 注入攻擊）
var validDomainPattern = regexp.MustCompile(`^[a-zA-Z0-9_]([a-zA-Z0-9\-_]*[a-zA-Z0-9_])?(\.[a-zA-Z0-9_]([a-zA-Z0-9\-_]*[a-zA-Z0-9_])?)*$`)

// IsValidDomain 檢查域名是否只含合法字元（用於外部呼叫）
func IsValidDomain(domain string) bool {
	return validDomainPattern.MatchString(domain)
}

// BackupHosts 將目前的 hosts 備份至指定的備份目錄
func BackupHosts(baseDir string) (string, error) {
	backupDir := filepath.Join(baseDir, "data", "backup", "hosts")
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T("無法建立備份目錄"), err)
	}

	timestamp := time.Now().Format("20060102150405")
	backupFileName := fmt.Sprintf("hosts_%s", timestamp)
	backupPath := filepath.Join(backupDir, backupFileName)

	if err := copyFile(HostsFilePath, backupPath); err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T("備份 hosts 失敗"), err)
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
func UpdateHosts(domains []string) error {
	if len(domains) == 0 {
		return nil
	}

	safeDomains := sanitizeDomains(domains)
	if len(safeDomains) == 0 {
		// 收集所有無效的域名，用於提供更具體的錯誤訊息
		var invalidDomains []string
		for _, d := range domains {
			if !validDomainPattern.MatchString(d) {
				invalidDomains = append(invalidDomains, d)
			}
		}
		return fmt.Errorf("%s", i18n.Tfmt("以下域名含非法字元: %v，請手動新增至 hosts", invalidDomains))
	}

	// 直接以追加模式開啟 hosts 檔案
	f, err := os.OpenFile(HostsFilePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("無法開啟 hosts 檔案進行寫入 (可能需要管理員權限)"), err)
	}
	defer f.Close()

	// 寫入時間戳和域名
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
		return nil, fmt.Errorf("%s: %w", i18n.T("無法讀取 hosts 檔案"), err)
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
