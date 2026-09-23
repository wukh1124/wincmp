package downloader

import (
	"archive/zip"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WriteCounter 用於監聽下載進度
type WriteCounter struct {
	Total      int64
	Current    int64
	OnProgress func(current, total int64)
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Current += int64(n)
	if wc.OnProgress != nil {
		wc.OnProgress(wc.Current, wc.Total)
	}
	return n, nil
}

// DownloadFile 下載檔案並回報進度，若遇 PHP 官方版本歸檔 (404) 自動嘗試 archives 降級備援
func DownloadFile(downloadURL, destPath string, progressCb func(current, total int64)) error {
	// 確保父目錄存在
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("無法建立目錄: %w", err)
	}

	urlsToTry := []string{downloadURL}
	// 若為 PHP 官方 release 連結且未包含 archives，遇 404 時嘗試自動降級至 archives 備援
	if strings.Contains(downloadURL, "windows.php.net/downloads/releases/") && !strings.Contains(downloadURL, "/archives/") {
		urlsToTry = append(urlsToTry, strings.Replace(downloadURL, "/downloads/releases/", "/downloads/releases/archives/", 1))
	}

	var lastErr error
	for i, u := range urlsToTry {
		out, err := os.Create(destPath)
		if err != nil {
			return fmt.Errorf("無法建立檔案: %w", err)
		}

		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			out.Close()
			lastErr = fmt.Errorf("建立下載請求失敗: %w", err)
			continue
		}
		req.Header.Set("User-Agent", "WinCMP-Downloader/2.0")

		client := &http.Client{Timeout: 600 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			out.Close()
			lastErr = fmt.Errorf("下載請求失敗: %w", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			out.Close()
			lastErr = fmt.Errorf("伺服器回應錯誤: %s", resp.Status)
			// 若非 404 或已經是最後一個嘗試，中斷後續嘗試
			if resp.StatusCode != http.StatusNotFound && i == len(urlsToTry)-1 {
				break
			}
			continue
		}

		counter := &WriteCounter{
			Total:      resp.ContentLength,
			OnProgress: progressCb,
		}

		_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
		resp.Body.Close()
		out.Close()

		if err != nil {
			lastErr = fmt.Errorf("寫入檔案失敗: %w", err)
			continue
		}

		return nil
	}

	return lastErr
}

// Unzip 解壓 zip 檔案到指定目錄
func Unzip(srcZip, destDir string) error {
	r, err := zip.OpenReader(srcZip)
	if err != nil {
		return fmt.Errorf("無法開啟 zip 檔案: %w", err)
	}
	defer r.Close()

	// 確保目標資料夾存在
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("無法建立解壓縮目錄: %w", err)
	}

	for _, f := range r.File {
		// 預防 Zip Slip 漏洞：檢查路徑是否有 ".."
		if strings.Contains(f.Name, "..") {
			continue
		}

		fpath := filepath.Join(destDir, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, f.Mode())
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)

		// 關閉資源
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

// CalculateSHA256 計算指定本機檔案的 SHA-256 雜湊值並傳回十六進位字串
func CalculateSHA256(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// TrustedHostSuffixes 定義受信任的依賴下載官方網域後綴
var TrustedHostSuffixes = []string{
	"github.com",
	"githubusercontent.com",
	"curl.se",
	"mariadb.org",
	"nodejs.org",
	"php.net",
}

// ValidateSecurity 檢查依賴項目之 URL 與 SHA-256 是否符合安全標準
func ValidateSecurity(targetURL, sha256Hex string) error {
	// 1. 嚴格檢查 SHA-256 是否存在且格式合法 (必須是 64 碼十六進位)
	trimmedSHA := strings.TrimSpace(sha256Hex)
	if trimmedSHA == "" {
		return fmt.Errorf("依賴項目未提供 SHA-256 校驗碼，基於安全考量拒絕下載")
	}
	if len(trimmedSHA) != 64 {
		return fmt.Errorf("依賴項目的 SHA-256 校驗碼格式無效，基於安全考量拒絕下載")
	}
	for _, r := range trimmedSHA {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return fmt.Errorf("依賴項目的 SHA-256 校驗碼格式無效，基於安全考量拒絕下載")
		}
	}

	// 2. 檢查 URL 是否合法且使用 HTTPS 協議
	trimmedURL := strings.TrimSpace(targetURL)
	u, err := url.Parse(trimmedURL)
	if err != nil || u.Host == "" {
		return fmt.Errorf("依賴項目的下載網址格式無效: %w", err)
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return fmt.Errorf("依賴項目的下載網址必須使用 HTTPS 加密協議")
	}

	// 3. 檢查 Host 是否符合受信任官方網域白名單
	hostname := strings.ToLower(u.Hostname())
	isTrusted := false
	for _, suffix := range TrustedHostSuffixes {
		if hostname == suffix || strings.HasSuffix(hostname, "."+suffix) {
			isTrusted = true
			break
		}
	}

	if !isTrusted {
		return fmt.Errorf("依賴項目下載來源非受信任網域 (%s)，基於安全考量拒絕下載", hostname)
	}

	return nil
}

