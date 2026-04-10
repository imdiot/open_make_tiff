package golibraw

/*
#include <libraw/libraw.h>
*/
import "C"

// Area mirrors libraw_area_t (top, left, bottom, right pixel coordinates).
type Area struct {
	Top    int16
	Left   int16
	Bottom int16
	Right  int16
}

// CanonMakernotes mirrors libraw_canon_makernotes_t.
type CanonMakernotes struct {
	ColorDataVer            int32
	ColorDataSubVer         int32
	SpecularWhiteLevel      int32
	NormalWhiteLevel        int32
	ChannelBlackLevel       [4]int32
	AverageBlackLevel       int32
	Multishot               [4]uint32
	MeteringMode            int16
	SpotMeteringMode        int16
	FlashMeteringMode       byte
	FlashExposureLock       int16
	ExposureMode            int16
	AESetting               int16
	ImageStabilization      int16
	FlashMode               int16
	FlashActivity           int16
	FlashBits               int16
	ManualFlashOutput       int16
	FlashOutput             int16
	FlashGuideNumber        int16
	ContinuousDrive         int16
	SensorWidth             int16
	SensorHeight            int16
	AFMicroAdjMode          int
	AFMicroAdjValue         float32
	MakernotesFlip          int16
	AutoRotateMode          int16
	RecordMode              int16
	SRAWQuality             int16
	WBI                     uint
	RFLensID                int16
	AutoLightingOptimizer   int
	HighlightTonePriority   int
	Quality                 int16
	CanonLog                int
	DefaultCropAbsolute     Area
	RecommendedImageArea    Area
	LeftOpticalBlack        Area
	UpperOpticalBlack       Area
	ActiveArea              Area
	ISOGain                 [2]int16
}

// SensorHighSpeedCrop mirrors libraw_sensor_highspeed_crop_t.
type SensorHighSpeedCrop struct {
	Left   uint16
	Top    uint16
	Width  uint16
	Height uint16
}

// NikonMakernotes mirrors libraw_nikon_makernotes_t.
type NikonMakernotes struct {
	ExposureBracketValue       float64
	ActiveDLighting            uint16
	ShootingMode               uint16
	ImageStabilization         [7]byte
	VibrationReduction         byte
	VRMode                     byte
	FlashSetting               string
	FlashType                  string
	FlashExposureCompensation  [4]byte
	ExternalFlashExposureComp  [4]byte
	FlashExposureBracketValue  [4]byte
	FlashMode                  byte
	FlashExposureCompensation2 int8
	FlashExposureCompensation3 int8
	FlashExposureCompensation4 int8
	FlashSource                byte
	FlashFirmware              [2]byte
	ExternalFlashFlags         byte
	FlashControlCommanderMode  byte
	FlashOutputAndCompensation byte
	FlashFocalLength           byte
	FlashGNDistance            byte
	FlashGroupControlMode      [4]byte
	FlashGroupOutputComp       [4]byte
	FlashColorFilter           byte
	NEFCompression             uint16
	ExposureMode               int
	ExposureProgram            int
	NMEshots                   int
	MEgainOn                   int
	MEWB                       [4]float64
	AFFineTune                 byte
	AFFineTuneIndex            byte
	AFFineTuneAdj              int8
	LensDataVersion            uint
	FlashInfoVersion           uint
	ColorBalanceVersion        uint
	Key                        byte
	NEFBitDepth                [4]uint16
	HighSpeedCropFormat        uint16
	SensorWidth                uint16
	SensorHeight               uint16
	ActiveDL                   uint16
	PictureControlVersion      uint
	PictureControlName         string
	PictureControlBase         string
	ShotInfoVersion            uint
	ShotInfoFirmware           string
	MakernotesFlip             int16
	RollAngle                  float64
	PitchAngle                 float64
	YawAngle                   float64
	SensorHighSpeedCrop       SensorHighSpeedCrop
}

// FujiMakernotes mirrors libraw_fuji_info_t.
type FujiMakernotes struct {
	ExpoMidPointShift        float32
	DynamicRange             uint16
	FilmMode                 uint16
	DynamicRangeSetting      uint16
	DevelopmentDynamicRange  uint16
	AutoDynamicRange         uint16
	DRangePriority           uint16
	DRangePriorityAuto       uint16
	DRangePriorityFixed      uint16
	FujiModel                string
	FujiModel2               string
	BrightnessCompensation   float32
	FocusMode                uint16
	AFMode                   uint16
	FocusPixel               [2]uint16
	PrioritySettings         uint16
	FocusSettings            uint
	AFCSettings              uint
	FocusWarning             uint16
	ImageStabilization       [3]uint16
	FlashMode                uint16
	WBPreset                 uint16
	ShutterType              uint16
	ExrMode                  uint16
	Macro                    uint16
	Rating                   uint
	CropMode                 uint16
	SerialSignature          string
	SensorID                 string
	RAFVersion               string
	RAFDataGeneration        int
	RAFDataVersion           uint16
	IsTSNERDTS               int
	DriveMode                int16
	BlackLevel               [9]uint16
	RAFDataImageSizeTable    [32]uint
	AutoBracketing           int
	SequenceNumber           int
	SeriesLength             int
	PixelShiftOffset         [2]float32
	ImageCount               int
}

// OlympusMakernotes mirrors libraw_olympus_makernotes_t.
type OlympusMakernotes struct {
	CameraType2          string
	ValidBits            uint16
	TagX640              uint
	TagX641              uint
	TagX642              uint
	TagX643              uint
	TagX644              uint
	TagX645              uint
	TagX646              uint
	TagX647              uint
	TagX648              uint
	TagX649              uint
	TagX650              uint
	TagX651              uint
	TagX652              uint
	TagX653              uint
	SensorCalibration    [2]int
	DriveMode            [5]uint16
	ColorSpace           uint16
	FocusMode            [2]uint16
	AutoFocus            uint16
	AFPoint              uint16
	AFAreas              [64]uint
	AFPointSelected      [5]float64
	AFResult             uint16
	AFFineTune           byte
	AFFineTuneAdj        [3]int16
	SpecialMode          [3]uint
	ZoomStepCount        uint16
	FocusStepCount       uint16
	FocusStepInfinity    uint16
	FocusStepNear        uint16
	FocusDistance        float64
	AspectFrame          [4]uint16
	StackedImage         [2]uint
	IsLiveND             byte
	LiveNDfactor         uint
	PanoramaMode         uint16
	PanoramaFrameNum     uint16
}

// SonyMakernotes mirrors libraw_sony_info_t.
type SonyMakernotes struct {
	CameraType                    uint16
	Sony0x9400Version              byte
	Sony0x9400ReleaseMode2         byte
	Sony0x9400SequenceImageNumber  uint
	Sony0x9400SequenceLength1      byte
	Sony0x9400SequenceFileNumber   uint
	Sony0x9400SequenceLength2      byte
	AFAreaModeSetting              byte
	AFAreaMode                     uint16
	FlexibleSpotPosition           [2]uint16
	AFPointSelected                byte
	AFPointSelected0x201e          byte
	NAFPointsUsed                  int16
	AFPointsUsed                   [10]byte
	AFTracking                     byte
	AFType                         byte
	FocusLocation                  [4]uint16
	FocusPosition                  uint16
	AFMicroAdjValue                int8
	AFMicroAdjOn                   int8
	AFMicroAdjRegisteredLenses     byte
	VariableLowPassFilter          uint16
	LongExposureNoiseReduction     uint
	HighISONoiseReduction          uint16
	HDR                            [2]uint16
	Group2010                      uint16
	Group9050                      uint16
	RealISOOffset                  uint16
	MeteringModeOffset             uint16
	ExposureProgramOffset          uint16
	ReleaseMode2Offset             uint16
	MinoltaCamID                   uint
	Firmware                       float32
	ImageCount3Offset              uint16
	ImageCount3                    uint
	ElectronicFrontCurtainShutter  uint
	MeteringMode2                  uint16
	SonyDateTime                   string
	ShotNumberSincePowerUp         uint
	PixelShiftGroupPrefix          uint16
	PixelShiftGroupID              uint
	NShotsInPixelShiftGroup        byte
	NumInPixelShiftGroup           byte
	PrdImageHeight                 uint16
	PrdImageWidth                  uint16
	PrdTotalBPS                    uint16
	PrdActiveBPS                   uint16
	PrdStorageMethod               uint16
	PrdBayerPattern                uint16
	SonyRawFileType                uint16
	RAWFileType                    uint16
	RawSizeType                    uint16
	Quality                        uint
	FileFormat                     uint16
	MetaVersion                    string
	AspectRatio                    float32
}

// KodakMakernotes mirrors libraw_kodak_makernotes_t.
type KodakMakernotes struct {
	BlackLevelTop      uint16
	BlackLevelBottom   uint16
	OffsetLeft         int16
	OffsetTop          int16
	ClipBlack          uint16
	ClipWhite          uint16
	ROMMCamDaylight    [3][3]float32
	ROMMCamTungsten    [3][3]float32
	ROMMCamFluorescent [3][3]float32
	ROMMCamFlash       [3][3]float32
	ROMMCamCustom      [3][3]float32
	ROMMCamAuto        [3][3]float32
	Val018percent      uint16
	Val100percent      uint16
	Val170percent      uint16
	MakerNoteKodak8a   int16
	ISOCalibrationGain float32
	AnalogISO          float32
}

// PanasonicMakernotes mirrors libraw_panasonic_makernotes_t.
type PanasonicMakernotes struct {
	Compression       uint16
	BlackLevelDim     uint16
	BlackLevel        [8]float32
	Multishot         uint
	Gamma             float32
	HighISOMultiplier [3]int
	FocusStepNear     int16
	FocusStepCount    int16
	ZoomPosition      uint
	LensManufacturer  uint
}

// PentaxMakernotes mirrors libraw_pentax_makernotes_t.
type PentaxMakernotes struct {
	DriveMode              [4]byte
	FocusMode              [2]uint16
	AFPointSelected        [2]uint16
	AFPointSelectedArea    uint16
	AFPointsInFocusVersion int
	AFPointsInFocus        uint
	FocusPosition          uint16
	DynamicRangeExpansion  [4]byte
	AFAdjustment           int16
	AFPointMode            byte
	MultiExposure          byte
	Quality                uint16
}

// PhaseOneMakernotes mirrors libraw_p1_makernotes_t.
type PhaseOneMakernotes struct {
	Software       string
	SystemType     string
	FirmwareString string
	SystemModel    string
}

// RicohMakernotes mirrors libraw_ricoh_makernotes_t.
type RicohMakernotes struct {
	AFStatus            uint16
	AFAreaXPosition     [2]uint
	AFAreaYPosition     [2]uint
	AFAreaMode          uint16
	SensorWidth         uint
	SensorHeight        uint
	CroppedImageWidth   uint
	CroppedImageHeight  uint
	WideAdapter         uint16
	CropMode            uint16
	NDFilter            uint16
	AutoBracketing      uint16
	MacroMode           uint16
	FlashMode           uint16
	FlashExposureComp   float64
	ManualFlashOutput   float64
}

// SamsungMakernotes mirrors libraw_samsung_makernotes_t.
type SamsungMakernotes struct {
	ImageSizeFull [4]uint
	ImageSizeCrop [4]uint
	ColorSpace    [2]int
	Key           [11]uint
	DigitalGain   float64
	DeviceType    int
	LensFirmware  string
}

// HasselbladMakernotes mirrors libraw_hasselblad_makernotes_t.
type HasselbladMakernotes struct {
	BaseISO                   int
	Gain                      float64
	Sensor                    string
	SensorUnit                string
	HostBody                  string
	SensorCode                int
	SensorSubCode             int
	CoatingCode               int
	Uncropped                 int
	CaptureSequenceInitiator  string
	SensorUnitConnector       string
	Format                    int
	NIFDCM                    [2]int
	RecommendedCrop           [2]int
	MNColorMatrix             [4][3]float64
}
