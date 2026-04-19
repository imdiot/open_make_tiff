//go:build windows

package golibraw

/*
#include <libraw/libraw.h>
*/
import "C"

import (
	"runtime"
	"syscall"
	"unsafe"
)

// OpenFile uses libraw_open_wfile for non-ASCII path support on Windows.
func (rp *RawProcessor) OpenFile(path string) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return err
	}

	wPath, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return checkError(-1, ErrFileOpenFailed)
	}

	rc := C.libraw_open_wfile(rp.res.handle, (*C.wchar_t)(unsafe.Pointer(wPath)))
	runtime.KeepAlive(wPath)
	return checkError(rc, ErrFileOpenFailed)
}
