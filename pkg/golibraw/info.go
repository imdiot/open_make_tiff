package golibraw

/*
#include <libraw/libraw.h>
*/
import "C"
import "unsafe"

// LibRaw error codes (enum LibRaw_errors).
const (
	ErrCodeSuccess                      = 0
	ErrCodeUnspecified                  = -1
	ErrCodeFileUnsupported              = -2
	ErrCodeNonexistentImage             = -3
	ErrCodeOutOfOrderCall               = -4
	ErrCodeNoThumbnail                  = -5
	ErrCodeUnsupportedThumbnail         = -6
	ErrCodeInputClosed                  = -7
	ErrCodeNotImplemented               = -8
	ErrCodeNonexistentThumbnail         = -9
	ErrCodeInsufficientMemory           = -100007
	ErrCodeDataError                    = -100008
	ErrCodeIOError                      = -100009
	ErrCodeCancelledByCallback          = -100010
	ErrCodeBadCrop                      = -100011
	ErrCodeTooBig                       = -100012
	ErrCodeMemPoolOverflow              = -100013
)

// cGoString safely converts a C string to Go, returning "" for nil.
func cGoString(s *C.char) string {
	if s == nil {
		return ""
	}
	return C.GoString(s)
}

// GetDecoderInfo returns information about the decoder used for the current image.
func (rp *RawProcessor) GetDecoderInfo() (DecoderInfo, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if err := rp.ensureOpen(); err != nil {
		return DecoderInfo{}, err
	}
	var di C.libraw_decoder_info_t
	rc := C.libraw_get_decoder_info(rp.res.handle, &di)
	if rc != C.LIBRAW_SUCCESS {
		return DecoderInfo{}, checkError(rc, ErrProcess)
	}
	return DecoderInfo{
		DecoderName:  cGoString(di.decoder_name),
		DecoderFlags: uint(di.decoder_flags),
	}, nil
}

// UnpackFunctionName returns the name of the unpacking function used.
func (rp *RawProcessor) UnpackFunctionName() (string, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if err := rp.ensureOpen(); err != nil {
		return "", err
	}
	return cGoString(C.libraw_unpack_function_name(rp.res.handle)), nil
}

// Capabilities returns the runtime capabilities bitmask.
func Capabilities() uint {
	return uint(C.libraw_capabilities())
}

// CameraList returns the list of supported cameras.
func CameraList() []string {
	count := int(C.libraw_cameraCount())
	if count <= 0 {
		return nil
	}
	list := C.libraw_cameraList()
	if list == nil {
		return nil
	}
	result := make([]string, count)
	for i := range count {
		result[i] = C.GoString(*(**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(list)) + uintptr(i)*unsafe.Sizeof(*list))))
	}
	return result
}

// StrProgress returns the description of a processing stage.
func StrProgress(stage int) string {
	return cGoString(C.libraw_strprogress(C.enum_LibRaw_progress(stage)))
}

func Version() string {
	return cGoString(C.libraw_version())
}

func VersionNumber() int {
	return int(C.libraw_versionNumber())
}

func CameraCount() int {
	return int(C.libraw_cameraCount())
}

func StrError(code int) string {
	return cGoString(C.libraw_strerror(C.int(code)))
}
