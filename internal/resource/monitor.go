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
	"github.com/shirou/gopsutil/v3/cpu"
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

	// Web WebView2 快取
	cachedWebPIDs    []int
	cachedWebProcs   []*process.Process
	lastWebPIDUpdate time.Time
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

// GetCPUAndRAM 獲取當前開發環境整體的 CPU 佔用與 RAM 總佔用 (含核心、Web視窗、各子服務的總合)
func (m *Monitor) GetCPUAndRAM() (totalCPU float64, totalRAM uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. 計算主程式 CPUPercent 與 RAM
	var coreCPU float64
	var coreRAM uint64
	if m.proc != nil {
		if cpu, err := m.proc.CPUPercent(); err == nil {
			normalized := cpu / float64(m.cpuCount)
			normalized = math.Round(normalized*10) / 10
			if normalized < 0 {
				normalized = 0
			} else if normalized > 100 {
				normalized = 100
			}
			coreCPU = normalized
		}

		if memInfo, err := m.proc.MemoryInfo(); err == nil {
			coreRAM = memInfo.RSS / 1024 / 1024
		}
	}
	totalCPU += coreCPU
	totalRAM += coreRAM

	// 2. 獲取所有運行服務 PID 列表
	var currentSvcPIDs []int
	pm, hasPM := m.procMgr.(interface {
		GetAllPIDs() []int
		GetServiceBreakdown() []procmgr.ServiceInfo
	})

	if hasPM {
		currentSvcPIDs = pm.GetAllPIDs()
	}

	// 3. Web 視窗介面 (WebView2) CPU & RAM 佔用
	// WebView2 進程為主進程之子進程，且不屬於已被註冊的服務
	if time.Since(m.lastWebPIDUpdate) > 5*time.Second {
		var webPIDs []int
		if procs, err := process.Processes(); err == nil {
			for _, p := range procs {
				ppid, err := p.Ppid()
				if err == nil && ppid == int32(m.pid) {
					pidVal := int(p.Pid)
					isService := false
					for _, svcPID := range currentSvcPIDs {
						if pidVal == svcPID {
							isService = true
							break
						}
					}
					if !isService {
						webPIDs = append(webPIDs, pidVal)
					}
				}
			}
		}
		m.cachedWebPIDs = webPIDs
		m.cachedWebProcs = make([]*process.Process, 0, len(webPIDs))
		for _, pid := range webPIDs {
			if p, err := process.NewProcess(int32(pid)); err == nil {
				m.cachedWebProcs = append(m.cachedWebProcs, p)
			}
		}
		m.lastWebPIDUpdate = time.Now()
	}

	var webCPU float64
	var webRAM uint64
	for _, p := range m.cachedWebProcs {
		if cpuPct, err := p.CPUPercent(); err == nil {
			normalized := cpuPct / float64(m.cpuCount)
			webCPU += normalized
		}
		if memInfo, err := p.MemoryInfo(); err == nil {
			webRAM += memInfo.RSS
		}
	}
	totalCPU += webCPU
	totalRAM += webRAM / 1024 / 1024

	// 4. 各依賴服務（Caddy, MariaDB, PHP 等）各自的 CPU & RAM
	if hasPM {
		services := pm.GetServiceBreakdown()
		for _, svc := range services {
			var svcCPU float64
			var svcRAM uint64
			for _, pid := range svc.PIDs {
				if pid <= 0 {
					continue
				}
				if p, err := process.NewProcess(int32(pid)); err == nil {
					if cpuPct, err := p.CPUPercent(); err == nil {
						normalized := cpuPct / float64(m.cpuCount)
						svcCPU += normalized
					}
					if memInfo, err := p.MemoryInfo(); err == nil {
						svcRAM += memInfo.RSS
					}
				}
			}
			totalCPU += svcCPU
			totalRAM += svcRAM / 1024 / 1024
		}
	}

	// 進行最後的小數點四捨五入
	totalCPU = math.Round(totalCPU*10) / 10
	return
}

type DetailedResources struct {
	SystemCPU float64                    `json:"systemCpu"`
	Core      ProcessResource            `json:"core"`
	Web       ProcessResource            `json:"web"`
	Services  map[string]ServiceResource `json:"services"`
}

type ProcessResource struct {
	CPU float64 `json:"cpu"`
	RAM uint64  `json:"ram"` // MB
}

type ServiceResource struct {
	Name string  `json:"name"`
	CPU  float64 `json:"cpu"`
	RAM  uint64  `json:"ram"` // MB
	PIDs []int   `json:"pids"`
}

// GetDetailedResourceUsage 獲取當前系統 CPU、WinCMP 核心、WebView2 介面及各運行服務的 CPU & RAM 佔用明細
func (m *Monitor) GetDetailedResourceUsage() (DetailedResources, error) {
	var dr DetailedResources
	dr.Services = make(map[string]ServiceResource)

	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. 系統總體 CPU 佔用率
	percent, err := cpu.Percent(0, false)
	if err == nil && len(percent) > 0 {
		dr.SystemCPU = math.Round(percent[0]*10) / 10
	}

	// 2. WinCMP 核心 CPU & RAM 佔用
	if m.proc != nil {
		if cpuPct, err := m.proc.CPUPercent(); err == nil {
			normalized := cpuPct / float64(m.cpuCount)
			normalized = math.Round(normalized*10) / 10
			if normalized < 0 {
				normalized = 0
			} else if normalized > 100 {
				normalized = 100
			}
			dr.Core.CPU = normalized
		}
		if memInfo, err := m.proc.MemoryInfo(); err == nil {
			dr.Core.RAM = memInfo.RSS / 1024 / 1024
		}
	}

	// 3. 獲取所有運行服務 PID 列表
	var currentSvcPIDs []int
	pm, hasPM := m.procMgr.(interface {
		GetAllPIDs() []int
		GetServiceBreakdown() []procmgr.ServiceInfo
	})

	if hasPM {
		currentSvcPIDs = pm.GetAllPIDs()
	}

	// 4. Web 視窗介面 (WebView2) CPU & RAM 佔用
	// WebView2 進程為主進程之子進程，且不屬於已被註冊的服務
	if time.Since(m.lastWebPIDUpdate) > 5*time.Second {
		var webPIDs []int
		if procs, err := process.Processes(); err == nil {
			for _, p := range procs {
				ppid, err := p.Ppid()
				if err == nil && ppid == int32(m.pid) {
					pidVal := int(p.Pid)
					// 如果這個 PID 不是已註冊之依賴服務 PID，即為 Web 視窗進程
					isService := false
					for _, svcPID := range currentSvcPIDs {
						if pidVal == svcPID {
							isService = true
							break
						}
					}
					if !isService {
						webPIDs = append(webPIDs, pidVal)
					}
				}
			}
		}
		m.cachedWebPIDs = webPIDs
		m.cachedWebProcs = make([]*process.Process, 0, len(webPIDs))
		for _, pid := range webPIDs {
			if p, err := process.NewProcess(int32(pid)); err == nil {
				m.cachedWebProcs = append(m.cachedWebProcs, p)
			}
		}
		m.lastWebPIDUpdate = time.Now()
	}

	var webCPU float64
	var webRAM uint64
	for _, p := range m.cachedWebProcs {
		if cpuPct, err := p.CPUPercent(); err == nil {
			normalized := cpuPct / float64(m.cpuCount)
			webCPU += normalized
		}
		if memInfo, err := p.MemoryInfo(); err == nil {
			webRAM += memInfo.RSS
		}
	}
	dr.Web.CPU = math.Round(webCPU*10) / 10
	dr.Web.RAM = webRAM / 1024 / 1024

	// 5. 各依賴服務（Caddy, MariaDB, PHP 等）各自的 CPU & RAM
	if hasPM {
		services := pm.GetServiceBreakdown()
		for _, svc := range services {
			var svcCPU float64
			var svcRAM uint64
			for _, pid := range svc.PIDs {
				if pid <= 0 {
					continue
				}
				if p, err := process.NewProcess(int32(pid)); err == nil {
					if cpuPct, err := p.CPUPercent(); err == nil {
						normalized := cpuPct / float64(m.cpuCount)
						svcCPU += normalized
					}
					if memInfo, err := p.MemoryInfo(); err == nil {
						svcRAM += memInfo.RSS
					}
				}
			}
			dr.Services[svc.Key] = ServiceResource{
				Name: svc.Name,
				CPU:  math.Round(svcCPU*10) / 10,
				RAM:  svcRAM / 1024 / 1024,
				PIDs: svc.PIDs,
			}
		}
	}

	return dr, nil
}


