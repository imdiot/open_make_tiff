//go:build !darwin

package manager

// initDNGShadowBundle is a no-op on non-macOS platforms.
func (m *Manager) initDNGShadowBundle() {}
