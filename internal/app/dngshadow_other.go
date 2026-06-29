//go:build !darwin

package app

// initDNGShadowBundle is a no-op on non-macOS platforms.
func initDNGShadowBundle(tmpDir string, enable bool) string { return "" }
