package goexiv2

import (
	"errors"
	"fmt"
)

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

// TagError is returned when a specific metadata tag cannot be accessed.
type TagError struct {
	Key string
	Op  string
	Msg string
}

func (e *TagError) Error() string {
	return fmt.Sprintf("goexiv2: %s %s: %s", e.Op, e.Key, e.Msg)
}
