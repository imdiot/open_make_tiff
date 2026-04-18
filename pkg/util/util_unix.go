//go:build !windows

package util

func AttachParentConsole() bool { return false }
func FreeParentConsole()        {}
