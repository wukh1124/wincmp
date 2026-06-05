package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"wincmp/internal/config"
	"wincmp/internal/downloader"
	"wincmp/internal/i18n"
	"wincmp/internal/scanner"
)

// getDependencyConfigPath 取得 dependencies.json 的本機存放路徑
func (a *App) getDependencyConfigPath() string {
	return filepath.Join(a.baseDir, "conf", "dependencies.json")
}

// loadDepConfig 載入 dependencies.json
func (a *App) loadDepConfig() (config.DependencyConfig, error) {
	return config.LoadDependencies(a.getDependencyConfigPath())
}

// GetDependencyConfig 獲取本機的依賴建議版本設定
func (a *App) GetDependencyConfig() (config.DependencyConfig, error) {
	return a.loadDepConfig()
}

// FetchRemoteDependencies 從遠端下載最新的依賴建議版本配置並與本地合併
func (a *App) FetchRemoteDependencies() (config.DependencyConfig, error) {
	url := "https://raw.githubusercontent.com/wktabdev/wincmp/main/conf/dependencies.json"
	if a.appCfg != nil && a.appCfg.Global.DependencyURL != "" {
		url = a.appCfg.Global.DependencyURL
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("獲取遠端依賴配置失敗: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("遠端伺服器回應錯誤狀態碼: %d", resp.StatusCode)
	}

	var newCfg config.DependencyConfig
	if err := json.NewDecoder(resp.Body).Decode(&newCfg); err != nil {
		return nil, fmt.Errorf("解析遠端依賴配置失敗: %w", err)
	}

	// 合併本地與遠端配置，保留本地有而遠端沒有的依賴項
	depCfgPath := a.getDependencyConfigPath()
	if localCfg, loadErr := config.LoadDependencies(depCfgPath); loadErr == nil {
		for k, v := range localCfg {
			if _, ok := newCfg[k]; !ok {
				newCfg[k] = v
			}
		}
	}

	// 儲存合併後的配置到本機
	if err := config.SaveDependencies(depCfgPath, newCfg); err != nil {
		return nil, fmt.Errorf("儲存依賴配置失敗: %w", err)
	}

	return newCfg, nil
}

// CheckMissingCoreDependencies 檢查核心依賴 (Caddy, PHP) 是否缺失
func (a *App) CheckMissingCoreDependencies() (map[string]bool, error) {
	res := make(map[string]bool)

	// 自動重新掃描以獲取最新狀態
	if _, err := a.ScanServices(); err != nil {
		a.handleErrorLog("system", "自動檢查依賴時掃描失敗", err)
	}

	res["caddy"] = len(a.scanRes.CaddyList) == 0
	res["php"] = len(a.scanRes.PHPList) == 0

	return res, nil
}

// DownloadDependency 異步啟動指定依賴的下載與解壓管道
func (a *App) DownloadDependency(key string) error {
	depCfg, err := a.loadDepConfig()
	if err != nil {
		return fmt.Errorf("無法載入依賴設定: %w", err)
	}

	item, ok := depCfg[key]
	if !ok {
		return fmt.Errorf("找不到依賴項目：%s", key)
	}

	// 以背景協程執行下載管道，避免阻塞 Wails 主執行緒
	go a.runDependencyDownloadPipeline(key, item)

	return nil
}

// runDependencyDownloadPipeline 執行具體的下載、解壓與目錄重命名等管道操作
func (a *App) runDependencyDownloadPipeline(key string, item config.DependencyItem) {
	binDir := filepath.Join(a.baseDir, "bin")

	// 1. 根據依賴 key 決定下載名稱、臨時 zip 路徑與安裝目標目錄
	var name, destZip, destDir string
	if key == "caddy" {
		name = "Caddy v" + item.Version
		destZip = filepath.Join(binDir, "caddy_"+item.Version+".zip")
		destDir = filepath.Join(binDir, "caddy", "caddy-"+item.Version)
	} else if key == "mariadb" {
		name = "MariaDB v" + item.Version
		destZip = filepath.Join(binDir, "mariadb_"+item.Version+".zip")
		destDir = filepath.Join(binDir, "mariadb")
	} else if key == "composer" {
		name = "Composer v" + item.Version
		destZip = filepath.Join(binDir, "composer", "composer-"+item.Version, "composer.phar")
		destDir = filepath.Join(binDir, "composer", "composer-"+item.Version)
	} else if key == "heidisql" {
		name = "HeidiSQL v" + item.Version
		destZip = filepath.Join(binDir, "heidisql_"+item.Version+".zip")
		destDir = filepath.Join(binDir, "heidisql", "heidisql-"+item.Version)
	} else if key == "node" {
		name = "Node.js v" + item.Version
		destZip = filepath.Join(binDir, "node_"+item.Version+".zip")
		destDir = filepath.Join(binDir, "node")
	} else if key == "mailpit" {
		name = "Mailpit v" + item.Version
		destZip = filepath.Join(binDir, "mailpit_"+item.Version+".zip")
		destDir = filepath.Join(binDir, "mailpit", "mailpit-"+item.Version)
	} else if strings.HasPrefix(key, "php") {
		name = "PHP v" + item.Version + " NTS"
		destZip = filepath.Join(binDir, "php_"+item.Version+".zip")
		destDir = filepath.Join(binDir, "php", "php-"+item.Version)
	} else {
		name = key + " v" + item.Version
		destZip = filepath.Join(binDir, key+"_"+item.Version+".zip")
		destDir = filepath.Join(binDir, key, key+"-"+item.Version)
	}

	a.handleLog("system", i18n.Tfmt("🚀 開始下載核心依賴: %s...", name))
	a.emitProgress(key, "downloading", 0, 0, 0, "")

	// 2. 下載檔案並透過 callback 發送進度事件
	err := downloader.DownloadFile(item.URL, destZip, func(current, total int64) {
		var percent float64 = 0
		if total > 0 {
			percent = float64(current) / float64(total)
		}
		a.emitProgress(key, "downloading", percent, current, total, "")
	})

	if err != nil {
		a.handleErrorLog("system", i18n.Tfmt("❌ 下載 %s 失敗", name), err)
		a.emitProgress(key, "error", 0, 0, 0, err.Error())
		return
	}

	// 3. 解壓縮處理
	if strings.HasSuffix(destZip, ".zip") {
		a.handleLog("system", i18n.Tfmt("📦 正在解壓縮 %s...", name))
		a.emitProgress(key, "extracting", 0.5, 0, 0, "")

		err = downloader.Unzip(destZip, destDir)
		if err != nil {
			a.handleErrorLog("system", i18n.Tfmt("❌ 解壓縮 %s 失敗", name), err)
			a.emitProgress(key, "error", 0, 0, 0, err.Error())
			return
		}

		// 解壓縮完成後刪除暫存 zip
		os.Remove(destZip)

		// 處理 MariaDB 目錄重新命名 (mariadb-11.4.2-winx64 -> mariadb-11.4.2)
		if key == "mariadb" {
			cleanVer := strings.TrimSuffix(item.Version, "-winx64")
			oldDir := filepath.Join(binDir, "mariadb", "mariadb-"+cleanVer+"-winx64")
			newDir := filepath.Join(binDir, "mariadb", "mariadb-"+cleanVer)
			if _, err := os.Stat(oldDir); err == nil {
				os.RemoveAll(newDir)
				if renameErr := os.Rename(oldDir, newDir); renameErr != nil {
					a.handleErrorLog("system", i18n.T("MariaDB 目錄重新命名失敗"), renameErr)
				}
			}
		}

		// 處理 Node.js 目錄重新命名 (node-v20.15.0-win-x64 -> node-20.15.0)
		if key == "node" {
			oldDir := filepath.Join(binDir, "node", "node-v"+item.Version+"-win-x64")
			newDir := filepath.Join(binDir, "node", "node-"+item.Version)
			if _, err := os.Stat(oldDir); err == nil {
				if _, err := os.Stat(newDir); os.IsNotExist(err) {
					if renameErr := os.Rename(oldDir, newDir); renameErr != nil {
						a.handleErrorLog("system", i18n.T("Node.js 目錄重新命名失敗"), renameErr)
					}
				}
			}
		}

		// 處理 Mailpit 目錄檔案挪動與搬移
		if key == "mailpit" {
			oldDir := filepath.Join(binDir, "mailpit", "mailpit-"+item.Version, "mailpit-windows-amd64")
			if _, err := os.Stat(oldDir); err == nil {
				files, err := os.ReadDir(oldDir)
				if err == nil {
					for _, file := range files {
						os.Rename(
							filepath.Join(oldDir, file.Name()),
							filepath.Join(binDir, "mailpit", "mailpit-"+item.Version, file.Name()),
						)
					}
					os.Remove(oldDir)
				}
			}
		}
	} else {
		// 非 zip 檔案處理 (例如 Composer.phar 獨立檔)
		if key == "composer" {
			batPath := filepath.Join(destDir, "composer.bat")
			batContent := `@php "%~dp0composer.phar" %*`
			if err := os.WriteFile(batPath, []byte(batContent), 0755); err != nil {
				a.handleErrorLog("system", i18n.T("建立 composer.bat 失敗"), err)
			}
		}
	}

	a.handleLog("system", i18n.Tfmt("✅ %s 安裝與配置成功！", name))

	// 4. 重新掃描二進位服務目錄
	scanRes, scanErr := scanner.ScanBinDir(a.baseDir)
	if scanErr != nil {
		a.handleErrorLog("system", i18n.T("安裝完成後重新掃描 bin 失敗"), scanErr)
	} else {
		a.scanRes = scanRes
		a.handleLog("system", i18n.T("重新掃描 bin 目錄完成，服務已就緒。"))
	}

	// 回報下載成功事件
	a.emitProgress(key, "completed", 1.0, 0, 0, "")
}

// emitProgress 發送依賴下載進度事件至 Wails 前端
func (a *App) emitProgress(key string, status string, percent float64, current, total int64, errStr string) {
	if a.ctx == nil {
		return
	}
	currentMB := float64(current) / 1024 / 1024
	totalMB := float64(total) / 1024 / 1024

	runtime.EventsEmit(a.ctx, "dependency_progress", map[string]interface{}{
		"key":       key,
		"status":    status,
		"percent":   percent,
		"currentMB": currentMB,
		"totalMB":   totalMB,
		"error":     errStr,
	})
}
