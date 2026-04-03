//go:build !windows

package golibtiff

/*
#cgo pkg-config: libtiff-4
#include <tiffio.h>
#include <stdlib.h>
*/
import "C"

import "unsafe"

func openTiffHandle(path string, mode OpenMode) (*C.TIFF, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	cMode := C.CString(string(mode))
	defer C.free(unsafe.Pointer(cMode))

	return C.TIFFOpen(cPath, cMode), nil
}
