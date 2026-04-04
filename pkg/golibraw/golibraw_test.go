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
	for _, name := range []string{"DNG.dng", "IMG_8000.CR2", "IMG_1104.CR3"} {
		p := filepath.Join("testdata", name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	t.Skip("no test RAW file found in testdata/ and GOLIBRAW_TEST_FILE not set")
	return ""
}

// openTestRAW creates a RawProcessor and opens the test file.
func openTestRAW(t *testing.T, opts ...Option) *RawProcessor {
	t.Helper()
	rp, err := New(opts...)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	t.Cleanup(func() { rp.Close() })
	if err := rp.OpenFile(testRAWPath(t)); err != nil {
		t.Fatalf("OpenFile() error: %v", err)
	}
	return rp
}

// openAndProcess creates a RawProcessor, opens, unpacks, and processes the test file.
func openAndProcess(t *testing.T, opts ...Option) *RawProcessor {
	t.Helper()
	rp := openTestRAW(t, opts...)
	if err := rp.Unpack(); err != nil {
		t.Fatalf("Unpack() error: %v", err)
	}
	if err := rp.Process(); err != nil {
		t.Fatalf("Process() error: %v", err)
	}
	return rp
}

// ── Processor lifecycle ──────────────────────────────────────────

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
	if v := Version(); v == "" {
		t.Fatal("Version() returned empty string")
	} else {
		t.Logf("LibRaw %s, %d cameras", v, CameraCount())
	}
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

	if info1.Make != info2.Make || info1.Model != info2.Model {
		t.Errorf("metadata mismatch after recycle: (%q,%q) vs (%q,%q)",
			info1.Make, info1.Model, info2.Make, info2.Model)
	}
}

func TestOpenBuffer(t *testing.T) {
	path := testRAWPath(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	rp := openTestRAW(t) // just for reference
	rp.Close()

	rp2, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp2.Close()

	if err := rp2.OpenBuffer(data); err != nil {
		t.Fatalf("OpenBuffer() error: %v", err)
	}
	if info := rp2.GetCameraInfo(); info.Make == "" {
		t.Fatal("GetCameraInfo().Make is empty after OpenBuffer")
	}
}

// ── Identify (metadata-only, no Unpack needed) ───────────────────

func TestIdentify(t *testing.T) {
	rp := openTestRAW(t)

	camera := rp.GetCameraInfo()
	if camera.Make == "" {
		t.Fatal("CameraInfo.Make is empty")
	}
	t.Logf("Camera: %s %s (maker=%d, normalized=%s/%s)",
		camera.Make, camera.Model, camera.MakerIndex,
		camera.NormalizedMake, camera.NormalizedModel)

	t.Run("ShootingInfo", func(t *testing.T) {
		si := rp.GetShootingInfo()
		t.Logf("DriveMode=%d FocusMode=%d MeteringMode=%d AFPoint=%d",
			si.DriveMode, si.FocusMode, si.MeteringMode, si.AFPoint)
		t.Logf("ExposureMode=%d ExposureProgram=%d ImageStabilization=%d",
			si.ExposureMode, si.ExposureProgram, si.ImageStabilization)
		t.Logf("BodySerial=%q InternalBodySerial=%q", si.BodySerial, si.InternalBodySerial)
	})

	t.Run("LensInfo", func(t *testing.T) {
		lens := rp.GetLensInfo()
		t.Logf("Lens: %s %s", lens.LensMake, lens.Lens)
		t.Logf("  Focal: %.1f-%.1fmm, EXIFMaxAp=f/%.1f, InternalSerial=%q",
			lens.MinFocal, lens.MaxFocal, lens.EXIFMaxAp, lens.InternalLensSerial)

		ml := rp.GetMakernotesLens()
		t.Logf("  Makernotes: mount=%d focalType=%d cur=%.1fmm f/%.1f",
			ml.LensMount, ml.FocalType, ml.CurFocal, ml.CurAp)
	})

	t.Run("ShootingParams", func(t *testing.T) {
		sp := rp.GetShootingParams()
		t.Logf("ISO=%.0f Shutter=%.6f Aperture=f/%.1f Focal=%.1fmm",
			sp.ISOSpeed, sp.Shutter, sp.Aperture, sp.FocalLen)
		t.Logf("Timestamp=%v Artist=%q", sp.Timestamp, sp.Artist)
	})

	t.Run("ImageSizes", func(t *testing.T) {
		s := rp.GetImageSizes()
		t.Logf("Raw: %dx%d  Image: %dx%d  Output: %dx%d  Flip=%d PixelAspect=%.6f",
			s.RawWidth, s.RawHeight, s.Width, s.Height, s.IWidth, s.IHeight, s.Flip, s.PixelAspectRatio)
		for i, c := range s.RawInsetCrops {
			if c.Width > 0 {
				t.Logf("RawInsetCrop[%d]: %dx%d at (%d,%d)", i, c.Width, c.Height, c.Left, c.Top)
			}
		}
	})

	t.Run("ColorData", func(t *testing.T) {
		cd := rp.GetColorData()
		t.Logf("Black=%d CBlack=%v", cd.Black, cd.CBlack)
		t.Logf("CamMul=%v  PreMul=%v", cd.CamMul, cd.PreMul)
		t.Logf("UniqueCameraModel=%q LocalizedCameraModel=%q",
			cd.UniqueCameraModel, cd.LocalizedCameraModel)
		t.Logf("HasICCProfile=%v AsShotWBApplied=%v ExifColorSpace=%d",
			cd.HasICCProfile, cd.AsShotWBApplied, cd.ExifColorSpace)
		if cd.CamMul[1] > 0 {
			t.Logf("CamMul EVs: R=%.2f B=%.2f (relative to G1)",
				cd.CamMul[0]/cd.CamMul[1], cd.CamMul[2]/cd.CamMul[1])
		}
	})

	t.Run("WhiteBalance", func(t *testing.T) {
		coeffs := rp.GetWBCoeffs()
		t.Logf("WB presets: %d", len(coeffs))
		for idx, c := range coeffs {
			t.Logf("  %d: R=%d G1=%d B=%d G2=%d", idx, c[0], c[1], c[2], c[3])
		}

		tc := rp.GetWBTempCoeffs()
		t.Logf("WB temp entries: %d", len(tc))
		for i, c := range tc {
			t.Logf("  #%d: %dK %v", i, c.CCT, c.Coeffs)
		}

		dl := rp.GetDNGLevels()
		t.Logf("DNG AsShotNeutral=%v BaselineExposure=%.3f", dl.AsShotNeutral, dl.BaselineExposure)
	})

	t.Run("DNGColor", func(t *testing.T) {
		for i := 0; i < 2; i++ {
			dc := rp.GetDNGColor(i)
			if dc.Illuminant != 0 {
				t.Logf("DNGColor[%d]: Illuminant=%d  ColorMatrix=%v", i, dc.Illuminant, dc.ColorMatrix)
			}
		}
	})

	t.Run("Thumbnail", func(t *testing.T) {
		ti := rp.GetThumbnailInfo()
		t.Logf("Thumbnail: %dx%d format=%d %d bytes", ti.Width, ti.Height, ti.Format, ti.Length)
	})

	t.Run("ICCProfile", func(t *testing.T) {
		if p := rp.GetICCProfile(); p == nil {
			t.Log("No embedded ICC profile")
		} else {
			t.Logf("ICC profile: %d bytes", len(p))
		}
	})

	t.Run("Temperatures", func(t *testing.T) {
		tmp := rp.GetTemperatures()
		t.Logf("Camera=%.2f Sensor=%.2f Lens=%.2f Ambient=%.2f RealISO=%.1f",
			tmp.CameraTemperature, tmp.SensorTemperature,
			tmp.LensTemperature, tmp.AmbientTemperature, tmp.RealISO)
		t.Logf("FlashEC=%.2f FlashGN=%.2f Firmware=%q", tmp.FlashEC, tmp.FlashGN, tmp.Firmware)
	})
}

// ── Processing pipeline ──────────────────────────────────────────

func TestProcess(t *testing.T) {
	rp := openAndProcess(t)

	img, err := rp.MakeMemImage()
	if err != nil {
		t.Fatalf("MakeMemImage() error: %v", err)
	}
	if img.Width == 0 || img.Height == 0 || len(img.Data) == 0 {
		t.Fatalf("invalid mem image: %dx%d, %d bytes", img.Width, img.Height, len(img.Data))
	}
	t.Logf("Mem image: %dx%d, %d colors, %d bits, %d bytes, format=%d",
		img.Width, img.Height, img.Colors, img.Bits, len(img.Data), img.Type)

	// write to temp file to verify output is valid
	tmpDir := t.TempDir()
	ext := ".ppm"
	if img.Type == ImageJPEG {
		ext = ".jpg"
	}
	outPath := filepath.Join(tmpDir, "output"+ext)
	if err := os.WriteFile(outPath, img.Data, 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}
	if stat, err := os.Stat(outPath); err != nil || stat.Size() == 0 {
		t.Fatalf("output file missing or empty")
	}
}

func TestProcessThumb(t *testing.T) {
	rp := openTestRAW(t)
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

func TestWritePPMTiff(t *testing.T) {
	rp := openAndProcess(t, WithTIFFOutput())

	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "output.tiff")
	if err := rp.WritePPMTiff(outPath); err != nil {
		t.Fatalf("WritePPMTiff() error: %v", err)
	}
	if stat, err := os.Stat(outPath); err != nil || stat.Size() == 0 {
		t.Fatalf("output TIFF missing or empty")
	}
}

// ── Options ──────────────────────────────────────────────────────

func TestOptions16Bit(t *testing.T) {
	rp := openAndProcess(t,
		With16BitOutput(),
		WithTIFFOutput(),
		WithCameraWB(),
		WithOutputColorSpace(ColorSpaceRaw),
		WithNoAutoBrightness(),
		WithInterpolationQuality(QualityAHD),
	)

	img, err := rp.MakeMemImage()
	if err != nil {
		t.Fatalf("MakeMemImage() error: %v", err)
	}
	if img.Bits != 16 {
		t.Errorf("Bits = %d, want 16", img.Bits)
	}
	t.Logf("16-bit TIFF: %dx%d, %d bits", img.Width, img.Height, img.Bits)
}

func TestWhiteBalanceModes(t *testing.T) {
	modes := []struct {
		name string
		opts []Option
	}{
		{"CameraWB", []Option{WithCameraWB()}},
		{"AutoWB", []Option{WithAutoWB()}},
		{"UserMul", []Option{WithUserMul(1.0, 1.0, 1.0, 1.0)}},
	}
	for _, m := range modes {
		t.Run(m.name, func(t *testing.T) {
			openAndProcess(t, m.opts...)
		})
	}
}

func TestRawParamsOptions(t *testing.T) {
	rp := openAndProcess(t,
		WithDNGSDK(DNGSDKNone),
		WithUseRawSpeed(0),
		WithRawOptions(0),
	)
	img, err := rp.MakeMemImage()
	if err != nil {
		t.Fatalf("MakeMemImage() error: %v", err)
	}
	if img.Width == 0 || img.Height == 0 {
		t.Fatalf("image dimensions zero: %dx%d", img.Width, img.Height)
	}
	t.Logf("RawParams: %dx%d, %d bytes", img.Width, img.Height, len(img.Data))
}

// ── DNG SDK ──────────────────────────────────────────────────────

func TestDNGSDK(t *testing.T) {
	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp.Close()

	if err := rp.EnableDNGSDK(); err != nil {
		t.Fatalf("EnableDNGSDK() error: %v", err)
	}
	if rp.dngHost == nil {
		t.Fatal("dngHost is nil, USE_DNGSDK may not be enabled")
	}
}

func TestDNGSDKProcess(t *testing.T) {
	rp, err := New(
		WithDNGSDK(DNGSDKDefault|DNGSDKXTrans),
		WithCameraWB(),
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp.Close()

	if err := rp.EnableDNGSDK(); err != nil {
		t.Fatalf("EnableDNGSDK() error: %v", err)
	}
	if err := rp.OpenFile(testRAWPath(t)); err != nil {
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

func TestDNGSDKAfterClose(t *testing.T) {
	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	rp.Close()

	if err := rp.EnableDNGSDK(); err != ErrAlreadyClosed {
		t.Fatalf("EnableDNGSDK after Close() = %v, want ErrAlreadyClosed", err)
	}
}
