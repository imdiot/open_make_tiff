package golibtiff

/*
#include <tiffio.h>
#include <stdlib.h>
#include "libtiff_bridge.h"
*/
import "C"

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

// TIFF represents an open TIFF file handle.
// It must be closed after use via Close() to release resources.
// A TIFF handle is NOT thread-safe; do not use concurrently from multiple goroutines.
type TIFF struct {
	tif       *C.TIFF
	closeOnce sync.Once
	cleanup   runtime.Cleanup
}

// OpenMode controls how a TIFF file is opened.
type OpenMode string

const (
	OpenRead      OpenMode = "r"
	OpenWrite     OpenMode = "w"
	OpenAppend    OpenMode = "a"
	OpenReadWrite OpenMode = "r+"
	OpenBigTIFF   OpenMode = "w8"
)

// --- Core Operations ---

// Open opens a TIFF file at the given path with the specified mode.
func Open(path string, mode OpenMode) (*TIFF, error) {
	C.clearOpenPhaseError()

	opts := C.TIFFOpenOptionsAlloc()
	defer C.TIFFOpenOptionsFree(opts)
	var handler C.TIFFErrorHandlerExtR
	C.getPerHandleErrorHandler(&handler)
	C.TIFFOpenOptionsSetErrorHandlerExtR(opts, handler, nil)
	C.TIFFOpenOptionsSetWarningHandlerExtR(opts, handler, nil)

	tif, err := openTiffHandle(path, mode, opts)
	if err != nil {
		if C.hasOpenPhaseError() != 0 {
			return nil, &OpenError{Path: path, Mode: mode, Msg: C.GoString(C.getOpenPhaseError())}
		}
		return nil, &OpenError{Path: path, Mode: mode, Msg: err.Error()}
	}

	C.attachErrorState(tif)

	t := &TIFF{tif: tif}
	t.cleanup = runtime.AddCleanup(t, func(tif *C.TIFF) {
		C.detachErrorState(tif)
		C.TIFFClose(tif)
	}, tif)
	return t, nil
}

// Close releases the TIFF file handle resources. Safe to call multiple times.
func (t *TIFF) Close() error {
	t.closeOnce.Do(func() {
		if t.tif != nil {
			t.cleanup.Stop()
			C.detachErrorState(t.tif)
			C.TIFFClose(t.tif)
			t.tif = nil
		}
	})
	return nil
}

// IsOpen reports whether the TIFF handle is still open and usable.
func (t *TIFF) IsOpen() bool {
	return t.tif != nil
}

// FileName returns the file name associated with the TIFF handle.
func (t *TIFF) FileName() string {
	if t.tif == nil {
		return ""
	}
	return C.GoString(C.TIFFFileName(t.tif))
}

func (t *TIFF) checkOpen() error {
	if t.tif == nil {
		return ErrClosed
	}
	return nil
}

func (t *TIFF) lastError() error {
	if C.hasHandleError(t.tif) != 0 {
		return fmt.Errorf("%s", C.GoString(C.getHandleError(t.tif)))
	}
	return nil
}

// --- Library info ---

// GetVersion returns the libtiff version string.
func GetVersion() string {
	return C.GoString(C.TIFFGetVersion())
}

// --- GetField ---

func (t *TIFF) GetFieldUint16(tag Tag) (uint16, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	C.clearHandleError(t.tif)
	var val C.uint16_t
	if C.tiffGetFieldU16(t.tif, C.uint32_t(tag), &val) == 0 {
		if err := t.lastError(); err != nil {
			return 0, &FieldError{Tag: tag, Op: "get", Msg: err.Error()}
		}
		return 0, &FieldError{Tag: tag, Op: "get", Msg: "field not found"}
	}
	return uint16(val), nil
}

func (t *TIFF) GetFieldUint32(tag Tag) (uint32, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	C.clearHandleError(t.tif)
	var val C.uint32_t
	if C.tiffGetFieldU32(t.tif, C.uint32_t(tag), &val) == 0 {
		if err := t.lastError(); err != nil {
			return 0, &FieldError{Tag: tag, Op: "get", Msg: err.Error()}
		}
		return 0, &FieldError{Tag: tag, Op: "get", Msg: "field not found"}
	}
	return uint32(val), nil
}

func (t *TIFF) GetFieldFloat(tag Tag) (float64, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	C.clearHandleError(t.tif)
	var val C.float
	if C.tiffGetFieldFloat(t.tif, C.uint32_t(tag), &val) == 0 {
		if err := t.lastError(); err != nil {
			return 0, &FieldError{Tag: tag, Op: "get", Msg: err.Error()}
		}
		return 0, &FieldError{Tag: tag, Op: "get", Msg: "field not found"}
	}
	return float64(val), nil
}

// GetFieldDouble reads a double-precision (64-bit) tag value.
// Use this for DOUBLE type tags or when full RATIONAL precision is needed
// (GetFieldFloat uses 32-bit C float which loses RATIONAL precision).
func (t *TIFF) GetFieldDouble(tag Tag) (float64, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	C.clearHandleError(t.tif)
	var val C.double
	if C.tiffGetFieldDouble(t.tif, C.uint32_t(tag), &val) == 0 {
		if err := t.lastError(); err != nil {
			return 0, &FieldError{Tag: tag, Op: "get", Msg: err.Error()}
		}
		return 0, &FieldError{Tag: tag, Op: "get", Msg: "field not found"}
	}
	return float64(val), nil
}

func (t *TIFF) GetFieldString(tag Tag) (string, error) {
	if err := t.checkOpen(); err != nil {
		return "", err
	}
	C.clearHandleError(t.tif)
	var val *C.char
	if C.tiffGetFieldString(t.tif, C.uint32_t(tag), &val) == 0 {
		if err := t.lastError(); err != nil {
			return "", &FieldError{Tag: tag, Op: "get", Msg: err.Error()}
		}
		return "", &FieldError{Tag: tag, Op: "get", Msg: "field not found"}
	}
	return C.GoString(val), nil
}

func (t *TIFF) GetFieldUint8(tag Tag) (uint8, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	C.clearHandleError(t.tif)
	var val C.uint8_t
	if C.tiffGetFieldU8(t.tif, C.uint32_t(tag), &val) == 0 {
		if err := t.lastError(); err != nil {
			return 0, &FieldError{Tag: tag, Op: "get", Msg: err.Error()}
		}
		return 0, &FieldError{Tag: tag, Op: "get", Msg: "field not found"}
	}
	return uint8(val), nil
}

func (t *TIFF) GetFieldUint64(tag Tag) (uint64, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	C.clearHandleError(t.tif)
	var val C.uint64_t
	if C.tiffGetFieldU64(t.tif, C.uint32_t(tag), &val) == 0 {
		if err := t.lastError(); err != nil {
			return 0, &FieldError{Tag: tag, Op: "get", Msg: err.Error()}
		}
		return 0, &FieldError{Tag: tag, Op: "get", Msg: "field not found"}
	}
	return uint64(val), nil
}

func (t *TIFF) GetFieldInt8(tag Tag) (int8, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	C.clearHandleError(t.tif)
	var val C.int8_t
	if C.tiffGetFieldS8(t.tif, C.uint32_t(tag), &val) == 0 {
		if err := t.lastError(); err != nil {
			return 0, &FieldError{Tag: tag, Op: "get", Msg: err.Error()}
		}
		return 0, &FieldError{Tag: tag, Op: "get", Msg: "field not found"}
	}
	return int8(val), nil
}

func (t *TIFF) GetFieldInt16(tag Tag) (int16, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	C.clearHandleError(t.tif)
	var val C.int16_t
	if C.tiffGetFieldS16(t.tif, C.uint32_t(tag), &val) == 0 {
		if err := t.lastError(); err != nil {
			return 0, &FieldError{Tag: tag, Op: "get", Msg: err.Error()}
		}
		return 0, &FieldError{Tag: tag, Op: "get", Msg: "field not found"}
	}
	return int16(val), nil
}

func (t *TIFF) GetFieldInt32(tag Tag) (int32, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	C.clearHandleError(t.tif)
	var val C.int32_t
	if C.tiffGetFieldS32(t.tif, C.uint32_t(tag), &val) == 0 {
		if err := t.lastError(); err != nil {
			return 0, &FieldError{Tag: tag, Op: "get", Msg: err.Error()}
		}
		return 0, &FieldError{Tag: tag, Op: "get", Msg: "field not found"}
	}
	return int32(val), nil
}

func (t *TIFF) GetFieldInt64(tag Tag) (int64, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	C.clearHandleError(t.tif)
	var val C.int64_t
	if C.tiffGetFieldS64(t.tif, C.uint32_t(tag), &val) == 0 {
		if err := t.lastError(); err != nil {
			return 0, &FieldError{Tag: tag, Op: "get", Msg: err.Error()}
		}
		return 0, &FieldError{Tag: tag, Op: "get", Msg: "field not found"}
	}
	return int64(val), nil
}

// GetFieldByteSlice reads a BYTE/UNDEFINED array field (e.g. ICC Profile, XMP, MakerNotes).
func (t *TIFF) GetFieldByteSlice(tag Tag) ([]byte, error) {
	if err := t.checkOpen(); err != nil {
		return nil, err
	}
	C.clearHandleError(t.tif)
	var data *C.uint8_t
	var count C.uint32_t
	if C.tiffGetFieldByteSlice(t.tif, C.uint32_t(tag), &data, &count) == 0 {
		if err := t.lastError(); err != nil {
			return nil, &FieldError{Tag: tag, Op: "get", Msg: err.Error()}
		}
		return nil, &FieldError{Tag: tag, Op: "get", Msg: "field not found"}
	}
	length := int(count)
	if length == 0 || data == nil {
		return nil, nil
	}
	result := make([]byte, length)
	copy(result, unsafe.Slice((*byte)(unsafe.Pointer(data)), length))
	return result, nil
}

func (t *TIFF) GetFieldUint16Slice(tag Tag) ([]uint16, error) {
	if err := t.checkOpen(); err != nil {
		return nil, err
	}
	C.clearHandleError(t.tif)
	var data *C.uint16_t
	var count C.uint16_t
	if C.tiffGetFieldU16Array(t.tif, C.uint32_t(tag), &data, &count) == 0 {
		if err := t.lastError(); err != nil {
			return nil, &FieldError{Tag: tag, Op: "get", Msg: err.Error()}
		}
		return nil, &FieldError{Tag: tag, Op: "get", Msg: "field not found"}
	}
	length := int(count)
	if length == 0 || data == nil {
		return nil, nil
	}
	result := make([]uint16, length)
	copy(result, unsafe.Slice((*uint16)(unsafe.Pointer(data)), length))
	return result, nil
}

func (t *TIFF) GetFieldUint32Slice(tag Tag) ([]uint32, error) {
	if err := t.checkOpen(); err != nil {
		return nil, err
	}
	C.clearHandleError(t.tif)
	var data *C.uint32_t
	var count C.uint32_t
	if C.tiffGetFieldU32Array(t.tif, C.uint32_t(tag), &data, &count) == 0 {
		if err := t.lastError(); err != nil {
			return nil, &FieldError{Tag: tag, Op: "get", Msg: err.Error()}
		}
		return nil, &FieldError{Tag: tag, Op: "get", Msg: "field not found"}
	}
	length := int(count)
	if length == 0 || data == nil {
		return nil, nil
	}
	result := make([]uint32, length)
	copy(result, unsafe.Slice((*uint32)(unsafe.Pointer(data)), length))
	return result, nil
}

// --- SetField ---

func (t *TIFF) SetFieldUint16(tag Tag, v uint16) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffSetFieldU16(t.tif, C.uint32_t(tag), C.uint16_t(v)) == 0 {
		if err := t.lastError(); err != nil {
			return &FieldError{Tag: tag, Op: "set", Msg: err.Error()}
		}
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	return nil
}

func (t *TIFF) SetFieldUint32(tag Tag, v uint32) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffSetFieldU32(t.tif, C.uint32_t(tag), C.uint32_t(v)) == 0 {
		if err := t.lastError(); err != nil {
			return &FieldError{Tag: tag, Op: "set", Msg: err.Error()}
		}
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	return nil
}

func (t *TIFF) SetFieldFloat(tag Tag, v float64) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffSetFieldFloat(t.tif, C.uint32_t(tag), C.float(v)) == 0 {
		if err := t.lastError(); err != nil {
			return &FieldError{Tag: tag, Op: "set", Msg: err.Error()}
		}
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	return nil
}

func (t *TIFF) SetFieldString(tag Tag, v string) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	cStr := C.CString(v)
	defer C.free(unsafe.Pointer(cStr))
	C.clearHandleError(t.tif)
	if C.tiffSetFieldString(t.tif, C.uint32_t(tag), cStr) == 0 {
		if err := t.lastError(); err != nil {
			return &FieldError{Tag: tag, Op: "set", Msg: err.Error()}
		}
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	return nil
}

func (t *TIFF) SetFieldUint16Slice(tag Tag, v []uint16) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	if len(v) == 0 {
		return nil
	}
	C.clearHandleError(t.tif)
	if C.tiffSetFieldU16Array(t.tif, C.uint32_t(tag), C.uint16_t(len(v)), (*C.uint16_t)(unsafe.Pointer(&v[0]))) == 0 {
		runtime.KeepAlive(v)
		if err := t.lastError(); err != nil {
			return &FieldError{Tag: tag, Op: "set", Msg: err.Error()}
		}
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	runtime.KeepAlive(v)
	return nil
}

func (t *TIFF) SetFieldUint32Slice(tag Tag, v []uint32) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	if len(v) == 0 {
		return nil
	}
	C.clearHandleError(t.tif)
	if C.tiffSetFieldU32Array(t.tif, C.uint32_t(tag), C.uint32_t(len(v)), (*C.uint32_t)(unsafe.Pointer(&v[0]))) == 0 {
		runtime.KeepAlive(v)
		if err := t.lastError(); err != nil {
			return &FieldError{Tag: tag, Op: "set", Msg: err.Error()}
		}
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	runtime.KeepAlive(v)
	return nil
}

// --- Extended SetField ---

// SetFieldByteSlice sets a byte-array field (e.g. ICC Profile, XMP, MakerNotes).
func (t *TIFF) SetFieldByteSlice(tag Tag, v []byte) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	if len(v) == 0 {
		return nil
	}
	C.clearHandleError(t.tif)
	if C.tiffSetFieldByteSlice(t.tif, C.uint32_t(tag), C.uint32_t(len(v)), (*C.uint8_t)(unsafe.Pointer(&v[0]))) == 0 {
		runtime.KeepAlive(v)
		if err := t.lastError(); err != nil {
			return &FieldError{Tag: tag, Op: "set", Msg: err.Error()}
		}
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	runtime.KeepAlive(v)
	return nil
}

// SetFieldC0ByteSlice sets a fixed-count byte-array field (no count argument).
func (t *TIFF) SetFieldC0ByteSlice(tag Tag, v []byte) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	if len(v) == 0 {
		return nil
	}
	C.clearHandleError(t.tif)
	if C.tiffSetFieldC0ByteSlice(t.tif, C.uint32_t(tag), (*C.uint8_t)(unsafe.Pointer(&v[0]))) == 0 {
		runtime.KeepAlive(v)
		if err := t.lastError(); err != nil {
			return &FieldError{Tag: tag, Op: "set", Msg: err.Error()}
		}
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	runtime.KeepAlive(v)
	return nil
}

// SetFieldC0Uint16Slice sets a fixed-count uint16-array field (no count argument).
func (t *TIFF) SetFieldC0Uint16Slice(tag Tag, v []uint16) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	if len(v) == 0 {
		return nil
	}
	C.clearHandleError(t.tif)
	if C.tiffSetFieldC0U16(t.tif, C.uint32_t(tag), (*C.uint16_t)(unsafe.Pointer(&v[0]))) == 0 {
		runtime.KeepAlive(v)
		if err := t.lastError(); err != nil {
			return &FieldError{Tag: tag, Op: "set", Msg: err.Error()}
		}
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	runtime.KeepAlive(v)
	return nil
}

// SetFieldC0Uint32Slice sets a fixed-count uint32-array field (no count argument).
func (t *TIFF) SetFieldC0Uint32Slice(tag Tag, v []uint32) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	if len(v) == 0 {
		return nil
	}
	C.clearHandleError(t.tif)
	if C.tiffSetFieldC0U32(t.tif, C.uint32_t(tag), (*C.uint32_t)(unsafe.Pointer(&v[0]))) == 0 {
		runtime.KeepAlive(v)
		if err := t.lastError(); err != nil {
			return &FieldError{Tag: tag, Op: "set", Msg: err.Error()}
		}
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	runtime.KeepAlive(v)
	return nil
}

// SetFieldFloatSlice sets a RATIONAL array field.
func (t *TIFF) SetFieldFloatSlice(tag Tag, v []float64) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	if len(v) == 0 {
		return nil
	}
	C.clearHandleError(t.tif)
	floats := make([]C.float, len(v))
	for i, f := range v {
		floats[i] = C.float(f)
	}
	if C.tiffSetFieldFloatSlice(t.tif, C.uint32_t(tag), C.int(len(v)), &floats[0]) == 0 {
		runtime.KeepAlive(floats)
		if err := t.lastError(); err != nil {
			return &FieldError{Tag: tag, Op: "set", Msg: err.Error()}
		}
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	runtime.KeepAlive(floats)
	return nil
}

// SetFieldUint64 sets a uint64 field.
func (t *TIFF) SetFieldUint64(tag Tag, v uint64) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffSetFieldU64(t.tif, C.uint32_t(tag), C.uint64_t(v)) == 0 {
		if err := t.lastError(); err != nil {
			return &FieldError{Tag: tag, Op: "set", Msg: err.Error()}
		}
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	return nil
}

func (t *TIFF) SetFieldInt8(tag Tag, v int8) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffSetFieldS8(t.tif, C.uint32_t(tag), C.int8_t(v)) == 0 {
		if err := t.lastError(); err != nil {
			return &FieldError{Tag: tag, Op: "set", Msg: err.Error()}
		}
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	return nil
}

func (t *TIFF) SetFieldInt16(tag Tag, v int16) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffSetFieldS16(t.tif, C.uint32_t(tag), C.int16_t(v)) == 0 {
		if err := t.lastError(); err != nil {
			return &FieldError{Tag: tag, Op: "set", Msg: err.Error()}
		}
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	return nil
}

func (t *TIFF) SetFieldInt32(tag Tag, v int32) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffSetFieldS32(t.tif, C.uint32_t(tag), C.int32_t(v)) == 0 {
		if err := t.lastError(); err != nil {
			return &FieldError{Tag: tag, Op: "set", Msg: err.Error()}
		}
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	return nil
}

func (t *TIFF) SetFieldInt64(tag Tag, v int64) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffSetFieldS64(t.tif, C.uint32_t(tag), C.int64_t(v)) == 0 {
		if err := t.lastError(); err != nil {
			return &FieldError{Tag: tag, Op: "set", Msg: err.Error()}
		}
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	return nil
}

// SetFieldUint8 sets a single-byte field.
func (t *TIFF) SetFieldUint8(tag Tag, v uint8) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffSetFieldU8(t.tif, C.uint32_t(tag), C.uint8_t(v)) == 0 {
		if err := t.lastError(); err != nil {
			return &FieldError{Tag: tag, Op: "set", Msg: err.Error()}
		}
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	return nil
}

// SetFieldC0FloatSlice sets a fixed-count float array field.
func (t *TIFF) SetFieldC0FloatSlice(tag Tag, v []float64) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	if len(v) == 0 {
		return nil
	}
	C.clearHandleError(t.tif)
	floats := make([]C.float, len(v))
	for i, f := range v {
		floats[i] = C.float(f)
	}
	if C.tiffSetFieldC0Float(t.tif, C.uint32_t(tag), &floats[0]) == 0 {
		runtime.KeepAlive(floats)
		if err := t.lastError(); err != nil {
			return &FieldError{Tag: tag, Op: "set", Msg: err.Error()}
		}
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	runtime.KeepAlive(floats)
	return nil
}

// SetFieldDouble sets a double-precision floating-point field (64-bit, no precision loss).
func (t *TIFF) SetFieldDouble(tag Tag, v float64) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffSetFieldDouble(t.tif, C.uint32_t(tag), C.double(v)) == 0 {
		if err := t.lastError(); err != nil {
			return &FieldError{Tag: tag, Op: "set", Msg: err.Error()}
		}
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	return nil
}

// SetFieldDoubleSlice sets a double-precision float array field with count.
func (t *TIFF) SetFieldDoubleSlice(tag Tag, v []float64) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	if len(v) == 0 {
		return nil
	}
	C.clearHandleError(t.tif)
	doubles := make([]C.double, len(v))
	for i, f := range v {
		doubles[i] = C.double(f)
	}
	if C.tiffSetFieldDoubleSlice(t.tif, C.uint32_t(tag), C.int(len(v)), &doubles[0]) == 0 {
		runtime.KeepAlive(doubles)
		if err := t.lastError(); err != nil {
			return &FieldError{Tag: tag, Op: "set", Msg: err.Error()}
		}
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	runtime.KeepAlive(doubles)
	return nil
}

// SetFieldC0DoubleSlice sets a fixed-count double array field (no count argument).
func (t *TIFF) SetFieldC0DoubleSlice(tag Tag, v []float64) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	if len(v) == 0 {
		return nil
	}
	C.clearHandleError(t.tif)
	doubles := make([]C.double, len(v))
	for i, f := range v {
		doubles[i] = C.double(f)
	}
	if C.tiffSetFieldC0Double(t.tif, C.uint32_t(tag), &doubles[0]) == 0 {
		runtime.KeepAlive(doubles)
		if err := t.lastError(); err != nil {
			return &FieldError{Tag: tag, Op: "set", Msg: err.Error()}
		}
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	runtime.KeepAlive(doubles)
	return nil
}

// UnsetField removes a tag from the current IFD.
func (t *TIFF) UnsetField(tag Tag) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffUnsetField(t.tif, C.uint32_t(tag)) == 0 {
		if err := t.lastError(); err != nil {
			return &FieldError{Tag: tag, Op: "unset", Msg: err.Error()}
		}
		return &FieldError{Tag: tag, Op: "unset", Msg: "failed"}
	}
	return nil
}

// --- Field introspection ---

// IsFieldKnown checks if a tag is registered in libtiff's field definitions.
func (t *TIFF) IsFieldKnown(tag Tag) bool {
	if t.tif == nil {
		return false
	}
	return C.tiffIsFieldKnown(t.tif, C.uint32_t(tag)) != 0
}

// GetFieldType returns the libtiff-registered TIFFDataType for a tag.
// Returns -1 if the tag is not registered.
func (t *TIFF) GetFieldType(tag Tag) DataType {
	if t.tif == nil {
		return -1
	}
	return DataType(C.tiffGetFieldType(t.tif, C.uint32_t(tag)))
}

// FieldPassCount reports whether a tag requires a count argument in TIFFSetField.
func (t *TIFF) FieldPassCount(tag Tag) bool {
	if t.tif == nil {
		return false
	}
	return C.tiffFieldPassCount(t.tif, C.uint32_t(tag)) != 0
}

// FieldWriteCount returns the number of values a tag expects.
func (t *TIFF) FieldWriteCount(tag Tag) int {
	if t.tif == nil {
		return 0
	}
	return int(C.tiffFieldWriteCount(t.tif, C.uint32_t(tag)))
}

// FieldSetGetSize returns the per-element storage size in bytes for the tag.
// Returns 4 for SETGET_*_FLOAT tags, 8 for SETGET_*_DOUBLE tags, -1 if unknown.
func (t *TIFF) FieldSetGetSize(tag Tag) int {
	if t.tif == nil {
		return -1
	}
	return int(C.tiffFieldSetGetSize(t.tif, C.uint32_t(tag)))
}

// --- GetFieldDefaulted ---

// GetFieldDefaultedUint16 reads a uint16 tag, returning the default value if unset.
func (t *TIFF) GetFieldDefaultedUint16(tag Tag) (uint16, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	var v C.uint16_t
	C.clearHandleError(t.tif)
	if C.tiffGetFieldDefaultedU16(t.tif, C.uint32_t(tag), &v) == 0 {
		if err := t.lastError(); err != nil {
			return 0, &FieldError{Tag: tag, Op: "get_defaulted", Msg: err.Error()}
		}
		return 0, &FieldError{Tag: tag, Op: "get_defaulted", Msg: "failed"}
	}
	return uint16(v), nil
}

// GetFieldDefaultedUint32 reads a uint32 tag, returning the default value if unset.
func (t *TIFF) GetFieldDefaultedUint32(tag Tag) (uint32, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	var v C.uint32_t
	C.clearHandleError(t.tif)
	if C.tiffGetFieldDefaultedU32(t.tif, C.uint32_t(tag), &v) == 0 {
		if err := t.lastError(); err != nil {
			return 0, &FieldError{Tag: tag, Op: "get_defaulted", Msg: err.Error()}
		}
		return 0, &FieldError{Tag: tag, Op: "get_defaulted", Msg: "failed"}
	}
	return uint32(v), nil
}

// GetFieldDefaultedFloat reads a float tag, returning the default value if unset.
func (t *TIFF) GetFieldDefaultedFloat(tag Tag) (float64, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	var v C.float
	C.clearHandleError(t.tif)
	if C.tiffGetFieldDefaultedFloat(t.tif, C.uint32_t(tag), &v) == 0 {
		if err := t.lastError(); err != nil {
			return 0, &FieldError{Tag: tag, Op: "get_defaulted", Msg: err.Error()}
		}
		return 0, &FieldError{Tag: tag, Op: "get_defaulted", Msg: "failed"}
	}
	return float64(v), nil
}

// GetFieldDefaultedString reads a string tag, returning the default value if unset.
func (t *TIFF) GetFieldDefaultedString(tag Tag) (string, error) {
	if err := t.checkOpen(); err != nil {
		return "", err
	}
	var v *C.char
	C.clearHandleError(t.tif)
	if C.tiffGetFieldDefaultedString(t.tif, C.uint32_t(tag), &v) == 0 {
		if err := t.lastError(); err != nil {
			return "", &FieldError{Tag: tag, Op: "get_defaulted", Msg: err.Error()}
		}
		return "", &FieldError{Tag: tag, Op: "get_defaulted", Msg: "failed"}
	}
	return C.GoString(v), nil
}

// --- Auto-dispatch ---

// SetFieldAny writes a value to a tag, automatically dispatching to the correct
// SetField* variant based on the Go type of value and the tag's field metadata
// (pass-count, storage size).
//
// Supported value types:
//   - int8, int16, int32, int64, uint8, uint16, uint32, uint64, float64, string
//   - []byte, []uint16, []uint32, []float64
func (t *TIFF) SetFieldAny(tag Tag, value any) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	passCount := t.FieldPassCount(tag)
	sz := t.FieldSetGetSize(tag)

	switch v := value.(type) {
	case uint8:
		return t.SetFieldUint8(tag, v)
	case uint16:
		return t.SetFieldUint16(tag, v)
	case uint32:
		return t.SetFieldUint32(tag, v)
	case uint64:
		return t.SetFieldUint64(tag, v)
	case int8:
		return t.SetFieldInt8(tag, v)
	case int16:
		return t.SetFieldInt16(tag, v)
	case int32:
		return t.SetFieldInt32(tag, v)
	case int64:
		return t.SetFieldInt64(tag, v)
	case float64:
		if sz == 4 {
			return t.SetFieldFloat(tag, v)
		}
		return t.SetFieldDouble(tag, v)
	case string:
		return t.SetFieldString(tag, v)
	case []byte:
		if passCount {
			return t.SetFieldByteSlice(tag, v)
		}
		return t.SetFieldC0ByteSlice(tag, v)
	case []uint16:
		if passCount {
			return t.SetFieldUint16Slice(tag, v)
		}
		return t.SetFieldC0Uint16Slice(tag, v)
	case []uint32:
		if passCount {
			return t.SetFieldUint32Slice(tag, v)
		}
		return t.SetFieldC0Uint32Slice(tag, v)
	case []float64:
		if passCount {
			if sz == 4 {
				return t.SetFieldFloatSlice(tag, v)
			}
			return t.SetFieldDoubleSlice(tag, v)
		}
		if sz == 4 {
			return t.SetFieldC0FloatSlice(tag, v)
		}
		return t.SetFieldC0DoubleSlice(tag, v)
	default:
		return fmt.Errorf("libtiff: unsupported value type %T for tag %d", value, uint32(tag))
	}
}

// GetFieldAny reads a tag's value, dispatching to the correct typed getter
// based on the tag's registered DataType and FieldPassCount.
//
// Mapping:
//
//	DataTypeShort, DataTypeLong, DataTypeLong8, DataTypeIFD, DataTypeIFD8  → uint16/uint32/uint64
//	DataTypeByte, DataTypeUndefined                                        → uint8 or []byte (passCount)
//	DataTypeSByte                                                         → int8 or []byte (passCount)
//	DataTypeSShort, DataTypeSLong, DataTypeSLong8                          → int16/int32/int64
//	DataTypeRational, DataTypeSRational, DataTypeFloat                     → float64
//	DataTypeDouble                                                         → float64 (via GetFieldDouble)
//	DataTypeASCII                                                          → string
func (t *TIFF) GetFieldAny(tag Tag) (any, error) {
	dt := t.GetFieldType(tag)
	if dt < 0 {
		return nil, &FieldError{Tag: tag, Op: "GetFieldAny", Msg: "unknown tag type"}
	}
	passCount := t.FieldPassCount(tag)

	switch dt {
	case DataTypeShort:
		return t.GetFieldUint16(tag)
	case DataTypeLong, DataTypeIFD:
		return t.GetFieldUint32(tag)
	case DataTypeLong8, DataTypeIFD8:
		return t.GetFieldUint64(tag)
	case DataTypeRational, DataTypeSRational, DataTypeFloat:
		return t.GetFieldFloat(tag)
	case DataTypeDouble:
		return t.GetFieldDouble(tag)
	case DataTypeASCII:
		return t.GetFieldString(tag)
	case DataTypeByte, DataTypeUndefined:
		if passCount {
			return t.GetFieldByteSlice(tag)
		}
		return t.GetFieldUint8(tag)
	case DataTypeSByte:
		if passCount {
			return t.GetFieldByteSlice(tag)
		}
		return t.GetFieldInt8(tag)
	case DataTypeSShort:
		return t.GetFieldInt16(tag)
	case DataTypeSLong:
		return t.GetFieldInt32(tag)
	case DataTypeSLong8:
		return t.GetFieldInt64(tag)
	default:
		return nil, &FieldError{Tag: tag, Op: "GetFieldAny", Msg: fmt.Sprintf("unsupported data type %d", dt)}
	}
}
