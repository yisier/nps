package main

import (
	"context"
	"embed"
	"io/fs"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"

	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var embeddedFiles embed.FS

//go:embed build/appicon.png
var trayIcon []byte

//go:embed build/appicon.ico
var trayIconICO []byte

func main() {
	app := NewApp()

	// Asset server expects the FS root to contain index.html — create a sub FS
	assets, err := fs.Sub(embeddedFiles, "frontend/dist")
	if err != nil {
		panic(err)
	}

	runErr := wails.Run(&options.App{
		Title:     "NPS 客户端",
		Width:     1000,
		Height:    600,
		MinWidth:  1000,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		OnBeforeClose: func(ctx context.Context) bool {
			if isQuitting() {
				return false
			}
			wailsRuntime.Hide(ctx)
			return true
		},
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
	})

	if runErr != nil {
		panic(runErr)
	}
}
