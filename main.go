package main

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"wincmp/internal/singleinstance"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed VERSION
var versionFile string

// AppVersion 定義應用程式版本，可在 build 時透過 -ldflags "-X main.AppVersion=vX.Y.Z" 動態注入。
// 如果沒有透過 -ldflags 注入，則會從 VERSION 檔案中自動讀取。
var AppVersion string

func init() {
	if AppVersion == "" {
		trimmed := strings.TrimSpace(versionFile)
		if trimmed != "" {
			AppVersion = "v" + trimmed
		} else {
			AppVersion = "v2.0.0" // 若 VERSION 檔案內容為空，則使用預設版號
		}
	}
}

func main() {
	isRestart := false
	for _, arg := range os.Args {
		if arg == "--restart" {
			isRestart = true
			break
		}
	}

	// ─── 單一執行個體防護 ──────────────────────────────
	var isFirst bool
	var err error
	if isRestart {
		// 如果是重啟，最多等待 10 秒讓舊進程完整退出（包含固定檔名 WinCMP.exe 與防毒掃描延遲）
		isFirst, err = singleinstance.TryAcquireForRestart(10 * time.Second)
	} else {
		isFirst, err = singleinstance.TryAcquire()
	}

	if err != nil {
		// Mutex 建立失敗，保守地繼續執行
		println("Warning: failed to check single instance mutex:", err.Error())
	} else if !isFirst {
		// 已有另一個 WinCMP 在執行，透過管道喚回現有視窗，然後退出
		singleinstance.BringExistingToFront()
		os.Exit(0)
	}
	defer singleinstance.Release()

	// 相容從 v2.1.9 等不同檔名舊版重啟：等待同目錄下舊 WinCMP 行程完全退出，
	// 確保舊版已完成設定檔寫入並透過 Job Object 釋放所有子服務 Port。
	if isRestart {
		if execPath, exeErr := os.Executable(); exeErr == nil {
			singleinstance.WaitForSiblingWinCMPExit(filepath.Dir(execPath), 10*time.Second)
		}
	}

	// 建立應用程式實例
	app := NewApp()
	app.isRestart = isRestart

	// 啟動 Wails 視窗應用程式
	err = wails.Run(&options.App{
		Title:     "WinCMP Control Panel",
		Width:     1440,
		Height:    840,
		MinWidth:  1100,
		MinHeight: 700,
		StartHidden: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		OnBeforeClose:    app.beforeClose,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}