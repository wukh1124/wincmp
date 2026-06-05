package main

import (
	"embed"
	"os"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"wincmp/internal/singleinstance"
)

//go:embed all:frontend/dist
var assets embed.FS

// AppVersion 定義應用程式版本，可在 build 時透過 -ldflags "-X main.AppVersion=vX.Y.Z" 動態注入
var AppVersion = "v2.0.0"

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
		// 如果是重啟，最多等待 2 秒讓舊進程完全釋放 Mutex
		for i := 0; i < 20; i++ {
			isFirst, err = singleinstance.TryAcquire()
			if err == nil && isFirst {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
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

	// 建立應用程式實例
	app := NewApp()

	// 啟動 Wails 視窗應用程式
	err = wails.Run(&options.App{
		Title:  "WinCMP Control Panel",
		Width:  1280,
		Height: 768,
		MinWidth:  1024,
		MinHeight: 700,
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
