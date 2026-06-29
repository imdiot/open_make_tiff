// Package cli implements the command-line batch conversion mode.
package cli

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync/atomic"
	"syscall"

	"open-make-tiff/internal/app"
	"open-make-tiff/internal/build"
	"open-make-tiff/internal/config"
	"open-make-tiff/pkg/icc"
	"open-make-tiff/pkg/util"
)

// Run parses CLI flags and runs a batch conversion. Returns the process exit
// code (1 if any file failed, 2 on usage error).
func Run(info build.Info) int {
	util.AttachParentConsole()
	defer util.FreeParentConsole()

	fs := flag.NewFlagSet("OpenMakeTiff", flag.ContinueOnError)

	noDNG := fs.Bool("no-dng", false, "disable Adobe DNG Converter")
	subfolder := fs.Bool("subfolder", false, "output to a \"make_tiff\" subfolder")
	compress := fs.Bool("compress", false, "enable LZW compression")
	profile := fs.String("profile", "", "ICC profile: "+profileList())
	workers := fs.Int("workers", config.MaxWorkers(), "number of parallel workers")
	keepLog := fs.Bool("keep-log", false, "keep log files after conversion")
	keepIntermediate := fs.Bool("keep-intermediate", false, "keep intermediate DNG/TIFF files")
	showVersion := fs.Bool("version", false, "print version and exit")

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: %s [flags] <input-file> [input-file...]\n\n", fs.Name())
		fmt.Fprintf(fs.Output(), "Converts RAW images to linear TIFF.\n\nFlags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(fs.Output(), "\nWithout arguments, launches the GUI.\n")
	}

	if err := fs.Parse(os.Args[1:]); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Println(info.ProductVersion)
		return 0
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

	cfg := &config.Config{
		DisableAdobeDNGConverter: *noDNG,
		EnableSubfolder:          *subfolder,
		EnableCompression:        *compress,
		ICCProfile:               *profile,
		Workers:                  *workers,
		KeepLogFiles:             *keepLog,
		KeepIntermediateFiles:    *keepIntermediate,
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	a := app.NewApp(ctx, cfg, logger)
	defer a.Close()

	tp := &textProgress{failed: new(atomic.Int32), done: make(chan struct{})}
	a.Convert(ctx, fs.Args(), tp)

	select {
	case <-tp.done:
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, "\nInterrupted, cleaning up...")
	}

	if tp.failed.Load() > 0 {
		return 1
	}
	return 0
}

// textProgress implements app.Progress, writing per-file status to stderr.
type textProgress struct {
	failed *atomic.Int32
	done   chan struct{}
}

func (t *textProgress) Start(total int) {
	fmt.Fprintf(os.Stderr, "Converting %d file(s)...\n", total)
}

func (t *textProgress) File(path string, result app.Result, _ error) {
	switch result {
	case app.ResultProcessing:
		fmt.Fprintf(os.Stderr, "  Converting: %s\n", path)
	case app.ResultSucceeded:
		fmt.Fprintf(os.Stderr, "  OK: %s\n", path)
	case app.ResultSkipped:
		fmt.Fprintf(os.Stderr, "  SKIP: %s\n", path)
	case app.ResultFailed:
		t.failed.Add(1)
		fmt.Fprintf(os.Stderr, "  FAIL: %s\n", path)
	}
}

func (t *textProgress) Done() { close(t.done) }

func profileList() string {
	names := make([]string, 0, len(icc.Profiles))
	for k := range icc.Profiles {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
