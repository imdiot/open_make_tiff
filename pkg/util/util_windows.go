//go:build windows

package util

import (
	"os"
	"syscall"
)

var (
	kernel32          = syscall.NewLazyDLL("kernel32.dll")
	procAttachConsole = kernel32.NewProc("AttachConsole")
	procFreeConsole   = kernel32.NewProc("FreeConsole")
	procGetStdHandle  = kernel32.NewProc("GetStdHandle")
	procSetStdHandle  = kernel32.NewProc("SetStdHandle")
	procGetFileType   = kernel32.NewProc("GetFileType")
)

const (
	ATTACH_PARENT_PROCESS = ^uint32(0)
	STD_INPUT_HANDLE      = ^uint32(9)  // -10
	STD_OUTPUT_HANDLE     = ^uint32(10) // -11
	STD_ERROR_HANDLE      = ^uint32(11) // -12

	FILE_TYPE_CHAR = 0x0002
)

var attached bool

func AttachParentConsole() bool {
	hOut, _, _ := procGetStdHandle.Call(uintptr(STD_OUTPUT_HANDLE))

	needAttach := true
	if hOut != 0 {
		ft, _, _ := procGetFileType.Call(hOut)
		if uint32(ft) != FILE_TYPE_CHAR {
			needAttach = false
		}
	}

	if needAttach {
		ret, _, _ := procAttachConsole.Call(uintptr(ATTACH_PARENT_PROCESS))
		if ret == 0 {
			return false
		}
		attached = true
	}

	if f := openStdHandle(STD_INPUT_HANDLE); f != nil {
		os.Stdin = f
	}
	if f := openStdHandle(STD_OUTPUT_HANDLE); f != nil {
		os.Stdout = f
	}
	if f := openStdHandle(STD_ERROR_HANDLE); f != nil {
		os.Stderr = f
	}

	return true
}

func FreeParentConsole() {
	if attached {
		procFreeConsole.Call()
		attached = false
	}
}

func openStdHandle(stdHandle uint32) *os.File {
	h, _, _ := procGetStdHandle.Call(uintptr(stdHandle))
	if h == 0 {
		return nil
	}
	f := os.NewFile(h, "CONOUT$")
	if f == nil {
		return nil
	}
	procSetStdHandle.Call(uintptr(stdHandle), f.Fd())
	return f
}
