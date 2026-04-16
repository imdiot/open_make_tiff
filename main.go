package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"open-make-tiff/pkg/exiftool"
	"open-make-tiff/pkg/icc"
	"open-make-tiff/pkg/manager"
	"open-make-tiff/pkg/runner"
	"open-make-tiff/pkg/util"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed wails.json
var wailsConfigContext []byte

type WailsConfig struct {
	Info struct {
		ProductName    string `json:"productName"`
		ProductVersion string `json:"productVersion"`
	} `json:"info"`
}

func main() {
	if len(os.Args) > 1 {
		os.Exit(runCLI())
	}

	var config WailsConfig
	if err := json.Unmarshal(wailsConfigContext, &config); err != nil {
		slog.Error("config parse failed", "err", err)
		return
	}

	mgr := manager.New()

	if err := wails.Run(&options.App{
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
		OnStartup:  mgr.OnStartup,
		OnShutdown: mgr.OnShutdown,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               "9424f8fb-426f-4df0-9476-f025f2a10da4",
			OnSecondInstanceLaunch: mgr.OnSecondInstanceLaunch,
		},
		Bind: []interface{}{
			mgr.Api(),
		},
	}); err != nil {
		slog.Error("wails run failed", "err", err)
	}
}

func runCLI() int {
	fs := flag.NewFlagSet("open-make-tiff", flag.ContinueOnError)

	noDNG := fs.Bool("no-dng", false, "disable Adobe DNG Converter")
	subfolder := fs.Bool("subfolder", false, "output to a \"make_tiff\" subfolder")
	compress := fs.Bool("compress", false, "enable LZW compression")
	profile := fs.String("profile", "", "ICC profile: "+profileList())

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: %s [flags] <input-file>\n\n", fs.Name())
		fmt.Fprintf(fs.Output(), "Converts a RAW image to linear TIFF.\n\nFlags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(fs.Output(), "\nWithout arguments, launches the GUI.\n")
	}

	if err := fs.Parse(os.Args[1:]); err != nil {
		return 2
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "Error: exactly one input file required")
		fs.Usage()
		return 2
	}

	inputPath := fs.Arg(0)

	if *profile != "" {
		if _, ok := icc.Profiles[*profile]; !ok {
			fmt.Fprintf(os.Stderr, "Error: unknown profile %q (available: %s)\n", *profile, profileList())
			return 2
		}
	}

	if _, err := os.Stat(inputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var et *exiftool.Exiftool
	if execPath, err := util.GetExiftoolExecutable(); err == nil {
		et, err = exiftool.New(
			exiftool.WithExecutable(execPath),
			exiftool.WithLazyInit(),
			exiftool.WithContext(ctx),
		)
		if err != nil {
			slog.Warn("exiftool init failed", "error", err)
			et = nil
		}
	}
	if et != nil {
		defer et.Close()
	}

	r := runner.New(runner.Config{
		EnableAdobeDNGConverter: !*noDNG,
		EnableSubfolder:         *subfolder,
		EnableCompression:       *compress,
		Profile:                 *profile,
	}, runner.WithRemoveIntermediate(), runner.WithExiftool(et))

	if err := r.Run(ctx, inputPath); err != nil {
		if errors.Is(err, runner.ErrDstFileExists) {
			fmt.Fprintf(os.Stderr, "Skipped: %v\n", err)
			return 2
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	return 0
}

func profileList() string {
	names := make([]string, 0, len(icc.Profiles))
	for k := range icc.Profiles {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
