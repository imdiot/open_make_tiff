package golibraw

/*
#cgo pkg-config: libraw_r
#cgo CXXFLAGS: -std=c++14
#cgo darwin CFLAGS: -mmacosx-version-min=10.13
#cgo darwin CXXFLAGS: -mmacosx-version-min=10.13
#cgo darwin LDFLAGS: -framework CoreServices
#cgo linux LDFLAGS: -lstdc++
#cgo windows LDFLAGS: -static-libgcc -Wl,-Bstatic -lstdc++ -lwinpthread
#include <libraw/libraw.h>
#include <stdlib.h>

extern void* golibraw_create_dng_host();
extern void golibraw_destroy_dng_host(void* host);
extern void golibraw_set_dng_host_for_raw(libraw_data_t* lr, void* host);

// C++ bridge functions (bridge.cpp)
extern int golibraw_is_fuji_rotated(libraw_data_t* lr);
extern int golibraw_is_sraw(libraw_data_t* lr);
extern int golibraw_sraw_midpoint(libraw_data_t* lr);
extern int golibraw_is_nikon_sraw(libraw_data_t* lr);
extern int golibraw_is_coolscan_nef(libraw_data_t* lr);
extern int golibraw_is_jpeg_thumb(libraw_data_t* lr);
extern int golibraw_is_floating_point(libraw_data_t* lr);
extern int golibraw_have_fpdata(libraw_data_t* lr);
extern int golibraw_error_count(libraw_data_t* lr);
extern int golibraw_thumb_ok(libraw_data_t* lr, long long maxsz);
extern int golibraw_raw_was_read(libraw_data_t* lr);
extern int golibraw_color(libraw_data_t* lr, int row, int col);
extern int golibraw_fc(libraw_data_t* lr, int row, int col);
extern int golibraw_fcol(libraw_data_t* lr, int row, int col);
extern int golibraw_adjust_maximum(libraw_data_t* lr);
extern int golibraw_raw2image_ex(libraw_data_t* lr, int do_subtract_black);
extern void golibraw_convert_float_to_int(libraw_data_t* lr, float dmin, float dmax, float dtarget);
extern void golibraw_get_mem_image_format(libraw_data_t* lr, int* width, int* height, int* colors, int* bps);
extern int golibraw_copy_mem_image(libraw_data_t* lr, void* scan0, int stride, int bgr);
extern int golibraw_set_make_from_index(libraw_data_t* lr, unsigned index);
extern int golibraw_set_rawspeed_camerafile(libraw_data_t* lr, char* filename);

static int golibraw_progress_cb(void* data, enum LibRaw_progress stage, int iteration, int expected) {
	return *((int*)data);
}

static void golibraw_register_cancel_cb(libraw_data_t* lr, int* flag) {
	libraw_set_progress_handler(lr, golibraw_progress_cb, flag);
}
*/
import "C"

import (
	"runtime"
	"sync"
	"unsafe"
)

func checkError(rc C.int, sentinel *Error) error {
	if rc == C.LIBRAW_SUCCESS {
		return nil
	}
	return &Error{
		Op:      sentinel.Op,
		Code:    int(rc),
		Message: cGoString(C.libraw_strerror(rc)),
	}
}

// rawRes holds C resources that must be freed when RawProcessor is collected.
type rawRes struct {
	handle     *C.libraw_data_t
	cancelFlag unsafe.Pointer
	dngHost    unsafe.Pointer
	cstrings   []unsafe.Pointer
	cbKey      callbackKey
}

// RawProcessor wraps libraw_data_t for RAW image processing.
type RawProcessor struct {
	res      *rawRes
	closed   bool
	mu       sync.Mutex
	cancelMu sync.Mutex
	cleanup  runtime.Cleanup
}

func New(opts ...Option) (*RawProcessor, error) {
	handle := C.libraw_init(0)
	if handle == nil {
		return nil, ErrInitFailed
	}

	flag := (*C.int)(C.malloc(C.size_t(unsafe.Sizeof(C.int(0)))))
	if flag == nil {
		C.libraw_close(handle)
		return nil, ErrInitFailed
	}
	*flag = 0
	C.golibraw_register_cancel_cb(handle, flag)

	rp := &RawProcessor{res: &rawRes{
		handle:     handle,
		cancelFlag: unsafe.Pointer(flag),
	}}
	cbKey := C.malloc(1)
	if cbKey == nil {
		C.libraw_close(handle)
		C.free(unsafe.Pointer(flag))
		return nil, ErrInitFailed
	}
	rp.res.cbKey = callbackKey(cbKey)
	rp.cleanup = runtime.AddCleanup(rp, func(r *rawRes) {
		unregisterCallback(r.cbKey)
		C.free(r.cbKey)
		for _, p := range r.cstrings {
			C.free(p)
		}
		if r.dngHost != nil {
			C.golibraw_destroy_dng_host(r.dngHost)
		}
		if r.cancelFlag != nil {
			C.free(r.cancelFlag)
		}
		C.libraw_close(r.handle)
	}, rp.res)

	cfg := defaultOptions()
	for _, o := range opts {
		o(&cfg)
	}
	rp.freeCStrings()
	applyConfigToHandle(rp.res.handle, &cfg, rp.trackCString)

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
	rp.cleanup.Stop()
	unregisterCallback(rp.res.cbKey)
	C.free(rp.res.cbKey)
	rp.res.cbKey = nil
	rp.freeCStrings()
	rp.cancelMu.Lock()
	flag := rp.res.cancelFlag
	rp.res.cancelFlag = nil
	rp.cancelMu.Unlock()
	if flag != nil {
		C.free(flag)
	}
	if rp.res.dngHost != nil {
		C.golibraw_destroy_dng_host(rp.res.dngHost)
		rp.res.dngHost = nil
	}
	C.libraw_close(rp.res.handle)
	rp.res.handle = nil

	return nil
}

// Recycle resets internal state so the processor can be reused for another file.
func (rp *RawProcessor) Recycle() {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if !rp.closed && rp.res.handle != nil {
		rp.cancelMu.Lock()
		if rp.res.cancelFlag != nil {
			*(*C.int)(rp.res.cancelFlag) = 0
		}
		rp.cancelMu.Unlock()
		rp.freeCStrings()
		C.libraw_recycle(rp.res.handle)
	}
}

// Cancel aborts the current C operation (Process, Unpack, etc.).
// The C function will return an error shortly after this is called.
// Safe to call from any goroutine — does not acquire mu.
func (rp *RawProcessor) Cancel() {
	rp.cancelMu.Lock()
	defer rp.cancelMu.Unlock()
	if rp.res.cancelFlag != nil {
		*(*C.int)(rp.res.cancelFlag) = 1
	}
}

func (rp *RawProcessor) freeCStrings() {
	for _, p := range rp.res.cstrings {
		C.free(p)
	}
	rp.res.cstrings = nil
	if rp.res.handle != nil {
		rp.res.handle.params.output_profile = nil
		rp.res.handle.params.camera_profile = nil
		rp.res.handle.params.bad_pixels = nil
		rp.res.handle.params.dark_frame = nil
	}
}

func (rp *RawProcessor) trackCString(s string) *C.char {
	cs := C.CString(s)
	rp.res.cstrings = append(rp.res.cstrings, unsafe.Pointer(cs))
	return cs
}

func (rp *RawProcessor) ensureOpen() error {
	if rp.closed || rp.res.handle == nil {
		return ErrAlreadyClosed
	}
	return nil
}

// isOpen returns true if the processor can be used.
func (rp *RawProcessor) isOpen() bool {
	return !rp.closed && rp.res.handle != nil
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
	rp.res.dngHost = unsafe.Pointer(host)
	C.golibraw_set_dng_host_for_raw(rp.res.handle, rp.res.dngHost)

	return nil
}
