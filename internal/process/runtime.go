package process

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	net "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
	"wincmp/internal/config"
	"wincmp/internal/preset"
)

// shellMetacharPattern 用於偵測命令注入的中繼字元
// 注意：反斜線 (\) 不在禁止清單中，因為 Windows 路徑必須使用反斜線（如 C:\Users\...）
var shellMetacharPattern = regexp.MustCompile("[&|;<>$\"!(){}\\[\\]`]")

// sanitizeRuntimeCommand 清理 Runtime 啟動指令中的危險中繼字元
// 允許的: 一般路徑、參數路徑、冒號、空格、斜線、反斜線（Windows路徑）、點、等號、百分比（佔位符）、減號、底線
// 禁止的: & | ; < > $ ` " ! ( ) { } [ ]
func sanitizeRuntimeCommand(cmd string) (string, error) {
	// 先處理佔位符，替換為暫時安全值以避免被誤判
	sanitized := cmd
	if shellMetacharPattern.MatchString(sanitized) {
		// 檢查是否為允許的特定模式（如 chcp 65001 >nul）
		// 允許 >nul（Windows 的靜默重導）但禁止其他重導
		temp := regexp.MustCompile(`>nul`).ReplaceAllString(sanitized, "___REDIRECT_NUL___")
		temp = regexp.MustCompile(`>NUL`).ReplaceAllString(temp, "___REDIRECT_NUL___")
		if shellMetacharPattern.MatchString(temp) {
			return "", fmt.Errorf("啟動指令含不安全的字元，已拒絕執行: %s", cmd)
		}
		sanitized = regexp.MustCompile(`___REDIRECT_NUL___`).ReplaceAllString(temp, ">nul")
	}
	return sanitized, nil
}

// validateDomainName 驗證域名只包含合法字元
func validateDomainName(domain string) bool {
	if domain == "" {
		return false
	}
	validDomain := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?)*$`)
	return validDomain.MatchString(domain)
}

// RuntimeServiceKey 取得 Runtime 服務的唯一鍵
func RuntimeServiceKey(projectID string) string {
	return "runtime_" + projectID
}

// CheckRuntimeEnv 檢查外部 Runtime 是否可用，回傳版本號和錯誤
func CheckRuntimeEnv(runtime string) (string, error) {
	switch runtime {
	case "python":
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "python", "-V")
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("Python 未安裝或未加入 PATH: %v", err)
		}
		return strings.TrimSpace(string(out)), nil
	case "go", "go_air", "go_run":
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "go", "version")
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("Go 未安裝或未加入 PATH: %v", err)
		}
		return strings.TrimSpace(string(out)), nil
	default:
		return "", nil
	}
}

// IsRuntimeTypeNeedEnvCheck 判斷是否需要檢查外部環境變數
func IsRuntimeTypeNeedEnvCheck(runtimeType string) bool {
	switch runtimeType {
	case "python", "go_air", "go_run", "go":
		return true
	default:
		return false
	}
}

// CheckSystemRuntimeAvailable 檢查系統 PATH 中是否有可用的 Node.js 或 Bun
// 回傳執行檔路徑和是否找到的布林值
func CheckSystemRuntimeAvailable(runtimeType string) (string, bool) {
	switch runtimeType {
	case "node":
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// 嘗試尋找 npm
		cmd := exec.CommandContext(ctx, "where", "npm")
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
		out, err := cmd.Output()
		if err == nil && len(out) > 0 {
			// 取第一個路徑（where 會回傳所有匹配項，每行一個）
			paths := strings.Split(strings.TrimSpace(string(out)), "\n")
			if len(paths) > 0 && paths[0] != "" {
				return strings.TrimSpace(paths[0]), true
			}
		}
		return "", false
	case "bun":
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// 嘗試尋找 bun
		cmd := exec.CommandContext(ctx, "where", "bun")
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
		out, err := cmd.Output()
		if err == nil && len(out) > 0 {
			paths := strings.Split(strings.TrimSpace(string(out)), "\n")
			if len(paths) > 0 && paths[0] != "" {
				return strings.TrimSpace(paths[0]), true
			}
		}
		return "", false
	default:
		return "", false
	}
}

// buildRuntimeCommand 根據 Preset 與 Runtime 類型建構執行指令
func buildRuntimeCommand(project config.ProjectConfig, exePath string) string {
	port := project.RuntimePort
	p := preset.GetPreset(project.Type)
	if port == 0 {
		port = p.DefaultPort
		if port == 0 {
			port = 3000
		}
	}
	host := "0.0.0.0"

	// 如果有自定義指令且非空，優先使用（Custom 類型或使用者手動覆寫）
	if project.Command != "" && (project.Type == "custom" || project.CommandDirty) {
		return replacePlaceholders(project.Command, port, host, project.RootPath, filepath.Dir(exePath))
	}

	// 根據專案類型與 Runtime 類型產生命令
	resolvedRuntime := project.RuntimeType
	if resolvedRuntime == "auto" {
		// auto resolve: 優先 bun，否則 node
		if strings.Contains(strings.ToLower(exePath), "bun") {
			resolvedRuntime = "bun"
		} else {
			resolvedRuntime = "node"
		}
	}

	switch project.Type {
	case "python", "python_django", "python_fastapi", "python_flask":
		return preset.BuildPythonCommandFromRoot(project.RootPath, port)

	case "go_api":
		if resolvedRuntime == "go_run" {
			return fmt.Sprintf("go run main.go")
		}
		return "air"

	case "pocketbase":
		return fmt.Sprintf("go run main.go serve --http=%s:%d", host, port)

	case "next", "nuxt", "astro", "vite":
		tmpl, ok := p.CommandTemplates[resolvedRuntime]
		if !ok {
			tmpl, ok = p.CommandTemplates["node"]
		}
		if !ok {
			tmpl, ok = p.CommandTemplates["bun"]
		}
		if ok && exePath != "" {
			cmd := replacePlaceholders(tmpl, port, host, project.RootPath, filepath.Dir(exePath))
			// 依據 exePath 判斷是否為 npm 或 bun
			isNPM := strings.Contains(strings.ToLower(exePath), "npm")
			if isNPM {
				// 保留 " run dev -- " 及之後的參數部分
				idx := strings.Index(cmd, " run ")
				if idx >= 0 {
					cmd = exePath + cmd[idx:]
				} else {
					cmd = exePath + " run dev"
				}
			} else {
				// bun: 保留 " run dev" 及之後的參數部分
				idx := strings.Index(cmd, " run ")
				if idx >= 0 {
					cmd = exePath + cmd[idx:]
				} else {
					cmd = exePath + " run dev"
				}
			}
			return cmd
		}

	default:
		// Custom 或未識別類型：如果有 Command 就用，否則嘗試以 exePath 啟動
		if project.Command != "" {
			return replacePlaceholders(project.Command, port, host, project.RootPath, filepath.Dir(exePath))
		}
		// Fallback: 使用舊邏輯
		return exePath + " run dev -- -p " + strconv.Itoa(port)
	}

	return ""
}

// replacePlaceholders 替換 WinCMP 佔位符
func replacePlaceholders(cmd string, port int, host, projectDir, binDir string) string {
	cmd = strings.ReplaceAll(cmd, "%PORT%", strconv.Itoa(port))
	cmd = strings.ReplaceAll(cmd, "%HOST%", host)
	cmd = strings.ReplaceAll(cmd, "%PROJECT_DIR%", projectDir)
	cmd = strings.ReplaceAll(cmd, "%BIN_DIR%", binDir)
	return cmd
}

// StartRuntime 啟動 Runtime 服務 (Node.js / Bun / Python / Go / Custom)
func (m *Manager) StartRuntime(project config.ProjectConfig, mode string, exePath string) error {
	serviceKey := RuntimeServiceKey(project.Name)

	if m.IsRunning(serviceKey) {
		m.log("runtime", "⚠️ [%s] Runtime 進程已經在執行中", project.Name)
		return fmt.Errorf("進程已在執行中")
	}

	rootPath := project.RootPath
	if rootPath == "" {
		m.errorLog("runtime", fmt.Sprintf("[%s] 專案根目錄未設定", project.Name), nil)
		return fmt.Errorf("專案根目錄未設定")
	}

	// 檢查外部 Runtime 是否可用
	if IsRuntimeTypeNeedEnvCheck(project.RuntimeType) {
		version, err := CheckRuntimeEnv(project.RuntimeType)
		if err != nil {
			m.errorLog("runtime", fmt.Sprintf("[%s] %v", project.Name, err), nil)
			return fmt.Errorf("%v", err)
		}
		if version != "" {
			m.log("runtime", "ℹ️ [%s] 偵測到 %s", project.Name, version)
		}
	}

	// 建構執行指令
	runtimeCmd := buildRuntimeCommand(project, exePath)
	if runtimeCmd == "" {
		m.errorLog("runtime", fmt.Sprintf("[%s] 無法建構啟動指令 (Runtime: %s, Type: %s)", project.Name, project.RuntimeType, project.Type), nil)
		return fmt.Errorf("無法建構啟動指令")
	}

	// 清理啟動指令中的危險中繼字元，防止命令注入
	var err error
	runtimeCmd, err = sanitizeRuntimeCommand(runtimeCmd)
	if err != nil {
		m.errorLog("runtime", fmt.Sprintf("[%s] 啟動指令安全性驗證失敗", project.Name), err)
		return fmt.Errorf("%v", err)
	}

	// 記錄日誌訊息方便偵錯
	binMode := "系統 PATH"
	if project.UseWinCMPBin {
		binMode = "WinCMP 內建路徑"
	}
	m.log("runtime", "🚀 [%s] 準備啟動。模式: %s", project.Name, binMode)
	m.log("runtime", "💻 [%s] 啟動指令: %s", project.Name, runtimeCmd)

	// 環境變數處理
	env := os.Environ()

	// UseWinCMPBin: 將 bin/ 內的對應執行檔路徑加到 PATH 前面
	bundledRuntimeTypes := map[string]bool{
		"node": true, "bun": true, "auto": true,
	}
	if project.UseWinCMPBin && bundledRuntimeTypes[project.RuntimeType] {
		if exePath != "" {
			binDir := filepath.Dir(exePath)
			if binDir != "." && binDir != "" {
				for i, e := range env {
					if strings.HasPrefix(strings.ToUpper(e), "PATH=") {
						env[i] = "PATH=" + binDir + ";" + e[5:]
						break
					}
				}
			}
		}
	}

	port := project.RuntimePort
	preset_ := preset.GetPreset(project.Type)
	if port == 0 {
		port = preset_.DefaultPort
		if port == 0 {
			port = 3000
		}
	}

	// 根據 Runtime 類型建構實際執行指令
	var startCmd *exec.Cmd
	switch project.RuntimeType {
	case "node", "bun", "auto":
		// Node.js / Bun 統一透過 cmd.exe 執行
		if mode == "Terminal" {
			innerCmd := "chcp 65001 >nul && " + runtimeCmd
			startCmd = exec.Command("cmd.exe", "/c", "start", "WinCMP Runtime: "+project.Name, "cmd.exe", "/k", innerCmd)
			startCmd.Dir = rootPath
			startCmd.Env = env
			startCmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
		} else {
			startCmd = exec.Command("cmd.exe", "/c", "chcp 65001 >nul && "+runtimeCmd)
			startCmd.Dir = rootPath
			startCmd.Env = env
			startCmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
		}
	case "python", "go_air", "go_run", "custom":
		// Python / Go / Custom 直接透過 cmd.exe 執行
		if mode == "Terminal" {
			innerCmd := "chcp 65001 >nul && " + runtimeCmd
			startCmd = exec.Command("cmd.exe", "/c", "start", "WinCMP Runtime: "+project.Name, "cmd.exe", "/k", innerCmd)
			startCmd.Dir = rootPath
			startCmd.Env = env
			startCmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
		} else {
			startCmd = exec.Command("cmd.exe", "/c", "chcp 65001 >nul && "+runtimeCmd)
			startCmd.Dir = rootPath
			startCmd.Env = env
			startCmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
		}
	default:
		// 預設行為
		if mode == "Terminal" {
			innerCmd := "chcp 65001 >nul && " + runtimeCmd
			startCmd = exec.Command("cmd.exe", "/c", "start", "WinCMP Runtime: "+project.Name, "cmd.exe", "/k", innerCmd)
			startCmd.Dir = rootPath
			startCmd.Env = env
			startCmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
		} else {
			startCmd = exec.Command("cmd.exe", "/c", "chcp 65001 >nul && "+runtimeCmd)
			startCmd.Dir = rootPath
			startCmd.Env = env
			startCmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
		}
	}

	runtimeLabel := project.RuntimeType
	if runtimeLabel == "" || runtimeLabel == "auto" {
		if strings.Contains(strings.ToLower(exePath), "bun") {
			runtimeLabel = "bun"
		} else {
			runtimeLabel = "node"
		}
	}

	if mode == "Terminal" {
		m.log("runtime", "▶️ [%s] 正在以 Terminal 模式啟動 %s", project.Name, runtimeLabel)

		if err := startCmd.Start(); err != nil {
			m.errorLog("runtime", fmt.Sprintf("[%s] 啟動失敗", project.Name), err)
			return err
		}

		go startCmd.Wait()

		// Terminal 模式用 Port 偵測管理生命週期
		m.registerRuntimeTerminal(serviceKey, project.Name, exePath, port)

		m.log("runtime", "✅ [%s] Terminal 模式已啟動 (Port: %d)", project.Name, port)
		return nil
	}

	// ─── Background 模式 ───
	m.log("runtime", "▶️ [%s] 正在以 Background 模式啟動 %s", project.Name, runtimeLabel)

	m.pipeRuntimeOutput(startCmd, "runtime", runtimeLabel+" ("+project.Name+")")

	if err := startCmd.Start(); err != nil {
		m.errorLog("runtime", fmt.Sprintf("[%s] 啟動失敗", project.Name), err)
		return err
	}

	m.register(serviceKey, runtimeLabel+" ("+project.Name+")", exePath, []*exec.Cmd{startCmd})

	// 背景監聽退出事件
	go func() {
		err := startCmd.Wait()
		if m.IsRunning(serviceKey) {
			if err != nil {
				m.errorLog("runtime", fmt.Sprintf("[%s] 異常退出", project.Name), err)
			} else {
				m.log("runtime", "ℹ️ [%s] Runtime 進程已結束", project.Name)
			}
			m.unregister(serviceKey)
		}
	}()

	// 背景 PID 動態更新
	go m.trackRuntimePIDs(serviceKey, project.Name, port, false)

	m.log("runtime", "✅ [%s] Background 模式已啟動 (PID: %d)", project.Name, startCmd.Process.Pid)
	return nil
}

// trackRuntimePIDs 定期透過 Port 偵測進程，並抓取整個進程樹的 PID
func (m *Manager) trackRuntimePIDs(serviceKey, projectName string, port int, isTerminal bool) {
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()

	time.Sleep(5 * time.Second)

	ctx := m.GetContext(serviceKey)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pid := findPIDByPort(port)
			if pid > 0 {
				pids := []int{pid}
				pids = append(pids, m.getChildPIDs(pid)...)
				m.UpdatePIDs(serviceKey, pids)
			} else if isTerminal {
				if m.IsRunning(serviceKey) {
					m.log("runtime", "ℹ️ [%s] Terminal 模式 Runtime 已停止 (Port %d 已釋放)", projectName, port)
					m.unregister(serviceKey)
				}
				return
			}
		}
	}
}

// registerRuntimeTerminal 為 Terminal 模式註冊服務狀態
func (m *Manager) registerRuntimeTerminal(serviceKey, projectName, exePath string, port int) {
	ctx, cancel := context.WithCancel(context.Background())

	runtimeLabel := "Runtime"
	m.mu.Lock()
	m.services[serviceKey] = &ServiceState{
		Name:      runtimeLabel + " (" + projectName + ")",
		Running:   true,
		ExePath:   exePath,
		Commands:  nil,
		PIDs:      nil,
		StartTime: time.Now(),
		Ctx:       ctx,
		Cancel:    cancel,
	}
	m.mu.Unlock()

	go m.trackRuntimePIDs(serviceKey, projectName, port, true)
}

// StopRuntime 停止 Runtime 服務
func (m *Manager) StopRuntime(project config.ProjectConfig) error {
	serviceKey := RuntimeServiceKey(project.Name)

	if !m.IsRunning(serviceKey) {
		return fmt.Errorf("服務 %s 未在運行", project.Name)
	}

	pids := m.GetPIDs(serviceKey)
	m.unregister(serviceKey)

	runtimeLabel := project.RuntimeType
	if runtimeLabel == "" {
		runtimeLabel = "node"
	}

	// 策略 1: 用已知 PID 殺進程樹 (Background 模式)
	for _, pid := range pids {
		killCmd := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid))
		killCmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
		if err := killCmd.Run(); err != nil {
			m.log("runtime", "⚠️ [%s] taskkill PID %d: %v", project.Name, pid, err)
		} else {
			m.log("runtime", "⏹️ [%s] 已停止 Runtime 進程樹 (PID: %d)", project.Name, pid)
		}
	}

	// 策略 2: 透過 Port 反查 PID 殺除 (Terminal 模式保底)
	port := project.RuntimePort
	if port > 0 {
		time.Sleep(500 * time.Millisecond)
		residualPID := findPIDByPort(port)
		if residualPID > 0 {
			killCmd := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(residualPID))
			killCmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
			if err := killCmd.Run(); err != nil {
				m.log("runtime", "⚠️ [%s] 備用 taskkill PID %d: %v", project.Name, residualPID, err)
			} else {
				m.log("runtime", "⏹️ [%s] 已透過 Port %d 停止殘留進程 (PID: %d)", project.Name, port, residualPID)
			}
		}
	}

	m.log("runtime", "⏹️ [%s] %s 已停止", project.Name, runtimeLabel)
	return nil
}

// pipeRuntimeOutput 將子程序的 stdout/stderr 透過管線傳送到 Terminal Logs
func (m *Manager) pipeRuntimeOutput(cmd *exec.Cmd, category string, serviceName string) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		m.errorLog(category, fmt.Sprintf("%s: 建立 stdout pipe 失敗", serviceName), err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		m.errorLog(category, fmt.Sprintf("%s: 建立 stderr pipe 失敗", serviceName), err)
	}

	go func() {
		if stdout == nil {
			return
		}
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				text := stripAnsi(string(buf[:n]))
				lines := strings.Split(text, "\n")
				for _, line := range lines {
					line = strings.TrimRight(line, "\r")
					if line != "" {
						m.log(category, "[%s] %s", serviceName, line)
					}
				}
			}
			if err != nil {
				return
			}
		}
	}()

	go func() {
		if stderr == nil {
			return
		}
		buf := make([]byte, 4096)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				text := stripAnsi(string(buf[:n]))
				lines := strings.Split(text, "\n")
				for _, line := range lines {
					line = strings.TrimRight(line, "\r")
					if line != "" {
						m.log(category, "[%s:err] %s", serviceName, line)
					}
				}
			}
			if err != nil {
				return
			}
		}
	}()
}

// findPIDByPort 透過 gopsutil 找到佔用某 Port 的 PID（取代 netstat shell 命令）
func findPIDByPort(port int) int {
	if port <= 0 {
		return 0
	}

	conns, err := net.Connections("tcp")
	if err != nil || len(conns) == 0 {
		return findPIDByPortFallback(port)
	}

	for _, conn := range conns {
		if conn.Status == "LISTEN" && int(conn.Laddr.Port) == port {
			return int(conn.Pid)
		}
	}
	return 0
}

// findPIDByPortFallback 降級回 netstat 方式查詢 PID（gopsutil 失敗時使用）
func findPIDByPortFallback(port int) int {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "cmd", "/c", fmt.Sprintf("netstat -ano | findstr :%d", port))
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
	out, err := cmd.CombinedOutput()
	if err != nil || len(out) == 0 {
		return 0
	}

	portSuffix := fmt.Sprintf(":%d", port)
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "LISTENING") {
			parts := strings.Fields(line)
			if len(parts) >= 5 {
				addr := parts[1]
				if strings.HasSuffix(addr, portSuffix) {
					pidStr := parts[len(parts)-1]
					pid, err := strconv.Atoi(pidStr)
					if err == nil && pid > 0 {
						return pid
					}
				}
			}
		}
	}
	return 0
}

// CheckRuntimeRunning 透過 netstat 檢測 Port 是否被佔用
func CheckRuntimeRunning(port int) bool {
	return findPIDByPort(port) > 0
}

// IsPortAvailable 檢查指定端口是否可用（未被佔用）
func IsPortAvailable(port int) bool {
	return !CheckRuntimeRunning(port)
}

// getChildPIDs 遞迴取得所有子進程 PID
func (m *Manager) getChildPIDs(parentPID int) []int {
	var pids []int
	if p, err := process.NewProcess(int32(parentPID)); err == nil {
		if children, err := p.Children(); err == nil {
			for _, c := range children {
				pids = append(pids, int(c.Pid))
				pids = append(pids, m.getChildPIDs(int(c.Pid))...)
			}
		}
	}
	return pids
}

// stripAnsi 移除 ANSI escape sequences
func stripAnsi(s string) string {
	var result strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' {
			i++
			if i < len(s) && s[i] == '[' {
				i++
				for i < len(s) && !(s[i] >= '@' && s[i] <= '~') {
					i++
				}
				if i < len(s) {
					i++
				}
			}
		} else {
			result.WriteByte(s[i])
			i++
		}
	}
	return result.String()
}

// ===== 向後相容: 舊版 API 別名 =====

// NodeServiceKey 向後相容，內部呼叫 RuntimeServiceKey
func NodeServiceKey(projectID string) string {
	return RuntimeServiceKey(projectID)
}

// StartNode 向後相容，內部呼叫 StartRuntime
func (m *Manager) StartNode(project config.ProjectConfig, mode string, exePath string) error {
	return m.StartRuntime(project, mode, exePath)
}

// StopNode 向後相容，內部呼叫 StopRuntime
func (m *Manager) StopNode(project config.ProjectConfig) error {
	return m.StopRuntime(project)
}

// CheckNodeRunning 向後相容，內部呼叫 CheckRuntimeRunning
func CheckNodeRunning(port int) bool {
	return CheckRuntimeRunning(port)
}
