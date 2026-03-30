package rawidentify

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
	ErrExecutableNotFound    = errors.New("raw-identify executable not found")
	ErrInputNotFound         = errors.New("input file not found")
	ErrNoInputFiles          = errors.New("no input files provided")
	ErrIdentificationFailed  = errors.New("raw identification failed")
	ErrMutuallyExclusiveOpts = errors.New("mutually exclusive output modes")
)

type IdentifyResult struct {
	Filename string
	Make     string
	Model    string
}

type Identifier struct {
	executable string
	defaults   Options
}

func New(opts ...Option) (*Identifier, error) {
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

	return &Identifier{
		executable: execPath,
		defaults:   cfg,
	}, nil
}

func GetDefaultExecutablePath() string {
	path, err := exec.LookPath("raw-identify")
	if err != nil {
		return ""
	}
	return path
}

func (i *Identifier) IsAvailable() bool {
	if i.executable == "" {
		return false
	}
	_, err := os.Stat(i.executable)
	return err == nil
}

func (i *Identifier) Executable() string {
	return i.executable
}

func (i *Identifier) Identify(ctx context.Context, input string, opts ...Option) (string, error) {
	cfg := i.mergeOptions(opts)

	if _, err := os.Stat(input); errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("%w: %s", ErrInputNotFound, input)
	} else if err != nil {
		return "", fmt.Errorf("failed to access input file: %w", err)
	}

	args, err := i.buildArgs(cfg, input)
	if err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, i.executable, args...)
	cmd.SysProcAttr = getSysProcAttr()

	var stdout bytes.Buffer
	if cfg.stdout != nil {
		cmd.Stdout = cfg.stdout
	} else {
		cmd.Stdout = &stdout
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	cfg.logger().Debug("executing raw-identify", "args", cmd.Args)

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		if stderrStr != "" {
			cfg.logger().Error("raw-identify stderr", "output", stderrStr)
			return "", fmt.Errorf("%w: %s", err, stderrStr)
		}
		return "", err
	}

	return stdout.String(), nil
}

func (i *Identifier) IdentifyMany(ctx context.Context, inputs []string, opts ...Option) (string, error) {
	if len(inputs) == 0 {
		return "", ErrNoInputFiles
	}

	cfg := i.mergeOptions(opts)

	for _, input := range inputs {
		if _, err := os.Stat(input); errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %s", ErrInputNotFound, input)
		} else if err != nil {
			return "", fmt.Errorf("failed to access input file: %w", err)
		}
	}

	args, err := i.buildArgs(cfg, inputs...)
	if err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, i.executable, args...)
	cmd.SysProcAttr = getSysProcAttr()

	var stdout bytes.Buffer
	if cfg.stdout != nil {
		cmd.Stdout = cfg.stdout
	} else {
		cmd.Stdout = &stdout
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	cfg.logger().Debug("executing raw-identify", "args", cmd.Args)

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		if stderrStr != "" {
			cfg.logger().Error("raw-identify stderr", "output", stderrStr)
			return "", fmt.Errorf("%w: %s", err, stderrStr)
		}
		return "", err
	}

	return stdout.String(), nil
}

func (i *Identifier) IdentifyAndParse(ctx context.Context, input string, opts ...Option) (*IdentifyResult, error) {
	output, err := i.Identify(ctx, input, opts...)
	if err != nil {
		return nil, err
	}

	return i.parseBasicOutput(input, output)
}

func (i *Identifier) parseBasicOutput(filename, output string) (*IdentifyResult, error) {
	prefix := filename + " is a "
	suffix := " image.\n"

	if !strings.HasPrefix(output, prefix) {
		return nil, fmt.Errorf("unexpected output format: %s", output)
	}

	middle := strings.TrimPrefix(output, prefix)
	middle = strings.TrimSuffix(middle, suffix)

	parts := strings.SplitN(middle, " ", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("cannot parse make/model from: %s", middle)
	}

	return &IdentifyResult{
		Filename: filename,
		Make:     parts[0],
		Model:    parts[1],
	}, nil
}

func (i *Identifier) mergeOptions(opts []Option) Options {
	merged := i.defaults
	for _, opt := range opts {
		opt(&merged)
	}
	return merged
}

func (i *Identifier) buildArgs(cfg Options, inputs ...string) ([]string, error) {
	args := make([]string, 0, 16)

	if err := validateOptions(cfg); err != nil {
		return nil, err
	}

	if cfg.modeSet {
		switch cfg.mode {
		case ModeVerbose:
			for j := 0; j < cfg.verbose; j++ {
				args = append(args, "-v")
			}
		case ModeWhiteBalance:
			args = append(args, "-w")
		case ModeUnpackFunction:
			args = append(args, "-u")
			if cfg.printFrameSet && cfg.printFrame {
				args = append(args, "-f")
			}
		case ModeSize:
			args = append(args, "-s")
			if cfg.halfSizeSet && cfg.halfSize {
				args = append(args, "-h")
			}
		}
	}

	if cfg.useEmbeddedColorSet && cfg.useEmbeddedColor {
		args = append(args, "+M")
	} else if cfg.disableEmbeddedColorSet && cfg.disableEmbeddedColor {
		args = append(args, "-M")
	}

	if cfg.inputFileList != "" {
		args = append(args, "-L", cfg.inputFileList)
	}

	if cfg.outputFile != "" {
		args = append(args, "-o", cfg.outputFile)
	}

	args = append(args, inputs...)

	return args, nil
}

func validateOptions(cfg Options) error {
	if cfg.printFrameSet && cfg.printFrame && (!cfg.modeSet || cfg.mode != ModeUnpackFunction) {
		return fmt.Errorf("%w: -f can only be used with -u", ErrMutuallyExclusiveOpts)
	}

	if cfg.halfSizeSet && cfg.halfSize && (!cfg.modeSet || cfg.mode != ModeSize) {
		return fmt.Errorf("%w: -h can only be used with -s", ErrMutuallyExclusiveOpts)
	}

	if cfg.useEmbeddedColorSet && cfg.disableEmbeddedColorSet {
		return fmt.Errorf("%w: -M and +M cannot be used together", ErrMutuallyExclusiveOpts)
	}

	return nil
}
