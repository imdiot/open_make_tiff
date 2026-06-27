package util

import (
	"os"
	"path/filepath"
	"runtime"
)

func GetExiftoolExecutable() (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "windows":
		p := filepath.Join(filepath.Dir(self), "third-party", "exiftool.exe")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
		// Dev (wails dev): CWD is the project root — use the vendored
		// exiftool so dev matches the bundled version, not a system install.
		if cwd, err := os.Getwd(); err == nil {
			dev := filepath.Join(cwd, "third-party", "windows-x64", "exiftool.exe")
			if _, err := os.Stat(dev); err == nil {
				return dev, nil
			}
		}
		return "", nil
	case "darwin":
		// Production: .app bundle (executable under Contents/MacOS/,
		// resources under Contents/Resources/).
		p := filepath.Join(filepath.Dir(self), "..", "Resources", "third-party", "exiftool")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
		// Dev (wails dev): CWD is the project root — use the vendored
		// exiftool so dev matches the bundled version, not a system install.
		if cwd, err := os.Getwd(); err == nil {
			dev := filepath.Join(cwd, "third-party", "macos-universal", "exiftool")
			if _, err := os.Stat(dev); err == nil {
				return dev, nil
			}
		}
		return "", nil
	}
	return "", nil
}
