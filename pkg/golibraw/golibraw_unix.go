//go:build !windows

package golibraw

/*
#include <libraw/libraw.h>
*/
import "C"

import "unsafe"

func (rp *RawProcessor) OpenFile(path string) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return err
	}

	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	rc := C.libraw_open_file(rp.res.handle, cPath)
	return checkError(rc, ErrFileOpenFailed)
}
