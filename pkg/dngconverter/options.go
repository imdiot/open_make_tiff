package dngconverter

import (
	"cmp"
	"log/slog"
)

type PreviewSize int

const (
	PreviewNone PreviewSize = iota
	PreviewMedium
	PreviewFull
)

func (p PreviewSize) String() string {
	switch p {
	case PreviewNone:
		return "none"
	case PreviewMedium:
		return "medium"
	case PreviewFull:
		return "full"
	default:
		return "unknown"
	}
}

type CameraRawVersion string

const (
	CameraRaw54  CameraRawVersion = "5.4"
	CameraRaw66  CameraRawVersion = "6.6"
	CameraRaw71  CameraRawVersion = "7.1"
	CameraRaw112 CameraRawVersion = "11.2"
	CameraRaw124 CameraRawVersion = "12.4"
	CameraRaw132 CameraRawVersion = "13.2"
	CameraRaw140 CameraRawVersion = "14.0"
	CameraRaw153 CameraRawVersion = "15.3"
)

func (v CameraRawVersion) String() string {
	return string(v)
}

type DNGVersion string

const (
	DNG11  DNGVersion = "1.1"
	DNG13  DNGVersion = "1.3"
	DNG14  DNGVersion = "1.4"
	DNG15  DNGVersion = "1.5"
	DNG16  DNGVersion = "1.6"
	DNG17  DNGVersion = "1.7"
	DNG171 DNGVersion = "1.7.1"
)

func (v DNGVersion) String() string {
	return string(v)
}

type Options struct {
	executable    string
	executableSet bool

	Logger *slog.Logger

	Compress        bool
	compressSet     bool
	Uncompressed    bool
	uncompressedSet bool
	Linear          bool
	linearSet       bool

	PreviewSize    PreviewSize
	previewSizeSet bool

	EmbedOriginal    bool
	embedOriginalSet bool
	EmbedFastLoad    bool
	embedFastLoadSet bool

	Lossy    bool
	lossySet bool

	MultiProcess    bool
	multiProcessSet bool

	OutputDir      string
	OutputFilename string

	CameraRawCompat CameraRawVersion
	DNGVersion      DNGVersion

	JXL            bool
	JXLDistance    float64
	JXLEffort      int
	LosslessJXL    bool
	LossyMosaicJXL bool
}

type Option func(*Options)

func WithExecutable(path string) Option {
	return func(o *Options) {
		o.executable = path
		o.executableSet = true
	}
}

func WithCompress(compress bool) Option {
	return func(o *Options) {
		o.Compress = compress
		o.compressSet = true
	}
}

func WithUncompressed(uncompressed bool) Option {
	return func(o *Options) {
		o.Uncompressed = uncompressed
		o.uncompressedSet = true
	}
}

func WithLinear(linear bool) Option {
	return func(o *Options) {
		o.Linear = linear
		o.linearSet = true
	}
}

func WithPreviewSize(size PreviewSize) Option {
	return func(o *Options) {
		o.PreviewSize = size
		o.previewSizeSet = true
	}
}

func WithEmbedOriginal(embed bool) Option {
	return func(o *Options) {
		o.EmbedOriginal = embed
		o.embedOriginalSet = true
	}
}

func WithEmbedFastLoad(embed bool) Option {
	return func(o *Options) {
		o.EmbedFastLoad = embed
		o.embedFastLoadSet = true
	}
}

func WithLossy(lossy bool) Option {
	return func(o *Options) {
		o.Lossy = lossy
		o.lossySet = true
	}
}

func WithMultiProcess(mp bool) Option {
	return func(o *Options) {
		o.MultiProcess = mp
		o.multiProcessSet = true
	}
}

func WithOutputDir(dir string) Option {
	return func(o *Options) {
		o.OutputDir = dir
	}
}

func WithOutputFilename(name string) Option {
	return func(o *Options) {
		o.OutputFilename = name
	}
}

func WithCameraRawCompat(version CameraRawVersion) Option {
	return func(o *Options) {
		o.CameraRawCompat = version
	}
}

func WithDNGVersion(version DNGVersion) Option {
	return func(o *Options) {
		o.DNGVersion = version
	}
}

func WithJXL(enabled bool) Option {
	return func(o *Options) {
		o.JXL = enabled
	}
}

func WithJXLDistance(distance float64) Option {
	if distance < 0.0 || distance > 6.0 {
		panic("jxl_distance must be between 0.0 and 6.0")
	}
	return func(o *Options) {
		o.JXL = true
		o.JXLDistance = distance
	}
}

func WithJXLEffort(effort int) Option {
	if effort < 1 || effort > 9 {
		panic("jxl_effort must be between 1 and 9")
	}
	return func(o *Options) {
		o.JXL = true
		o.JXLEffort = effort
	}
}

func WithLosslessJXL(enabled bool) Option {
	return func(o *Options) {
		o.LosslessJXL = enabled
		if enabled {
			o.JXL = true
		}
	}
}

func WithLossyMosaicJXL(enabled bool) Option {
	return func(o *Options) {
		o.LossyMosaicJXL = enabled
		if enabled {
			o.JXL = true
		}
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

func defaultOptions() Options {
	return Options{}
}

func (o *Options) logger() *slog.Logger {
	return cmp.Or(o.Logger, slog.Default())
}
