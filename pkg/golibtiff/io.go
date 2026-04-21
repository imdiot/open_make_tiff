package golibtiff

/*
#include <tiffio.h>
#include <stdlib.h>
#include "libtiff_bridge.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"
)

// --- Info Queries (direct C API calls) ---

func (t *TIFF) IsTiled() bool {
	if t.tif == nil {
		return false
	}
	return C.TIFFIsTiled(t.tif) != 0
}

func (t *TIFF) ScanlineSize() int {
	if t.tif == nil {
		return 0
	}
	return int(C.TIFFScanlineSize(t.tif))
}

func (t *TIFF) StripSize() int {
	if t.tif == nil {
		return 0
	}
	return int(C.TIFFStripSize(t.tif))
}

// DefaultStripSize returns the default RowsPerStrip value (8192 / scanline_size).
func (t *TIFF) DefaultStripSize() uint32 {
	if t.tif == nil {
		return 0
	}
	return uint32(C.TIFFDefaultStripSize(t.tif, 0))
}

func (t *TIFF) TileSize() int {
	if t.tif == nil {
		return 0
	}
	return int(C.TIFFTileSize(t.tif))
}

func (t *TIFF) NumberOfStrips() uint32 {
	if t.tif == nil {
		return 0
	}
	return uint32(C.TIFFNumberOfStrips(t.tif))
}

func (t *TIFF) NumberOfTiles() uint32 {
	if t.tif == nil {
		return 0
	}
	return uint32(C.TIFFNumberOfTiles(t.tif))
}

func (t *TIFF) IsBigTIFF() bool {
	if t.tif == nil {
		return false
	}
	return C.TIFFIsBigTIFF(t.tif) != 0
}

func (t *TIFF) IsByteSwapped() bool {
	if t.tif == nil {
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
func (t *TIFF) ReadEncodedStrip(strip uint32, buf []byte, size int) (int, error) {
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

func (t *TIFF) ReadRawStrip(strip uint32, buf []byte, size int) (int, error) {
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
func (t *TIFF) ReadEncodedTile(tile uint32, buf []byte, size int) (int, error) {
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

// --- Strile low-level access ---

// StrileOffset returns the byte offset of the given strip or tile.
func (t *TIFF) StrileOffset(strile uint32) uint64 {
	if t.tif == nil {
		return 0
	}
	return uint64(C.tiffGetStrileOffset(t.tif, C.uint32_t(strile)))
}

// StrileByteCount returns the byte count of the given strip or tile.
func (t *TIFF) StrileByteCount(strile uint32) uint64 {
	if t.tif == nil {
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
func (t *TIFF) TileRowSize() int {
	if t.tif == nil {
		return 0
	}
	return int(C.TIFFTileRowSize(t.tif))
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
	if t.tif == nil {
		return 0
	}
	return uint64(C.tiffCurrentDirOffset(t.tif))
}

// --- Size queries ---

// RawStripSize returns the number of bytes in a raw (compressed) strip.
// Useful for allocating buffers before calling ReadRawStrip.
func (t *TIFF) RawStripSize(strip uint32) int {
	if t.tif == nil {
		return -1
	}
	return int(C.TIFFRawStripSize(t.tif, C.uint32_t(strip)))
}

// RasterScanlineSize returns the number of bytes in a decoded scanline
// (may differ from ScanlineSize for planar-configured images).
func (t *TIFF) RasterScanlineSize() int {
	if t.tif == nil {
		return 0
	}
	return int(C.TIFFRasterScanlineSize(t.tif))
}

// VStripSize returns the number of bytes for nrows of data.
func (t *TIFF) VStripSize(nrows uint32) int {
	if t.tif == nil {
		return 0
	}
	return int(C.TIFFVStripSize(t.tif, C.uint32_t(nrows)))
}

// VTileSize returns the number of bytes for nrows of tile data.
func (t *TIFF) VTileSize(nrows uint32) int {
	if t.tif == nil {
		return 0
	}
	return int(C.TIFFVTileSize(t.tif, C.uint32_t(nrows)))
}

// --- Tile coordinate operations ---

// ComputeTile returns the tile number for a pixel at (x, y, z) in sample s.
func (t *TIFF) ComputeTile(x, y, z uint32, sample uint16) uint32 {
	if t.tif == nil {
		return 0
	}
	return uint32(C.TIFFComputeTile(t.tif, C.uint32_t(x), C.uint32_t(y), C.uint32_t(z), C.uint16_t(sample)))
}

// ComputeStrip returns the strip number for a row in the given sample.
func (t *TIFF) ComputeStrip(row uint32, sample uint16) uint32 {
	if t.tif == nil {
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
	if t.tif == nil {
		return 0, 0
	}
	var tw, th C.uint32_t
	C.tiffDefaultTileSize(t.tif, &tw, &th)
	return uint32(tw), uint32(th)
}

// --- Basic RGBA ---

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

// --- RGBA extended interfaces ---

// ReadRGBAImageOriented reads the whole image as RGBA, applying the given orientation.
// The orientation parameter uses ORIENTATION_* constants (e.g. OrientationTopLeft).
// If stopOnError is true, reading stops at the first error.
func (t *TIFF) ReadRGBAImageOriented(buf []uint32, orientation Orientation, stopOnError bool) error {
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
