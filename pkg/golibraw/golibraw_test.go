package golibraw

import (
	"os"
	"path/filepath"
	"testing"
)

// testRAWPath returns a test RAW file path from GOLIBRAW_TEST_FILE env or testdata/.
func testRAWPath(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("GOLIBRAW_TEST_FILE"); p != "" {
		return p
	}
	candidates := []string{"DNG.dng", "IMG_8000.CR2", "IMG_1104.CR3"}
	for _, name := range candidates {
		p := filepath.Join("testdata", name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	t.Skip("no test RAW file found in testdata/ and GOLIBRAW_TEST_FILE not set")
	return ""
}

func TestNewClose(t *testing.T) {
	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	if err := rp.Close(); err != nil {
		t.Fatalf("first Close() error: %v", err)
	}
	if err := rp.Close(); err != nil {
		t.Fatalf("second Close() error: %v", err)
	}
}

func TestVersion(t *testing.T) {
	v := Version()
	if v == "" {
		t.Fatal("Version() returned empty string")
	}
	t.Logf("LibRaw version: %s", v)
}

func TestCameraCount(t *testing.T) {
	count := CameraCount()
	if count <= 0 {
		t.Fatalf("CameraCount() = %d, want > 0", count)
	}
	t.Logf("Supported cameras: %d", count)
}

func TestOpenFile(t *testing.T) {
	path := testRAWPath(t)

	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp.Close()

	if err := rp.OpenFile(path); err != nil {
		t.Fatalf("OpenFile() error: %v", err)
	}

	info := rp.GetCameraInfo()
	if info.Make == "" {
		t.Fatal("GetCameraInfo().Make is empty")
	}
	t.Logf("Camera: %s %s", info.Make, info.Model)
}

func TestOpenBuffer(t *testing.T) {
	path := testRAWPath(t)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp.Close()

	if err := rp.OpenBuffer(data); err != nil {
		t.Fatalf("OpenBuffer() error: %v", err)
	}

	info := rp.GetCameraInfo()
	if info.Make == "" {
		t.Fatal("GetCameraInfo().Make is empty after OpenBuffer")
	}
}

func TestUnpackAndProcess(t *testing.T) {
	path := testRAWPath(t)

	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp.Close()

	if err := rp.OpenFile(path); err != nil {
		t.Fatalf("OpenFile() error: %v", err)
	}

	if err := rp.Unpack(); err != nil {
		t.Fatalf("Unpack() error: %v", err)
	}

	sizes := rp.GetImageSizes()
	if sizes.Width == 0 || sizes.Height == 0 {
		t.Fatalf("image dimensions are zero: %dx%d", sizes.Width, sizes.Height)
	}
	t.Logf("Image size: %dx%d (raw: %dx%d)", sizes.Width, sizes.Height, sizes.RawWidth, sizes.RawHeight)

	if err := rp.Process(); err != nil {
		t.Fatalf("Process() error: %v", err)
	}
}

func TestMakeMemImage(t *testing.T) {
	path := testRAWPath(t)

	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp.Close()

	if err := rp.OpenFile(path); err != nil {
		t.Fatalf("OpenFile() error: %v", err)
	}
	if err := rp.Unpack(); err != nil {
		t.Fatalf("Unpack() error: %v", err)
	}
	if err := rp.Process(); err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	img, err := rp.MakeMemImage()
	if err != nil {
		t.Fatalf("MakeMemImage() error: %v", err)
	}

	if img.Width == 0 || img.Height == 0 {
		t.Fatalf("mem image dimensions are zero: %dx%d", img.Width, img.Height)
	}
	if len(img.Data) == 0 {
		t.Fatal("mem image data is empty")
	}

	t.Logf("Mem image: %dx%d, %d colors, %d bits, %d bytes, format=%d",
		img.Width, img.Height, img.Colors, img.Bits, len(img.Data), img.Type)

	tmpDir := t.TempDir()
	ext := ".ppm"
	if img.Type == ImageBitmap {
		ext = ".ppm"
	} else if img.Type == ImageJPEG {
		ext = ".jpg"
	}
	outPath := filepath.Join(tmpDir, "output"+ext)
	if err := os.WriteFile(outPath, img.Data, 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	stat, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("Stat() error: %v", err)
	}
	if stat.Size() == 0 {
		t.Fatal("output file is empty")
	}
	t.Logf("Written %d bytes to %s", stat.Size(), outPath)
}

func TestMakeMemThumb(t *testing.T) {
	path := testRAWPath(t)

	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp.Close()

	if err := rp.OpenFile(path); err != nil {
		t.Fatalf("OpenFile() error: %v", err)
	}
	if err := rp.UnpackThumb(); err != nil {
		t.Skipf("UnpackThumb() error: %v", err)
	}

	img, err := rp.MakeMemThumb()
	if err != nil {
		t.Fatalf("MakeMemThumb() error: %v", err)
	}

	if len(img.Data) == 0 {
		t.Fatal("thumbnail data is empty")
	}
	t.Logf("Thumbnail: %dx%d, format=%d, %d bytes", img.Width, img.Height, img.Type, len(img.Data))
}

func TestMetadata(t *testing.T) {
	path := testRAWPath(t)

	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp.Close()

	if err := rp.OpenFile(path); err != nil {
		t.Fatalf("OpenFile() error: %v", err)
	}

	camera := rp.GetCameraInfo()
	t.Logf("Make: %s", camera.Make)
	t.Logf("Model: %s", camera.Model)
	t.Logf("Software: %s", camera.Software)
	t.Logf("Colors: %d", camera.Colors)

	lens := rp.GetLensInfo()
	t.Logf("Lens: %s %s", lens.LensMake, lens.Lens)
	t.Logf("Focal: %.1f-%.1fmm", lens.MinFocal, lens.MaxFocal)

	shooting := rp.GetShootingParams()
	t.Logf("ISO: %.0f", shooting.ISOSpeed)
	t.Logf("Shutter: %.6f", shooting.Shutter)
	t.Logf("Aperture: f/%.1f", shooting.Aperture)
	t.Logf("Focal: %.1fmm", shooting.FocalLen)
	t.Logf("Timestamp: %v", shooting.Timestamp)

	sizes := rp.GetImageSizes()
	t.Logf("Size: %dx%d (raw %dx%d)", sizes.Width, sizes.Height, sizes.RawWidth, sizes.RawHeight)
}

func TestOptions(t *testing.T) {
	path := testRAWPath(t)

	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp.Close()

	if err := rp.OpenFile(path); err != nil {
		t.Fatalf("OpenFile() error: %v", err)
	}

	if err := rp.ApplyOptions(
		With16BitOutput(),
		WithTIFFOutput(),
		WithCameraWB(),
		WithOutputColorSpace(ColorSpaceRaw),
		WithNoAutoBrightness(),
		WithInterpolationQuality(QualityAHD),
	); err != nil {
		t.Fatalf("ApplyOptions() error: %v", err)
	}

	if err := rp.Unpack(); err != nil {
		t.Fatalf("Unpack() error: %v", err)
	}
	if err := rp.Process(); err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	img, err := rp.MakeMemImage()
	if err != nil {
		t.Fatalf("MakeMemImage() error: %v", err)
	}

	if img.Bits != 16 {
		t.Errorf("Bits = %d, want 16", img.Bits)
	}
	t.Logf("16-bit TIFF image: %dx%d, %d bits", img.Width, img.Height, img.Bits)
}

func TestRecycle(t *testing.T) {
	path := testRAWPath(t)

	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp.Close()

	if err := rp.OpenFile(path); err != nil {
		t.Fatalf("first OpenFile() error: %v", err)
	}
	info1 := rp.GetCameraInfo()

	rp.Recycle()

	if err := rp.OpenFile(path); err != nil {
		t.Fatalf("second OpenFile() error: %v", err)
	}
	info2 := rp.GetCameraInfo()

	if info1.Make != info2.Make {
		t.Errorf("Make mismatch after recycle: %q vs %q", info1.Make, info2.Make)
	}
	if info1.Model != info2.Model {
		t.Errorf("Model mismatch after recycle: %q vs %q", info1.Model, info2.Model)
	}
}

func TestWritePPMTiff(t *testing.T) {
	path := testRAWPath(t)

	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp.Close()

	if err := rp.ApplyOptions(WithTIFFOutput()); err != nil {
		t.Fatalf("ApplyOptions() error: %v", err)
	}

	if err := rp.OpenFile(path); err != nil {
		t.Fatalf("OpenFile() error: %v", err)
	}
	if err := rp.Unpack(); err != nil {
		t.Fatalf("Unpack() error: %v", err)
	}
	if err := rp.Process(); err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "output.tiff")
	if err := rp.WritePPMTiff(outPath); err != nil {
		t.Fatalf("WritePPMTiff() error: %v", err)
	}

	stat, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("Stat() error: %v", err)
	}
	if stat.Size() == 0 {
		t.Fatal("output TIFF is empty")
	}
	t.Logf("Written TIFF: %d bytes", stat.Size())
}

func TestRawParamsOptions(t *testing.T) {
	path := testRAWPath(t)

	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp.Close()

	if err := rp.ApplyOptions(
		WithDNGSDK(0),
		WithUseRawSpeed(0),
		WithRawOptions(0),
	); err != nil {
		t.Fatalf("ApplyOptions(rawparams) error: %v", err)
	}

	if err := rp.OpenFile(path); err != nil {
		t.Fatalf("OpenFile() error: %v", err)
	}
	if err := rp.Unpack(); err != nil {
		t.Fatalf("Unpack() error: %v", err)
	}
	if err := rp.Process(); err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	img, err := rp.MakeMemImage()
	if err != nil {
		t.Fatalf("MakeMemImage() error: %v", err)
	}
	if img.Width == 0 || img.Height == 0 {
		t.Fatalf("image dimensions zero: %dx%d", img.Width, img.Height)
	}
	t.Logf("RawParams: %dx%d, %d bytes", img.Width, img.Height, len(img.Data))
}

func TestWhiteBalanceFields(t *testing.T) {
	path := testRAWPath(t)

	t.Run("CameraWB", func(t *testing.T) {
		rp, err := New()
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}
		defer rp.Close()

		if err := rp.ApplyOptions(WithCameraWB()); err != nil {
			t.Fatalf("ApplyOptions(WithCameraWB) error: %v", err)
		}
		if err := rp.OpenFile(path); err != nil {
			t.Fatalf("OpenFile() error: %v", err)
		}
		if err := rp.Unpack(); err != nil {
			t.Fatalf("Unpack() error: %v", err)
		}
		if err := rp.Process(); err != nil {
			t.Fatalf("Process() error: %v", err)
		}
	})

	t.Run("AutoWB", func(t *testing.T) {
		rp, err := New()
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}
		defer rp.Close()

		if err := rp.ApplyOptions(WithAutoWB()); err != nil {
			t.Fatalf("ApplyOptions(WithAutoWB) error: %v", err)
		}
		if err := rp.OpenFile(path); err != nil {
			t.Fatalf("OpenFile() error: %v", err)
		}
		if err := rp.Unpack(); err != nil {
			t.Fatalf("Unpack() error: %v", err)
		}
		if err := rp.Process(); err != nil {
			t.Fatalf("Process() error: %v", err)
		}
	})

	t.Run("UserMul", func(t *testing.T) {
		rp, err := New()
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}
		defer rp.Close()

		if err := rp.ApplyOptions(WithUserMul(1.0, 1.0, 1.0, 1.0)); err != nil {
			t.Fatalf("ApplyOptions(WithUserMul) error: %v", err)
		}
		if err := rp.OpenFile(path); err != nil {
			t.Fatalf("OpenFile() error: %v", err)
		}
		if err := rp.Unpack(); err != nil {
			t.Fatalf("Unpack() error: %v", err)
		}
		if err := rp.Process(); err != nil {
			t.Fatalf("Process() error: %v", err)
		}
	})
}

func TestEnableDNGSDK(t *testing.T) {
	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp.Close()

	if err := rp.EnableDNGSDK(); err != nil {
		t.Fatalf("EnableDNGSDK() error: %v", err)
	}

	if rp.dngHost == nil {
		t.Fatal("EnableDNGSDK(): dngHost is nil, USE_DNGSDK may not be enabled")
	}
	t.Log("DNG SDK host created successfully")
}

func TestDNGSDKProcess(t *testing.T) {
	path := testRAWPath(t)

	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp.Close()

	if err := rp.EnableDNGSDK(); err != nil {
		t.Fatalf("EnableDNGSDK() error: %v", err)
	}

	if err := rp.ApplyOptions(
		WithDNGSDK(47), // LIBRAW_DNG_DEFAULT
		WithCameraWB(),
	); err != nil {
		t.Fatalf("ApplyOptions() error: %v", err)
	}

	if err := rp.OpenFile(path); err != nil {
		t.Fatalf("OpenFile() error: %v", err)
	}
	if err := rp.Unpack(); err != nil {
		t.Fatalf("Unpack() error: %v", err)
	}
	if err := rp.Process(); err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	img, err := rp.MakeMemImage()
	if err != nil {
		t.Fatalf("MakeMemImage() error: %v", err)
	}
	if img.Width == 0 || img.Height == 0 {
		t.Fatalf("image dimensions zero: %dx%d", img.Width, img.Height)
	}
	t.Logf("DNG SDK processed: %dx%d, %d bytes", img.Width, img.Height, len(img.Data))
}

func TestEnableDNGSDKAfterClose(t *testing.T) {
	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	rp.Close()

	if err := rp.EnableDNGSDK(); err != ErrAlreadyClosed {
		t.Fatalf("EnableDNGSDK after Close() = %v, want ErrAlreadyClosed", err)
	}
}
