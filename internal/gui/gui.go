// Package gui runs the Wails desktop frontend. It is the GUI counterpart of
// internal/cli: both wrap the framework-agnostic internal/app.
package gui

import (
	"context"
	"embed"
	"fmt"
	"log/slog"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wails_runtime "github.com/wailsapp/wails/v2/pkg/runtime"

	"open-make-tiff/internal/app"
	"open-make-tiff/internal/build"
	"open-make-tiff/internal/config"
	"open-make-tiff/pkg/manager"
	"open-make-tiff/pkg/util"
)

// Run launches the Wails window. assets (the embedded frontend) is passed in
// because //go:embed cannot reach frontend/dist from this package. productName
// and version drive the window title.
func Run(assets embed.FS, info build.Info) {
	ctx := context.Background()
	omt := app.NewApp(ctx, config.Load(), nil)
	api := manager.NewApi(omt)

	if err := wails.Run(&options.App{
		Title:         fmt.Sprintf("%s - %s", info.ProductName, info.ProductVersion),
		Width:         512,
		Height:        384,
		DisableResize: true,
		StartHidden:   true,
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop: true,
		},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: func(c context.Context) {
			api.SetContext(c)
			wails_runtime.WindowSetAlwaysOnTop(c, omt.GetConfig().EnableWindowTop)
			// Mirror the original semantics: warn only when ExifTool is
			// installed but failed to initialize.
			if omt.Exiftool() == nil {
				if _, err := util.GetExiftoolExecutable(); err == nil {
					wails_runtime.MessageDialog(c, wails_runtime.MessageDialogOptions{
						Type:    wails_runtime.WarningDialog,
						Title:   "ExifTool",
						Message: "ExifTool was found but failed to initialize; metadata will not be written.",
					})
				}
			}
		},
		OnShutdown: func(context.Context) { omt.Close() },
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "9424f8fb-426f-4df0-9476-f025f2a10da4",
			OnSecondInstanceLaunch: func(options.SecondInstanceData) {
				wails_runtime.WindowUnminimise(api.Context())
				wails_runtime.Show(api.Context())
				wails_runtime.WindowSetAlwaysOnTop(api.Context(), omt.GetConfig().EnableWindowTop)
			},
		},
		Bind: []interface{}{
			api,
		},
	}); err != nil {
		slog.Error("wails run failed", "err", err)
	}
}
