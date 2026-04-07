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

	count := img.EXIF.Count()
	if count == 0 {
		t.Fatal("expected EXIF tags in JPEG")
	}

	for i := range count {
		key, err := img.EXIF.Key(i)
		if err != nil {
			t.Fatalf("EXIF.Key(%d): %v", i, err)
		}
		if !img.EXIF.Has(key) {
			t.Errorf("EXIF.Has(%q) false after EXIF.Key(%d) returned it", key, i)
		}
		tag, ok := img.EXIF.Tag(key)
		if !ok {
			t.Logf("  [%d] %s: EXIF.Tag not found", i, key)
		} else {
			t.Logf("  [%d] %s = %s (tag=%d type=%d ifd=%d)",
				i, key, tag.Value, tag.TagID, tag.TypeID, tag.IfdID)
		}
	}
}

func TestExifTypedAccess(t *testing.T) {
	img := openMetadata(t, "DSC_3079.jpg")

	t.Run("Int", func(t *testing.T) {
		tag, ok := img.EXIF.Tag("Exif.Photo.Flash")
		if !ok {
			t.Fatal("Exif.Photo.Flash not found")
		}
		val, err := tag.Int()
		if err != nil {
			t.Fatalf("tag.Int(): %v", err)
		}
		t.Logf("Flash: %d (tagID=%d)", val, tag.TagID)
	})

	t.Run("Float", func(t *testing.T) {
		tag, ok := img.EXIF.Tag("Exif.Photo.FNumber")
		if !ok {
			t.Fatal("Exif.Photo.FNumber not found")
		}
		fnum, err := tag.Float()
		if err != nil {
			t.Fatalf("tag.Float(): %v", err)
		}
		if fnum <= 0 {
			t.Errorf("FNumber = %.1f, want > 0", fnum)
		}
		t.Logf("FNumber: %.1f", fnum)

		tag2, ok := img.EXIF.Tag("Exif.Photo.ExposureTime")
		if !ok {
			t.Fatal("Exif.Photo.ExposureTime not found")
		}
		exp, err := tag2.Float()
		if err != nil {
			t.Fatalf("tag.Float(ExposureTime): %v", err)
		}
		t.Logf("ExposureTime: %v", exp)
	})

	t.Run("Binary", func(t *testing.T) {
		tag, ok := img.EXIF.Tag("Exif.Photo.MakerNote")
		if !ok {
			t.Skip("no MakerNote in test file")
		}
		if !tag.Binary() {
			t.Fatal("expected MakerNote to have raw bytes")
		}
		if len(tag.Raw) == 0 {
			t.Fatal("expected non-empty MakerNote raw data")
		}
		t.Logf("MakerNote: %d bytes", len(tag.Raw))
	})

	t.Run("NonexistentKey", func(t *testing.T) {
		_, ok := img.EXIF.Tag("Exif.Photo.NoTag")
		if ok {
			t.Error("expected false for nonexistent key")
		}
	})

	t.Run("RawFields", func(t *testing.T) {
		tag, ok := img.EXIF.Tag("Exif.Image.Make")
		if !ok {
			t.Fatal("Exif.Image.Make not found")
		}
		if tag.TagID == 0 {
			t.Error("expected non-zero TagID for Make")
		}
		if tag.TypeID == 0 {
			t.Error("expected non-zero TypeID for Make")
		}
		if tag.Size == 0 {
			t.Error("expected non-zero Size for Make")
		}
		t.Logf("Make: tag=%d type=%d count=%d size=%d ifd=%d",
			tag.TagID, tag.TypeID, tag.Count, tag.Size, tag.IfdID)
	})
}

// ---------------------------------------------------------------------------
// GPS
// ---------------------------------------------------------------------------

func TestReadGPS(t *testing.T) {
	img := openMetadata(t, "GPS.jpg")

	lat, ok := img.EXIF.Tag("Exif.GPSInfo.GPSLatitude")
	if !ok {
		t.Fatal("GPSLatitude not found")
	}
	lon, ok := img.EXIF.Tag("Exif.GPSInfo.GPSLongitude")
	if !ok {
		t.Fatal("GPSLongitude not found")
	}
	t.Logf("GPS: %s, %s", lat.Value, lon.Value)
}

// ---------------------------------------------------------------------------
// IPTC
// ---------------------------------------------------------------------------

func TestReadIPTC(t *testing.T) {
	img := openMetadata(t, "IPTC.jpg")

	count := img.IPTC.Count()
	if count == 0 {
		t.Fatal("expected IPTC tags")
	}
	t.Logf("IPTC tags: %d", count)

	for i := range count {
		key, err := img.IPTC.Key(i)
		if err != nil {
			t.Fatalf("IPTC.Key(%d): %v", i, err)
		}
		if !img.IPTC.Has(key) {
			t.Errorf("IPTC.Has(%q) false after IPTC.Key(%d)", key, i)
		}
		tag, _ := img.IPTC.Tag(key)
		t.Logf("  [%d] %s = %s (tag=%d record=%d)",
			i, key, tag.Value, tag.TagID, tag.Record)
	}

	tag, ok := img.IPTC.Tag("Iptc.Application2.Caption")
	if !ok {
		t.Fatal("IPTC Caption not found")
	}
	if tag.Value == "" {
		t.Error("expected non-empty Caption")
	}
}

// ---------------------------------------------------------------------------
// XMP
// ---------------------------------------------------------------------------

func TestReadXMP(t *testing.T) {
	img := openMetadata(t, "XMP.jpg")

	count := img.XMP.Count()
	if count == 0 {
		t.Fatal("expected XMP tags")
	}
	t.Logf("XMP tags: %d", count)

	for i := range count {
		key, err := img.XMP.Key(i)
		if err != nil {
			t.Fatalf("XMP.Key(%d): %v", i, err)
		}
		if !img.XMP.Has(key) {
			t.Errorf("XMP.Has(%q) false after XMP.Key(%d)", key, i)
		}
		tag, _ := img.XMP.Tag(key)
		t.Logf("  [%d] %s = %s (tagID=%d)", i, key, tag.Value, tag.TagID)
	}

	packet := img.XMPPacket
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
			if img.EXIF.Count() == 0 {
				t.Error("expected EXIF tags")
			}
			t.Logf("EXIF: %d, IPTC: %d, XMP: %d",
				img.EXIF.Count(), img.IPTC.Count(), img.XMP.Count())
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
			if img.EXIF.Count() != 0 {
				t.Errorf("EXIF = %d, want 0", img.EXIF.Count())
			}
			if img.IPTC.Count() != 0 {
				t.Errorf("IPTC = %d, want 0", img.IPTC.Count())
			}
			if img.XMP.Count() != 0 {
				t.Errorf("XMP = %d, want 0", img.XMP.Count())
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

	t.Run("EXIF.Count", func(t *testing.T) {
		if v := img.EXIF.Count(); v != 0 {
			t.Errorf("got %d, want 0", v)
		}
	})

	t.Run("EXIF.Tag", func(t *testing.T) {
		_, ok := img.EXIF.Tag("Exif.Image.Make")
		if ok {
			t.Error("expected false after close")
		}
	})

	t.Run("XMPPacket", func(t *testing.T) {
		if img.XMPPacket != "" {
			t.Error("expected empty XMPPacket after close")
		}
	})
}

func TestExifKeyOutOfRange(t *testing.T) {
	img := openMetadata(t, "DSC_3079.jpg")

	for _, idx := range []int{-1, 99999} {
		_, err := img.EXIF.Key(idx)
		if err == nil {
			t.Errorf("EXIF.Key(%d) should return error", idx)
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
			_ = img.EXIF.Count()
			_, ok := img.EXIF.Tag("Exif.Image.Make")
			if !ok {
				errCh <- errors.New("Exif.Image.Make not found")
				return
			}
			errCh <- nil
		}()
	}

	for range n {
		if err := <-errCh; err != nil {
			t.Errorf("concurrent read error: %v", err)
		}
	}
}
