package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"open-make-tiff/pkg/manager"
)

//go:embed all:frontend/dist
//go:embed wails.json
var assets embed.FS

type WailsConfig struct {
	Info struct {
		ProductName    string `json:"productName"`
		ProductVersion string `json:"productVersion"`
	} `json:"info"`
}

func main() {
	b, err := assets.ReadFile("wails.json")
	if err != nil {
		slog.Error("Error:", err.Error())
		return
	}

	var config WailsConfig
	if err = json.Unmarshal(b, &config); err != nil {
		slog.Error("Error:", err.Error())
		return
	}

	mgr := manager.New()

	if err = wails.Run(&options.App{
		Title:         fmt.Sprintf("%s - %s", config.Info.ProductName, config.Info.ProductVersion),
		Width:         512,
		Height:        384,
		DisableResize: true,
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop: true,
		},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: mgr.OnStartup,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               "9424f8fb-426f-4df0-9476-f025f2a10da4",
			OnSecondInstanceLaunch: mgr.OnSecondInstanceLaunch,
		},
		Bind: []interface{}{
			mgr.Api(),
		},
	}); err != nil {
		slog.Error("Error:", err.Error())
	}
}
