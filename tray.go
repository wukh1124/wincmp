package main

import (
	_ "embed"

	"fyne.io/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"wincmp/internal/i18n"
)

//go:embed build/windows/icon.ico
var trayIcon []byte

// setupSystray 在背景啟動系統托盤服務
func (a *App) setupSystray() {
	go func() {
		systray.Run(a.onTrayReady, a.onTrayExit)
	}()
}

// onTrayReady 當托盤準備就緒時進行配置並監聽事件
func (a *App) onTrayReady() {
	systray.SetIcon(trayIcon)
	systray.SetTooltip("WinCMP")

	// 建立選單並保存指標，以便後續動態更新語系
	a.trayShowItem = systray.AddMenuItem(i18n.T("顯示 WinCMP"), i18n.T("顯示 WinCMP 主視窗"))
	a.trayQuitItem = systray.AddMenuItem(i18n.T("完全退出 (Quit)"), i18n.T("完全退出應用程式"))

	// 監聽托盤項目的點擊事件
	go func() {
		for {
			select {
			case <-a.trayShowItem.ClickedCh:
				if a.ctx != nil {
					runtime.WindowShow(a.ctx)
				}
			case <-a.trayQuitItem.ClickedCh:
				a.quitApp()
				return
			}
		}
	}()
}

// onTrayExit 系統托盤關閉時的清理
func (a *App) onTrayExit() {
	// 此處無需額外清理，因為進程即將結束
}

// updateTrayMenu 動態刷新系統托盤的語系
func (a *App) updateTrayMenu() {
	if a.trayShowItem != nil {
		a.trayShowItem.SetTitle(i18n.T("顯示 WinCMP"))
		a.trayShowItem.SetTooltip(i18n.T("顯示 WinCMP 主視窗"))
	}
	if a.trayQuitItem != nil {
		a.trayQuitItem.SetTitle(i18n.T("完全退出 (Quit)"))
		a.trayQuitItem.SetTooltip(i18n.T("完全退出應用程式"))
	}
}

// quitApp 安全關閉應用程式
func (a *App) quitApp() {
	a.quittingMu.Lock()
	a.quitting = true
	a.quittingMu.Unlock()

	// 觸發 Wails 的關閉流程，以確保執行 OnShutdown 並關閉所有子進程
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}
