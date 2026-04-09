package golibtiff

/*
#include <tiffio.h>
#include <stdlib.h>
#include <string.h>

// Error capture via thread-local storage, using global error handler.
static __thread char tiffLastErrMsg[1024] = {0};
static __thread int tiffHasErr = 0;

static void globalErrorHandler(const char *module, const char *fmt, va_list ap) {
	(void)module;
	vsnprintf(tiffLastErrMsg, sizeof(tiffLastErrMsg), fmt, ap);
	tiffHasErr = 1;
}

static void initErrorCapture(void) {
	TIFFSetErrorHandler(globalErrorHandler);
	TIFFSetWarningHandler(NULL);
}

static void clearLastTIFFError(void) { tiffHasErr = 0; tiffLastErrMsg[0] = '\0'; }
static int hasLastTIFFError(void) { return tiffHasErr; }
static const char *getLastTIFFError(void) { return tiffLastErrMsg; }

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
// Returns 1 if known, 0 if unknown (calling TIFFSetField on unknown tags may crash).
static int tiffIsFieldKnown(TIFF *t, uint32_t tag) {
	return TIFFFieldWithTag(t, tag) != NULL;
}
// Get the registered TIFFDataType for a tag.
// Returns -1 if tag is unknown. TIFFDataType: 1=BYTE, 2=ASCII, 3=SHORT, 4=LONG,
// 5=RATIONAL, 6=SBYTE, 7=UNDEFINED, 8=SSHORT, 9=SLONG, 10=SRATIONAL, 11=FLOAT, 12=DOUBLE, 13=IFD.
static int tiffGetFieldType(TIFF *t, uint32_t tag) {
	const TIFFField *f = TIFFFieldWithTag(t, tag);
	return f ? (int)TIFFFieldDataType(f) : -1;
}
// Check whether a tag requires a count argument in TIFFSetField.
// Returns 1 if count is needed, 0 if not (C0/fixed-count tags), -1 if unknown.
static int tiffFieldPassCount(TIFF *t, uint32_t tag) {
	const TIFFField *f = TIFFFieldWithTag(t, tag);
	return f ? (int)TIFFFieldPassCount(f) : -1;
}
// Get the write count for a tag. Positive = fixed count, -1 = TIFF_VARIABLE,
// -2 = TIFF_VARIABLE2, -3 = TIFF_SPP. Returns 0 if unknown.
static int tiffFieldWriteCount(TIFF *t, uint32_t tag) {
	const TIFFField *f = TIFFFieldWithTag(t, tag);
	return f ? (int)TIFFFieldWriteCount(f) : 0;
}

// Diagnostic: test GPS Sub-IFD creation (minimal reproduction).
static int tiffTestCreateGPSSubIFD(TIFF *tif, uint64_t *gpsOffset) {
    unsigned char gpsVersion[4] = {2, 2, 0, 1};
    uint64_t off = 0;
    clearLastTIFFError();
    if (TIFFWriteDirectory(tif) == 0) {
        snprintf(tiffLastErrMsg, sizeof(tiffLastErrMsg), "WriteDirectory failed");
        return -1;
    }
    if (TIFFSetDirectory(tif, 0) == 0) {
        snprintf(tiffLastErrMsg, sizeof(tiffLastErrMsg), "SetDirectory(0) failed");
        return -1;
    }
    if (TIFFCreateGPSDirectory(tif) != 0) {
        snprintf(tiffLastErrMsg, sizeof(tiffLastErrMsg), "CreateGPSDirectory failed");
        return -1;
    }
    if (!TIFFSetField(tif, 0, gpsVersion)) {
        snprintf(tiffLastErrMsg, sizeof(tiffLastErrMsg), "SetField GPSVersionID failed");
        return -1;
    }
    if (TIFFWriteCustomDirectory(tif, &off) == 0) {
        snprintf(tiffLastErrMsg, sizeof(tiffLastErrMsg),
                 "WriteCustomDirectory failed: %s", tiffLastErrMsg);
        return -1;
    }
    *gpsOffset = off;
    return 0;
}
*/
import "C"

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

var initOnce sync.Once

// TIFF represents an open TIFF file handle.
// It must be closed after use via Close() to release resources.
// A TIFF handle is NOT thread-safe; do not use concurrently from multiple goroutines.
type TIFF struct {
	tif *C.TIFF
}

// OpenMode controls how a TIFF file is opened.
type OpenMode string

const (
	OpenRead      OpenMode = "r"
	OpenWrite     OpenMode = "w"
	OpenAppend    OpenMode = "a"
	OpenReadWrite OpenMode = "r+"
)

// --- Core Operations ---

// Open opens a TIFF file at the given path with the specified mode.
func Open(path string, mode OpenMode) (*TIFF, error) {
	initOnce.Do(func() { C.initErrorCapture() })
	C.clearLastTIFFError()

	tif, err := openTiffHandle(path, mode)
	if err != nil {
		return nil, &OpenError{Path: path, Mode: mode, Msg: err.Error()}
	}
	if tif == nil {
		if C.hasLastTIFFError() != 0 {
			return nil, &OpenError{Path: path, Mode: mode, Msg: C.GoString(C.getLastTIFFError())}
		}
		return nil, &OpenError{Path: path, Mode: mode}
	}

	t := &TIFF{tif: tif}
	runtime.SetFinalizer(t, (*TIFF).Close)
	return t, nil
}

// Close releases the TIFF file handle resources. Safe to call multiple times.
func (t *TIFF) Close() error {
	if t.tif != nil {
		C.TIFFClose(t.tif)
		t.tif = nil
		runtime.SetFinalizer(t, nil)
	}
	return nil
}

// FileName returns the file name associated with the TIFF handle.
func (t *TIFF) FileName() string {
	return C.GoString(C.TIFFFileName(t.tif))
}

func (t *TIFF) checkOpen() error {
	if t.tif == nil {
		return errors.New("libtiff: handle is closed")
	}
	return nil
}

func lastError() error {
	if C.hasLastTIFFError() != 0 {
		return fmt.Errorf("%s", C.GoString(C.getLastTIFFError()))
	}
	return nil
}

// --- GetField ---

func (t *TIFF) GetFieldUint16(tag Tag) (uint16, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	C.clearLastTIFFError()
	var val C.uint16_t
	if C.tiffGetFieldU16(t.tif, C.uint32_t(tag), &val) == 0 {
		return 0, &FieldError{Tag: tag, Op: "get", Msg: "field not found"}
	}
	return uint16(val), nil
}

func (t *TIFF) GetFieldUint32(tag Tag) (uint32, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	C.clearLastTIFFError()
	var val C.uint32_t
	if C.tiffGetFieldU32(t.tif, C.uint32_t(tag), &val) == 0 {
		return 0, &FieldError{Tag: tag, Op: "get", Msg: "field not found"}
	}
	return uint32(val), nil
}

func (t *TIFF) GetFieldFloat(tag Tag) (float64, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	C.clearLastTIFFError()
	var val C.float
	if C.tiffGetFieldFloat(t.tif, C.uint32_t(tag), &val) == 0 {
		return 0, &FieldError{Tag: tag, Op: "get", Msg: "field not found"}
	}
	return float64(val), nil
}

func (t *TIFF) GetFieldString(tag Tag) (string, error) {
	if err := t.checkOpen(); err != nil {
		return "", err
	}
	C.clearLastTIFFError()
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
	C.clearLastTIFFError()
	var val C.uint8_t
	if C.tiffGetFieldU8(t.tif, C.uint32_t(tag), &val) == 0 {
		return 0, &FieldError{Tag: tag, Op: "get", Msg: "field not found"}
	}
	return uint8(val), nil
}

func (t *TIFF) GetFieldUint64(tag Tag) (uint64, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	C.clearLastTIFFError()
	var val C.uint64_t
	if C.tiffGetFieldU64(t.tif, C.uint32_t(tag), &val) == 0 {
		return 0, &FieldError{Tag: tag, Op: "get", Msg: "field not found"}
	}
	return uint64(val), nil
}

func (t *TIFF) GetFieldUint16Slice(tag Tag) ([]uint16, error) {
	if err := t.checkOpen(); err != nil {
		return nil, err
	}
	C.clearLastTIFFError()
	var data *C.uint16_t
	var count C.uint16_t
	if C.tiffGetFieldU16Array(t.tif, C.uint32_t(tag), &data, &count) == 0 {
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
	C.clearLastTIFFError()
	var data *C.uint32_t
	var count C.uint32_t
	if C.tiffGetFieldU32Array(t.tif, C.uint32_t(tag), &data, &count) == 0 {
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
	C.clearLastTIFFError()
	if C.tiffSetFieldU16(t.tif, C.uint32_t(tag), C.uint16_t(v)) == 0 {
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	return nil
}

func (t *TIFF) SetFieldUint32(tag Tag, v uint32) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearLastTIFFError()
	if C.tiffSetFieldU32(t.tif, C.uint32_t(tag), C.uint32_t(v)) == 0 {
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	return nil
}

func (t *TIFF) SetFieldFloat(tag Tag, v float64) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearLastTIFFError()
	if C.tiffSetFieldFloat(t.tif, C.uint32_t(tag), C.float(v)) == 0 {
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
	C.clearLastTIFFError()
	if C.tiffSetFieldString(t.tif, C.uint32_t(tag), cStr) == 0 {
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
	C.clearLastTIFFError()
	if C.tiffSetFieldU16Array(t.tif, C.uint32_t(tag), C.uint16_t(len(v)), (*C.uint16_t)(unsafe.Pointer(&v[0]))) == 0 {
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	return nil
}

func (t *TIFF) SetFieldUint32Slice(tag Tag, v []uint32) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	if len(v) == 0 {
		return nil
	}
	C.clearLastTIFFError()
	if C.tiffSetFieldU32Array(t.tif, C.uint32_t(tag), C.uint32_t(len(v)), (*C.uint32_t)(unsafe.Pointer(&v[0]))) == 0 {
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
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
	C.clearLastTIFFError()
	if C.TIFFReadScanline(t.tif, unsafe.Pointer(&buf[0]), C.uint32_t(row), 0) < 0 {
		if err := lastError(); err != nil {
			return &ReadError{Op: "scanline", Msg: err.Error()}
		}
		return &ReadError{Op: "scanline", Msg: fmt.Sprintf("row %d", row)}
	}
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
	C.clearLastTIFFError()
	cSize := C.tmsize_t(size)
	if cSize <= 0 {
		cSize = C.tmsize_t(len(buf))
	}
	n := C.TIFFReadEncodedStrip(t.tif, C.uint32_t(strip), unsafe.Pointer(&buf[0]), cSize)
	if n < 0 {
		if err := lastError(); err != nil {
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
	C.clearLastTIFFError()
	n := C.TIFFReadRawStrip(t.tif, C.uint32_t(strip), unsafe.Pointer(&buf[0]), C.tmsize_t(size))
	if n < 0 {
		if err := lastError(); err != nil {
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
	w, h := t.Width(), t.Height()
	if w == 0 || h == 0 {
		return &ReadError{Op: "rgba_image", Msg: "invalid dimensions"}
	}
	C.clearLastTIFFError()
	if C.tiffReadRGBAImage(t.tif, C.uint32_t(w), C.uint32_t(h), (*C.uint32_t)(unsafe.Pointer(&buf[0]))) == 0 {
		if err := lastError(); err != nil {
			return &ReadError{Op: "rgba_image", Msg: err.Error()}
		}
		return &ReadError{Op: "rgba_image", Msg: "failed"}
	}
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
	C.clearLastTIFFError()
	if C.TIFFWriteScanline(t.tif, unsafe.Pointer(&buf[0]), C.uint32_t(row), 0) < 0 {
		if err := lastError(); err != nil {
			return &WriteError{Op: "scanline", Msg: err.Error()}
		}
		return &WriteError{Op: "scanline", Msg: fmt.Sprintf("row %d", row)}
	}
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
	C.clearLastTIFFError()
	n := C.TIFFWriteEncodedStrip(t.tif, C.uint32_t(strip), unsafe.Pointer(&data[0]), C.tmsize_t(len(data)))
	if n < 0 {
		if err := lastError(); err != nil {
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
	C.clearLastTIFFError()
	n := C.TIFFWriteRawStrip(t.tif, C.uint32_t(strip), unsafe.Pointer(&data[0]), C.tmsize_t(len(data)))
	if n < 0 {
		if err := lastError(); err != nil {
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
	C.clearLastTIFFError()
	cSize := C.tmsize_t(size)
	if cSize <= 0 {
		cSize = C.tmsize_t(len(buf))
	}
	n := C.tiffReadEncodedTile(t.tif, C.uint32_t(tile), unsafe.Pointer(&buf[0]), cSize)
	if n < 0 {
		if err := lastError(); err != nil {
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
	C.clearLastTIFFError()
	n := C.tiffWriteEncodedTile(t.tif, C.uint32_t(tile), unsafe.Pointer(&data[0]), C.tmsize_t(len(data)))
	if n < 0 {
		if err := lastError(); err != nil {
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
		if err := lastError(); err != nil {
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
	C.clearLastTIFFError()
	if C.TIFFSetDirectory(t.tif, C.tdir_t(dirnum)) == 0 {
		if err := lastError(); err != nil {
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
	C.clearLastTIFFError()
	if C.TIFFWriteDirectory(t.tif) == 0 {
		if err := lastError(); err != nil {
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
	C.clearLastTIFFError()
	if C.tiffCheckpointDirectory(t.tif) == 0 {
		if err := lastError(); err != nil {
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
	C.clearLastTIFFError()
	if C.TIFFSetSubDirectory(t.tif, C.uint64_t(offset)) == 0 {
		if err := lastError(); err != nil {
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
	C.clearLastTIFFError()
	if C.tiffReadEXIFDirectory(t.tif, C.uint64_t(offset)) == 0 {
		if err := lastError(); err != nil {
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
// For tags that take (count, data) arguments.
func (t *TIFF) SetFieldByteSlice(tag Tag, v []byte) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	if len(v) == 0 {
		return nil
	}
	C.clearLastTIFFError()
	if C.tiffSetFieldByteSlice(t.tif, C.uint32_t(tag), C.uint32_t(len(v)), (*C.uint8_t)(unsafe.Pointer(&v[0]))) == 0 {
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	return nil
}

// SetFieldC0ByteSlice sets a fixed-count byte-array field (no count argument).
// For C0 tags like DNGVersion (BYTE[4]) that use TIFF_SETGET_C0_UINT8.
func (t *TIFF) SetFieldC0ByteSlice(tag Tag, v []byte) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	if len(v) == 0 {
		return nil
	}
	C.clearLastTIFFError()
	if C.tiffSetFieldC0ByteSlice(t.tif, C.uint32_t(tag), (*C.uint8_t)(unsafe.Pointer(&v[0]))) == 0 {
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	return nil
}

// CreateEXIFDirectory creates a new EXIF Sub-IFD.
// After calling this, use SetField* to populate EXIF tags, then WriteCustomDirectory.
func (t *TIFF) CreateEXIFDirectory() error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearLastTIFFError()
	if C.tiffCreateEXIFDirectory(t.tif) != 0 {
		if err := lastError(); err != nil {
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
	C.clearLastTIFFError()
	var offset C.uint64_t
	if C.tiffWriteCustomDirectory(t.tif, &offset) == 0 {
		if err := lastError(); err != nil {
			return 0, fmt.Errorf("libtiff: WriteCustomDirectory: %w", err)
		}
		return 0, errors.New("libtiff: WriteCustomDirectory failed")
	}
	return uint64(offset), nil
}

// SetFieldFloatSlice sets a RATIONAL array field by writing each value as a separate
// TIFFSetField call. Used for tags like AsShotNeutral (RATIONAL[3]) and LensInfo (RATIONAL[4]).
//
// Note: This uses TIFFSetField directly with the tag and a float64 value per element.
// For EXIF tags managed by TIFFCreateEXIFDirectory, the standard EXIF tag definitions
// handle the RATIONAL encoding automatically.
func (t *TIFF) SetFieldFloatSlice(tag Tag, v []float64) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	if len(v) == 0 {
		return nil
	}
	C.clearLastTIFFError()
	// Build a C float array and use TIFFSetField with count + pointer.
	floats := make([]C.float, len(v))
	for i, f := range v {
		floats[i] = C.float(f)
	}
	// Use the existing byte-slice setter pattern with count + data.
	// For RATIONAL arrays in custom directories, libtiff expects (count, float*).
	if C.tiffSetFieldFloatSlice(t.tif, C.uint32_t(tag), C.int(len(v)), &floats[0]) == 0 {
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	return nil
}

// SetFieldUint64 sets a uint64 field (e.g. EXIFIFD pointer after WriteCustomDirectory).
func (t *TIFF) SetFieldUint64(tag Tag, v uint64) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearLastTIFFError()
	if C.tiffSetFieldU64(t.tif, C.uint32_t(tag), C.uint64_t(v)) == 0 {
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	return nil
}

// SetFieldUint8 sets a single-byte field (e.g. SceneType: SETGET_UINT8).
func (t *TIFF) SetFieldUint8(tag Tag, v uint8) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearLastTIFFError()
	if C.tiffSetFieldU8(t.tif, C.uint32_t(tag), C.uint8_t(v)) == 0 {
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	return nil
}

// SetFieldC0FloatSlice sets a fixed-count float array field (e.g. LensSpecification: SETGET_C0_FLOAT).
// Unlike SetFieldFloatSlice, this does NOT pass a count argument — libtiff knows the fixed count.
func (t *TIFF) SetFieldC0FloatSlice(tag Tag, v []float64) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	if len(v) == 0 {
		return nil
	}
	C.clearLastTIFFError()
	floats := make([]C.float, len(v))
	for i, f := range v {
		floats[i] = C.float(f)
	}
	if C.tiffSetFieldC0Float(t.tif, C.uint32_t(tag), &floats[0]) == 0 {
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	return nil
}

// SetFieldDouble sets a double-precision floating-point field (64-bit, no precision loss).
func (t *TIFF) SetFieldDouble(tag Tag, v float64) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearLastTIFFError()
	if C.tiffSetFieldDouble(t.tif, C.uint32_t(tag), C.double(v)) == 0 {
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
	C.clearLastTIFFError()
	doubles := make([]C.double, len(v))
	for i, f := range v {
		doubles[i] = C.double(f)
	}
	if C.tiffSetFieldDoubleSlice(t.tif, C.uint32_t(tag), C.int(len(v)), &doubles[0]) == 0 {
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
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
	C.clearLastTIFFError()
	doubles := make([]C.double, len(v))
	for i, f := range v {
		doubles[i] = C.double(f)
	}
	if C.tiffSetFieldC0Double(t.tif, C.uint32_t(tag), &doubles[0]) == 0 {
		return &FieldError{Tag: tag, Op: "set", Msg: "failed"}
	}
	return nil
}

// CreateGPSDirectory creates a new GPS Sub-IFD.
// After calling this, use SetField* to populate GPS tags, then WriteCustomDirectory.
func (t *TIFF) CreateGPSDirectory() error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearLastTIFFError()
	if C.tiffCreateGPSDirectory(t.tif) != 0 {
		if err := lastError(); err != nil {
			return fmt.Errorf("libtiff: CreateGPSDirectory: %w", err)
		}
		return errors.New("libtiff: CreateGPSDirectory failed")
	}
	return nil
}

// TestCreateGPSSubIFD tests GPS Sub-IFD creation entirely in C.
// Diagnostic method to isolate CGO vs libtiff issues.
func (t *TIFF) TestCreateGPSSubIFD() (uint64, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	C.clearLastTIFFError()
	var off C.uint64_t
	if C.tiffTestCreateGPSSubIFD(t.tif, &off) != 0 {
		if err := lastError(); err != nil {
			return 0, fmt.Errorf("libtiff: TestCreateGPSSubIFD: %w", err)
		}
		return 0, errors.New("libtiff: TestCreateGPSSubIFD failed")
	}
	return uint64(off), nil
}

// IsFieldKnown checks if a tag is registered in libtiff's field definitions.
// Tags not known to libtiff cannot be written with TIFFSetField and may cause crashes.
func (t *TIFF) IsFieldKnown(tag Tag) bool {
	if err := t.checkOpen(); err != nil {
		return false
	}
	return C.tiffIsFieldKnown(t.tif, C.uint32_t(tag)) != 0
}

// GetFieldType returns the libtiff-registered TIFFDataType for a tag.
// Returns -1 if the tag is not registered.
// TIFFDataType values: 1=BYTE, 2=ASCII, 3=SHORT, 4=LONG, 5=RATIONAL,
// 6=SBYTE, 7=UNDEFINED, 8=SSHORT, 9=SLONG, 10=SRATIONAL, 11=FLOAT, 12=DOUBLE, 13=IFD.
func (t *TIFF) GetFieldType(tag Tag) int {
	if err := t.checkOpen(); err != nil {
		return -1
	}
	return int(C.tiffGetFieldType(t.tif, C.uint32_t(tag)))
}

// FieldPassCount reports whether a tag requires a count argument in TIFFSetField.
// Returns true for normal array tags (count needed), false for C0/fixed-count tags (no count).
// Returns false for unknown tags.
func (t *TIFF) FieldPassCount(tag Tag) bool {
	if err := t.checkOpen(); err != nil {
		return false
	}
	return C.tiffFieldPassCount(t.tif, C.uint32_t(tag)) != 0
}

// FieldWriteCount returns the number of values a tag expects.
// Positive = fixed count (1 = scalar, >1 = fixed array).
// Negative = variable (-1=TIFF_VARIABLE, -2=TIFF_VARIABLE2, -3=TIFF_SPP).
// Returns 0 for unknown tags.
func (t *TIFF) FieldWriteCount(tag Tag) int {
	if err := t.checkOpen(); err != nil {
		return 0
	}
	return int(C.tiffFieldWriteCount(t.tif, C.uint32_t(tag)))
}
