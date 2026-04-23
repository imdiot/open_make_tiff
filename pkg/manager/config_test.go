package manager

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNewConfigDefaults(t *testing.T) {
	cfg := NewConfig()
	if cfg.ICCProfile != "" {
		t.Errorf("ICCProfile = %q, want empty", cfg.ICCProfile)
	}
	if cfg.Workers != MaxWorkers() {
		t.Errorf("Workers = %d, want %d", cfg.Workers, MaxWorkers())
	}
	if cfg.DisableAdobeDNGConverter || cfg.EnableWindowTop || cfg.EnableSubfolder || cfg.EnableCompression {
		t.Error("bool fields should default to false")
	}
}

func TestValidateConfigIllegalProfile(t *testing.T) {
	m := New()
	m.config = &Config{ICCProfile: "nonexistent_profile"}
	m.validateConfig()
	if m.config.ICCProfile != "" {
		t.Errorf("ICCProfile = %q, want empty after validation", m.config.ICCProfile)
	}
}

func TestValidateConfigWorkersOutOfRange(t *testing.T) {
	m := New()
	m.config = &Config{Workers: 0}
	m.validateConfig()
	if m.config.Workers != MaxWorkers() {
		t.Errorf("Workers = %d, want MaxWorkers %d", m.config.Workers, MaxWorkers())
	}

	m.config = &Config{Workers: 999}
	m.validateConfig()
	if m.config.Workers != MaxWorkers() {
		t.Errorf("Workers = %d, want MaxWorkers %d", m.config.Workers, MaxWorkers())
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	ctx := t.Context()
	m := New(WithContext(ctx))
	cfg := &Config{
		EnableSubfolder:   true,
		EnableCompression: true,
		ICCProfile:        "sRGB",
		Workers:           2,
	}
	m.config = cfg
	m.saveConfig()

	m2 := New(WithContext(ctx))
	m2.loadConfig()
	m2.validateConfig()
	got := m2.config
	if got.EnableSubfolder != cfg.EnableSubfolder {
		t.Errorf("EnableSubfolder = %v, want %v", got.EnableSubfolder, cfg.EnableSubfolder)
	}
	if got.EnableCompression != cfg.EnableCompression {
		t.Errorf("EnableCompression = %v, want %v", got.EnableCompression, cfg.EnableCompression)
	}
	if got.ICCProfile != cfg.ICCProfile {
		t.Errorf("ICCProfile = %q, want %q", got.ICCProfile, cfg.ICCProfile)
	}
	if got.Workers != cfg.Workers {
		t.Errorf("Workers = %d, want %d", got.Workers, cfg.Workers)
	}

	// cleanup: restore default config
	m2.config = NewConfig()
	m2.saveConfig()
}

func TestLoadConfigNotExist(t *testing.T) {
	ctx := t.Context()
	m := New(WithContext(ctx))

	// Ensure config file doesn't exist
	path := m.configPath()
	os.Remove(path)

	m.loadConfig()
	if m.config.Workers != MaxWorkers() {
		t.Errorf("Workers = %d, want MaxWorkers %d", m.config.Workers, MaxWorkers())
	}
	// loadConfig should have created the file via saveConfig
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("config file not created")
	}

	// cleanup
	os.Remove(path)
}

func TestLoadCorruptJSON(t *testing.T) {
	ctx := t.Context()
	m := New(WithContext(ctx))
	path := m.configPath()

	// Write corrupt JSON
	os.MkdirAll(filepath.Dir(path), 0755)
	os.WriteFile(path, []byte("{invalid json"), 0644)

	m.loadConfig()
	// Should fall back to defaults (NewConfig)
	if m.config.Workers != MaxWorkers() {
		t.Errorf("Workers after corrupt load = %d, want MaxWorkers %d", m.config.Workers, MaxWorkers())
	}

	// cleanup
	os.Remove(path)
}

func TestConfigFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file permissions not supported on Windows")
	}
	ctx := t.Context()
	m := New(WithContext(ctx))
	m.config = NewConfig()
	m.saveConfig()

	path := m.configPath()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0644 {
		t.Errorf("permissions = %04o, want 0644", got)
	}

	// cleanup
	os.Remove(path)
}
