//go:build !darwin

package util

// BundleCreateFn creates all Shadow Bundle content under wrapperPath.
type BundleCreateFn func(wrapperPath string) error

// ShadowBundle is a no-op on non-macOS platforms.
func ShadowBundle(tmpDir, appName string, createFn BundleCreateFn) (string, error) {
	return "", nil
}
