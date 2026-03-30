package rawidentify

import (
	"io"
	"log/slog"
)

type OutputMode int

const (
	ModeBasic OutputMode = iota
	ModeVerbose
	ModeWhiteBalance
	ModeUnpackFunction
	ModeSize
)

func (o OutputMode) String() string {
	switch o {
	case ModeBasic:
		return "basic"
	case ModeVerbose:
		return "verbose"
	case ModeWhiteBalance:
		return "whitebalance"
	case ModeUnpackFunction:
		return "unpack"
	case ModeSize:
		return "size"
	default:
		return "unknown"
	}
}

type Options struct {
	executable string
	Logger     *slog.Logger

	mode    OutputMode
	modeSet bool

	verbose    int
	verboseSet bool

	printFrame    bool
	printFrameSet bool

	halfSize    bool
	halfSizeSet bool

	useEmbeddedColor        bool
	useEmbeddedColorSet     bool
	disableEmbeddedColor    bool
	disableEmbeddedColorSet bool

	inputFileList string
	outputFile    string

	stdout io.Writer
}

type Option func(*Options)

func WithVerbose(level int) Option {
	return func(o *Options) {
		o.verbose = level
		o.verboseSet = true
		if level > 0 {
			o.mode = ModeVerbose
			o.modeSet = true
		}
	}
}

func WithWhiteBalance() Option {
	return func(o *Options) {
		o.mode = ModeWhiteBalance
		o.modeSet = true
	}
}

func WithUnpackFunction(printFrame bool) Option {
	return func(o *Options) {
		o.mode = ModeUnpackFunction
		o.modeSet = true
		o.printFrame = printFrame
		o.printFrameSet = true
	}
}

func WithSize(halfSize bool) Option {
	return func(o *Options) {
		o.mode = ModeSize
		o.modeSet = true
		o.halfSize = halfSize
		o.halfSizeSet = true
	}
}

func WithEmbeddedColorMatrix(use bool) Option {
	return func(o *Options) {
		if use {
			o.useEmbeddedColor = true
			o.useEmbeddedColorSet = true
		} else {
			o.disableEmbeddedColor = true
			o.disableEmbeddedColorSet = true
		}
	}
}

func WithInputFileList(filename string) Option {
	return func(o *Options) {
		o.inputFileList = filename
	}
}

func WithOutputFile(filename string) Option {
	return func(o *Options) {
		o.outputFile = filename
	}
}

func WithExecutable(path string) Option {
	return func(o *Options) {
		o.executable = path
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

func WithStdout(w io.Writer) Option {
	return func(o *Options) {
		o.stdout = w
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
