package golibraw

/*
#cgo pkg-config: libraw_r
#include <libraw/libraw.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

// 错误定义
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

// RawProcessor 封装 libraw_data_t 句柄，提供 RAW 图像处理能力。
type RawProcessor struct {
	handle   *C.libraw_data_t
	closed   bool
	mu       sync.Mutex
	cstrings []unsafe.Pointer // 跟踪 C 字符串分配，在 Close/Recycle 时释放
}

// New 创建一个新的 RawProcessor 实例。
func New() (*RawProcessor, error) {
	handle := C.libraw_init(0)
	if handle == nil {
		return nil, ErrInitFailed
	}

	rp := &RawProcessor{handle: handle}
	runtime.SetFinalizer(rp, (*RawProcessor).Close)

	return rp, nil
}

// Close 释放 RawProcessor 持有的所有资源。幂等，可安全多次调用。
func (rp *RawProcessor) Close() error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.closed {
		return nil
	}

	rp.closed = true
	runtime.SetFinalizer(rp, nil)
	rp.freeCStrings()
	C.libraw_close(rp.handle)
	rp.handle = nil

	return nil
}

// Recycle 重置内部状态以便复用同一 processor 处理不同文件。
func (rp *RawProcessor) Recycle() {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if !rp.closed && rp.handle != nil {
		rp.freeCStrings()
		C.libraw_recycle(rp.handle)
	}
}

// freeCStrings 释放通过 cstrings 跟踪的 C 内存。
func (rp *RawProcessor) freeCStrings() {
	for _, p := range rp.cstrings {
		C.free(p)
	}
	rp.cstrings = nil
}

// trackCString 分配 C 字符串并注册到 processor 的跟踪列表中。
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

// librawError 将 C 返回码转换为包含错误描述的 Go error。
func librawError(rc C.int, wrap error) error {
	if rc == C.LIBRAW_SUCCESS {
		return nil
	}
	msg := C.GoString(C.libraw_strerror(rc))
	return fmt.Errorf("%w: %s (code %d)", wrap, msg, int(rc))
}

// Version 返回 LibRaw 版本字符串。
func Version() string {
	return C.GoString(C.libraw_version())
}

// VersionNumber 返回 LibRaw 数字版本号。
func VersionNumber() int {
	return int(C.libraw_versionNumber())
}

// CameraCount 返回支持的相机数量。
func CameraCount() int {
	return int(C.libraw_cameraCount())
}

// StrError 将 LibRaw 错误码转换为人类可读的错误描述。
func StrError(code int) string {
	return C.GoString(C.libraw_strerror(C.int(code)))
}
