package golibtiff

/*
#include <tiffio.h>
#include <stdlib.h>
#include <stdint.h>
#include "libtiff_bridge.h"

// Forward-declare the Go export trampolines.
extern tmsize_t goReadWriteProc(void *bridge, void *buf, tmsize_t size);
extern toff_t   goSeekProc(void *bridge, toff_t offset, int whence);
extern int      goCloseProc(void *bridge);
extern toff_t   goSizeProc(void *bridge);
extern int      goMapFileProc(void *bridge, char **base, toff_t *size);
extern void     goUnmapFileProc(void *bridge, void *base, toff_t size);

// Wrapper functions matching TIFF*Proc signatures, delegating to Go via bridge.
static tmsize_t cReadWriteProc(thandle_t bridge, void *buf, tmsize_t size) {
    return goReadWriteProc(bridge, buf, size);
}
static toff_t   cSeekProc(thandle_t bridge, toff_t offset, int whence) {
    return goSeekProc(bridge, offset, whence);
}
static int      cCloseProc(thandle_t bridge) {
    return goCloseProc(bridge);
}
static toff_t   cSizeProc(thandle_t bridge) {
    return goSizeProc(bridge);
}
static int      cMapFileProc(thandle_t bridge, void **pbase, toff_t *size) {
    char **base = (char **)pbase;
    return goMapFileProc(bridge, base, size);
}
static void     cUnmapFileProc(thandle_t bridge, void *base, toff_t size) {
    goUnmapFileProc(bridge, base, size);
}

// Single C entry point that calls TIFFClientOpenExt with the Go callback wrappers.
static TIFF *cClientOpenExt(const char *name, const char *mode, void *bridge, TIFFOpenOptions *opts) {
    return TIFFClientOpenExt(name, mode, bridge,
        cReadWriteProc, cReadWriteProc,
        cSeekProc, cCloseProc, cSizeProc,
        cMapFileProc, cUnmapFileProc, opts);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// ReadWriteProc is called to read/write data from/to the custom I/O source.
// Returns bytes transferred or an error.
type ReadWriteProc func(buf []byte) (int, error)

// SeekProc is called to seek within the custom I/O source.
// whence: 0=SEEK_SET, 1=SEEK_CUR, 2=SEEK_END. Returns new offset or error.
type SeekProc func(offset int64, whence int) (int64, error)

// CloseProc is called when the TIFF handle is closed.
type CloseProc func() error

// SizeProc returns the total size of the custom I/O source.
type SizeProc func() (int64, error)

// MapFileProc optionally maps the I/O source into memory.
// If nil, libtiff falls back to ReadWriteProc for reading.
type MapFileProc func() ([]byte, error)

// UnmapFileProc optionally unmaps a previously mapped buffer.
type UnmapFileProc func()

// ClientIO defines the callbacks required for TIFFClientOpen-based I/O.
type ClientIO struct {
	ReadWriteProc ReadWriteProc
	SeekProc      SeekProc
	CloseProc     CloseProc
	SizeProc      SizeProc
	MapFileProc   MapFileProc   // optional
	UnmapFileProc UnmapFileProc // optional
}

var (
	clientIOMu    sync.Mutex
	clientIOSeq   atomic.Int64
	clientIOStore = make(map[int64]*clientIOEntry)
)

type clientIOEntry struct {
	io         ClientIO
	writeMode  bool
	idPtr      *int64 // heap-allocated; keeps the pointer passed to libtiff alive
	mappedBuf  []byte // set by MapFileProc, cleared by UnmapFileProc to prevent GC
}

func registerClientIO(io ClientIO, writeMode bool) *int64 {
	idPtr := new(int64)
	*idPtr = clientIOSeq.Add(1)
	clientIOMu.Lock()
	clientIOStore[*idPtr] = &clientIOEntry{io: io, writeMode: writeMode, idPtr: idPtr}
	clientIOMu.Unlock()
	return idPtr
}

func unregisterClientIO(id int64) {
	clientIOMu.Lock()
	delete(clientIOStore, id)
	clientIOMu.Unlock()
}

func getEntry(id int64) (*clientIOEntry, bool) {
	clientIOMu.Lock()
	defer clientIOMu.Unlock()
	e, ok := clientIOStore[id]
	return e, ok
}

// OpenWithCallbacks opens a TIFF using custom I/O callbacks instead of a file path.
// The name parameter is used only for error messages. The ClientIO callbacks
// provide all I/O operations. The returned TIFF must be closed to release resources.
//
// The close callback is called when the TIFF handle is closed (via Close() or finalizer).
func OpenWithCallbacks(name string, mode OpenMode, io ClientIO) (*TIFF, error) {
	if io.ReadWriteProc == nil || io.SeekProc == nil || io.CloseProc == nil || io.SizeProc == nil {
		return nil, errors.New("libtiff: ReadWriteProc, SeekProc, CloseProc, and SizeProc are required")
	}

	writeMode := mode != OpenRead
	idPtr := registerClientIO(io, writeMode)

	C.clearOpenPhaseError()

	opts := C.TIFFOpenOptionsAlloc()
	defer C.TIFFOpenOptionsFree(opts)
	var handler C.TIFFErrorHandlerExtR
	C.getPerHandleErrorHandler(&handler)
	C.TIFFOpenOptionsSetErrorHandlerExtR(opts, handler, nil)
	C.TIFFOpenOptionsSetWarningHandlerExtR(opts, handler, nil)

	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	cMode := C.CString(string(mode))
	defer C.free(unsafe.Pointer(cMode))

	tif := C.cClientOpenExt(cName, cMode, unsafe.Pointer(idPtr), opts)

	if tif == nil {
		unregisterClientIO(*idPtr)
		if C.hasOpenPhaseError() != 0 {
			return nil, &OpenError{Path: name, Mode: mode, Msg: C.GoString(C.getOpenPhaseError())}
		}
		return nil, &OpenError{Path: name, Mode: mode}
	}

	C.attachErrorState(tif)

	t := &TIFF{tif: tif}
	t.cleanup = runtime.AddCleanup(t, func(tif *C.TIFF) {
		C.detachErrorState(tif)
		C.TIFFClose(tif)
	}, tif)
	return t, nil
}

// --- CGo export trampolines ---
// These functions are called from C and delegate to the Go callbacks stored in clientIOStore.

//export goReadWriteProc
func goReadWriteProc(bridge unsafe.Pointer, buf unsafe.Pointer, size C.tmsize_t) C.tmsize_t {
	id := *(*int64)(bridge)
	entry, ok := getEntry(id)
	if !ok {
		return -1
	}
	if size <= 0 {
		return 0
	}
	goBuf := unsafe.Slice((*byte)(buf), int(size))
	n, err := entry.io.ReadWriteProc(goBuf)
	if err != nil {
		return -1
	}
	return C.tmsize_t(n)
}

//export goSeekProc
func goSeekProc(bridge unsafe.Pointer, offset C.toff_t, whence C.int) C.toff_t {
	id := *(*int64)(bridge)
	entry, ok := getEntry(id)
	if !ok {
		return ^C.toff_t(0) // -1 in unsigned
	}
	newOffset, err := entry.io.SeekProc(int64(offset), int(whence))
	if err != nil {
		return ^C.toff_t(0)
	}
	return C.toff_t(newOffset)
}

//export goCloseProc
func goCloseProc(bridge unsafe.Pointer) C.int {
	id := *(*int64)(bridge)
	entry, ok := getEntry(id)
	if ok {
		_ = entry.io.CloseProc()
		unregisterClientIO(id)
	}
	return 0
}

//export goSizeProc
func goSizeProc(bridge unsafe.Pointer) C.toff_t {
	id := *(*int64)(bridge)
	entry, ok := getEntry(id)
	if !ok {
		return ^C.toff_t(0)
	}
	size, err := entry.io.SizeProc()
	if err != nil {
		return ^C.toff_t(0)
	}
	return C.toff_t(size)
}

//export goMapFileProc
func goMapFileProc(bridge unsafe.Pointer, base **C.char, size *C.toff_t) C.int {
	id := *(*int64)(bridge)
	entry, ok := getEntry(id)
	if !ok {
		return 0
	}
	if entry.io.MapFileProc == nil {
		return 0
	}
	data, err := entry.io.MapFileProc()
	if err != nil {
		return 0
	}
	if len(data) == 0 {
		return 0
	}
	*base = (*C.char)(unsafe.Pointer(&data[0]))
	*size = C.toff_t(len(data))
	entry.mappedBuf = data // prevent GC until UnmapFileProc
	return 1
}

//export goUnmapFileProc
func goUnmapFileProc(bridge unsafe.Pointer, base unsafe.Pointer, size C.toff_t) {
	id := *(*int64)(bridge)
	entry, ok := getEntry(id)
	if ok {
		entry.mappedBuf = nil // allow GC of mapped buffer
		if entry.io.UnmapFileProc != nil {
			entry.io.UnmapFileProc()
		}
	}
}

// --- Buffer-based convenience ---

// OpenFromBuffer opens a TIFF for reading from an in-memory byte buffer.
// The buffer contents are copied; modifications to the original slice after
// opening do not affect the TIFF.
func OpenFromBuffer(data []byte) (*TIFF, error) {
	if len(data) == 0 {
		return nil, errors.New("libtiff: empty buffer")
	}
	// Copy to avoid caller mutation.
	buf := make([]byte, len(data))
	copy(buf, data)

	var offset int64

	io := ClientIO{
		ReadWriteProc: func(dst []byte) (int, error) {
			if offset >= int64(len(buf)) {
				return 0, nil
			}
			n := copy(dst, buf[offset:])
			offset += int64(n)
			return n, nil
		},
		SeekProc: func(off int64, whence int) (int64, error) {
			switch whence {
			case 0: // SEEK_SET
				offset = off
			case 1: // SEEK_CUR
				offset += off
			case 2: // SEEK_END
				offset = int64(len(buf)) + off
			default:
				return 0, fmt.Errorf("libtiff: invalid whence %d", whence)
			}
			if offset < 0 {
				offset = 0
			}
			return offset, nil
		},
		CloseProc: func() error {
			buf = nil
			return nil
		},
		SizeProc: func() (int64, error) {
			return int64(len(buf)), nil
		},
		MapFileProc: func() ([]byte, error) {
			return buf, nil
		},
		UnmapFileProc: func() {},
	}

	return OpenWithCallbacks("buffer", OpenRead, io)
}

// WriteToBuffer opens a TIFF for writing to an in-memory buffer.
// Returns the TIFF handle and a function to retrieve the accumulated buffer contents.
// The buffer grows as data is written.
func WriteToBuffer() (*TIFF, func() []byte, error) {
	var buf []byte
	var offset int64

	io := ClientIO{
		ReadWriteProc: func(src []byte) (int, error) {
			if offset > int64(len(buf)) {
				return 0, fmt.Errorf("libtiff: write past end of buffer")
			}
			end := offset + int64(len(src))
			if end > int64(cap(buf)) {
				newCap := max(int64(cap(buf))*2, end)
				newBuf := make([]byte, len(buf), newCap)
				copy(newBuf, buf)
				buf = newBuf
			}
			if end > int64(len(buf)) {
				buf = buf[:end]
			}
			n := copy(buf[offset:], src)
			offset += int64(n)
			return n, nil
		},
		SeekProc: func(off int64, whence int) (int64, error) {
			switch whence {
			case 0:
				offset = off
			case 1:
				offset += off
			case 2:
				offset = int64(len(buf)) + off
			default:
				return 0, fmt.Errorf("libtiff: invalid whence %d", whence)
			}
			if offset < 0 {
				offset = 0
			}
			return offset, nil
		},
		CloseProc: func() error {
			return nil
		},
		SizeProc: func() (int64, error) {
			return int64(len(buf)), nil
		},
	}

	tif, err := OpenWithCallbacks("buffer", OpenWrite, io)
	if err != nil {
		return nil, nil, err
	}

	getBuffer := func() []byte {
		result := make([]byte, len(buf))
		copy(result, buf)
		return result
	}

	return tif, getBuffer, nil
}
