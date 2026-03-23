package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ServiceInfo 描述一個被掃描到的服務
type ServiceInfo struct {
	Name    string // 服務名稱 (caddy, mariadb, php)
	Version string // 版本號 (如 "8.4.3", "8.2.30")
	ExePath string // 完整執行檔路徑
}

// PHPVersionInfo 描述一個 PHP 版本及其 Port 配置
type PHPVersionInfo struct {
	Version   string // 版本號 (如 "8.2.30")
	ExePath   string // php-cgi.exe 路徑
	MajorMin  string // 主版本.次版本 (如 "8.2")
	PortBase  int    // Port 基數 (如 38200)
	PortCount int    // 行程數量 (預設 3)
}

// ScanResult 掃描結果
type ScanResult struct {
	CaddyList    []ServiceInfo
	MariaDBList  []ServiceInfo
	PHPList      []PHPVersionInfo
	SkippedPHP   []string // 記錄被略過的舊 Patch 版本 (如 "8.2.28")
	HeidiSQLPath string   // HeidiSQL 執行檔路徑 (若找到)
}

// ScanBinDir 掃描 bin/ 目錄，偵測已安裝的服務與版本
func ScanBinDir(baseDir string) (*ScanResult, error) {
	result := &ScanResult{}

	binDir := filepath.Join(baseDir, "bin")

	// 1. 掃描 Caddy 版本
	caddyBaseDir := filepath.Join(binDir, "caddy")
	if entries, err := os.ReadDir(caddyBaseDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				// 檢查是否直接放 caddy.exe 在 bin/caddy/ (舊版相容)
				if entry.Name() == "caddy.exe" {
					caddyExe := filepath.Join(caddyBaseDir, "caddy.exe")
					result.CaddyList = append(result.CaddyList, ServiceInfo{
						Name:    "caddy",
						Version: "latest",
						ExePath: caddyExe,
					})
				}
				continue
			}

			// 檢查 caddy-x.y.z 格式的資料夾
			if strings.HasPrefix(entry.Name(), "caddy-") {
				caddyExe := filepath.Join(caddyBaseDir, entry.Name(), "caddy.exe")
				if _, err := os.Stat(caddyExe); err == nil {
					version := strings.TrimPrefix(entry.Name(), "caddy-")
					result.CaddyList = append(result.CaddyList, ServiceInfo{
						Name:    "caddy",
						Version: version,
						ExePath: caddyExe,
					})
				}
			}
		}
	}

	// 2. 掃描 MariaDB 版本
	mariadbDir := filepath.Join(binDir, "mariadb")
	if entries, err := os.ReadDir(mariadbDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "mariadb-") {
				continue
			}
			// 驗證 mariadbd.exe 存在
			mariadbdExe := filepath.Join(mariadbDir, entry.Name(), "bin", "mariadbd.exe")
			if _, err := os.Stat(mariadbdExe); err == nil {
				version := strings.TrimPrefix(entry.Name(), "mariadb-")
				result.MariaDBList = append(result.MariaDBList, ServiceInfo{
					Name:    "mariadb",
					Version: version,
					ExePath: mariadbdExe,
				})
			}
		}
	}

	// 3. 掃描 PHP 版本
	phpDir := filepath.Join(binDir, "php")
	phpMap := make(map[string]PHPVersionInfo) // Key: Major.Minor

	if entries, err := os.ReadDir(phpDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "php-") {
				continue
			}
			// 驗證 php-cgi.exe 存在
			phpCgiExe := filepath.Join(phpDir, entry.Name(), "php-cgi.exe")
			if _, err := os.Stat(phpCgiExe); err == nil {
				version := strings.TrimPrefix(entry.Name(), "php-")
				majorMin := extractMajorMinor(version)

				// 檢查是否已有同次版本的 PHP，若有則保留較新版本
				if existing, ok := phpMap[majorMin]; ok {
					if version > existing.Version {
						// 記錄舊版本被略過
						result.SkippedPHP = append(result.SkippedPHP, existing.Version)
						// 更新為較新版本
						portBase := calcPHPPortBase(majorMin)
						phpMap[majorMin] = PHPVersionInfo{
							Version:   version,
							ExePath:   phpCgiExe,
							MajorMin:  majorMin,
							PortBase:  portBase,
							PortCount: 3,
						}
					} else if version < existing.Version {
						// 目前掃描到的比 Map 中的舊
						result.SkippedPHP = append(result.SkippedPHP, version)
					}
				} else {
					portBase := calcPHPPortBase(majorMin)
					phpMap[majorMin] = PHPVersionInfo{
						Version:   version,
						ExePath:   phpCgiExe,
						MajorMin:  majorMin,
						PortBase:  portBase,
						PortCount: 3,
					}
				}
			}
		}
	}

	// 將 Map 轉回 Slice
	for _, info := range phpMap {
		result.PHPList = append(result.PHPList, info)
	}

	// 排序：PHP 版本由高到低
	sort.Slice(result.PHPList, func(i, j int) bool {
		return result.PHPList[i].Version > result.PHPList[j].Version
	})

	// 4. 掃描 HeidiSQL
	heidisqlDir := filepath.Join(binDir, "heidisql")
	if entries, err := os.ReadDir(heidisqlDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			exePath := filepath.Join(heidisqlDir, entry.Name(), "heidisql.exe")
			if _, err := os.Stat(exePath); err == nil {
				result.HeidiSQLPath = exePath
				break // 只取第一個找到的
			}
		}
	}

	return result, nil
}

// extractMajorMinor 從完整版本號中擷取主版本.次版本 (如 "8.2.30" → "8.2")
func extractMajorMinor(version string) string {
	parts := strings.SplitN(version, ".", 3)
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return version
}

// calcPHPPortBase 根據 PHP 主版本.次版本計算 Port 基數
// 規則：3<主版本><次版本>00，例如 PHP 8.2 → 38200，PHP 7.3 → 37300
func calcPHPPortBase(majorMinor string) int {
	parts := strings.SplitN(majorMinor, ".", 2)
	if len(parts) != 2 {
		return 38000 // fallback
	}
	major := 0
	minor := 0
	fmt.Sscanf(parts[0], "%d", &major)
	fmt.Sscanf(parts[1], "%d", &minor)
	// 規格：3<主版本><次版本>00，例如 PHP 8.2 → 38200，PHP 7.3 → 37300
	return 30000 + major*1000 + minor*100
}

// GetPHPPorts 取得某個 PHP 版本的所有 Port 列表
func (p *PHPVersionInfo) GetPHPPorts() []int {
	ports := make([]int, p.PortCount)
	for i := range ports {
		ports[i] = p.PortBase + i // 從 0 開始: 38200, 38201, 38202...
	}
	return ports
}

// GetPortRangeStr 取得 Port 範圍的字串表示 (如 "9821-9823")
func (p *PHPVersionInfo) GetPortRangeStr() string {
	ports := p.GetPHPPorts()
	if len(ports) == 0 {
		return ""
	}
	if len(ports) == 1 {
		return fmt.Sprintf("%d", ports[0])
	}
	return fmt.Sprintf("%d-%d", ports[0], ports[len(ports)-1])
}
