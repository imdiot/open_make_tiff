package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"open-make-tiff/pkg/manager"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	m := manager.New()

	err := wails.Run(&options.App{
		Title:         "open make tiff",
		Width:         512,
		Height:        384,
		DisableResize: true,
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop: true,
		},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: m.OnStartup,
		Bind: []interface{}{
			m,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
