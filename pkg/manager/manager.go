package manager

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"

	wails_runtime "github.com/wailsapp/wails/v2/pkg/runtime"

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
	ICCProfile               string `json:"icc_profile,omitempty"`
	Workers                  int    `json:"workers,omitempty"`
}

func newConfig() *Config {
	return &Config{
		ICCProfile: "",
		Workers:    runtime.NumCPU(),
	}
}

type Manager struct {
	ctx     context.Context
	config  *Config
	setting *Setting
	mu      sync.RWMutex
	running atomic.Bool
}

func New() *Manager {
	setting := &Setting{
		WorkerNums:              make([]*WorkerNumOption, 0),
		Profiles:                make([]*ProfileOption, 0),
		EnableAdobeDNGConverter: util.EnableAdobeDNGConverter(),
	}
	for i := 1; i <= runtime.NumCPU(); i++ {
		setting.WorkerNums = append(setting.WorkerNums, &WorkerNumOption{Value: i, Label: fmt.Sprintf("%d", i)})
	}
	setting.Profiles = append(setting.Profiles, &ProfileOption{Value: "", Label: "none"})
	for k, v := range icc.Profiles {
		setting.Profiles = append(setting.Profiles, &ProfileOption{Value: v.Name(), Label: k})
	}
	slices.SortStableFunc(setting.Profiles, func(a, b *ProfileOption) int { return cmp.Compare(a.Value, b.Value) })
	return &Manager{
		config:  newConfig(),
		setting: setting,
	}
}

func (m *Manager) OnStartup(ctx context.Context) {
	m.ctx = ctx
	m.loadConfig()
	m.checkConfig()
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
	err = json.Unmarshal(b, cfg)
	if err != nil {
		return
	}

	m.config = cfg
}

func (m *Manager) saveConfig() {
	fmt.Println("saveConfig")
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

	if err := os.WriteFile(path, b, 0755); err != nil {
		return
	}
}

func (m *Manager) GetSetting() *Setting {
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

	m.checkConfig()
	m.saveConfig()
	return m.config
}

func (m *Manager) checkConfig() {
	wails_runtime.WindowSetAlwaysOnTop(m.ctx, m.config.EnableWindowTop)
	if m.config.ICCProfile != "" {
		_, ok := icc.Profiles[m.config.ICCProfile]
		if !ok {
			m.config.ICCProfile = ""
		}
	}
	if m.config.Workers < 1 || m.config.Workers > runtime.NumCPU() {
		m.config.Workers = runtime.NumCPU()
	}
}

func (m *Manager) Convert(paths []string) {
	if m.running.Load() {
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	cfg := m.config

	go func() {
		m.running.Store(true)
		wails_runtime.EventsEmit(m.ctx, "omt:convert:started")
		defer func() {
			m.running.Store(false)
			wails_runtime.EventsEmit(m.ctx, "omt:convert:finished")
		}()

		semaphoreWorkerCh := make(chan struct{}, m.config.Workers)
		var wg sync.WaitGroup
		for _, path := range paths {
			select {
			case <-m.ctx.Done():
				break
			case semaphoreWorkerCh <- struct{}{}:
				wg.Go(func() {
					defer func() { <-semaphoreWorkerCh }()

					f, err := os.Stat(path)
					if err != nil {
						return
					}
					if f.IsDir() {
						return
					}
					if !f.Mode().IsRegular() {
						return
					}
					wails_runtime.EventsEmit(m.ctx, "omt:convert:file:started", path)
					r := runner.New(runner.Config{
						EnableAdobeDNGConverter: !cfg.DisableAdobeDNGConverter,
						EnableSubfolder:         cfg.EnableSubfolder,
						Profile:                 cfg.ICCProfile,
					})
					fmt.Println(path)
					err = r.Run(m.ctx, path)
					if err != nil {
						fmt.Println(err)
					}
				})
			}
		}
		wg.Wait()
	}()
}
