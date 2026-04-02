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

// DNGSDKFlags controls which DNG features the DNG SDK decodes.
// Maps to LibRaw LIBRAW_DNG_* bitmask (rawparams.use_dngsdk).
type DNGSDKFlags int

const (
	DNGSDKNone    DNGSDKFlags = 0
	DNGSDKFloat   DNGSDKFlags = 1  // LIBRAW_DNG_FLOAT
	DNGSDKLinear  DNGSDKFlags = 2  // LIBRAW_DNG_LINEAR
	DNGSDKDeflate DNGSDKFlags = 4  // LIBRAW_DNG_DEFLATE
	DNGSDKXTrans  DNGSDKFlags = 8  // LIBRAW_DNG_XTRANS
	DNGSDKOther   DNGSDKFlags = 16 // LIBRAW_DNG_OTHER
	DNGSDK8Bit    DNGSDKFlags = 32 // LIBRAW_DNG_8BIT

	DNGSDKDefault = DNGSDKFloat | DNGSDKLinear | DNGSDKDeflate | DNGSDK8Bit // LibRaw LIBRAW_DNG_DEFAULT
	DNGSDKAll     = DNGSDKFloat | DNGSDKLinear | DNGSDKDeflate | DNGSDKXTrans | DNGSDKOther | DNGSDK8Bit
)

// RawSpeedFlags controls RawSpeed decoder usage.
// Maps to LibRaw rawparams.use_rawspeed bitmask.
type RawSpeedFlags int

const (
	RawSpeedV1Use           RawSpeedFlags = 1      // LIBRAW_RAWSPEEDV1_USE
	RawSpeedV1FailOnUnknown RawSpeedFlags = 1 << 1 // LIBRAW_RAWSPEEDV1_FAILONUNKNOWN
	RawSpeedV1IgnoreErrors  RawSpeedFlags = 1 << 2 // LIBRAW_RAWSPEEDV1_IGNOREERRORS
	RawSpeedV3Use           RawSpeedFlags = 1 << 8 // LIBRAW_RAWSPEEDV3_USE
	RawSpeedV3FailOnUnknown RawSpeedFlags = 1 << 9 // LIBRAW_RAWSPEEDV3_FAILONUNKNOWN
	RawSpeedV3IgnoreErrors  RawSpeedFlags = 1 << 10
)

// RawOptions controls LibRaw rawparams.options bitmask.
type RawOptions uint

const (
	RawOptPentaxPSAllFrames             RawOptions = 1
	RawOptConvertFloatToInt             RawOptions = 1 << 1
	RawOptARQSkipChannelSwap            RawOptions = 1 << 2
	RawOptNoRotateKodakThumbs           RawOptions = 1 << 3
	RawOptUsePPM16Thumbs                RawOptions = 1 << 5
	RawOptDontCheckDNGIlluminant        RawOptions = 1 << 6
	RawOptDNGSDKZeroCopy                RawOptions = 1 << 7
	RawOptZeroFiltersMonochromeTiffs    RawOptions = 1 << 8
	RawOptDNGAddEnhanced                RawOptions = 1 << 9
	RawOptDNGAddPreviews                RawOptions = 1 << 10
	RawOptDNGPreferLargestImage         RawOptions = 1 << 11
	RawOptDNGStage2                     RawOptions = 1 << 12
	RawOptDNGStage3                     RawOptions = 1 << 13
	RawOptDNGAllowSizeChange            RawOptions = 1 << 14
	RawOptDNGDisableWBAdjust            RawOptions = 1 << 15
	RawOptProvideNonStandardWB          RawOptions = 1 << 16
	RawOptCameraWBFallbackDaylight      RawOptions = 1 << 17
	RawOptCheckThumbnailsKnownVendors   RawOptions = 1 << 18
	RawOptCheckThumbnailsAllVendors     RawOptions = 1 << 19
	RawOptDNGStage2IfPresent            RawOptions = 1 << 20
	RawOptDNGStage3IfPresent            RawOptions = 1 << 21
	RawOptDNGAddMasks                   RawOptions = 1 << 22
	RawOptCanonIgnoreMakernotesRotation RawOptions = 1 << 23
	RawOptAllowJPEGXLPreviews           RawOptions = 1 << 24
	RawOptCanonCheckCameraAutoRotation  RawOptions = 1 << 26
	RawOptDNGStage23IfPresentJPGJXL     RawOptions = 1 << 27
)
