package tiffcp

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var (
	ErrExecutableNotFound = errors.New("tiffcp executable not found")
	ErrInputNotFound      = errors.New("input file not found")
	ErrOutputNotSpecified = errors.New("output file must be specified")
	ErrNoInputFiles       = errors.New("no input files provided")
	ErrConversionFailed   = errors.New("tiffcp conversion failed")
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
	path, err := exec.LookPath("tiffcp")
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

// Convert processes a single TIFF file.
// Unlike other converters, tiffcp requires explicit output path.
func (c *Converter) Convert(ctx context.Context, input, output string, opts ...Option) error {
	return c.ConvertMany(ctx, []string{input}, output, opts...)
}

// ConvertMany processes multiple TIFF files into a single output.
// Unlike other converters, tiffcp requires explicit output path.
func (c *Converter) ConvertMany(ctx context.Context, inputs []string, output string, opts ...Option) error {
	if len(inputs) == 0 {
		return ErrNoInputFiles
	}

	if output == "" {
		return ErrOutputNotSpecified
	}

	cfg := c.mergeOptions(opts)

	for _, input := range inputs {
		if _, err := os.Stat(input); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrInputNotFound, input)
		} else if err != nil {
			return fmt.Errorf("failed to access input file: %w", err)
		}
	}

	args := c.buildArgs(cfg, inputs, output)

	cmd := exec.CommandContext(ctx, c.executable, args...)
	cmd.SysProcAttr = getSysProcAttr()

	if cfg.stdout != nil {
		cmd.Stdout = cfg.stdout
	}
	if cfg.stderr != nil {
		cmd.Stderr = cfg.stderr
	}

	cfg.logger().Debug("executing tiffcp", "args", cmd.Args)

	if cfg.stdout != nil || cfg.stderr != nil {
		if err := cmd.Run(); err != nil {
			return err
		}
	} else {
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w: %s", ErrConversionFailed, string(output))
		}
	}

	if cfg.checkStderr && cfg.stderr != nil {
		if buf, ok := cfg.stderr.(*bytes.Buffer); ok && buf.Len() > 0 {
			return fmt.Errorf("stderr: %s", buf.String())
		}
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

func (c *Converter) buildArgs(cfg Options, inputs []string, output string) []string {
	args := make([]string, 0, 32)

	if cfg.appendSet && cfg.Append {
		args = append(args, "-a")
	}

	if cfg.bigTIFFSet && cfg.BigTIFF {
		args = append(args, "-8")
	}

	if cfg.byteOrderSet {
		switch cfg.ByteOrder {
		case ByteOrderBig:
			args = append(args, "-B")
		case ByteOrderLittle:
			args = append(args, "-L")
		}
	}

	if cfg.ignoreErrorsSet && cfg.IgnoreErrors {
		args = append(args, "-i")
	}

	if cfg.disableMmapSet && cfg.DisableMmap {
		args = append(args, "-M")
	}

	if cfg.disableStripChopSet && cfg.DisableStripChop {
		args = append(args, "-C")
	}

	if cfg.Compression.Type != 0 {
		args = append(args, c.buildCompressionArgs(cfg.Compression)...)
	}

	if cfg.outputStripsSet && cfg.OutputStrips {
		args = append(args, "-s")
	}

	if cfg.outputTilesSet && cfg.OutputTiles {
		args = append(args, "-t")
	}

	if cfg.rowsPerStripSet {
		args = append(args, "-r", fmt.Sprintf("%d", cfg.RowsPerStrip))
	}

	if cfg.tileWidthSet {
		args = append(args, "-w", fmt.Sprintf("%d", cfg.TileWidth))
	}

	if cfg.tileLengthSet {
		args = append(args, "-l", fmt.Sprintf("%d", cfg.TileLength))
	}

	if cfg.planarConfigSet {
		switch cfg.PlanarConfig {
		case PlanarConfigContig:
			args = append(args, "-p", "contig")
		case PlanarConfigSeparate:
			args = append(args, "-p", "separate")
		}
	}

	if cfg.fillOrderSet {
		switch cfg.FillOrder {
		case FillOrderLSB2MSB:
			args = append(args, "-f", "lsb2msb")
		case FillOrderMSB2LSB:
			args = append(args, "-f", "msb2lsb")
		}
	}

	if cfg.commaSeparatorSet {
		args = append(args, fmt.Sprintf("-,=%s", cfg.CommaSeparator))
	}

	for _, input := range inputs {
		inputSpec := input

		if cfg.formatSpecifierSet && cfg.FormatSpecifier != "" {
			inputSpec = fmt.Sprintf("%s%s", inputSpec, cfg.FormatSpecifier)
		}

		if cfg.imageIndexSet {
			inputSpec = fmt.Sprintf("%s,%d", inputSpec, cfg.ImageIndex)
		}

		args = append(args, inputSpec)
	}

	args = append(args, output)

	return args
}

func (c *Converter) buildCompressionArgs(opts CompressionOptions) []string {
	var args []string

	switch opts.Type {
	case CompressionNone:
		args = append(args, "-c", "none")

	case CompressionLZW:
		if opts.Predictor > 0 {
			args = append(args, "-c", fmt.Sprintf("lzw:%d", opts.Predictor))
		} else {
			args = append(args, "-c", "lzw")
		}

	case CompressionDeflate:
		parts := []string{"zip"}
		if opts.Predictor > 0 {
			parts = append(parts, fmt.Sprintf("%d", opts.Predictor))
		}
		if opts.DeflatePreset > 0 {
			parts = append(parts, fmt.Sprintf("p%d", opts.DeflatePreset))
		}
		args = append(args, "-c", strings.Join(parts, ":"))

	case CompressionJPEG:
		if opts.JPEGColorSpace == "r" {
			if opts.JPEGQuality > 0 {
				args = append(args, "-c", fmt.Sprintf("jpeg:r:%d", opts.JPEGQuality))
			} else {
				args = append(args, "-c", "jpeg:r")
			}
		} else {
			if opts.JPEGQuality > 0 {
				args = append(args, "-c", fmt.Sprintf("jpeg:%d", opts.JPEGQuality))
			} else {
				args = append(args, "-c", "jpeg")
			}
		}

	case CompressionLERC:
		parts := []string{"lerc"}
		if opts.LERCPreset > 0 {
			parts = append(parts, fmt.Sprintf("p%d", opts.LERCPreset))
		}
		if opts.LERCMaxZError > 0 {
			parts = append(parts, fmt.Sprintf("%.6f", opts.LERCMaxZError))
		}
		if opts.LERCSubCodec == 1 {
			parts = append(parts, "s1")
		} else if opts.LERCSubCodec == 2 {
			parts = append(parts, "s2")
		}
		if len(parts) > 1 {
			args = append(args, "-c", strings.Join(parts, ":"))
		} else {
			args = append(args, "-c", "lerc")
		}

	case CompressionLZMA:
		if opts.LZMAPreset > 0 {
			args = append(args, "-c", fmt.Sprintf("lzma:p%d", opts.LZMAPreset))
		} else {
			args = append(args, "-c", "lzma")
		}

	case CompressionZSTD:
		if opts.ZSTDLevel > 0 {
			args = append(args, "-c", fmt.Sprintf("zstd:p%d", opts.ZSTDLevel))
		} else {
			args = append(args, "-c", "zstd")
		}

	case CompressionWEBP:
		if opts.WEBPLossless {
			args = append(args, "-c", "webp:lossless")
		} else if opts.WEBPQuality > 0 {
			args = append(args, "-c", fmt.Sprintf("webp:%.1f", opts.WEBPQuality))
		} else {
			args = append(args, "-c", "webp")
		}

	case CompressionJBIG:
		if opts.JBIGOptions != "" {
			args = append(args, "-c", fmt.Sprintf("jbig:%s", opts.JBIGOptions))
		} else {
			args = append(args, "-c", "jbig")
		}

	case CompressionPackbits:
		args = append(args, "-c", "packbits")

	case CompressionCCITTFAX3:
		if opts.JBIGOptions != "" {
			args = append(args, "-c", fmt.Sprintf("g3:%s", opts.JBIGOptions))
		} else {
			args = append(args, "-c", "g3")
		}

	case CompressionCCITTFAX4:
		args = append(args, "-c", "g4")

	case CompressionSGILOG:
		args = append(args, "-c", "sgilog")

	case CompressionCCITTRLE:
		args = append(args, "-c", "rle")

	default:
	}

	return args
}
