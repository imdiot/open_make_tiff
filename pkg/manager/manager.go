package manager

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/options"
	wails_runtime "github.com/wailsapp/wails/v2/pkg/runtime"

	"open-make-tiff/pkg/dngconverter"
	"open-make-tiff/pkg/exiftool"
	"open-make-tiff/pkg/icc"
	"open-make-tiff/pkg/runner"
	"open-make-tiff/pkg/util"
)

type WorkerNumOption struct {
	Value int    `json:"value"`
	Label string `json:"label"`
}

type ProfileOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type Setting struct {
	WorkerNums              []*WorkerNumOption `json:"worker_nums"`
	Profiles                []*ProfileOption   `json:"profiles"`
	EnableAdobeDNGConverter bool               `json:"enable_adobe_dng_converter"`
}

type Config struct {
	DisableAdobeDNGConverter bool   `json:"disable_adobe_dng_converter,omitempty"`
	EnableWindowTop          bool   `json:"enable_window_top,omitempty"`
	EnableSubfolder          bool   `json:"enable_subfolder,omitempty"`
	EnableCompression        bool   `json:"enable_compression,omitempty"`
	ICCProfile               string `json:"icc_profile,omitempty"`
	Workers                  int    `json:"workers,omitempty"`
	KeepLogFiles             bool   `json:"-"`
	KeepIntermediateFiles    bool   `json:"-"`
}

func MaxWorkers() int {
	return max(runtime.NumCPU()/2, 1)
}

type EventEmitter func(event string, data ...any)

type ManagerOption func(*Manager)

func WithEventEmitter(emit EventEmitter) ManagerOption {
	return func(m *Manager) { m.emit = emit }
}

func WithContext(ctx context.Context) ManagerOption {
	return func(m *Manager) {
		ctx, cancel := context.WithCancel(ctx)
		m.ctx = ctx
		m.cancel = cancel
		if m.et == nil {
			if execPath, err := util.GetExiftoolExecutable(); err == nil {
				m.et, err = exiftool.New(
					exiftool.WithExecutable(execPath),
					exiftool.WithLazyInit(),
					exiftool.WithContext(ctx),
				)
				if err != nil {
					slog.Warn("exiftool init failed", "error", err)
				}
			}
		}
	}
}

func newConfig() *Config {
	return &Config{
		ICCProfile: "",
		Workers:    MaxWorkers(),
	}
}

type Manager struct {
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.RWMutex
	running atomic.Bool
	config  *Config
	setting *Setting
	et      *exiftool.Exiftool
	wg      sync.WaitGroup
	emit    EventEmitter

	tmpDir                 *util.TempDir
	dngConverterExecutable string
}

func New(opts ...ManagerOption) *Manager {
	setting := &Setting{
		WorkerNums:              make([]*WorkerNumOption, 0),
		Profiles:                make([]*ProfileOption, 0),
		EnableAdobeDNGConverter: func() bool { _, err := dngconverter.New(); return err == nil }(),
	}
	for i := 1; i <= MaxWorkers(); i++ {
		setting.WorkerNums = append(setting.WorkerNums, &WorkerNumOption{Value: i, Label: fmt.Sprintf("%d", i)})
	}
	setting.Profiles = append(setting.Profiles, &ProfileOption{Value: "", Label: "none"})
	for k, v := range icc.Profiles {
		setting.Profiles = append(setting.Profiles, &ProfileOption{Value: v.Name(), Label: k})
	}
	slices.SortStableFunc(setting.Profiles, func(a, b *ProfileOption) int { return cmp.Compare(a.Value, b.Value) })

	m := &Manager{
		config:  newConfig(),
		setting: setting,
	}

	for _, opt := range opts {
		opt(m)
	}

	if td, err := util.NewTempDir("omt-"); err != nil {
		slog.Warn("temp dir failed", "error", err)
	} else {
		m.tmpDir = td
	}

	m.initDNGShadowBundle()

	return m
}

func (m *Manager) Api() *Api {
	return &Api{m: m}
}

func (m *Manager) OnStartup(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	m.ctx = ctx
	m.cancel = cancel

	m.loadConfig()
	m.validateConfig()
	wails_runtime.WindowSetAlwaysOnTop(m.ctx, m.config.EnableWindowTop)

	if m.emit == nil {
		m.emit = func(event string, data ...any) {
			wails_runtime.EventsEmit(m.ctx, event, data...)
		}
	}

	if execPath, err := util.GetExiftoolExecutable(); err == nil {
		m.et, err = exiftool.New(exiftool.WithExecutable(execPath), exiftool.WithLazyInit(), exiftool.WithContext(ctx))
		if err != nil {
			slog.Warn("exiftool init failed", "error", err)
			wails_runtime.MessageDialog(m.ctx, wails_runtime.MessageDialogOptions{
				Type:    wails_runtime.WarningDialog,
				Title:   "ExifTool",
				Message: fmt.Sprintf("ExifTool init failed: %v", err),
			})
		}
	}
}

func (m *Manager) OnSecondInstanceLaunch(_ options.SecondInstanceData) {
	wails_runtime.WindowUnminimise(m.ctx)
	wails_runtime.Show(m.ctx)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.loadConfig()
	m.validateConfig()
	wails_runtime.WindowSetAlwaysOnTop(m.ctx, m.config.EnableWindowTop)
}

func (m *Manager) OnShutdown(_ context.Context) {
	if m.cancel != nil {
		m.cancel()
	}

	if m.et != nil {
		slog.Debug("OnShutdown: closing exiftool", "at", time.Now().Format("15:04:05.000"))
		m.et.Close()
		slog.Debug("OnShutdown: exiftool closed", "at", time.Now().Format("15:04:05.000"))
	}

	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		slog.Debug("OnShutdown: wg done", "at", time.Now().Format("15:04:05.000"))
	case <-time.After(time.Second):
		slog.Debug("OnShutdown: wg timeout", "at", time.Now().Format("15:04:05.000"))
	}

	if m.tmpDir != nil {
		if err := m.tmpDir.Cleanup(); err != nil {
			slog.Debug("OnShutdown: temp dir cleanup", "error", err)
		}
	}
}

func (m *Manager) configPath() string {
	path, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(path, "open-make-tiff.json")
}

func (m *Manager) loadConfig() {
	path := m.configPath()
	if path == "" {
		return
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		m.saveConfig()
		return
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return
	}

	cfg := newConfig()
	if err = json.Unmarshal(b, cfg); err != nil {
		return
	}

	m.config = cfg
}

func (m *Manager) saveConfig() {
	path := m.configPath()
	if path == "" {
		return
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}

	cfg := m.config
	b, err := json.Marshal(cfg)
	if err != nil {
		return
	}

	if err = os.WriteFile(path, b, 0755); err != nil {
		return
	}
}

func (m *Manager) validateConfig() {
	if m.config.ICCProfile != "" {
		_, ok := icc.Profiles[m.config.ICCProfile]
		if !ok {
			m.config.ICCProfile = ""
		}
	}
	if m.config.Workers < 1 || m.config.Workers > MaxWorkers() {
		m.config.Workers = MaxWorkers()
	}
}

func (m *Manager) GetSetting() *Setting {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.setting
}

func (m *Manager) GetConfig() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.config
}

func (m *Manager) SetConfig(cfg *Config) *Config {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config = cfg

	m.validateConfig()
	m.saveConfig()
	return m.config
}

func (m *Manager) Convert(paths []string) {
	if !m.running.CompareAndSwap(false, true) {
		return
	}

	m.wg.Go(func() {
		m.emit("omt:convert:started")
		defer func() {
			m.running.Store(false)
			if m.ctx.Err() == nil {
				m.emit("omt:convert:finished")
			}
		}()

		semaphoreWorkerCh := make(chan struct{}, m.config.Workers)
		var wg sync.WaitGroup

	loop:
		for _, path := range paths {
			select {
			case <-m.ctx.Done():
				break loop
			case semaphoreWorkerCh <- struct{}{}:
				wg.Go(func() {
					defer func() {
						<-semaphoreWorkerCh
						if r := recover(); r != nil {
							slog.Warn("panic", "error", r)
							m.emit("omt:convert:file:error", path)
						}
					}()

					f, err := os.Stat(path)
					if err != nil {
						return
					}
					if f.IsDir() || !f.Mode().IsRegular() {
						return
					}
					m.emit("omt:convert:file:started", path)

					m.mu.RLock()
					cfg := m.config
					m.mu.RUnlock()

					runnerOpts := []runner.Option{
						runner.WithExiftool(m.et),
					}
					if m.dngConverterExecutable != "" {
						runnerOpts = append(runnerOpts, runner.WithDNGConverterExecutable(m.dngConverterExecutable))
					}

					if err = runner.New(runner.Config{
						EnableAdobeDNGConverter: !cfg.DisableAdobeDNGConverter,
						EnableSubfolder:         cfg.EnableSubfolder,
						EnableCompression:       cfg.EnableCompression,
						Profile:                 cfg.ICCProfile,
						KeepLogFiles:            cfg.KeepLogFiles,
						KeepIntermediateFiles:   cfg.KeepIntermediateFiles,
					}, runnerOpts...).Run(m.ctx, path); err != nil {
						if errors.Is(err, runner.ErrDstFileExists) {
							m.emit("omt:convert:file:skipped", path)
						} else {
							slog.Warn("convert", "error", err)
							m.emit("omt:convert:file:error", path)
						}
					} else {
						m.emit("omt:convert:file:success", path)
					}
				})
			}
		}
		wg.Wait()
	})
}
