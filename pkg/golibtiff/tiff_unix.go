//go:build !windows

package golibtiff

/*
#cgo pkg-config: libtiff-4
#include <tiffio.h>
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"unsafe"
)

func openTiffHandle(path string, mode OpenMode, opts *C.TIFFOpenOptions) (*C.TIFF, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	cMode := C.CString(string(mode))
	defer C.free(unsafe.Pointer(cMode))

	tif := C.TIFFOpenExt(cPath, cMode, opts)
	if tif == nil {
		return nil, fmt.Errorf("libtiff: failed to open %q (mode %s)", path, mode)
	}
	return tif, nil
}
