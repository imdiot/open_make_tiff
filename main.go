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
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               "9424f8fb-426f-4df0-9476-f025f2a10da4",
			OnSecondInstanceLaunch: m.OnSecondInstanceLaunch,
		},
		Bind: []interface{}{
			m.Api(),
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
