//go:build windows

package util

import "syscall"

func GetSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
		//StartupInfo: &syscall.StartupInfo{
		//	ShowWindow: syscall.SW_HIDE,
		//	Flags:      syscall.STARTF_USESHOWWINDOW,
		//},
	}
}
