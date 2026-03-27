package process

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"wincmp/internal/config"
)

// NodeServiceKey 取得 Node 服務的唯一鍵
func NodeServiceKey(projectID string) string {
	return "node_" + projectID
}

// StartNode 啟動 Node.js (npm run dev)
// exePath 為 npm.cmd 的完整路徑 (例如 bin/node/node-24.14.1/npm.cmd)
func (m *Manager) StartNode(project config.ProjectConfig, mode string, exePath string) error {
	serviceKey := NodeServiceKey(project.ID)

	if m.IsRunning(serviceKey) {
		m.log("node", "⚠️ [%s] Node 進程已經在執行中", project.Name)
		return fmt.Errorf("進程已在執行中")
	}

	rootPath := project.RootPath
	if rootPath == "" {
		m.errorLog("node", fmt.Sprintf("[%s] 專案根目錄未設定", project.Name), nil)
		return fmt.Errorf("專案根目錄未設定")
	}

	if exePath == "" {
		m.errorLog("node", fmt.Sprintf("[%s] 找不到對應的 Node 執行檔", project.Name), nil)
		return fmt.Errorf("找不到 Node 執行檔")
	}

	// 取得 npm.cmd 所在目錄，將其加入 PATH 前面，確保 node.exe 也能被找到
	npmDir := filepath.Dir(exePath)
	env := os.Environ()
	for i, e := range env {
		if strings.HasPrefix(strings.ToUpper(e), "PATH=") {
			env[i] = "PATH=" + npmDir + ";" + e[5:]
			break
		}
	}

	if mode == "Terminal" {
		// ─── Terminal 模式 ───────────────────────────────
		// 使用 cmd /c start 彈出獨立的終端機視窗
		// 父 cmd.exe 會馬上退出，改用 Port 偵測追蹤運行狀態
		m.log("node", "▶️ [%s] 正在以 Terminal 模式啟動 Node", project.Name)

		innerCmd := "chcp 65001 >nul && " + exePath + " run dev -- -p " + strconv.Itoa(project.NodePort)
		cmd := exec.Command("cmd.exe", "/c", "start", "WinCMP Node: "+project.Name, "cmd.exe", "/k", innerCmd)
		cmd.Dir = rootPath
		cmd.Env = env
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}

		if err := cmd.Start(); err != nil {
			m.errorLog("node", fmt.Sprintf("[%s] 啟動失敗", project.Name), err)
			return err
		}

		go cmd.Wait()

		// Terminal 模式用 Port 偵測管理生命週期
		m.registerTerminalNode(serviceKey, project.Name, exePath, project.NodePort)

		m.log("node", "✅ [%s] Terminal 模式已啟動 (Port: %d)", project.Name, project.NodePort)
		return nil
	}

	// ─── Background 模式 ───────────────────────────────
	m.log("node", "▶️ [%s] 正在以 Background 模式啟動 Node", project.Name)

	cmd := exec.Command("cmd.exe", "/c", "chcp 65001 >nul && "+exePath+" run dev -- -p "+strconv.Itoa(project.NodePort))
	cmd.Dir = rootPath
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}

	// 使用 pipeOutput 但透過 stripAnsi 過濾 ANSI escape codes
	m.pipeNodeOutput(cmd, "node", project.Name)

	if err := cmd.Start(); err != nil {
		m.errorLog("node", fmt.Sprintf("[%s] 啟動失敗", project.Name), err)
		return err
	}

	m.register(serviceKey, "Node ("+project.Name+")", exePath, []*exec.Cmd{cmd})

	// 背景監聽退出事件 — 但不報「異常退出」如果是我們主動 Stop 觸發的
	go func() {
		err := cmd.Wait()
		if m.IsRunning(serviceKey) {
			// 非預期退出（不是由 StopNode 觸發的）
			if err != nil {
				m.errorLog("node", fmt.Sprintf("[%s] 異常退出", project.Name), err)
			} else {
				m.log("node", "ℹ️ [%s] Node 進程已結束", project.Name)
			}
			m.unregister(serviceKey)
		}
	}()

	m.log("node", "✅ [%s] Background 模式已啟動 (PID: %d)", project.Name, cmd.Process.Pid)
	return nil
}

// pipeNodeOutput 將子程序的 stdout/stderr 透過管線傳送到 Terminal Logs
// 會過濾 ANSI escape codes 避免日誌亂碼
func (m *Manager) pipeNodeOutput(cmd *exec.Cmd, category string, serviceName string) {
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	go func() {
		if stdout == nil {
			return
		}
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				text := stripAnsi(string(buf[:n]))
				// 按行輸出，避免半行覆蓋
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

// stripAnsi 移除 ANSI escape sequences（如 \x1b[32m 彩色碼、游標移動等）
func stripAnsi(s string) string {
	var result strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' {
			// 跳過 ESC 序列
			i++
			if i < len(s) && s[i] == '[' {
				i++
				// 跳至序列終止字元 (@ 到 ~)
				for i < len(s) && !(s[i] >= '@' && s[i] <= '~') {
					i++
				}
				if i < len(s) {
					i++ // 跳過終止字元
				}
			}
		} else {
			result.WriteByte(s[i])
			i++
		}
	}
	return result.String()
}

// registerTerminalNode 為 Terminal 模式註冊服務狀態，透過 Port 偵測生命週期
func (m *Manager) registerTerminalNode(serviceKey, projectName, exePath string, port int) {
	ctx, cancel := context.WithCancel(context.Background())

	m.mu.Lock()
	m.services[serviceKey] = &ServiceState{
		Name:      "Node (" + projectName + ")",
		Running:   true,
		ExePath:   exePath,
		Commands:  nil,
		PIDs:      nil,
		StartTime: time.Now(),
		Ctx:       ctx,
		Cancel:    cancel,
	}
	m.mu.Unlock()

	// 背景定期檢查 Port 是否仍被佔用
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		// 先等 Node 啟動完成
		time.Sleep(8 * time.Second)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !CheckNodeRunning(port) {
					if m.IsRunning(serviceKey) {
						m.log("node", "ℹ️ [%s] Terminal 模式 Node 已停止 (Port %d 已釋放)", projectName, port)
						m.unregister(serviceKey)
					}
					return
				}
			}
		}
	}()
}

// StopNode 停止 Node.js
func (m *Manager) StopNode(project config.ProjectConfig) error {
	serviceKey := NodeServiceKey(project.ID)

	if !m.IsRunning(serviceKey) {
		return fmt.Errorf("服務 %s 未在運行", project.Name)
	}

	// 先取出 PID（unregister 後會清空）
	pids := m.GetPIDs(serviceKey)

	// 標記為「正在停止」，避免 waitForExit 報異常退出
	m.unregister(serviceKey)

	// 策略 1: 用已知 PID 殺進程樹 (Background 模式)
	for _, pid := range pids {
		killCmd := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid))
		killCmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
		if err := killCmd.Run(); err != nil {
			// 進程可能已結束，不一定是真正的錯誤
			m.log("node", "⚠️ [%s] taskkill PID %d: %v", project.Name, pid, err)
		} else {
			m.log("node", "⏹️ [%s] 已停止 Node 進程樹 (PID: %d)", project.Name, pid)
		}
	}

	// 策略 2: 透過 Port 反查 PID 殺除 (Terminal 模式 或 Background 沒殺乾淨時的保底)
	if project.NodePort > 0 {
		// 等一下讓 taskkill 先完成
		time.Sleep(500 * time.Millisecond)
		nodePID := findNodePIDByPort(project.NodePort)
		if nodePID > 0 {
			killCmd := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(nodePID))
			killCmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
			if err := killCmd.Run(); err != nil {
				m.log("node", "⚠️ [%s] 備用 taskkill PID %d: %v", project.Name, nodePID, err)
			} else {
				m.log("node", "⏹️ [%s] 已透過 Port %d 停止殘留進程 (PID: %d)", project.Name, project.NodePort, nodePID)
			}
		}
	}

	m.log("node", "⏹️ [%s] Node 已停止", project.Name)
	return nil
}

// findNodePIDByPort 透過 netstat 找到佔用某 Port 的 PID
func findNodePIDByPort(port int) int {
	if port <= 0 {
		return 0
	}

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
				// 精確匹配 Port (避免 :3000 匹配到 :30001)
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

// CheckNodeRunning 透過 netstat 檢測 Node 是否佔用了給定的 port
func CheckNodeRunning(port int) bool {
	return findNodePIDByPort(port) > 0
}

// IsPortAvailable 檢查指定端口是否可用（未被佔用）
func IsPortAvailable(port int) bool {
	return !CheckNodeRunning(port)
}
