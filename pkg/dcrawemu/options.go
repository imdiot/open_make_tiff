package dcrawemu

import (
	"io"
	"log/slog"
)

type WhiteBalanceMode int

const (
	WBCamera WhiteBalanceMode = iota
	WBAverage
	WBCustom
	WBGreyBox
)

func (w WhiteBalanceMode) String() string {
	switch w {
	case WBCamera:
		return "camera"
	case WBAverage:
		return "average"
	case WBCustom:
		return "custom"
	case WBGreyBox:
		return "greybox"
	default:
		return "unknown"
	}
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

func (q InterpolationQuality) String() string {
	switch q {
	case QualityLinear:
		return "linear"
	case QualityVNG:
		return "vng"
	case QualityPPG:
		return "ppg"
	case QualityAHD:
		return "ahd"
	case QualityDCB:
		return "dcb"
	case QualityDHT:
		return "dht"
	case QualityAAHD:
		return "aahd"
	default:
		return "unknown"
	}
}

type HighlightMode int

const (
	HighlightClip HighlightMode = iota
	HighlightUnclip
	HighlightBlend
	HighlightRebuild
)

func (h HighlightMode) String() string {
	switch h {
	case HighlightClip:
		return "clip"
	case HighlightUnclip:
		return "unclip"
	case HighlightBlend:
		return "blend"
	case HighlightRebuild:
		return "rebuild"
	default:
		return "unknown"
	}
}

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

func (c ColorSpace) String() string {
	switch c {
	case ColorSpaceRaw:
		return "raw"
	case ColorSpacesRGB:
		return "srgb"
	case ColorSpaceAdobe:
		return "adobe"
	case ColorSpaceWide:
		return "wide"
	case ColorSpaceProPhoto:
		return "prophoto"
	case ColorSpaceXYZ:
		return "xyz"
	case ColorSpaceACES:
		return "aces"
	case ColorSpaceDCIP3:
		return "dcip3"
	case ColorSpaceRec2020:
		return "rec2020"
	default:
		return "unknown"
	}
}

type FlipMode int

const (
	FlipNone  FlipMode = 0
	Flip180   FlipMode = 3
	Flip90CCW FlipMode = 5
	Flip90CW  FlipMode = 6
)

func (f FlipMode) String() string {
	switch f {
	case FlipNone:
		return "none"
	case Flip180:
		return "180"
	case Flip90CCW:
		return "90ccw"
	case Flip90CW:
		return "90cw"
	default:
		return "unknown"
	}
}

type OutputFormat int

const (
	OutputFormatNone   OutputFormat = iota
	OutputFormatTIFF                // -Z tiff
	OutputFormatStdout              // -Z -
)

func (f OutputFormat) String() string {
	switch f {
	case OutputFormatTIFF:
		return "tiff"
	case OutputFormatStdout:
		return "-"
	default:
		return "none"
	}
}

type FBDDMode int

const (
	FBDDDisabled FBDDMode = iota
	FBDDLight
	FBDDFull
)

func (f FBDDMode) String() string {
	switch f {
	case FBDDDisabled:
		return "disabled"
	case FBDDLight:
		return "light"
	case FBDDFull:
		return "full"
	default:
		return "unknown"
	}
}

type Options struct {
	executable string
	Logger     *slog.Logger

	whiteBalanceMode     WhiteBalanceMode
	whiteBalanceModeSet  bool
	customWhiteBalance   [4]float64
	customWhiteBalanceSet bool
	greyBox              [4]int
	greyBoxSet           bool

	useCameraMatrix      bool
	useCameraMatrixSet   bool
	chromaticAberration  [2]float64
	chromaticAberrationSet bool
	outputColorSpace     ColorSpace
	outputColorSpaceSet  bool
	outputProfile        string
	cameraProfile        string

	badPixels string
	darkFrame string

	darkness  int
	darknessSet bool
	saturation int
	saturationSet bool
	brightness float64
	brightnessSet bool
	highlightMode HighlightMode
	highlightModeSet bool
	noAutoBright  bool
	noAutoBrightSet bool
	exposureShift float64
	exposurePreserve float64
	exposureCorrectionSet bool

	quality InterpolationQuality
	qualitySet bool
	halfSize bool
	halfSizeSet bool
	fourColorRGB bool
	fourColorRGBSet bool
	medianPasses int
	medianPassesSet bool
	greenMatching bool
	greenMatchingSet bool
	noInterpolation bool
	noInterpolationSet bool

	noiseThreshold float64
	noiseThresholdSet bool
	fbddMode FBDDMode
	fbddModeSet bool
	dcbIterations int
	dcbIterationsSet bool
	dcbEnhance bool
	dcbEnhanceSet bool

	outputTIFF bool
	outputTIFFSet bool
	outputBPS  int
	outputBPSSet bool
	linear     bool
	gamma      [2]float64
	gammaSet bool

	flip        FlipMode
	flipSet     bool
	noFujiRotate bool
	noFujiRotateSet bool
	cropBox     [4]int
	cropBoxSet  bool

	shotSelect int
	shotSelectSet bool
	rawOptions int
	rawOptionsSet bool
	adjustMaxThreshold float64
	adjustMaxThresholdSet bool

	dngSDK     bool
	dngSDKSet  bool
	arsbits    int
	arsbitsSet bool
	outputFormat    OutputFormat
	outputFormatSet bool

	useFileIO    bool
	useMmap      bool
	useMem       bool
	outputSuffix string
	outputFile   string

	verbose int
	timing  bool

	workingDir  string
	stdout      io.Writer
}

type Option func(*Options)

func WithExecutable(path string) Option {
	return func(o *Options) {
		o.executable = path
	}
}

func WithCameraWhiteBalance() Option {
	return func(o *Options) {
		o.whiteBalanceMode = WBCamera
		o.whiteBalanceModeSet = true
	}
}

func WithAverageWhiteBalance() Option {
	return func(o *Options) {
		o.whiteBalanceMode = WBAverage
		o.whiteBalanceModeSet = true
	}
}

func WithCustomWhiteBalance(r, g, b, g2 float64) Option {
	return func(o *Options) {
		o.customWhiteBalance = [4]float64{r, g, b, g2}
		o.whiteBalanceMode = WBCustom
		o.whiteBalanceModeSet = true
		o.customWhiteBalanceSet = true
	}
}

func WithGreyBoxWhiteBalance(x, y, w, h int) Option {
	return func(o *Options) {
		o.greyBox = [4]int{x, y, w, h}
		o.whiteBalanceMode = WBGreyBox
		o.whiteBalanceModeSet = true
		o.greyBoxSet = true
	}
}

func WithEmbeddedColorMatrix(use bool) Option {
	return func(o *Options) {
		o.useCameraMatrix = use
		o.useCameraMatrixSet = true
	}
}

func WithChromaticAberrationCorrection(red, blue float64) Option {
	return func(o *Options) {
		o.chromaticAberration = [2]float64{red, blue}
		o.chromaticAberrationSet = true
	}
}

func WithOutputColorSpace(space ColorSpace) Option {
	return func(o *Options) {
		o.outputColorSpace = space
		o.outputColorSpaceSet = true
	}
}

func WithOutputProfile(path string) Option {
	return func(o *Options) {
		o.outputProfile = path
	}
}

func WithCameraProfile(path string) Option {
	return func(o *Options) {
		o.cameraProfile = path
	}
}

func WithBadPixelsFile(path string) Option {
	return func(o *Options) {
		o.badPixels = path
	}
}

func WithDarkFrame(path string) Option {
	return func(o *Options) {
		o.darkFrame = path
	}
}

func WithDarkness(level int) Option {
	return func(o *Options) {
		o.darkness = level
		o.darknessSet = true
	}
}

func WithSaturation(level int) Option {
	return func(o *Options) {
		o.saturation = level
		o.saturationSet = true
	}
}

func WithBrightness(brightness float64) Option {
	return func(o *Options) {
		o.brightness = brightness
		o.brightnessSet = true
	}
}

func WithHighlightMode(mode HighlightMode) Option {
	return func(o *Options) {
		o.highlightMode = mode
		o.highlightModeSet = true
	}
}

func WithNoAutoBrightness() Option {
	return func(o *Options) {
		o.noAutoBright = true
		o.noAutoBrightSet = true
	}
}

func WithExposureCorrection(shift, preserve float64) Option {
	return func(o *Options) {
		o.exposureShift = shift
		o.exposurePreserve = preserve
		o.exposureCorrectionSet = true
	}
}

func WithInterpolationQuality(quality InterpolationQuality) Option {
	return func(o *Options) {
		o.quality = quality
		o.qualitySet = true
	}
}

func WithHalfSize() Option {
	return func(o *Options) {
		o.halfSize = true
		o.halfSizeSet = true
	}
}

func WithFourColorRGB() Option {
	return func(o *Options) {
		o.fourColorRGB = true
		o.fourColorRGBSet = true
	}
}

func WithMedianFilter(passes int) Option {
	return func(o *Options) {
		o.medianPasses = passes
		o.medianPassesSet = true
	}
}

func WithGreenMatching() Option {
	return func(o *Options) {
		o.greenMatching = true
		o.greenMatchingSet = true
	}
}

func WithNoInterpolation() Option {
	return func(o *Options) {
		o.noInterpolation = true
		o.noInterpolationSet = true
	}
}

func WithWaveletDenoising(threshold float64) Option {
	return func(o *Options) {
		o.noiseThreshold = threshold
		o.noiseThresholdSet = true
	}
}

func WithFBDD(mode FBDDMode) Option {
	return func(o *Options) {
		o.fbddMode = mode
		o.fbddModeSet = true
	}
}

func WithDCBIterations(iterations int) Option {
	return func(o *Options) {
		o.dcbIterations = iterations
		o.dcbIterationsSet = true
	}
}

func WithDCBEnhance() Option {
	return func(o *Options) {
		o.dcbEnhance = true
		o.dcbEnhanceSet = true
	}
}

func WithTIFFOutput() Option {
	return func(o *Options) {
		o.outputTIFF = true
		o.outputTIFFSet = true
	}
}

func With16BitOutput() Option {
	return func(o *Options) {
		o.outputBPS = 16
		o.outputBPSSet = true
	}
}

func WithLinear16Bit() Option {
	return func(o *Options) {
		o.linear = true
		o.outputBPS = 16
		o.outputBPSSet = true
	}
}

func WithGamma(power, toeSlope float64) Option {
	return func(o *Options) {
		o.gamma = [2]float64{power, toeSlope}
		o.gammaSet = true
	}
}

func WithFlip(mode FlipMode) Option {
	return func(o *Options) {
		o.flip = mode
		o.flipSet = true
	}
}

func WithNoFujiRotate() Option {
	return func(o *Options) {
		o.noFujiRotate = true
		o.noFujiRotateSet = true
	}
}

func WithCropBox(x, y, w, h int) Option {
	return func(o *Options) {
		o.cropBox = [4]int{x, y, w, h}
		o.cropBoxSet = true
	}
}

func WithShotSelect(index int) Option {
	return func(o *Options) {
		o.shotSelect = index
		o.shotSelectSet = true
	}
}

func WithRawOptions(options int) Option {
	return func(o *Options) {
		o.rawOptions = options
		o.rawOptionsSet = true
	}
}

func WithAdjustMaxThreshold(threshold float64) Option {
	return func(o *Options) {
		o.adjustMaxThreshold = threshold
		o.adjustMaxThresholdSet = true
	}
}

func WithOutputSuffix(suffix string) Option {
	return func(o *Options) {
		o.outputSuffix = suffix
	}
}

func WithOutputFile(filename string) Option {
	return func(o *Options) {
		o.outputFile = filename
	}
}

func WithFileIO() Option {
	return func(o *Options) {
		o.useFileIO = true
	}
}

func WithMmapIO() Option {
	return func(o *Options) {
		o.useMmap = true
	}
}

func WithMemIO() Option {
	return func(o *Options) {
		o.useMem = true
	}
}

func WithVerbose(level int) Option {
	return func(o *Options) {
		o.verbose = level
	}
}

func WithTiming() Option {
	return func(o *Options) {
		o.timing = true
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

func WithWorkingDir(dir string) Option {
	return func(o *Options) {
		o.workingDir = dir
	}
}

func WithStdout(w io.Writer) Option {
	return func(o *Options) {
		o.stdout = w
	}
}

func WithDNGSDK(enabled bool) Option {
	return func(o *Options) {
		o.dngSDK = enabled
		o.dngSDKSet = true
	}
}

func WithARSBits(bits int) Option {
	return func(o *Options) {
		o.arsbits = bits
		o.arsbitsSet = true
	}
}

func WithOutputFormat(format OutputFormat) Option {
	return func(o *Options) {
		o.outputFormat = format
		o.outputFormatSet = true
	}
}

func defaultOptions() Options {
	return Options{}
}

func (o *Options) logger() *slog.Logger {
	if o.Logger != nil {
		return o.Logger
	}
	return slog.Default()
}
