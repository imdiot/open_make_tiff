package golibraw

/*
#include <libraw/libraw.h>
*/
import "C"
import (
	"runtime"
	"unsafe"
)

func (rp *RawProcessor) OpenBuffer(data []byte) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return err
	}

	if len(data) == 0 {
		return ErrBufferOpen
	}

	rc := C.libraw_open_buffer(rp.res.handle, unsafe.Pointer(&data[0]), C.size_t(len(data)))
	runtime.KeepAlive(data)
	return checkError(rc, ErrBufferOpen)
}

// Unpack unpacks raw data. Must be called after OpenFile/OpenBuffer.
func (rp *RawProcessor) Unpack() error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return err
	}

	rc := C.libraw_unpack(rp.res.handle)
	return checkError(rc, ErrUnpack)
}

func (rp *RawProcessor) UnpackThumb() error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return err
	}

	rc := C.libraw_unpack_thumb(rp.res.handle)
	return checkError(rc, ErrUnpackThumb)
}

// Process runs dcraw-style processing. Must be called after Unpack.
func (rp *RawProcessor) Process() error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return err
	}

	rc := C.libraw_dcraw_process(rp.res.handle)
	return checkError(rc, ErrProcess)
}

// MakeMemImage converts processed image to memory. Must be called after Process.
func (rp *RawProcessor) MakeMemImage() (*ProcessedImage, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return nil, err
	}

	var errc C.int
	img := C.libraw_dcraw_make_mem_image(rp.res.handle, &errc)
	if img == nil {
		return nil, checkError(errc, ErrMemImage)
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
	img := C.libraw_dcraw_make_mem_thumb(rp.res.handle, &errc)
	if img == nil {
		return nil, checkError(errc, ErrMemImage)
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

	rc := C.libraw_dcraw_ppm_tiff_writer(rp.res.handle, cPath)
	return checkError(rc, ErrWriteFailed)
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

	rc := C.libraw_dcraw_thumb_writer(rp.res.handle, cPath)
	return checkError(rc, ErrWriteFailed)
}

// GetThumbnailList returns all available thumbnails.
func (rp *RawProcessor) GetThumbnailList() ([]ThumbnailItem, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return nil, err
	}

	tl := rp.res.handle.thumbs_list
	count := int(tl.thumbcount)
	if count <= 0 {
		return nil, nil
	}
	if count > 8 {
		count = 8
	}

	result := make([]ThumbnailItem, count)
	for i := 0; i < count; i++ {
		t := tl.thumblist[i]
		result[i] = ThumbnailItem{
			Format: ThumbnailFormat(t.tformat),
			Width:  uint16(t.twidth),
			Height: uint16(t.theight),
			Flip:   uint16(t.tflip),
			Length: uint(t.tlength),
			Misc:   uint(t.tmisc),
			Offset: int64(t.toffset),
		}
	}
	return result, nil
}

// UnpackThumbAt unpacks the thumbnail at the given index (0-based).
func (rp *RawProcessor) UnpackThumbAt(index int) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return err
	}

	rc := C.libraw_unpack_thumb_ex(rp.res.handle, C.int(index))
	return checkError(rc, ErrUnpackThumb)
}

// OpenBayer opens raw Bayer data directly (not a standard RAW file).
// procflags: processing flags, bayerPattern: Bayer CFA pattern (0=RGGB, 1=GRBG, 2=GBRG, 3=BGGR),
// unusedBits: number of unused bits per sample, otherflags: additional flags, blackLevel: black level.
func (rp *RawProcessor) OpenBayer(data []byte, rawWidth, rawHeight uint16, leftMargin, topMargin, rightMargin, bottomMargin uint16, procflags, bayerPattern byte, unusedBits, otherflags, blackLevel uint) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return err
	}

	if len(data) == 0 {
		return ErrBufferOpen
	}

	rc := C.libraw_open_bayer(
		rp.res.handle,
		(*C.uchar)(unsafe.Pointer(&data[0])),
		C.uint(len(data)),
		C.ushort(rawWidth), C.ushort(rawHeight),
		C.ushort(leftMargin), C.ushort(topMargin),
		C.ushort(rightMargin), C.ushort(bottomMargin),
		C.uchar(procflags), C.uchar(bayerPattern),
		C.uint(unusedBits), C.uint(otherflags),
		C.uint(blackLevel),
	)
	runtime.KeepAlive(data)
	return checkError(rc, ErrBufferOpen)
}

func copyProcessedImage(img *C.libraw_processed_image_t) (*ProcessedImage, error) {
	dataSize := C.uint(img.data_size)
	if dataSize == 0 || dataSize > 0x7FFFFFFF {
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
