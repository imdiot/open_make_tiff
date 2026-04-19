package golibraw

/*
#include <libraw/libraw.h>

extern int golibraw_adjust_maximum(libraw_data_t* lr);
extern int golibraw_raw2image_ex(libraw_data_t* lr, int do_subtract_black);
extern void golibraw_convert_float_to_int(libraw_data_t* lr, float dmin, float dmax, float dtarget);
extern void golibraw_get_mem_image_format(libraw_data_t* lr, int* width, int* height, int* colors, int* bps);
extern int golibraw_copy_mem_image(libraw_data_t* lr, void* scan0, int stride, int bgr);
extern int golibraw_set_make_from_index(libraw_data_t* lr, unsigned index);
extern int golibraw_set_rawspeed_camerafile(libraw_data_t* lr, char* filename);
*/
import "C"
import (
	"runtime"
	"unsafe"
)

// Raw2Image converts RAW data to a 4-channel image without running the full processing pipeline.
func (rp *RawProcessor) Raw2Image() error {
	return rp.Raw2ImageEx(true)
}

// Raw2ImageEx converts RAW data to a 4-channel image without running the full processing pipeline.
// If subBlack is true, subtracts black level during conversion.
func (rp *RawProcessor) Raw2ImageEx(subBlack bool) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if err := rp.ensureOpen(); err != nil {
		return err
	}
	sub := 0
	if subBlack {
		sub = 1
	}
	rc := C.golibraw_raw2image_ex(rp.res.handle, C.int(sub))
	return checkError(rc, ErrProcess)
}

// SubtractBlack subtracts black level from image data.
func (rp *RawProcessor) SubtractBlack() error {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if err := rp.ensureOpen(); err != nil {
		return err
	}
	C.libraw_subtract_black(rp.res.handle)
	return nil
}

// AdjustMaximum adjusts the maximum value after subtracting black.
func (rp *RawProcessor) AdjustMaximum() error {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if err := rp.ensureOpen(); err != nil {
		return err
	}
	rc := C.golibraw_adjust_maximum(rp.res.handle)
	return checkError(rc, ErrProcess)
}

// AdjustSizesInfoOnly recalculates sizes without processing.
func (rp *RawProcessor) AdjustSizesInfoOnly() error {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if err := rp.ensureOpen(); err != nil {
		return err
	}
	rc := C.libraw_adjust_sizes_info_only(rp.res.handle)
	return checkError(rc, ErrProcess)
}

// FreeImage releases memory allocated by Raw2Image/Raw2ImageEx.
func (rp *RawProcessor) FreeImage() {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if rp.closed || rp.res.handle == nil {
		return
	}
	C.libraw_free_image(rp.res.handle)
}

// ConvertFloatToInt converts floating-point RAW data to integer.
// dmin/dmax define the input range, dtarget is the target midpoint.
func (rp *RawProcessor) ConvertFloatToInt(dmin, dmax, dtarget float32) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if rp.closed || rp.res.handle == nil {
		return
	}
	C.golibraw_convert_float_to_int(rp.res.handle, C.float(dmin), C.float(dmax), C.float(dtarget))
}

// RecycleDatastream recycles internal state but keeps the data stream open.
func (rp *RawProcessor) RecycleDatastream() {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if rp.closed || rp.res.handle == nil {
		return
	}
	C.libraw_recycle_datastream(rp.res.handle)
}

// GetMemImageFormat returns the output format without allocating memory.
func (rp *RawProcessor) GetMemImageFormat() (width, height, colors, bps int) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if rp.closed || rp.res.handle == nil {
		return 0, 0, 0, 0
	}
	var w, h, c, b C.int
	C.golibraw_get_mem_image_format(rp.res.handle, &w, &h, &c, &b)
	return int(w), int(h), int(c), int(b)
}

// CopyMemImage copies processed image directly into the provided buffer.
// stride is the number of bytes per row. If bgr is true, swaps R and B channels.
func (rp *RawProcessor) CopyMemImage(scan0 []byte, stride int, bgr bool) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if err := rp.ensureOpen(); err != nil {
		return err
	}
	if len(scan0) == 0 {
		return ErrMemImage
	}
	if stride <= 0 || stride > 0x7FFFFFFF {
		return ErrMemImage
	}
	b := 0
	if bgr {
		b = 1
	}
	rc := C.golibraw_copy_mem_image(rp.res.handle, unsafe.Pointer(&scan0[0]), C.int(stride), C.int(b))
	runtime.KeepAlive(scan0)
	return checkError(rc, ErrMemImage)
}

// SetMakeFromIndex sets camera make/model from the manufacturer index.
func (rp *RawProcessor) SetMakeFromIndex(index uint) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if err := rp.ensureOpen(); err != nil {
		return err
	}
	rc := C.golibraw_set_make_from_index(rp.res.handle, C.uint(index))
	return checkError(rc, ErrProcess)
}

// SetRawSpeedCameraFile sets a custom RawSpeed camera configuration file.
func (rp *RawProcessor) SetRawSpeedCameraFile(path string) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if err := rp.ensureOpen(); err != nil {
		return err
	}
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	rc := C.golibraw_set_rawspeed_camerafile(rp.res.handle, cPath)
	return checkError(rc, ErrProcess)
}
