package tiffcp

import (
	"testing"
)

func TestCompressionTypeString(t *testing.T) {
	tests := []struct {
		name     string
		ct       CompressionType
		expected string
	}{
		{"None", CompressionNone, "none"},
		{"LZW", CompressionLZW, "lzw"},
		{"Deflate", CompressionDeflate, "zip"},
		{"JPEG", CompressionJPEG, "jpeg"},
		{"LERC", CompressionLERC, "lerc"},
		{"LZMA", CompressionLZMA, "lzma"},
		{"ZSTD", CompressionZSTD, "zstd"},
		{"WEBP", CompressionWEBP, "webp"},
		{"JBIG", CompressionJBIG, "jbig"},
		{"Packbits", CompressionPackbits, "packbits"},
		{"CCITTFAX3", CompressionCCITTFAX3, "g3"},
		{"CCITTFAX4", CompressionCCITTFAX4, "g4"},
		{"SGILOG", CompressionSGILOG, "sgilog"},
		{"Unknown", CompressionType(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ct.String(); got != tt.expected {
				t.Errorf("CompressionType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPlanarConfigString(t *testing.T) {
	tests := []struct {
		name     string
		pc       PlanarConfig
		expected string
	}{
		{"Contig", PlanarConfigContig, "contig"},
		{"Separate", PlanarConfigSeparate, "separate"},
		{"Unknown", PlanarConfig(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pc.String(); got != tt.expected {
				t.Errorf("PlanarConfig.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestByteOrderString(t *testing.T) {
	tests := []struct {
		name     string
		bo       ByteOrder
		expected string
	}{
		{"Native", ByteOrderNative, "native"},
		{"Big", ByteOrderBig, "big"},
		{"Little", ByteOrderLittle, "little"},
		{"Unknown", ByteOrder(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bo.String(); got != tt.expected {
				t.Errorf("ByteOrder.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFillOrderString(t *testing.T) {
	tests := []struct {
		name     string
		fo       FillOrder
		expected string
	}{
		{"Default", FillOrderDefault, "default"},
		{"LSB2MSB", FillOrderLSB2MSB, "lsb2msb"},
		{"MSB2LSB", FillOrderMSB2LSB, "msb2lsb"},
		{"Unknown", FillOrder(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fo.String(); got != tt.expected {
				t.Errorf("FillOrder.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBuildCompressionArgs(t *testing.T) {
	c := &Converter{}

	tests := []struct {
		name     string
		opts     CompressionOptions
		expected []string
	}{
		{
			name:     "None",
			opts:     CompressionOptions{Type: CompressionNone},
			expected: []string{"-c", "none"},
		},
		{
			name:     "LZW",
			opts:     CompressionOptions{Type: CompressionLZW},
			expected: []string{"-c", "lzw"},
		},
		{
			name:     "LZW with predictor",
			opts:     CompressionOptions{Type: CompressionLZW, Predictor: 2},
			expected: []string{"-c", "lzw:2"},
		},
		{
			name:     "Deflate",
			opts:     CompressionOptions{Type: CompressionDeflate},
			expected: []string{"-c", "zip"},
		},
		{
			name:     "Deflate with predictor and preset",
			opts:     CompressionOptions{Type: CompressionDeflate, Predictor: 2, DeflatePreset: 9},
			expected: []string{"-c", "zip:2:p9"},
		},
		{
			name:     "JPEG",
			opts:     CompressionOptions{Type: CompressionJPEG, JPEGQuality: 75},
			expected: []string{"-c", "jpeg:75"},
		},
		{
			name:     "JPEG RGB",
			opts:     CompressionOptions{Type: CompressionJPEG, JPEGQuality: 90, JPEGColorSpace: "r"},
			expected: []string{"-c", "jpeg:r:90"},
		},
		{
			name:     "LERC",
			opts:     CompressionOptions{Type: CompressionLERC, LERCMaxZError: 0.5},
			expected: []string{"-c", "lerc:0.500000"},
		},
		{
			name:     "LERC with preset and maxZError",
			opts:     CompressionOptions{Type: CompressionLERC, LERCPreset: 2, LERCMaxZError: 0.1},
			expected: []string{"-c", "lerc:p2:0.100000"},
		},
		{
			name:     "LERC Deflate",
			opts:     CompressionOptions{Type: CompressionLERC, LERCMaxZError: 0.5, LERCSubCodec: 1},
			expected: []string{"-c", "lerc:0.500000:s1"},
		},
		{
			name:     "LERC ZSTD",
			opts:     CompressionOptions{Type: CompressionLERC, LERCMaxZError: 0.5, LERCSubCodec: 2},
			expected: []string{"-c", "lerc:0.500000:s2"},
		},
		{
			name:     "LZMA",
			opts:     CompressionOptions{Type: CompressionLZMA, LZMAPreset: 6},
			expected: []string{"-c", "lzma:p6"},
		},
		{
			name:     "ZSTD",
			opts:     CompressionOptions{Type: CompressionZSTD, ZSTDLevel: 15},
			expected: []string{"-c", "zstd:p15"},
		},
		{
			name:     "WEBP lossless",
			opts:     CompressionOptions{Type: CompressionWEBP, WEBPLossless: true},
			expected: []string{"-c", "webp:lossless"},
		},
		{
			name:     "WEBP quality",
			opts:     CompressionOptions{Type: CompressionWEBP, WEBPQuality: 85.0},
			expected: []string{"-c", "webp:85.0"},
		},
		{
			name:     "JBIG",
			opts:     CompressionOptions{Type: CompressionJBIG},
			expected: []string{"-c", "jbig"},
		},
		{
			name:     "Packbits",
			opts:     CompressionOptions{Type: CompressionPackbits},
			expected: []string{"-c", "packbits"},
		},
		{
			name:     "CCITT FAX3",
			opts:     CompressionOptions{Type: CompressionCCITTFAX3},
			expected: []string{"-c", "g3"},
		},
		{
			name:     "CCITT FAX4",
			opts:     CompressionOptions{Type: CompressionCCITTFAX4},
			expected: []string{"-c", "g4"},
		},
		{
			name:     "SGILOG",
			opts:     CompressionOptions{Type: CompressionSGILOG},
			expected: []string{"-c", "sgilog"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.buildCompressionArgs(tt.opts)
			if len(got) != len(tt.expected) {
				t.Errorf("buildCompressionArgs() length = %v, want %v", len(got), len(tt.expected))
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("buildCompressionArgs()[%d] = %v, want %v", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestBuildArgs(t *testing.T) {
	c := &Converter{}

	tests := []struct {
		name     string
		opts     Options
		inputs   []string
		output   string
		expected []string
	}{
		{
			name:     "basic copy",
			opts:     Options{},
			inputs:   []string{"input.tif"},
			output:   "output.tif",
			expected: []string{"input.tif", "output.tif"},
		},
		{
			name:     "append mode",
			opts:     Options{Append: true, appendSet: true},
			inputs:   []string{"input.tif"},
			output:   "output.tif",
			expected: []string{"-a", "input.tif", "output.tif"},
		},
		{
			name:     "BigTIFF",
			opts:     Options{BigTIFF: true, bigTIFFSet: true},
			inputs:   []string{"input.tif"},
			output:   "output.tif",
			expected: []string{"-8", "input.tif", "output.tif"},
		},
		{
			name:     "big endian",
			opts:     Options{ByteOrder: ByteOrderBig, byteOrderSet: true},
			inputs:   []string{"input.tif"},
			output:   "output.tif",
			expected: []string{"-B", "input.tif", "output.tif"},
		},
		{
			name:     "little endian",
			opts:     Options{ByteOrder: ByteOrderLittle, byteOrderSet: true},
			inputs:   []string{"input.tif"},
			output:   "output.tif",
			expected: []string{"-L", "input.tif", "output.tif"},
		},
		{
			name:     "ignore errors",
			opts:     Options{IgnoreErrors: true, ignoreErrorsSet: true},
			inputs:   []string{"input.tif"},
			output:   "output.tif",
			expected: []string{"-i", "input.tif", "output.tif"},
		},
		{
			name:     "stripped output",
			opts:     Options{OutputStrips: true, outputStripsSet: true},
			inputs:   []string{"input.tif"},
			output:   "output.tif",
			expected: []string{"-s", "input.tif", "output.tif"},
		},
		{
			name:     "tiled output",
			opts:     Options{OutputTiles: true, outputTilesSet: true},
			inputs:   []string{"input.tif"},
			output:   "output.tif",
			expected: []string{"-t", "input.tif", "output.tif"},
		},
		{
			name:     "rows per strip",
			opts:     Options{RowsPerStrip: 64, rowsPerStripSet: true},
			inputs:   []string{"input.tif"},
			output:   "output.tif",
			expected: []string{"-r", "64", "input.tif", "output.tif"},
		},
		{
			name:     "tile size",
			opts:     Options{TileWidth: 256, TileLength: 256, tileWidthSet: true, tileLengthSet: true},
			inputs:   []string{"input.tif"},
			output:   "output.tif",
			expected: []string{"-w", "256", "-l", "256", "input.tif", "output.tif"},
		},
		{
			name:     "planar config contig",
			opts:     Options{PlanarConfig: PlanarConfigContig, planarConfigSet: true},
			inputs:   []string{"input.tif"},
			output:   "output.tif",
			expected: []string{"-p", "contig", "input.tif", "output.tif"},
		},
		{
			name:     "planar config separate",
			opts:     Options{PlanarConfig: PlanarConfigSeparate, planarConfigSet: true},
			inputs:   []string{"input.tif"},
			output:   "output.tif",
			expected: []string{"-p", "separate", "input.tif", "output.tif"},
		},
		{
			name:     "fill order lsb2msb",
			opts:     Options{FillOrder: FillOrderLSB2MSB, fillOrderSet: true},
			inputs:   []string{"input.tif"},
			output:   "output.tif",
			expected: []string{"-f", "lsb2msb", "input.tif", "output.tif"},
		},
		{
			name:     "fill order msb2lsb",
			opts:     Options{FillOrder: FillOrderMSB2LSB, fillOrderSet: true},
			inputs:   []string{"input.tif"},
			output:   "output.tif",
			expected: []string{"-f", "msb2lsb", "input.tif", "output.tif"},
		},
		{
			name:     "image index",
			opts:     Options{ImageIndex: 2, imageIndexSet: true},
			inputs:   []string{"input.tif"},
			output:   "output.tif",
			expected: []string{"input.tif,2", "output.tif"},
		},
		{
			name:     "format specifier",
			opts:     Options{FormatSpecifier: "%0", formatSpecifierSet: true},
			inputs:   []string{"input.tif"},
			output:   "output.tif",
			expected: []string{"input.tif%0", "output.tif"},
		},
		{
			name:     "comma separator",
			opts:     Options{CommaSeparator: "%", commaSeparatorSet: true},
			inputs:   []string{"input.tif"},
			output:   "output.tif",
			expected: []string{"-,=%", "input.tif", "output.tif"},
		},
		{
			name:     "LZW compression",
			opts:     Options{Compression: CompressionOptions{Type: CompressionLZW}},
			inputs:   []string{"input.tif"},
			output:   "output.tif",
			expected: []string{"-c", "lzw", "input.tif", "output.tif"},
		},
		{
			name:     "LZW compression with predictor",
			opts:     Options{Compression: CompressionOptions{Type: CompressionLZW, Predictor: 2}},
			inputs:   []string{"input.tif"},
			output:   "output.tif",
			expected: []string{"-c", "lzw:2", "input.tif", "output.tif"},
		},
		{
			name:     "multiple inputs",
			opts:     Options{},
			inputs:   []string{"input1.tif", "input2.tif"},
			output:   "output.tif",
			expected: []string{"input1.tif", "input2.tif", "output.tif"},
		},
		{
			name: "combined options",
			opts: Options{
				BigTIFF:    true,
				bigTIFFSet: true,
				Compression: CompressionOptions{
					Type:      CompressionLZW,
					Predictor: 2,
				},
			},
			inputs:   []string{"input.tif"},
			output:   "output.tif",
			expected: []string{"-8", "-c", "lzw:2", "input.tif", "output.tif"},
		},
		{
			name: "format specifier with image index",
			opts: Options{
				FormatSpecifier:   "%0",
				formatSpecifierSet: true,
				ImageIndex:        1,
				imageIndexSet:     true,
			},
			inputs:   []string{"input.tif"},
			output:   "output.tif",
			expected: []string{"input.tif%0,1", "output.tif"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.buildArgs(tt.opts, tt.inputs, tt.output)
			if len(got) != len(tt.expected) {
				t.Errorf("buildArgs() length = %v, want %v", len(got), len(tt.expected))
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("buildArgs()[%d] = %v, want %v", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestMergeOptions(t *testing.T) {
	defaults := Options{
		BigTIFF:    false,
		bigTIFFSet: false,
	}

	c := &Converter{
		defaults: defaults,
	}

	tests := []struct {
		name     string
		opts     []Option
		expected Options
	}{
		{
			name:     "no options",
			opts:     nil,
			expected: defaults,
		},
		{
			name:     "with big tiff",
			opts:     []Option{WithBigTIFF(true)},
			expected: Options{BigTIFF: true, bigTIFFSet: true},
		},
		{
			name:     "with LZW compression",
			opts:     []Option{WithLZWCompression(2)},
			expected: Options{Compression: CompressionOptions{Type: CompressionLZW, Predictor: 2}},
		},
		{
			name: "multiple options",
			opts: []Option{
				WithBigTIFF(true),
				WithLZWCompression(2),
				WithByteOrder(ByteOrderBig),
			},
			expected: Options{
				BigTIFF:    true,
				bigTIFFSet: true,
				ByteOrder:  ByteOrderBig,
				byteOrderSet: true,
				Compression: CompressionOptions{
					Type:      CompressionLZW,
					Predictor: 2,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.mergeOptions(tt.opts)
			if got.BigTIFF != tt.expected.BigTIFF {
				t.Errorf("mergeOptions() BigTIFF = %v, want %v", got.BigTIFF, tt.expected.BigTIFF)
			}
			if got.bigTIFFSet != tt.expected.bigTIFFSet {
				t.Errorf("mergeOptions() bigTIFFSet = %v, want %v", got.bigTIFFSet, tt.expected.bigTIFFSet)
			}
			if got.ByteOrder != tt.expected.ByteOrder {
				t.Errorf("mergeOptions() ByteOrder = %v, want %v", got.ByteOrder, tt.expected.ByteOrder)
			}
			if got.Compression.Type != tt.expected.Compression.Type {
				t.Errorf("mergeOptions() Compression.Type = %v, want %v", got.Compression.Type, tt.expected.Compression.Type)
			}
			if got.Compression.Predictor != tt.expected.Compression.Predictor {
				t.Errorf("mergeOptions() Compression.Predictor = %v, want %v", got.Compression.Predictor, tt.expected.Compression.Predictor)
			}
		})
	}
}
