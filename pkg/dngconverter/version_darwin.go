//go:build darwin

package dngconverter

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func readExecutableVersion(path string) (string, error) {
	// Derive Info.plist from executable: Contents/MacOS/foo → Contents/Info.plist
	plistPath := filepath.Join(filepath.Dir(filepath.Dir(path)), "Info.plist")

	out, err := exec.Command("defaults", "read", plistPath, "CFBundleShortVersionString").Output()
	if err != nil {
		return "", fmt.Errorf("read plist version: %w", err)
	}

	ver := strings.TrimSpace(string(out))
	if ver == "" {
		return "", fmt.Errorf("empty version from plist")
	}

	// Normalize to major.minor format.
	parts := strings.SplitN(ver, ".", 3)
	if len(parts) < 2 {
		return ver, nil
	}
	return parts[0] + "." + parts[1], nil
}
