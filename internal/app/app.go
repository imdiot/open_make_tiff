// Package app orchestrates batch conversion: it owns the shared dependencies
// (ExifTool process, temp dir, DNG shadow bundle) and fans files out across a
// worker pool. It is framework-agnostic and driven by both the CLI and GUI.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"open-make-tiff/internal/config"
	"open-make-tiff/internal/convert"
	"open-make-tiff/pkg/dngconverter"
	"open-make-tiff/pkg/exiftool"
	"open-make-tiff/pkg/util"
)

// App owns shared conversion dependencies and runs batches. It is safe to call
// Convert/GetConfig/SetConfig concurrently from the UI thread while workers run.
type App struct {
	cfg       *config.Config
	converter *convert.Converter
	et        *exiftool.Exiftool // nil when ExifTool is unavailable
	tmpDir    *util.TempDir
	dngExec   string // macOS shadow-bundle path; "" otherwise
	log       *slog.Logger

	mu      sync.RWMutex
	running atomic.Bool
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewApp wires up shared dependencies from cfg. ExifTool and DNG Converter
// failures are logged but non-fatal: conversion proceeds without metadata or
// without the DNG Converter path as applicable.
func NewApp(ctx context.Context, cfg *config.Config, log *slog.Logger) *App {
	if log == nil {
		log = slog.Default()
	}
	a := &App{cfg: cfg, log: log}
	a.ctx, a.cancel = context.WithCancel(ctx)

	if td, err := util.NewTempDir("omt-"); err != nil {
		log.Warn("temp dir failed", "error", err)
	} else {
		a.tmpDir = td
	}

	tmpPath := ""
	if a.tmpDir != nil {
		tmpPath = a.tmpDir.Path()
	}
	a.dngExec = initDNGShadowBundle(tmpPath, !cfg.DisableAdobeDNGConverter)

	if execPath, err := util.GetExiftoolExecutable(); err == nil {
		if et, err := exiftool.New(
			exiftool.WithExecutable(execPath),
			exiftool.WithLazyInit(),
			exiftool.WithContext(a.ctx),
		); err != nil {
			log.Warn("exiftool init failed", "error", err)
		} else {
			a.et = et
		}
	}

	a.converter = convert.New(convert.WithExiftool(a.et), convert.WithDNGExecutable(a.dngExec))
	return a
}

// Exiftool returns the shared ExifTool handle, or nil when unavailable. The GUI
// uses this to warn the user that metadata will not be written.
func (a *App) Exiftool() *exiftool.Exiftool { return a.et }

// DNGConverterAvailable reports whether Adobe DNG Converter is installed. The
// GUI uses this to pick the state of the "without Adobe DNG Converter" box.
func (a *App) DNGConverterAvailable() bool {
	p := dngconverter.GetDefaultExecutablePath()
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil
}

// GetConfig returns the current configuration (safe for concurrent reads).
func (a *App) GetConfig() *config.Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cfg
}

// SetConfig replaces the configuration, validates and persists it, and returns
// the validated config. Called by the GUI when the user changes a setting.
func (a *App) SetConfig(cfg *config.Config) *config.Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	cfg.Validate()
	a.cfg = cfg
	if err := cfg.Save(); err != nil {
		a.log.Warn("config save failed", "error", err)
	}
	return a.cfg
}

// Convert filters regular files, fans them out across cfg.Workers workers, and
// reports progress via p. Re-entrant calls are ignored while a batch runs. ctx
// cancellation aborts in-flight work; p.Done is called once the batch finishes
// (only when ctx was not cancelled).
func (a *App) Convert(ctx context.Context, paths []string, p Progress) {
	if !a.running.CompareAndSwap(false, true) {
		return
	}
	a.wg.Go(func() {
		defer func() {
			a.running.Store(false)
			if ctx.Err() == nil {
				p.Done()
			}
		}()

		files := make([]string, 0, len(paths))
		for _, path := range paths {
			f, err := os.Stat(path)
			if err != nil || f.IsDir() || !f.Mode().IsRegular() {
				continue
			}
			files = append(files, path)
		}
		p.Start(len(files))

		a.mu.RLock()
		cfg := a.cfg
		a.mu.RUnlock()

		semaphore := make(chan struct{}, max(cfg.Workers, 1))
		var fileWG sync.WaitGroup

	loop:
		for _, path := range files {
			select {
			case <-ctx.Done():
				break loop
			case semaphore <- struct{}{}:
				fileWG.Go(func() {
					defer func() {
						if r := recover(); r != nil {
							a.log.Warn("panic", "error", r)
							p.File(path, ResultFailed, fmt.Errorf("%v", r))
						}
					}()
					defer func() { <-semaphore }()

					p.File(path, ResultProcessing, nil)
					err := a.converter.Convert(ctx, path, cfg)
					switch {
					case err == nil:
						p.File(path, ResultSucceeded, nil)
					case errors.Is(err, convert.ErrDstExists):
						p.File(path, ResultSkipped, nil)
					default:
						a.log.Warn("convert", "error", err)
						p.File(path, ResultFailed, err)
					}
				})
			}
		}
		fileWG.Wait()
	})
}

// Close releases shared resources: cancels the context, closes ExifTool, waits
// briefly for in-flight workers, and removes the temp dir.
func (a *App) Close() {
	if a.cancel != nil {
		a.cancel()
	}
	if a.et != nil {
		_ = a.et.Close()
	}
	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
	}
	if a.tmpDir != nil {
		_ = a.tmpDir.Cleanup()
	}
}
