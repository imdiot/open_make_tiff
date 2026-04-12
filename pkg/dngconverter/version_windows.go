//go:build windows

package dngconverter

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modversion                  = windows.NewLazySystemDLL("version.dll")
	procGetFileVersionInfoSizeW = modversion.NewProc("GetFileVersionInfoSizeW")
	procGetFileVersionInfoW     = modversion.NewProc("GetFileVersionInfoW")
	procVerQueryValueW          = modversion.NewProc("VerQueryValueW")
)

func readExecutableVersion(path string) (string, error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	size, _, err := procGetFileVersionInfoSizeW.Call(
		uintptr(unsafe.Pointer(pathPtr)), 0,
	)
	if size == 0 {
		return "", fmt.Errorf("GetFileVersionInfoSize: %w", err)
	}

	buf := make([]byte, size)
	rc, _, err := procGetFileVersionInfoW.Call(
		uintptr(unsafe.Pointer(pathPtr)), 0, size,
		uintptr(unsafe.Pointer(&buf[0])),
	)
	if rc == 0 {
		return "", fmt.Errorf("GetFileVersionInfo: %w", err)
	}

	rootPath, _ := syscall.UTF16PtrFromString(`\`)

	var ptr uintptr
	var ptrLen uintptr
	rc, _, err = procVerQueryValueW.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(rootPath)),
		uintptr(unsafe.Pointer(&ptr)),
		uintptr(unsafe.Pointer(&ptrLen)),
	)
	if rc == 0 {
		return "", fmt.Errorf("VerQueryValue: %w", err)
	}

	// VS_FIXEDFILEINFO layout (6 × uint32):
	//   [0] dwSignature        [1] dwStrucVersion
	//   [2] dwFileVersionMS    [3] dwFileVersionLS
	//   [4] dwProductVersionMS [5] dwProductVersionLS
	// FileVersionMS = (major << 16) | minor
	info := (*[6]uint32)(unsafe.Pointer(&ptr))
	major := info[2] >> 16
	minor := info[2] & 0xFFFF
	return fmt.Sprintf("%d.%d", major, minor), nil
}
