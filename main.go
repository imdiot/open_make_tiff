package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
	"syscall"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"open-make-tiff/pkg/icc"
	"open-make-tiff/pkg/manager"
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
	util.AttachParentConsole()
	defer util.FreeParentConsole()

	fs := flag.NewFlagSet("open-make-tiff", flag.ContinueOnError)

	noDNG := fs.Bool("no-dng", false, "disable Adobe DNG Converter")
	subfolder := fs.Bool("subfolder", false, "output to a \"make_tiff\" subfolder")
	compress := fs.Bool("compress", false, "enable LZW compression")
	profile := fs.String("profile", "", "ICC profile: "+profileList())
	workers := fs.Int("workers", max(runtime.NumCPU()/2, 1), "number of parallel workers")
	keepLog := fs.Bool("keep-log", false, "keep log files after conversion")
	keepIntermediate := fs.Bool("keep-intermediate", false, "keep intermediate DNG/TIFF files")

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: %s [flags] <input-file> [input-file...]\n\n", fs.Name())
		fmt.Fprintf(fs.Output(), "Converts RAW images to linear TIFF.\n\nFlags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(fs.Output(), "\nWithout arguments, launches the GUI.\n")
	}

	if err := fs.Parse(os.Args[1:]); err != nil {
		return 2
	}

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Error: at least one input file required")
		fs.Usage()
		return 2
	}

	if *profile != "" {
		if _, ok := icc.Profiles[*profile]; !ok {
			fmt.Fprintf(os.Stderr, "Error: unknown profile %q (available: %s)\n", *profile, profileList())
			return 2
		}
	}

	if *workers < 1 {
		fmt.Fprintln(os.Stderr, "Error: workers must be >= 1")
		return 2
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var failed atomic.Int32
	done := make(chan struct{})

	mgr := manager.New(
		manager.WithContext(ctx),
		manager.WithEventEmitter(func(event string, data ...any) {
			switch event {
			case "omt:convert:file:success":
				fmt.Fprintf(os.Stderr, "  OK: %s\n", data[0])
			case "omt:convert:file:skipped":
				fmt.Fprintf(os.Stderr, "  SKIP: %s\n", data[0])
			case "omt:convert:file:error":
				failed.Add(1)
				fmt.Fprintf(os.Stderr, "  FAIL: %s\n", data[0])
			case "omt:convert:finished":
				close(done)
			}
		}),
	)

	mgr.SetConfig(&manager.Config{
		DisableAdobeDNGConverter: *noDNG,
		EnableSubfolder:         *subfolder,
		EnableCompression:       *compress,
		ICCProfile:              *profile,
		Workers:                 *workers,
		KeepLogFiles:            *keepLog,
		KeepIntermediateFiles:   *keepIntermediate,
	})

	mgr.Convert(fs.Args())
	select {
	case <-done:
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, "\nInterrupted, cleaning up...")
	}

	mgr.Shutdown()

	if failed.Load() > 0 {
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
