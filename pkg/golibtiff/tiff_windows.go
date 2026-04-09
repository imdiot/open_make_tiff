//go:build windows

package golibtiff

/*
#cgo pkg-config: libtiff-4
#include <tiffio.h>
#include <stdlib.h>
*/
import "C"

import (
	"syscall"
	"unsafe"
)

func openTiffHandle(path string, mode OpenMode, opts *C.TIFFOpenOptions) (*C.TIFF, error) {
	wPath, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}

	cMode := C.CString(string(mode))
	defer C.free(unsafe.Pointer(cMode))

	return C.TIFFOpenWExt((*C.wchar_t)(unsafe.Pointer(wPath)), cMode, opts), nil
}
