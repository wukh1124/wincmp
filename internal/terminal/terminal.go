//go:build windows
package terminal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"sync"

	"github.com/UserExistsError/conpty"
)

// Session 代表一個與 ConPTY 綁定的終端連線會話
type Session struct {
	ID       string
	Cpty     *conpty.ConPty
	ProjName string
	Cwd      string
	Active   bool
	mu       sync.Mutex
}

// Manager 負責管理所有活躍中的終端會話
type Manager struct {
	sessions map[string]*Session
	mu       sync.Mutex
}

// NewManager 建立一個新的終端管理器
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
	}
}

// generateSessionID 產生隨機的 Session ID
func generateSessionID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "session_fallback"
	}
	return hex.EncodeToString(b)
}

// StartTerminal 啟動一個新的終端會話，並以指定的 shellPath 與工作路徑執行
func (m *Manager) StartTerminal(
	projName string,
	shellPath string,
	cwd string,
	cols int,
	rows int,
	onOutput func(data string),
	onExit func(),
) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. 建立 ConPTY 實例
	cpty, err := conpty.Start(
		shellPath,
		conpty.ConPtyDimensions(cols, rows),
		conpty.ConPtyWorkDir(cwd),
	)
	if err != nil {
		return "", fmt.Errorf("啟動 Pseudo Console 失敗: %w", err)
	}

	sessionID := generateSessionID()
	session := &Session{
		ID:       sessionID,
		Cpty:     cpty,
		ProjName: projName,
		Cwd:      cwd,
		Active:   true,
	}

	m.sessions[sessionID] = session

	// 2. 異步讀取終端輸出
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := cpty.Read(buf)
			if n > 0 {
				onOutput(string(buf[:n]))
			}
			if err != nil {
				break
			}
		}

		// 進程結束或連線中斷
		session.mu.Lock()
		session.Active = false
		session.mu.Unlock()

		m.mu.Lock()
		delete(m.sessions, sessionID)
		m.mu.Unlock()

		// 呼叫退出回呼函數
		onExit()
	}()

	return sessionID, nil
}

// Write 寫入資料到指定的終端會話
func (m *Manager) Write(sessionID string, data string) error {
	m.mu.Lock()
	session, exists := m.sessions[sessionID]
	m.mu.Unlock()

	if !exists {
		return fmt.Errorf("找不到終端會話: %s", sessionID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if !session.Active {
		return fmt.Errorf("終端會話已關閉: %s", sessionID)
	}

	_, err := io.WriteString(session.Cpty, data)
	if err != nil {
		return fmt.Errorf("寫入終端失敗: %w", err)
	}

	return nil
}

// Resize 調整指定終端會話的視窗大小 (Cols, Rows)
func (m *Manager) Resize(sessionID string, cols int, rows int) error {
	m.mu.Lock()
	session, exists := m.sessions[sessionID]
	m.mu.Unlock()

	if !exists {
		return fmt.Errorf("找不到終端會話: %s", sessionID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if !session.Active {
		return fmt.Errorf("終端會話已關閉: %s", sessionID)
	}

	if err := session.Cpty.Resize(cols, rows); err != nil {
		return fmt.Errorf("調整終端視窗大小失敗: %w", err)
	}

	return nil
}

// Stop 停止指定的終端會話，並關閉進程
func (m *Manager) Stop(sessionID string) {
	m.mu.Lock()
	session, exists := m.sessions[sessionID]
	m.mu.Unlock()

	if !exists {
		return
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.Active {
		session.Cpty.Close()
		session.Active = false
	}
}

// StopAll 結束並關閉所有活躍中的終端會話
func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, session := range m.sessions {
		session.mu.Lock()
		if session.Active {
			session.Cpty.Close()
			session.Active = false
		}
		session.mu.Unlock()
	}

	m.sessions = make(map[string]*Session)
}

// WaitBlock 異步等待進程結束 (可選調用)
func (session *Session) WaitBlock(ctx context.Context) (uint32, error) {
	return session.Cpty.Wait(ctx)
}
