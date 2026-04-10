package golibraw

/*
#include <libraw/libraw.h>
*/
import "C"

import "unsafe"

// RawImageData holds pointers to the various RAW data representations.
// The returned slices share C memory and become invalid after any subsequent
// processor operation (Process, Recycle, Close, etc.).
type RawImageData struct {
	RawImage    []uint16
	Color4Image []uint16 // row-major, 4 channels per pixel
	Color3Image []uint16 // row-major, 3 channels per pixel
	FloatImage  []float32
	Float3Image []float32 // row-major, 3 channels per pixel
	Float4Image []float32 // row-major, 4 channels per pixel
}

// GetRawData returns zero-copy views into the internal RAW data buffers.
// At most one slice will be non-nil, depending on the image format.
// The caller must not hold the returned slices across processor operations.
func (rp *RawProcessor) GetRawData() (RawImageData, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return RawImageData{}, err
	}

	rd := rp.handle.rawdata
	pixels := int(rd.sizes.raw_width) * int(rd.sizes.raw_height)
	if pixels <= 0 {
		return RawImageData{}, nil
	}

	var result RawImageData

	if rd.raw_image != nil {
		result.RawImage = unsafe.Slice((*uint16)(unsafe.Pointer(rd.raw_image)), pixels)
	} else if rd.color4_image != nil {
		result.Color4Image = unsafe.Slice((*uint16)(unsafe.Pointer(rd.color4_image)), pixels*4)
	} else if rd.color3_image != nil {
		result.Color3Image = unsafe.Slice((*uint16)(unsafe.Pointer(rd.color3_image)), pixels*3)
	} else if rd.float_image != nil {
		result.FloatImage = unsafe.Slice((*float32)(unsafe.Pointer(rd.float_image)), pixels)
	} else if rd.float3_image != nil {
		result.Float3Image = unsafe.Slice((*float32)(unsafe.Pointer(rd.float3_image)), pixels*3)
	} else if rd.float4_image != nil {
		result.Float4Image = unsafe.Slice((*float32)(unsafe.Pointer(rd.float4_image)), pixels*4)
	}

	return result, nil
}
