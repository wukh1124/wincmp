package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"wincmp/internal/downloader"
)

// GitHubRelease 代表 GitHub API 回傳的 Release 結構
type GitHubRelease struct {
	TagName     string `json:"tag_name"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               int64  `json:"size"`
	} `json:"assets"`
}

// ReleaseInfo 提供給前端的版本資訊
type ReleaseInfo struct {
	HasUpdate      bool   `json:"has_update"`
	LatestVersion  string `json:"latest_version"`
	ReleaseNotes   string `json:"release_notes"`
	ReleaseNotesZh string `json:"release_notes_zh"`
	ReleaseNotesEn string `json:"release_notes_en"`
	PublishedAt    string `json:"published_at"`
	DownloadURL    string `json:"download_url"`
	AssetType      string `json:"asset_type"` // "exe" 或 "zip"
	ExpectedSHA256 string `json:"expected_sha256"`
}

// OfficialReleaseURLPrefix 定義官方 GitHub Release 下載網址前綴
const OfficialReleaseURLPrefix = "https://github.com/wukh1124/wincmp/releases/download/"

// ValidateReleaseURL 檢查更新下載 URL 是否符合官方倉庫釋出規則
func ValidateReleaseURL(targetURL string) error {
	trimmed := strings.TrimSpace(targetURL)
	if !strings.HasPrefix(trimmed, OfficialReleaseURLPrefix) {
		return fmt.Errorf("更新下載網址非官方指定來源，基於安全考量拒絕下載")
	}
	return nil
}

// ValidateExecutablePE 檢查二進位檔案是否為合法的 Windows PE 執行檔且大小合理
func ValidateExecutablePE(filePath string) error {
	fi, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("無法取得檔案狀態: %w", err)
	}
	// WinCMP 為 Wails 編譯應用，正常至少大於 5MB；設定門檻 1MB 以杜絕 404 HTML 或微小錯誤文字檔
	if fi.Size() < 1024*1024 {
		return fmt.Errorf("下載之更新檔案大小異常 (%d 位元組)，已中止更新覆蓋", fi.Size())
	}

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("無法開啟檔案進行格式驗證: %w", err)
	}
	defer f.Close()

	header := make([]byte, 2)
	if _, err := io.ReadFull(f, header); err != nil {
		return fmt.Errorf("讀取執行檔標頭失敗: %w", err)
	}

	// Windows PE 檔案標頭魔術數字為 "MZ" (0x4D, 0x5A)
	if header[0] != 'M' || header[1] != 'Z' {
		return fmt.Errorf("下載之更新檔案非合法的 Windows 執行檔 (缺少 MZ 標頭)，已中止更新覆蓋")
	}
	return nil
}

// ParseChecksumsContent 解析 checksums.txt 內容為檔名對應 SHA-256 的 map
func ParseChecksumsContent(content string) map[string]string {
	res := make(map[string]string)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			hash := strings.TrimSpace(fields[0])
			filename := strings.TrimSpace(fields[1])
			filename = strings.TrimPrefix(filename, "*")
			filename = filepath.Base(filename)
			if len(hash) == 64 {
				res[strings.ToLower(filename)] = strings.ToLower(hash)
			}
		}
	}
	return res
}

// GetExpectedSHA256 取得當前緩存 Release 中特定下載連結對應的預期 SHA-256
func GetExpectedSHA256(downloadURL string) string {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if cachedRelease != nil && cachedRelease.DownloadURL == downloadURL {
		return cachedRelease.ExpectedSHA256
	}
	return ""
}

var (
	cacheMu       sync.Mutex
	cachedRelease *ReleaseInfo
	lastCheckTime time.Time
)

// CheckNewVersion 檢查是否有新版本 (有快取就直接返回快取)
func CheckNewVersion(currentVersion string) (*ReleaseInfo, error) {
	return CheckNewVersionOpt(currentVersion, false)
}

// CheckNewVersionOpt 檢查是否有新版本，支援強制刷新選項
func CheckNewVersionOpt(currentVersion string, force bool) (*ReleaseInfo, error) {
	cacheMu.Lock()
	if !force && cachedRelease != nil {
		res := *cachedRelease
		cacheMu.Unlock()
		return &res, nil
	}
	cacheMu.Unlock()

	url := "https://api.github.com/repos/wukh1124/wincmp/releases/latest"
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "wincmp-updater")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("連線 GitHub API 失敗: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub 回應狀態碼錯誤: %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("解析 GitHub Release 失敗: %w", err)
	}

	hasUpdate := compareVersions(release.TagName, currentVersion) > 0

	var downloadURL string
	var assetType string

	// 1. 優先搜尋 .exe
	for _, asset := range release.Assets {
		if strings.HasSuffix(strings.ToLower(asset.Name), ".exe") && strings.Contains(strings.ToLower(asset.Name), "wincmp") {
			downloadURL = asset.BrowserDownloadURL
			assetType = "exe"
			break
		}
	}

	// 2. 如果沒有單獨 exe，尋找 Windows 的 zip 檔
	if downloadURL == "" {
		for _, asset := range release.Assets {
			nameLower := strings.ToLower(asset.Name)
			if strings.HasSuffix(nameLower, ".zip") && (strings.Contains(nameLower, "win") || strings.Contains(nameLower, "windows")) {
				downloadURL = asset.BrowserDownloadURL
				assetType = "zip"
				break
			}
		}
	}

	// 3. 兜底，找第一個 zip 檔
	if downloadURL == "" {
		for _, asset := range release.Assets {
			if strings.HasSuffix(strings.ToLower(asset.Name), ".zip") {
				downloadURL = asset.BrowserDownloadURL
				assetType = "zip"
				break
			}
		}
	}

	// 4. 解析對應資產的 SHA-256（若有 checksums.txt 則優先取得）
	var expectedSHA string
	var checksumURL string
	for _, asset := range release.Assets {
		nameLower := strings.ToLower(asset.Name)
		if nameLower == "checksums.txt" || nameLower == "sha256sums.txt" || nameLower == "checksum.txt" {
			checksumURL = asset.BrowserDownloadURL
			break
		}
	}

	if checksumURL != "" && downloadURL != "" {
		checksumContent := fetchRawContent(client, checksumURL)
		checksumMap := ParseChecksumsContent(checksumContent)
		targetFile := filepath.Base(downloadURL)
		if hash, ok := checksumMap[strings.ToLower(targetFile)]; ok {
			expectedSHA = hash
		}
	}

	// Fallback：若無 checksums.txt，嘗試從 Release Body 尋找 64 碼 hex
	if expectedSHA == "" && downloadURL != "" && release.Body != "" {
		targetFile := filepath.Base(downloadURL)
		expectedSHA = extractSHAFromReleaseBody(release.Body, targetFile)
	}

	// 獲取中英文 Release Notes 內容 (從 raw.githubusercontent.com 下載)
	zhNotesURL := fmt.Sprintf("https://raw.githubusercontent.com/wukh1124/wincmp/main/release_note/%s/release_notes_zh.md", release.TagName)
	enNotesURL := fmt.Sprintf("https://raw.githubusercontent.com/wukh1124/wincmp/main/release_note/%s/release_notes.md", release.TagName)

	zhNotes := fetchRawContent(client, zhNotesURL)
	enNotes := fetchRawContent(client, enNotesURL)

	// fallback
	if zhNotes == "" {
		zhNotes = release.Body
	}
	if enNotes == "" {
		enNotes = release.Body
	}

	info := &ReleaseInfo{
		HasUpdate:      hasUpdate,
		LatestVersion:  release.TagName,
		ReleaseNotes:   release.Body,
		ReleaseNotesZh: zhNotes,
		ReleaseNotesEn: enNotes,
		PublishedAt:    release.PublishedAt,
		DownloadURL:    downloadURL,
		AssetType:      assetType,
		ExpectedSHA256: expectedSHA,
	}

	cacheMu.Lock()
	cachedRelease = info
	lastCheckTime = time.Now()
	cacheMu.Unlock()

	return info, nil
}

// extractSHAFromReleaseBody 從 Release Body 中嘗試找出指定資產檔名對應的 SHA-256
func extractSHAFromReleaseBody(body, filename string) string {
	lines := strings.Split(body, "\n")
	targetLower := strings.ToLower(filename)
	for _, line := range lines {
		lineLower := strings.ToLower(line)
		if strings.Contains(lineLower, targetLower) || strings.Contains(lineLower, "sha256") || strings.Contains(lineLower, "sha-256") {
			fields := strings.Fields(line)
			for _, f := range fields {
				cleanF := strings.Trim(f, "`\"':(),[]")
				if len(cleanF) == 64 && isHex(cleanF) {
					return strings.ToLower(cleanF)
				}
			}
		}
	}
	return ""
}

func isHex(s string) bool {
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

// fetchRawContent 輔助函數：從 URL 獲取純文字內容
func fetchRawContent(client *http.Client, url string) string {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "wincmp-updater")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return string(bodyBytes)
}

// DownloadAndUpdate 下載並替換二進位檔，成功後返回新執行檔路徑
func DownloadAndUpdate(url string, assetType string, expectedSHA256 string, baseDir string, progressCb func(current, total int64)) (string, error) {
	// 0. 驗證 URL 白名單，必須為官方 GitHub Release 下載位址
	if err := ValidateReleaseURL(url); err != nil {
		return "", err
	}

	tempDir := filepath.Join(baseDir, "data", "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("無法建立臨時目錄: %w", err)
	}

	var tempExePath string
	var newExeName string

	if assetType == "exe" {
		tempExePath = filepath.Join(tempDir, "wincmp_new.exe")
		if err := downloader.DownloadFile(url, tempExePath, progressCb); err != nil {
			_ = os.Remove(tempExePath)
			return "", fmt.Errorf("下載新版本執行檔失敗：%w。\n\n建議指引：\n請檢查網路連線或直接前往官方 Releases 手動下載更新：\n%s", err, url)
		}

		// 完整性校驗：若有提供預期 SHA-256，進行比對
		if expectedSHA256 != "" {
			actualSHA, err := downloader.CalculateSHA256(tempExePath)
			if err != nil {
				_ = os.Remove(tempExePath)
				return "", fmt.Errorf("計算新版本雜湊值失敗: %w", err)
			}
			if !strings.EqualFold(actualSHA, expectedSHA256) {
				_ = os.Remove(tempExePath)
				return "", fmt.Errorf("新版本完整性校驗 (SHA-256) 失敗，已中止更新覆蓋 (預期: %s, 實際: %s)。\n\n建議指引：\n請直接前往官方 Releases 手動下載安裝包：\n%s", expectedSHA256, actualSHA, url)
			}
		}

		newExeName = filepath.Base(url)
		if !strings.HasSuffix(strings.ToLower(newExeName), ".exe") {
			newExeName = "wincmp.exe"
		}
	} else {
		// zip 流程
		tempZipPath := filepath.Join(tempDir, "update.zip")
		if err := downloader.DownloadFile(url, tempZipPath, progressCb); err != nil {
			_ = os.Remove(tempZipPath)
			return "", fmt.Errorf("下載新版本壓縮包失敗：%w。\n\n建議指引：\n請檢查網路連線或直接前往官方 Releases 手動下載更新：\n%s", err, url)
		}

		// 完整性校驗：若有提供預期 SHA-256，在解壓前進行比對
		if expectedSHA256 != "" {
			actualSHA, err := downloader.CalculateSHA256(tempZipPath)
			if err != nil {
				_ = os.Remove(tempZipPath)
				return "", fmt.Errorf("計算新版本壓縮檔雜湊值失敗: %w", err)
			}
			if !strings.EqualFold(actualSHA, expectedSHA256) {
				_ = os.Remove(tempZipPath)
				return "", fmt.Errorf("新版本完整性校驗 (SHA-256) 失敗，已中止更新覆蓋 (預期: %s, 實際: %s)。\n\n建議指引：\n請直接前往官方 Releases 手動下載安裝包：\n%s", expectedSHA256, actualSHA, url)
			}
		}

		extractDir := filepath.Join(tempDir, "extracted")
		_ = os.RemoveAll(extractDir)

		if err := downloader.Unzip(tempZipPath, extractDir); err != nil {
			_ = os.Remove(tempZipPath)
			return "", fmt.Errorf("解壓縮新版本 zip 失敗：%w。\n\n建議指引：\n壓縮包可能未下載完整，請直接手動下載安裝：\n%s", err, url)
		}
		_ = os.Remove(tempZipPath)

		// 在解壓縮目錄下尋找 exe 檔案
		var foundExe string
		err := filepath.WalkDir(extractDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".exe") {
				foundExe = path
				return filepath.SkipAll // 找到第一個 exe 就停止
			}
			return nil
		})
		if err != nil || foundExe == "" {
			_ = os.RemoveAll(extractDir)
			return "", fmt.Errorf("在壓縮包中找不到任何 exe 執行檔。\n\n建議指引：\n請直接手動下載官方安裝包：\n%s", url)
		}

		newExeName = filepath.Base(foundExe)
		tempExePath = filepath.Join(tempDir, "wincmp_new.exe")
		_ = os.Remove(tempExePath)
		if err := os.Rename(foundExe, tempExePath); err != nil {
			_ = os.RemoveAll(extractDir)
			return "", fmt.Errorf("搬移解壓後的 exe 失敗: %w", err)
		}
		_ = os.RemoveAll(extractDir)
	}

	// 檢查新 exe 是否確實存在
	if _, err := os.Stat(tempExePath); err != nil {
		return "", fmt.Errorf("無法驗證下載的執行檔: %w", err)
	}

	// 安全性格式檢驗：驗證是否為合法 Windows PE 執行檔 (MZ 標頭及大小)
	if err := ValidateExecutablePE(tempExePath); err != nil {
		_ = os.Remove(tempExePath)
		return "", fmt.Errorf("%w。\n\n建議指引：\n請手動前往官方 Releases 頁面下載標準安裝檔案：\n%s", err, url)
	}

	// 執行檔案重命名與替換
	execPath, err := os.Executable()
	if err != nil {
		_ = os.Remove(tempExePath)
		return "", fmt.Errorf("無法取得當前進程的執行檔路徑: %w", err)
	}

	oldPath := execPath + ".old"
	_ = os.Remove(oldPath) // 先刪除舊的備份（如果存在）

	// 將當前運行的 exe 重命名為 .old
	if err := os.Rename(execPath, oldPath); err != nil {
		_ = os.Remove(tempExePath)
		return "", fmt.Errorf("無法將當前執行檔重命名，可能權限不足: %w", err)
	}

	// 將新下載的 exe 移動到原執行目錄，並保持新版本原檔名
	newExePath := filepath.Join(filepath.Dir(execPath), newExeName)
	_ = os.Remove(newExePath) // 先刪除同名的新執行檔（如果存在）
	if err := os.Rename(tempExePath, newExePath); err != nil {
		// 若移動失敗，嘗試還原舊版，以防崩潰
		_ = os.Rename(oldPath, execPath)
		_ = os.Remove(tempExePath)
		return "", fmt.Errorf("搬移新執行檔失敗，已自動還原原版本: %w", err)
	}

	return newExePath, nil
}

// RollbackUpdate 當新版本啟動失敗時，緊急將新 exe 移除並將 .old 復原回原本的 exe
func RollbackUpdate(newExePath string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("無法取得執行檔路徑: %w", err)
	}
	return RollbackUpdateWithPath(execPath, newExePath)
}

// RollbackUpdateWithPath 針對指定執行路徑執行回滾與復原
func RollbackUpdateWithPath(execPath, newExePath string) error {
	oldPath := execPath + ".old"
	if _, statErr := os.Stat(oldPath); statErr == nil {
		// 刪除可能損壞或被攔截的新版 exe
		if newExePath != "" {
			_ = os.Remove(newExePath)
		}
		// 還原舊版本
		if renameErr := os.Rename(oldPath, execPath); renameErr != nil {
			return fmt.Errorf("還原舊版本執行檔失敗: %w", renameErr)
		}
	}
	return nil
}

// CleanupOldVersion 清理殘留的舊版本檔案
func CleanupOldVersion(baseDir string) {
	// 異步延遲刪除，確保舊進程已完全釋放鎖定退出
	go func() {
		time.Sleep(3 * time.Second)

		// 1. 清理 baseDir 下的 .exe.old 檔案
		if files, err := os.ReadDir(baseDir); err == nil {
			for _, f := range files {
				if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), ".exe.old") {
					_ = os.Remove(filepath.Join(baseDir, f.Name()))
				}
			}
		}

		// 2. 清理執行檔所在目錄下的 .exe.old 檔案 (如果與 baseDir 不同)
		if execPath, err := os.Executable(); err == nil {
			execDir := filepath.Dir(execPath)
			if execDir != baseDir {
				if files, err := os.ReadDir(execDir); err == nil {
					for _, f := range files {
						if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), ".exe.old") {
							_ = os.Remove(filepath.Join(execDir, f.Name()))
						}
					}
				}
			}
		}
	}()

	// 同時順手清空臨時 temp 目錄
	tempDir := filepath.Join(baseDir, "data", "temp")
	_ = os.RemoveAll(tempDir)
}

// compareVersions 比較兩個版本號字串大小 (v1 < v2 回傳 -1，v1 > v2 回傳 1，相等回傳 0)
func compareVersions(v1, v2 string) int {
	clean := func(v string) string {
		v = strings.TrimPrefix(v, "v")
		v = strings.Split(v, "-")[0]
		return v
	}
	v1 = clean(v1)
	v2 = clean(v2)

	p1 := strings.Split(v1, ".")
	p2 := strings.Split(v2, ".")

	for i := 0; i < len(p1) || i < len(p2); i++ {
		var n1, n2 int
		if i < len(p1) {
			n1, _ = strconv.Atoi(p1[i])
		}
		if i < len(p2) {
			n2, _ = strconv.Atoi(p2[i])
		}
		if n1 < n2 {
			return -1
		} else if n1 > n2 {
			return 1
		}
	}
	return 0
}
