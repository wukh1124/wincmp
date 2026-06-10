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
	}

	cacheMu.Lock()
	cachedRelease = info
	lastCheckTime = time.Now()
	cacheMu.Unlock()

	return info, nil
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
func DownloadAndUpdate(url string, assetType string, baseDir string, progressCb func(current, total int64)) (string, error) {
	tempDir := filepath.Join(baseDir, "data", "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("無法建立臨時目錄: %w", err)
	}

	var tempExePath string
	var newExeName string

	if assetType == "exe" {
		tempExePath = filepath.Join(tempDir, "wincmp_new.exe")
		if err := downloader.DownloadFile(url, tempExePath, progressCb); err != nil {
			return "", fmt.Errorf("下載 exe 失敗: %w", err)
		}
		newExeName = filepath.Base(url)
		if !strings.HasSuffix(strings.ToLower(newExeName), ".exe") {
			newExeName = "wincmp.exe"
		}
	} else {
		// zip 流程
		tempZipPath := filepath.Join(tempDir, "update.zip")
		if err := downloader.DownloadFile(url, tempZipPath, progressCb); err != nil {
			return "", fmt.Errorf("下載 zip 失敗: %w", err)
		}

		extractDir := filepath.Join(tempDir, "extracted")
		_ = os.RemoveAll(extractDir)

		if err := downloader.Unzip(tempZipPath, extractDir); err != nil {
			os.Remove(tempZipPath)
			return "", fmt.Errorf("解壓縮 zip 失敗: %w", err)
		}
		os.Remove(tempZipPath)

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
			return "", fmt.Errorf("在壓縮包中找不到任何 exe 執行檔")
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

	// 執行檔案重命名與替換
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("無法取得當前進程的執行檔路徑: %w", err)
	}

	oldPath := execPath + ".old"
	_ = os.Remove(oldPath) // 先刪除舊的備份（如果存在）

	// 將當前運行的 exe 重命名為 .old
	if err := os.Rename(execPath, oldPath); err != nil {
		return "", fmt.Errorf("無法將當前執行檔重命名，可能權限不足: %w", err)
	}

	// 將新下載的 exe 移動到原執行目錄，並保持新版本原檔名
	newExePath := filepath.Join(filepath.Dir(execPath), newExeName)
	_ = os.Remove(newExePath) // 先刪除同名的新執行檔（如果存在）
	if err := os.Rename(tempExePath, newExePath); err != nil {
		// 若移動失敗，嘗試還原舊版，以防崩潰
		_ = os.Rename(oldPath, execPath)
		return "", fmt.Errorf("搬移新執行檔失敗: %w", err)
	}

	return newExePath, nil
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
