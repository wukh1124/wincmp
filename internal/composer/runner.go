package composer

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"wincmp/internal/i18n"
)

const (
	// DefaultHardTimeout 單次 Composer 指令最長執行時間 (10 分鐘)
	DefaultHardTimeout = 600 * time.Second
	// DefaultIdleTimeout 連續無輸出閒置超時時間 (3 分鐘)
	DefaultIdleTimeout = 180 * time.Second
)

// RunOptions 描述單次 Composer 執行參數
type RunOptions struct {
	ProjectName     string
	WorkDir         string
	PHPExe          string
	PHPVersion      string
	ComposerPhar    string
	ComposerVersion string
	Args            []string
	// ExtraArgsPrefix 測試用內部前置參數（正常流程為 nil）
	ExtraArgsPrefix []string
}

type activeRun struct {
	id         string
	project    string
	cmd        *exec.Cmd
	jobHandle  windows.Handle
	cancelOnce sync.Once
	cancelCh   chan struct{}
	cancelType string // "stopped" | "hard_timeout" | "idle_timeout"
	mu         sync.Mutex
}

func (r *activeRun) requestCancel(reason string) {
	r.cancelOnce.Do(func() {
		r.mu.Lock()
		r.cancelType = reason
		r.mu.Unlock()
		close(r.cancelCh)
		r.terminateTree()
	})
}

func (r *activeRun) getCancelType() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cancelType
}

func (r *activeRun) terminateTree() {
	r.mu.Lock()
	job := r.jobHandle
	r.jobHandle = 0
	var pid int
	if r.cmd != nil && r.cmd.Process != nil {
		pid = r.cmd.Process.Pid
	}
	r.mu.Unlock()

	if job != 0 {
		_ = windows.TerminateJobObject(job, 1)
		_ = windows.CloseHandle(job)
	}
	if pid > 0 {
		killCmd := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid))
		killCmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
		_ = killCmd.Run()
	}
}

// Manager 管理進行中的 Composer 執行工作與歷史紀錄
type Manager struct {
	baseDir     string
	history     *HistoryStore
	HardTimeout time.Duration
	IdleTimeout time.Duration

	mu   sync.Mutex
	runs map[string]*activeRun
}

// NewManager 建立 Composer 執行管理器
func NewManager(baseDir string) *Manager {
	return &Manager{
		baseDir:     baseDir,
		history:     NewHistoryStore(baseDir),
		HardTimeout: DefaultHardTimeout,
		IdleTimeout: DefaultIdleTimeout,
		runs:        make(map[string]*activeRun),
	}
}

// History 回傳歷史紀錄儲存器
func (m *Manager) History() *HistoryStore {
	return m.history
}

// Start 非同步啟動一次 Composer 指令執行並回傳 runID
func (m *Manager) Start(
	opts RunOptions,
	onOutput func(runID string, chunk string),
	onExit func(runID string, entry HistoryEntry),
) (string, error) {
	if err := ValidateProjectName(opts.ProjectName); err != nil {
		return "", err
	}
	if len(opts.Args) == 0 {
		return "", fmt.Errorf("%s", i18n.T("請輸入要執行的 Composer 指令"))
	}
	if info, err := os.Stat(opts.WorkDir); err != nil || !info.IsDir() {
		return "", fmt.Errorf("%s: %s", i18n.T("找不到有效的專案目錄"), opts.WorkDir)
	}
	if _, err := os.Stat(opts.PHPExe); err != nil {
		return "", fmt.Errorf("%s: %s", i18n.T("找不到對應的 php.exe"), opts.PHPExe)
	}
	if opts.ComposerPhar != "" {
		if _, err := os.Stat(opts.ComposerPhar); err != nil {
			return "", fmt.Errorf("%s: %s", i18n.T("找不到 composer.phar"), opts.ComposerPhar)
		}
	}

	runID := newRunID()
	startedAt := time.Now()

	cmdArgs := buildPHPComposerArgs(m.baseDir, opts)

	cmd := exec.Command(opts.PHPExe, cmdArgs...)
	cmd.Dir = opts.WorkDir
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	cmd.Env = buildIsolatedEnv(filepath.Dir(opts.PHPExe), m.HardTimeout)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("建立 stdout 管線失敗: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("建立 stderr 管線失敗: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T("啟動 Composer 進程失敗"), err)
	}

	jobHandle := attachProcessToJob(cmd.Process.Pid)
	run := &activeRun{
		id:        runID,
		project:   opts.ProjectName,
		cmd:       cmd,
		jobHandle: jobHandle,
		cancelCh:  make(chan struct{}),
	}

	m.mu.Lock()
	m.runs[runID] = run
	m.mu.Unlock()

	go m.waitAndRecord(run, opts, startedAt, stdoutPipe, stderrPipe, onOutput, onExit)
	return runID, nil
}

// Stop 主動停止指定 runID 的執行工作
func (m *Manager) Stop(runID string) error {
	m.mu.Lock()
	run, ok := m.runs[runID]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("%s", i18n.T("該執行工作已結束或不存在"))
	}
	run.requestCancel("stopped")
	return nil
}

// StopAll 在程式關閉時停止所有執行中的 Composer 工作
func (m *Manager) StopAll() {
	m.mu.Lock()
	list := make([]*activeRun, 0, len(m.runs))
	for _, r := range m.runs {
		list = append(list, r)
	}
	m.mu.Unlock()

	for _, r := range list {
		r.requestCancel("stopped")
	}
}

func (m *Manager) waitAndRecord(
	run *activeRun,
	opts RunOptions,
	startedAt time.Time,
	stdoutPipe io.ReadCloser,
	stderrPipe io.ReadCloser,
	onOutput func(runID string, chunk string),
	onExit func(runID string, entry HistoryEntry),
) {
	defer func() {
		m.mu.Lock()
		delete(m.runs, run.id)
		m.mu.Unlock()

		run.mu.Lock()
		job := run.jobHandle
		run.jobHandle = 0
		run.mu.Unlock()
		if job != 0 {
			_ = windows.CloseHandle(job)
		}
	}()

	buf := NewHeadTailBuffer(HeadLimitBytes, TailLimitBytes)
	activityCh := make(chan struct{}, 32)
	var outMu sync.Mutex
	var pipeWG sync.WaitGroup

	readStream := func(r io.Reader) {
		defer pipeWG.Done()
		chunk := make([]byte, 4096)
		for {
			n, err := r.Read(chunk)
			if n > 0 {
				data := chunk[:n]
				buf.Write(data)
				select {
				case activityCh <- struct{}{}:
				default:
				}
				if onOutput != nil {
					outMu.Lock()
					onOutput(run.id, string(data))
					outMu.Unlock()
				}
			}
			if err != nil {
				return
			}
		}
	}

	pipeWG.Add(2)
	go readStream(stdoutPipe)
	go readStream(stderrPipe)

	waitDone := make(chan error, 1)
	go func() {
		pipeWG.Wait()
		waitDone <- run.cmd.Wait()
	}()

	hardDur := m.HardTimeout
	if hardDur <= 0 {
		hardDur = DefaultHardTimeout
	}
	idleDur := m.IdleTimeout
	if idleDur <= 0 {
		idleDur = DefaultIdleTimeout
	}

	hardTimer := time.NewTimer(hardDur)
	idleTimer := time.NewTimer(idleDur)
	defer hardTimer.Stop()
	defer idleTimer.Stop()

	var waitErr error
loop:
	for {
		select {
		case waitErr = <-waitDone:
			break loop
		case <-activityCh:
			if !idleTimer.Stop() {
				select {
				case <-idleTimer.C:
				default:
				}
			}
			idleTimer.Reset(idleDur)
		case <-hardTimer.C:
			run.requestCancel("hard_timeout")
			diag := "\r\n\x1b[31m" + i18n.Tfmt("[WinCMP] 執行總時間超過 %d 秒上限，已自動終止進程。", int(hardDur.Seconds())) + "\x1b[0m\r\n"
			buf.Write([]byte(diag))
			if onOutput != nil {
				outMu.Lock()
				onOutput(run.id, diag)
				outMu.Unlock()
			}
		case <-idleTimer.C:
			run.requestCancel("idle_timeout")
			diag := "\r\n\x1b[33m" + i18n.Tfmt("[WinCMP] 超過 %d 秒無任何輸出，疑似網路卡死或等待互動輸入，已自動終止。", int(idleDur.Seconds())) + "\x1b[0m\r\n"
			buf.Write([]byte(diag))
			if onOutput != nil {
				outMu.Lock()
				onOutput(run.id, diag)
				outMu.Unlock()
			}
		case <-run.cancelCh:
			// 等待 waitDone 收尾，避免重複觸發
			waitErr = <-waitDone
			break loop
		}
	}

	finishedAt := time.Now()
	exitCode := 0
	status := "success"
	errorReason := ""

	cancelType := run.getCancelType()
	switch cancelType {
	case "stopped":
		status = "stopped"
		errorReason = "stopped_by_user"
		exitCode = -1
	case "hard_timeout":
		status = "timeout"
		errorReason = fmt.Sprintf("hard_timeout_%ds", int(hardDur.Seconds()))
		exitCode = -1
	case "idle_timeout":
		status = "timeout"
		errorReason = fmt.Sprintf("idle_timeout_%ds", int(idleDur.Seconds()))
		exitCode = -1
	default:
		if waitErr != nil {
			status = "error"
			var exitErr *exec.ExitError
			if errors.As(waitErr, &exitErr) {
				exitCode = exitErr.ExitCode()
				errorReason = fmt.Sprintf("exit_code_%d", exitCode)
			} else {
				exitCode = -1
				errorReason = waitErr.Error()
			}
		}
	}

	output, truncated, origBytes, origLines := buf.Result()
	entry := HistoryEntry{
		ID:              run.id,
		Project:         opts.ProjectName,
		Command:         strings.Join(opts.Args, " "),
		Args:            opts.Args,
		PHPVersion:      opts.PHPVersion,
		ComposerVersion: opts.ComposerVersion,
		StartedAt:       startedAt.Format(time.RFC3339),
		FinishedAt:      finishedAt.Format(time.RFC3339),
		DurationMs:      finishedAt.Sub(startedAt).Milliseconds(),
		ExitCode:        exitCode,
		Status:          status,
		ErrorReason:     errorReason,
		Output:          output,
		Truncated:       truncated,
		OriginalBytes:   origBytes,
		OriginalLines:   origLines,
	}

	_ = m.history.Append(entry)
	if onExit != nil {
		onExit(run.id, entry)
	}
}

func buildPHPComposerArgs(baseDir string, opts RunOptions) []string {
	var cmdArgs []string
	if len(opts.ExtraArgsPrefix) > 0 {
		cmdArgs = append(cmdArgs, opts.ExtraArgsPrefix...)
		cmdArgs = append(cmdArgs, opts.Args...)
		return cmdArgs
	}

	phpDir := filepath.Dir(opts.PHPExe)
	extDir := filepath.Join(phpDir, "ext")
	phpIniPath := filepath.Join(baseDir, "conf", "php", "php.ini")
	cacertPath := filepath.Join(baseDir, "conf", "ssl", "cacert.pem")

	// 注入 WinCMP 統一的 php.ini、對應版本的 extension_dir 與 SSL CA 憑證，
	// 確保 openssl / curl / mbstring / zip 等擴充模組在 PHP CLI 模式下正常載入。
	if _, err := os.Stat(phpIniPath); err == nil {
		cmdArgs = append(cmdArgs, "-c", phpIniPath)
	}
	if info, err := os.Stat(extDir); err == nil && info.IsDir() {
		cmdArgs = append(cmdArgs, "-d", "extension_dir="+extDir)
	}
	if _, err := os.Stat(cacertPath); err == nil {
		cacertSlashPath := filepath.ToSlash(cacertPath)
		cmdArgs = append(cmdArgs,
			"-d", "openssl.cafile="+cacertSlashPath,
			"-d", "curl.cainfo="+cacertSlashPath,
		)
	}

	cmdArgs = append(cmdArgs, "-d", "memory_limit=-1", opts.ComposerPhar, "--ansi")
	cmdArgs = append(cmdArgs, opts.Args...)
	return cmdArgs
}

func buildIsolatedEnv(phpDir string, hardTimeout time.Duration) []string {
	timeoutSec := int(hardTimeout.Seconds())
	if timeoutSec <= 0 {
		timeoutSec = int(DefaultHardTimeout.Seconds())
	}

	env := os.Environ()
	out := make([]string, 0, len(env)+4)
	pathUpdated := false
	for _, kv := range env {
		upper := strings.ToUpper(kv)
		if strings.HasPrefix(upper, "PATH=") {
			origPath := kv[5:]
			out = append(out, "PATH="+phpDir+string(os.PathListSeparator)+origPath)
			pathUpdated = true
			continue
		}
		if strings.HasPrefix(upper, "COMPOSER_NO_INTERACTION=") ||
			strings.HasPrefix(upper, "COMPOSER_PROCESS_TIMEOUT=") {
			continue
		}
		out = append(out, kv)
	}
	if !pathUpdated {
		out = append(out, "PATH="+phpDir)
	}
	out = append(out,
		"COMPOSER_NO_INTERACTION=1",
		fmt.Sprintf("COMPOSER_PROCESS_TIMEOUT=%d", timeoutSec),
		"TERM=xterm-256color",
		"ANSICON=1",
	)
	return out
}

func attachProcessToJob(pid int) windows.Handle {
	if pid <= 0 {
		return 0
	}
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0
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
		_ = windows.CloseHandle(job)
		return 0
	}
	hProc, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		_ = windows.CloseHandle(job)
		return 0
	}
	defer windows.CloseHandle(hProc)

	if err := windows.AssignProcessToJobObject(job, hProc); err != nil {
		_ = windows.CloseHandle(job)
		return 0
	}
	return job
}

func newRunID() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%d-%s", time.Now().UnixMilli(), hex.EncodeToString(b[:]))
}
