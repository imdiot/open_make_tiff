package dngconverter

import (
	"errors"
	"runtime"
	"slices"
	"testing"
)

func TestBuildArgs(t *testing.T) {
	tests := []struct {
		name   string
		opts   []Option
		inputs []string
		want   []string
	}{
		{
			name:   "no options - uses tool defaults",
			opts:   nil,
			inputs: []string{"test.nef"},
			want: func() []string {
				// Windows requires at least one option to prevent UI display
				if runtime.GOOS == "windows" {
					return []string{"-c", "test.nef"}
				}
				return []string{"test.nef"}
			}(),
		},
		{
			name:   "with compress option",
			opts:   []Option{WithCompress(true)},
			inputs: []string{"test.nef"},
			want:   []string{"-c", "test.nef"},
		},
		{
			name:   "with compress disabled",
			opts:   []Option{WithCompress(false)},
			inputs: []string{"test.nef"},
			want: func() []string {
				if runtime.GOOS == "windows" {
					return []string{"-c", "test.nef"}
				}
				return []string{"test.nef"}
			}(),
		},
		{
			name:   "with output directory",
			opts:   []Option{WithOutputDir("/output")},
			inputs: []string{"test.nef"},
			want:   []string{"-d", "/output", "test.nef"},
		},
		{
			name:   "with lossy compression",
			opts:   []Option{WithLossy(true)},
			inputs: []string{"test.nef"},
			want:   []string{"-lossy", "test.nef"},
		},
		{
			name:   "with multi-process",
			opts:   []Option{WithMultiProcess(true)},
			inputs: []string{"test.nef"},
			want:   []string{"-mp", "test.nef"},
		},
		{
			name:   "with embed fast load",
			opts:   []Option{WithEmbedFastLoad(true)},
			inputs: []string{"test.nef"},
			want:   []string{"-fl", "test.nef"},
		},
		{
			name:   "with DNG version",
			opts:   []Option{WithDNGVersion(DNG16)},
			inputs: []string{"test.nef"},
			want:   []string{"-dng1.6", "test.nef"},
		},
		{
			name:   "with Camera Raw compatibility",
			opts:   []Option{WithCameraRawCompat(CameraRaw153)},
			inputs: []string{"test.nef"},
			want:   []string{"-cr15.3", "test.nef"},
		},
		{
			name:   "with JPEG XL",
			opts:   []Option{WithJXL(true), WithJXLDistance(0.5), WithJXLEffort(7)},
			inputs: []string{"test.nef"},
			want:   []string{"-jxl", "-jxl_distance", "0.5", "-jxl_effort", "7", "test.nef"},
		},
		{
			name:   "with lossless JXL",
			opts:   []Option{WithLosslessJXL(true)},
			inputs: []string{"test.nef"},
			want:   []string{"-jxl", "-losslessJXL", "test.nef"},
		},
		{
			name:   "with lossless JXL disabled",
			opts:   []Option{WithLosslessJXL(false)},
			inputs: []string{"test.nef"},
			want: func() []string {
				if runtime.GOOS == "windows" {
					return []string{"-c", "test.nef"}
				}
				return []string{"test.nef"}
			}(),
		},
		{
			name:   "with lossy mosaic JXL",
			opts:   []Option{WithLossyMosaicJXL(true)},
			inputs: []string{"test.nef"},
			want:   []string{"-jxl", "-lossyMosaicJXL", "test.nef"},
		},
		{
			name:   "with lossy mosaic JXL disabled",
			opts:   []Option{WithLossyMosaicJXL(false)},
			inputs: []string{"test.nef"},
			want: func() []string {
				if runtime.GOOS == "windows" {
					return []string{"-c", "test.nef"}
				}
				return []string{"test.nef"}
			}(),
		},
		{
			name:   "lossless and lossy mosaic JXL combined",
			opts:   []Option{WithLosslessJXL(true), WithLossyMosaicJXL(true)},
			inputs: []string{"test.nef"},
			want:   []string{"-jxl", "-losslessJXL", "-lossyMosaicJXL", "test.nef"},
		},
		{
			name:   "with output filename",
			opts:   []Option{WithOutputFilename("output.dng"), WithOutputDir("/out")},
			inputs: []string{"test.nef"},
			want:   []string{"-d", "/out", "-o", "output.dng", "test.nef"},
		},
		{
			name:   "multiple inputs with multi-process",
			opts:   []Option{WithMultiProcess(true)},
			inputs: []string{"file1.nef", "file2.nef"},
			want:   []string{"-mp", "file1.nef", "file2.nef"},
		},
		{
			name:   "uncompressed output",
			opts:   []Option{WithUncompressed(true)},
			inputs: []string{"test.nef"},
			want:   []string{"-u", "test.nef"},
		},
		{
			name:   "linear output",
			opts:   []Option{WithLinear(true)},
			inputs: []string{"test.nef"},
			want:   []string{"-l", "test.nef"},
		},
		{
			name:   "uncompressed and linear",
			opts:   []Option{WithUncompressed(true), WithLinear(true)},
			inputs: []string{"test.nef"},
			want:   []string{"-u", "-l", "test.nef"},
		},
		{
			name:   "no preview",
			opts:   []Option{WithPreviewSize(PreviewNone)},
			inputs: []string{"test.nef"},
			want:   []string{"-p0", "test.nef"},
		},
		{
			name:   "medium preview (tool default)",
			opts:   []Option{WithPreviewSize(PreviewMedium)},
			inputs: []string{"test.nef"},
			want:   []string{"-p1", "test.nef"},
		},
		{
			name:   "full preview",
			opts:   []Option{WithPreviewSize(PreviewFull)},
			inputs: []string{"test.nef"},
			want:   []string{"-p2", "test.nef"},
		},
		{
			name:   "embed original",
			opts:   []Option{WithEmbedOriginal(true)},
			inputs: []string{"test.nef"},
			want:   []string{"-e", "test.nef"},
		},
		{
			name: "combine multiple options",
			opts: []Option{
				WithCompress(true),
				WithPreviewSize(PreviewNone),
				WithEmbedFastLoad(true),
				WithMultiProcess(true),
			},
			inputs: []string{"test.nef"},
			want:   []string{"-c", "-p0", "-fl", "-mp", "test.nef"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := defaultOptions()
			for _, opt := range tt.opts {
				if opt != nil {
					opt(&cfg)
				}
			}

			c := &Converter{
				executable: "/mock/dngconverter",
				defaults:   cfg,
			}

			got := c.buildArgs(cfg, tt.inputs...)

			if !slices.Equal(got, tt.want) {
				t.Errorf("got  %v\nwant %v", got, tt.want)
			}
		})
	}
}

func TestMergeOptions(t *testing.T) {
	c := &Converter{
		executable: "/mock/dngconverter",
		defaults:   defaultOptions(),
	}

	// Test override with temporary options
	merged := c.mergeOptions([]Option{
		WithLossy(true),
		WithDNGVersion(DNG17),
	})

	if !merged.lossySet {
		t.Error("expected lossySet to be true after WithLossy")
	}

	if !merged.Lossy {
		t.Error("expected Lossy to be true after override")
	}

	// Verify defaults are not modified
	if c.defaults.lossySet {
		t.Error("defaults should not be modified")
	}

	// Verify DNG version was overridden
	if merged.DNGVersion != DNG17 {
		t.Errorf("expected DNGVersion = %s, got %s", DNG17, merged.DNGVersion)
	}
}

func TestPreviewSizeString(t *testing.T) {
	tests := []struct {
		p    PreviewSize
		want string
	}{
		{PreviewNone, "none"},
		{PreviewMedium, "medium"},
		{PreviewFull, "full"},
		{PreviewSize(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.p.String(); got != tt.want {
				t.Errorf("PreviewSize.String() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestGetDefaultExecutablePath(t *testing.T) {
	path := GetDefaultExecutablePath()
	if path == "" {
		t.Error("expected non-empty path")
	}
	t.Logf("Default path: %s", path)
}

func TestIsAvailable(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Skipf("DNG Converter not available: %v", err)
	}

	// Test IsAvailable instance method
	available := c.IsAvailable()
	t.Logf("IsAvailable() = %v", available)

	// Test Executable method
	execPath := c.Executable()
	t.Logf("Executable() = %s", execPath)

	if !available {
		t.Error("expected IsAvailable() to return true when New() succeeds")
	}
	if execPath == "" {
		t.Error("expected Executable() to return non-empty path")
	}
}

func TestConvertErrors(t *testing.T) {
	c := &Converter{
		executable: "/nonexistent/dngconverter",
		defaults:   defaultOptions(),
	}

	ctx := t.Context()

	// Test with non-existent input file
	err := c.Convert(ctx, "/nonexistent/input.nef")
	if err == nil {
		t.Error("expected error for non-existent input file")
	}
	if !errors.Is(err, ErrInputNotFound) {
		t.Errorf("expected ErrInputNotFound, got %v", err)
	}

	// Test ConvertMany with empty inputs
	err = c.ConvertMany(ctx, []string{})
	if err == nil {
		t.Error("expected error for empty inputs")
	}
	if !errors.Is(err, ErrNoInputFiles) {
		t.Errorf("expected ErrNoInputFiles, got %v", err)
	}
}

func TestNoOptionsBehavior(t *testing.T) {
	// Test that no options results in minimal arguments
	c := &Converter{
		executable: "/mock/dngconverter",
		defaults:   defaultOptions(),
	}

	cfg := defaultOptions() // No options set
	got := c.buildArgs(cfg, "test.nef")

	// Windows requires at least one option to prevent UI display
	expected := []string{"test.nef"}
	if runtime.GOOS == "windows" {
		expected = []string{"-c", "test.nef"}
	}
	if !slices.Equal(got, expected) {
		t.Errorf("got  %v\nwant %v", got, expected)
	}
}

func TestJXLOptionPanics(t *testing.T) {
	tests := []struct {
		name    string
		f       func()
		wantPanic bool
	}{
		{"JXLDistance too low", func() { WithJXLDistance(-0.1) }, true},
		{"JXLDistance too high", func() { WithJXLDistance(6.1) }, true},
		{"JXLEffort too low", func() { WithJXLEffort(0) }, true},
		{"JXLEffort too high", func() { WithJXLEffort(10) }, true},
		{"JXLDistance boundary 0.0", func() { WithJXLDistance(0.0) }, false},
		{"JXLDistance boundary 6.0", func() { WithJXLDistance(6.0) }, false},
		{"JXLEffort boundary 1", func() { WithJXLEffort(1) }, false},
		{"JXLEffort boundary 9", func() { WithJXLEffort(9) }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if tt.wantPanic && r == nil {
					t.Error("expected panic")
				}
				if !tt.wantPanic && r != nil {
					t.Errorf("unexpected panic: %v", r)
				}
			}()
			tt.f()
		})
	}
}
