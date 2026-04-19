package golibraw

/*
#include <libraw/libraw.h>
*/
import "C"

type Option func(*options)

type options struct {
	useCameraWB   bool
	useCameraWBSet bool
	useAutoWB     bool
	useAutoWBSet  bool
	userMul       [4]float32
	userMulSet    bool

	// use_camera_matrix: 0=off, 1=on, -1=unset
	useCameraMatrix int

	outputColorSpace    ColorSpace
	outputColorSpaceSet bool
	outputProfile       string
	cameraProfile       string

	highlightMode    HighlightMode
	highlightModeSet bool

	brightness    float32
	brightnessSet bool
	noAutoBright  bool
	noAutoBrightSet bool

	interpolationQuality    InterpolationQuality
	interpolationQualitySet bool
	halfSize    bool
	halfSizeSet bool
	fourColorRGB    bool
	fourColorRGBSet bool
	medianPasses    int
	medianPassesSet bool
	greenMatching    bool
	greenMatchingSet bool

	outputBPS    int
	outputBPSSet bool
	outputTIFF   bool
	outputTIFFSet bool
	gammaPower   float64
	gammaToeSlope float64
	gammaSet     bool

	flip    FlipMode
	flipSet bool
	noFujiRotate    bool
	noFujiRotateSet bool
	cropBox    [4]uint
	cropBoxSet bool

	noiseThreshold    float32
	noiseThresholdSet bool
	fbddMode    FBDDMode
	fbddModeSet bool
	dcbIterations    int
	dcbIterationsSet bool
	dcbEnhance    bool
	dcbEnhanceSet bool

	shotSelect    uint
	shotSelectSet bool
	adjustMaxThreshold  float32
	adjustMaxThresholdSet bool

	userBlack    int
	userBlackSet bool
	userSat      int
	userSatSet   bool

	badPixels string
	darkFrame string

	expShift    float32
	expPreser   float32
	expCorrecSet bool

	noAutoScale    bool
	noAutoScaleSet bool
	noInterpolation    bool
	noInterpolationSet bool

	chromaticAberration    [2]float64
	chromaticAberrationSet bool

	dngSDK       DNGSDKFlags
	dngSDKSet    bool
	useRawSpeed  RawSpeedFlags
	useRawSpeedSet bool
	rawOptions   RawOptions
	rawOptionsSet bool

	greybox              [4]uint
	greyboxSet           bool
	userCBlack           [4]int
	userCBlackSet        bool
	autoBrightThreshold  float32
	autoBrightThresholdSet bool
	phaseOneCorrection   bool
	phaseOneCorrectionSet bool
	outputFlags          int
	outputFlagsSet       bool

	rawSpecials              uint
	rawSpecialsSet           bool
	maxRawMemory             uint
	maxRawMemorySet          bool
	sonyARW2Posterization     int
	sonyARW2PosterizationSet  bool
	coolScanNEFGamma          float32
	coolScanNEFGammaSet       bool
}

// ApplyOptions applies functional options to the processor output params.
// Must be called before Process.
func (rp *RawProcessor) ApplyOptions(opts ...Option) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return err
	}

	cfg := defaultOptions()
	for _, o := range opts {
		o(&cfg)
	}
	rp.freeCStrings()
	applyConfigToHandle(rp.res.handle, &cfg, rp.trackCString)

	return nil
}

func defaultOptions() options {
	return options{}
}

func applyConfigToHandle(handle *C.libraw_data_t, cfg *options, alloc func(string) *C.char) {
	params := &handle.params

	if cfg.useCameraWBSet {
		params.use_camera_wb = boolToCInt(cfg.useCameraWB)
	}

	if cfg.useAutoWBSet {
		params.use_auto_wb = boolToCInt(cfg.useAutoWB)
	}

	if cfg.userMulSet {
		params.user_mul[0] = C.float(cfg.userMul[0])
		params.user_mul[1] = C.float(cfg.userMul[1])
		params.user_mul[2] = C.float(cfg.userMul[2])
		params.user_mul[3] = C.float(cfg.userMul[3])
	}

	if cfg.useCameraMatrix >= 0 {
		params.use_camera_matrix = C.int(cfg.useCameraMatrix)
	}

	if cfg.outputColorSpaceSet {
		params.output_color = C.int(cfg.outputColorSpace)
	}

	if cfg.outputProfile != "" {
		params.output_profile = alloc(cfg.outputProfile)
	}

	if cfg.cameraProfile != "" {
		params.camera_profile = alloc(cfg.cameraProfile)
	}

	if cfg.highlightModeSet {
		params.highlight = C.int(cfg.highlightMode)
	}

	if cfg.brightnessSet {
		params.bright = C.float(cfg.brightness)
	}

	if cfg.noAutoBrightSet {
		params.no_auto_bright = boolToCInt(cfg.noAutoBright)
	}

	if cfg.interpolationQualitySet {
		params.user_qual = C.int(cfg.interpolationQuality)
	}

	if cfg.halfSizeSet {
		params.half_size = boolToCInt(cfg.halfSize)
	}

	if cfg.fourColorRGBSet {
		params.four_color_rgb = boolToCInt(cfg.fourColorRGB)
	}

	if cfg.medianPassesSet {
		params.med_passes = C.int(cfg.medianPasses)
	}

	if cfg.greenMatchingSet {
		params.green_matching = boolToCInt(cfg.greenMatching)
	}

	if cfg.outputBPSSet {
		params.output_bps = C.int(cfg.outputBPS)
	}

	if cfg.outputTIFFSet {
		params.output_tiff = boolToCInt(cfg.outputTIFF)
	}

	if cfg.gammaSet {
		params.gamm[0] = C.double(cfg.gammaPower)
		params.gamm[1] = C.double(cfg.gammaToeSlope)
	}

	if cfg.flipSet {
		params.user_flip = C.int(cfg.flip)
	}

	if cfg.noFujiRotateSet {
		params.use_fuji_rotate = boolToCInt(!cfg.noFujiRotate)
	}

	if cfg.cropBoxSet {
		params.cropbox[0] = C.uint(cfg.cropBox[0])
		params.cropbox[1] = C.uint(cfg.cropBox[1])
		params.cropbox[2] = C.uint(cfg.cropBox[2])
		params.cropbox[3] = C.uint(cfg.cropBox[3])
	}

	if cfg.noiseThresholdSet {
		params.threshold = C.float(cfg.noiseThreshold)
	}

	if cfg.fbddModeSet {
		params.fbdd_noiserd = C.int(cfg.fbddMode)
	}

	if cfg.dcbIterationsSet {
		params.dcb_iterations = C.int(cfg.dcbIterations)
	}

	if cfg.dcbEnhanceSet {
		params.dcb_enhance_fl = boolToCInt(cfg.dcbEnhance)
	}

	if cfg.shotSelectSet {
		handle.rawparams.shot_select = C.uint(cfg.shotSelect)
	}

	if cfg.adjustMaxThresholdSet {
		params.adjust_maximum_thr = C.float(cfg.adjustMaxThreshold)
	}

	if cfg.userBlackSet {
		params.user_black = C.int(cfg.userBlack)
	}

	if cfg.userSatSet {
		params.user_sat = C.int(cfg.userSat)
	}

	if cfg.badPixels != "" {
		params.bad_pixels = alloc(cfg.badPixels)
	}

	if cfg.darkFrame != "" {
		params.dark_frame = alloc(cfg.darkFrame)
	}

	if cfg.expCorrecSet {
		params.exp_correc = 1
		params.exp_shift = C.float(cfg.expShift)
		params.exp_preser = C.float(cfg.expPreser)
	}

	if cfg.noAutoScaleSet {
		params.no_auto_scale = boolToCInt(cfg.noAutoScale)
	}

	if cfg.noInterpolationSet {
		params.no_interpolation = boolToCInt(cfg.noInterpolation)
	}

	if cfg.chromaticAberrationSet {
		params.aber[0] = C.double(cfg.chromaticAberration[0])
		params.aber[1] = C.double(cfg.chromaticAberration[1])
	}

	if cfg.dngSDKSet {
		handle.rawparams.use_dngsdk = C.int(cfg.dngSDK)
	}

	if cfg.useRawSpeedSet {
		handle.rawparams.use_rawspeed = C.int(cfg.useRawSpeed)
	}

	if cfg.rawOptionsSet {
		handle.rawparams.options = C.uint(cfg.rawOptions)
	}

	if cfg.greyboxSet {
		params.greybox[0] = C.uint(cfg.greybox[0])
		params.greybox[1] = C.uint(cfg.greybox[1])
		params.greybox[2] = C.uint(cfg.greybox[2])
		params.greybox[3] = C.uint(cfg.greybox[3])
	}

	if cfg.userCBlackSet {
		params.user_cblack[0] = C.int(cfg.userCBlack[0])
		params.user_cblack[1] = C.int(cfg.userCBlack[1])
		params.user_cblack[2] = C.int(cfg.userCBlack[2])
		params.user_cblack[3] = C.int(cfg.userCBlack[3])
	}

	if cfg.autoBrightThresholdSet {
		params.auto_bright_thr = C.float(cfg.autoBrightThreshold)
	}

	if cfg.phaseOneCorrectionSet {
		params.use_p1_correction = boolToCInt(cfg.phaseOneCorrection)
	}

	if cfg.outputFlagsSet {
		params.output_flags = C.int(cfg.outputFlags)
	}

	if cfg.rawSpecialsSet {
		handle.rawparams.specials = C.uint(cfg.rawSpecials)
	}

	if cfg.maxRawMemorySet {
		handle.rawparams.max_raw_memory_mb = C.uint(cfg.maxRawMemory)
	}

	if cfg.sonyARW2PosterizationSet {
		handle.rawparams.sony_arw2_posterization_thr = C.int(cfg.sonyARW2Posterization)
	}

	if cfg.coolScanNEFGammaSet {
		handle.rawparams.coolscan_nef_gamma = C.float(cfg.coolScanNEFGamma)
	}
}

func boolToCInt(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

func WithCameraWB() Option {
	return func(o *options) {
		o.useCameraWB = true
		o.useCameraWBSet = true
	}
}

func WithAutoWB() Option {
	return func(o *options) {
		o.useAutoWB = true
		o.useAutoWBSet = true
	}
}

// WithUserMul sets params.user_mul (r, g, b, g2).
func WithUserMul(r, g, b, g2 float32) Option {
	return func(o *options) {
		o.userMul = [4]float32{r, g, b, g2}
		o.userMulSet = true
	}
}

func WithEmbeddedColorMatrix(use bool) Option {
	return func(o *options) {
		if use {
			o.useCameraMatrix = 1
		} else {
			o.useCameraMatrix = 0
		}
	}
}

func WithOutputColorSpace(space ColorSpace) Option {
	return func(o *options) {
		o.outputColorSpace = space
		o.outputColorSpaceSet = true
	}
}

func WithOutputProfile(path string) Option {
	return func(o *options) {
		o.outputProfile = path
	}
}

func WithCameraProfile(path string) Option {
	return func(o *options) {
		o.cameraProfile = path
	}
}

func WithHighlightMode(mode HighlightMode) Option {
	return func(o *options) {
		o.highlightMode = mode
		o.highlightModeSet = true
	}
}

func WithBrightness(brightness float32) Option {
	return func(o *options) {
		o.brightness = brightness
		o.brightnessSet = true
	}
}

func WithNoAutoBrightness() Option {
	return func(o *options) {
		o.noAutoBright = true
		o.noAutoBrightSet = true
	}
}

func WithInterpolationQuality(quality InterpolationQuality) Option {
	return func(o *options) {
		o.interpolationQuality = quality
		o.interpolationQualitySet = true
	}
}

func WithHalfSize() Option {
	return func(o *options) {
		o.halfSize = true
		o.halfSizeSet = true
	}
}

func WithFourColorRGB() Option {
	return func(o *options) {
		o.fourColorRGB = true
		o.fourColorRGBSet = true
	}
}

func WithMedianFilter(passes int) Option {
	return func(o *options) {
		o.medianPasses = passes
		o.medianPassesSet = true
	}
}

func WithGreenMatching() Option {
	return func(o *options) {
		o.greenMatching = true
		o.greenMatchingSet = true
	}
}

func With16BitOutput() Option {
	return func(o *options) {
		o.outputBPS = 16
		o.outputBPSSet = true
	}
}

func WithTIFFOutput() Option {
	return func(o *options) {
		o.outputTIFF = true
		o.outputTIFFSet = true
	}
}

// WithGamma sets params.gamm (power, toe_slope).
func WithGamma(power, toeSlope float64) Option {
	return func(o *options) {
		o.gammaPower = power
		o.gammaToeSlope = toeSlope
		o.gammaSet = true
	}
}

func WithFlip(mode FlipMode) Option {
	return func(o *options) {
		o.flip = mode
		o.flipSet = true
	}
}

func WithNoFujiRotate() Option {
	return func(o *options) {
		o.noFujiRotate = true
		o.noFujiRotateSet = true
	}
}

// WithCropBox sets params.cropbox (x, y, width, height).
func WithCropBox(x, y, w, h uint) Option {
	return func(o *options) {
		o.cropBox = [4]uint{x, y, w, h}
		o.cropBoxSet = true
	}
}

func WithWaveletDenoising(threshold float32) Option {
	return func(o *options) {
		o.noiseThreshold = threshold
		o.noiseThresholdSet = true
	}
}

func WithFBDD(mode FBDDMode) Option {
	return func(o *options) {
		o.fbddMode = mode
		o.fbddModeSet = true
	}
}

func WithDCBIterations(iterations int) Option {
	return func(o *options) {
		o.dcbIterations = iterations
		o.dcbIterationsSet = true
	}
}

func WithDCBEnhance() Option {
	return func(o *options) {
		o.dcbEnhance = true
		o.dcbEnhanceSet = true
	}
}

func WithShotSelect(index uint) Option {
	return func(o *options) {
		o.shotSelect = index
		o.shotSelectSet = true
	}
}

func WithAdjustMaxThreshold(threshold float32) Option {
	return func(o *options) {
		o.adjustMaxThreshold = threshold
		o.adjustMaxThresholdSet = true
	}
}

func WithDarkness(level int) Option {
	return func(o *options) {
		o.userBlack = level
		o.userBlackSet = true
	}
}

func WithSaturation(level int) Option {
	return func(o *options) {
		o.userSat = level
		o.userSatSet = true
	}
}

func WithBadPixelsFile(path string) Option {
	return func(o *options) {
		o.badPixels = path
	}
}

func WithDarkFrame(path string) Option {
	return func(o *options) {
		o.darkFrame = path
	}
}

// WithExposureCorrection sets params.exp_shift and exp_preser.
func WithExposureCorrection(shift, preserve float32) Option {
	return func(o *options) {
		o.expShift = shift
		o.expPreser = preserve
		o.expCorrecSet = true
	}
}

func WithNoAutoScale() Option {
	return func(o *options) {
		o.noAutoScale = true
		o.noAutoScaleSet = true
	}
}

func WithNoInterpolation() Option {
	return func(o *options) {
		o.noInterpolation = true
		o.noInterpolationSet = true
	}
}

// WithChromaticAberration sets params.aber (red, blue).
func WithChromaticAberration(red, blue float64) Option {
	return func(o *options) {
		o.chromaticAberration = [2]float64{red, blue}
		o.chromaticAberrationSet = true
	}
}

// WithDNGSDK sets rawparams.use_dngsdk (LIBRAW_DNG_* bitmask).
func WithDNGSDK(flags DNGSDKFlags) Option {
	return func(o *options) {
		o.dngSDK = flags
		o.dngSDKSet = true
	}
}

func WithUseRawSpeed(flags RawSpeedFlags) Option {
	return func(o *options) {
		o.useRawSpeed = flags
		o.useRawSpeedSet = true
	}
}

func WithRawOptions(opts RawOptions) Option {
	return func(o *options) {
		o.rawOptions = opts
		o.rawOptionsSet = true
	}
}

// WithGreyBox sets params.greybox (x1, y1, x2, y2).
func WithGreyBox(x1, y1, x2, y2 uint) Option {
	return func(o *options) {
		o.greybox = [4]uint{x1, y1, x2, y2}
		o.greyboxSet = true
	}
}

// WithUserCBlack sets params.user_cblack (r, g, b, g2).
func WithUserCBlack(r, g, b, g2 int) Option {
	return func(o *options) {
		o.userCBlack = [4]int{r, g, b, g2}
		o.userCBlackSet = true
	}
}

func WithAutoBrightThreshold(thr float32) Option {
	return func(o *options) {
		o.autoBrightThreshold = thr
		o.autoBrightThresholdSet = true
	}
}

func WithPhaseOneCorrection() Option {
	return func(o *options) {
		o.phaseOneCorrection = true
		o.phaseOneCorrectionSet = true
	}
}

func WithOutputFlags(flags int) Option {
	return func(o *options) {
		o.outputFlags = flags
		o.outputFlagsSet = true
	}
}

func WithRawSpecials(flags uint) Option {
	return func(o *options) {
		o.rawSpecials = flags
		o.rawSpecialsSet = true
	}
}

func WithMaxRawMemory(mb uint) Option {
	return func(o *options) {
		o.maxRawMemory = mb
		o.maxRawMemorySet = true
	}
}

func WithSonyARW2Posterization(thr int) Option {
	return func(o *options) {
		o.sonyARW2Posterization = thr
		o.sonyARW2PosterizationSet = true
	}
}

func WithCoolScanNEFGamma(gamma float32) Option {
	return func(o *options) {
		o.coolScanNEFGamma = gamma
		o.coolScanNEFGammaSet = true
	}
}
