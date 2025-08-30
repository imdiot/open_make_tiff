package util

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

func EnableAdobeDNGConverter() bool {
	_, err := os.Stat(GetAdobeDNGConverterExecutable())
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		return true
	}
	return false
}

func GetAdobeDNGConverterExecutable() string {
	switch runtime.GOOS {
	case "windows":
		return "C:\\Program Files\\Adobe\\Adobe DNG Converter\\Adobe DNG Converter.exe"
	case "darwin":
		return "/Applications/Adobe DNG Converter.app/Contents/MacOS/Adobe DNG Converter"
	}
	return ""
}

func GetDcrawEmuExecutable() (string, error) {
	//return "D:\\Work\\open-make-tiff\\bin\\third-party\\dcraw_emu.exe", nil
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
	//return "D:\\Work\\open-make-tiff\\bin\\third-party\\tiffcp.exe", nil
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

func GetExiv2Executable() (string, error) {
	//return "D:\\Work\\open-make-tiff\\bin\\third-party\\exiv2.exe", nil
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(filepath.Dir(self), "third-party", "exiv2.exe"), nil
	case "darwin":
		return filepath.Join(filepath.Dir(self), "third-party", "exiv2"), nil
	}
	return "", nil
}

func GetExiftoolExecutable() (string, error) {
	//return "D:\\Work\\open-make-tiff\\bin\\third-party\\exiv2.exe", nil
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
