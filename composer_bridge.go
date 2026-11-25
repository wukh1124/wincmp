package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"wincmp/internal/composer"
	"wincmp/internal/config"
	"wincmp/internal/i18n"
	"wincmp/internal/scanner"
)

func (a *App) ensureComposerManager() *composer.Manager {
	if a.composerMgr == nil {
		a.composerMgr = composer.NewManager(a.baseDir)
	}
	return a.composerMgr
}

// RunComposerCommand 於指定 PHP 專案根目錄執行 Composer 指令，並透過事件推送串流輸出
func (a *App) RunComposerCommand(projName string, composerVer string, args []string) (string, error) {
	if a.appCfg == nil {
		return "", fmt.Errorf("%s", i18n.T("設定檔尚未載入"))
	}

	var proj *config.ProjectConfig
	for i := range a.appCfg.Projects {
		if a.appCfg.Projects[i].Name == projName {
			proj = &a.appCfg.Projects[i]
			break
		}
	}
	if proj == nil {
		return "", fmt.Errorf("%s: %s", i18n.T("找不到指定專案"), projName)
	}

	scanRes, err := a.ensureScanResult()
	if err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T("掃描服務版本失敗"), err)
	}

	// 解析專案對應的 PHP 版本與 php.exe
	var matchedPHP *scanner.PHPVersionInfo
	for i := range scanRes.PHPList {
		if scanRes.PHPList[i].MajorMin == proj.PHPVersion {
			matchedPHP = &scanRes.PHPList[i]
			break
		}
	}
	if matchedPHP == nil {
		return "", fmt.Errorf("%s", i18n.Tfmt("專案設定的 PHP %s 尚未安裝，請先至依賴庫管理安裝", proj.PHPVersion))
	}
	phpExe := filepath.Join(filepath.Dir(matchedPHP.ExePath), "php.exe")
	if _, err := os.Stat(phpExe); err != nil {
		return "", fmt.Errorf("%s: %s", i18n.T("找不到對應的 php.exe"), phpExe)
	}

	// 解析 Composer 版本與 composer.phar
	if len(scanRes.ComposerList) == 0 {
		return "", fmt.Errorf("%s", i18n.T("尚未安裝 Composer，請先至依賴庫管理下載安裝"))
	}
	var matchedComp *scanner.ServiceInfo
	if composerVer != "" {
		for i := range scanRes.ComposerList {
			if scanRes.ComposerList[i].Version == composerVer {
				matchedComp = &scanRes.ComposerList[i]
				break
			}
		}
	}
	if matchedComp == nil {
		matchedComp = &scanRes.ComposerList[0]
	}
	composerPhar := filepath.Join(filepath.Dir(matchedComp.ExePath), "composer.phar")
	if _, err := os.Stat(composerPhar); err != nil {
		return "", fmt.Errorf("%s: %s", i18n.T("找不到 composer.phar"), composerPhar)
	}

	workDir := a.appCfg.GetProjectPhysicalRoot(*proj, a.baseDir)
	mgr := a.ensureComposerManager()

	opts := composer.RunOptions{
		ProjectName:     proj.Name,
		WorkDir:         workDir,
		PHPExe:          phpExe,
		PHPVersion:      matchedPHP.Version,
		ComposerPhar:    composerPhar,
		ComposerVersion: matchedComp.Version,
		Args:            args,
	}

	return mgr.Start(
		opts,
		func(runID string, chunk string) {
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "composer_output", map[string]string{
					"runId": runID,
					"data":  chunk,
				})
			}
		},
		func(runID string, entry composer.HistoryEntry) {
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "composer_exit", map[string]interface{}{
					"runId": runID,
					"entry": entry,
				})
			}
		},
	)
}

// StopComposerCommand 停止執行中的 Composer 工作
func (a *App) StopComposerCommand(runID string) error {
	return a.ensureComposerManager().Stop(runID)
}

// GetComposerHistory 分頁讀取專案的 Composer 指令歷史紀錄
func (a *App) GetComposerHistory(projName string, offset int, limit int) (*composer.HistoryPage, error) {
	return a.ensureComposerManager().History().List(projName, offset, limit)
}

// ClearComposerHistory 清除專案的 Composer 指令歷史紀錄
func (a *App) ClearComposerHistory(projName string) error {
	return a.ensureComposerManager().History().Clear(projName)
}
