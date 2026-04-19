package dngconverter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
)

var (
	ErrExecutableNotFound = errors.New("Adobe DNG Converter executable not found")
	ErrInputNotFound      = errors.New("input file not found")
	ErrNoInputFiles       = errors.New("no input files provided")
	ErrConversionFailed   = errors.New("DNG conversion failed")
)

var defaultExecutablePaths = map[string]string{
	"darwin":  "/Applications/Adobe DNG Converter.app/Contents/MacOS/Adobe DNG Converter",
	"windows": `C:\Program Files\Adobe\Adobe DNG Converter\Adobe DNG Converter.exe`,
}

func GetDefaultExecutablePath() string {
	return defaultExecutablePaths[runtime.GOOS]
}

type Converter struct {
	executable  string
	defaults    Options
	versionOnce func() (string, error)
}

func New(opts ...Option) (*Converter, error) {
	cfg := defaultOptions()
	for _, opt := range opts {
		opt(&cfg)
	}

	var execPath string
	if cfg.executableSet {
		execPath = cfg.executable
	} else {
		execPath = GetDefaultExecutablePath()
	}
	if execPath == "" {
		return nil, ErrExecutableNotFound
	}

	if _, err := os.Stat(execPath); err != nil {
		return nil, fmt.Errorf("%w at %s", ErrExecutableNotFound, execPath)
	}

	cfg.executable = execPath

	return &Converter{
		executable:  execPath,
		defaults:    cfg,
		versionOnce: sync.OnceValues(func() (string, error) {
			return readExecutableVersion(execPath)
		}),
	}, nil
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

// Version returns the Adobe DNG Converter version in "major.minor" format (e.g. "17.5").
func (c *Converter) Version() (string, error) {
	return c.versionOnce()
}

func (c *Converter) Convert(ctx context.Context, input string, opts ...Option) error {
	cfg := c.mergeOptions(opts)

	if _, err := os.Stat(input); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: %s", ErrInputNotFound, input)
	} else if err != nil {
		return fmt.Errorf("failed to access input file: %w", err)
	}

	return c.runConvert(ctx, cfg, c.buildArgs(cfg, input))
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

	return c.runConvert(ctx, cfg, c.buildArgs(cfg, inputs...))
}

func (c *Converter) runConvert(ctx context.Context, cfg Options, args []string) error {
	cmd := exec.CommandContext(ctx, c.executable, args...)
	cmd.SysProcAttr = getSysProcAttr()
	cfg.logger().Debug("executing DNG Converter", "args", cmd.Args)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", ErrConversionFailed, string(output))
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

func (c *Converter) buildArgs(cfg Options, inputs ...string) []string {
	args := make([]string, 0, 16)

	if cfg.uncompressedSet && cfg.Uncompressed {
		args = append(args, "-u")
	}
	if cfg.linearSet && cfg.Linear {
		args = append(args, "-l")
	}
	if cfg.compressSet && cfg.Compress {
		args = append(args, "-c")
	}

	if cfg.previewSizeSet {
		switch cfg.PreviewSize {
		case PreviewNone:
			args = append(args, "-p0")
		case PreviewMedium:
			args = append(args, "-p1")
		case PreviewFull:
			args = append(args, "-p2")
		}
	}

	if cfg.embedOriginalSet && cfg.EmbedOriginal {
		args = append(args, "-e")
	}
	if cfg.embedFastLoadSet && cfg.EmbedFastLoad {
		args = append(args, "-fl")
	}

	if cfg.lossySet && cfg.Lossy {
		args = append(args, "-lossy")
	}

	if cfg.multiProcessSet && cfg.MultiProcess {
		args = append(args, "-mp")
	}

	if cfg.OutputDir != "" {
		args = append(args, "-d", cfg.OutputDir)
	}

	if cfg.OutputFilename != "" && len(inputs) == 1 {
		args = append(args, "-o", cfg.OutputFilename)
	}

	if cfg.CameraRawCompat != "" {
		args = append(args, fmt.Sprintf("-cr%s", cfg.CameraRawCompat))
	}

	if cfg.DNGVersion != "" {
		args = append(args, fmt.Sprintf("-dng%s", cfg.DNGVersion))
	}

	if cfg.JXL {
		args = append(args, "-jxl")
	}
	if cfg.JXLDistance > 0 {
		args = append(args, "-jxl_distance", fmt.Sprintf("%.1f", cfg.JXLDistance))
	}
	if cfg.JXLEffort > 0 {
		args = append(args, "-jxl_effort", fmt.Sprintf("%d", cfg.JXLEffort))
	}
	if cfg.LosslessJXL {
		args = append(args, "-losslessJXL")
	}
	if cfg.LossyMosaicJXL {
		args = append(args, "-lossyMosaicJXL")
	}

	// Windows requires at least one option to prevent UI display
	if runtime.GOOS == "windows" && len(args) == 0 {
		args = append(args, "-c")
	}

	args = append(args, inputs...)

	return args
}
