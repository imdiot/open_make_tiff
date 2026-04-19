package golibraw

import (
	"errors"
	"fmt"
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
	ErrInvalidIndex   = &Error{Op: "index"}
)
