package golibtiff

/*
#include <tiffio.h>
#include <stdlib.h>
#include "libtiff_bridge.h"

// Forward-declare internal libtiff functions not exposed in tiffio.h.
// Note: _TIFFGetFields/_TIFFGetExifFields/_TIFFGetGpsFields are internal
// symbols not exported by vcpkg's static libtiff. Use public APIs instead:
//   TIFFCreateDirectory instead of TIFFCreateCustomDirectory(tif, _TIFFGetFields())
//   TIFFReadEXIFDirectory instead of TIFFReadCustomDirectory(tif, off, _TIFFGetExifFields())
//   TIFFReadGPSDirectory instead of TIFFReadCustomDirectory(tif, off, _TIFFGetGpsFields())

// Typed getters (avoid variadic TIFFGetField from Go).
static int tiffGetFieldU16(TIFF *t, uint32_t tag, uint16_t *v) { return TIFFGetField(t, tag, v); }
static int tiffGetFieldU32(TIFF *t, uint32_t tag, uint32_t *v) { return TIFFGetField(t, tag, v); }
static int tiffGetFieldFloat(TIFF *t, uint32_t tag, float *v) { return TIFFGetField(t, tag, v); }
static int tiffGetFieldDouble(TIFF *t, uint32_t tag, double *v) { return TIFFGetField(t, tag, v); }
static int tiffGetFieldString(TIFF *t, uint32_t tag, const char **v) { return TIFFGetField(t, tag, v); }
static int tiffGetFieldU16Array(TIFF *t, uint32_t tag, uint16_t **v, uint16_t *c) { return TIFFGetField(t, tag, c, v); }
static int tiffGetFieldU32Array(TIFF *t, uint32_t tag, uint32_t **v, uint32_t *c) { return TIFFGetField(t, tag, c, v); }
static int tiffGetFieldU8(TIFF *t, uint32_t tag, uint8_t *v) { return TIFFGetField(t, tag, v); }
static int tiffGetFieldU64(TIFF *t, uint32_t tag, uint64_t *v) { return TIFFGetField(t, tag, v); }
static int tiffReadEXIFDirectory(TIFF *t, uint64_t off) { return TIFFReadEXIFDirectory(t, (toff_t)off); }

// Typed setters (avoid variadic TIFFSetField from Go).
static int tiffSetFieldU16(TIFF *t, uint32_t tag, uint16_t v) { return TIFFSetField(t, tag, v); }
static int tiffSetFieldU32(TIFF *t, uint32_t tag, uint32_t v) { return TIFFSetField(t, tag, v); }
static int tiffSetFieldFloat(TIFF *t, uint32_t tag, float v) { return TIFFSetField(t, tag, v); }
static int tiffSetFieldString(TIFF *t, uint32_t tag, const char *v) { return TIFFSetField(t, tag, v); }
static int tiffSetFieldU16Array(TIFF *t, uint32_t tag, uint16_t c, uint16_t *v) { return TIFFSetField(t, tag, c, v); }
static int tiffSetFieldU32Array(TIFF *t, uint32_t tag, uint32_t c, uint32_t *v) { return TIFFSetField(t, tag, c, v); }

static int tiffReadRGBAImage(TIFF *t, uint32_t w, uint32_t h, uint32_t *buf) {
    return TIFFReadRGBAImage(t, w, h, buf, 0);
}

static tmsize_t tiffReadEncodedTile(TIFF *t, uint32_t tile, void *buf, tmsize_t size) {
    return TIFFReadEncodedTile(t, tile, buf, size);
}
static tmsize_t tiffWriteEncodedTile(TIFF *t, uint32_t tile, void *buf, tmsize_t size) {
    return TIFFWriteEncodedTile(t, tile, buf, size);
}

// Byte-slice setter (count + data for UNDEFINED/BYTE arrays like ICC, XMP, MakerNotes).
static int tiffSetFieldByteSlice(TIFF *t, uint32_t tag, uint32_t c, uint8_t *v) {
    return TIFFSetField(t, tag, c, v);
}
// C0 byte-slice setter (no count arg, for fixed-count BYTE arrays like DNGVersion).
static int tiffSetFieldC0ByteSlice(TIFF *t, uint32_t tag, uint8_t *v) {
    return TIFFSetField(t, tag, v);
}
// C0 uint16-slice setter (no count arg, for fixed-count SHORT arrays like SubjectLocation).
static int tiffSetFieldC0U16(TIFF *t, uint32_t tag, uint16_t *v) {
    return TIFFSetField(t, tag, v);
}
// C0 uint32-slice setter (no count arg, for fixed-count LONG arrays).
static int tiffSetFieldC0U32(TIFF *t, uint32_t tag, uint32_t *v) {
    return TIFFSetField(t, tag, v);
}
// EXIF Sub-IFD creation and writing.
static int tiffCreateEXIFDirectory(TIFF *t) {
    return TIFFCreateEXIFDirectory(t);
}
static int tiffWriteCustomDirectory(TIFF *t, uint64_t *offset) {
    return TIFFWriteCustomDirectory(t, offset);
}
// Float-array setter (count + float* for RATIONAL array tags).
static int tiffSetFieldFloatSlice(TIFF *t, uint32_t tag, int c, float *v) {
    return TIFFSetField(t, tag, c, v);
}
// uint64 setter (for EXIFIFD pointer tag).
static int tiffSetFieldU64(TIFF *t, uint32_t tag, uint64_t v) {
    return TIFFSetField(t, tag, v);
}
// CheckpointDirectory writes state to disk without closing the IFD.
static int tiffCheckpointDirectory(TIFF *t) {
    return TIFFCheckpointDirectory(t);
}
// Single-byte setter (SceneType: SETGET_UINT8).
static int tiffSetFieldU8(TIFF *t, uint32_t tag, uint8_t v) {
    return TIFFSetField(t, tag, v);
}
// C0 float-array setter (LensSpecification: SETGET_C0_FLOAT, fixed 4 floats, no count arg).
static int tiffSetFieldC0Float(TIFF *t, uint32_t tag, float *v) {
    return TIFFSetField(t, tag, v);
}
// Double setters (64-bit, no precision loss for RATIONAL values).
static int tiffSetFieldDouble(TIFF *t, uint32_t tag, double v) {
    return TIFFSetField(t, tag, v);
}
static int tiffSetFieldDoubleSlice(TIFF *t, uint32_t tag, int c, double *v) {
    return TIFFSetField(t, tag, c, v);
}
static int tiffSetFieldC0Double(TIFF *t, uint32_t tag, double *v) {
    return TIFFSetField(t, tag, v);
}
// GPS Sub-IFD creation.
static int tiffCreateGPSDirectory(TIFF *t) {
    return TIFFCreateGPSDirectory(t);
}
// Check if a tag is registered in libtiff's field definitions.
static int tiffIsFieldKnown(TIFF *t, uint32_t tag) {
    return TIFFFieldWithTag(t, tag) != NULL;
}
// Get the registered TIFFDataType for a tag.
static int tiffGetFieldType(TIFF *t, uint32_t tag) {
    const TIFFField *f = TIFFFieldWithTag(t, tag);
    return f ? (int)TIFFFieldDataType(f) : -1;
}
// Check whether a tag requires a count argument in TIFFSetField.
static int tiffFieldPassCount(TIFF *t, uint32_t tag) {
    const TIFFField *f = TIFFFieldWithTag(t, tag);
    return f ? (int)TIFFFieldPassCount(f) : -1;
}
// Get the write count for a tag.
static int tiffFieldWriteCount(TIFF *t, uint32_t tag) {
    const TIFFField *f = TIFFFieldWithTag(t, tag);
    return f ? (int)TIFFFieldWriteCount(f) : 0;
}
// Get the internal storage size for a tag's values (4=float, 8=double).
static int tiffFieldSetGetSize(TIFF *t, uint32_t tag) {
    const TIFFField *f = TIFFFieldWithTag(t, tag);
    return f ? TIFFFieldSetGetSize(f) : -1;
}
// Byte-slice getter (count + data for UNDEFINED/BYTE arrays like ICC, XMP).
static int tiffGetFieldByteSlice(TIFF *t, uint32_t tag, uint8_t **v, uint32_t *c) {
    return TIFFGetField(t, tag, c, v);
}
// UnsetField removes a tag from the current IFD.
static int tiffUnsetField(TIFF *t, uint32_t tag) { return TIFFUnsetField(t, tag); }
// RGBA strip/tile readers.
static int tiffReadRGBAStrip(TIFF *t, uint32_t strip, uint32_t *buf) {
    return TIFFReadRGBAStrip(t, strip, buf);
}
static int tiffReadRGBATile(TIFF *t, uint32_t tile, uint32_t *buf) {
    return TIFFReadRGBATile(t, tile, 0, buf);
}
// Directory/IFD operations.
static int tiffReadGPSDirectory(TIFF *t, uint64_t off) {
    return TIFFReadGPSDirectory(t, (toff_t)off);
}
static int tiffCreateDirectory(TIFF *t) { return TIFFCreateDirectory(t); }
static int tiffRewriteDirectory(TIFF *t) { return TIFFRewriteDirectory(t); }
static int tiffUnlinkDirectory(TIFF *t, uint16_t d) {
    return TIFFUnlinkDirectory(t, (tdir_t)d);
}
// GetFieldDefaulted typed getters.
static int tiffGetFieldDefaultedU16(TIFF *t, uint32_t tag, uint16_t *v) {
    return TIFFGetFieldDefaulted(t, tag, v);
}
static int tiffGetFieldDefaultedU32(TIFF *t, uint32_t tag, uint32_t *v) {
    return TIFFGetFieldDefaulted(t, tag, v);
}
static int tiffGetFieldDefaultedFloat(TIFF *t, uint32_t tag, float *v) {
    return TIFFGetFieldDefaulted(t, tag, v);
}
static int tiffGetFieldDefaultedString(TIFF *t, uint32_t tag, const char **v) {
    return TIFFGetFieldDefaulted(t, tag, v);
}
// Strile low-level access.
static uint64_t tiffGetStrileOffset(TIFF *t, uint32_t s) {
    return TIFFGetStrileOffset(t, s);
}
static uint64_t tiffGetStrileByteCount(TIFF *t, uint32_t s) {
    return TIFFGetStrileByteCount(t, s);
}
static uint64_t tiffGetStrileOffsetWithErr(TIFF *t, uint32_t s, int *e) {
    return TIFFGetStrileOffsetWithErr(t, s, e);
}
static uint64_t tiffGetStrileByteCountWithErr(TIFF *t, uint32_t s, int *e) {
    return TIFFGetStrileByteCountWithErr(t, s, e);
}
// ReadFromUserBuffer: decompress user-provided compressed data.
static int tiffReadFromUserBuffer(TIFF *t, uint32_t strile, void *in, tmsize_t insz, void *out, tmsize_t outsz) {
    return TIFFReadFromUserBuffer(t, strile, in, insz, out, outsz);
}
// RGBA extended interfaces.
static int tiffReadRGBAImageOriented(TIFF *t, uint32_t w, uint32_t h, uint32_t *buf, int orient, int stop) {
    return TIFFReadRGBAImageOriented(t, w, h, buf, orient, stop);
}
static int tiffReadRGBAStripExt(TIFF *t, uint32_t strip, uint32_t *buf, int stop) {
    return TIFFReadRGBAStripExt(t, strip, buf, stop);
}
static int tiffReadRGBATileExt(TIFF *t, uint32_t tw, uint32_t th, uint32_t *buf, int stop) {
    return TIFFReadRGBATileExt(t, tw, th, buf, stop);
}
static uint64_t tiffCurrentDirOffset(TIFF *t) { return TIFFCurrentDirOffset(t); }
static void tiffDefaultTileSize(TIFF *t, uint32_t *tw, uint32_t *th) { TIFFDefaultTileSize(t, tw, th); }
*/
import "C"

import (
	"errors"
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
		return nil, &OpenError{Path: path, Mode: mode, Msg: err.Error()}
	}
	if tif == nil {
		if C.hasOpenPhaseError() != 0 {
			return nil, &OpenError{Path: path, Mode: mode, Msg: C.GoString(C.getOpenPhaseError())}
		}
		return nil, &OpenError{Path: path, Mode: mode}
	}

	C.attachErrorState(tif)

	t := &TIFF{tif: tif}
	runtime.SetFinalizer(t, (*TIFF).Close)
	return t, nil
}

// Close releases the TIFF file handle resources. Safe to call multiple times.
func (t *TIFF) Close() error {
	t.closeOnce.Do(func() {
		if t.tif != nil {
			C.detachErrorState(t.tif)
			C.TIFFClose(t.tif)
			t.tif = nil
			runtime.SetFinalizer(t, nil)
		}
	})
	return nil
}

// FileName returns the file name associated with the TIFF handle.
func (t *TIFF) FileName() string {
	if err := t.checkOpen(); err != nil {
		return ""
	}
	return C.GoString(C.TIFFFileName(t.tif))
}

func (t *TIFF) checkOpen() error {
	if t.tif == nil {
		return errors.New("libtiff: handle is closed")
	}
	return nil
}

func (t *TIFF) lastError() error {
	if C.hasHandleError(t.tif) != 0 {
		return fmt.Errorf("%s", C.GoString(C.getHandleError(t.tif)))
	}
	return nil
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

// --- Info Queries (direct C API calls) ---

func (t *TIFF) IsTiled() bool {
	if err := t.checkOpen(); err != nil {
		return false
	}
	return C.TIFFIsTiled(t.tif) != 0
}

func (t *TIFF) ScanlineSize() int64 {
	if err := t.checkOpen(); err != nil {
		return 0
	}
	return int64(C.TIFFScanlineSize(t.tif))
}

func (t *TIFF) StripSize() int64 {
	if err := t.checkOpen(); err != nil {
		return 0
	}
	return int64(C.TIFFStripSize(t.tif))
}

// DefaultStripSize returns the default RowsPerStrip value (8192 / scanline_size).
func (t *TIFF) DefaultStripSize() uint32 {
	if err := t.checkOpen(); err != nil {
		return 0
	}
	return uint32(C.TIFFDefaultStripSize(t.tif, 0))
}

func (t *TIFF) TileSize() int64 {
	if err := t.checkOpen(); err != nil {
		return 0
	}
	return int64(C.TIFFTileSize(t.tif))
}

func (t *TIFF) NumberOfStrips() uint32 {
	if err := t.checkOpen(); err != nil {
		return 0
	}
	return uint32(C.TIFFNumberOfStrips(t.tif))
}

func (t *TIFF) NumberOfTiles() uint32 {
	if err := t.checkOpen(); err != nil {
		return 0
	}
	return uint32(C.TIFFNumberOfTiles(t.tif))
}

func (t *TIFF) IsBigTIFF() bool {
	if err := t.checkOpen(); err != nil {
		return false
	}
	return C.TIFFIsBigTIFF(t.tif) != 0
}

func (t *TIFF) IsByteSwapped() bool {
	if err := t.checkOpen(); err != nil {
		return false
	}
	return C.TIFFIsByteSwapped(t.tif) != 0
}

// --- Read Operations ---

func (t *TIFF) ReadScanline(buf []byte, row uint32) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	if len(buf) == 0 {
		return errors.New("libtiff: empty buffer for ReadScanline")
	}
	C.clearHandleError(t.tif)
	if C.TIFFReadScanline(t.tif, unsafe.Pointer(&buf[0]), C.uint32_t(row), 0) < 0 {
		runtime.KeepAlive(buf)
		if err := t.lastError(); err != nil {
			return &ReadError{Op: "scanline", Msg: err.Error()}
		}
		return &ReadError{Op: "scanline", Msg: fmt.Sprintf("row %d", row)}
	}
	runtime.KeepAlive(buf)
	return nil
}

// ReadEncodedStrip reads decoded strip data into buf. Returns bytes read.
// If size <= 0, reads StripSize() bytes.
func (t *TIFF) ReadEncodedStrip(strip uint32, buf []byte, size int64) (int, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	if len(buf) == 0 {
		return 0, errors.New("libtiff: empty buffer for ReadEncodedStrip")
	}
	C.clearHandleError(t.tif)
	cSize := C.tmsize_t(size)
	if cSize <= 0 {
		cSize = C.tmsize_t(len(buf))
	}
	n := C.TIFFReadEncodedStrip(t.tif, C.uint32_t(strip), unsafe.Pointer(&buf[0]), cSize)
	runtime.KeepAlive(buf)
	if n < 0 {
		if err := t.lastError(); err != nil {
			return 0, &ReadError{Op: "encoded_strip", Msg: err.Error()}
		}
		return 0, &ReadError{Op: "encoded_strip", Msg: fmt.Sprintf("strip %d", strip)}
	}
	return int(n), nil
}

func (t *TIFF) ReadRawStrip(strip uint32, buf []byte, size int64) (int, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	if len(buf) == 0 {
		return 0, errors.New("libtiff: empty buffer for ReadRawStrip")
	}
	C.clearHandleError(t.tif)
	n := C.TIFFReadRawStrip(t.tif, C.uint32_t(strip), unsafe.Pointer(&buf[0]), C.tmsize_t(size))
	runtime.KeepAlive(buf)
	if n < 0 {
		if err := t.lastError(); err != nil {
			return 0, &ReadError{Op: "raw_strip", Msg: err.Error()}
		}
		return 0, &ReadError{Op: "raw_strip", Msg: fmt.Sprintf("strip %d", strip)}
	}
	return int(n), nil
}

// ReadRGBAImage reads the entire image as RGBA into buf (width*height uint32 values).
func (t *TIFF) ReadRGBAImage(buf []uint32) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	w, err := t.Width()
	if err != nil {
		return &ReadError{Op: "rgba_image", Msg: err.Error()}
	}
	h, err := t.Height()
	if err != nil {
		return &ReadError{Op: "rgba_image", Msg: err.Error()}
	}
	if w == 0 || h == 0 {
		return &ReadError{Op: "rgba_image", Msg: "invalid dimensions"}
	}
	required := int(w) * int(h)
	if len(buf) < required {
		return &ReadError{Op: "rgba_image", Msg: fmt.Sprintf("buffer too small: need %d pixels, got %d", required, len(buf))}
	}
	C.clearHandleError(t.tif)
	if C.tiffReadRGBAImage(t.tif, C.uint32_t(w), C.uint32_t(h), (*C.uint32_t)(unsafe.Pointer(&buf[0]))) == 0 {
		runtime.KeepAlive(buf)
		if err := t.lastError(); err != nil {
			return &ReadError{Op: "rgba_image", Msg: err.Error()}
		}
		return &ReadError{Op: "rgba_image", Msg: "failed"}
	}
	runtime.KeepAlive(buf)
	return nil
}

func (t *TIFF) RGBAImageOK() error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	if C.TIFFRGBAImageOK(t.tif, nil) == 0 {
		return &ReadError{Op: "rgba_ok", Msg: "image cannot be read as RGBA"}
	}
	return nil
}

// --- Write Operations ---

func (t *TIFF) WriteScanline(buf []byte, row uint32) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	if len(buf) == 0 {
		return errors.New("libtiff: empty buffer for WriteScanline")
	}
	C.clearHandleError(t.tif)
	if C.TIFFWriteScanline(t.tif, unsafe.Pointer(&buf[0]), C.uint32_t(row), 0) < 0 {
		runtime.KeepAlive(buf)
		if err := t.lastError(); err != nil {
			return &WriteError{Op: "scanline", Msg: err.Error()}
		}
		return &WriteError{Op: "scanline", Msg: fmt.Sprintf("row %d", row)}
	}
	runtime.KeepAlive(buf)
	return nil
}

// WriteEncodedStrip writes decoded data to a strip. Returns bytes written.
func (t *TIFF) WriteEncodedStrip(strip uint32, data []byte) (int, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	if len(data) == 0 {
		return 0, errors.New("libtiff: empty data for WriteEncodedStrip")
	}
	C.clearHandleError(t.tif)
	n := C.TIFFWriteEncodedStrip(t.tif, C.uint32_t(strip), unsafe.Pointer(&data[0]), C.tmsize_t(len(data)))
	runtime.KeepAlive(data)
	if n < 0 {
		if err := t.lastError(); err != nil {
			return 0, &WriteError{Op: "encoded_strip", Msg: err.Error()}
		}
		return 0, &WriteError{Op: "encoded_strip", Msg: fmt.Sprintf("strip %d", strip)}
	}
	return int(n), nil
}

func (t *TIFF) WriteRawStrip(strip uint32, data []byte) (int, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	if len(data) == 0 {
		return 0, errors.New("libtiff: empty data for WriteRawStrip")
	}
	C.clearHandleError(t.tif)
	n := C.TIFFWriteRawStrip(t.tif, C.uint32_t(strip), unsafe.Pointer(&data[0]), C.tmsize_t(len(data)))
	runtime.KeepAlive(data)
	if n < 0 {
		if err := t.lastError(); err != nil {
			return 0, &WriteError{Op: "raw_strip", Msg: err.Error()}
		}
		return 0, &WriteError{Op: "raw_strip", Msg: fmt.Sprintf("strip %d", strip)}
	}
	return int(n), nil
}

// --- Tile Operations ---

// ReadEncodedTile reads decoded tile data into buf. Returns bytes read.
// If size <= 0, reads TileSize() bytes.
func (t *TIFF) ReadEncodedTile(tile uint32, buf []byte, size int64) (int, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	if len(buf) == 0 {
		return 0, errors.New("libtiff: empty buffer for ReadEncodedTile")
	}
	C.clearHandleError(t.tif)
	cSize := C.tmsize_t(size)
	if cSize <= 0 {
		cSize = C.tmsize_t(len(buf))
	}
	n := C.tiffReadEncodedTile(t.tif, C.uint32_t(tile), unsafe.Pointer(&buf[0]), cSize)
	runtime.KeepAlive(buf)
	if n < 0 {
		if err := t.lastError(); err != nil {
			return 0, &ReadError{Op: "encoded_tile", Msg: err.Error()}
		}
		return 0, &ReadError{Op: "encoded_tile", Msg: fmt.Sprintf("tile %d", tile)}
	}
	return int(n), nil
}

// WriteEncodedTile writes decoded data to a tile. Returns bytes written.
func (t *TIFF) WriteEncodedTile(tile uint32, data []byte) (int, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	if len(data) == 0 {
		return 0, errors.New("libtiff: empty data for WriteEncodedTile")
	}
	C.clearHandleError(t.tif)
	n := C.tiffWriteEncodedTile(t.tif, C.uint32_t(tile), unsafe.Pointer(&data[0]), C.tmsize_t(len(data)))
	runtime.KeepAlive(data)
	if n < 0 {
		if err := t.lastError(); err != nil {
			return 0, &WriteError{Op: "encoded_tile", Msg: err.Error()}
		}
		return 0, &WriteError{Op: "encoded_tile", Msg: fmt.Sprintf("tile %d", tile)}
	}
	return int(n), nil
}

func (t *TIFF) Flush() error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	if C.TIFFFlush(t.tif) == 0 {
		if err := t.lastError(); err != nil {
			return &WriteError{Op: "flush", Msg: err.Error()}
		}
		return &WriteError{Op: "flush", Msg: "failed"}
	}
	return nil
}

// --- Directory Operations ---

func (t *TIFF) NumberOfDirectories() uint32 {
	if err := t.checkOpen(); err != nil {
		return 0
	}
	return uint32(C.TIFFNumberOfDirectories(t.tif))
}

func (t *TIFF) CurrentDirectory() uint32 {
	if err := t.checkOpen(); err != nil {
		return 0
	}
	return uint32(C.TIFFCurrentDirectory(t.tif))
}

func (t *TIFF) SetDirectory(dirnum uint32) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.TIFFSetDirectory(t.tif, C.tdir_t(dirnum)) == 0 {
		if err := t.lastError(); err != nil {
			return err
		}
		return errors.New("libtiff: failed to set directory")
	}
	return nil
}

// ReadDirectory reads the next directory. Returns true if a directory was read.
func (t *TIFF) ReadDirectory() bool {
	if err := t.checkOpen(); err != nil {
		return false
	}
	return C.TIFFReadDirectory(t.tif) != 0
}

func (t *TIFF) WriteDirectory() error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.TIFFWriteDirectory(t.tif) == 0 {
		if err := t.lastError(); err != nil {
			return err
		}
		return errors.New("libtiff: failed to write directory")
	}
	return nil
}

// CheckpointDirectory writes the current IFD state to disk without closing it.
// This is needed before creating EXIF sub-IFDs.
func (t *TIFF) CheckpointDirectory() error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffCheckpointDirectory(t.tif) == 0 {
		if err := t.lastError(); err != nil {
			return err
		}
		return errors.New("libtiff: failed to checkpoint directory")
	}
	return nil
}

func (t *TIFF) SetSubDirectory(offset uint64) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.TIFFSetSubDirectory(t.tif, C.uint64_t(offset)) == 0 {
		if err := t.lastError(); err != nil {
			return err
		}
		return errors.New("libtiff: failed to set subdirectory")
	}
	return nil
}

// ReadEXIFDirectory reads the EXIF Sub-IFD at the given offset.
// Unlike SetSubDirectory, this does not require ImageLength/ImageWidth.
func (t *TIFF) ReadEXIFDirectory(offset uint64) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffReadEXIFDirectory(t.tif, C.uint64_t(offset)) == 0 {
		if err := t.lastError(); err != nil {
			return err
		}
		return errors.New("libtiff: failed to read EXIF directory")
	}
	return nil
}

func (t *TIFF) LastDirectory() bool {
	if err := t.checkOpen(); err != nil {
		return false
	}
	return C.TIFFLastDirectory(t.tif) != 0
}

// --- EXIF Sub-IFD Operations ---

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

// CreateEXIFDirectory creates a new EXIF Sub-IFD.
func (t *TIFF) CreateEXIFDirectory() error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffCreateEXIFDirectory(t.tif) != 0 {
		if err := t.lastError(); err != nil {
			return fmt.Errorf("libtiff: CreateEXIFDirectory: %w", err)
		}
		return errors.New("libtiff: CreateEXIFDirectory failed")
	}
	return nil
}

// WriteCustomDirectory writes the current directory as a custom (unlinked) IFD
// and returns its byte offset. Used for writing EXIF Sub-IFDs.
func (t *TIFF) WriteCustomDirectory() (uint64, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	C.clearHandleError(t.tif)
	var offset C.uint64_t
	if C.tiffWriteCustomDirectory(t.tif, &offset) == 0 {
		if err := t.lastError(); err != nil {
			return 0, fmt.Errorf("libtiff: WriteCustomDirectory: %w", err)
		}
		return 0, errors.New("libtiff: WriteCustomDirectory failed")
	}
	return uint64(offset), nil
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

// CreateGPSDirectory creates a new GPS Sub-IFD.
func (t *TIFF) CreateGPSDirectory() error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffCreateGPSDirectory(t.tif) != 0 {
		if err := t.lastError(); err != nil {
			return fmt.Errorf("libtiff: CreateGPSDirectory: %w", err)
		}
		return errors.New("libtiff: CreateGPSDirectory failed")
	}
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

// ReadRGBAStrip reads a single strip as RGBA into buf.
// The buffer must be large enough to hold stripWidth * imageHeight uint32 values.
func (t *TIFF) ReadRGBAStrip(strip uint32, buf []uint32) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	if len(buf) == 0 {
		return errors.New("libtiff: empty buffer for ReadRGBAStrip")
	}
	C.clearHandleError(t.tif)
	if C.tiffReadRGBAStrip(t.tif, C.uint32_t(strip), (*C.uint32_t)(unsafe.Pointer(&buf[0]))) == 0 {
		runtime.KeepAlive(buf)
		if err := t.lastError(); err != nil {
			return &ReadError{Op: "rgba_strip", Msg: err.Error()}
		}
		return &ReadError{Op: "rgba_strip", Msg: fmt.Sprintf("strip %d", strip)}
	}
	runtime.KeepAlive(buf)
	return nil
}

// ReadRGBATile reads a single tile as RGBA into buf.
// The buffer must be large enough to hold tileWidth * tileHeight uint32 values.
func (t *TIFF) ReadRGBATile(tile uint32, buf []uint32) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	if len(buf) == 0 {
		return errors.New("libtiff: empty buffer for ReadRGBATile")
	}
	C.clearHandleError(t.tif)
	if C.tiffReadRGBATile(t.tif, C.uint32_t(tile), (*C.uint32_t)(unsafe.Pointer(&buf[0]))) == 0 {
		runtime.KeepAlive(buf)
		if err := t.lastError(); err != nil {
			return &ReadError{Op: "rgba_tile", Msg: err.Error()}
		}
		return &ReadError{Op: "rgba_tile", Msg: fmt.Sprintf("tile %d", tile)}
	}
	runtime.KeepAlive(buf)
	return nil
}

// IsFieldKnown checks if a tag is registered in libtiff's field definitions.
func (t *TIFF) IsFieldKnown(tag Tag) bool {
	if err := t.checkOpen(); err != nil {
		return false
	}
	return C.tiffIsFieldKnown(t.tif, C.uint32_t(tag)) != 0
}

// GetFieldType returns the libtiff-registered TIFFDataType for a tag.
// Returns -1 if the tag is not registered.
func (t *TIFF) GetFieldType(tag Tag) int {
	if err := t.checkOpen(); err != nil {
		return -1
	}
	return int(C.tiffGetFieldType(t.tif, C.uint32_t(tag)))
}

// FieldPassCount reports whether a tag requires a count argument in TIFFSetField.
func (t *TIFF) FieldPassCount(tag Tag) bool {
	if err := t.checkOpen(); err != nil {
		return false
	}
	return C.tiffFieldPassCount(t.tif, C.uint32_t(tag)) != 0
}

// FieldWriteCount returns the number of values a tag expects.
func (t *TIFF) FieldWriteCount(tag Tag) int {
	if err := t.checkOpen(); err != nil {
		return 0
	}
	return int(C.tiffFieldWriteCount(t.tif, C.uint32_t(tag)))
}

// FieldSetGetSize returns the per-element storage size in bytes for the tag.
// Returns 4 for SETGET_*_FLOAT tags, 8 for SETGET_*_DOUBLE tags, -1 if unknown.
func (t *TIFF) FieldSetGetSize(tag Tag) int {
	if err := t.checkOpen(); err != nil {
		return -1
	}
	return int(C.tiffFieldSetGetSize(t.tif, C.uint32_t(tag)))
}

// --- Directory/IFD operations ---

// ReadGPSDirectory reads the GPS Sub-IFD at the given byte offset.
func (t *TIFF) ReadGPSDirectory(offset uint64) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffReadGPSDirectory(t.tif, C.uint64_t(offset)) == 0 {
		if err := t.lastError(); err != nil {
			return err
		}
		return errors.New("libtiff: failed to read GPS directory")
	}
	return nil
}

// CreateDirectory creates a new blank IFD and switches to it.
func (t *TIFF) CreateDirectory() error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffCreateDirectory(t.tif) != 0 {
		if err := t.lastError(); err != nil {
			return err
		}
		return errors.New("libtiff: failed to create directory")
	}
	return nil
}

// FreeDirectory releases internal data associated with the current IFD.
func (t *TIFF) FreeDirectory() {
	if err := t.checkOpen(); err != nil {
		return
	}
	C.TIFFFreeDirectory(t.tif)
}

// RewriteDirectory rewrites the directory at the end of the file.
// Useful for updating an existing TIFF in-place.
func (t *TIFF) RewriteDirectory() error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffRewriteDirectory(t.tif) == 0 {
		if err := t.lastError(); err != nil {
			return err
		}
		return errors.New("libtiff: failed to rewrite directory")
	}
	return nil
}

// UnlinkDirectory removes the IFD at the given index from the directory chain.
func (t *TIFF) UnlinkDirectory(dirNum uint16) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffUnlinkDirectory(t.tif, C.uint16_t(dirNum)) == 0 {
		if err := t.lastError(); err != nil {
			return err
		}
		return errors.New("libtiff: failed to unlink directory")
	}
	return nil
}

// --- Tag enumeration ---

// TagListCount returns the number of tags defined in the current IFD.
func (t *TIFF) TagListCount() int {
	if err := t.checkOpen(); err != nil {
		return 0
	}
	return int(C.TIFFGetTagListCount(t.tif))
}

// TagListEntry returns the tag number at the given index.
func (t *TIFF) TagListEntry(index int) Tag {
	if err := t.checkOpen(); err != nil {
		return 0
	}
	return Tag(C.TIFFGetTagListEntry(t.tif, C.int(index)))
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

// --- Strile low-level access ---

// StrileOffset returns the byte offset of the given strip or tile.
func (t *TIFF) StrileOffset(strile uint32) uint64 {
	if err := t.checkOpen(); err != nil {
		return 0
	}
	return uint64(C.tiffGetStrileOffset(t.tif, C.uint32_t(strile)))
}

// StrileByteCount returns the byte count of the given strip or tile.
func (t *TIFF) StrileByteCount(strile uint32) uint64 {
	if err := t.checkOpen(); err != nil {
		return 0
	}
	return uint64(C.tiffGetStrileByteCount(t.tif, C.uint32_t(strile)))
}

// StrileOffsetWithErr returns the byte offset of the given strip or tile, with error reporting.
func (t *TIFF) StrileOffsetWithErr(strile uint32) (uint64, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	var cerr C.int
	offset := uint64(C.tiffGetStrileOffsetWithErr(t.tif, C.uint32_t(strile), &cerr))
	if cerr != 0 {
		if err := t.lastError(); err != nil {
			return 0, err
		}
		return 0, errors.New("libtiff: strile offset error")
	}
	return offset, nil
}

// StrileByteCountWithErr returns the byte count of the given strip or tile, with error reporting.
func (t *TIFF) StrileByteCountWithErr(strile uint32) (uint64, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	var cerr C.int
	count := uint64(C.tiffGetStrileByteCountWithErr(t.tif, C.uint32_t(strile), &cerr))
	if cerr != 0 {
		if err := t.lastError(); err != nil {
			return 0, err
		}
		return 0, errors.New("libtiff: strile byte count error")
	}
	return count, nil
}

// --- Raw Tile I/O ---

// ReadRawTile reads raw (compressed) tile data into buf.
func (t *TIFF) ReadRawTile(tile uint32, buf []byte) (int, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	if len(buf) == 0 {
		return 0, errors.New("libtiff: empty buffer for ReadRawTile")
	}
	C.clearHandleError(t.tif)
	n := C.TIFFReadRawTile(t.tif, C.uint32_t(tile), unsafe.Pointer(&buf[0]), C.tmsize_t(len(buf)))
	runtime.KeepAlive(buf)
	if n < 0 {
		if err := t.lastError(); err != nil {
			return 0, &ReadError{Op: "read_raw_tile", Msg: err.Error()}
		}
		return 0, &ReadError{Op: "read_raw_tile", Msg: fmt.Sprintf("tile %d", tile)}
	}
	return int(n), nil
}

// WriteRawTile writes raw (compressed) tile data.
func (t *TIFF) WriteRawTile(tile uint32, data []byte) (int, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	if len(data) == 0 {
		return 0, errors.New("libtiff: empty data for WriteRawTile")
	}
	C.clearHandleError(t.tif)
	n := C.TIFFWriteRawTile(t.tif, C.uint32_t(tile), unsafe.Pointer(&data[0]), C.tmsize_t(len(data)))
	runtime.KeepAlive(data)
	if n < 0 {
		if err := t.lastError(); err != nil {
			return 0, &WriteError{Op: "write_raw_tile", Msg: err.Error()}
		}
		return 0, &WriteError{Op: "write_raw_tile", Msg: fmt.Sprintf("tile %d", tile)}
	}
	return int(n), nil
}

// TileRowSize returns the number of bytes in a decoded row of a tile.
func (t *TIFF) TileRowSize() int64 {
	if err := t.checkOpen(); err != nil {
		return 0
	}
	return int64(C.TIFFTileRowSize(t.tif))
}

// --- ReadFromUserBuffer ---

// ReadFromUserBuffer decompresses user-provided compressed data (inbuf) into outbuf.
// The strile parameter identifies the strip or tile index.
func (t *TIFF) ReadFromUserBuffer(strile uint32, inbuf, outbuf []byte) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	if len(inbuf) == 0 || len(outbuf) == 0 {
		return errors.New("libtiff: empty buffer for ReadFromUserBuffer")
	}
	C.clearHandleError(t.tif)
	if C.tiffReadFromUserBuffer(
		t.tif, C.uint32_t(strile),
		unsafe.Pointer(&inbuf[0]), C.tmsize_t(len(inbuf)),
		unsafe.Pointer(&outbuf[0]), C.tmsize_t(len(outbuf)),
	) == 0 {
		runtime.KeepAlive(inbuf)
		runtime.KeepAlive(outbuf)
		if err := t.lastError(); err != nil {
		return &ReadError{Op: "read_from_user_buffer", Msg: err.Error()}
		}
		return &ReadError{Op: "read_from_user_buffer", Msg: fmt.Sprintf("strile %d", strile)}
	}
	runtime.KeepAlive(inbuf)
		runtime.KeepAlive(outbuf)
	return nil
}

// --- RGBA extended interfaces ---

// ReadRGBAImageOriented reads the whole image as RGBA, applying the given orientation.
// The orientation parameter uses ORIENTATION_* constants (e.g. OrientationTopLeft).
// If stopOnError is true, reading stops at the first error.
func (t *TIFF) ReadRGBAImageOriented(buf []uint32, orientation int, stopOnError bool) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	w, err := t.Width()
	if err != nil {
		return err
	}
	h, err := t.Height()
	if err != nil {
		return err
	}
	required := int(w) * int(h)
	if len(buf) < required {
		return &ReadError{Op: "rgba_oriented", Msg: fmt.Sprintf("buffer too small: got %d, need %d", len(buf), required)}
	}
	var stop C.int
	if stopOnError {
		stop = 1
	}
	C.clearHandleError(t.tif)
	if C.tiffReadRGBAImageOriented(t.tif, C.uint32_t(w), C.uint32_t(h), (*C.uint32_t)(unsafe.Pointer(&buf[0])), C.int(orientation), stop) == 0 {
		runtime.KeepAlive(buf)
		if err := t.lastError(); err != nil {
			return &ReadError{Op: "rgba_oriented", Msg: err.Error()}
		}
		return &ReadError{Op: "rgba_oriented", Msg: "failed"}
	}
		runtime.KeepAlive(buf)
	return nil
}

// ReadRGBAStripExt reads a single strip as RGBA with stop-on-error control.
func (t *TIFF) ReadRGBAStripExt(strip uint32, buf []uint32, stopOnError bool) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	if len(buf) == 0 {
		return errors.New("libtiff: empty buffer for ReadRGBAStripExt")
	}
	var stop C.int
	if stopOnError {
		stop = 1
	}
	C.clearHandleError(t.tif)
	if C.tiffReadRGBAStripExt(t.tif, C.uint32_t(strip), (*C.uint32_t)(unsafe.Pointer(&buf[0])), stop) == 0 {
		runtime.KeepAlive(buf)
		if err := t.lastError(); err != nil {
			return &ReadError{Op: "rgba_strip_ext", Msg: err.Error()}
		}
		return &ReadError{Op: "rgba_strip_ext", Msg: fmt.Sprintf("strip %d", strip)}
	}
	runtime.KeepAlive(buf)
	return nil
}

// ReadRGBATileExt reads a single tile as RGBA with stop-on-error control.
func (t *TIFF) ReadRGBATileExt(tile uint32, buf []uint32, stopOnError bool) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	if len(buf) == 0 {
		return errors.New("libtiff: empty buffer for ReadRGBATileExt")
	}
	tw, err := t.TileWidth()
	if err != nil {
		return err
	}
	th, err := t.TileLength()
	if err != nil {
		return err
	}
	var stop C.int
	if stopOnError {
		stop = 1
	}
	C.clearHandleError(t.tif)
	if C.tiffReadRGBATileExt(t.tif, C.uint32_t(tw), C.uint32_t(th), (*C.uint32_t)(unsafe.Pointer(&buf[0])), stop) == 0 {
		runtime.KeepAlive(buf)
		if err := t.lastError(); err != nil {
			return &ReadError{Op: "rgba_tile_ext", Msg: err.Error()}
		}
		return &ReadError{Op: "rgba_tile_ext", Msg: fmt.Sprintf("tile %d", tile)}
	}
	runtime.KeepAlive(buf)
	return nil
}

// --- FlushData ---

// FlushData flushes pending data to the file without updating the directory.
func (t *TIFF) FlushData() error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.TIFFFlushData(t.tif) == 0 {
		if err := t.lastError(); err != nil {
			return err
		}
		return errors.New("libtiff: failed to flush data")
	}
	return nil
}
// CurrentDirOffset returns the byte offset of the current IFD in the file.
func (t *TIFF) CurrentDirOffset() uint64 {
	if err := t.checkOpen(); err != nil {
		return 0
	}
	return uint64(C.tiffCurrentDirOffset(t.tif))
}

// --- Size queries ---

// RawStripSize returns the number of bytes in a raw (compressed) strip.
// Useful for allocating buffers before calling ReadRawStrip.
func (t *TIFF) RawStripSize(strip uint32) int64 {
	if err := t.checkOpen(); err != nil {
		return -1
	}
	return int64(C.TIFFRawStripSize(t.tif, C.uint32_t(strip)))
}

// RasterScanlineSize returns the number of bytes in a decoded scanline
// (may differ from ScanlineSize for planar-configured images).
func (t *TIFF) RasterScanlineSize() int64 {
	if err := t.checkOpen(); err != nil {
		return 0
	}
	return int64(C.TIFFRasterScanlineSize(t.tif))
}

// VStripSize returns the number of bytes for nrows of data.
func (t *TIFF) VStripSize(nrows uint32) int64 {
	if err := t.checkOpen(); err != nil {
		return 0
	}
	return int64(C.TIFFVStripSize(t.tif, C.uint32_t(nrows)))
}

// VTileSize returns the number of bytes for nrows of tile data.
func (t *TIFF) VTileSize(nrows uint32) int64 {
	if err := t.checkOpen(); err != nil {
		return 0
	}
	return int64(C.TIFFVTileSize(t.tif, C.uint32_t(nrows)))
}

// --- Tile coordinate operations ---

// ComputeTile returns the tile number for a pixel at (x, y, z) in sample s.
func (t *TIFF) ComputeTile(x, y, z uint32, sample uint16) uint32 {
	if err := t.checkOpen(); err != nil {
		return 0
	}
	return uint32(C.TIFFComputeTile(t.tif, C.uint32_t(x), C.uint32_t(y), C.uint32_t(z), C.uint16_t(sample)))
}

// ComputeStrip returns the strip number for a row in the given sample.
func (t *TIFF) ComputeStrip(row uint32, sample uint16) uint32 {
	if err := t.checkOpen(); err != nil {
		return 0
	}
	return uint32(C.TIFFComputeStrip(t.tif, C.uint32_t(row), C.uint16_t(sample)))
}

// ReadTile reads and decompresses the tile containing pixel (x, y, z)
// in sample s into buf. Returns the number of bytes read.
func (t *TIFF) ReadTile(x, y, z uint32, sample uint16, buf []byte) (int, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	if len(buf) == 0 {
		return 0, errors.New("libtiff: empty buffer for ReadTile")
	}
	C.clearHandleError(t.tif)
	n := C.TIFFReadTile(t.tif, unsafe.Pointer(&buf[0]), C.uint32_t(x), C.uint32_t(y), C.uint32_t(z), C.uint16_t(sample))
	runtime.KeepAlive(buf)
	if n < 0 {
		if err := t.lastError(); err != nil {
			return 0, &ReadError{Op: "tile", Msg: err.Error()}
		}
		return 0, &ReadError{Op: "tile", Msg: fmt.Sprintf("tile (%d,%d,%d) sample %d", x, y, z, sample)}
	}
	return int(n), nil
}

// WriteTile compresses and writes data to the tile containing pixel (x, y, z)
// in sample s. Returns the number of bytes written.
func (t *TIFF) WriteTile(x, y, z uint32, sample uint16, data []byte) (int, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	if len(data) == 0 {
		return 0, errors.New("libtiff: empty data for WriteTile")
	}
	C.clearHandleError(t.tif)
	n := C.TIFFWriteTile(t.tif, unsafe.Pointer(&data[0]), C.uint32_t(x), C.uint32_t(y), C.uint32_t(z), C.uint16_t(sample))
	runtime.KeepAlive(data)
	if n < 0 {
		if err := t.lastError(); err != nil {
			return 0, &WriteError{Op: "tile", Msg: err.Error()}
		}
		return 0, &WriteError{Op: "tile", Msg: fmt.Sprintf("tile (%d,%d,%d) sample %d", x, y, z, sample)}
	}
	return int(n), nil
}

// --- Tile defaults ---

// DefaultTileSize returns the default tile width and height for the image.
func (t *TIFF) DefaultTileSize() (uint32, uint32) {
	if err := t.checkOpen(); err != nil {
		return 0, 0
	}
	var tw, th C.uint32_t
	C.tiffDefaultTileSize(t.tif, &tw, &th)
	return uint32(tw), uint32(th)
}

// --- Auto-dispatch ---

// SetFieldAny writes a value to a tag, automatically dispatching to the correct
// SetField* variant based on the Go type of value and the tag's field metadata
// (pass-count, storage size).
//
// Supported value types:
//   - uint8, uint16, uint32, uint64, float64, string
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

// --- Library info ---

// GetVersion returns the libtiff version string.
func GetVersion() string {
	return C.GoString(C.TIFFGetVersion())
}
