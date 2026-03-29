package tiffcp

import (
	"io"
	"log/slog"
)

type CompressionType int

const (
	CompressionNone CompressionType = iota
	CompressionLZW
	CompressionDeflate
	CompressionJPEG
	CompressionLERC
	CompressionLZMA
	CompressionZSTD
	CompressionWEBP
	CompressionJBIG
	CompressionPackbits
	CompressionCCITTRLE
	CompressionCCITTFAX3
	CompressionCCITTFAX4
	CompressionSGILOG
	CompressionSGILOG24
)

func (c CompressionType) String() string {
	switch c {
	case CompressionNone:
		return "none"
	case CompressionLZW:
		return "lzw"
	case CompressionDeflate:
		return "zip"
	case CompressionJPEG:
		return "jpeg"
	case CompressionLERC:
		return "lerc"
	case CompressionLZMA:
		return "lzma"
	case CompressionZSTD:
		return "zstd"
	case CompressionWEBP:
		return "webp"
	case CompressionJBIG:
		return "jbig"
	case CompressionPackbits:
		return "packbits"
	case CompressionCCITTRLE:
		return "rle"
	case CompressionCCITTFAX3:
		return "g3"
	case CompressionCCITTFAX4:
		return "g4"
	case CompressionSGILOG:
		return "sgilog"
	case CompressionSGILOG24:
		return "sgilog24"
	default:
		return "unknown"
	}
}

type PlanarConfig int

const (
	PlanarConfigContig PlanarConfig = iota
	PlanarConfigSeparate
)

func (p PlanarConfig) String() string {
	switch p {
	case PlanarConfigContig:
		return "contig"
	case PlanarConfigSeparate:
		return "separate"
	default:
		return "unknown"
	}
}

type ByteOrder int

const (
	ByteOrderNative ByteOrder = iota
	ByteOrderBig
	ByteOrderLittle
)

func (b ByteOrder) String() string {
	switch b {
	case ByteOrderNative:
		return "native"
	case ByteOrderBig:
		return "big"
	case ByteOrderLittle:
		return "little"
	default:
		return "unknown"
	}
}

type FillOrder int

const (
	FillOrderDefault FillOrder = iota
	FillOrderLSB2MSB
	FillOrderMSB2LSB
)

func (f FillOrder) String() string {
	switch f {
	case FillOrderDefault:
		return "default"
	case FillOrderLSB2MSB:
		return "lsb2msb"
	case FillOrderMSB2LSB:
		return "msb2lsb"
	default:
		return "unknown"
	}
}

type CompressionOptions struct {
	Type CompressionType
	Predictor int
	DeflatePreset int
	JPEGQuality int
	JPEGColorSpace string
	LERCPreset int
	LERCMaxZError float64
	LERCSubCodec int
	LZMAPreset int
	ZSTDLevel int
	WEBPLossless bool
	WEBPQuality float64
	JBIGOptions string
}

type Options struct {
	executable string
	Logger     *slog.Logger

	Append            bool
	appendSet         bool
	BigTIFF           bool
	bigTIFFSet        bool
	ByteOrder         ByteOrder
	byteOrderSet      bool
	IgnoreErrors      bool
	ignoreErrorsSet   bool
	DisableMmap       bool
	disableMmapSet    bool
	DisableStripChop  bool
	disableStripChopSet bool

	Compression CompressionOptions

	OutputStrips  bool
	outputStripsSet bool
	OutputTiles   bool
	outputTilesSet bool
	RowsPerStrip  int
	rowsPerStripSet bool
	TileWidth     int
	tileWidthSet  bool
	TileLength    int
	tileLengthSet bool
	PlanarConfig  PlanarConfig
	planarConfigSet bool

	FillOrder   FillOrder
	fillOrderSet bool

	ImageIndex        int
	imageIndexSet     bool
	FormatSpecifier   string
	formatSpecifierSet bool
	CommaSeparator    string
	commaSeparatorSet bool

	stdout      io.Writer
	stderr      io.Writer
	checkStderr bool
}

type Option func(*Options)

func WithExecutable(path string) Option {
	return func(o *Options) {
		o.executable = path
	}
}

func WithAppend(append bool) Option {
	return func(o *Options) {
		o.Append = append
		o.appendSet = true
	}
}

func WithBigTIFF(bigTIFF bool) Option {
	return func(o *Options) {
		o.BigTIFF = bigTIFF
		o.bigTIFFSet = true
	}
}

func WithByteOrder(order ByteOrder) Option {
	return func(o *Options) {
		o.ByteOrder = order
		o.byteOrderSet = true
	}
}

func WithIgnoreErrors(ignore bool) Option {
	return func(o *Options) {
		o.IgnoreErrors = ignore
		o.ignoreErrorsSet = true
	}
}

func WithDisableMmap(disable bool) Option {
	return func(o *Options) {
		o.DisableMmap = disable
		o.disableMmapSet = true
	}
}

func WithDisableStripChop(disable bool) Option {
	return func(o *Options) {
		o.DisableStripChop = disable
		o.disableStripChopSet = true
	}
}

func WithNoCompression() Option {
	return func(o *Options) {
		o.Compression = CompressionOptions{Type: CompressionNone}
	}
}

func WithLZWCompression(predictor int) Option {
	return func(o *Options) {
		o.Compression = CompressionOptions{
			Type:      CompressionLZW,
			Predictor: predictor,
		}
	}
}

func WithDeflateCompression(preset, predictor int) Option {
	return func(o *Options) {
		o.Compression = CompressionOptions{
			Type:           CompressionDeflate,
			DeflatePreset:  preset,
			Predictor:      predictor,
		}
	}
}

func WithJPEGCompression(quality int) Option {
	return func(o *Options) {
		o.Compression = CompressionOptions{
			Type:        CompressionJPEG,
			JPEGQuality: quality,
		}
	}
}

func WithJPEGLossless() Option {
	return func(o *Options) {
		o.Compression = CompressionOptions{
			Type:           CompressionJPEG,
			JPEGQuality:    100,
			JPEGColorSpace: "r",
		}
	}
}

func WithJPEGRGB(quality int) Option {
	return func(o *Options) {
		o.Compression = CompressionOptions{
			Type:           CompressionJPEG,
			JPEGQuality:    quality,
			JPEGColorSpace: "r",
		}
	}
}

func WithLERCCompression(preset int, maxZError float64) Option {
	return func(o *Options) {
		o.Compression = CompressionOptions{
			Type:         CompressionLERC,
			LERCPreset:   preset,
			LERCMaxZError: maxZError,
		}
	}
}

func WithLERCDeflate(maxZError float64) Option {
	return func(o *Options) {
		o.Compression = CompressionOptions{
			Type:         CompressionLERC,
			LERCMaxZError: maxZError,
			LERCSubCodec: 1,
		}
	}
}

func WithLERCZSTD(maxZError float64) Option {
	return func(o *Options) {
		o.Compression = CompressionOptions{
			Type:         CompressionLERC,
			LERCMaxZError: maxZError,
			LERCSubCodec: 2,
		}
	}
}

func WithLZMACompression(preset int) Option {
	return func(o *Options) {
		o.Compression = CompressionOptions{
			Type:       CompressionLZMA,
			LZMAPreset: preset,
		}
	}
}

func WithZSTDCompression(level int) Option {
	return func(o *Options) {
		o.Compression = CompressionOptions{
			Type:      CompressionZSTD,
			ZSTDLevel: level,
		}
	}
}

func WithWEBPLossless() Option {
	return func(o *Options) {
		o.Compression = CompressionOptions{
			Type:         CompressionWEBP,
			WEBPLossless: true,
		}
	}
}

func WithWEBPCompression(quality float64) Option {
	return func(o *Options) {
		o.Compression = CompressionOptions{
			Type:        CompressionWEBP,
			WEBPQuality: quality,
		}
	}
}

func WithJBIGCompression() Option {
	return func(o *Options) {
		o.Compression = CompressionOptions{
			Type: CompressionJBIG,
		}
	}
}

func WithJBIGCompressionOptions(opts string) Option {
	return func(o *Options) {
		o.Compression = CompressionOptions{
			Type:        CompressionJBIG,
			JBIGOptions: opts,
		}
	}
}

func WithPackbitsCompression() Option {
	return func(o *Options) {
		o.Compression = CompressionOptions{
			Type: CompressionPackbits,
		}
	}
}

func WithCCITTFAX3Compression() Option {
	return func(o *Options) {
		o.Compression = CompressionOptions{
			Type: CompressionCCITTFAX3,
		}
	}
}

func WithCCITTFAX3CompressionOptions(opts string) Option {
	return func(o *Options) {
		o.Compression = CompressionOptions{
			Type:        CompressionCCITTFAX3,
			JBIGOptions: opts,
		}
	}
}

func WithCCITTFAX4Compression() Option {
	return func(o *Options) {
		o.Compression = CompressionOptions{
			Type: CompressionCCITTFAX4,
		}
	}
}

func WithSGILOGCompression() Option {
	return func(o *Options) {
		o.Compression = CompressionOptions{
			Type: CompressionSGILOG,
		}
	}
}

func WithStrippedOutput(rowsPerStrip int) Option {
	return func(o *Options) {
		o.OutputStrips = true
		o.outputStripsSet = true
		if rowsPerStrip > 0 {
			o.RowsPerStrip = rowsPerStrip
			o.rowsPerStripSet = true
		}
	}
}

func WithTiledOutput() Option {
	return func(o *Options) {
		o.OutputTiles = true
		o.outputTilesSet = true
	}
}

func WithTileSize(width, length int) Option {
	return func(o *Options) {
		o.OutputTiles = true
		o.outputTilesSet = true
		o.TileWidth = width
		o.tileWidthSet = true
		o.TileLength = length
		o.tileLengthSet = true
	}
}

func WithPlanarConfig(config PlanarConfig) Option {
	return func(o *Options) {
		o.PlanarConfig = config
		o.planarConfigSet = true
	}
}

func WithFillOrder(order FillOrder) Option {
	return func(o *Options) {
		o.FillOrder = order
		o.fillOrderSet = true
	}
}

func WithExtractImage(index int) Option {
	return func(o *Options) {
		o.ImageIndex = index
		o.imageIndexSet = true
	}
}

func WithFormatSpecifier(specifier string) Option {
	return func(o *Options) {
		o.FormatSpecifier = specifier
		o.formatSpecifierSet = true
	}
}

func WithCommaSeparator(separator string) Option {
	return func(o *Options) {
		o.CommaSeparator = separator
		o.commaSeparatorSet = true
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

func WithStderr(w io.Writer) Option {
	return func(o *Options) {
		o.stderr = w
	}
}

func WithCheckStderr(check bool) Option {
	return func(o *Options) {
		o.checkStderr = check
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
