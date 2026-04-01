package golibraw

/*
#include <libraw/libraw.h>
*/
import "C"
import "unsafe"

// OpenBuffer 从内存缓冲区打开 RAW 图像数据。
func (rp *RawProcessor) OpenBuffer(data []byte) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return err
	}

	rc := C.libraw_open_buffer(rp.handle, unsafe.Pointer(&data[0]), C.size_t(len(data)))
	return librawError(rc, ErrBufferOpen)
}

// Unpack 解包 RAW 数据。必须在 OpenFile/OpenBuffer 之后调用。
func (rp *RawProcessor) Unpack() error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return err
	}

	rc := C.libraw_unpack(rp.handle)
	return librawError(rc, ErrUnpack)
}

// UnpackThumb 解包缩略图数据。
func (rp *RawProcessor) UnpackThumb() error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return err
	}

	rc := C.libraw_unpack_thumb(rp.handle)
	return librawError(rc, ErrUnpackThumb)
}

// Process 执行 dcraw 风格的图像处理。必须在 Unpack 之后调用。
// 处理参数需在调用前通过 ApplyOptions 设置。
func (rp *RawProcessor) Process() error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return err
	}

	rc := C.libraw_dcraw_process(rp.handle)
	return librawError(rc, ErrProcess)
}

// MakeMemImage 将处理后的图像转为内存数据。
// 返回的 ProcessedImage.Data 包含完整的 PPM 或 TIFF 文件数据。
// 必须在 Process 之后调用。
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

// MakeMemThumb 将缩略图转为内存数据。
// 必须在 UnpackThumb 之后调用。
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

// WritePPMTiff 将处理后的图像写为 PPM/TIFF 文件。
// 必须在 Process 之后调用。
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

// WriteThumb 将缩略图写入文件。
// 必须在 UnpackThumb 之后调用。
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

// copyProcessedImage 将 C 的 libraw_processed_image_t 数据拷贝到 Go 结构体。
// 调用方负责在返回后释放 C 内存。
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
