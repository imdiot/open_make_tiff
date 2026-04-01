package golibraw

/*
#include <libraw/libraw.h>
*/
import "C"

import "time"

type ImageFormat int

const (
	ImageJPEG   ImageFormat = C.LIBRAW_IMAGE_JPEG
	ImageBitmap ImageFormat = C.LIBRAW_IMAGE_BITMAP
)

// ImageSizes mirrors libraw_image_sizes_t.
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

// CameraInfo mirrors libraw_iparams_t.
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

// LensInfo mirrors libraw_lensinfo_t.
type LensInfo struct {
	LensMake                string
	Lens                    string
	LensSerial              string
	MinFocal                float32
	MaxFocal                float32
	MaxAp4MinFocal          float32
	MaxAp4MaxFocal          float32
	CurFocal                float32
	CurAp                   float32
	FocalLengthIn35mmFormat uint16
}

// ShootingParams mirrors libraw_imgother_t.
type ShootingParams struct {
	ISOSpeed  float32
	Shutter   float32
	Aperture  float32
	FocalLen  float32
	Timestamp time.Time
	Artist    string
	Desc      string
}

// ProcessedImage mirrors libraw_processed_image_t.
// Data contains a complete PPM/TIFF/JPEG file ready to be written to disk.
type ProcessedImage struct {
	Type   ImageFormat
	Width  uint16
	Height uint16
	Colors uint16
	Bits   uint16
	Data   []byte
}

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

type HighlightMode int

const (
	HighlightClip HighlightMode = iota
	HighlightUnclip
	HighlightBlend
	HighlightRebuild
)

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

type FlipMode int

const (
	FlipNone  FlipMode = 0
	Flip180   FlipMode = 3
	Flip90CCW FlipMode = 5
	Flip90CW  FlipMode = 6
)

type FBDDMode int

const (
	FBDDDisabled FBDDMode = iota
	FBDDLight
	FBDDFull
)
