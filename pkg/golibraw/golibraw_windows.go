//go:build windows

package golibraw

/*
#include <libraw/libraw.h>
*/
import "C"

import (
	"syscall"
	"unsafe"
)

// OpenFile 打开 RAW 图像文件。
// Windows 上使用 libraw_open_wfile 支持非 ASCII 路径。
func (rp *RawProcessor) OpenFile(path string) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return err
	}

	wPath, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return librawError(-1, ErrFileOpenFailed)
	}

	rc := C.libraw_open_wfile(rp.handle, (*C.wchar_t)(unsafe.Pointer(wPath)))
	return librawError(rc, ErrFileOpenFailed)
}
