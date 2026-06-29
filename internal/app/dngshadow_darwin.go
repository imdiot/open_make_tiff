//go:build darwin

package app

import (
	"log/slog"
	"os"
	"path/filepath"

	"howett.net/plist"

	"open-make-tiff/pkg/dngconverter"
	"open-make-tiff/pkg/util"
)

// symlinkIfExists creates a symlink only when src exists.
func symlinkIfExists(src, dst string) error {
	if _, err := os.Stat(src); err != nil {
		return nil
	}
	return os.Symlink(src, dst)
}

// initDNGShadowBundle creates a Shadow Bundle wrapper around the Adobe DNG
// Converter so it runs without claiming a Dock icon. Returns the wrapper's
// executable path, or "" when disabled, unavailable, or on failure.
func initDNGShadowBundle(tmpDir string, enable bool) string {
	if tmpDir == "" || !enable {
		return ""
	}

	dngExec := dngconverter.GetDefaultExecutablePath()
	if _, err := os.Stat(dngExec); err != nil {
		return ""
	}

	dngBundle := filepath.Dir(filepath.Dir(filepath.Dir(dngExec)))
	appName := filepath.Base(dngBundle)

	wrapperPath, err := util.ShadowBundle(tmpDir, appName, func(wp string) error {
		macOSPath := filepath.Join(wp, "Contents", "MacOS")
		if err := os.MkdirAll(macOSPath, 0755); err != nil {
			return err
		}
		if err := os.Symlink(dngExec, filepath.Join(macOSPath, filepath.Base(dngExec))); err != nil {
			return err
		}
		if err := symlinkIfExists(filepath.Join(dngBundle, "Contents", "Frameworks"), filepath.Join(wp, "Contents", "Frameworks")); err != nil {
			return err
		}
		if err := symlinkIfExists(filepath.Join(dngBundle, "Contents", "Resources"), filepath.Join(wp, "Contents", "Resources")); err != nil {
			return err
		}

		data, err := os.ReadFile(filepath.Join(dngBundle, "Contents", "Info.plist"))
		if err != nil {
			return err
		}
		var dict map[string]any
		if _, err := plist.Unmarshal(data, &dict); err != nil {
			return err
		}
		dict["LSUIElement"] = true
		out, err := plist.Marshal(dict, plist.XMLFormat)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(wp, "Contents", "Info.plist"), out, 0644)
	})
	if err != nil {
		slog.Warn("shadow bundle failed", "error", err)
		return ""
	}
	slog.Info("shadow bundle created", "path", wrapperPath)
	return filepath.Join(wrapperPath, "Contents", "MacOS", filepath.Base(dngExec))
}
