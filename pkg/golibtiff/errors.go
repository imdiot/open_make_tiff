package golibtiff

import "fmt"

type OpenError struct {
	Path string
	Mode OpenMode
	Msg  string
}

func (e *OpenError) Error() string {
	if e.Msg != "" {
		return fmt.Sprintf("libtiff: open %q mode %q: %s", e.Path, e.Mode, e.Msg)
	}
	return fmt.Sprintf("libtiff: failed to open %q with mode %q", e.Path, e.Mode)
}

type FieldError struct {
	Tag Tag
	Op  string
	Msg string
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("libtiff: %s field %d: %s", e.Op, uint32(e.Tag), e.Msg)
}

type ReadError struct {
	Op  string
	Msg string
}

func (e *ReadError) Error() string {
	return fmt.Sprintf("libtiff: read %s: %s", e.Op, e.Msg)
}

type WriteError struct {
	Op  string
	Msg string
}

func (e *WriteError) Error() string {
	return fmt.Sprintf("libtiff: write %s: %s", e.Op, e.Msg)
}
