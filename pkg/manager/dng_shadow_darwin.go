//go:build darwin

package manager

import (
	"log/slog"
	"os"
	"path/filepath"

	"howett.net/plist"

	"open-make-tiff/pkg/dngconverter"
	"open-make-tiff/pkg/util"
)

// symlinkIfExists creates a symlink only if src exists.
func symlinkIfExists(src, dst string) error {
	if _, err := os.Stat(src); err != nil {
		return nil
	}
	return os.Symlink(src, dst)
}

// initDNGShadowBundle creates a Shadow Bundle to suppress the DNG Converter Dock icon.
func (m *Manager) initDNGShadowBundle() {
	if m.tmpDir == nil || !m.setting.EnableAdobeDNGConverter {
		return
	}

	dngExec := dngconverter.GetDefaultExecutablePath()
	if _, err := os.Stat(dngExec); err != nil {
		return
	}

	dngBundle := filepath.Dir(filepath.Dir(filepath.Dir(dngExec)))
	appName := filepath.Base(dngBundle)

	wrapperPath, err := util.ShadowBundle(m.tmpDir.Path(), appName, func(wp string) error {
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
	} else {
		m.dngConverterExecutable = filepath.Join(wrapperPath, "Contents", "MacOS", filepath.Base(dngExec))
		slog.Info("shadow bundle created", "path", wrapperPath)
	}
}
