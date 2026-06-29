// Command open-make-tiff converts RAW camera images into linear TIFF files
// without hidden color adjustments. With no arguments it launches the GUI;
// with file arguments it runs a CLI batch conversion.
package main

import (
	"embed"
	"log/slog"
	"os"

	"open-make-tiff/internal/build"
	"open-make-tiff/internal/cli"
	"open-make-tiff/internal/gui"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed wails.json
var wailsConfigContext []byte

func main() {
	info, err := build.Parse(wailsConfigContext)
	if err != nil {
		slog.Error("config parse failed", "err", err)
		os.Exit(1)
	}

	if len(os.Args) > 1 {
		os.Exit(cli.Run(info))
	}

	gui.Run(assets, info)
}
