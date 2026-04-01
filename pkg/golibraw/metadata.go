package golibraw

/*
#include <libraw/libraw.h>
*/
import "C"

import "time"

// GetImageSizes 返回图像尺寸信息。
func (rp *RawProcessor) GetImageSizes() ImageSizes {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.closed || rp.handle == nil {
		return ImageSizes{}
	}

	s := rp.handle.sizes
	return ImageSizes{
		RawHeight:  uint16(s.raw_height),
		RawWidth:   uint16(s.raw_width),
		Height:     uint16(s.height),
		Width:      uint16(s.width),
		TopMargin:  uint16(s.top_margin),
		LeftMargin: uint16(s.left_margin),
		IHeight:    uint16(s.iheight),
		IWidth:     uint16(s.iwidth),
		Flip:       int(s.flip),
	}
}

// GetCameraInfo 返回相机和图像基本参数。
func (rp *RawProcessor) GetCameraInfo() CameraInfo {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.closed || rp.handle == nil {
		return CameraInfo{}
	}

	ip := C.libraw_get_iparams(rp.handle)
	return CameraInfo{
		Make:            C.GoString(&ip.make[0]),
		Model:           C.GoString(&ip.model[0]),
		NormalizedMake:  C.GoString(&ip.normalized_make[0]),
		NormalizedModel: C.GoString(&ip.normalized_model[0]),
		Software:        C.GoString(&ip.software[0]),
		RawCount:        uint(ip.raw_count),
		DNGVersion:      uint(ip.dng_version),
		IsFoveon:        ip.is_foveon != 0,
		Colors:          int(ip.colors),
		CDesc:           C.GoString(&ip.cdesc[0]),
	}
}

// GetLensInfo 返回镜头信息。
func (rp *RawProcessor) GetLensInfo() LensInfo {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.closed || rp.handle == nil {
		return LensInfo{}
	}

	li := C.libraw_get_lensinfo(rp.handle)
	return LensInfo{
		LensMake:              C.GoString(&li.LensMake[0]),
		Lens:                  C.GoString(&li.Lens[0]),
		LensSerial:            C.GoString(&li.LensSerial[0]),
		MinFocal:              float32(li.MinFocal),
		MaxFocal:              float32(li.MaxFocal),
		MaxAp4MinFocal:        float32(li.MaxAp4MinFocal),
		MaxAp4MaxFocal:        float32(li.MaxAp4MaxFocal),
		CurFocal:              float32(li.makernotes.CurFocal),
		CurAp:                 float32(li.makernotes.CurAp),
		FocalLengthIn35mmFormat: uint16(li.FocalLengthIn35mmFormat),
	}
}

// GetShootingParams 返回拍摄参数。
func (rp *RawProcessor) GetShootingParams() ShootingParams {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.closed || rp.handle == nil {
		return ShootingParams{}
	}

	ot := C.libraw_get_imgother(rp.handle)
	ts := time.Unix(int64(ot.timestamp), 0)
	return ShootingParams{
		ISOSpeed:  float32(ot.iso_speed),
		Shutter:   float32(ot.shutter),
		Aperture:  float32(ot.aperture),
		FocalLen:  float32(ot.focal_len),
		Timestamp: ts,
		Artist:    C.GoString(&ot.artist[0]),
		Desc:      C.GoString(&ot.desc[0]),
	}
}
