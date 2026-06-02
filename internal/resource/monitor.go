package resource

import (
	"context"
	"fmt"
	"math"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"github.com/shirou/gopsutil/v3/process"

	"wincmp/internal/i18n"
	procmgr "wincmp/internal/process"
)

const (
	EnableStackTotal = true

	updateInterval = time.Second
)

type Monitor struct {
	pid      int
	proc     *process.Process
	cpuCount int
	ticker   *time.Ticker
	cancel   context.CancelFunc
	mu       sync.RWMutex

	procMgr interface{}

	// tooltip 明細節流
	breakdownMu   sync.Mutex
	lastBreakdown string
	lastBreakTime time.Time

	// PID 快取：僅在 PID 列表變化時重新建立 process 物件
	cachedPIDs    []int
	cachedProcs   []*process.Process
	lastPIDUpdate time.Time
}

func NewAppResourceMonitor(procMgr interface{}) *Monitor {
	pid := os.Getpid()
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		proc = nil
	}

	return &Monitor{
		pid:      pid,
		proc:     proc,
		cpuCount: runtime.NumCPU(),
		ticker:   time.NewTicker(updateInterval),
		procMgr:  procMgr,
	}
}

type hoverDetector interface {
	IsHovered() bool
}

func (m *Monitor) Start(label fyne.CanvasObject) {
	statusLabel, ok := label.(interface{ SetText(string) })
	if !ok {
		return
	}

	// tooltip 動態更新（需要 SetToolTip 介面）
	tooltipLabel, hasTooltip := label.(interface{ SetToolTip(string) })

	// 懸停偵測介面
	detector, hasDetector := label.(hoverDetector)

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.ticker.C:
			m.mu.RLock()
			ramStr, cpuStr, stackStr := m.fetchResourceData()
			m.mu.RUnlock()

			var text string
			if EnableStackTotal && stackStr != "" {
				text = fmt.Sprintf("WinCMP RAM: %s | CPU: %s | Stack Total: %s", ramStr, cpuStr, stackStr)
			} else {
				text = fmt.Sprintf("WinCMP RAM: %s | CPU: %s", ramStr, cpuStr)
			}

			// 計算 tooltip 明細（只有在懸停且啟用時才計算，節省資源）
			var tooltip string
			if hasTooltip && EnableStackTotal && (!hasDetector || detector.IsHovered()) {
				tooltip = m.FetchStackBreakdown(ramStr, cpuStr)
			}

			fyne.Do(func() {
				statusLabel.SetText(text)
				if hasTooltip && tooltip != "" {
					tooltipLabel.SetToolTip(tooltip)
				}
			})
		}
	}
}

func (m *Monitor) fetchResourceData() (ramStr, cpuStr, stackStr string) {
	ramStr = "-- MB"
	cpuStr = "-- %"

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.proc != nil {
		if memInfo, err := m.proc.MemoryInfo(); err == nil {
			ramMB := memInfo.RSS / 1024 / 1024
			ramStr = fmt.Sprintf("%d MB", ramMB)
		}

		if cpuPercent, err := m.proc.CPUPercent(); err == nil {
			normalized := cpuPercent / float64(m.cpuCount)
			normalized = math.Round(normalized*10) / 10
			if normalized < 0 {
				normalized = 0
			} else if normalized > 100 {
				normalized = 100
			}
			cpuStr = fmt.Sprintf("%.1f%%", normalized)
		}
	}

	if EnableStackTotal {
		pm, ok := m.procMgr.(interface {
			GetAllPIDs() []int
		})
		if ok {
			currentPIDs := pm.GetAllPIDs()

			// 快取 PID 列表：僅在列表變化時重新建立 process 物件
			pidsChanged := len(currentPIDs) != len(m.cachedPIDs)
			if !pidsChanged {
				for i, pid := range currentPIDs {
					if i >= len(m.cachedPIDs) || pid != m.cachedPIDs[i] {
						pidsChanged = true
						break
					}
				}
			}

			if pidsChanged || time.Since(m.lastPIDUpdate) > 5*time.Second {
				m.cachedPIDs = make([]int, len(currentPIDs))
				copy(m.cachedPIDs, currentPIDs)
				m.cachedProcs = make([]*process.Process, 0, len(currentPIDs))
				for _, pid := range currentPIDs {
					if pid <= 0 {
						continue
					}
					if p, err := process.NewProcess(int32(pid)); err == nil {
						m.cachedProcs = append(m.cachedProcs, p)
					}
				}
				m.lastPIDUpdate = time.Now()
			}

			var totalRAM uint64
			for _, p := range m.cachedProcs {
				if memInfo, err := p.MemoryInfo(); err == nil {
					totalRAM += memInfo.RSS
				}
			}
			if totalRAM > 0 {
				stackStr = fmt.Sprintf("%d MB", totalRAM/1024/1024)
			}
		}
	}

	return
}

// FetchStackBreakdown 計算各服務的 RAM 明細，格式化為 tooltip 文字
// 內建 1 秒節流：若距上次計算 < 1s，直接回傳快取結果
func (m *Monitor) FetchStackBreakdown(ramStr, cpuStr string) string {
	m.breakdownMu.Lock()
	defer m.breakdownMu.Unlock()

	if time.Since(m.lastBreakTime) < time.Second && m.lastBreakdown != "" {
		return m.lastBreakdown
	}

	pm, ok := m.procMgr.(interface {
		GetServiceBreakdown() []procmgr.ServiceInfo
	})
	if !ok {
		return ""
	}

	services := pm.GetServiceBreakdown()
	if len(services) == 0 {
		var sb strings.Builder
		sb.WriteString(i18n.T("WinCMP 資源監控") + "\n\n")
		sb.WriteString(i18n.Tfmt("主程式 RAM:   %s", ramStr) + "\n")
		sb.WriteString(i18n.Tfmt("主程式 CPU:   %s", cpuStr) + "\n\n")
		sb.WriteString(i18n.T("目前沒有啟動中的子服務"))
		m.lastBreakdown = sb.String()
		m.lastBreakTime = time.Now()
		return m.lastBreakdown
	}

	// 按 key 排序以確保順序穩定（caddy → mariadb → node → php）
	sort.Slice(services, func(i, j int) bool {
		return services[i].Key < services[j].Key
	})

	var lines []string
	var totalRAM uint64

	for _, svc := range services {
		var svcRAM uint64
		for _, pid := range svc.PIDs {
			if pid <= 0 {
				continue
			}
			if p, err := process.NewProcess(int32(pid)); err == nil {
				if memInfo, err := p.MemoryInfo(); err == nil {
					svcRAM += memInfo.RSS
				}
			}
		}
		totalRAM += svcRAM

		ramMB := svcRAM / 1024 / 1024
		pidCount := len(svc.PIDs)

		if pidCount > 1 {
			lines = append(lines, fmt.Sprintf("%-18s %4d MB  (%d PIDs)", svc.Name, ramMB, pidCount))
		} else {
			lines = append(lines, fmt.Sprintf("%-18s %4d MB", svc.Name, ramMB))
		}
	}

	totalMB := totalRAM / 1024 / 1024

	var sb strings.Builder
	sb.WriteString(i18n.T("WinCMP 資源監控") + "\n\n")
	sb.WriteString(i18n.Tfmt("主程式 RAM:   %s", ramStr) + "\n")
	sb.WriteString(i18n.Tfmt("主程式 CPU:   %s", cpuStr) + "\n\n")

	sb.WriteString(i18n.T("── Stack Total 明細 ──") + "\n")
	for _, line := range lines {
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	sb.WriteString("──────────────────────\n")
	sb.WriteString(fmt.Sprintf("Stack Total:       %4d MB", totalMB))

	m.lastBreakdown = sb.String()
	m.lastBreakTime = time.Now()
	return m.lastBreakdown
}

func (m *Monitor) Stop() {
	m.ticker.Stop()
	if m.cancel != nil {
		m.cancel()
	}
}
