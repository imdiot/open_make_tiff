package golibtiff

/*
#include <tiffio.h>
#include <stdlib.h>
#include "libtiff_bridge.h"
*/
import "C"

import (
	"errors"
	"fmt"
)

// --- Directory Operations ---

func (t *TIFF) NumberOfDirectories() uint32 {
	if err := t.checkOpen(); err != nil {
		return 0
	}
	return uint32(C.TIFFNumberOfDirectories(t.tif))
}

func (t *TIFF) CurrentDirectory() uint32 {
	if err := t.checkOpen(); err != nil {
		return 0
	}
	return uint32(C.TIFFCurrentDirectory(t.tif))
}

func (t *TIFF) SetDirectory(dirnum uint32) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.TIFFSetDirectory(t.tif, C.tdir_t(dirnum)) == 0 {
		if err := t.lastError(); err != nil {
			return err
		}
		return errors.New("libtiff: failed to set directory")
	}
	return nil
}

// ReadDirectory reads the next directory. Returns true if a directory was read.
func (t *TIFF) ReadDirectory() bool {
	if err := t.checkOpen(); err != nil {
		return false
	}
	return C.TIFFReadDirectory(t.tif) != 0
}

func (t *TIFF) WriteDirectory() error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.TIFFWriteDirectory(t.tif) == 0 {
		if err := t.lastError(); err != nil {
			return err
		}
		return errors.New("libtiff: failed to write directory")
	}
	return nil
}

// CheckpointDirectory writes the current IFD state to disk without closing it.
// This is needed before creating EXIF sub-IFDs.
func (t *TIFF) CheckpointDirectory() error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffCheckpointDirectory(t.tif) == 0 {
		if err := t.lastError(); err != nil {
			return err
		}
		return errors.New("libtiff: failed to checkpoint directory")
	}
	return nil
}

func (t *TIFF) SetSubDirectory(offset uint64) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.TIFFSetSubDirectory(t.tif, C.uint64_t(offset)) == 0 {
		if err := t.lastError(); err != nil {
			return err
		}
		return errors.New("libtiff: failed to set subdirectory")
	}
	return nil
}

// ReadEXIFDirectory reads the EXIF Sub-IFD at the given offset.
// Unlike SetSubDirectory, this does not require ImageLength/ImageWidth.
func (t *TIFF) ReadEXIFDirectory(offset uint64) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffReadEXIFDirectory(t.tif, C.uint64_t(offset)) == 0 {
		if err := t.lastError(); err != nil {
			return err
		}
		return errors.New("libtiff: failed to read EXIF directory")
	}
	return nil
}

func (t *TIFF) LastDirectory() bool {
	if err := t.checkOpen(); err != nil {
		return false
	}
	return C.TIFFLastDirectory(t.tif) != 0
}

// --- EXIF Sub-IFD Operations ---

// CreateEXIFDirectory creates a new EXIF Sub-IFD.
func (t *TIFF) CreateEXIFDirectory() error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffCreateEXIFDirectory(t.tif) != 0 {
		if err := t.lastError(); err != nil {
			return fmt.Errorf("libtiff: CreateEXIFDirectory: %w", err)
		}
		return errors.New("libtiff: CreateEXIFDirectory failed")
	}
	return nil
}

// WriteCustomDirectory writes the current directory as a custom (unlinked) IFD
// and returns its byte offset. Used for writing EXIF Sub-IFDs.
func (t *TIFF) WriteCustomDirectory() (uint64, error) {
	if err := t.checkOpen(); err != nil {
		return 0, err
	}
	C.clearHandleError(t.tif)
	var offset C.uint64_t
	if C.tiffWriteCustomDirectory(t.tif, &offset) == 0 {
		if err := t.lastError(); err != nil {
			return 0, fmt.Errorf("libtiff: WriteCustomDirectory: %w", err)
		}
		return 0, errors.New("libtiff: WriteCustomDirectory failed")
	}
	return uint64(offset), nil
}

// --- GPS Sub-IFD Operations ---

// CreateGPSDirectory creates a new GPS Sub-IFD.
func (t *TIFF) CreateGPSDirectory() error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffCreateGPSDirectory(t.tif) != 0 {
		if err := t.lastError(); err != nil {
			return fmt.Errorf("libtiff: CreateGPSDirectory: %w", err)
		}
		return errors.New("libtiff: CreateGPSDirectory failed")
	}
	return nil
}

// --- Extended Directory/IFD operations ---

// ReadGPSDirectory reads the GPS Sub-IFD at the given byte offset.
func (t *TIFF) ReadGPSDirectory(offset uint64) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffReadGPSDirectory(t.tif, C.uint64_t(offset)) == 0 {
		if err := t.lastError(); err != nil {
			return err
		}
		return errors.New("libtiff: failed to read GPS directory")
	}
	return nil
}

// CreateDirectory creates a new blank IFD and switches to it.
func (t *TIFF) CreateDirectory() error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffCreateDirectory(t.tif) != 0 {
		if err := t.lastError(); err != nil {
			return err
		}
		return errors.New("libtiff: failed to create directory")
	}
	return nil
}

// FreeDirectory releases internal data associated with the current IFD.
func (t *TIFF) FreeDirectory() {
	if err := t.checkOpen(); err != nil {
		return
	}
	C.TIFFFreeDirectory(t.tif)
}

// RewriteDirectory rewrites the directory at the end of the file.
// Useful for updating an existing TIFF in-place.
func (t *TIFF) RewriteDirectory() error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffRewriteDirectory(t.tif) == 0 {
		if err := t.lastError(); err != nil {
			return err
		}
		return errors.New("libtiff: failed to rewrite directory")
	}
	return nil
}

// UnlinkDirectory removes the IFD at the given index from the directory chain.
func (t *TIFF) UnlinkDirectory(dirNum uint16) error {
	if err := t.checkOpen(); err != nil {
		return err
	}
	C.clearHandleError(t.tif)
	if C.tiffUnlinkDirectory(t.tif, C.uint16_t(dirNum)) == 0 {
		if err := t.lastError(); err != nil {
			return err
		}
		return errors.New("libtiff: failed to unlink directory")
	}
	return nil
}

// --- Tag enumeration ---

// TagListCount returns the number of tags defined in the current IFD.
func (t *TIFF) TagListCount() int {
	if err := t.checkOpen(); err != nil {
		return 0
	}
	return int(C.TIFFGetTagListCount(t.tif))
}

// TagListEntry returns the tag number at the given index.
func (t *TIFF) TagListEntry(index int) Tag {
	if err := t.checkOpen(); err != nil {
		return 0
	}
	return Tag(C.TIFFGetTagListEntry(t.tif, C.int(index)))
}
