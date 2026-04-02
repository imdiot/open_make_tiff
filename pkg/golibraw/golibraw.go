package golibraw

/*
#cgo pkg-config: libraw_r
#cgo CXXFLAGS: -std=c++17 -DUSE_DNGSDK
#cgo LDFLAGS: -lstdc++
#include <libraw/libraw.h>

extern void* golibraw_create_dng_host();
extern void golibraw_destroy_dng_host(void* host);
extern void golibraw_set_dng_host_for_raw(libraw_data_t* lr, void* host);
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
	handle   *C.libraw_data_t
	dngHost  unsafe.Pointer
	closed   bool
	mu       sync.Mutex
	cstrings []unsafe.Pointer
}

func New(opts ...Option) (*RawProcessor, error) {
	handle := C.libraw_init(0)
	if handle == nil {
		return nil, ErrInitFailed
	}

	rp := &RawProcessor{handle: handle}
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
		rp.freeCStrings()
		C.libraw_recycle(rp.handle)
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
