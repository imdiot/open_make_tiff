//go:build darwin

package util

import "path/filepath"

// BundleCreateFn creates all Shadow Bundle content under wrapperPath.
type BundleCreateFn func(wrapperPath string) error

// ShadowBundle computes the .app wrapper path, calls createFn to populate it, and returns the path.
func ShadowBundle(tmpDir, appName string, createFn BundleCreateFn) (string, error) {
	wrapperPath := filepath.Join(tmpDir, appName)
	if err := createFn(wrapperPath); err != nil {
		return "", err
	}
	return wrapperPath, nil
}
