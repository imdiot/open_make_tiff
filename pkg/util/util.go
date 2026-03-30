package util

import (
	"os"
	"path/filepath"
	"runtime"
)

func GetDcrawEmuExecutable() (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(filepath.Dir(self), "third-party", "dcraw_emu.exe"), nil
	case "darwin":
		return filepath.Join(filepath.Dir(self), "third-party", "dcraw_emu"), nil
	}
	return "", nil
}

func GetTiffcpExecutable() (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(filepath.Dir(self), "third-party", "tiffcp.exe"), nil
	case "darwin":
		return filepath.Join(filepath.Dir(self), "third-party", "tiffcp"), nil
	}
	return "", nil
}

func GetExiftoolExecutable() (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(filepath.Dir(self), "third-party", "exiftool.exe"), nil
	case "darwin":
		return filepath.Join(filepath.Dir(self), "third-party", "exiftool"), nil
	}
	return "", nil
}
