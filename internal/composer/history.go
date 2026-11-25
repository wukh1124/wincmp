package composer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

const (
	// MaxHistoryPerProject 單一專案於磁碟保留的最多執行紀錄筆數
	MaxHistoryPerProject = 50
)

var validProjectNameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// HistoryEntry 單筆 Composer 指令執行歷史紀錄
type HistoryEntry struct {
	ID              string   `json:"id"`
	Project         string   `json:"project"`
	Command         string   `json:"command"`
	Args            []string `json:"args"`
	PHPVersion      string   `json:"php_version"`
	ComposerVersion string   `json:"composer_version"`
	StartedAt       string   `json:"started_at"`
	FinishedAt      string   `json:"finished_at"`
	DurationMs      int64    `json:"duration_ms"`
	ExitCode        int      `json:"exit_code"`
	Status          string   `json:"status"` // "success" | "error" | "stopped" | "timeout"
	ErrorReason     string   `json:"error_reason"`
	Output          string   `json:"output"`
	Truncated       bool     `json:"truncated"`
	OriginalBytes   int      `json:"original_bytes"`
	OriginalLines   int      `json:"original_lines"`
}

// HistoryPage 分頁查詢回傳結構（Items 依時間由舊到新排列，方便終端由上往下顯示）
type HistoryPage struct {
	Items   []HistoryEntry `json:"items"`
	Total   int            `json:"total"`
	HasMore bool           `json:"has_more"`
}

// HistoryStore 管理各專案 Composer 歷史紀錄的 JSONL 檔案持久化
type HistoryStore struct {
	baseDir    string
	maxEntries int
	mu         sync.Mutex
}

// NewHistoryStore 建立歷史儲存器
func NewHistoryStore(baseDir string) *HistoryStore {
	return &HistoryStore{
		baseDir:    baseDir,
		maxEntries: MaxHistoryPerProject,
	}
}

// ValidateProjectName 嚴格白名單校驗專案名稱，防止路徑遍歷
func ValidateProjectName(name string) error {
	if name == "" || !validProjectNameRe.MatchString(name) {
		return fmt.Errorf("invalid project name: %q", name)
	}
	return nil
}

func (s *HistoryStore) historyDir() string {
	return filepath.Join(s.baseDir, "data", "composer_history")
}

func (s *HistoryStore) historyPath(projName string) (string, error) {
	if err := ValidateProjectName(projName); err != nil {
		return "", err
	}
	dir := filepath.Clean(s.historyDir())
	target := filepath.Clean(filepath.Join(dir, projName+".jsonl"))
	if !strings.HasPrefix(target, dir+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid history path")
	}
	return target, nil
}

func (s *HistoryStore) readAllLocked(path string) ([]HistoryEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []HistoryEntry
	scanner := bufio.NewScanner(f)
	// 放寬單行緩衝至 256 KB，確保包含 32 KB output 的 JSON 行能順利讀取
	scanner.Buffer(make([]byte, 64*1024), 256*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var item HistoryEntry
		if err := json.Unmarshal([]byte(line), &item); err == nil {
			entries = append(entries, item)
		}
	}
	if err := scanner.Err(); err != nil {
		return entries, err
	}
	return entries, nil
}

// Append 新增一筆執行紀錄，超過上限時自動淘汰最舊紀錄並原子寫回磁碟
func (s *HistoryStore) Append(entry HistoryEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path, err := s.historyPath(entry.Project)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	entries, err := s.readAllLocked(path)
	if err != nil {
		return err
	}

	entries = append(entries, entry)
	if s.maxEntries > 0 && len(entries) > s.maxEntries {
		entries = entries[len(entries)-s.maxEntries:]
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()

	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	writer := bufio.NewWriter(tmpFile)
	for _, item := range entries {
		raw, err := json.Marshal(item)
		if err != nil {
			_ = tmpFile.Close()
			return err
		}
		if _, err := writer.Write(raw); err != nil {
			_ = tmpFile.Close()
			return err
		}
		if err := writer.WriteByte('\n'); err != nil {
			_ = tmpFile.Close()
			return err
		}
	}
	if err := writer.Flush(); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}

// List 分頁讀取指定專案的執行紀錄（offset=0 為最新起算，回傳的 Items 依舊到新排列）
func (s *HistoryStore) List(projName string, offset, limit int) (*HistoryPage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	path, err := s.historyPath(projName)
	if err != nil {
		return nil, err
	}

	entries, err := s.readAllLocked(path)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 5
	}
	if offset < 0 {
		offset = 0
	}

	total := len(entries)
	end := total - offset
	if end <= 0 {
		return &HistoryPage{Items: []HistoryEntry{}, Total: total, HasMore: false}, nil
	}
	start := end - limit
	if start < 0 {
		start = 0
	}

	items := make([]HistoryEntry, end-start)
	copy(items, entries[start:end])

	return &HistoryPage{
		Items:   items,
		Total:   total,
		HasMore: start > 0,
	}, nil
}

// Clear 清除指定專案的所有 Composer 執行歷史
func (s *HistoryStore) Clear(projName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path, err := s.historyPath(projName)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
