package golibraw

/*
#cgo pkg-config: libraw_r
#cgo CXXFLAGS: -std=c++14 -DUSE_DNGSDK
#cgo darwin CFLAGS: -mmacosx-version-min=10.13
#cgo darwin CXXFLAGS: -mmacosx-version-min=10.13
#cgo darwin LDFLAGS: -framework CoreServices
#cgo !darwin LDFLAGS: -lstdc++
#include <libraw/libraw.h>
#include <stdlib.h>

extern void* golibraw_create_dng_host();
extern void golibraw_destroy_dng_host(void* host);
extern void golibraw_set_dng_host_for_raw(libraw_data_t* lr, void* host);

static int golibraw_progress_cb(void* data, enum LibRaw_progress stage, int iteration, int expected) {
	return *((int*)data);
}

static void golibraw_register_cancel_cb(libraw_data_t* lr, int* flag) {
	libraw_set_progress_handler(lr, golibraw_progress_cb, flag);
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

var (
	ErrInitFailed     = errors.New("libraw: initialization failed")
	ErrAlreadyClosed  = errors.New("libraw: processor already closed")
	ErrFileOpenFailed = errors.New("libraw: failed to open file")
	ErrBufferOpen     = errors.New("libraw: failed to open buffer")
	ErrUnpack         = errors.New("libraw: failed to unpack raw data")
	ErrUnpackThumb    = errors.New("libraw: failed to unpack thumbnail")
	ErrProcess        = errors.New("libraw: processing failed")
	ErrMemImage       = errors.New("libraw: failed to create memory image")
	ErrWriteFailed    = errors.New("libraw: failed to write output")
)

// RawProcessor wraps libraw_data_t for RAW image processing.
type RawProcessor struct {
	handle     *C.libraw_data_t
	dngHost    unsafe.Pointer
	closed     bool
	mu         sync.Mutex
	cancelMu   sync.Mutex
	cstrings   []unsafe.Pointer
	cancelFlag unsafe.Pointer // *C.int, C-allocated
}

func New(opts ...Option) (*RawProcessor, error) {
	handle := C.libraw_init(0)
	if handle == nil {
		return nil, ErrInitFailed
	}

	flag := (*C.int)(C.malloc(C.size_t(unsafe.Sizeof(C.int(0)))))
	*flag = 0
	C.golibraw_register_cancel_cb(handle, flag)

	rp := &RawProcessor{handle: handle, cancelFlag: unsafe.Pointer(flag)}
	runtime.SetFinalizer(rp, (*RawProcessor).Close)

	cfg := defaultOptions()
	for _, o := range opts {
		o(&cfg)
	}
	rp.freeCStrings()
	applyConfigToHandle(rp.handle, &cfg, rp.trackCString)

	return rp, nil
}

// Close releases all resources. Idempotent.
func (rp *RawProcessor) Close() error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.closed {
		return nil
	}

	rp.closed = true
	runtime.SetFinalizer(rp, nil)
	rp.freeCStrings()
	rp.cancelMu.Lock()
	flag := rp.cancelFlag
	rp.cancelFlag = nil
	rp.cancelMu.Unlock()
	if flag != nil {
		C.free(flag)
	}
	if rp.dngHost != nil {
		C.golibraw_destroy_dng_host(rp.dngHost)
		rp.dngHost = nil
	}
	C.libraw_close(rp.handle)
	rp.handle = nil

	return nil
}

// Recycle resets internal state so the processor can be reused for another file.
func (rp *RawProcessor) Recycle() {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if !rp.closed && rp.handle != nil {
		rp.cancelMu.Lock()
		if rp.cancelFlag != nil {
			*(*C.int)(rp.cancelFlag) = 0
		}
		rp.cancelMu.Unlock()
		rp.freeCStrings()
		C.libraw_recycle(rp.handle)
	}
}

// Cancel aborts the current C operation (Process, Unpack, etc.).
// The C function will return an error shortly after this is called.
// Safe to call from any goroutine — does not acquire mu.
func (rp *RawProcessor) Cancel() {
	rp.cancelMu.Lock()
	defer rp.cancelMu.Unlock()
	if rp.cancelFlag != nil {
		*(*C.int)(rp.cancelFlag) = 1
	}
}

func (rp *RawProcessor) freeCStrings() {
	for _, p := range rp.cstrings {
		C.free(p)
	}
	rp.cstrings = nil
}

func (rp *RawProcessor) trackCString(s string) *C.char {
	cs := C.CString(s)
	rp.cstrings = append(rp.cstrings, unsafe.Pointer(cs))
	return cs
}

func (rp *RawProcessor) ensureOpen() error {
	if rp.closed || rp.handle == nil {
		return ErrAlreadyClosed
	}
	return nil
}

func librawError(rc C.int, wrap error) error {
	if rc == C.LIBRAW_SUCCESS {
		return nil
	}
	msg := C.GoString(C.libraw_strerror(rc))
	return fmt.Errorf("%w: %s (code %d)", wrap, msg, int(rc))
}

func Version() string {
	return C.GoString(C.libraw_version())
}

func VersionNumber() int {
	return int(C.libraw_versionNumber())
}

func CameraCount() int {
	return int(C.libraw_cameraCount())
}

func StrError(code int) string {
	return C.GoString(C.libraw_strerror(C.int(code)))
}

// EnableDNGSDK creates a DNG SDK dng_host and binds it to the processor.
// Requires USE_DNGSDK to be defined at compile time; otherwise a no-op.
func (rp *RawProcessor) EnableDNGSDK() error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return err
	}

	host := C.golibraw_create_dng_host()
	if host == nil {
		return nil
	}
	rp.dngHost = unsafe.Pointer(host)
	C.golibraw_set_dng_host_for_raw(rp.handle, rp.dngHost)

	return nil
}
