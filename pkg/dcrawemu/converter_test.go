package dcrawemu

import (
	"context"
	"errors"
	"fmt"
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
			want:   []string{"test.nef"},
		},
		{
			name:   "with camera white balance",
			opts:   []Option{WithCameraWhiteBalance()},
			inputs: []string{"test.nef"},
			want:   []string{"-w", "test.nef"},
		},
		{
			name:   "with average white balance",
			opts:   []Option{WithAverageWhiteBalance()},
			inputs: []string{"test.nef"},
			want:   []string{"-a", "test.nef"},
		},
		{
			name:   "with custom white balance",
			opts:   []Option{WithCustomWhiteBalance(1.2, 1.0, 1.5, 1.0)},
			inputs: []string{"test.nef"},
			want:   []string{"-r", "1.2000", "1.0000", "1.5000", "1.0000", "test.nef"},
		},
		{
			name:   "with grey box white balance",
			opts:   []Option{WithGreyBoxWhiteBalance(100, 100, 200, 200)},
			inputs: []string{"test.nef"},
			want:   []string{"-A", "100", "100", "200", "200", "test.nef"},
		},
		{
			name:   "with embedded color matrix",
			opts:   []Option{WithEmbeddedColorMatrix(true)},
			inputs: []string{"test.nef"},
			want:   []string{"+M", "test.nef"},
		},
		{
			name:   "without embedded color matrix",
			opts:   []Option{WithEmbeddedColorMatrix(false)},
			inputs: []string{"test.nef"},
			want:   []string{"-M", "test.nef"},
		},
		{
			name:   "with chromatic aberration correction",
			opts:   []Option{WithChromaticAberrationCorrection(1.0, 1.5)},
			inputs: []string{"test.nef"},
			want:   []string{"-C", "1.0000", "1.5000", "test.nef"},
		},
		{
			name:   "with output color space sRGB",
			opts:   []Option{WithOutputColorSpace(ColorSpacesRGB)},
			inputs: []string{"test.nef"},
			want:   []string{"-o", "1", "test.nef"},
		},
		{
			name:   "with output profile",
			opts:   []Option{WithOutputProfile("/path/to/profile.icc")},
			inputs: []string{"test.nef"},
			want:   []string{"-o", "/path/to/profile.icc", "test.nef"},
		},
		{
			name:   "with camera profile embed",
			opts:   []Option{WithCameraProfile("embed")},
			inputs: []string{"test.nef"},
			want:   []string{"-p", "embed", "test.nef"},
		},
		{
			name:   "with TIFF output",
			opts:   []Option{WithTIFFOutput()},
			inputs: []string{"test.nef"},
			want:   []string{"-T", "test.nef"},
		},
		{
			name:   "with 16-bit output",
			opts:   []Option{With16BitOutput()},
			inputs: []string{"test.nef"},
			want:   []string{"-6", "test.nef"},
		},
		{
			name:   "with linear 16-bit output",
			opts:   []Option{WithLinear16Bit()},
			inputs: []string{"test.nef"},
			want:   []string{"-4", "test.nef"},
		},
		{
			name:   "with gamma",
			opts:   []Option{WithGamma(2.222, 4.5)},
			inputs: []string{"test.nef"},
			want:   []string{"-g", "2.2220", "4.5000", "test.nef"},
		},
		{
			name:   "with AHD quality",
			opts:   []Option{WithInterpolationQuality(QualityAHD)},
			inputs: []string{"test.nef"},
			want:   []string{"-q", "3", "test.nef"},
		},
		{
			name:   "with half size",
			opts:   []Option{WithHalfSize()},
			inputs: []string{"test.nef"},
			want:   []string{"-h", "test.nef"},
		},
		{
			name:   "with four color RGB",
			opts:   []Option{WithFourColorRGB()},
			inputs: []string{"test.nef"},
			want:   []string{"-f", "test.nef"},
		},
		{
			name:   "with median filter",
			opts:   []Option{WithMedianFilter(1)},
			inputs: []string{"test.nef"},
			want:   []string{"-m", "1", "test.nef"},
		},
		{
			name:   "with green matching",
			opts:   []Option{WithGreenMatching()},
			inputs: []string{"test.nef"},
			want:   []string{"-G", "test.nef"},
		},
		{
			name:   "with no auto brightness",
			opts:   []Option{WithNoAutoBrightness()},
			inputs: []string{"test.nef"},
			want:   []string{"-W", "test.nef"},
		},
		{
			name:   "with brightness",
			opts:   []Option{WithBrightness(1.2)},
			inputs: []string{"test.nef"},
			want:   []string{"-b", "1.2000", "test.nef"},
		},
		{
			name:   "with highlight mode rebuild",
			opts:   []Option{WithHighlightMode(HighlightRebuild)},
			inputs: []string{"test.nef"},
			want:   []string{"-H", "3", "test.nef"},
		},
		{
			name:   "with flip 180",
			opts:   []Option{WithFlip(Flip180)},
			inputs: []string{"test.nef"},
			want:   []string{"-t", "3", "test.nef"},
		},
		{
			name:   "with no Fuji rotate",
			opts:   []Option{WithNoFujiRotate()},
			inputs: []string{"test.nef"},
			want:   []string{"-j", "test.nef"},
		},
		{
			name:   "with crop box",
			opts:   []Option{WithCropBox(10, 10, 1000, 1000)},
			inputs: []string{"test.nef"},
			want:   []string{"-B", "10", "10", "1000", "1000", "test.nef"},
		},
		{
			name:   "with shot select",
			opts:   []Option{WithShotSelect(0)},
			inputs: []string{"test.nef"},
			want:   []string{"-s", "0", "test.nef"},
		},
		{
			name:   "with wavelet denoising",
			opts:   []Option{WithWaveletDenoising(0.5)},
			inputs: []string{"test.nef"},
			want:   []string{"-n", "0.5000", "test.nef"},
		},
		{
			name:   "with FBDD light",
			opts:   []Option{WithFBDD(FBDDLight)},
			inputs: []string{"test.nef"},
			want:   []string{"-fbdd", "1", "test.nef"},
		},
		{
			name:   "with DCB iterations",
			opts:   []Option{WithDCBIterations(2)},
			inputs: []string{"test.nef"},
			want:   []string{"-dcbi", "2", "test.nef"},
		},
		{
			name:   "with DCB enhance",
			opts:   []Option{WithDCBEnhance()},
			inputs: []string{"test.nef"},
			want:   []string{"-dcbe", "test.nef"},
		},
		{
			name:   "with output suffix",
			opts:   []Option{WithOutputSuffix(".tif")},
			inputs: []string{"test.nef"},
			want:   []string{"-Z", ".tif", "test.nef"},
		},
		{
			name:   "with verbose",
			opts:   []Option{WithVerbose(2)},
			inputs: []string{"test.nef"},
			want:   []string{"-v", "-v", "test.nef"},
		},
		{
			name:   "with timing",
			opts:   []Option{WithTiming()},
			inputs: []string{"test.nef"},
			want:   []string{"-timing", "test.nef"},
		},
		{
			name:   "combine multiple options",
			opts:   []Option{
				WithTIFFOutput(),
				WithCameraWhiteBalance(),
				WithInterpolationQuality(QualityAHD),
			},
			inputs: []string{"test.nef"},
			want:   []string{"-w", "-q", "3", "-T", "test.nef"},
		},
		{
			name:   "multiple input files",
			opts:   []Option{WithTIFFOutput()},
			inputs: []string{"file1.nef", "file2.cr3", "file3.arw"},
			want:   []string{"-T", "file1.nef", "file2.cr3", "file3.arw"},
		},
		{
			name:   "with darkness and saturation",
			opts:   []Option{WithDarkness(10), WithSaturation(10000)},
			inputs: []string{"test.nef"},
			want:   []string{"-k", "10", "-S", "10000", "test.nef"},
		},
		{
			name:   "with exposure correction",
			opts:   []Option{WithExposureCorrection(0.5, 0.8)},
			inputs: []string{"test.nef"},
			want:   []string{"-aexpo", "0.5000", "0.8000", "test.nef"},
		},
	}

	c := &Converter{
		executable: "/mock/dcraw_emu",
		defaults:   defaultOptions(),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := c.mergeOptions(tt.opts)
			got, err := c.buildArgs(cfg, tt.inputs...)
			if err != nil {
				t.Fatalf("buildArgs() error = %v", err)
			}

			if len(got) != len(tt.want) {
				t.Errorf("buildArgs() length = %d, want %d", len(got), len(tt.want))
				t.Errorf("got:  %v", got)
				t.Errorf("want: %v", tt.want)
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("buildArgs()[%d] = %s, want %s", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestMergeOptions(t *testing.T) {
	defaultOpts := []Option{WithTIFFOutput()}
	c, err := New(defaultOpts...)
	if err != nil {
		t.Skipf("cannot create converter: %v", err)
	}

	// Temp options should override defaults
	tempOpts := []Option{WithCameraWhiteBalance()}
	cfg := c.mergeOptions(tempOpts)

	if cfg.whiteBalanceMode != WBCamera {
		t.Errorf("expected whiteBalanceMode = WBCamera, got %v", cfg.whiteBalanceMode)
	}

	// Default options should be preserved
	if !cfg.outputTIFF {
		t.Error("expected default outputTIFF to be preserved")
	}
}

func TestEnumString(t *testing.T) {
	tests := []struct {
		name     string
		enum     fmt.Stringer
		expected string
	}{
		{"WhiteBalanceMode camera", WBCamera, "camera"},
		{"WhiteBalanceMode average", WBAverage, "average"},
		{"InterpolationQuality AHD", QualityAHD, "ahd"},
		{"InterpolationQuality VNG", QualityVNG, "vng"},
		{"HighlightMode blend", HighlightBlend, "blend"},
		{"ColorSpace sRGB", ColorSpacesRGB, "srgb"},
		{"ColorSpace Adobe", ColorSpaceAdobe, "adobe"},
		{"FlipMode 90CCW", Flip90CCW, "90ccw"},
		{"FBDDMode light", FBDDLight, "light"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.enum.String(); got != tt.expected {
				t.Errorf("String() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestGetDefaultExecutablePath(t *testing.T) {
	// Verifies function doesn't panic
	path := GetDefaultExecutablePath()
	t.Logf("GetDefaultExecutablePath() = %s", path)
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "no options - uses default from PATH",
			opts:    nil,
			wantErr: true, // error unless dcraw_emu is in system PATH
		},
		{
			name:    "with non-existent executable path",
			opts:    []Option{WithExecutable("/mock/dcraw_emu")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := New(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && c == nil {
				t.Error("New() should return non-nil converter")
			}
			if !tt.wantErr && c != nil {
				if c.executable == "" {
					t.Error("New() should set executable path")
				}
			}
		})
	}
}

func TestConvertErrors(t *testing.T) {
	c := &Converter{
		executable: "/nonexistent/dcraw_emu",
		defaults:   defaultOptions(),
	}

	ctx := context.Background()

	err := c.Convert(ctx, "/nonexistent/input.nef")
	if err == nil {
		t.Error("expected error for non-existent input file")
	} else if !errors.Is(err, ErrInputNotFound) {
		t.Errorf("expected ErrInputNotFound, got %v", err)
	}

	err = c.ConvertMany(ctx, []string{})
	if err == nil {
		t.Error("expected error for empty input list")
	} else if !errors.Is(err, ErrNoInputFiles) {
		t.Errorf("expected ErrNoInputFiles, got %v", err)
	}
}

func TestConverterMethods(t *testing.T) {
	c := &Converter{
		executable: "/mock/dcraw_emu",
		defaults:   defaultOptions(),
	}

	if got := c.Executable(); got != "/mock/dcraw_emu" {
		t.Errorf("Executable() = %s, want /mock/dcraw_emu", got)
	}

	if c.IsAvailable() {
		t.Error("IsAvailable() should return false for non-existent executable")
	}

	emptyC := &Converter{executable: "", defaults: defaultOptions()}
	if emptyC.IsAvailable() {
		t.Error("IsAvailable() should return false for empty executable path")
	}
	if emptyC.Executable() != "" {
		t.Errorf("Executable() = %s, want empty string", emptyC.Executable())
	}
}
