//go:build !windows

package exiftool

func assignToJob(_ int) error { return nil }
