package main

import (
	"embed"
	"log"
	"runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}
	frameless := runtime.GOOS == "windows"
	err = wails.Run(&options.App{
		Title:                    "PuffSQL - Lightweight SQL Client",
		Width:                    1440,
		Height:                   900,
		MinWidth:                 1368,
		MinHeight:                600,
		Frameless:                frameless,
		BackgroundColour:         &options.RGBA{R: 24, G: 26, B: 31, A: 255},
		DisableResize:            false,
		AssetServer:              &assetserver.Options{Assets: assets},
		OnStartup:                app.Startup,
		Bind:                     []any{app},
		Windows:                  &windows.Options{WebviewIsTransparent: false, WindowIsTranslucent: false},
		EnableDefaultContextMenu: false,
	})
	if err != nil {
		log.Fatal(err)
	}
}
