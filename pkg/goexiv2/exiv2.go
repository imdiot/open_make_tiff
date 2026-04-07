package goexiv2

/*
#cgo pkg-config: exiv2
#cgo CXXFLAGS: -std=c++17
#cgo darwin CFLAGS: -mmacosx-version-min=10.13
#cgo darwin CXXFLAGS: -mmacosx-version-min=10.13
#cgo !darwin LDFLAGS: -static-libstdc++ -static-libgcc

#include "wrapper.h"
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"unsafe"
)

// --- Tag types ---

// Tag holds raw metadata fields common to all families (EXIF, IPTC, XMP).
type Tag struct {
	TagID  uint16 // Numeric tag ID from file (0 for XMP)
	TypeID int    // Value type enum (exiv2 TypeId)
	Count  int    // Number of value components
	Size   int    // Byte length of value
	Key    string // Standard identifier (e.g. "Exif.Image.Make")
	Value  string // String representation (toString)
	Raw    []byte // Raw binary data (binary tags only)
}

// Int parses the value as int64.
func (t Tag) Int() (int64, error) {
	f, err := t.Float()
	if err != nil {
		return 0, err
	}
	return int64(f), nil
}

// Float parses the value as float64, handling rational "n/d" and multi-component formats.
func (t Tag) Float() (float64, error) {
	return parseFloatComponent(t.Value)
}

// Binary reports whether raw bytes are present.
func (t Tag) Binary() bool {
	return len(t.Raw) > 0
}

// ExifTag extends Tag with EXIF-specific raw fields.
type ExifTag struct {
	Tag
	IfdID int // IFD identifier (ifdId)
}

// IptcTag extends Tag with IPTC-specific raw fields.
type IptcTag struct {
	Tag
	Record uint16 // Record number from file
}

// XmpTag wraps Tag for XMP entries (TagID is always 0).
type XmpTag struct {
	Tag
}

// --- Metadata[T] generic container ---

// Metadata is a generic ordered container for tags of a single family.
type Metadata[T any] struct {
	tags []T
	keys []string       // parallel to tags, for Key(i) without interface constraint
	idx  map[string]int // key → tags index (first occurrence)
}

// newMetadata builds a Metadata from a tag slice, using keyFn to extract each tag's key.
func newMetadata[T any](tags []T, keyFn func(T) string) *Metadata[T] {
	m := &Metadata[T]{
		tags: tags,
		keys: make([]string, len(tags)),
		idx:  make(map[string]int, len(tags)),
	}
	for i, t := range tags {
		k := keyFn(t)
		m.keys[i] = k
		if _, exists := m.idx[k]; !exists {
			m.idx[k] = i
		}
	}
	return m
}

// Count returns the total number of tags.
func (m *Metadata[T]) Count() int {
	if m == nil {
		return 0
	}
	return len(m.tags)
}

// Key returns the key string at index i.
func (m *Metadata[T]) Key(i int) (string, error) {
	if m == nil || i < 0 || i >= len(m.keys) {
		return "", fmt.Errorf("goexiv2: key: index out of range")
	}
	return m.keys[i], nil
}

// Has reports whether the given key exists.
func (m *Metadata[T]) Has(key string) bool {
	if m == nil {
		return false
	}
	_, ok := m.idx[key]
	return ok
}

// Tag returns the tag entry for the given key.
func (m *Metadata[T]) Tag(key string) (T, bool) {
	if m == nil {
		var zero T
		return zero, false
	}
	i, ok := m.idx[key]
	if !ok {
		var zero T
		return zero, false
	}
	return m.tags[i], true
}

// Tags returns all tags in order.
func (m *Metadata[T]) Tags() []T {
	if m == nil {
		return nil
	}
	return m.tags
}

// --- C → Go conversion helpers ---

func convertBase(t C.GoExiv2Tag) Tag {
	return Tag{
		TagID:  uint16(t.tag),
		TypeID: int(t.type_id),
		Count:  int(t.count),
		Size:   int(t.size),
		Key:    C.GoString(t.key),
		Value:  C.GoString(t.value),
	}
}

func convertRaw(t C.GoExiv2Tag) []byte {
	if t.raw != nil && t.raw_len > 0 {
		return C.GoBytes(unsafe.Pointer(t.raw), t.raw_len)
	}
	return nil
}

func newExifTags(arr C.GoExiv2TagArray) []ExifTag {
	n := int(arr.count)
	if n == 0 || arr.tags == nil {
		return nil
	}
	out := make([]ExifTag, 0, n)
	for _, t := range unsafe.Slice(arr.tags, n) {
		base := convertBase(t)
		base.Raw = convertRaw(t)
		out = append(out, ExifTag{
			Tag:   base,
			IfdID: int(t.ifd_id),
		})
	}
	return out
}

func newIptcTags(arr C.GoExiv2TagArray) []IptcTag {
	n := int(arr.count)
	if n == 0 || arr.tags == nil {
		return nil
	}
	out := make([]IptcTag, 0, n)
	for _, t := range unsafe.Slice(arr.tags, n) {
		base := convertBase(t)
		base.Raw = convertRaw(t)
		out = append(out, IptcTag{
			Tag:    base,
			Record: uint16(t.record),
		})
	}
	return out
}

func newXmpTags(arr C.GoExiv2TagArray) []XmpTag {
	n := int(arr.count)
	if n == 0 || arr.tags == nil {
		return nil
	}
	out := make([]XmpTag, 0, n)
	for _, t := range unsafe.Slice(arr.tags, n) {
		base := convertBase(t)
		base.Raw = convertRaw(t)
		out = append(out, XmpTag{Tag: base})
	}
	return out
}

// parseFloatComponent parses the first component of an exif value:
//   - plain number: "2.8" → 2.8
//   - rational: "14/5" → 2.8
//   - multi-component: "40/1 30/1 0/1" → first component 40.0
func parseFloatComponent(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if sp := strings.IndexByte(s, ' '); sp >= 0 {
		s = s[:sp]
	}
	if idx := strings.IndexByte(s, '/'); idx >= 0 {
		num, err1 := strconv.ParseFloat(s[:idx], 64)
		den, err2 := strconv.ParseFloat(s[idx+1:], 64)
		if err1 != nil || err2 != nil || den == 0 {
			return 0, fmt.Errorf("invalid rational %q", s)
		}
		return num / den, nil
	}
	return strconv.ParseFloat(s, 64)
}

// --- Image ---

// Image wraps an exiv2 image handle for reading EXIF/IPTC/XMP metadata.
type Image struct {
	handle unsafe.Pointer
	closed bool
	mu     sync.Mutex

	EXIF      *Metadata[ExifTag]
	IPTC      *Metadata[IptcTag]
	XMP       *Metadata[XmpTag]
	XMPPacket string
}

// Open opens an image file by path (UTF-8) and returns an Image for metadata reading.
func Open(path string) (*Image, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	handle := C.goexiv2_open(cPath)
	if handle == nil {
		return nil, &OpenError{Path: path, Msg: goLastError()}
	}

	img := &Image{handle: handle}
	runtime.SetFinalizer(img, (*Image).Close)
	return img, nil
}

// Close releases the image handle. Safe to call multiple times.
func (img *Image) Close() error {
	img.mu.Lock()
	defer img.mu.Unlock()
	if img.closed {
		return nil
	}
	img.closed = true
	runtime.SetFinalizer(img, nil)
	C.goexiv2_close(img.handle)
	img.handle = nil
	img.EXIF = nil
	img.IPTC = nil
	img.XMP = nil
	img.XMPPacket = ""
	return nil
}

// ReadMetadata reads all metadata (EXIF, IPTC, XMP) from the image
// and populates EXIF, IPTC, XMP, and XMPPacket fields.
func (img *Image) ReadMetadata() error {
	img.mu.Lock()
	defer img.mu.Unlock()
	if img.closed {
		return ErrClosed
	}

	md := C.goexiv2_read_metadata(img.handle)
	if md == nil {
		return &ReadError{Op: "read_metadata", Msg: goLastError()}
	}
	defer C.goexiv2_metadata_free(md)

	img.EXIF = newMetadata(newExifTags(md.exif), func(t ExifTag) string { return t.Key })
	img.IPTC = newMetadata(newIptcTags(md.iptc), func(t IptcTag) string { return t.Key })
	img.XMP = newMetadata(newXmpTags(md.xmp), func(t XmpTag) string { return t.Key })
	if md.xmp_packet != nil {
		img.XMPPacket = C.GoString(md.xmp_packet)
	}
	return nil
}

// --- Utility ---

// Version returns the exiv2 library version string.
func Version() string {
	return C.GoString(C.goexiv2_version())
}

func goLastError() string {
	if s := C.goexiv2_get_last_error(); s != nil {
		return C.GoString(s)
	}
	return "unknown error"
}
