package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// setConfigHome redirects the config file into a per-test temp directory so
// tests never touch the real user config.
func setConfigHome(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("AppData", dir)
	} else {
		t.Setenv("HOME", dir)
	}
}

func TestDefault(t *testing.T) {
	cfg := Default()
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

func TestValidateIllegalProfile(t *testing.T) {
	cfg := &Config{ICCProfile: "nonexistent_profile"}
	cfg.Validate()
	if cfg.ICCProfile != "" {
		t.Errorf("ICCProfile = %q, want empty after validation", cfg.ICCProfile)
	}
}

func TestValidateWorkersOutOfRange(t *testing.T) {
	cfg := &Config{Workers: 0}
	cfg.Validate()
	if cfg.Workers != MaxWorkers() {
		t.Errorf("Workers = %d, want %d", cfg.Workers, MaxWorkers())
	}

	cfg = &Config{Workers: 999}
	cfg.Validate()
	if cfg.Workers != MaxWorkers() {
		t.Errorf("Workers = %d, want %d", cfg.Workers, MaxWorkers())
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	setConfigHome(t)

	cfg := &Config{
		EnableSubfolder:   true,
		EnableCompression: true,
		ICCProfile:        "sRGB",
		Workers:           2,
	}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	got := Load()
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
}

func TestLoadNotExist(t *testing.T) {
	setConfigHome(t)

	cfg := Load()
	if cfg.Workers != MaxWorkers() {
		t.Errorf("Workers = %d, want %d", cfg.Workers, MaxWorkers())
	}
	// Load should have created the file with defaults.
	if _, err := os.Stat(Path()); os.IsNotExist(err) {
		t.Error("config file not created")
	}
}

func TestLoadCorruptJSON(t *testing.T) {
	setConfigHome(t)

	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{invalid json"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := Load()
	if cfg.Workers != MaxWorkers() {
		t.Errorf("Workers after corrupt load = %d, want %d", cfg.Workers, MaxWorkers())
	}
}

func TestPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file permissions not supported on Windows")
	}
	setConfigHome(t)

	if err := Default().Save(); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(Path())
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0644 {
		t.Errorf("permissions = %04o, want 0644", got)
	}
}
