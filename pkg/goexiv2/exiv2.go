package goexiv2

/*
#cgo pkg-config: exiv2
#cgo CXXFLAGS: -std=c++17
#cgo darwin CFLAGS: -mmacosx-version-min=10.13
#cgo darwin CXXFLAGS: -mmacosx-version-min=10.13
#cgo !darwin LDFLAGS: -static-libstdc++ -static-libgcc

#include "wrapper.h"
#include <stdlib.h>
*/
import "C"

import (
	"runtime"
	"sync"
	"unsafe"
)

// Image wraps an exiv2 Image handle for reading EXIF/IPTC/XMP metadata.
type Image struct {
	handle unsafe.Pointer
	closed bool
	mu     sync.Mutex
}

// Open opens an image file by path (UTF-8) and returns an Image for metadata reading.
func Open(path string) (*Image, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	handle := C.goexiv2_open(cPath)
	if handle == nil {
		return nil, &OpenError{Path: path, Msg: goLastError()}
	}

	img := &Image{handle: handle}
	runtime.SetFinalizer(img, (*Image).Close)
	return img, nil
}

// Close releases the image handle. Safe to call multiple times.
func (img *Image) Close() error {
	img.mu.Lock()
	defer img.mu.Unlock()
	if img.closed {
		return nil
	}
	img.closed = true
	runtime.SetFinalizer(img, nil)
	C.goexiv2_close(img.handle)
	img.handle = nil
	return nil
}

func (img *Image) checkOpen() error {
	if img.closed || img.handle == nil {
		return ErrClosed
	}
	return nil
}

// ReadMetadata reads all metadata (EXIF, IPTC, XMP) from the image.
func (img *Image) ReadMetadata() error {
	img.mu.Lock()
	defer img.mu.Unlock()
	if err := img.checkOpen(); err != nil {
		return err
	}
	if C.goexiv2_read_metadata(img.handle) != 0 {
		return &ReadError{Op: "read_metadata", Msg: goLastError()}
	}
	return nil
}

// --- EXIF ---

// ExifCount returns the number of EXIF tags.
func (img *Image) ExifCount() int {
	img.mu.Lock()
	defer img.mu.Unlock()
	return int(C.goexiv2_exif_count(img.handle))
}

// ExifKey returns the EXIF tag key at the given index (e.g., "Exif.IFD0.Make").
func (img *Image) ExifKey(i int) (string, error) {
	img.mu.Lock()
	defer img.mu.Unlock()
	cs := C.goexiv2_exif_get_key(img.handle, C.int(i))
	if cs == nil {
		return "", &TagError{Op: "exif_key", Msg: "index out of range"}
	}
	defer C.goexiv2_free(unsafe.Pointer(cs))
	return C.GoString(cs), nil
}

// ExifHas reports whether the given EXIF key exists.
func (img *Image) ExifHas(key string) bool {
	img.mu.Lock()
	defer img.mu.Unlock()
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	return C.goexiv2_exif_has_key(img.handle, cKey) != 0
}

// ExifString returns the string value of the given EXIF key.
func (img *Image) ExifString(key string) (string, error) {
	img.mu.Lock()
	defer img.mu.Unlock()
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	cs := C.goexiv2_exif_get_string(img.handle, cKey)
	if cs == nil {
		return "", &TagError{Key: key, Op: "exif_get_string", Msg: "not found"}
	}
	defer C.goexiv2_free(unsafe.Pointer(cs))
	return C.GoString(cs), nil
}

// ExifLong returns the int64 value of the given EXIF key.
func (img *Image) ExifLong(key string) (int64, error) {
	img.mu.Lock()
	defer img.mu.Unlock()
	if err := img.checkOpen(); err != nil {
		return 0, err
	}
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	if C.goexiv2_exif_has_key(img.handle, cKey) == 0 {
		return 0, &TagError{Key: key, Op: "exif_get_long", Msg: "not found"}
	}
	return int64(C.goexiv2_exif_get_long(img.handle, cKey)), nil
}

// ExifDouble returns the float64 value of the given EXIF key.
func (img *Image) ExifDouble(key string) (float64, error) {
	img.mu.Lock()
	defer img.mu.Unlock()
	if err := img.checkOpen(); err != nil {
		return 0, err
	}
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	if C.goexiv2_exif_has_key(img.handle, cKey) == 0 {
		return 0, &TagError{Key: key, Op: "exif_get_double", Msg: "not found"}
	}
	return float64(C.goexiv2_exif_get_double(img.handle, cKey)), nil
}

// ExifBytes returns the raw byte value of the given EXIF key (e.g., MakerNote).
func (img *Image) ExifBytes(key string) ([]byte, error) {
	img.mu.Lock()
	defer img.mu.Unlock()
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	var length C.int
	ptr := C.goexiv2_exif_get_bytes(img.handle, cKey, &length)
	if ptr == nil || length == 0 {
		return nil, &TagError{Key: key, Op: "exif_get_bytes", Msg: "not found"}
	}
	return C.GoBytes(unsafe.Pointer(ptr), length), nil
}

// --- IPTC ---

// IPTCCount returns the number of IPTC tags.
func (img *Image) IPTCCount() int {
	img.mu.Lock()
	defer img.mu.Unlock()
	return int(C.goexiv2_iptc_count(img.handle))
}

// IPTCKey returns the IPTC tag key at the given index.
func (img *Image) IPTCKey(i int) (string, error) {
	img.mu.Lock()
	defer img.mu.Unlock()
	cs := C.goexiv2_iptc_get_key(img.handle, C.int(i))
	if cs == nil {
		return "", &TagError{Op: "iptc_key", Msg: "index out of range"}
	}
	defer C.goexiv2_free(unsafe.Pointer(cs))
	return C.GoString(cs), nil
}

// IPTCHas reports whether the given IPTC key exists.
func (img *Image) IPTCHas(key string) bool {
	img.mu.Lock()
	defer img.mu.Unlock()
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	return C.goexiv2_iptc_has_key(img.handle, cKey) != 0
}

// IPTCString returns the string value of the given IPTC key.
func (img *Image) IPTCString(key string) (string, error) {
	img.mu.Lock()
	defer img.mu.Unlock()
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	cs := C.goexiv2_iptc_get_string(img.handle, cKey)
	if cs == nil {
		return "", &TagError{Key: key, Op: "iptc_get_string", Msg: "not found"}
	}
	defer C.goexiv2_free(unsafe.Pointer(cs))
	return C.GoString(cs), nil
}

// --- XMP ---

// XMPCount returns the number of XMP tags.
func (img *Image) XMPCount() int {
	img.mu.Lock()
	defer img.mu.Unlock()
	return int(C.goexiv2_xmp_count(img.handle))
}

// XMPKey returns the XMP tag key at the given index.
func (img *Image) XMPKey(i int) (string, error) {
	img.mu.Lock()
	defer img.mu.Unlock()
	cs := C.goexiv2_xmp_get_key(img.handle, C.int(i))
	if cs == nil {
		return "", &TagError{Op: "xmp_key", Msg: "index out of range"}
	}
	defer C.goexiv2_free(unsafe.Pointer(cs))
	return C.GoString(cs), nil
}

// XMPHas reports whether the given XMP key exists.
func (img *Image) XMPHas(key string) bool {
	img.mu.Lock()
	defer img.mu.Unlock()
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	return C.goexiv2_xmp_has_key(img.handle, cKey) != 0
}

// XMPString returns the string value of the given XMP key.
func (img *Image) XMPString(key string) (string, error) {
	img.mu.Lock()
	defer img.mu.Unlock()
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	cs := C.goexiv2_xmp_get_string(img.handle, cKey)
	if cs == nil {
		return "", &TagError{Key: key, Op: "xmp_get_string", Msg: "not found"}
	}
	defer C.goexiv2_free(unsafe.Pointer(cs))
	return C.GoString(cs), nil
}

// XMPPacket returns the raw XMP packet as a string (complete XML).
// Returns empty string if the image has no XMP packet.
func (img *Image) XMPPacket() (string, error) {
	img.mu.Lock()
	defer img.mu.Unlock()
	if err := img.checkOpen(); err != nil {
		return "", err
	}
	cs := C.goexiv2_xmp_packet(img.handle)
	if cs == nil {
		return "", nil
	}
	defer C.goexiv2_free(unsafe.Pointer(cs))
	return C.GoString(cs), nil
}

// --- Utility ---

// Version returns the exiv2 library version string.
func Version() string {
	return C.GoString(C.goexiv2_version())
}

func goLastError() string {
	if s := C.goexiv2_get_last_error(); s != nil {
		return C.GoString(s)
	}
	return "unknown error"
}
