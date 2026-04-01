package golibraw

/*
#include <libraw/libraw.h>
*/
import "C"

import "time"

// ImageFormat 表示处理后的图像数据格式。
type ImageFormat int

const (
	ImageJPEG   ImageFormat = C.LIBRAW_IMAGE_JPEG
	ImageBitmap ImageFormat = C.LIBRAW_IMAGE_BITMAP
)

// ImageSizes 对应 libraw_image_sizes_t，描述图像尺寸和裁切信息。
type ImageSizes struct {
	RawHeight  uint16
	RawWidth   uint16
	Height     uint16
	Width      uint16
	TopMargin  uint16
	LeftMargin uint16
	IHeight    uint16
	IWidth     uint16
	Flip       int
}

// CameraInfo 对应 libraw_iparams_t，描述相机和图像基本参数。
type CameraInfo struct {
	Make             string
	Model            string
	NormalizedMake   string
	NormalizedModel  string
	Software         string
	RawCount         uint
	DNGVersion       uint
	IsFoveon         bool
	Colors           int
	CDesc            string
}

// LensInfo 对应 libraw_lensinfo_t 中的关键镜头信息。
type LensInfo struct {
	LensMake              string
	Lens                  string
	LensSerial            string
	MinFocal              float32
	MaxFocal              float32
	MaxAp4MinFocal        float32
	MaxAp4MaxFocal        float32
	CurFocal              float32
	CurAp                 float32
	FocalLengthIn35mmFormat uint16
}

// ShootingParams 对应 libraw_imgother_t 中的拍摄参数。
type ShootingParams struct {
	ISOSpeed  float32
	Shutter   float32
	Aperture  float32
	FocalLen  float32
	Timestamp time.Time
	Artist    string
	Desc      string
}

// ProcessedImage 对应 libraw_processed_image_t，包含处理后的图像数据。
// Data 字段包含完整的 PPM/TIFF/JPEG 文件数据，可直接写入文件。
type ProcessedImage struct {
	Type   ImageFormat
	Width  uint16
	Height uint16
	Colors uint16
	Bits   uint16
	Data   []byte
}

// WhiteBalanceMode 白平衡模式。
type WhiteBalanceMode int

const (
	WBCamera WhiteBalanceMode = iota
	WBAverage
	WBCustom
	WBGreyBox
)

// InterpolationQuality 插值质量。
type InterpolationQuality int

const (
	QualityLinear InterpolationQuality = iota
	QualityVNG
	QualityPPG
	QualityAHD
	QualityDCB
	QualityDHT  = 11
	QualityAAHD = 12
)

// HighlightMode 高光处理模式。
type HighlightMode int

const (
	HighlightClip HighlightMode = iota
	HighlightUnclip
	HighlightBlend
	HighlightRebuild
)

// ColorSpace 输出色彩空间。
type ColorSpace int

const (
	ColorSpaceRaw ColorSpace = iota
	ColorSpacesRGB
	ColorSpaceAdobe
	ColorSpaceWide
	ColorSpaceProPhoto
	ColorSpaceXYZ
	ColorSpaceACES
	ColorSpaceDCIP3
	ColorSpaceRec2020
)

// FlipMode 翻转模式。
type FlipMode int

const (
	FlipNone  FlipMode = 0
	Flip180   FlipMode = 3
	Flip90CCW FlipMode = 5
	Flip90CW  FlipMode = 6
)

// FBDDMode FBDD 噪声 Reduction 模式。
type FBDDMode int

const (
	FBDDDisabled FBDDMode = iota
	FBDDLight
	FBDDFull
)
