package golibraw

/*
#include <libraw/libraw.h>
*/
import "C"

// Option 通过 functional option 模式设置处理参数。
type Option func(*options)

type options struct {
	whiteBalanceMode     WhiteBalanceMode
	whiteBalanceModeSet  bool
	customWhiteBalance   [4]float32
	customWhiteBalanceSet bool

	useCameraMatrix     int // 0=off, 1=on, -1=unset
	useAutoWB           bool
	useAutoWBSet        bool

	outputColorSpace    ColorSpace
	outputColorSpaceSet bool
	outputProfile       string
	cameraProfile       string

	highlightMode       HighlightMode
	highlightModeSet    bool

	brightness          float32
	brightnessSet       bool
	noAutoBright        bool
	noAutoBrightSet     bool

	interpolationQuality InterpolationQuality
	interpolationQualitySet bool
	halfSize            bool
	halfSizeSet         bool
	fourColorRGB        bool
	fourColorRGBSet     bool
	medianPasses        int
	medianPassesSet     bool
	greenMatching       bool
	greenMatchingSet    bool

	outputBPS           int
	outputBPSSet        bool
	outputTIFF          bool
	outputTIFFSet       bool
	gammaPower          float64
	gammaToeSlope       float64
	gammaSet            bool

	flip                FlipMode
	flipSet             bool
	noFujiRotate        bool
	noFujiRotateSet     bool
	cropBox             [4]uint
	cropBoxSet          bool

	noiseThreshold      float32
	noiseThresholdSet   bool
	fbddMode            FBDDMode
	fbddModeSet         bool
	dcbIterations       int
	dcbIterationsSet    bool
	dcbEnhance          bool
	dcbEnhanceSet       bool

	shotSelect          uint
	shotSelectSet       bool
	adjustMaxThreshold  float32
	adjustMaxThresholdSet bool

	userBlack           int
	userBlackSet        bool
	userSat             int
	userSatSet          bool

	badPixels           string
	darkFrame           string

	expShift            float32
	expPreser           float32
	expCorrecSet        bool

	noAutoScale         bool
	noAutoScaleSet      bool
	noInterpolation     bool
	noInterpolationSet  bool

	chromaticAberration [2]float64
	chromaticAberrationSet bool
}

// ApplyOptions 将 functional options 应用到 processor 的输出参数上。
// 必须在 Process 之前调用。
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
	applyConfigToHandle(rp.handle, &cfg, rp.trackCString)

	return nil
}

func defaultOptions() options {
	return options{}
}

// applyConfigToHandle 将 Go options 映射到 C 的 libraw_output_params_t。
// alloc 参数用于分配 C 字符串并跟踪生命周期，避免 use-after-free。
func applyConfigToHandle(handle *C.libraw_data_t, cfg *options, alloc func(string) *C.char) {
	params := &handle.params

	if cfg.whiteBalanceModeSet {
		// 先清除所有 WB 标志
		params.use_camera_wb = 0
		params.use_auto_wb = 0

		switch cfg.whiteBalanceMode {
		case WBCamera:
			params.use_camera_wb = 1
		case WBAverage:
			params.use_auto_wb = 1
		case WBCustom:
			if cfg.customWhiteBalanceSet {
				for i := range 4 {
					C.libraw_set_user_mul(handle, C.int(i), C.float(cfg.customWhiteBalance[i]))
				}
			}
		}
	}

	if cfg.useCameraMatrix >= 0 {
		params.use_camera_matrix = C.int(cfg.useCameraMatrix)
	}

	if cfg.useAutoWBSet {
		if cfg.useAutoWB {
			params.use_auto_wb = 1
		} else {
			params.use_auto_wb = 0
		}
	}

	if cfg.outputColorSpaceSet {
		C.libraw_set_output_color(handle, C.int(cfg.outputColorSpace))
	}

	if cfg.outputProfile != "" {
		params.output_profile = alloc(cfg.outputProfile)
	}

	if cfg.cameraProfile != "" {
		params.camera_profile = alloc(cfg.cameraProfile)
	}

	if cfg.highlightModeSet {
		C.libraw_set_highlight(handle, C.int(cfg.highlightMode))
	}

	if cfg.brightnessSet {
		C.libraw_set_bright(handle, C.float(cfg.brightness))
	}

	if cfg.noAutoBrightSet {
		val := 0
		if cfg.noAutoBright {
			val = 1
		}
		C.libraw_set_no_auto_bright(handle, C.int(val))
	}

	if cfg.interpolationQualitySet {
		C.libraw_set_demosaic(handle, C.int(cfg.interpolationQuality))
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
		C.libraw_set_output_bps(handle, C.int(cfg.outputBPS))
	}

	if cfg.outputTIFFSet {
		val := 0
		if cfg.outputTIFF {
			val = 1
		}
		C.libraw_set_output_tif(handle, C.int(val))
	}

	if cfg.gammaSet {
		C.libraw_set_gamma(handle, 0, C.float(cfg.gammaPower))
		C.libraw_set_gamma(handle, 1, C.float(cfg.gammaToeSlope))
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
		C.libraw_set_fbdd_noiserd(handle, C.int(cfg.fbddMode))
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
		C.libraw_set_adjust_maximum_thr(handle, C.float(cfg.adjustMaxThreshold))
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
}

func boolToCInt(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

// --- Functional Option 函数 ---

// WithCameraWhiteBalance 使用相机的白平衡设置。
func WithCameraWhiteBalance() Option {
	return func(o *options) {
		o.whiteBalanceMode = WBCamera
		o.whiteBalanceModeSet = true
	}
}

// WithAutoWhiteBalance 使用自动白平衡。
func WithAutoWhiteBalance() Option {
	return func(o *options) {
		o.useAutoWB = true
		o.useAutoWBSet = true
	}
}

// WithCustomWhiteBalance 使用自定义白平衡系数 (r, g, b, g2)。
func WithCustomWhiteBalance(r, g, b, g2 float32) Option {
	return func(o *options) {
		o.customWhiteBalance = [4]float32{r, g, b, g2}
		o.customWhiteBalanceSet = true
		o.whiteBalanceMode = WBCustom
		o.whiteBalanceModeSet = true
	}
}

// WithEmbeddedColorMatrix 设置是否使用嵌入式色彩矩阵。
func WithEmbeddedColorMatrix(use bool) Option {
	return func(o *options) {
		if use {
			o.useCameraMatrix = 1
		} else {
			o.useCameraMatrix = 0
		}
	}
}

// WithOutputColorSpace 设置输出色彩空间。
func WithOutputColorSpace(space ColorSpace) Option {
	return func(o *options) {
		o.outputColorSpace = space
		o.outputColorSpaceSet = true
	}
}

// WithOutputProfile 设置输出 ICC Profile 路径。
func WithOutputProfile(path string) Option {
	return func(o *options) {
		o.outputProfile = path
	}
}

// WithCameraProfile 设置相机 ICC Profile 路径。
func WithCameraProfile(path string) Option {
	return func(o *options) {
		o.cameraProfile = path
	}
}

// WithHighlightMode 设置高光处理模式。
func WithHighlightMode(mode HighlightMode) Option {
	return func(o *options) {
		o.highlightMode = mode
		o.highlightModeSet = true
	}
}

// WithBrightness 设置亮度值。
func WithBrightness(brightness float32) Option {
	return func(o *options) {
		o.brightness = brightness
		o.brightnessSet = true
	}
}

// WithNoAutoBrightness 禁用自动亮度调整。
func WithNoAutoBrightness() Option {
	return func(o *options) {
		o.noAutoBright = true
		o.noAutoBrightSet = true
	}
}

// WithInterpolationQuality 设置插值质量。
func WithInterpolationQuality(quality InterpolationQuality) Option {
	return func(o *options) {
		o.interpolationQuality = quality
		o.interpolationQualitySet = true
	}
}

// WithHalfSize 输出半尺寸图像。
func WithHalfSize() Option {
	return func(o *options) {
		o.halfSize = true
		o.halfSizeSet = true
	}
}

// WithFourColorRGB 使用四通道 RGB 插值。
func WithFourColorRGB() Option {
	return func(o *options) {
		o.fourColorRGB = true
		o.fourColorRGBSet = true
	}
}

// WithMedianFilter 设置中值滤波次数。
func WithMedianFilter(passes int) Option {
	return func(o *options) {
		o.medianPasses = passes
		o.medianPassesSet = true
	}
}

// WithGreenMatching 启用绿色通道匹配。
func WithGreenMatching() Option {
	return func(o *options) {
		o.greenMatching = true
		o.greenMatchingSet = true
	}
}

// With16BitOutput 设置 16 位输出。
func With16BitOutput() Option {
	return func(o *options) {
		o.outputBPS = 16
		o.outputBPSSet = true
	}
}

// WithTIFFOutput 设置 TIFF 格式输出。
func WithTIFFOutput() Option {
	return func(o *options) {
		o.outputTIFF = true
		o.outputTIFFSet = true
	}
}

// WithGamma 设置 gamma 值 (power, toe_slope)。
func WithGamma(power, toeSlope float64) Option {
	return func(o *options) {
		o.gammaPower = power
		o.gammaToeSlope = toeSlope
		o.gammaSet = true
	}
}

// WithFlip 设置翻转模式。
func WithFlip(mode FlipMode) Option {
	return func(o *options) {
		o.flip = mode
		o.flipSet = true
	}
}

// WithNoFujiRotate 禁用富士旋转。
func WithNoFujiRotate() Option {
	return func(o *options) {
		o.noFujiRotate = true
		o.noFujiRotateSet = true
	}
}

// WithCropBox 设置裁切区域 (x, y, width, height)。
func WithCropBox(x, y, w, h uint) Option {
	return func(o *options) {
		o.cropBox = [4]uint{x, y, w, h}
		o.cropBoxSet = true
	}
}

// WithWaveletDenoising 设置小波降噪阈值。
func WithWaveletDenoising(threshold float32) Option {
	return func(o *options) {
		o.noiseThreshold = threshold
		o.noiseThresholdSet = true
	}
}

// WithFBDD 设置 FBDD 噪声 Reduction 模式。
func WithFBDD(mode FBDDMode) Option {
	return func(o *options) {
		o.fbddMode = mode
		o.fbddModeSet = true
	}
}

// WithDCBIterations 设置 DCB 插值迭代次数。
func WithDCBIterations(iterations int) Option {
	return func(o *options) {
		o.dcbIterations = iterations
		o.dcbIterationsSet = true
	}
}

// WithDCBEnhance 启用 DCB 增强。
func WithDCBEnhance() Option {
	return func(o *options) {
		o.dcbEnhance = true
		o.dcbEnhanceSet = true
	}
}

// WithShotSelect 选择多帧 RAW 中的指定帧。
func WithShotSelect(index uint) Option {
	return func(o *options) {
		o.shotSelect = index
		o.shotSelectSet = true
	}
}

// WithAdjustMaxThreshold 设置最大值调整阈值。
func WithAdjustMaxThreshold(threshold float32) Option {
	return func(o *options) {
		o.adjustMaxThreshold = threshold
		o.adjustMaxThresholdSet = true
	}
}

// WithDarkness 设置黑电平。
func WithDarkness(level int) Option {
	return func(o *options) {
		o.userBlack = level
		o.userBlackSet = true
	}
}

// WithSaturation 设置饱和度。
func WithSaturation(level int) Option {
	return func(o *options) {
		o.userSat = level
		o.userSatSet = true
	}
}

// WithBadPixelsFile 设置坏点文件路径。
func WithBadPixelsFile(path string) Option {
	return func(o *options) {
		o.badPixels = path
	}
}

// WithDarkFrame 设置暗帧文件路径。
func WithDarkFrame(path string) Option {
	return func(o *options) {
		o.darkFrame = path
	}
}

// WithExposureCorrection 设置曝光修正 (shift, preserve)。
func WithExposureCorrection(shift, preserve float32) Option {
	return func(o *options) {
		o.expShift = shift
		o.expPreser = preserve
		o.expCorrecSet = true
	}
}

// WithNoAutoScale 禁用自动缩放。
func WithNoAutoScale() Option {
	return func(o *options) {
		o.noAutoScale = true
		o.noAutoScaleSet = true
	}
}

// WithNoInterpolation 禁用插值。
func WithNoInterpolation() Option {
	return func(o *options) {
		o.noInterpolation = true
		o.noInterpolationSet = true
	}
}

// WithChromaticAberration 设置色差校正 (red, blue)。
func WithChromaticAberration(red, blue float64) Option {
	return func(o *options) {
		o.chromaticAberration = [2]float64{red, blue}
		o.chromaticAberrationSet = true
	}
}
