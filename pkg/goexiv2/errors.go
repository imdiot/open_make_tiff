package goexiv2

import (
	"errors"
	"fmt"
)

// ErrClosed is returned when attempting to use an image that has already been closed.
var ErrClosed = errors.New("goexiv2: image already closed")

// OpenError is returned when an image file cannot be opened.
type OpenError struct {
	Path string
	Msg  string
}

func (e *OpenError) Error() string {
	return fmt.Sprintf("goexiv2: open %q: %s", e.Path, e.Msg)
}

// ReadError is returned when metadata cannot be read from an image.
type ReadError struct {
	Op  string
	Msg string
}

func (e *ReadError) Error() string {
	return fmt.Sprintf("goexiv2: %s: %s", e.Op, e.Msg)
}
