package golibraw

/*
#include <libraw/libraw.h>
*/
import "C"
import "unsafe"

func (rp *RawProcessor) OpenBuffer(data []byte) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return err
	}

	rc := C.libraw_open_buffer(rp.handle, unsafe.Pointer(&data[0]), C.size_t(len(data)))
	return librawError(rc, ErrBufferOpen)
}

// Unpack unpacks raw data. Must be called after OpenFile/OpenBuffer.
func (rp *RawProcessor) Unpack() error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return err
	}

	rc := C.libraw_unpack(rp.handle)
	return librawError(rc, ErrUnpack)
}

func (rp *RawProcessor) UnpackThumb() error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return err
	}

	rc := C.libraw_unpack_thumb(rp.handle)
	return librawError(rc, ErrUnpackThumb)
}

// Process runs dcraw-style processing. Must be called after Unpack.
func (rp *RawProcessor) Process() error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return err
	}

	rc := C.libraw_dcraw_process(rp.handle)
	return librawError(rc, ErrProcess)
}

// MakeMemImage converts processed image to memory. Must be called after Process.
func (rp *RawProcessor) MakeMemImage() (*ProcessedImage, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return nil, err
	}

	var errc C.int
	img := C.libraw_dcraw_make_mem_image(rp.handle, &errc)
	if img == nil {
		return nil, librawError(errc, ErrMemImage)
	}
	defer C.libraw_dcraw_clear_mem(img)

	return copyProcessedImage(img)
}

// MakeMemThumb converts thumbnail to memory. Must be called after UnpackThumb.
func (rp *RawProcessor) MakeMemThumb() (*ProcessedImage, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return nil, err
	}

	var errc C.int
	img := C.libraw_dcraw_make_mem_thumb(rp.handle, &errc)
	if img == nil {
		return nil, librawError(errc, ErrMemImage)
	}
	defer C.libraw_dcraw_clear_mem(img)

	return copyProcessedImage(img)
}

// WritePPMTiff writes processed image as PPM/TIFF. Must be called after Process.
func (rp *RawProcessor) WritePPMTiff(outputPath string) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return err
	}

	cPath := C.CString(outputPath)
	defer C.free(unsafe.Pointer(cPath))

	rc := C.libraw_dcraw_ppm_tiff_writer(rp.handle, cPath)
	return librawError(rc, ErrWriteFailed)
}

// WriteThumb writes thumbnail to file. Must be called after UnpackThumb.
func (rp *RawProcessor) WriteThumb(outputPath string) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return err
	}

	cPath := C.CString(outputPath)
	defer C.free(unsafe.Pointer(cPath))

	rc := C.libraw_dcraw_thumb_writer(rp.handle, cPath)
	return librawError(rc, ErrWriteFailed)
}

func copyProcessedImage(img *C.libraw_processed_image_t) (*ProcessedImage, error) {
	dataSize := C.uint(img.data_size)
	if dataSize == 0 {
		return nil, ErrMemImage
	}

	data := C.GoBytes(unsafe.Pointer(&img.data[0]), C.int(dataSize))

	return &ProcessedImage{
		Type:   ImageFormat(img._type),
		Width:  uint16(img.width),
		Height: uint16(img.height),
		Colors: uint16(img.colors),
		Bits:   uint16(img.bits),
		Data:   data,
	}, nil
}
