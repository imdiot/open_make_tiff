// Package config defines the persistent application configuration and the
// selectable option lists (worker counts, ICC profiles) shown in the GUI/CLI.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"

	"open-make-tiff/pkg/icc"
)

// Config is the persistent application configuration. JSON tags must stay
// snake_case to remain compatible with existing user config files written by
// previous releases.
type Config struct {
	DisableAdobeDNGConverter bool   `json:"disable_adobe_dng_converter,omitzero"`
	EnableWindowTop          bool   `json:"enable_window_top,omitzero"`
	EnableSubfolder          bool   `json:"enable_subfolder,omitzero"`
	EnableCompression        bool   `json:"enable_compression,omitzero"`
	ICCProfile               string `json:"icc_profile,omitempty"`
	Workers                  int    `json:"workers,omitzero"`

	// KeepLogFiles and KeepIntermediateFiles are runtime-only (CLI flags) and
	// are never persisted.
	KeepLogFiles          bool `json:"-"`
	KeepIntermediateFiles bool `json:"-"`
}

// Default returns a Config populated with default values.
func Default() *Config {
	return &Config{
		ICCProfile: "",
		Workers:    MaxWorkers(),
	}
}

// MaxWorkers returns the default worker count: half the CPU cores, at least 1.
func MaxWorkers() int {
	return max(runtime.NumCPU()/2, 1)
}

// Validate clamps workers into the valid range and clears an unknown ICC
// profile. It mutates the receiver in place.
func (c *Config) Validate() {
	if c.ICCProfile != "" {
		if _, ok := icc.Profiles[c.ICCProfile]; !ok {
			c.ICCProfile = ""
		}
	}
	if c.Workers < 1 || c.Workers > MaxWorkers() {
		c.Workers = MaxWorkers()
	}
}

// Path returns the config file location under the user config directory, or ""
// if the directory cannot be determined.
func Path() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "open-make-tiff.json")
}

// Load reads the config from Path. A missing file is created with defaults; an
// unreadable or unparseable file yields defaults. Load never returns an error.
func Load() *Config {
	path := Path()
	if path == "" {
		return Default()
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		cfg := Default()
		_ = cfg.Save()
		return cfg
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return Default()
	}

	cfg := Default()
	if err := json.Unmarshal(b, cfg); err != nil {
		return Default()
	}
	cfg.Validate()
	return cfg
}

// Save writes the config to Path with mode 0644, creating the parent directory
// if needed.
func (c *Config) Save() error {
	path := Path()
	if path == "" {
		return errors.New("config: cannot determine user config directory")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}
