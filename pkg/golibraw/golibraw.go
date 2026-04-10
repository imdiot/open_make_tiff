package golibraw

/*
#cgo pkg-config: libraw_r
#cgo CXXFLAGS: -std=c++14
#cgo darwin CFLAGS: -mmacosx-version-min=10.13
#cgo darwin CXXFLAGS: -mmacosx-version-min=10.13
#cgo darwin LDFLAGS: -framework CoreServices
#cgo !darwin LDFLAGS: -lstdc++
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
	"errors"
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

// Error represents a LibRaw operation failure with error code and message.
// Use errors.Is(err, ErrUnpack) to check the operation category,
// or errors.As(err, &lrErr) to access Code and Message.
type Error struct {
	Op      string // operation that failed (e.g. "unpack", "process")
	Code    int    // LibRaw error code (negative values, see ErrCode* constants)
	Message string // human-readable message from libraw_strerror
}

func (e *Error) Error() string { return fmt.Sprintf("libraw: %s failed: %s (code %d)", e.Op, e.Message, e.Code) }

func (e *Error) Is(target error) bool {
	if t, ok := target.(*Error); ok {
		return e.Op == t.Op
	}
	return false
}

// Sentinel errors for errors.Is matching. Each carries the operation name as Op.
var (
	ErrInitFailed     = &Error{Op: "init"}
	ErrAlreadyClosed  = errors.New("libraw: processor already closed")
	ErrFileOpenFailed = &Error{Op: "open_file"}
	ErrBufferOpen     = &Error{Op: "open_buffer"}
	ErrUnpack         = &Error{Op: "unpack"}
	ErrUnpackThumb    = &Error{Op: "unpack_thumb"}
	ErrProcess        = &Error{Op: "process"}
	ErrMemImage       = &Error{Op: "dcraw_process"}
	ErrWriteFailed    = &Error{Op: "write"}
	ErrBadCrop        = &Error{Op: "crop"}
)

func checkError(rc C.int, sentinel *Error) error {
	if rc == C.LIBRAW_SUCCESS {
		return nil
	}
	return &Error{
		Op:      sentinel.Op,
		Code:    int(rc),
		Message: C.GoString(C.libraw_strerror(rc)),
	}
}

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
	if flag == nil {
		C.libraw_close(handle)
		return nil, ErrInitFailed
	}
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
	unregisterCallback(callbackKey(unsafe.Pointer(rp)))
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
	if rp.handle != nil {
		rp.handle.params.output_profile = nil
		rp.handle.params.camera_profile = nil
		rp.handle.params.bad_pixels = nil
		rp.handle.params.dark_frame = nil
	}
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

// isOpen returns true if the processor can be used.
func (rp *RawProcessor) isOpen() bool {
	return !rp.closed && rp.handle != nil
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
