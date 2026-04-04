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

// RawInsetCrop mirrors libraw_raw_inset_crop_t.
type RawInsetCrop struct {
	Left   uint16
	Top    uint16
	Width  uint16
	Height uint16
}

// ImageSizes mirrors libraw_image_sizes_t.
type ImageSizes struct {
	RawHeight       uint16
	RawWidth        uint16
	Height          uint16
	Width           uint16
	TopMargin       uint16
	LeftMargin      uint16
	IHeight         uint16
	IWidth          uint16
	Flip            int
	PixelAspectRatio float64
	RawInsetCrops   [2]RawInsetCrop
}

// CameraInfo mirrors libraw_iparams_t.
type CameraInfo struct {
	Make            string
	Model           string
	NormalizedMake  string
	NormalizedModel string
	Software        string
	RawCount        uint
	DNGVersion      uint
	IsFoveon        bool
	Colors          int
	CDesc           string
	MakerIndex      uint
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
	InternalLensSerial      string
	EXIFMaxAp               float32
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

// ShootingInfo mirrors libraw_shootinginfo_t.
type ShootingInfo struct {
	DriveMode          int16
	FocusMode          int16
	MeteringMode       int16
	AFPoint            int16
	ExposureMode       int16
	ExposureProgram    int16
	ImageStabilization int16
	BodySerial         string
	InternalBodySerial string
}

// FocalType mirrors LIBRAW_FT_* constants.
type FocalType int16

const (
	FocalTypeUndefined FocalType = iota
	FocalTypePrime
	FocalTypeZoom
	FocalTypeZoomConstantAp
	FocalTypeZoomVariableAp
)

// MakernotesLensInfo mirrors libraw_makernotes_lens_t.
type MakernotesLensInfo struct {
	Lens                  string
	LensFormat            uint16
	LensMount             uint16
	CamID                 uint64
	CameraFormat          uint16
	CameraMount           uint16
	Body                  string
	FocalType             FocalType
	LensFeaturesPre       string
	LensFeaturesSuf       string
	MinFocal              float32
	MaxFocal              float32
	MaxAp4MinFocal        float32
	MaxAp4MaxFocal        float32
	MinAp4MinFocal        float32
	MinAp4MaxFocal        float32
	MaxAp                 float32
	MinAp                 float32
	CurFocal              float32
	CurAp                 float32
	MaxAp4CurFocal        float32
	MinAp4CurFocal        float32
	MinFocusDistance      float32
	FocusRangeIndex       float32
	LensFStops            float32
	TeleconverterID       uint64
	Teleconverter         string
	AdapterID             uint64
	Adapter               string
	AttachmentID          uint64
	Attachment            string
	FocalUnits            uint16
	FocalLengthIn35mmFormat float32
}

// SensorTemperatures mirrors the temperature/flash subset of libraw_metadata_common_t.
type SensorTemperatures struct {
	CameraTemperature      float32
	SensorTemperature      float32
	SensorTemperature2     float32
	LensTemperature        float32
	AmbientTemperature     float32
	BatteryTemperature     float32
	ExifAmbientTemperature float32
	FlashEC                float32
	FlashGN                float32
	RealISO                float32
	Firmware               string
}

// ThumbnailFormat mirrors LIBRAW_THUMBNAIL_* constants.
type ThumbnailFormat int

const (
	ThumbUnknown ThumbnailFormat = iota
	ThumbJPEG
	ThumbBitmap
	ThumbBitmap16
	ThumbLayer
	ThumbRollei
	ThumbH265
	ThumbJPEGXL
)

// ThumbnailInfo mirrors libraw_thumbnail_t.
type ThumbnailInfo struct {
	Format ThumbnailFormat
	Width  uint16
	Height uint16
	Length uint
	Colors int
}

// WBIndex mirrors LIBRAW_WBI_* constants.
type WBIndex int

const (
	WBUnknown     WBIndex = 0
	WBDaylight    WBIndex = 1
	WBFluorescent WBIndex = 2
	WBTungsten    WBIndex = 3
	WBFlash       WBIndex = 4
	WBFineWeather WBIndex = 9
	WBCloudy      WBIndex = 10
	WBShade       WBIndex = 11
	WBFL_D        WBIndex = 12
	WBFL_N        WBIndex = 13
	WBFL_W        WBIndex = 14
	WBFL_WW       WBIndex = 15
	WBFL_L        WBIndex = 16
	WBIll_A       WBIndex = 17
	WBIll_B       WBIndex = 18
	WBIll_C       WBIndex = 19
	WBD55         WBIndex = 20
	WBD65         WBIndex = 21
	WBD75         WBIndex = 22
	WBD50         WBIndex = 23
	WBStudioTung  WBIndex = 24
	WBSunset      WBIndex = 64
	WBAsShot      WBIndex = 81
	WBAuto        WBIndex = 82
	WBCustom      WBIndex = 83
	WBAuto1       WBIndex = 85
	WBAuto2       WBIndex = 86
	WBAuto3       WBIndex = 87
	WBAuto4       WBIndex = 88
	WBCustom1     WBIndex = 90
	WBCustom2     WBIndex = 91
	WBCustom3     WBIndex = 92
	WBCustom4     WBIndex = 93
	WBCustom5     WBIndex = 94
	WBCustom6     WBIndex = 95
	WBPCSet1      WBIndex = 96
	WBPCSet2      WBIndex = 97
	WBPCSet3      WBIndex = 98
	WBPCSet4      WBIndex = 99
	WBPCSet5      WBIndex = 100
	WBMeasured    WBIndex = 110
	WBBW          WBIndex = 120
	WBKelvin      WBIndex = 254
	WBOther       WBIndex = 255
	WBNone        WBIndex = 0xffff
)

// WBTempCoeff holds one entry from WBCT_Coeffs: CCT + R G1 B G2 coefficients.
type WBTempCoeff struct {
	CCT    int
	Coeffs [4]float32
}

// DNGColorInfo mirrors libraw_dng_color_t.
type DNGColorInfo struct {
	Illuminant    uint16
	Calibration   [4][4]float32
	ColorMatrix   [4][3]float32
	ForwardMatrix [3][4]float32
}

// DNGLevels mirrors the white balance subset of libraw_dng_levels_t.
type DNGLevels struct {
	AsShotNeutral    [4]float32
	BaselineExposure float32
	AnalogBalance    [4]float32
}

// ColorData mirrors selected fields from libraw_colordata_t.
type ColorData struct {
	Black                 uint
	CBlack                [4]uint
	LinearMax             [4]uint
	CamMul                [4]float32
	PreMul                [4]float32
	CMatrix               [3][4]float32
	CCM                   [3][4]float32
	RGBCam                [3][4]float32
	CamXYZ                [4][3]float32
	UniqueCameraModel     string
	LocalizedCameraModel  string
	ImageUniqueID         string
	RawDataUniqueID       string
	OriginalRawFileName   string
	Model2                string
	HasICCProfile         bool
	ICCProfileLength      uint
	ExifColorSpace        int
	AsShotWBApplied       bool
}
