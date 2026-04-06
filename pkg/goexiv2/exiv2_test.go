package goexiv2

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func testFile(name string) string {
	return filepath.Join("testdata", name)
}

// openMetadata is a test helper that opens a file, reads metadata, and
// schedules cleanup. Fatals on any error.
func openMetadata(t *testing.T, name string) *Image {
	t.Helper()
	img, err := Open(testFile(name))
	if err != nil {
		t.Fatalf("Open %s: %v", name, err)
	}
	if err := img.ReadMetadata(); err != nil {
		img.Close()
		t.Fatalf("ReadMetadata %s: %v", name, err)
	}
	t.Cleanup(func() { img.Close() })
	return img
}

// ---------------------------------------------------------------------------
// Lifecycle & basics
// ---------------------------------------------------------------------------

func TestVersion(t *testing.T) {
	v := Version()
	if v == "" {
		t.Fatal("Version() returned empty")
	}
	t.Logf("exiv2 version: %s", v)
}

func TestOpenNotFound(t *testing.T) {
	_, err := Open("/nonexistent/file.tiff")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestDoubleClose(t *testing.T) {
	img, err := Open(testFile("DSC_3079.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	if err := img.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := img.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}

// ---------------------------------------------------------------------------
// EXIF reading
// ---------------------------------------------------------------------------

func TestReadJPEG(t *testing.T) {
	img := openMetadata(t, "DSC_3079.jpg")

	count := img.ExifCount()
	if count == 0 {
		t.Fatal("expected EXIF tags in JPEG")
	}

	// Verify all keys are iterable and Has/String round-trip works.
	for i := 0; i < count; i++ {
		key, err := img.ExifKey(i)
		if err != nil {
			t.Fatalf("ExifKey(%d): %v", i, err)
		}
		if !img.ExifHas(key) {
			t.Errorf("ExifHas(%q) false after ExifKey(%d) returned it", key, i)
		}
		val, err := img.ExifString(key)
		if err != nil {
			t.Logf("  [%d] %s: ExifString error: %v", i, key, err)
		} else {
			t.Logf("  [%d] %s = %s", i, key, val)
		}
	}
}

func TestExifTypedAccess(t *testing.T) {
	img := openMetadata(t, "DSC_3079.jpg")

	t.Run("Long", func(t *testing.T) {
		val, err := img.ExifLong("Exif.Photo.Flash")
		if err != nil {
			t.Fatalf("ExifLong(Flash): %v", err)
		}
		t.Logf("Flash: %d", val)
	})

	t.Run("Double", func(t *testing.T) {
		fnum, err := img.ExifDouble("Exif.Photo.FNumber")
		if err != nil {
			t.Fatalf("ExifDouble(FNumber): %v", err)
		}
		if fnum <= 0 {
			t.Errorf("FNumber = %.1f, want > 0", fnum)
		}
		t.Logf("FNumber: %.1f", fnum)

		exp, err := img.ExifDouble("Exif.Photo.ExposureTime")
		if err != nil {
			t.Fatalf("ExifDouble(ExposureTime): %v", err)
		}
		t.Logf("ExposureTime: %v", exp)
	})

	t.Run("Bytes", func(t *testing.T) {
		if !img.ExifHas("Exif.Photo.MakerNote") {
			t.Skip("no MakerNote in test file")
		}
		data, err := img.ExifBytes("Exif.Photo.MakerNote")
		if err != nil {
			t.Fatalf("ExifBytes(MakerNote): %v", err)
		}
		if len(data) == 0 {
			t.Fatal("expected non-empty MakerNote")
		}
		t.Logf("MakerNote: %d bytes", len(data))
	})

	t.Run("NonexistentKey", func(t *testing.T) {
		for _, fn := range []struct {
			name string
			fn   func() error
		}{
			{"ExifLong", func() error { _, err := img.ExifLong("Exif.Photo.NoTag"); return err }},
			{"ExifDouble", func() error { _, err := img.ExifDouble("Exif.Photo.NoTag"); return err }},
			{"ExifBytes", func() error { _, err := img.ExifBytes("Exif.Photo.NoTag"); return err }},
			{"ExifString", func() error { _, err := img.ExifString("Exif.Photo.NoTag"); return err }},
		} {
			if err := fn.fn(); err == nil {
				t.Errorf("%s: expected error for nonexistent key", fn.name)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// GPS
// ---------------------------------------------------------------------------

func TestReadGPS(t *testing.T) {
	img := openMetadata(t, "GPS.jpg")

	lat, err := img.ExifString("Exif.GPSInfo.GPSLatitude")
	if err != nil {
		t.Fatalf("GPSLatitude: %v", err)
	}
	lon, err := img.ExifString("Exif.GPSInfo.GPSLongitude")
	if err != nil {
		t.Fatalf("GPSLongitude: %v", err)
	}
	t.Logf("GPS: %s, %s", lat, lon)
}

// ---------------------------------------------------------------------------
// IPTC
// ---------------------------------------------------------------------------

func TestReadIPTC(t *testing.T) {
	img := openMetadata(t, "IPTC.jpg")

	count := img.IPTCCount()
	if count == 0 {
		t.Fatal("expected IPTC tags")
	}
	t.Logf("IPTC tags: %d", count)

	for i := 0; i < count; i++ {
		key, err := img.IPTCKey(i)
		if err != nil {
			t.Fatalf("IPTCKey(%d): %v", i, err)
		}
		if !img.IPTCHas(key) {
			t.Errorf("IPTCHas(%q) false after IPTCKey(%d)", key, i)
		}
		val, _ := img.IPTCString(key)
		t.Logf("  [%d] %s = %s", i, key, val)
	}

	caption, err := img.IPTCString("Iptc.Application2.Caption")
	if err != nil {
		t.Fatalf("IPTCString(Caption): %v", err)
	}
	if caption == "" {
		t.Error("expected non-empty Caption")
	}
}

// ---------------------------------------------------------------------------
// XMP
// ---------------------------------------------------------------------------

func TestReadXMP(t *testing.T) {
	img := openMetadata(t, "XMP.jpg")

	count := img.XMPCount()
	if count == 0 {
		t.Fatal("expected XMP tags")
	}
	t.Logf("XMP tags: %d", count)

	for i := 0; i < count; i++ {
		key, err := img.XMPKey(i)
		if err != nil {
			t.Fatalf("XMPKey(%d): %v", i, err)
		}
		if !img.XMPHas(key) {
			t.Errorf("XMPHas(%q) false after XMPKey(%d)", key, i)
		}
		val, _ := img.XMPString(key)
		t.Logf("  [%d] %s = %s", i, key, val)
	}

	packet, err := img.XMPPacket()
	if err != nil {
		t.Fatalf("XMPPacket: %v", err)
	}
	if packet == "" {
		t.Fatal("expected non-empty XMP packet")
	}
	if !strings.Contains(packet, "<?xpacket") && !strings.Contains(packet, "<x:xmpmeta") {
		t.Error("XMP packet doesn't look like valid XML")
	}
}

// ---------------------------------------------------------------------------
// Multiple formats
// ---------------------------------------------------------------------------

func TestReadFormats(t *testing.T) {
	tests := []struct {
		name string
		file string
	}{
		{"TIFF", "exiv2-bug1044.tif"},
		{"PNG", "1343_exif.png"},
		{"DNG", "DNG.dng"},
		{"NEF", "Nikon.nef"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			img := openMetadata(t, tc.file)
			if img.ExifCount() == 0 {
				t.Error("expected EXIF tags")
			}
			t.Logf("EXIF: %d, IPTC: %d, XMP: %d",
				img.ExifCount(), img.IPTCCount(), img.XMPCount())
		})
	}
}

// ---------------------------------------------------------------------------
// Empty files (no metadata)
// ---------------------------------------------------------------------------

func TestEmptyFiles(t *testing.T) {
	tests := []struct {
		name string
		file string
	}{
		{"JPEG", "exiv2-empty.jpg"},
		{"PNG", "1343_empty.png"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			img := openMetadata(t, tc.file)
			if img.ExifCount() != 0 {
				t.Errorf("EXIF = %d, want 0", img.ExifCount())
			}
			if img.IPTCCount() != 0 {
				t.Errorf("IPTC = %d, want 0", img.IPTCCount())
			}
			if img.XMPCount() != 0 {
				t.Errorf("XMP = %d, want 0", img.XMPCount())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Error paths
// ---------------------------------------------------------------------------

func TestOperationsAfterClose(t *testing.T) {
	img, err := Open(testFile("DSC_3079.jpg"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := img.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	t.Run("ReadMetadata", func(t *testing.T) {
		if err := img.ReadMetadata(); !errors.Is(err, ErrClosed) {
			t.Errorf("got %v, want ErrClosed", err)
		}
	})

	t.Run("ExifCount", func(t *testing.T) {
		if v := img.ExifCount(); v != 0 {
			t.Errorf("got %d, want 0", v)
		}
	})

	t.Run("ExifString", func(t *testing.T) {
		_, err := img.ExifString("Exif.Image.Make")
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("ExifLong", func(t *testing.T) {
		_, err := img.ExifLong("Exif.Photo.Flash")
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("ExifDouble", func(t *testing.T) {
		_, err := img.ExifDouble("Exif.Photo.FNumber")
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("XMPPacket", func(t *testing.T) {
		_, err := img.XMPPacket()
		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestExifKeyOutOfRange(t *testing.T) {
	img := openMetadata(t, "DSC_3079.jpg")

	for _, idx := range []int{-1, 99999} {
		_, err := img.ExifKey(idx)
		if err == nil {
			t.Errorf("ExifKey(%d) should return error", idx)
		}
	}
}

// ---------------------------------------------------------------------------
// Concurrency
// ---------------------------------------------------------------------------

func TestConcurrentRead(t *testing.T) {
	img := openMetadata(t, "DSC_3079.jpg")

	const n = 10
	errCh := make(chan error, n)

	for range n {
		go func() {
			_ = img.ExifCount()
			_, err := img.ExifString("Exif.Image.Make")
			errCh <- err
		}()
	}

	for range n {
		if err := <-errCh; err != nil {
			t.Errorf("concurrent read error: %v", err)
		}
	}
}
