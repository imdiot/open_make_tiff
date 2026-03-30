package dcrawemu

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

var (
	ErrExecutableNotFound      = errors.New("dcraw_emu executable not found")
	ErrInputNotFound           = errors.New("input file not found")
	ErrNoInputFiles            = errors.New("no input files provided")
	ErrConversionFailed        = errors.New("raw conversion failed")
	ErrMutuallyExclusiveOptions = errors.New("mutually exclusive options")
)

type Converter struct {
	executable string
	defaults   Options
}

func New(opts ...Option) (*Converter, error) {
	cfg := defaultOptions()
	for _, opt := range opts {
		opt(&cfg)
	}

	execPath := cmp.Or(cfg.executable, GetDefaultExecutablePath())
	if execPath == "" {
		return nil, ErrExecutableNotFound
	}

	if _, err := os.Stat(execPath); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrExecutableNotFound, execPath)
	}

	cfg.executable = execPath

	return &Converter{
		executable: execPath,
		defaults:   cfg,
	}, nil
}

func GetDefaultExecutablePath() string {
	path, err := exec.LookPath("dcraw_emu")
	if err != nil {
		return ""
	}
	return path
}

func (c *Converter) IsAvailable() bool {
	if c.executable == "" {
		return false
	}
	_, err := os.Stat(c.executable)
	return err == nil
}

func (c *Converter) Executable() string {
	return c.executable
}

func (c *Converter) Convert(ctx context.Context, input string, opts ...Option) error {
	cfg := c.mergeOptions(opts)

	if _, err := os.Stat(input); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: %s", ErrInputNotFound, input)
	} else if err != nil {
		return fmt.Errorf("failed to access input file: %w", err)
	}

	if cfg.outputFile != "" && cfg.stdout != nil {
		return fmt.Errorf("%w: WithOutputFile and WithStdout cannot be used together", ErrMutuallyExclusiveOptions)
	}
	if cfg.outputFile != "" && cfg.outputFormatSet {
		return fmt.Errorf("%w: WithOutputFile and WithOutputFormat cannot be used together", ErrMutuallyExclusiveOptions)
	}

	args, err := c.buildArgs(cfg, input)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, c.executable, args...)
	cmd.SysProcAttr = getSysProcAttr()

	if cfg.workingDir != "" {
		cmd.Dir = cfg.workingDir
	}

	if cfg.stdout != nil {
		cmd.Stdout = cfg.stdout
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	cfg.logger().Debug("executing dcraw_emu", "args", cmd.Args)

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		if stderrStr != "" {
			cfg.logger().Error("dcraw_emu stderr", "output", stderrStr)
			return fmt.Errorf("%w: %s", err, stderrStr)
		}
		return err
	}

	return nil
}

func (c *Converter) ConvertMany(ctx context.Context, inputs []string, opts ...Option) error {
	if len(inputs) == 0 {
		return ErrNoInputFiles
	}

	cfg := c.mergeOptions(opts)

	for _, input := range inputs {
		if _, err := os.Stat(input); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrInputNotFound, input)
		} else if err != nil {
			return fmt.Errorf("failed to access input file: %w", err)
		}
	}

	args, err := c.buildArgs(cfg, inputs...)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, c.executable, args...)
	cmd.SysProcAttr = getSysProcAttr()

	if cfg.workingDir != "" {
		cmd.Dir = cfg.workingDir
	}

	if cfg.stdout != nil {
		cmd.Stdout = cfg.stdout
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	cfg.logger().Debug("executing dcraw_emu", "args", cmd.Args)

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		if stderrStr != "" {
			cfg.logger().Error("dcraw_emu stderr", "output", stderrStr)
			return fmt.Errorf("%w: %s", err, stderrStr)
		}
		return err
	}

	return nil
}

func (c *Converter) mergeOptions(opts []Option) Options {
	merged := c.defaults
	for _, opt := range opts {
		opt(&merged)
	}
	return merged
}

func (c *Converter) buildArgs(cfg Options, inputs ...string) ([]string, error) {
	args := make([]string, 0, 32)

	if cfg.whiteBalanceModeSet {
		switch cfg.whiteBalanceMode {
		case WBCamera:
			args = append(args, "-w")
		case WBAverage:
			args = append(args, "-a")
		case WBCustom:
			if cfg.customWhiteBalanceSet {
				wb := cfg.customWhiteBalance
				args = append(args, "-r",
					fmt.Sprintf("%.4f", wb[0]),
					fmt.Sprintf("%.4f", wb[1]),
					fmt.Sprintf("%.4f", wb[2]),
					fmt.Sprintf("%.4f", wb[3]))
			}
		case WBGreyBox:
			if cfg.greyBoxSet {
				gb := cfg.greyBox
				args = append(args, "-A",
					fmt.Sprintf("%d", gb[0]),
					fmt.Sprintf("%d", gb[1]),
					fmt.Sprintf("%d", gb[2]),
					fmt.Sprintf("%d", gb[3]))
			}
		}
	}

	if cfg.useCameraMatrixSet {
		if cfg.useCameraMatrix {
			args = append(args, "+M")
		} else {
			args = append(args, "-M")
		}
	}

	if cfg.chromaticAberrationSet {
		ca := cfg.chromaticAberration
		args = append(args, "-C",
			fmt.Sprintf("%.4f", ca[0]),
			fmt.Sprintf("%.4f", ca[1]))
	}

	if cfg.outputColorSpaceSet {
		args = append(args, "-o", fmt.Sprintf("%d", cfg.outputColorSpace))
	} else if cfg.outputProfile != "" {
		args = append(args, "-o", cfg.outputProfile)
	}

	if cfg.cameraProfile != "" {
		args = append(args, "-p", cfg.cameraProfile)
	}

	if cfg.badPixels != "" {
		args = append(args, "-P", cfg.badPixels)
	}

	if cfg.darkFrame != "" {
		args = append(args, "-K", cfg.darkFrame)
	}

	if cfg.darknessSet {
		args = append(args, "-k", fmt.Sprintf("%d", cfg.darkness))
	}

	if cfg.saturationSet {
		args = append(args, "-S", fmt.Sprintf("%d", cfg.saturation))
	}

	if cfg.brightnessSet {
		args = append(args, "-b", fmt.Sprintf("%.4f", cfg.brightness))
	}

	if cfg.highlightModeSet {
		args = append(args, "-H", fmt.Sprintf("%d", cfg.highlightMode))
	}

	if cfg.noAutoBrightSet && cfg.noAutoBright {
		args = append(args, "-W")
	}

	if cfg.exposureCorrectionSet {
		args = append(args, "-aexpo",
			fmt.Sprintf("%.4f", cfg.exposureShift),
			fmt.Sprintf("%.4f", cfg.exposurePreserve))
	}

	if cfg.qualitySet {
		args = append(args, "-q", fmt.Sprintf("%d", cfg.quality))
	}

	if cfg.halfSizeSet && cfg.halfSize {
		args = append(args, "-h")
	}

	if cfg.fourColorRGBSet && cfg.fourColorRGB {
		args = append(args, "-f")
	}

	if cfg.medianPassesSet {
		args = append(args, "-m", fmt.Sprintf("%d", cfg.medianPasses))
	}

	if cfg.greenMatchingSet && cfg.greenMatching {
		args = append(args, "-G")
	}

	if cfg.noInterpolationSet && cfg.noInterpolation {
		args = append(args, "-disinterp")
	}

	if cfg.noiseThresholdSet {
		args = append(args, "-n", fmt.Sprintf("%.4f", cfg.noiseThreshold))
	}

	if cfg.fbddModeSet {
		args = append(args, "-fbdd", fmt.Sprintf("%d", cfg.fbddMode))
	}

	if cfg.dcbIterationsSet {
		args = append(args, "-dcbi", fmt.Sprintf("%d", cfg.dcbIterations))
	}

	if cfg.dcbEnhanceSet && cfg.dcbEnhance {
		args = append(args, "-dcbe")
	}

	if cfg.outputTIFFSet && cfg.outputTIFF {
		args = append(args, "-T")
	}

	if cfg.outputBPSSet {
		if cfg.linear {
			args = append(args, "-4")
		} else if cfg.outputBPS == 16 {
			args = append(args, "-6")
		}
	}

	if cfg.gammaSet {
		g := cfg.gamma
		args = append(args, "-g",
			fmt.Sprintf("%.4f", g[0]),
			fmt.Sprintf("%.4f", g[1]))
	}

	if cfg.flipSet {
		args = append(args, "-t", fmt.Sprintf("%d", cfg.flip))
	}

	if cfg.noFujiRotateSet && cfg.noFujiRotate {
		args = append(args, "-j")
	}

	if cfg.cropBoxSet {
		cb := cfg.cropBox
		args = append(args, "-B",
			fmt.Sprintf("%d", cb[0]),
			fmt.Sprintf("%d", cb[1]),
			fmt.Sprintf("%d", cb[2]),
			fmt.Sprintf("%d", cb[3]))
	}

	if cfg.shotSelectSet {
		args = append(args, "-s", fmt.Sprintf("%d", cfg.shotSelect))
	}

	if cfg.rawOptionsSet {
		args = append(args, "-R", fmt.Sprintf("%d", cfg.rawOptions))
	}

	if cfg.adjustMaxThresholdSet {
		args = append(args, "-c", fmt.Sprintf("%.4f", cfg.adjustMaxThreshold))
	}

	if cfg.dngSDKSet && cfg.dngSDK {
		args = append(args, "-dngsdk")
	}

	if cfg.arsbitsSet {
		args = append(args, "-arsbits", fmt.Sprintf("%d", cfg.arsbits))
	}

	if cfg.useFileIO {
		args = append(args, "-F")
	} else if cfg.useMmap {
		args = append(args, "-mmap")
	} else if cfg.useMem {
		args = append(args, "-mem")
	}

	if cfg.outputFile != "" {
		args = append(args, "-Z", cfg.outputFile)
	} else if cfg.outputFormatSet {
		switch cfg.outputFormat {
		case OutputFormatTIFF:
			args = append(args, "-Z", "tiff")
		case OutputFormatStdout:
			args = append(args, "-Z", "-")
		}
	} else if cfg.outputSuffix != "" {
		args = append(args, "-Z", cfg.outputSuffix)
	}

	if cfg.verbose > 0 {
		for i := 0; i < cfg.verbose; i++ {
			args = append(args, "-v")
		}
	}

	if cfg.timing {
		args = append(args, "-timing")
	}

	args = append(args, inputs...)

	return args, nil
}
