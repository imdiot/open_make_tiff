package golibraw

import "time"

type ImageFormat int

const (
	ImageJPEG   ImageFormat = 1 // LIBRAW_IMAGE_JPEG
	ImageBitmap ImageFormat = 2 // LIBRAW_IMAGE_BITMAP
	ImageJPEGXL ImageFormat = 3 // LIBRAW_IMAGE_JPEGXL
	ImageH265   ImageFormat = 4 // LIBRAW_IMAGE_H265
)

// InsetCropMask is a bitmask selecting which raw_inset_crop entries to check.
type InsetCropMask uint

const (
	InsetCropDefaultMask InsetCropMask = 1 << iota // raw_inset_crops[0] (DefaultCropOrigin/DefaultCropSize)
	InsetCropUserMask                              // raw_inset_crops[1] (UserCrop)
	InsetCropAllMask     InsetCropMask = InsetCropDefaultMask | InsetCropUserMask
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
	RawHeight        uint16
	RawWidth         uint16
	Height           uint16
	Width            uint16
	TopMargin        uint16
	LeftMargin       uint16
	IHeight          uint16
	IWidth           uint16
	Flip             int
	PixelAspectRatio float64
	RawPitch         uint
	RawAspect        uint16
	Mask             [8][4]int
	RawInsetCrops    [2]RawInsetCrop
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
	Filters         uint
	XTrans          [6][6]int8
	XTransAbs       [6][6]int8
	CDesc           string
	MakerIndex      uint
	XMPLen          uint
	XMPData         []byte
}

// LensInfo mirrors libraw_lensinfo_t.
// NikonLensInfo mirrors libraw_nikonlens_t.
type NikonLensInfo struct {
	EffectiveMaxAp float32
	LensIDNumber   byte
	LensFStops     byte
	MCUVersion     byte
	LensType       byte
}

// DNGLensInfo mirrors libraw_dnglens_t.
type DNGLensInfo struct {
	MinFocal       float32
	MaxFocal       float32
	MaxAp4MinFocal float32
	MaxAp4MaxFocal float32
}

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
	Nikon                   NikonLensInfo
	DNG                     DNGLensInfo
}

// ShootingParams mirrors libraw_imgother_t.
type ShootingParams struct {
	ISOSpeed      float32
	Shutter       float32
	Aperture      float32
	FocalLen      float32
	Timestamp     time.Time
	ShotOrder     uint
	Artist        string
	Desc          string
	AnalogBalance [4]float32
}

// GPSInfo mirrors libraw_gps_info_t.
type GPSInfo struct {
	Latitude     [3]float32 // degrees, minutes, seconds
	Longitude    [3]float32 // degrees, minutes, seconds
	GPSTimestamp [3]float32 // hours, minutes, seconds
	Altitude     float32
	AltRef       byte
	LatRef       byte
	LongRef      byte
	GPSStatus    byte
	GPSParsed    bool
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

// InterpolationQuality controls the demosaic algorithm for Bayer-pattern RAW images.
// Values 0-4 and 11-12 are always available. Values 5-9 require the GPL2 demosaic
// pack; value 10 requires the GPL3 demosaic pack. For Fuji X-Trans sensors, quality
// is ignored and xtrans_interpolate is used instead.
type InterpolationQuality int

const (
	QualityLinear InterpolationQuality = iota
	QualityVNG
	QualityPPG
	QualityAHD
	QualityDCB
	QualityModifiedAHD                           // 5  GPL2
	QualityAFD                                   // 6  GPL2
	QualityVCD                                   // 7  GPL2
	QualityVCDAHD                                // 8  GPL2
	QualityLMMSE                                 // 9  GPL2
	QualityAMaZE                                 // 10 GPL3
	QualityDHT
	QualityAAHD
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
	ColorSpaceRaw               ColorSpace = iota // LIBRAW_COLORSPACE_NotFound
	ColorSpacesRGB                                // LIBRAW_COLORSPACE_sRGB
	ColorSpaceAdobe                               // LIBRAW_COLORSPACE_AdobeRGB
	ColorSpaceWide                                // LIBRAW_COLORSPACE_WideGamutRGB
	ColorSpaceProPhoto                            // LIBRAW_COLORSPACE_ProPhotoRGB
	ColorSpaceICC                                 // LIBRAW_COLORSPACE_ICC
	ColorSpaceUncalibrated                        // LIBRAW_COLORSPACE_Uncalibrated
	ColorSpaceCameraLinearUniWB                   // LIBRAW_COLORSPACE_CameraLinearUniWB
	ColorSpaceCameraLinear                        // LIBRAW_COLORSPACE_CameraLinear
	ColorSpaceCameraGammaUniWB                    // LIBRAW_COLORSPACE_CameraGammaUniWB
	ColorSpaceCameraGamma                         // LIBRAW_COLORSPACE_CameraGamma
	ColorSpaceMonoLinear                          // LIBRAW_COLORSPACE_MonochromeLinear
	ColorSpaceMonoGamma                           // LIBRAW_COLORSPACE_MonochromeGamma
	ColorSpaceRec2020           ColorSpace = 12   // LIBRAW_COLORSPACE_Rec2020
	ColorSpaceUnknown           ColorSpace = 255  // LIBRAW_COLORSPACE_Unknown
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
	Lens                    string
	LensFormat              uint16
	LensMount               uint16
	CamID                   uint64
	CameraFormat            uint16
	CameraMount             uint16
	Body                    string
	FocalType               FocalType
	LensFeaturesPre         string
	LensFeaturesSuf         string
	MinFocal                float32
	MaxFocal                float32
	MaxAp4MinFocal          float32
	MaxAp4MaxFocal          float32
	MinAp4MinFocal          float32
	MinAp4MaxFocal          float32
	MaxAp                   float32
	MinAp                   float32
	CurFocal                float32
	CurAp                   float32
	MaxAp4CurFocal          float32
	MinAp4CurFocal          float32
	MinFocusDistance        float32
	FocusRangeIndex         float32
	LensFStops              float32
	TeleconverterID         uint64
	Teleconverter           string
	AdapterID               uint64
	Adapter                 string
	AttachmentID            uint64
	Attachment              string
	FocalUnits              uint16
	FocalLengthIn35mmFormat float32
}

// SensorTemperatures mirrors the temperature/flash subset of libraw_metadata_common_t.
type SensorTemperatures struct {
	CameraTemperature        float32
	SensorTemperature        float32
	SensorTemperature2       float32
	LensTemperature          float32
	AmbientTemperature       float32
	BatteryTemperature       float32
	ExifAmbientTemperature   float32
	FlashEC                  float32
	FlashGN                  float32
	RealISO                  float32
	Firmware                 string
	ExifHumidity             float32
	ExifPressure             float32
	ExifWaterDepth           float32
	ExifAcceleration         float32
	ExifCameraElevationAngle float32
	ExifExposureIndex        float32
	ColorSpace               uint16
	ExposureCalibrationShift float32
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

// ThumbnailItem mirrors libraw_thumbnail_item_t (internal thumbnail list entry).
type ThumbnailItem struct {
	Format ThumbnailFormat
	Width  uint16
	Height uint16
	Flip   uint16
	Length uint
	Misc   uint
	Offset int64
}

// WBIndex mirrors LIBRAW_WBI_* constants.
type WBIndex int

const (
	WBUnknown         WBIndex = 0
	WBDaylight        WBIndex = 1
	WBFluorescent     WBIndex = 2
	WBTungsten        WBIndex = 3
	WBFlash           WBIndex = 4
	WBFineWeather     WBIndex = 9
	WBCloudy          WBIndex = 10
	WBShade           WBIndex = 11
	WBFL_D            WBIndex = 12
	WBFL_N            WBIndex = 13
	WBFL_W            WBIndex = 14
	WBFL_WW           WBIndex = 15
	WBFL_L            WBIndex = 16
	WBIll_A           WBIndex = 17
	WBIll_B           WBIndex = 18
	WBIll_C           WBIndex = 19
	WBD55             WBIndex = 20
	WBD65             WBIndex = 21
	WBD75             WBIndex = 22
	WBD50             WBIndex = 23
	WBStudioTung      WBIndex = 24
	WBSunset          WBIndex = 64
	WBUnderwater      WBIndex = 65
	WBFluorescentHigh WBIndex = 66
	WBHTMercury       WBIndex = 67
	WBAsShot          WBIndex = 81
	WBAuto            WBIndex = 82
	WBCustom          WBIndex = 83
	WBAuto1           WBIndex = 85
	WBAuto2           WBIndex = 86
	WBAuto3           WBIndex = 87
	WBAuto4           WBIndex = 88
	WBCustom1         WBIndex = 90
	WBCustom2         WBIndex = 91
	WBCustom3         WBIndex = 92
	WBCustom4         WBIndex = 93
	WBCustom5         WBIndex = 94
	WBCustom6         WBIndex = 95
	WBPCSet1          WBIndex = 96
	WBPCSet2          WBIndex = 97
	WBPCSet3          WBIndex = 98
	WBPCSet4          WBIndex = 99
	WBPCSet5          WBIndex = 100
	WBMeasured        WBIndex = 110
	WBBW              WBIndex = 120
	WBKelvin          WBIndex = 254
	WBOther           WBIndex = 255
	WBNone            WBIndex = 0xffff
)

// WBTempCoeff holds one entry from WBCT_Coeffs: CCT + R G1 B G2 coefficients.
type WBTempCoeff struct {
	CCT    int
	Coeffs [4]float32
}

// DNGColorInfo mirrors libraw_dng_color_t.
type DNGColorInfo struct {
	ParsedFields  uint
	Illuminant    uint16
	Calibration   [4][4]float32
	ColorMatrix   [4][3]float32
	ForwardMatrix [3][4]float32
}

// DNGLevels mirrors libraw_dng_levels_t.
type DNGLevels struct {
	AsShotNeutral       [4]float32
	BaselineExposure    float32
	AnalogBalance       [4]float32
	DngBlack            uint
	DngFBlack           float32
	DngWhiteLevel       [4]uint
	DefaultCrop         [4]uint16  // Origin and size (left, top, width, height)
	UserCrop            [4]float32 // top-left-bottom-right relative to default_crop
	PreviewColorSpace   uint
	LinearResponseLimit float32
}

// PhaseOneData mirrors libraw ph1_t (Phase One specific metadata).
type PhaseOneData struct {
	Format   int
	KeyOff   int
	Tag21a   int
	TBlack   int
	SplitCol int
	BlackCol int
	SplitRow int
	BlackRow int
	Tag210   float32
}

// P1Color mirrors libraw_P1_color_t (Phase One ROMM camera data).
type P1Color struct {
	ROMMCam [9]float32
}

// ColorData mirrors selected fields from libraw_colordata_t.
type ColorData struct {
	Black                uint
	CBlack               [4]uint
	LinearMax            [4]uint
	Maximum              uint
	DataMaximum          uint
	FMaximum             float32
	FNorm                float32
	RawBPS               uint
	FlashUsed            float32
	CanonEV              float32
	CamMul               [4]float32
	PreMul               [4]float32
	CMatrix              [3][4]float32
	CCM                  [3][4]float32
	RGBCam               [3][4]float32
	CamXYZ               [4][3]float32
	White                [8][8]uint16
	PhaseOneData         PhaseOneData
	P1Color              [2]P1Color
	UniqueCameraModel    string
	LocalizedCameraModel string
	ImageUniqueID        string
	RawDataUniqueID      string
	OriginalRawFileName  string
	Model2               string
	HasICCProfile        bool
	ICCProfileLength     uint
	BlackStat            [8]uint
	ExifColorSpace       int
	AsShotWBApplied      bool
}

// Warning flags (enum LibRaw_warnings).
type Warning uint

const (
	WarnNone                 Warning = 0
	WarnBadCameraWB          Warning = 1 << 2
	WarnNoMetadata           Warning = 1 << 3
	WarnNoJpegLib            Warning = 1 << 4
	WarnNoEmbeddedProfile    Warning = 1 << 5
	WarnNoInputProfile       Warning = 1 << 6
	WarnBadOutputProfile     Warning = 1 << 7
	WarnNoBadpixelmap        Warning = 1 << 8
	WarnBadDarkFrameFile     Warning = 1 << 9
	WarnBadDarkFrameDim      Warning = 1 << 10
	WarnRawSpeedProblem      Warning = 1 << 12
	WarnRawSpeedUnsupported  Warning = 1 << 13
	WarnRawSpeedProcessed    Warning = 1 << 14
	WarnFallbackToAHD        Warning = 1 << 15
	WarnParseFujiProcessed   Warning = 1 << 16
	WarnDNGSDKProcessed      Warning = 1 << 17
	WarnDNGImagesReordered   Warning = 1 << 18
	WarnDNGStage2Applied     Warning = 1 << 19
	WarnDNGStage3Applied     Warning = 1 << 20
	WarnRawSpeed3Problem     Warning = 1 << 21
	WarnRawSpeed3Unsupported Warning = 1 << 22
	WarnRawSpeed3Processed   Warning = 1 << 23
	WarnRawSpeed3NotListed   Warning = 1 << 24
	WarnVendorCropSuggested  Warning = 1 << 25
	WarnDNGNotProcessed      Warning = 1 << 26
	WarnDNGNotParsed         Warning = 1 << 27
)

// Progress flags (enum LibRaw_progress).
type ProgressFlag uint

const (
	ProgressStart          ProgressFlag = 0
	ProgressOpen           ProgressFlag = 1
	ProgressIdentify       ProgressFlag = 1 << 1
	ProgressSizeAdjust     ProgressFlag = 1 << 2
	ProgressLoadRaw        ProgressFlag = 1 << 3
	ProgressRaw2Image      ProgressFlag = 1 << 4
	ProgressRemoveZeroes   ProgressFlag = 1 << 5
	ProgressBadPixels      ProgressFlag = 1 << 6
	ProgressDarkFrame      ProgressFlag = 1 << 7
	ProgressFoveonInterp   ProgressFlag = 1 << 8
	ProgressScaleColors    ProgressFlag = 1 << 9
	ProgressPreInterpolate ProgressFlag = 1 << 10
	ProgressInterpolate    ProgressFlag = 1 << 11
	ProgressMixGreen       ProgressFlag = 1 << 12
	ProgressMedianFilter   ProgressFlag = 1 << 13
	ProgressHighlights     ProgressFlag = 1 << 14
	ProgressFujiRotate     ProgressFlag = 1 << 15
	ProgressFlip           ProgressFlag = 1 << 16
	ProgressApplyProfile   ProgressFlag = 1 << 17
	ProgressConvertRGB     ProgressFlag = 1 << 18
	ProgressStretch        ProgressFlag = 1 << 19
	ProgressThumbLoad      ProgressFlag = 1 << 28
)

// CameraMakerIndex mirrors enum LibRaw_cameramaker_index.
type CameraMakerIndex int

const (
	MakerUnknown CameraMakerIndex = iota
	MakerAgfa
	MakerAlcatel
	MakerApple
	MakerAptina
	MakerAVT
	MakerBaumer
	MakerBroadcom
	MakerCanon
	MakerCasio
	MakerCINE
	MakerClauss
	MakerContax
	MakerCreative
	MakerDJI
	MakerDXO
	MakerEpson
	MakerFoculus
	MakerFujifilm
	MakerGeneric
	MakerGione
	MakerGITUP
	MakerGoogle
	MakerGoPro
	MakerHasselblad
	MakerHTC
	MakerIMobile
	MakerImacon
	MakerJKImaging
	MakerKodak
	MakerKonica
	MakerLeaf
	MakerLeica
	MakerLenovo
	MakerLG
	MakerLogitech
	MakerMamiya
	MakerMatrix
	MakerMeizu
	MakerMicron
	MakerMinolta
	MakerMotorola
	MakerNGM
	MakerNikon
	MakerNokia
	MakerOlympus
	MakerOmniVision
	MakerPanasonic
	MakerParrot
	MakerPentax
	MakerPhaseOne
	MakerPhotoControl
	MakerPhotron
	MakerPixelink
	MakerPolaroid
	MakerRED
	MakerRicoh
	MakerRollei
	MakerRoverShot
	MakerSamsung
	MakerSigma
	MakerSinar
	MakerSMaL
	MakerSony
	MakerSTMicro
	MakerTHL
	MakerVLUU
	MakerXiaomi
	MakerXiaoyi
	MakerYI
	MakerYuneec
	MakerZeiss
	MakerOnePlus
	MakerISG
	MakerVIVO
	MakerHMDGlobal
	MakerHuawei
	MakerRaspberryPi
	MakerOmDigital
)

// BayerPattern mirrors LIBRAW_OPENBAYER_* constants.
type BayerPattern byte

const (
	BayerRGGB BayerPattern = 0x94
	BayerBGGR BayerPattern = 0x16
	BayerGRBG BayerPattern = 0x61
	BayerGBRG BayerPattern = 0x49
)

// RawSpecial mirrors enum LibRaw_rawspecial_t.
type RawSpecial uint

const (
	RawSpecialNone           RawSpecial = 0
	RawSpecialSonyARW2Base   RawSpecial = 1
	RawSpecialSonyARW2Delta  RawSpecial = 1 << 1
	RawSpecialSonyARW2Zero   RawSpecial = 1 << 2
	RawSpecialSonyARW2ToVal  RawSpecial = 1 << 3
	RawSpecialSonyARW2All    RawSpecial = 15
	RawSpecialNoDP2QInterpRG RawSpecial = 1 << 4
	RawSpecialNoDP2QInterpAF RawSpecial = 1 << 5
	RawSpecialSRawNoRGB      RawSpecial = 1 << 6
	RawSpecialSRawNoInterp   RawSpecial = 1 << 7
)

// DecoderFlag mirrors enum LibRaw_decoder_flags (returned by GetDecoderInfo).
type DecoderFlag uint

const (
	DecoderHasCurve          DecoderFlag = 1 << 4
	DecoderSonyARW2          DecoderFlag = 1 << 5
	DecoderTryRawSpeed       DecoderFlag = 1 << 6
	DecoderOwnAlloc          DecoderFlag = 1 << 7
	DecoderFixedMaxC         DecoderFlag = 1 << 8
	DecoderAdobeCopyPixel    DecoderFlag = 1 << 9
	DecoderLegacyWithMargins DecoderFlag = 1 << 10
	Decoder3Channel          DecoderFlag = 1 << 11
	DecoderSinar4Shot        DecoderFlag = 1 << 11
	DecoderFlatData          DecoderFlag = 1 << 12
	DecoderFlatBG2Swapped    DecoderFlag = 1 << 13
	DecoderUnsupportedFormat DecoderFlag = 1 << 14
	DecoderNotSet            DecoderFlag = 1 << 15
	DecoderTryRawSpeed3      DecoderFlag = 1 << 16
)

// Exception mirrors enum LibRaw_exceptions.
type Exception int

const (
	ExceptionNone Exception = iota
	ExceptionAlloc
	ExceptionDecodeRAW
	ExceptionDecodeJPEG
	ExceptionIOEOF
	ExceptionIOCorrupt
	ExceptionCancelledByCallback
	ExceptionBadCrop
	ExceptionIOBadFile
	ExceptionDecodeJPEG2000
	ExceptionTooBig
	ExceptionMemPool
	ExceptionUnsupportedFormat
)

// CameraMount mirrors enum LibRaw_camera_mounts.
type CameraMount uint16

const (
	MountUnknown CameraMount = iota
	MountAlpa
	MountC
	MountCanonEFM
	MountCanonEFS
	MountCanonEF
	MountCanonRF
	MountContaxN
	MountContax645
	MountFT
	MountMFT
	MountFujiGF
	MountFujiGX
	MountFujiX
	MountHasselbladH
	MountHasselbladV
	MountHasselbladXCD
	MountLeicaM
	MountLeicaR
	MountLeicaS
	MountLeicaSL
	MountLeicaTL
	MountLPSL
	MountMamiya67
	MountMamiya645
	MountMinoltaA
	MountNikonCX
	MountNikonF
	MountNikonZ
	MountPhaseOneIXMMV
	MountPhaseOneIXMRS
	MountPhaseOneIXM
	MountPentax645
	MountPentaxK
	MountPentaxQ
	MountRicohModule
	MountRolleiBayonet
	MountSamsungNXM
	MountSamsungNX
	MountSigmaX3F
	MountSonyE
	MountLF
	MountDigitalBack
	MountFixedLens
	MountILUM
)

// CameraFormat mirrors enum LibRaw_camera_formats.
type CameraFormat uint16

const (
	FormatUnknown CameraFormat = iota
	FormatAPSC
	FormatFF
	FormatMF
	FormatAPSH
	Format1Inch
	Format1div2p3Inch
	Format1div1p7Inch
	FormatFT
	FormatCrop645
	FormatLeicaS
	Format645
	Format66
	Format69
	FormatLF
	FormatLeicaDMR
	Format67
	FormatSigmaAPSC
	FormatSigmaMerrill
	FormatSigmaAPSH
	Format3648
	Format68
)

// HasselbladFormat mirrors LIBRAW_HF_* constants.
type HasselbladFormat uint

const (
	HFUnknown HasselbladFormat = iota
	HF3FR
	HFFFF
	HFImacon
	HFHasselbladDNG
	HFAdobeDNG
	HFAdobeDNGPhocusDNG
)

// LibRaw configuration constants.
const (
	CBlackSize        = 4104
	ThumbnailMaxCount = 8
	AFDataMaxCount    = 4
	IFDMaxCount       = 10
)

// Capability flags (enum LibRaw_runtime_capabilities).
type Capability uint

const (
	CapRawSpeed Capability = 1 << iota
	CapDNGSDK
	CapGPRSDK
	CapUnicodePaths
	CapX3FTools
	CapRPI6BY9
	CapZlib
	CapJPEG
	CapRawSpeed3
	CapRawSpeedBits
)

// DecoderInfo holds information about the RAW decoder used.
type DecoderInfo struct {
	DecoderName  string
	DecoderFlags uint
}

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
