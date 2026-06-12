package process

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// ServiceState 表示一個被管理的服務狀態
type ServiceState struct {
	Name      string      // 服務名稱 (如 "Caddy", "MariaDB", "PHP-CGI 8.2.30")
	Running   bool        // 是否正在運行
	ExePath   string      // 啟動時使用的執行檔路徑 (用於 Reload 等操作)
	Commands  []*exec.Cmd // 該服務的所有子程序（PHP 可能有多個）
	PIDs      []int       // 所有子程序的 PID
	StartTime time.Time
	Ctx       context.Context
	Cancel    context.CancelFunc
	JobHandle windows.Handle // 該服務專屬的 Windows Job Object
}

// LogFunc 用於傳送日誌到 UI 的回調函數型別
type LogFunc func(category string, msg string)

// ErrorLogFunc 用於傳送錯誤日誌的回調函數型別
type ErrorLogFunc func(category string, contextMsg string, err error)

// Manager 程序管理器，負責管理所有子程序的生命週期
type Manager struct {
	mu       sync.Mutex
	services map[string]*ServiceState // key: 服務唯一識別 (如 "caddy", "mariadb-11.4", "php-8.2.30")
	baseDir  string                   // 專案根目錄（所有路徑以此為基準）
	logFn    LogFunc                  // 日誌回調
	errLogFn ErrorLogFunc             // 錯誤日誌回調
}

// NewManager 建立新的程序管理器
func NewManager(baseDir string, logFn LogFunc, errLogFn ErrorLogFunc) *Manager {
	m := &Manager{
		services: make(map[string]*ServiceState),
		baseDir:  baseDir,
		logFn:    logFn,
		errLogFn: errLogFn,
	}

	// 綁定當前程序到 Job Object 以確保非預期崩潰時能自動清理子處理程序
	if err := initJobObject(); err != nil {
		m.errorLog("system", "初始化 Windows Job Object 限制失敗 (子程序可能無法自動關閉)", err)
	}

	return m
}

// log 透過回調函數發送一般日誌
func (m *Manager) log(category string, format string, args ...interface{}) {
	if m.logFn != nil {
		m.logFn(category, fmt.Sprintf(format, args...))
	}
}

// errorLog 透過回調函數發送錯誤日誌
func (m *Manager) errorLog(category string, contextMsg string, err error) {
	if m.errLogFn != nil {
		m.errLogFn(category, contextMsg, err)
	} else if m.logFn != nil {
		// Fallback to regular log if error log not provided
		m.logFn(category, fmt.Sprintf("❌ %s: %v", contextMsg, err))
	}
}

// GetBaseDir 取得專案根目錄
func (m *Manager) GetBaseDir() string {
	return m.baseDir
}

// IsRunning 檢查指定服務是否正在運行
func (m *Manager) IsRunning(serviceKey string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.services[serviceKey]
	if !exists {
		return false
	}
	return state.Running
}

// GetPIDs 取得指定服務的所有 PID
func (m *Manager) GetPIDs(serviceKey string) []int {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.services[serviceKey]
	if !exists {
		return nil
	}
	return state.PIDs
}

func (m *Manager) GetAllPIDs() []int {
	m.mu.Lock()
	defer m.mu.Unlock()

	var all []int
	for _, state := range m.services {
		for _, pid := range state.PIDs {
			if pid > 0 {
				all = append(all, pid)
			}
		}
	}
	return all
}

// ServiceInfo 用於 tooltip 明細顯示的服務資訊
type ServiceInfo struct {
	Key  string // 服務識別 key（如 "caddy", "php-8.2.30", "node_abc123"）
	Name string // 顯示名稱（如 "Caddy (2.11.1)", "Node (broker_portal)"）
	PIDs []int  // 該服務所有 PID
}

// GetServiceBreakdown 取得所有正在運行的服務及其 PID 列表（按服務分類）
func (m *Manager) GetServiceBreakdown() []ServiceInfo {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []ServiceInfo
	for key, state := range m.services {
		if !state.Running || len(state.PIDs) == 0 {
			continue
		}
		pids := make([]int, 0, len(state.PIDs))
		for _, pid := range state.PIDs {
			if pid > 0 {
				pids = append(pids, pid)
			}
		}
		if len(pids) > 0 {
			result = append(result, ServiceInfo{
				Key:  key,
				Name: state.Name,
				PIDs: pids,
			})
		}
	}
	return result
}

// register 註冊一個新的服務狀態
func (m *Manager) register(serviceKey, name, exePath string, cmds []*exec.Cmd) {
	pids := make([]int, len(cmds))
	for i, cmd := range cmds {
		if cmd.Process != nil {
			pids[i] = cmd.Process.Pid
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	m.mu.Lock()
	m.services[serviceKey] = &ServiceState{
		Name:      name,
		Running:   true,
		ExePath:   exePath,
		Commands:  cmds,
		PIDs:      pids,
		StartTime: time.Now(),
		Ctx:       ctx,
		Cancel:    cancel,
	}
	m.mu.Unlock()
}

// registerWithJob 註冊一個帶有 Job Object 的新服務狀態
func (m *Manager) registerWithJob(serviceKey, name, exePath string, cmds []*exec.Cmd, job windows.Handle) {
	pids := make([]int, len(cmds))
	for i, cmd := range cmds {
		if cmd.Process != nil {
			pids[i] = cmd.Process.Pid
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	m.mu.Lock()
	m.services[serviceKey] = &ServiceState{
		Name:      name,
		Running:   true,
		ExePath:   exePath,
		Commands:  cmds,
		PIDs:      pids,
		StartTime: time.Now(),
		Ctx:       ctx,
		Cancel:    cancel,
		JobHandle: job,
	}
	m.mu.Unlock()
}

// UpdatePIDs 手動更新指定服務的 PID 列表（用於啟動後才能確定的進程）
func (m *Manager) UpdatePIDs(serviceKey string, pids []int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if state, exists := m.services[serviceKey]; exists {
		state.PIDs = pids
	}
}

// RemovePID 從指定服務的 PID 列表中移除單一 PID（用於子程序異常退出時更新狀態）
func (m *Manager) RemovePID(serviceKey string, pid int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if state, exists := m.services[serviceKey]; exists {
		filtered := make([]int, 0, len(state.PIDs))
		for _, p := range state.PIDs {
			if p != pid {
				filtered = append(filtered, p)
			}
		}
		state.PIDs = filtered
	}
}

// GetContext 取得服務的 context (用於監聽服務結束)
func (m *Manager) GetContext(serviceKey string) context.Context {
	m.mu.Lock()
	defer m.mu.Unlock()
	if state, exists := m.services[serviceKey]; exists {
		return state.Ctx
	}
	return context.Background()
}

// GetStartTime 取得服務啟動時間
func (m *Manager) GetStartTime(serviceKey string) time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	if state, exists := m.services[serviceKey]; exists {
		return state.StartTime
	}
	return time.Time{}
}

// GetExePath 取得指定服務的執行檔路徑
func (m *Manager) GetExePath(serviceKey string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.services[serviceKey]
	if !exists {
		return ""
	}
	return state.ExePath
}

// unregister 移除服務狀態
func (m *Manager) unregister(serviceKey string) {
	m.mu.Lock()
	if state, exists := m.services[serviceKey]; exists {
		if state.Cancel != nil {
			state.Cancel()
		}
		if state.JobHandle != 0 {
			windows.CloseHandle(state.JobHandle)
			state.JobHandle = 0
		}
		state.Running = false
		state.Commands = nil
		state.PIDs = nil
	}
	m.mu.Unlock()
}

// StopAll 停止所有正在運行的服務（程式關閉時呼叫）
func (m *Manager) StopAll() {
	m.mu.Lock()
	keys := make([]string, 0, len(m.services))
	for k, s := range m.services {
		if s.Running {
			keys = append(keys, k)
		}
	}
	m.mu.Unlock()

	for _, key := range keys {
		m.stopService(key)
	}
}

// stopService 通用停止服務邏輯
func (m *Manager) stopService(serviceKey string) error {
	m.mu.Lock()
	state, exists := m.services[serviceKey]
	if !exists || !state.Running {
		m.mu.Unlock()
		return fmt.Errorf("服務 %s 未在運行", serviceKey)
	}
	cmds := state.Commands
	m.mu.Unlock()

	var lastErr error
	for _, cmd := range cmds {
		if cmd != nil && cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil {
				// 程序可能已經結束
				if !isProcessFinished(err) {
					lastErr = err
					m.errorLog("system", fmt.Sprintf("無法終止程序 PID %d", cmd.Process.Pid), err)
				}
			}
		}
	}

	m.unregister(serviceKey)
	return lastErr
}

// createCommand 建立一個以 baseDir 為工作目錄的 exec.Cmd
func (m *Manager) createCommand(exePath string, args ...string) *exec.Cmd {
	cmd := exec.Command(exePath, args...)
	cmd.Dir = m.baseDir
	// 繼承環境變數
	cmd.Env = os.Environ()
	// Windows 下完全無視窗 (防止閃視窗)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
	return cmd
}

// pipeOutput 將子程序的 stdout/stderr 透過管線傳送到 Terminal Logs
func (m *Manager) pipeOutput(cmd *exec.Cmd, category string, serviceName string) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		m.errorLog(category, fmt.Sprintf("%s: 建立 stdout pipe 失敗", serviceName), err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		m.errorLog(category, fmt.Sprintf("%s: 建立 stderr pipe 失敗", serviceName), err)
	}

	// 讀取 stdout
	go func() {
		if stdout == nil {
			return
		}
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			m.log(category, "[%s] %s", serviceName, scanner.Text())
		}
	}()

	// 讀取 stderr
	go func() {
		if stderr == nil {
			return
		}
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			m.log(category, "[%s:err] %s", serviceName, scanner.Text())
		}
	}()
}

// waitForExit 在 goroutine 中等待程序退出，並更新狀態
func (m *Manager) waitForExit(cmd *exec.Cmd, serviceKey, category, serviceName string) {
	err := cmd.Wait()
	if m.IsRunning(serviceKey) {
		// 非預期退出（不是由 Stop 觸發的）
		if err != nil {
			m.errorLog(category, fmt.Sprintf("%s 異常退出", serviceName), err)
		} else {
			m.log(category, "ℹ️ %s 已退出", serviceName)
		}
		m.unregister(serviceKey)
	}
}

// isProcessFinished 判斷錯誤是否表示程序已經結束
func isProcessFinished(err error) bool {
	if err == nil {
		return false
	}
	// Go 1.20+ 提供 os.ErrProcessDone
	if errors.Is(err, os.ErrProcessDone) {
		return true
	}
	// 回相容：檢查是否為 "syscall: Wait..." 或各種已完成程序錯誤
	var sysErr syscall.Errno
	if errors.As(err, &sysErr) {
		return false // syscall 錯誤通常不是「程序已完成」
	}
	// 最終回退：字串比對（只對未型別化的錯誤）
	msg := err.Error()
	return msg == "os: process already finished" ||
		msg == "os: process already released"
}
