package main

import (
	"encoding/json"
	"fmt"
	"io"
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
	url := config.DefaultDependencyURL
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

// CheckMissingCoreDependencies 檢查核心依賴 (Caddy) 是否缺失
func (a *App) CheckMissingCoreDependencies() (map[string]bool, error) {
	res := make(map[string]bool)

	// 自動重新掃描以獲取最新狀態
	if _, err := a.ScanServices(); err != nil {
		a.handleErrorLog("system", "自動檢查依賴時掃描失敗", err)
	}

	res["caddy"] = len(a.scanRes.CaddyList) == 0

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
	} else if key == "redis" {
		name = "Redis v" + item.Version
		destZip = filepath.Join(binDir, "redis_"+item.Version+".zip")
		destDir = filepath.Join(binDir, "redis", "redis-"+item.Version)
	} else if strings.HasPrefix(key, "php_redis") {
		name = "PHP Redis Extension (" + key + ") v" + item.Version
		destZip = filepath.Join(binDir, key+"_"+item.Version+".zip")
		destDir = filepath.Join(binDir, "temp_"+key)
	} else if strings.HasPrefix(key, "php") {
		name = "PHP v" + item.Version + " NTS"
		destZip = filepath.Join(binDir, "php_"+item.Version+".zip")
		destDir = filepath.Join(binDir, "php", "php-"+item.Version)
	} else {
		name = key + " v" + item.Version
		destZip = filepath.Join(binDir, key+"_"+item.Version+".zip")
		destDir = filepath.Join(binDir, key, key+"-"+item.Version)
	}

	// 取得當前依賴設定來源 URL
	depURL := config.DefaultDependencyURL
	if a.appCfg != nil && a.appCfg.Global.DependencyURL != "" {
		depURL = a.appCfg.Global.DependencyURL
	}

	// 1.5 安全性預檢：驗證下載網址合法性、HTTPS 協議、官方白名單與 SHA-256 必填規則
	if err := downloader.ValidateSecurity(item.URL, item.SHA256); err != nil {
		diagErr := fmt.Errorf(
			"%s",
			i18n.Tfmt(
				"%s\n\n建議排查指引：\n1. 依賴目錄來源：%s\n2. 可能原因：官方目錄尚未發布該版本，或本地設定檔缺少安全雜湊值。\n3. 建議操作：請嘗試在右上角點擊「檢查更新」同步最新設定；若為開發測試環境，請確認 conf/dependencies.json 是否已填入正確的 sha256。",
				i18n.T(err.Error()), depURL,
			),
		)
		a.handleErrorLog("system", i18n.Tfmt("安全性檢查失敗：%s", name), diagErr)
		a.emitProgress(key, "error", 0, 0, 0, diagErr.Error())
		return
	}

	a.handleLog("system", i18n.Tfmt("開始下載核心依賴: %s...", name))
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
		_ = os.Remove(destZip)
		failErr := fmt.Errorf(
			"%s",
			i18n.Tfmt("依賴下載失敗：%v\n\n建議指引與診斷資訊：\n1. 依賴目錄來源：%s\n2. 請檢查您的網路連線或代理設定後重試。\n3. 若持續失敗，可手動下載：%s\n4. 並解壓放置於以下 bin 目錄位置：%s", err, depURL, item.URL, destDir),
		)
		a.handleErrorLog("system", i18n.Tfmt("下載 %s 失敗", name), failErr)
		a.emitProgress(key, "error", 0, 0, 0, failErr.Error())
		return
	}

	// 2.5 進行 SHA-256 完整性校驗（強制必須校驗）
	a.handleLog("system", i18n.Tfmt("正在校驗 %s 的完整性...", name))
	shaVal, shaErr := downloader.CalculateSHA256(destZip)
	if shaErr != nil {
		a.handleErrorLog("system", i18n.Tfmt("計算 %s 的 SHA-256 失敗", name), shaErr)
		a.emitProgress(key, "error", 0, 0, 0, shaErr.Error())
		_ = os.Remove(destZip)
		return
	}
	if !strings.EqualFold(shaVal, item.SHA256) {
		mismatchErr := fmt.Errorf(
			"%s",
			i18n.Tfmt("SHA-256 完整性校驗失敗！下載的檔案可能損毀、不完整或遭受中間人篡改。\n\n建議指引與診斷資訊：\n1. 依賴目錄來源：%s\n2. 請先嘗試在右上角點擊「檢查更新」取得最新校驗資訊，然後重試下載。\n3. 若問題持續，可手動下載：%s\n4. 並解壓放置於以下 bin 目錄位置：%s", depURL, item.URL, destDir),
		)
		a.handleErrorLog("system", i18n.Tfmt("%s 的完整性校驗失敗", name), mismatchErr)
		a.emitProgress(key, "error", 0, 0, 0, mismatchErr.Error())
		_ = os.Remove(destZip)
		return
	}
	a.handleLog("system", i18n.Tfmt("%s 的 SHA-256 校驗成功！", name))

	// 3. 解壓縮處理
	if strings.HasSuffix(destZip, ".zip") {
		a.handleLog("system", i18n.Tfmt("正在解壓縮 %s...", name))
		a.emitProgress(key, "extracting", 0.5, 0, 0, "")

		err = downloader.Unzip(destZip, destDir)
		if err != nil {
			_ = os.RemoveAll(destDir) // 清理殘留目錄，防半殘依賴
			_ = os.Remove(destZip)    // 清理損壞壓縮包
			unzipErr := fmt.Errorf(
				"%s",
				i18n.Tfmt("解壓縮 %s 失敗：%v\n\n建議指引：\n檔案可能損毀。請手動下載：%s\n並解壓放置於以下目錄：%s", name, err, item.URL, destDir),
			)
			a.handleErrorLog("system", i18n.Tfmt("解壓縮 %s 失敗", name), unzipErr)
			a.emitProgress(key, "error", 0, 0, 0, unzipErr.Error())
			return
		}

		// 解壓縮完成後刪除暫存 zip
		_ = os.Remove(destZip)

		// 處理 MariaDB 目錄重新命名 (mariadb-11.4.2-winx64 -> mariadb-11.4.2)
		if key == "mariadb" {
			cleanVer := strings.TrimSuffix(item.Version, "-winx64")
			oldDir := filepath.Join(binDir, "mariadb", "mariadb-"+cleanVer+"-winx64")
			newDir := filepath.Join(binDir, "mariadb", "mariadb-"+cleanVer)
			if _, err := os.Stat(oldDir); err == nil {
				if _, statErr := os.Stat(newDir); statErr == nil {
					_ = os.RemoveAll(newDir)
					time.Sleep(100 * time.Millisecond) // 等待 NTFS 標記刪除佇列釋放 handle
				}
				if renameErr := renameWithRetry(oldDir, newDir, 5, 150*time.Millisecond); renameErr != nil {
					a.handleErrorLog("system", i18n.T("MariaDB 目錄重新命名失敗"), renameErr)
				}
			}
		}

		// 處理 Node.js 目錄重新命名 (node-v20.15.0-win-x64 -> node-20.15.0)
		if key == "node" {
			oldDir := filepath.Join(binDir, "node", "node-v"+item.Version+"-win-x64")
			newDir := filepath.Join(binDir, "node", "node-"+item.Version)
			if _, err := os.Stat(oldDir); err == nil {
				if _, statErr := os.Stat(newDir); statErr == nil {
					_ = os.RemoveAll(newDir)
					time.Sleep(100 * time.Millisecond) // 等待 NTFS 標記刪除佇列釋放 handle
				}
				if renameErr := renameWithRetry(oldDir, newDir, 5, 150*time.Millisecond); renameErr != nil {
					a.handleErrorLog("system", i18n.T("Node.js 目錄重新命名失敗"), renameErr)
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

		// 處理 php_redis 擴充套件搬移
		if strings.HasPrefix(key, "php_redis") {
			dllSource := filepath.Join(destDir, "php_redis.dll")
			if _, err := os.Stat(dllSource); err == nil {
				targetMajorMin := ""
				if strings.HasSuffix(key, "_82") {
					targetMajorMin = "8.2"
				} else if strings.HasSuffix(key, "_83") {
					targetMajorMin = "8.3"
				} else if strings.HasSuffix(key, "_84") {
					targetMajorMin = "8.4"
				} else if strings.HasSuffix(key, "_73") {
					targetMajorMin = "7.3"
				} else if strings.HasSuffix(key, "_74") {
					targetMajorMin = "7.4"
				}

				phpBaseDir := filepath.Join(binDir, "php")
				copiedCount := 0
				if entries, rErr := os.ReadDir(phpBaseDir); rErr == nil {
					for _, entry := range entries {
						if entry.IsDir() && strings.HasPrefix(entry.Name(), "php-"+targetMajorMin) {
							extPath := filepath.Join(phpBaseDir, entry.Name(), "ext")
							_ = os.MkdirAll(extPath, 0755)
							targetDll := filepath.Join(extPath, "php_redis.dll")
							if copyErr := copyFile(dllSource, targetDll); copyErr == nil {
								copiedCount++
								a.handleLog("system", i18n.Tfmt("  ✓ 已安裝 php_redis.dll 至: %s", targetDll))
							} else {
								a.handleErrorLog("system", i18n.Tfmt("安裝 php_redis.dll 至 %s 失敗", targetDll), copyErr)
							}
						}
					}
				}
				if copiedCount == 0 {
					a.handleLog("system", i18n.Tfmt("ℹ️ 尚未檢測到 PHP %s 的安裝目錄，已保留暫存擴充檔案於 %s", targetMajorMin, destDir))
				} else {
					_ = os.RemoveAll(destDir)
				}
			} else {
				a.handleErrorLog("system", i18n.Tfmt("解壓縮目錄未找到 php_redis.dll: %s", destDir), nil)
			}
		}

		// 處理 PHP 核心安裝/更新時，自動安裝對應的 php_redis 擴充套件
		if strings.HasPrefix(key, "php") && !strings.HasPrefix(key, "php_redis") {
			verSuffix := strings.TrimPrefix(key, "php")
			verSuffix = strings.TrimPrefix(verSuffix, "_")
			redisKey := "php_redis_" + verSuffix
			if depConfig, loadErr := a.loadDepConfig(); loadErr == nil {
				if redisItem, ok := depConfig[redisKey]; ok {
					a.handleLog("system", i18n.Tfmt("正在為 %s 自動配置 Redis 擴充套件 (%s)...", name, redisKey))
					a.installPHPRedisExtension(redisKey, redisItem, destDir)
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

// copyFile 複製檔案輔助函式
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// renameWithRetry 提供具備退避重試的重命名機制，防範 Windows 上因防毒即時掃描或 NTFS 刪除延遲造成的瞬時 Access is denied
func renameWithRetry(oldDir, newDir string, retries int, delay time.Duration) error {
	var err error
	for i := 0; i < retries; i++ {
		err = os.Rename(oldDir, newDir)
		if err == nil {
			return nil
		}
		time.Sleep(delay)
	}
	return err
}

// installPHPRedisExtension 下載並安裝指定版本的 php_redis 擴充套件到指定的 PHP 目錄
func (a *App) installPHPRedisExtension(key string, item config.DependencyItem, phpInstallDir string) {
	binDir := filepath.Join(a.baseDir, "bin")
	tempZip := filepath.Join(binDir, "temp_"+key+".zip")
	tempExtractDir := filepath.Join(binDir, "temp_"+key)
	defer func() {
		_ = os.Remove(tempZip)
		_ = os.RemoveAll(tempExtractDir)
	}()

	// 1. 安全性預檢：驗證 URL 合法性、HTTPS 協議、白名單與 SHA-256 必填規則
	if err := downloader.ValidateSecurity(item.URL, item.SHA256); err != nil {
		a.handleErrorLog("system", i18n.Tfmt("自動配置 Redis 擴充安全性檢查失敗: %s", key), err)
		return
	}

	// 2. 下載 redis 擴充 zip
	err := downloader.DownloadFile(item.URL, tempZip, nil)
	if err != nil {
		a.handleErrorLog("system", i18n.Tfmt("自動下載 Redis 擴充失敗: %s", key), err)
		return
	}

	// 2.5 進行 SHA-256 完整性校驗
	shaVal, shaErr := downloader.CalculateSHA256(tempZip)
	if shaErr != nil {
		a.handleErrorLog("system", i18n.Tfmt("計算 Redis 擴充 (%s) SHA-256 失敗", key), shaErr)
		return
	}
	if !strings.EqualFold(shaVal, item.SHA256) {
		mismatchErr := fmt.Errorf("SHA-256 完整性校驗失敗！預期 %s, 實際 %s", item.SHA256, shaVal)
		a.handleErrorLog("system", i18n.Tfmt("Redis 擴充 (%s) 完整性校驗失敗", key), mismatchErr)
		return
	}

	// 3. 解壓縮
	err = downloader.Unzip(tempZip, tempExtractDir)
	if err != nil {
		a.handleErrorLog("system", i18n.Tfmt("自動解壓 Redis 擴充失敗: %s", key), err)
		return
	}

	// 3. 複製 php_redis.dll 到 phpInstallDir/ext/
	dllSource := filepath.Join(tempExtractDir, "php_redis.dll")
	if _, err := os.Stat(dllSource); err != nil {
		a.handleErrorLog("system", i18n.Tfmt("解壓縮目錄未找到 php_redis.dll: %s", tempExtractDir), err)
		return
	}

	extDir := filepath.Join(phpInstallDir, "ext")
	_ = os.MkdirAll(extDir, 0755)
	targetDll := filepath.Join(extDir, "php_redis.dll")
	if copyErr := copyFile(dllSource, targetDll); copyErr != nil {
		a.handleErrorLog("system", i18n.Tfmt("複製 php_redis.dll 至 %s 失敗", targetDll), copyErr)
		return
	}

	a.handleLog("system", i18n.Tfmt("已成功自動配置 php_redis.dll 至: %s", targetDll))
}

// UninstallDependency 移除指定本機已安裝的依賴
func (a *App) UninstallDependency(key string) error {
	binDir := filepath.Join(a.baseDir, "bin")

	if strings.HasPrefix(key, "php_redis_") {
		verSuffix := strings.TrimPrefix(key, "php_redis_")
		targetMajorMin := verSuffix
		if len(verSuffix) == 2 {
			targetMajorMin = string(verSuffix[0]) + "." + string(verSuffix[1])
		}
		phpBaseDir := filepath.Join(binDir, "php")
		removed := false
		if entries, err := os.ReadDir(phpBaseDir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() && strings.HasPrefix(entry.Name(), "php-"+targetMajorMin) {
					targetDll := filepath.Join(phpBaseDir, entry.Name(), "ext", "php_redis.dll")
					if _, statErr := os.Stat(targetDll); statErr == nil {
						_ = os.Remove(targetDll)
						removed = true
					}
				}
			}
		}
		if !removed {
			return fmt.Errorf("未找到可移除的 Redis 擴充檔案")
		}
		a.handleLog("system", i18n.Tfmt("已成功移除 %s 擴充套件", key))
	} else if strings.HasPrefix(key, "php") {
		verSuffix := strings.TrimPrefix(key, "php")
		verSuffix = strings.TrimPrefix(verSuffix, "_")
		targetMajorMin := verSuffix
		if len(verSuffix) == 2 {
			targetMajorMin = string(verSuffix[0]) + "." + string(verSuffix[1])
		}

		if a.procMgr != nil {
			_ = a.procMgr.StopPHPCGI(targetMajorMin)
		}

		phpBaseDir := filepath.Join(binDir, "php")
		removed := false
		if entries, err := os.ReadDir(phpBaseDir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() && strings.HasPrefix(entry.Name(), "php-"+targetMajorMin) {
					targetPath := filepath.Join(phpBaseDir, entry.Name())
					if err := os.RemoveAll(targetPath); err != nil {
						return fmt.Errorf("移除 PHP %s 目錄失敗: %w", entry.Name(), err)
					}
					removed = true
				}
			}
		}
		if !removed {
			return fmt.Errorf("未找到可移除的 PHP %s 目錄", targetMajorMin)
		}
		a.handleLog("system", i18n.Tfmt("已成功移除 PHP %s 執行環境", targetMajorMin))
	} else if key == "redis" {
		if a.procMgr != nil {
			_ = a.procMgr.StopRedis()
		}
		redisDir := filepath.Join(binDir, "redis")
		if err := os.RemoveAll(redisDir); err != nil {
			return fmt.Errorf("移除 Redis 目錄失敗: %w", err)
		}
		a.handleLog("system", i18n.T("已成功移除 Redis 快取服務"))
	} else if key == "mailpit" {
		if a.procMgr != nil {
			_ = a.procMgr.StopMailpit()
		}
		mailpitDir := filepath.Join(binDir, "mailpit")
		if err := os.RemoveAll(mailpitDir); err != nil {
			return fmt.Errorf("移除 Mailpit 目錄失敗: %w", err)
		}
		a.handleLog("system", i18n.T("已成功移除 Mailpit 服務"))
	} else if key == "node" {
		nodeDir := filepath.Join(binDir, "node")
		if err := os.RemoveAll(nodeDir); err != nil {
			return fmt.Errorf("移除 Node.js 目錄失敗: %w", err)
		}
		a.handleLog("system", i18n.T("已成功移除 Node.js 環境"))
	} else if key == "composer" {
		composerDir := filepath.Join(binDir, "composer")
		if err := os.RemoveAll(composerDir); err != nil {
			return fmt.Errorf("移除 Composer 目錄失敗: %w", err)
		}
		a.handleLog("system", i18n.T("已成功移除 Composer 套件"))
	} else if key == "heidisql" {
		heidisqlDir := filepath.Join(binDir, "heidisql")
		if err := os.RemoveAll(heidisqlDir); err != nil {
			return fmt.Errorf("移除 HeidiSQL 目錄失敗: %w", err)
		}
		a.handleLog("system", i18n.T("已成功移除 HeidiSQL 工具"))
	} else {
		targetDir := filepath.Join(binDir, key)
		if err := os.RemoveAll(targetDir); err != nil {
			return fmt.Errorf("移除 %s 目錄失敗: %w", key, err)
		}
		a.handleLog("system", i18n.Tfmt("已成功移除 %s", key))
	}

	// 重新掃描二進位服務目錄
	if scanRes, scanErr := scanner.ScanBinDir(a.baseDir); scanErr == nil {
		a.scanRes = scanRes
	}
	return nil
}

// OpenDependencyFolder 用系統檔案總管開啟指定依賴元件的安裝資料夾
func (a *App) OpenDependencyFolder(key string) error {
	binDir := filepath.Join(a.baseDir, "bin")
	var targetDir string

	// 先確保掃描資訊存在
	if a.scanRes == nil {
		if res, err := scanner.ScanBinDir(a.baseDir); err == nil {
			a.scanRes = res
		}
	}

	if strings.HasPrefix(key, "php_redis_") {
		verSuffix := strings.TrimPrefix(key, "php_redis_")
		targetMajorMin := verSuffix
		if len(verSuffix) == 2 {
			targetMajorMin = string(verSuffix[0]) + "." + string(verSuffix[1])
		}
		if a.scanRes != nil {
			for _, p := range a.scanRes.PHPList {
				if p.MajorMin == targetMajorMin && p.ExePath != "" {
					extDir := filepath.Join(filepath.Dir(p.ExePath), "ext")
					if _, err := os.Stat(extDir); err == nil {
						targetDir = extDir
						break
					}
				}
			}
		}
		if targetDir == "" {
			targetDir = filepath.Join(binDir, "php")
		}
	} else if strings.HasPrefix(key, "php") {
		verSuffix := strings.TrimPrefix(key, "php")
		verSuffix = strings.TrimPrefix(verSuffix, "_")
		targetMajorMin := verSuffix
		if len(verSuffix) == 2 {
			targetMajorMin = string(verSuffix[0]) + "." + string(verSuffix[1])
		}
		if a.scanRes != nil {
			for _, p := range a.scanRes.PHPList {
				if p.MajorMin == targetMajorMin && p.ExePath != "" {
					targetDir = filepath.Dir(p.ExePath)
					break
				}
			}
		}
		if targetDir == "" {
			phpBaseDir := filepath.Join(binDir, "php")
			if entries, err := os.ReadDir(phpBaseDir); err == nil {
				for _, entry := range entries {
					if entry.IsDir() && strings.HasPrefix(entry.Name(), "php-"+targetMajorMin) {
						targetDir = filepath.Join(phpBaseDir, entry.Name())
						break
					}
				}
			}
			if targetDir == "" {
				targetDir = phpBaseDir
			}
		}
	} else {
		if a.scanRes != nil {
			var exePath string
			switch key {
			case "caddy":
				if len(a.scanRes.CaddyList) > 0 {
					exePath = a.scanRes.CaddyList[0].ExePath
				}
			case "mariadb":
				if len(a.scanRes.MariaDBList) > 0 {
					exePath = a.scanRes.MariaDBList[0].ExePath
					if exePath != "" {
						targetDir = filepath.Dir(filepath.Dir(exePath))
					}
				}
			case "redis":
				if len(a.scanRes.RedisList) > 0 {
					exePath = a.scanRes.RedisList[0].ExePath
				}
			case "composer":
				if len(a.scanRes.ComposerList) > 0 {
					exePath = a.scanRes.ComposerList[0].ExePath
				}
			case "node":
				if len(a.scanRes.NodeList) > 0 {
					exePath = a.scanRes.NodeList[0].ExePath
				}
			case "mailpit":
				if len(a.scanRes.MailpitList) > 0 {
					exePath = a.scanRes.MailpitList[0].ExePath
				}
			case "heidisql":
				if len(a.scanRes.HeidiSQLList) > 0 {
					exePath = a.scanRes.HeidiSQLList[0].ExePath
				}
			}

			if targetDir == "" && exePath != "" {
				targetDir = filepath.Dir(exePath)
			}
		}

		if targetDir == "" {
			targetDir = filepath.Join(binDir, key)
		}
	}

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		parentDir := filepath.Dir(targetDir)
		if _, pErr := os.Stat(parentDir); pErr == nil {
			targetDir = parentDir
		} else {
			return fmt.Errorf("%s: %s", i18n.T("目錄不存在或尚未安裝"), targetDir)
		}
	}

	return a.OpenFolder(targetDir)
}



