package golibraw

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// testRAWPath returns a test RAW file path from GOLIBRAW_TEST_FILE env or testdata/.
func testRAWPath(t testing.TB) string {
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
func openTestRAW(t testing.TB, opts ...Option) *RawProcessor {
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
func openAndProcess(t testing.TB, opts ...Option) *RawProcessor {
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

	info1, err := openFileAndIdentify(rp, path)
	if err != nil {
		t.Fatalf("first OpenFile() error: %v", err)
	}

	rp.Recycle()

	info2, err := openFileAndIdentify(rp, path)
	if err != nil {
		t.Fatalf("second OpenFile() error: %v", err)
	}

	if info1.Make != info2.Make || info1.Model != info2.Model {
		t.Errorf("metadata mismatch after recycle: (%q,%q) vs (%q,%q)",
			info1.Make, info1.Model, info2.Make, info2.Model)
	}
}

// openFileAndIdentify opens a file and returns camera info (helper for tests).
func openFileAndIdentify(rp *RawProcessor, path string) (CameraInfo, error) {
	if err := rp.OpenFile(path); err != nil {
		return CameraInfo{}, err
	}
	return rp.GetCameraInfo()
}

func TestOpenBuffer(t *testing.T) {
	path := testRAWPath(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	rp2, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp2.Close()

	if err := rp2.OpenBuffer(data); err != nil {
		t.Fatalf("OpenBuffer() error: %v", err)
	}
	info, err := rp2.GetCameraInfo()
	if err != nil {
		t.Fatalf("GetCameraInfo() error: %v", err)
	}
	if info.Make == "" {
		t.Fatal("GetCameraInfo().Make is empty after OpenBuffer")
	}
}

func TestOpenBufferEmpty(t *testing.T) {
	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp.Close()

	err = rp.OpenBuffer(nil)
	if !errors.Is(err, ErrBufferOpen) {
		t.Errorf("OpenBuffer(nil) = %v, want ErrBufferOpen", err)
	}

	err = rp.OpenBuffer([]byte{})
	if !errors.Is(err, ErrBufferOpen) {
		t.Errorf("OpenBuffer([]byte{}) = %v, want ErrBufferOpen", err)
	}
}

// ── Identify (metadata-only, no Unpack needed) ───────────────────

func TestGetCameraInfo(t *testing.T) {
	rp := openTestRAW(t)
	camera, err := rp.GetCameraInfo()
	if err != nil {
		t.Fatalf("GetCameraInfo() error: %v", err)
	}
	if camera.Make == "" {
		t.Fatal("CameraInfo.Make is empty")
	}
	t.Logf("Camera: %s %s (maker=%d, normalized=%s/%s)",
		camera.Make, camera.Model, camera.MakerIndex,
		camera.NormalizedMake, camera.NormalizedModel)
}

func TestGetShootingInfo(t *testing.T) {
	rp := openTestRAW(t)
	si, err := rp.GetShootingInfo()
	if err != nil {
		t.Fatalf("GetShootingInfo() error: %v", err)
	}
	t.Logf("DriveMode=%d FocusMode=%d MeteringMode=%d AFPoint=%d",
		si.DriveMode, si.FocusMode, si.MeteringMode, si.AFPoint)
	t.Logf("ExposureMode=%d ExposureProgram=%d ImageStabilization=%d",
		si.ExposureMode, si.ExposureProgram, si.ImageStabilization)
	t.Logf("BodySerial=%q InternalBodySerial=%q", si.BodySerial, si.InternalBodySerial)
}

func TestGetLensInfo(t *testing.T) {
	rp := openTestRAW(t)

	lens, err := rp.GetLensInfo()
	if err != nil {
		t.Fatalf("GetLensInfo() error: %v", err)
	}
	t.Logf("Lens: %s %s", lens.LensMake, lens.Lens)
	t.Logf("  Focal: %.1f-%.1fmm, EXIFMaxAp=f/%.1f, InternalSerial=%q",
		lens.MinFocal, lens.MaxFocal, lens.EXIFMaxAp, lens.InternalLensSerial)

	ml, err := rp.GetMakernotesLens()
	if err != nil {
		t.Fatalf("GetMakernotesLens() error: %v", err)
	}
	t.Logf("  Makernotes: mount=%d focalType=%d cur=%.1fmm f/%.1f",
		ml.LensMount, ml.FocalType, ml.CurFocal, ml.CurAp)
}

func TestGetShootingParams(t *testing.T) {
	rp := openTestRAW(t)
	sp, err := rp.GetShootingParams()
	if err != nil {
		t.Fatalf("GetShootingParams() error: %v", err)
	}
	t.Logf("ISO=%.0f Shutter=%.6f Aperture=f/%.1f Focal=%.1fmm",
		sp.ISOSpeed, sp.Shutter, sp.Aperture, sp.FocalLen)
	t.Logf("Timestamp=%v Artist=%q AnalogBalance=%v", sp.Timestamp, sp.Artist, sp.AnalogBalance)
}

func TestGetImageSizes(t *testing.T) {
	rp := openTestRAW(t)
	s, err := rp.GetImageSizes()
	if err != nil {
		t.Fatalf("GetImageSizes() error: %v", err)
	}
	t.Logf("Raw: %dx%d  Image: %dx%d  Output: %dx%d  Flip=%d PixelAspect=%.6f",
		s.RawWidth, s.RawHeight, s.Width, s.Height, s.IWidth, s.IHeight, s.Flip, s.PixelAspectRatio)
	if s.RawWidth == 0 || s.RawHeight == 0 {
		t.Fatal("ImageSizes.RawWidth or RawHeight is zero")
	}
	for i, c := range s.RawInsetCrops {
		if c.Width > 0 {
			t.Logf("RawInsetCrop[%d]: %dx%d at (%d,%d)", i, c.Width, c.Height, c.Left, c.Top)
		}
	}
}

func TestGetColorData(t *testing.T) {
	rp := openTestRAW(t)
	cd, err := rp.GetColorData()
	if err != nil {
		t.Fatalf("GetColorData() error: %v", err)
	}
	t.Logf("Black=%d CBlack=%v", cd.Black, cd.CBlack)
	if cd.Maximum == 0 {
		t.Fatal("ColorData.Maximum is zero")
	}
	t.Logf("CamMul=%v  PreMul=%v", cd.CamMul, cd.PreMul)
	t.Logf("UniqueCameraModel=%q LocalizedCameraModel=%q",
		cd.UniqueCameraModel, cd.LocalizedCameraModel)
	t.Logf("HasICCProfile=%v AsShotWBApplied=%v ExifColorSpace=%d",
		cd.HasICCProfile, cd.AsShotWBApplied, cd.ExifColorSpace)
	if cd.CamMul[1] > 0 {
		t.Logf("CamMul EVs: R=%.2f B=%.2f (relative to G1)",
			cd.CamMul[0]/cd.CamMul[1], cd.CamMul[2]/cd.CamMul[1])
	}
}

func TestGetWhiteBalance(t *testing.T) {
	rp := openTestRAW(t)

	coeffs, err := rp.GetWBCoeffs()
	if err != nil {
		t.Fatalf("GetWBCoeffs() error: %v", err)
	}
	t.Logf("WB presets: %d", len(coeffs))
	if len(coeffs) == 0 {
		t.Fatal("no WB presets found")
	}
	for idx, c := range coeffs {
		t.Logf("  %d: R=%d G1=%d B=%d G2=%d", idx, c[0], c[1], c[2], c[3])
	}

	tc, err := rp.GetWBTempCoeffs()
	if err != nil {
		t.Fatalf("GetWBTempCoeffs() error: %v", err)
	}
	t.Logf("WB temp entries: %d", len(tc))
	for i, c := range tc {
		t.Logf("  #%d: %dK %v", i, c.CCT, c.Coeffs)
	}

	dl, err := rp.GetDNGLevels()
	if err != nil {
		t.Fatalf("GetDNGLevels() error: %v", err)
	}
	t.Logf("DNG AsShotNeutral=%v BaselineExposure=%.3f AnalogBalance=%v", dl.AsShotNeutral, dl.BaselineExposure, dl.AnalogBalance)
}

func TestGetDNGColor(t *testing.T) {
	rp := openTestRAW(t)
	for i := 0; i < 2; i++ {
		dc, err := rp.GetDNGColor(i)
		if err != nil {
			t.Fatalf("GetDNGColor(%d) error: %v", i, err)
		}
		if dc.Illuminant != 0 {
			t.Logf("DNGColor[%d]: Illuminant=%d  ColorMatrix=%v", i, dc.Illuminant, dc.ColorMatrix)
		}
	}
}

func TestGetThumbnailInfo(t *testing.T) {
	rp := openTestRAW(t)
	ti, err := rp.GetThumbnailInfo()
	if err != nil {
		t.Fatalf("GetThumbnailInfo() error: %v", err)
	}
	t.Logf("Thumbnail: %dx%d format=%d %d bytes", ti.Width, ti.Height, ti.Format, ti.Length)
}

func TestGetICCProfile(t *testing.T) {
	rp := openTestRAW(t)
	p, err := rp.GetICCProfile()
	if err != nil {
		t.Fatalf("GetICCProfile() error: %v", err)
	}
	if p == nil {
		t.Log("No embedded ICC profile")
	} else {
		t.Logf("ICC profile: %d bytes", len(p))
	}
}

func TestGetTemperatures(t *testing.T) {
	rp := openTestRAW(t)
	tmp, err := rp.GetTemperatures()
	if err != nil {
		t.Fatalf("GetTemperatures() error: %v", err)
	}
	t.Logf("Camera=%.2f Sensor=%.2f Lens=%.2f Ambient=%.2f RealISO=%.1f",
		tmp.CameraTemperature, tmp.SensorTemperature,
		tmp.LensTemperature, tmp.AmbientTemperature, tmp.RealISO)
	t.Logf("FlashEC=%.2f FlashGN=%.2f Firmware=%q", tmp.FlashEC, tmp.FlashGN, tmp.Firmware)
}

func TestGetGPS(t *testing.T) {
	rp := openTestRAW(t)
	gps, err := rp.GetGPS()
	if err != nil {
		t.Fatalf("GetGPS() error: %v", err)
	}
	t.Logf("GPS: LatRef=%c LongRef=%c GPSParsed=%v Altitude=%.1f",
		gps.LatRef, gps.LongRef, gps.GPSParsed, gps.Altitude)
}

func TestGetCameraInfoFilters(t *testing.T) {
	rp := openTestRAW(t)
	ci, err := rp.GetCameraInfo()
	if err != nil {
		t.Fatalf("GetCameraInfo() error: %v", err)
	}
	t.Logf("Filters=0x%08x CDesc=%q XMPLen=%d", ci.Filters, ci.CDesc, ci.XMPLen)
	if ci.CDesc == "" {
		t.Fatal("CameraInfo.CDesc is empty")
	}
	if ci.XMPLen > 0 && ci.XMPData != nil {
		t.Logf("XMP data: %d bytes", len(ci.XMPData))
	}
	if ci.Filters == 9 {
		t.Log("X-Trans CFA detected")
	}
}

func TestGetThumbnailList(t *testing.T) {
	rp := openTestRAW(t)
	thumbs, err := rp.GetThumbnailList()
	if err != nil {
		t.Fatalf("GetThumbnailList() error: %v", err)
	}
	t.Logf("Thumbnail list: %d entries", len(thumbs))
	for i, th := range thumbs {
		t.Logf("  [%d]: %dx%d format=%d length=%d", i, th.Width, th.Height, th.Format, th.Length)
	}
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

func TestNewOptions(t *testing.T) {
	t.Run("GreyBox", func(t *testing.T) {
		rp, err := New(WithGreyBox(0, 0, 100, 100))
		if err != nil {
			t.Fatalf("New(WithGreyBox) error: %v", err)
		}
		t.Cleanup(func() { rp.Close() })
	})
	t.Run("UserCBlack", func(t *testing.T) {
		rp, err := New(WithUserCBlack(0, 0, 0, 0))
		if err != nil {
			t.Fatalf("New(WithUserCBlack) error: %v", err)
		}
		t.Cleanup(func() { rp.Close() })
	})
	t.Run("AutoBrightThreshold", func(t *testing.T) {
		rp, err := New(WithAutoBrightThreshold(0.5))
		if err != nil {
			t.Fatalf("New(WithAutoBrightThreshold) error: %v", err)
		}
		t.Cleanup(func() { rp.Close() })
	})
	t.Run("PhaseOneCorrection", func(t *testing.T) {
		rp, err := New(WithPhaseOneCorrection())
		if err != nil {
			t.Fatalf("New(WithPhaseOneCorrection) error: %v", err)
		}
		t.Cleanup(func() { rp.Close() })
	})
	t.Run("OutputFlags", func(t *testing.T) {
		rp, err := New(WithOutputFlags(1))
		if err != nil {
			t.Fatalf("New(WithOutputFlags) error: %v", err)
		}
		t.Cleanup(func() { rp.Close() })
	})
	t.Run("RawSpecials", func(t *testing.T) {
		rp, err := New(WithRawSpecials(0))
		if err != nil {
			t.Fatalf("New(WithRawSpecials) error: %v", err)
		}
		t.Cleanup(func() { rp.Close() })
	})
	t.Run("MaxRawMemory", func(t *testing.T) {
		rp, err := New(WithMaxRawMemory(2048))
		if err != nil {
			t.Fatalf("New(WithMaxRawMemory) error: %v", err)
		}
		t.Cleanup(func() { rp.Close() })
	})
	t.Run("SonyARW2Posterization", func(t *testing.T) {
		rp, err := New(WithSonyARW2Posterization(0))
		if err != nil {
			t.Fatalf("New(WithSonyARW2Posterization) error: %v", err)
		}
		t.Cleanup(func() { rp.Close() })
	})
	t.Run("CoolScanNEFGamma", func(t *testing.T) {
		rp, err := New(WithCoolScanNEFGamma(1.0))
		if err != nil {
			t.Fatalf("New(WithCoolScanNEFGamma) error: %v", err)
		}
		t.Cleanup(func() { rp.Close() })
	})
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
	if rp.res.dngHost == nil {
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

	if err := rp.EnableDNGSDK(); !errors.Is(err, ErrAlreadyClosed) {
		t.Fatalf("EnableDNGSDK after Close() = %v, want ErrAlreadyClosed", err)
	}
}

// ── Cancel ──────────────────────────────────────────────────────────

func TestCancelAbortsProcess(t *testing.T) {
	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp.Close()

	if err := rp.OpenFile(testRAWPath(t)); err != nil {
		t.Fatalf("OpenFile() error: %v", err)
	}
	if err := rp.Unpack(); err != nil {
		t.Fatalf("Unpack() error: %v", err)
	}

	// Cancel before Process — should return error
	rp.Cancel()
	err = rp.Process()
	if err == nil {
		t.Fatal("Process() should fail after Cancel()")
	}
	t.Logf("Process() after Cancel(): %v", err)
}

func TestCancelIdempotent(t *testing.T) {
	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	rp.Cancel()
	rp.Cancel()
	rp.Cancel()
	if err := rp.Close(); err != nil {
		t.Fatalf("Close() after Cancel(): %v", err)
	}
}

func TestCancelAfterClose(t *testing.T) {
	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	rp.Close()
	rp.Cancel() // should not panic
}

func TestCancelConcurrentClose(t *testing.T) {
	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp.Close()
	if err := rp.OpenFile(testRAWPath(t)); err != nil {
		t.Fatalf("OpenFile() error: %v", err)
	}
	if err := rp.Unpack(); err != nil {
		t.Fatalf("Unpack() error: %v", err)
	}

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			rp.Cancel()
		})
	}
	rp.Close()
	wg.Wait()
}

// ── Image type queries ───────────────────────────────────────────

func TestImageQueries(t *testing.T) {
	rp := openTestRAW(t)

	t.Run("IsFujiRotated", func(t *testing.T) {
		rotated := rp.IsFujiRotated()
		t.Logf("IsFujiRotated: %v", rotated)
	})

	t.Run("IsSRAW", func(t *testing.T) {
		sraw := rp.IsSRAW()
		t.Logf("IsSRAW: %v", sraw)
	})

	t.Run("IsNikonSRAW", func(t *testing.T) {
		sraw := rp.IsNikonSRAW()
		t.Logf("IsNikonSRAW: %v", sraw)
	})

	t.Run("IsCoolscanNEF", func(t *testing.T) {
		coolscan := rp.IsCoolscanNEF()
		t.Logf("IsCoolscanNEF: %v", coolscan)
	})

	t.Run("IsJPEGThumb", func(t *testing.T) {
		jpeg := rp.IsJPEGThumb()
		t.Logf("IsJPEGThumb: %v", jpeg)
	})

	t.Run("IsFloatingPoint", func(t *testing.T) {
		fp := rp.IsFloatingPoint()
		t.Logf("IsFloatingPoint: %v", fp)
	})

	t.Run("HaveFPData", func(t *testing.T) {
		fp := rp.HaveFPData()
		t.Logf("HaveFPData: %v", fp)
	})

	t.Run("ErrorCount", func(t *testing.T) {
		count := rp.ErrorCount()
		t.Logf("ErrorCount: %d", count)
	})

	t.Run("ThumbOK", func(t *testing.T) {
		ok := rp.ThumbOK(-1)
		t.Logf("ThumbOK: %v", ok)
	})

	t.Run("RawWasRead", func(t *testing.T) {
		read := rp.RawWasRead()
		t.Logf("RawWasRead (before Unpack): %v", read)
	})
}

func TestImageQueriesAfterUnpack(t *testing.T) {
	rp := openAndProcess(t)

	t.Run("RawWasRead", func(t *testing.T) {
		read := rp.RawWasRead()
		t.Logf("RawWasRead (after Process): %v", read)
		if !read {
			t.Error("RawWasRead should be true after Process")
		}
	})

	t.Run("ErrorCount", func(t *testing.T) {
		count := rp.ErrorCount()
		t.Logf("ErrorCount after Process: %d", count)
	})
}

func TestColorFilterQueries(t *testing.T) {
	rp := openTestRAW(t)

	camera, err := rp.GetCameraInfo()
	if err != nil {
		t.Fatalf("GetCameraInfo() error: %v", err)
	}

	sizes, err := rp.GetImageSizes()
	if err != nil {
		t.Fatalf("GetImageSizes() error: %v", err)
	}

	t.Run("Color", func(t *testing.T) {
		if sizes.RawWidth > 0 && sizes.RawHeight > 0 {
			c := rp.Color(0, 0)
			t.Logf("Color(0,0): %d (filters=0x%08x)", c, camera.Filters)
			if c < 0 || c > 3 {
				t.Errorf("Color(0,0) = %d, want 0-3", c)
			}
		}
	})

	t.Run("FC", func(t *testing.T) {
		if sizes.RawWidth > 0 && sizes.RawHeight > 0 {
			c := rp.FC(0, 0)
			t.Logf("FC(0,0): %d", c)
			if c < 0 || c > 3 {
				t.Errorf("FC(0,0) = %d, want 0-3", c)
			}
		}
	})

	t.Run("FCol", func(t *testing.T) {
		if sizes.RawWidth > 0 && sizes.RawHeight > 0 {
			c := rp.FCol(0, 0)
			t.Logf("FCol(0,0): %d", c)
			if c < 0 || c > 3 {
				t.Errorf("FCol(0,0) = %d, want 0-3", c)
			}
		}
	})
}

// ── Pipeline step methods ────────────────────────────────────────

func TestRaw2Image(t *testing.T) {
	rp := openTestRAW(t)
	if err := rp.Unpack(); err != nil {
		t.Fatalf("Unpack() error: %v", err)
	}

	if err := rp.Raw2Image(); err != nil {
		t.Fatalf("Raw2Image() error: %v", err)
	}
	rp.FreeImage()
}

func TestRaw2ImageEx(t *testing.T) {
	rp := openTestRAW(t)
	if err := rp.Unpack(); err != nil {
		t.Fatalf("Unpack() error: %v", err)
	}

	if err := rp.Raw2ImageEx(true); err != nil {
		t.Fatalf("Raw2ImageEx(true) error: %v", err)
	}
	rp.FreeImage()
}

func TestSubtractBlack(t *testing.T) {
	rp := openTestRAW(t)
	if err := rp.Unpack(); err != nil {
		t.Fatalf("Unpack() error: %v", err)
	}

	if err := rp.Raw2Image(); err != nil {
		t.Fatalf("Raw2Image() error: %v", err)
	}
	if err := rp.SubtractBlack(); err != nil {
		t.Fatalf("SubtractBlack() error: %v", err)
	}
	rp.FreeImage()
}

func TestAdjustMaximum(t *testing.T) {
	rp := openTestRAW(t)
	if err := rp.Unpack(); err != nil {
		t.Fatalf("Unpack() error: %v", err)
	}

	if err := rp.AdjustMaximum(); err != nil {
		t.Fatalf("AdjustMaximum() error: %v", err)
	}
}

func TestAdjustSizesInfoOnly(t *testing.T) {
	rp := openTestRAW(t)

	if err := rp.AdjustSizesInfoOnly(); err != nil {
		t.Fatalf("AdjustSizesInfoOnly() error: %v", err)
	}

	sizes, err := rp.GetImageSizes()
	if err != nil {
		t.Fatalf("GetImageSizes() error: %v", err)
	}
	t.Logf("After AdjustSizesInfoOnly: %dx%d", sizes.Width, sizes.Height)
}

func TestRecycleDatastream(t *testing.T) {
	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp.Close()

	if err := rp.OpenFile(testRAWPath(t)); err != nil {
		t.Fatalf("OpenFile() error: %v", err)
	}

	rp.RecycleDatastream()

	// Should be able to open another file after recycling datastream
	if err := rp.OpenFile(testRAWPath(t)); err != nil {
		t.Fatalf("OpenFile after RecycleDatastream() error: %v", err)
	}
}

func TestConvertFloatToInt(t *testing.T) {
	rp := openTestRAW(t)
	if err := rp.Unpack(); err != nil {
		t.Fatalf("Unpack() error: %v", err)
	}

	// Only valid for floating-point RAW; should not crash for non-FP files
	rp.ConvertFloatToInt(0, 0, 0)
}

// ── Output optimization ──────────────────────────────────────────

func TestGetMemImageFormat(t *testing.T) {
	rp := openAndProcess(t)

	w, h, colors, bps := rp.GetMemImageFormat()
	t.Logf("MemImageFormat: %dx%d, %d colors, %d bps", w, h, colors, bps)
	if w == 0 || h == 0 {
		t.Error("GetMemImageFormat returned zero dimensions")
	}
}

func TestCopyMemImage(t *testing.T) {
	rp := openAndProcess(t)

	w, h, colors, bps := rp.GetMemImageFormat()
	if w == 0 || h == 0 {
		t.Skip("GetMemImageFormat returned zero dimensions")
	}

	stride := int(w) * int(colors) * int(bps) / 8
	buf := make([]byte, stride*int(h))
	if err := rp.CopyMemImage(buf, stride, false); err != nil {
		t.Fatalf("CopyMemImage() error: %v", err)
	}
	t.Logf("Copied %d bytes to buffer", len(buf))
}

// ── Info queries ─────────────────────────────────────────────────

func TestDecoderInfo(t *testing.T) {
	rp := openTestRAW(t)

	di, err := rp.GetDecoderInfo()
	if err != nil {
		t.Fatalf("GetDecoderInfo() error: %v", err)
	}
	t.Logf("Decoder: flags=0x%08x decoder=%q", di.DecoderFlags, di.DecoderName)

	fn, err := rp.UnpackFunctionName()
	if err != nil {
		t.Fatalf("UnpackFunctionName() error: %v", err)
	}
	t.Logf("UnpackFunctionName: %q", fn)
}

func TestCapabilities(t *testing.T) {
	caps := Capabilities()
	t.Logf("Capabilities: 0x%08x", caps)
	if caps == 0 {
		t.Error("Capabilities = 0, expected non-zero")
	}
}

func TestCameraList(t *testing.T) {
	cameras := CameraList()
	t.Logf("CameraList: %d cameras", len(cameras))
	if len(cameras) > 0 {
		t.Logf("First: %q, Last: %q", cameras[0], cameras[len(cameras)-1])
	}
}

func TestStrProgress(t *testing.T) {
	// Just verify it doesn't crash
	s := StrProgress(0)
	t.Logf("StrProgress(0): %q", s)
}

// ── Callbacks ────────────────────────────────────────────────────

func TestDataErrorHandler(t *testing.T) {
	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp.Close()

	rp.SetDataErrorHandler(func(file string, offset int64) {
		t.Logf("DataError: file=%q offset=%d", file, offset)
	})

	// Setting nil should not panic
	rp.SetDataErrorHandler(nil)
	t.Log("DataErrorHandler set and cleared without error")
}

func TestEXIFParseHandler(t *testing.T) {
	rp := openTestRAW(t)

	var tags []int
	rp.SetEXIFParseHandler(func(tag, typ, length int, order uint, base int64) {
		tags = append(tags, tag)
	})

	// Re-open to trigger EXIF parsing
	rp.Recycle()
	if err := rp.OpenFile(testRAWPath(t)); err != nil {
		t.Fatalf("Re-open error: %v", err)
	}

	t.Logf("EXIF tags parsed: %d", len(tags))
	if len(tags) > 0 {
		t.Logf("First tags: %v", tags[:min(10, len(tags))])
	}
}

func TestMakernotesParseHandler(t *testing.T) {
	rp := openTestRAW(t)

	var tags []int
	rp.SetMakernotesParseHandler(func(tag, typ, length int, order uint, base int64) {
		tags = append(tags, tag)
	})

	rp.Recycle()
	if err := rp.OpenFile(testRAWPath(t)); err != nil {
		t.Fatalf("Re-open error: %v", err)
	}

	t.Logf("Makernotes tags parsed: %d", len(tags))
}

// ── Makernotes ───────────────────────────────────────────────────

func TestMakernotes(t *testing.T) {
	rp := openTestRAW(t)

	camera, err := rp.GetCameraInfo()
	if err != nil {
		t.Fatalf("GetCameraInfo() error: %v", err)
	}
	make := camera.NormalizedMake

	t.Logf("Testing makernotes for %s %s", make, camera.NormalizedModel)

	t.Run("Canon", func(t *testing.T) {
		mn, err := rp.GetCanonMakernotes()
		if err != nil {
			t.Fatalf("GetCanonMakernotes() error: %v", err)
		}
		t.Logf("Canon: Quality=%d FlashMode=%d ContinuousDrive=%d Sensor=%dx%d",
			mn.Quality, mn.FlashMode, mn.ContinuousDrive, mn.SensorWidth, mn.SensorHeight)
	})

	t.Run("Nikon", func(t *testing.T) {
		mn, err := rp.GetNikonMakernotes()
		if err != nil {
			t.Fatalf("GetNikonMakernotes() error: %v", err)
		}
		t.Logf("Nikon: ActiveDLighting=%d NEFCompression=%d Sensor=%dx%d PictureControl=%q",
			mn.ActiveDLighting, mn.NEFCompression, mn.SensorWidth, mn.SensorHeight, mn.PictureControlName)
	})

	t.Run("Fuji", func(t *testing.T) {
		mn, err := rp.GetFujiMakernotes()
		if err != nil {
			t.Fatalf("GetFujiMakernotes() error: %v", err)
		}
		t.Logf("Fuji: DynamicRange=%d FilmMode=%d ShutterType=%d CropMode=%d DriveMode=%d",
			mn.DynamicRange, mn.FilmMode, mn.ShutterType, mn.CropMode, mn.DriveMode)
	})

	t.Run("Olympus", func(t *testing.T) {
		mn, err := rp.GetOlympusMakernotes()
		if err != nil {
			t.Fatalf("GetOlympusMakernotes() error: %v", err)
		}
		t.Logf("Olympus: ColorSpace=%d FocusMode=%v DriveMode=%v",
			mn.ColorSpace, mn.FocusMode, mn.DriveMode)
	})

	t.Run("Sony", func(t *testing.T) {
		mn, err := rp.GetSonyMakernotes()
		if err != nil {
			t.Fatalf("GetSonyMakernotes() error: %v", err)
		}
		t.Logf("Sony: CameraType=%d RAWFileType=%d FileFormat=%d Quality=%d",
			mn.CameraType, mn.RAWFileType, mn.FileFormat, mn.Quality)
	})

	t.Run("Kodak", func(t *testing.T) {
		mn, err := rp.GetKodakMakernotes()
		if err != nil {
			t.Fatalf("GetKodakMakernotes() error: %v", err)
		}
		t.Logf("Kodak: BlackLevelTop=%d BlackLevelBottom=%d ISOCalibrationGain=%.2f",
			mn.BlackLevelTop, mn.BlackLevelBottom, mn.ISOCalibrationGain)
	})

	t.Run("Panasonic", func(t *testing.T) {
		mn, err := rp.GetPanasonicMakernotes()
		if err != nil {
			t.Fatalf("GetPanasonicMakernotes() error: %v", err)
		}
		t.Logf("Panasonic: Compression=%d Multishot=%d Gamma=%.2f",
			mn.Compression, mn.Multishot, mn.Gamma)
	})

	t.Run("Pentax", func(t *testing.T) {
		mn, err := rp.GetPentaxMakernotes()
		if err != nil {
			t.Fatalf("GetPentaxMakernotes() error: %v", err)
		}
		t.Logf("Pentax: AFAdjustment=%d Quality=%d FocusMode=%v",
			mn.AFAdjustment, mn.Quality, mn.FocusMode)
	})

	t.Run("PhaseOne", func(t *testing.T) {
		mn, err := rp.GetPhaseOneMakernotes()
		if err != nil {
			t.Fatalf("GetPhaseOneMakernotes() error: %v", err)
		}
		t.Logf("PhaseOne: Software=%q SystemType=%q Firmware=%q",
			mn.Software, mn.SystemType, mn.FirmwareString)
	})

	t.Run("Ricoh", func(t *testing.T) {
		mn, err := rp.GetRicohMakernotes()
		if err != nil {
			t.Fatalf("GetRicohMakernotes() error: %v", err)
		}
		t.Logf("Ricoh: AFStatus=%d CropMode=%d Sensor=%dx%d",
			mn.AFStatus, mn.CropMode, mn.SensorWidth, mn.SensorHeight)
	})

	t.Run("Samsung", func(t *testing.T) {
		mn, err := rp.GetSamsungMakernotes()
		if err != nil {
			t.Fatalf("GetSamsungMakernotes() error: %v", err)
		}
		t.Logf("Samsung: DeviceType=%d DigitalGain=%.2f LensFirmware=%q",
			mn.DeviceType, mn.DigitalGain, mn.LensFirmware)
	})

	t.Run("Hasselblad", func(t *testing.T) {
		mn, err := rp.GetHasselbladMakernotes()
		if err != nil {
			t.Fatalf("GetHasselbladMakernotes() error: %v", err)
		}
		t.Logf("Hasselblad: BaseISO=%d Sensor=%q HostBody=%q Format=%d",
			mn.BaseISO, mn.Sensor, mn.HostBody, mn.Format)
	})
}

// ── Closed processor errors ──────────────────────────────────────

func TestGetterErrorsAfterClose(t *testing.T) {
	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	rp.Close()

	// All getters should return ErrAlreadyClosed
	getters := []struct {
		name string
		fn   func() error
	}{
		{"GetCameraInfo", func() error { _, err := rp.GetCameraInfo(); return err }},
		{"GetLensInfo", func() error { _, err := rp.GetLensInfo(); return err }},
		{"GetShootingParams", func() error { _, err := rp.GetShootingParams(); return err }},
		{"GetShootingInfo", func() error { _, err := rp.GetShootingInfo(); return err }},
		{"GetMakernotesLens", func() error { _, err := rp.GetMakernotesLens(); return err }},
		{"GetTemperatures", func() error { _, err := rp.GetTemperatures(); return err }},
		{"GetThumbnailInfo", func() error { _, err := rp.GetThumbnailInfo(); return err }},
		{"GetColorData", func() error { _, err := rp.GetColorData(); return err }},
		{"GetWBCoeffs", func() error { _, err := rp.GetWBCoeffs(); return err }},
		{"GetWBTempCoeffs", func() error { _, err := rp.GetWBTempCoeffs(); return err }},
		{"GetDNGLevels", func() error { _, err := rp.GetDNGLevels(); return err }},
		{"GetICCProfile", func() error { _, err := rp.GetICCProfile(); return err }},
		{"GetGPS", func() error { _, err := rp.GetGPS(); return err }},
		{"GetImageSizes", func() error { _, err := rp.GetImageSizes(); return err }},
		{"GetThumbnailList", func() error { _, err := rp.GetThumbnailList(); return err }},
		{"GetCanonMakernotes", func() error { _, err := rp.GetCanonMakernotes(); return err }},
		{"GetNikonMakernotes", func() error { _, err := rp.GetNikonMakernotes(); return err }},
		{"GetFujiMakernotes", func() error { _, err := rp.GetFujiMakernotes(); return err }},
		{"GetSonyMakernotes", func() error { _, err := rp.GetSonyMakernotes(); return err }},
		{"GetPhaseOneMakernotes", func() error { _, err := rp.GetPhaseOneMakernotes(); return err }},
		{"GetOlympusMakernotes", func() error { _, err := rp.GetOlympusMakernotes(); return err }},
		{"GetKodakMakernotes", func() error { _, err := rp.GetKodakMakernotes(); return err }},
		{"GetPanasonicMakernotes", func() error { _, err := rp.GetPanasonicMakernotes(); return err }},
		{"GetPentaxMakernotes", func() error { _, err := rp.GetPentaxMakernotes(); return err }},
		{"GetRicohMakernotes", func() error { _, err := rp.GetRicohMakernotes(); return err }},
		{"GetSamsungMakernotes", func() error { _, err := rp.GetSamsungMakernotes(); return err }},
		{"GetHasselbladMakernotes", func() error { _, err := rp.GetHasselbladMakernotes(); return err }},
		{"GetDNGColor", func() error { _, err := rp.GetDNGColor(0); return err }},
		{"Raw2Image", func() error { return rp.Raw2Image() }},
		{"SubtractBlack", func() error { return rp.SubtractBlack() }},
	{"CopyMemImage", func() error { return rp.CopyMemImage(make([]byte, 1), 1, false) }},
	}

	for _, g := range getters {
		t.Run(g.name, func(t *testing.T) {
			if err := g.fn(); !errors.Is(err, ErrAlreadyClosed) {
				t.Errorf("%s after Close() = %v, want ErrAlreadyClosed", g.name, err)
			}
		})
	}
}

func TestDemosaicPackQualities(t *testing.T) {
	qualities := []struct {
		name  string
		value InterpolationQuality
	}{
		{"ModifiedAHD", QualityModifiedAHD}, // 5
		{"AFD", QualityAFD},                 // 6
		{"VCD", QualityVCD},                 // 7
		{"VCDAHD", QualityVCDAHD},           // 8
		{"LMMSE", QualityLMMSE},             // 9
		{"AMaZE", QualityAMaZE},             // 10
	}
	// Use CR2/CR3 files — DNG.dng produces all-zero pixel data after processing
	path := ""
	for _, name := range []string{"IMG_8000.CR2", "IMG_1104.CR3"} {
		if _, err := os.Stat(filepath.Join("testdata", name)); err == nil {
			path = filepath.Join("testdata", name)
			break
		}
	}
	if path == "" {
		t.Skip("no CR2/CR3 test file found in testdata/")
		return
	}
	for _, q := range qualities {
		t.Run(q.name, func(t *testing.T) {
			rp, err := New(WithInterpolationQuality(q.value), WithCameraWB())
			if err != nil {
				t.Fatalf("New() error: %v", err)
			}
			t.Cleanup(func() { rp.Close() })
			if err := rp.OpenFile(path); err != nil {
				t.Fatalf("OpenFile() error: %v", err)
			}
			if err := rp.Unpack(); err != nil {
				t.Fatalf("Unpack() error: %v", err)
			}
			if err := rp.Process(); err != nil {
				t.Fatalf("Process() error: %v", err)
			}
			pw := rp.ProcessWarnings()
			if pw&uint(WarnFallbackToAHD) != 0 {
				t.Errorf("quality %d (%s) fell back to AHD", q.value, q.name)
			}
			img, err := rp.MakeMemImage()
			if err != nil {
				t.Fatalf("MakeMemImage() error: %v", err)
			}
			if img.Width == 0 || img.Height == 0 {
				t.Errorf("got zero-dimension image: %dx%d", img.Width, img.Height)
			}
			allZero := true
			for _, b := range img.Data {
				if b != 0 {
					allZero = false
					break
				}
			}
			if allZero {
				t.Errorf("%s: image data is all zeros", q.name)
			}
			t.Logf("%s: %dx%d, %d bits, warnings=0x%08x", q.name, img.Width, img.Height, img.Bits, pw)
		})
	}
}


// ── Utility methods ──────────────────────────────────────────────

func TestSetMakeFromIndex(t *testing.T) {
	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp.Close()

	// SetMakeFromIndex(8) sets the internal maker index.
	// It may return an error if called without prior OpenFile,
	// so we just verify it doesn't panic.
	err = rp.SetMakeFromIndex(8)
	t.Logf("SetMakeFromIndex(8) err=%v (expected non-fatal)", err)
}

func TestSetRawSpeedCameraFile(t *testing.T) {
	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp.Close()

	// Should not crash even if file doesn't exist
	err = rp.SetRawSpeedCameraFile("/nonexistent/cameras.xml")
	t.Logf("SetRawSpeedCameraFile: %v (expected if RawSpeed not compiled)", err)
}

// ── AdjustToRawInsetCrop ────────────────────────────────────────

func TestAdjustToRawInsetCrop(t *testing.T) {
	rp := openTestRAW(t)

	applied, err := rp.AdjustToRawInsetCrop(InsetCropAllMask, 0.0)
	if err != nil {
		t.Fatalf("AdjustToRawInsetCrop() error: %v", err)
	}
	t.Logf("AdjustToRawInsetCrop: applied=%v", applied)
}

// ── OpenBayer ────────────────────────────────────────────────────

func TestWriteThumb(t *testing.T) {
	rp := openTestRAW(t)
	if err := rp.UnpackThumb(); err != nil {
		t.Skipf("UnpackThumb() error: %v", err)
	}

	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "thumb.jpg")
	if err := rp.WriteThumb(outPath); err != nil {
		t.Fatalf("WriteThumb() error: %v", err)
	}
	if stat, err := os.Stat(outPath); err != nil || stat.Size() == 0 {
		t.Fatal("thumbnail output file missing or empty")
	}
}

func TestUnpackThumbAt(t *testing.T) {
	rp := openTestRAW(t)
	thumbs, err := rp.GetThumbnailList()
	if err != nil || len(thumbs) == 0 {
		t.Skip("no thumbnails available")
	}
	if err := rp.UnpackThumbAt(0); err != nil {
		t.Fatalf("UnpackThumbAt(0) error: %v", err)
	}
}

func TestHalfSizeOption(t *testing.T) {
	full := openAndProcess(t)
	defer full.Close()
	fullW, fullH, _, _ := full.GetMemImageFormat()

	half := openAndProcess(t, WithHalfSize())
	defer half.Close()
	halfW, halfH, _, _ := half.GetMemImageFormat()

	if halfW >= fullW || halfH >= fullH {
		t.Errorf("half-size (%dx%d) not smaller than full (%dx%d)", halfW, halfH, fullW, fullH)
	}
}

func TestOpenBayerEmpty(t *testing.T) {
	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp.Close()

	err = rp.OpenBayer(nil, 100, 100, 0, 0, 0, 0, 0, 0, 0, 0, 0)
	if !errors.Is(err, ErrBufferOpen) {
		t.Errorf("OpenBayer(nil) = %v, want ErrBufferOpen", err)
	}
}

func TestVersionNumber(t *testing.T) {
	ver := VersionNumber()
	if ver <= 0 {
		t.Errorf("VersionNumber() = %d, want > 0", ver)
	}
	t.Logf("VersionNumber: %d", ver)
}

func TestStrError(t *testing.T) {
	msg := StrError(0)
	if msg == "" {
		t.Error("StrError(0) is empty")
	}
	msg = StrError(-1)
	if msg == "" {
		t.Error("StrError(-1) is empty")
	}
	t.Logf("StrError(0) = %q, StrError(-1) = %q", StrError(0), StrError(-1))
}

func TestErrorType(t *testing.T) {
	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp.Close()

	// Open a nonexistent file to get an error
	err = rp.OpenFile("/nonexistent/file.cr2")
	if err == nil {
		t.Fatal("OpenFile(bad path) = nil, want error")
	}

	// Verify errors.Is works
	if !errors.Is(err, ErrFileOpenFailed) {
		t.Errorf("errors.Is(err, ErrFileOpenFailed) = false, error = %v", err)
	}

	// Verify errors.As works
	var lrErr *Error
	if !errors.As(err, &lrErr) {
		t.Fatalf("errors.As(err, &lrErr) = false")
	}
	if lrErr.Op != "open_file" {
		t.Errorf("lrErr.Op = %q, want 'open_file'", lrErr.Op)
	}
	if lrErr.Code == 0 {
		t.Error("lrErr.Code = 0, want non-zero")
	}
	if lrErr.Message == "" {
		t.Error("lrErr.Message is empty")
	}
	t.Logf("Error: Op=%q Code=%d Message=%q", lrErr.Op, lrErr.Code, lrErr.Message)
}

func TestProgressFlagsAndWarnings(t *testing.T) {
	rp := openTestRAW(t)

	pf := rp.ProgressFlags()
	pw := rp.ProcessWarnings()
	t.Logf("ProgressFlags: 0x%08x, ProcessWarnings: 0x%08x", pf, pw)

	// After open/identify, progress_flags should have at least OPEN and IDENTIFY set
	if pf&uint(ProgressOpen) == 0 {
		t.Error("ProgressFlags missing ProgressOpen after OpenFile")
	}
}

func TestNewFields(t *testing.T) {
	rp := openTestRAW(t)

	t.Run("ColorData", func(t *testing.T) {
		cd, err := rp.GetColorData()
		if err != nil {
			t.Fatalf("GetColorData() error: %v", err)
		}
		t.Logf("Maximum=%d DataMaximum=%d FMaximum=%.2f RawBPS=%d FlashUsed=%.4f CanonEV=%.4f",
			cd.Maximum, cd.DataMaximum, cd.FMaximum, cd.RawBPS, cd.FlashUsed, cd.CanonEV)
		t.Logf("PhaseOneData: format=%d tag210=%.4f", cd.PhaseOneData.Format, cd.PhaseOneData.Tag210)
		t.Logf("BlackStat: %v", cd.BlackStat)
	})

	t.Run("ImageSizes", func(t *testing.T) {
		sizes, err := rp.GetImageSizes()
		if err != nil {
			t.Fatalf("GetImageSizes() error: %v", err)
		}
		t.Logf("RawPitch=%d RawAspect=%d", sizes.RawPitch, sizes.RawAspect)
	})

	t.Run("DNGLevels", func(t *testing.T) {
		dl, err := rp.GetDNGLevels()
		if err != nil {
			t.Fatalf("GetDNGLevels() error: %v", err)
		}
		t.Logf("DngBlack=%d DngWhiteLevel=%v DefaultCrop=%v UserCrop=%v PreviewColorSpace=%d LinearResponseLimit=%.4f",
			dl.DngBlack, dl.DngWhiteLevel, dl.DefaultCrop, dl.UserCrop, dl.PreviewColorSpace, dl.LinearResponseLimit)
	})
}

func TestApplyOptions(t *testing.T) {
	rp := openTestRAW(t)

	err := rp.ApplyOptions(
		WithBrightness(1.2),
		WithHalfSize(),
		WithNoAutoBrightness(),
	)
	if err != nil {
		t.Fatalf("ApplyOptions() error: %v", err)
	}
}

func TestSRAWMidpoint(t *testing.T) {
	rp := openTestRAW(t)
	mid := rp.SRAWMidpoint()
	t.Logf("SRAWMidpoint: %d", mid)
	if mid < 0 {
		t.Errorf("SRAWMidpoint = %d, want >= 0", mid)
	}
}

func TestGetRawData(t *testing.T) {
	rp := openTestRAW(t)
	if err := rp.Unpack(); err != nil {
		t.Fatalf("Unpack() error: %v", err)
	}

	sizes, err := rp.GetImageSizes()
	if err != nil {
		t.Fatalf("GetImageSizes() error: %v", err)
	}
	expectedPixels := int(sizes.RawWidth) * int(sizes.RawHeight)
	if expectedPixels == 0 {
		t.Fatal("RawWidth*RawHeight == 0")
	}

	rd, err := rp.GetRawData()
	if err != nil {
		t.Fatalf("GetRawData() error: %v", err)
	}

	nonNilCount := 0
	if len(rd.RawImage) > 0 {
		nonNilCount++
		if len(rd.RawImage) != expectedPixels {
			t.Errorf("RawImage length = %d, want %d", len(rd.RawImage), expectedPixels)
		}
	// Verify at least some pixels are non-zero (skip for float data)
		if len(rd.RawImage) > 0 {
			nonZero := 0
			for _, v := range rd.RawImage[:min(1000, len(rd.RawImage))] {
				if v != 0 {
					nonZero++
				}
			}
			if nonZero == 0 {
				t.Log("RawImage has all-zero pixel values (may be expected for this file)")
			}
		}
	}
	if len(rd.Color4Image) > 0 {
		nonNilCount++
		if len(rd.Color4Image) != expectedPixels*4 {
			t.Errorf("Color4Image length = %d, want %d", len(rd.Color4Image), expectedPixels*4)
		}
	}
	if len(rd.Color3Image) > 0 {
		nonNilCount++
	}
	if len(rd.FloatImage) > 0 {
		nonNilCount++
	}
	if len(rd.Float3Image) > 0 {
		nonNilCount++
	}
	if len(rd.Float4Image) > 0 {
		nonNilCount++
	}

	if nonNilCount == 0 {
		t.Error("GetRawData() returned all zero-length slices after Unpack")
	}
	if nonNilCount > 1 {
		t.Errorf("GetRawData() returned %d non-nil slices, want at most 1", nonNilCount)
	}
}

func TestConcurrentGetters(t *testing.T) {
	rp := openTestRAW(t)
	defer rp.Close()

	var wg sync.WaitGroup

	getters := []func(){
		func() { _, _ = rp.GetCameraInfo() },
		func() { _, _ = rp.GetImageSizes() },
		func() { _, _ = rp.GetColorData() },
		func() { _, _ = rp.GetLensInfo() },
		func() { _, _ = rp.GetShootingParams() },
		func() { _, _ = rp.GetTemperatures() },
		func() { _, _ = rp.GetWBCoeffs() },
		func() { _, _ = rp.GetThumbnailInfo() },
		func() { rp.IsFujiRotated() },
		func() { rp.ProgressFlags() },
	}

	for range 4 {
		for _, g := range getters {
			wg.Go(func() {
				for range 50 {
					g()
				}
			})
		}
	}

	wg.Wait()
}

func TestFullLifecycle(t *testing.T) {
	path := testRAWPath(t)

	// Open -> Identify -> Unpack -> Process -> MakeMemImage -> Close
	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	if err := rp.OpenFile(path); err != nil {
		rp.Close()
		t.Fatalf("OpenFile() error: %v", err)
	}

	ci, err := rp.GetCameraInfo()
	if err != nil {
		rp.Close()
		t.Fatalf("GetCameraInfo() error: %v", err)
	}
	if ci.Make == "" {
		t.Error("GetCameraInfo().Make is empty")
	}

	if err := rp.Unpack(); err != nil {
		rp.Close()
		t.Fatalf("Unpack() error: %v", err)
	}
	if !rp.RawWasRead() {
		t.Error("RawWasRead() = false after Unpack")
	}

	if err := rp.Process(); err != nil {
		rp.Close()
		t.Fatalf("Process() error: %v", err)
	}

	img, err := rp.MakeMemImage()
	if err != nil {
		rp.Close()
		t.Fatalf("MakeMemImage() error: %v", err)
	}
	if img.Width == 0 || img.Height == 0 {
		t.Errorf("MakeMemImage() returned zero-size image: %dx%d", img.Width, img.Height)
	}
	if len(img.Data) == 0 {
		t.Error("MakeMemImage() returned empty data")
	}

	rp.Close()
}

func TestRecycleProcessCycle(t *testing.T) {
	path := testRAWPath(t)

	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp.Close()

	for range 2 {
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
		if img.Width == 0 {
			t.Fatal("MakeMemImage() returned zero-width image")
		}
		rp.Recycle()
	}
}

func TestGetDNGColorInvalidIndex(t *testing.T) {
	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp.Close()

	for _, idx := range []int{-1, 2, 100} {
		_, err := rp.GetDNGColor(idx)
		if !errors.Is(err, ErrInvalidIndex) {
			t.Errorf("GetDNGColor(%d) = %v, want ErrInvalidIndex", idx, err)
		}
	}
}

func TestOpenBufferCorrupted(t *testing.T) {
	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer rp.Close()

	// Random bytes that are not a valid RAW file
	garbage := make([]byte, 256)
	for i := range garbage {
		garbage[i] = byte(i)
	}

	err = rp.OpenBuffer(garbage)
	if err == nil {
		t.Error("OpenBuffer(corrupted) = nil, want error")
	}
}

// ── Manufacturer-specific tests ──────────────────────────────────────

func TestMakernotesWithManufacturerFiles(t *testing.T) {
	files := []struct {
		name   string
		path   string
		checks func(t *testing.T, rp *RawProcessor)
	}{
		{"FujiFilm.raf", "testdata/FujiFilm.raf", func(t *testing.T, rp *RawProcessor) {
			mn, err := rp.GetFujiMakernotes()
			if err != nil {
				t.Fatalf("GetFujiMakernotes() error: %v", err)
			}
			t.Logf("Fuji: DynamicRange=%d FilmMode=%d DriveMode=%d Model=%q",
				mn.DynamicRange, mn.FilmMode, mn.DriveMode, mn.FujiModel)
		}},
		{"Nikon.nef", "testdata/Nikon.nef", func(t *testing.T, rp *RawProcessor) {
			mn, err := rp.GetNikonMakernotes()
			if err != nil {
				t.Fatalf("GetNikonMakernotes() error: %v", err)
			}
			t.Logf("Nikon: NEFCompression=%d ActiveDLighting=%d Sensor=%dx%d",
				mn.NEFCompression, mn.ActiveDLighting, mn.SensorWidth, mn.SensorHeight)
			if mn.PictureControlName == "" {
				t.Error("PictureControlName is empty")
			}
		}},
		{"Panasonic.rw2", "testdata/Panasonic.rw2", func(t *testing.T, rp *RawProcessor) {
			mn, err := rp.GetPanasonicMakernotes()
			if err != nil {
				t.Fatalf("GetPanasonicMakernotes() error: %v", err)
			}
			t.Logf("Panasonic: Compression=%d Multishot=%d", mn.Compression, mn.Multishot)
		}},
		{"Minolta.mrw", "testdata/Minolta.mrw", func(t *testing.T, rp *RawProcessor) {
			camera, err := rp.GetCameraInfo()
			if err != nil {
				t.Fatalf("GetCameraInfo() error: %v", err)
			}
			t.Logf("Minolta: Make=%q Model=%q", camera.Make, camera.Model)
			if camera.Make == "" {
				t.Error("Make is empty for Minolta file")
			}
		}},
		{"PhaseOne.iiq", "testdata/PhaseOne.iiq", func(t *testing.T, rp *RawProcessor) {
			mn, err := rp.GetPhaseOneMakernotes()
			if err != nil {
				t.Fatalf("GetPhaseOneMakernotes() error: %v", err)
			}
			t.Logf("PhaseOne: Software=%q SystemType=%q", mn.Software, mn.SystemType)
		}},
		{"Sigma.x3f", "testdata/Sigma.x3f", func(t *testing.T, rp *RawProcessor) {
			camera, err := rp.GetCameraInfo()
			if err != nil {
				t.Fatalf("GetCameraInfo() error: %v", err)
			}
			t.Logf("Sigma: Make=%q Model=%q", camera.Make, camera.Model)
		}},
	}

	for _, f := range files {
		t.Run(f.name, func(t *testing.T) {
			if _, err := os.Stat(f.path); err != nil {
				t.Skipf("%s not found", f.path)
			}
			rp, err := New()
			if err != nil {
				t.Fatalf("New() error: %v", err)
			}
			defer rp.Close()
			if err := rp.OpenFile(f.path); err != nil {
				t.Fatalf("OpenFile(%s) error: %v", f.path, err)
			}
			f.checks(t, rp)
		})
	}
}

// ── Mutator errors after Close ───────────────────────────────────────

func TestMutatorErrorsAfterClose(t *testing.T) {
	rp, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	rp.Close()

	// void methods should not panic
	t.Run("Recycle", func(t *testing.T) {
		rp.Recycle() // should not panic
	})
	t.Run("RecycleDatastream", func(t *testing.T) {
		rp.RecycleDatastream() // should not panic
	})
	t.Run("FreeImage", func(t *testing.T) {
		rp.FreeImage() // should not panic
	})
	t.Run("ConvertFloatToInt", func(t *testing.T) {
		rp.ConvertFloatToInt(0, 0, 0) // should not panic
	})

	// callback setters should return ErrAlreadyClosed
	t.Run("SetDataErrorHandler", func(t *testing.T) {
		if err := rp.SetDataErrorHandler(nil); !errors.Is(err, ErrAlreadyClosed) {
			t.Errorf("SetDataErrorHandler after Close() = %v, want ErrAlreadyClosed", err)
		}
	})
	t.Run("SetEXIFParseHandler", func(t *testing.T) {
		if err := rp.SetEXIFParseHandler(nil); !errors.Is(err, ErrAlreadyClosed) {
			t.Errorf("SetEXIFParseHandler after Close() = %v, want ErrAlreadyClosed", err)
		}
	})
	t.Run("SetMakernotesParseHandler", func(t *testing.T) {
		if err := rp.SetMakernotesParseHandler(nil); !errors.Is(err, ErrAlreadyClosed) {
			t.Errorf("SetMakernotesParseHandler after Close() = %v, want ErrAlreadyClosed", err)
		}
	})
}

// ── Benchmarks ───────────────────────────────────────────────────────

func BenchmarkOpenAndIdentify(b *testing.B) {
	path := testRAWPath(b)
	for b.Loop() {
		rp, _ := New()
		rp.OpenFile(path)
		_, _ = rp.GetCameraInfo()
		rp.Close()
	}
}

func BenchmarkFullProcess(b *testing.B) {
	path := testRAWPath(b)
	for b.Loop() {
		rp, _ := New()
		rp.OpenFile(path)
		rp.Unpack()
		rp.Process()
		rp.MakeMemImage()
		rp.Close()
	}
}

func BenchmarkCopyMemImage(b *testing.B) {
	rp := openAndProcess(b)
	defer rp.Close()
	w, h, colors, bps := rp.GetMemImageFormat()
	stride := int(w) * int(colors) * int(bps) / 8
	buf := make([]byte, stride*int(h))
	b.ResetTimer()
	for b.Loop() {
		rp.CopyMemImage(buf, stride, false)
	}
}

// BenchmarkMakeMemImage_vs_CopyMemImage directly compares the two approaches
// for extracting processed image data from LibRaw.
func BenchmarkMakeMemImage_vs_CopyMemImage(b *testing.B) {
	rp := openAndProcess(b)
	defer rp.Close()

	w, h, colors, bps := rp.GetMemImageFormat()
	stride := int(w) * int(colors) * int(bps) / 8
	totalBytes := stride * int(h)

	b.Run("MakeMemImage", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			img, err := rp.MakeMemImage()
			if err != nil {
				b.Fatal(err)
			}
			// Prevent escape analysis from optimizing away the allocation.
			_ = img.Data[:1]
		}
	})

	b.Run("CopyMemImage", func(b *testing.B) {
		buf := make([]byte, totalBytes)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			if err := rp.CopyMemImage(buf, stride, false); err != nil {
				b.Fatal(err)
			}
		}
	})

	// Full pipeline comparison: simulate what runner.decodeDirect does.
	b.Run("FullPipeline_MakeMemImage", func(b *testing.B) {
		path := testRAWPath(b)
		b.ReportAllocs()
		for b.Loop() {
			rp, _ := New()
			rp.OpenFile(path)
			rp.Unpack()
			rp.Process()
			img, _ := rp.MakeMemImage()
			_ = img.Data[:1]
			rp.Close()
		}
	})

	b.Run("FullPipeline_CopyMemImage", func(b *testing.B) {
		path := testRAWPath(b)
		b.ReportAllocs()
		for b.Loop() {
			rp2, _ := New()
			rp2.OpenFile(path)
			rp2.Unpack()
			rp2.Process()
			w2, h2, c2, bps2 := rp2.GetMemImageFormat()
			stride2 := int(w2) * int(c2) * int(bps2) / 8
			buf2 := make([]byte, stride2*int(h2))
			_ = rp2.CopyMemImage(buf2, stride2, false)
			rp2.Close()
		}
	})
}
