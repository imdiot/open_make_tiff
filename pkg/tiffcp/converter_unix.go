//go:build !windows

package tiffcp

import "syscall"

func getSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid: true,
	}
}
