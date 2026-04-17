//go:build windows

package util

import (
	"os"
	"syscall"
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procAttachConsole  = kernel32.NewProc("AttachConsole")
	procGetStdHandle   = kernel32.NewProc("GetStdHandle")
)

const (
	ATTACH_PARENT_PROCESS = ^uint32(0)  // 0xFFFFFFFF
	STD_OUTPUT_HANDLE     = ^uint32(10) // 0xFFFFFFF5 = -11
	STD_ERROR_HANDLE      = ^uint32(11) // 0xFFFFFFF4 = -12
)

func AttachParentConsole() bool {
	ret, _, _ := procAttachConsole.Call(uintptr(ATTACH_PARENT_PROCESS))
	if ret == 0 {
		return false
	}

	stdout, _, _ := procGetStdHandle.Call(uintptr(STD_OUTPUT_HANDLE))
	stderr, _, _ := procGetStdHandle.Call(uintptr(STD_ERROR_HANDLE))
	os.Stdout = os.NewFile(uintptr(stdout), "stdout")
	os.Stderr = os.NewFile(uintptr(stderr), "stderr")
	return true
}
