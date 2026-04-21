package golibtiff

import (
	"errors"
	"math"
	"path/filepath"
	"testing"
)

// --- Helpers ---

func createTestTIFF(t *testing.T, path string, w, h uint32) {
	t.Helper()
	tif, err := Open(path, OpenWrite)
	if err != nil {
		t.Fatalf("Open write: %v", err)
	}

	if err := tif.SetFieldUint32(TagImageWidth, w); err != nil {
		t.Fatalf("SetField width: %v", err)
	}
	if err := tif.SetFieldUint32(TagImageLength, h); err != nil {
		t.Fatalf("SetField height: %v", err)
	}
	if err := tif.SetFieldUint16(TagBitsPerSample, 8); err != nil {
		t.Fatalf("SetField bits: %v", err)
	}
	if err := tif.SetFieldUint16(TagSamplesPerPixel, 3); err != nil {
		t.Fatalf("SetField samples: %v", err)
	}
	if err := tif.SetFieldUint16(TagCompression, uint16(CompressionNone)); err != nil {
		t.Fatalf("SetField compression: %v", err)
	}
	if err := tif.SetFieldUint16(TagPhotometric, uint16(PhotometricRGB)); err != nil {
		t.Fatalf("SetField photometric: %v", err)
	}
	if err := tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig)); err != nil {
		t.Fatalf("SetField planarconfig: %v", err)
	}
	if err := tif.SetFieldString(TagSoftware, "golibtiff test"); err != nil {
		t.Fatalf("SetField software: %v", err)
	}

	scanline := make([]byte, w*3)
	for row := uint32(0); row < h; row++ {
		for i := range scanline {
			scanline[i] = byte(row % 256)
		}
		if err := tif.WriteScanline(scanline, row); err != nil {
			t.Fatalf("WriteScanline %d: %v", row, err)
		}
	}
	tif.Close()
}

func createGrayscaleTIFF(t *testing.T, path string, w, h uint32) {
	t.Helper()
	tif, err := Open(path, OpenWrite)
	if err != nil {
		t.Fatalf("Open write: %v", err)
	}
	tif.SetFieldUint32(TagImageWidth, w)
	tif.SetFieldUint32(TagImageLength, h)
	tif.SetFieldUint16(TagBitsPerSample, 8)
	tif.SetFieldUint16(TagSamplesPerPixel, 1)
	tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
	tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
	tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))
	scanline := make([]byte, w)
	for row := uint32(0); row < h; row++ {
		for i := range scanline {
			scanline[i] = byte(row % 256)
		}
		if err := tif.WriteScanline(scanline, row); err != nil {
			t.Fatalf("WriteScanline %d: %v", row, err)
		}
	}
	tif.Close()
}

// writeMultiPageTIFF creates a TIFF with nDirectories pages.
// Each page is a single-pixel grayscale image; the pixel value equals the page index.
func writeMultiPageTIFF(t *testing.T, path string) {
	t.Helper()
	tif, err := Open(path, OpenWrite)
	if err != nil {
		t.Fatalf("Open write: %v", err)
	}

	for i := range nDirectories {
		tif.SetFieldUint32(TagImageWidth, 1)
		tif.SetFieldUint32(TagImageLength, 1)
		tif.SetFieldUint16(TagBitsPerSample, 8)
		tif.SetFieldUint16(TagSamplesPerPixel, 1)
		tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
		tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
		tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))

		pixel := []byte{byte(i)}
		if err := tif.WriteScanline(pixel, 0); err != nil {
			t.Fatalf("WriteScanline page %d: %v", i, err)
		}
		if i < nDirectories-1 {
			if err := tif.WriteDirectory(); err != nil {
				t.Fatalf("WriteDirectory page %d: %v", i, err)
			}
		}
	}
	tif.Close()
}

func openFixture(t *testing.T, name string) *TIFF {
	t.Helper()
	path := filepath.Join("testdata", name)
	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open %s: %v", name, err)
	}
	return tif
}

// --- Core Open/Close ---

func TestOpenClose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.tif")
	createTestTIFF(t, path, 8, 8)

	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open read: %v", err)
	}

	name := tif.FileName()
	if name != path {
		t.Errorf("FileName = %q, want %q", name, path)
	}

	tif.Close()
	// Double close should not panic.
	tif.Close()
}

func TestOpenNonexistent(t *testing.T) {
	t.Parallel()
	_, err := Open("/nonexistent/path/test.tif", OpenRead)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestClosedHandle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "closed.tif")
	createTestTIFF(t, path, 8, 8)

	tif, _ := Open(path, OpenRead)
	tif.Close()

	v, err := tif.Width()
	if err == nil {
		t.Error("Width on closed handle should error")
	} else if v != 0 {
		t.Error("Width on closed handle should return 0")
	}
	if tif.IsTiled() {
		t.Error("IsTiled on closed handle should return false")
	}
	if tif.ReadDirectory() {
		t.Error("ReadDirectory on closed handle should return false")
	}
	if _, err := tif.GetFieldUint16(TagCompression); err == nil {
		t.Error("GetFieldUint16 on closed handle should error")
	}
}

// --- Read/Write round-trip ---

func TestWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.tif")

	const w, h = 64, 32
	createTestTIFF(t, path, w, h)

	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	if v, err := tif.Width(); err != nil {
		t.Fatalf("Width: %v", err)
	} else if v != w {
		t.Errorf("Width = %d, want %d", v, w)
	}
	if v, err := tif.Height(); err != nil {
		t.Fatalf("Height: %v", err)
	} else if v != h {
		t.Errorf("Height = %d, want %d", v, h)
	}
	if v, err := tif.BitsPerSample(); err != nil {
		t.Fatalf("BitsPerSample: %v", err)
	} else if v != 8 {
		t.Errorf("BitsPerSample = %d, want 8", v)
	}
	if v, err := tif.SamplesPerPixel(); err != nil {
		t.Fatalf("SamplesPerPixel: %v", err)
	} else if v != 3 {
		t.Errorf("SamplesPerPixel = %d, want 3", v)
	}
	if v, err := tif.Compression(); err != nil {
		t.Fatalf("Compression: %v", err)
	} else if v != uint16(CompressionNone) {
		t.Errorf("Compression = %d, want %d", v, CompressionNone)
	}
	if v, err := tif.Photometric(); err != nil {
		t.Fatalf("Photometric: %v", err)
	} else if v != uint16(PhotometricRGB) {
		t.Errorf("Photometric = %d, want %d", v, PhotometricRGB)
	}
	if v, err := tif.PlanarConfig(); err != nil {
		t.Fatalf("PlanarConfig: %v", err)
	} else if v != uint16(PlanarConfigContig) {
		t.Errorf("PlanarConfig = %d, want %d", v, PlanarConfigContig)
	}

	sw, err := tif.Software()
	if err != nil {
		t.Errorf("Software error: %v", err)
	}
	if sw != "golibtiff test" {
		t.Errorf("Software = %q, want %q", sw, "golibtiff test")
	}

	scanlineSize := tif.ScanlineSize()
	if scanlineSize != int(w*3) {
		t.Errorf("ScanlineSize = %d, want %d", scanlineSize, w*3)
	}
	buf := make([]byte, scanlineSize)
	for row := uint32(0); row < h; row++ {
		if err := tif.ReadScanline(buf, row); err != nil {
			t.Fatalf("ReadScanline %d: %v", row, err)
		}
		expected := byte(row % 256)
		for i := range buf {
			if buf[i] != expected {
				t.Fatalf("row %d byte %d = %d, want %d", row, i, buf[i], expected)
			}
		}
	}
}

func TestCompression(t *testing.T) {
	tests := []struct {
		name        string
		compression Compression
		w, h        uint32
	}{
		{"LZW", CompressionLZW, 128, 64},
		{"Deflate", CompressionDeflate, 64, 32},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, tc.name+".tif")
			// create file with specified compression
			tif, err := Open(path, OpenWrite)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			tif.SetFieldUint32(TagImageWidth, tc.w)
			tif.SetFieldUint32(TagImageLength, tc.h)
			tif.SetFieldUint16(TagBitsPerSample, 8)
			tif.SetFieldUint16(TagSamplesPerPixel, 1)
			tif.SetFieldUint16(TagCompression, uint16(tc.compression))
			tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
			tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))
			tif.SetFieldUint32(TagRowsPerStrip, tc.h)
			data := make([]byte, tc.w*tc.h)
			for i := range data {
				data[i] = byte(i % 256)
			}
			if _, err := tif.WriteEncodedStrip(0, data); err != nil {
				t.Fatalf("WriteEncodedStrip: %v", err)
			}
			tif.Close()

			// read back
			tif2, err := Open(path, OpenRead)
			if err != nil {
				t.Fatalf("Open read: %v", err)
			}
			defer tif2.Close()
			if v, err := tif2.Compression(); err != nil {
				t.Fatalf("Compression: %v", err)
			} else if v != uint16(tc.compression) {
				t.Errorf("Compression = %d, want %d", v, tc.compression)
			}
			buf := make([]byte, tc.w*tc.h)
			n, err := tif2.ReadEncodedStrip(0, buf, -1)
			if err != nil {
				t.Fatalf("ReadEncodedStrip: %v", err)
			}
			if n != int(tc.w*tc.h) {
				t.Errorf("ReadEncodedStrip returned %d bytes, want %d", n, tc.w*tc.h)
			}
			for i := range tc.w * tc.h {
				if buf[i] != byte(i%256) {
					t.Errorf("byte %d mismatch: got %d, want %d", i, buf[i], byte(i%256))
					break
				}
			}
		})
	}
}

func TestWriteAndReadEncodedStrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "strip.tif")

	const w, h = 32, 16

	func() {
		tif, err := Open(path, OpenWrite)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer tif.Close()

		tif.SetFieldUint32(TagImageWidth, w)
		tif.SetFieldUint32(TagImageLength, h)
		tif.SetFieldUint16(TagBitsPerSample, 8)
		tif.SetFieldUint16(TagSamplesPerPixel, 1)
		tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
		tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
		tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))
		tif.SetFieldUint32(TagRowsPerStrip, h)

		data := make([]byte, w*h)
		for i := range data {
			data[i] = byte(i % 256)
		}
		n, err := tif.WriteEncodedStrip(0, data)
		if err != nil {
			t.Fatalf("WriteEncodedStrip: %v", err)
		}
		if n != w*h {
			t.Errorf("WriteEncodedStrip wrote %d bytes, want %d", n, w*h)
		}
	}()

	tif2, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open read: %v", err)
	}
	defer tif2.Close()

	if tif2.NumberOfStrips() != 1 {
		t.Errorf("NumberOfStrips = %d, want 1", tif2.NumberOfStrips())
	}

	buf := make([]byte, w*h)
	n, err := tif2.ReadEncodedStrip(0, buf, -1)
	if err != nil {
		t.Fatalf("ReadEncodedStrip: %v", err)
	}
	if n != w*h {
		t.Errorf("ReadEncodedStrip returned %d bytes, want %d", n, w*h)
	}
	for i := range buf {
		if buf[i] != byte(i%256) {
			t.Errorf("byte %d = %d, want %d", i, buf[i], byte(i%256))
			break
		}
	}
}

// --- Field get/set ---

func TestSetFieldString(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tags.tif")

	func() {
		tif, err := Open(path, OpenWrite)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer tif.Close()

		tif.SetFieldUint32(TagImageWidth, 1)
		tif.SetFieldUint32(TagImageLength, 1)
		tif.SetFieldUint16(TagBitsPerSample, 8)
		tif.SetFieldUint16(TagSamplesPerPixel, 1)
		tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
		tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
		tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))
		tif.SetFieldString(TagArtist, "test artist")
		tif.SetFieldString(TagDocumentName, "test doc")
		tif.SetFieldFloat(TagXResolution, 72.0)
		tif.SetFieldFloat(TagYResolution, 72.0)
		tif.SetFieldUint16(TagResolutionUnit, uint16(ResolutionUnitInch))

		buf := []byte{0}
		tif.WriteScanline(buf, 0)
	}()

	tif2, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif2.Close()

	artist, err := tif2.GetFieldString(TagArtist)
	if err != nil || artist != "test artist" {
		t.Errorf("Artist = %q, err=%v", artist, err)
	}

	doc, err := tif2.GetFieldString(TagDocumentName)
	if err != nil || doc != "test doc" {
		t.Errorf("DocumentName = %q, err=%v", doc, err)
	}

	xres, err := tif2.XResolution()
	if err != nil || xres != 72.0 {
		t.Errorf("XResolution = %v, err=%v", xres, err)
	}

	yres, err := tif2.YResolution()
	if err != nil || yres != 72.0 {
		t.Errorf("YResolution = %v, err=%v", yres, err)
	}
}

func TestGetFieldUint16Slice(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "extrasamples.tif")

	func() {
		tif, err := Open(path, OpenWrite)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer tif.Close()

		tif.SetFieldUint32(TagImageWidth, 4)
		tif.SetFieldUint32(TagImageLength, 4)
		tif.SetFieldUint16(TagBitsPerSample, 8)
		tif.SetFieldUint16(TagSamplesPerPixel, 2)
		tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
		tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
		tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))

		// ExtraSamples is a true C16_UINT16 array tag.
		if err := tif.SetFieldUint16Slice(TagExtraSamples, []uint16{1}); err != nil {
			t.Fatalf("SetFieldUint16Slice ExtraSamples: %v", err)
		}

		scanline := make([]byte, 4*2)
		if err := tif.WriteScanline(scanline, 0); err != nil {
			t.Fatalf("WriteScanline: %v", err)
		}
	}()

	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open read: %v", err)
	}
	defer tif.Close()

	es, err := tif.GetFieldUint16Slice(TagExtraSamples)
	if err != nil {
		t.Fatalf("GetFieldUint16Slice ExtraSamples: %v", err)
	}
	if len(es) != 1 {
		t.Fatalf("ExtraSamples length = %d, want 1", len(es))
	}
	if es[0] != 1 {
		t.Errorf("ExtraSamples[0] = %d, want 1", es[0])
	}
}

func TestGetFieldUint32RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "u32.tif")

	const want uint32 = 2 // FILETYPE_REDUCEDIMAGE

	func() {
		tif, err := Open(path, OpenWrite)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer tif.Close()

		tif.SetFieldUint32(TagImageWidth, 4)
		tif.SetFieldUint32(TagImageLength, 4)
		tif.SetFieldUint16(TagBitsPerSample, 8)
		tif.SetFieldUint16(TagSamplesPerPixel, 1)
		tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
		tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
		tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))
		// NewSubfileType is a uint32 tag that libtiff does not override.
		tif.SetFieldUint32(TagNewSubfileType, want)

		data := make([]byte, 4*4)
		if _, err := tif.WriteEncodedStrip(0, data); err != nil {
			t.Fatalf("WriteEncodedStrip: %v", err)
		}
	}()

	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open read: %v", err)
	}
	defer tif.Close()

	got, err := tif.GetFieldUint32(TagNewSubfileType)
	if err != nil {
		t.Fatalf("GetFieldUint32 NewSubfileType: %v", err)
	}
	if got != want {
		t.Errorf("NewSubfileType = %d, want %d", got, want)
	}
}

func TestSetFieldUint32SliceEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "u32slice.tif")

	tif, err := Open(path, OpenWrite)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	tif.SetFieldUint32(TagImageWidth, 1)
	tif.SetFieldUint32(TagImageLength, 1)
	tif.SetFieldUint16(TagBitsPerSample, 8)
	tif.SetFieldUint16(TagSamplesPerPixel, 1)
	tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
	tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
	tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))

	// Empty slice should be a no-op.
	if err := tif.SetFieldUint32Slice(TagSubIFD, nil); err != nil {
		t.Errorf("SetFieldUint32Slice(nil) error: %v", err)
	}

	data := []byte{0}
	if err := tif.WriteScanline(data, 0); err != nil {
		t.Fatalf("WriteScanline: %v", err)
	}
}

func TestSetFieldFloatSlice(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "floatslice.tif")

	want := []float64{1.0, 1.0, 1.0}

	func() {
		tif, err := Open(path, OpenWrite)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer tif.Close()

		tif.SetFieldUint32(TagImageWidth, 1)
		tif.SetFieldUint32(TagImageLength, 1)
		tif.SetFieldUint16(TagBitsPerSample, 8)
		tif.SetFieldUint16(TagSamplesPerPixel, 3)
		tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
		tif.SetFieldUint16(TagPhotometric, uint16(PhotometricRGB))
		tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))

		if err := tif.SetFieldFloatSlice(TagAsShotNeutral, want); err != nil {
			t.Fatalf("SetFieldFloatSlice: %v", err)
		}

		scanline := make([]byte, 3)
		if err := tif.WriteScanline(scanline, 0); err != nil {
			t.Fatalf("WriteScanline: %v", err)
		}
	}()

	// Verify the file can be opened and has correct dimensions.
	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open read: %v", err)
	}
	defer tif.Close()

	wv, _ := tif.Width()
	hv, _ := tif.Height()
	if wv != 1 || hv != 1 {
		t.Errorf("dimensions = %dx%d, want 1x1", wv, hv)
	}
}

// --- Info queries ---

func TestIsTiled(t *testing.T) {
	dir := t.TempDir()
	stripPath := filepath.Join(dir, "strip.tif")
	createTestTIFF(t, stripPath, 32, 32)

	tif, err := Open(stripPath, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	if tif.IsTiled() {
		t.Error("strip image should not report as tiled")
	}
}

func TestDefaultStripSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "defaultstrip.tif")

	tif, err := Open(path, OpenWrite)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	tif.SetFieldUint32(TagImageWidth, 64)
	tif.SetFieldUint16(TagBitsPerSample, 8)
	tif.SetFieldUint16(TagSamplesPerPixel, 3)

	rps := tif.DefaultStripSize()
	if rps == 0 {
		t.Error("DefaultStripSize returned 0, expected non-zero")
	}
}

func TestIsByteSwapped(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "native.tif")
	createTestTIFF(t, path, 4, 4)

	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	// On a little-endian system with a little-endian TIFF, IsByteSwapped should be false.
	// We can't force the opposite, but we verify it doesn't panic and returns a bool.
	_ = tif.IsByteSwapped()
}

func TestFlush(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "flush.tif")

	tif, err := Open(path, OpenWrite)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	tif.SetFieldUint32(TagImageWidth, 1)
	tif.SetFieldUint32(TagImageLength, 1)
	tif.SetFieldUint16(TagBitsPerSample, 8)
	tif.SetFieldUint16(TagSamplesPerPixel, 1)
	tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
	tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
	tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))

	scanline := []byte{42}
	if err := tif.WriteScanline(scanline, 0); err != nil {
		t.Fatalf("WriteScanline: %v", err)
	}

	if err := tif.Flush(); err != nil {
		t.Errorf("Flush: %v", err)
	}
}

// --- Multi-page (multi-IFD) tests ---

const nDirectories = 10

func TestMultiPageNumberOfDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "multi.tif")
	writeMultiPageTIFF(t, path)

	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	count := tif.NumberOfDirectories()
	if count != nDirectories {
		t.Errorf("NumberOfDirectories = %d, want %d", count, nDirectories)
	}
}

func TestMultiPageSetDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "multi.tif")
	writeMultiPageTIFF(t, path)

	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	for i := range nDirectories {
		if err := tif.SetDirectory(uint32(i)); err != nil {
			t.Fatalf("SetDirectory(%d): %v", i, err)
		}
		if cur := tif.CurrentDirectory(); cur != uint32(i) {
			t.Errorf("After SetDirectory(%d), CurrentDirectory = %d", i, cur)
		}
		buf := make([]byte, 1)
		if err := tif.ReadScanline(buf, 0); err != nil {
			t.Fatalf("ReadScanline page %d: %v", i, err)
		}
		if buf[0] != byte(i) {
			t.Errorf("Page %d pixel = %d, want %d", i, buf[0], i)
		}
	}

	for i := nDirectories - 1; i > 0; i-- {
		if err := tif.SetDirectory(uint32(i)); err != nil {
			t.Fatalf("SetDirectory(%d) reverse: %v", i, err)
		}
		buf := make([]byte, 1)
		if err := tif.ReadScanline(buf, 0); err != nil {
			t.Fatalf("ReadScanline reverse page %d: %v", i, err)
		}
		if buf[0] != byte(i) {
			t.Errorf("Reverse page %d pixel = %d, want %d", i, buf[0], i)
		}
	}
}

func TestMultiPageReadDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "multi_read.tif")
	writeMultiPageTIFF(t, path)

	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	buf := make([]byte, 1)
	if err := tif.ReadScanline(buf, 0); err != nil {
		t.Fatalf("ReadScanline page 0: %v", err)
	}
	if buf[0] != 0 {
		t.Errorf("Page 0 pixel = %d, want 0", buf[0])
	}

	for page := 1; page < nDirectories; page++ {
		if !tif.ReadDirectory() {
			t.Fatalf("ReadDirectory() returned false at page %d", page)
		}
		if err := tif.ReadScanline(buf, 0); err != nil {
			t.Fatalf("ReadScanline page %d: %v", page, err)
		}
		if buf[0] != byte(page) {
			t.Errorf("Page %d pixel = %d, want %d", page, buf[0], page)
		}
	}

	if tif.ReadDirectory() {
		t.Error("ReadDirectory() should return false after last directory")
	}
}

func TestMultiPageLastDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lastdir.tif")
	writeMultiPageTIFF(t, path)

	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	if err := tif.SetDirectory(nDirectories - 1); err != nil {
		t.Fatalf("SetDirectory last: %v", err)
	}
	if !tif.LastDirectory() {
		t.Error("LastDirectory() should be true on last page")
	}

	if err := tif.SetDirectory(0); err != nil {
		t.Fatalf("SetDirectory 0: %v", err)
	}
	if tif.LastDirectory() {
		t.Error("LastDirectory() should be false on first page (multi-page TIFF)")
	}
}

func TestMultiPageSetDirectoryInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid_dir.tif")
	writeMultiPageTIFF(t, path)

	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	err = tif.SetDirectory(nDirectories + 10)
	if err == nil {
		t.Error("SetDirectory with out-of-range index should return error")
	}
}

func TestSubDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir.tif")

	func() {
		tif, err := Open(path, OpenWrite)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer tif.Close()

		tif.SetFieldUint32(TagImageWidth, 4)
		tif.SetFieldUint32(TagImageLength, 4)
		tif.SetFieldUint16(TagBitsPerSample, 8)
		tif.SetFieldUint16(TagSamplesPerPixel, 1)
		tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
		tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
		tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))

		data := make([]byte, 16)
		for i := range data {
			data[i] = byte(i)
		}
		if _, err := tif.WriteEncodedStrip(0, data); err != nil {
			t.Fatalf("WriteEncodedStrip main: %v", err)
		}
		if err := tif.WriteDirectory(); err != nil {
			t.Fatalf("WriteDirectory: %v", err)
		}

		// Second page (sub-IFD conceptually).
		tif.SetFieldUint32(TagImageWidth, 2)
		tif.SetFieldUint32(TagImageLength, 2)
		data2 := []byte{10, 20, 30, 40}
		if _, err := tif.WriteEncodedStrip(0, data2); err != nil {
			t.Fatalf("WriteEncodedStrip sub: %v", err)
		}
	}()

	tif2, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open read: %v", err)
	}
	defer tif2.Close()

	if tif2.NumberOfDirectories() != 2 {
		t.Errorf("NumberOfDirectories = %d, want 2", tif2.NumberOfDirectories())
	}
}

// --- BigTIFF ---

func TestBigTIFF(t *testing.T) {
	t.Run("w8_mode", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "bigtiff.tif")

		tif, err := Open(path, OpenBigTIFF)
		if err != nil {
			t.Fatalf("Open BigTIFF: %v", err)
		}

		tif.SetFieldUint32(TagImageWidth, 8)
		tif.SetFieldUint32(TagImageLength, 8)
		tif.SetFieldUint16(TagBitsPerSample, 8)
		tif.SetFieldUint16(TagSamplesPerPixel, 1)
		tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
		tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
		tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))

		scanline := make([]byte, 8)
		for row := uint32(0); row < 8; row++ {
			for i := range scanline {
				scanline[i] = byte(int(row)*10 + i)
			}
			if err := tif.WriteScanline(scanline, row); err != nil {
				t.Fatalf("WriteScanline %d: %v", row, err)
			}
		}
		tif.Close()

		tif2, err := Open(path, OpenRead)
		if err != nil {
			t.Fatalf("Open BigTIFF read: %v", err)
		}
		defer tif2.Close()

		if !tif2.IsBigTIFF() {
			t.Error("IsBigTIFF() = false, want true")
		}

		for row := uint32(0); row < 8; row++ {
			buf := make([]byte, 8)
			if err := tif2.ReadScanline(buf, row); err != nil {
				t.Fatalf("ReadScanline %d: %v", row, err)
			}
			for i := range buf {
				expected := byte(int(row)*10 + i)
				if buf[i] != expected {
					t.Errorf("row %d col %d = %d, want %d", row, i, buf[i], expected)
				}
			}
		}
	})

	t.Run("constant_mode", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "big_test.tiff")

		tif, err := Open(path, OpenBigTIFF)
		if err != nil {
			t.Fatalf("Open BigTIFF: %v", err)
		}

		w, h := uint32(100), uint32(100)
		tif.SetFieldUint32(TagImageWidth, w)
		tif.SetFieldUint32(TagImageLength, h)
		tif.SetFieldUint16(TagBitsPerSample, 8)
		tif.SetFieldUint16(TagSamplesPerPixel, 1)
		tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
		tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
		tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))

		for row := range h {
			scanline := make([]byte, w)
			if err := tif.WriteScanline(scanline, row); err != nil {
				t.Fatalf("WriteScanline %d: %v", row, err)
			}
		}
		tif.Close()

		readTif, err := Open(path, OpenRead)
		if err != nil {
			t.Fatalf("Open for read: %v", err)
		}
		defer readTif.Close()

		rw, _ := readTif.Width()
		rh, _ := readTif.Height()
		if rw != w || rh != h {
			t.Errorf("dimensions = %dx%d, want %dx%d", rw, rh, w, h)
		}
	})
}

func TestBigTIFFMultiPage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bigtiff_multi.tif")

	const pages = 5

	func() {
		tif, err := Open(path, OpenBigTIFF)
		if err != nil {
			t.Fatalf("Open BigTIFF: %v", err)
		}
		defer tif.Close()

		for p := range pages {
			tif.SetFieldUint32(TagImageWidth, 4)
			tif.SetFieldUint32(TagImageLength, 4)
			tif.SetFieldUint16(TagBitsPerSample, 8)
			tif.SetFieldUint16(TagSamplesPerPixel, 1)
			tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
			tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
			tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))

			data := make([]byte, 16)
			for i := range data {
				data[i] = byte(p*16 + i)
			}
			if _, err := tif.WriteEncodedStrip(0, data); err != nil {
				t.Fatalf("WriteEncodedStrip page %d: %v", p, err)
			}
			if p < pages-1 {
				if err := tif.WriteDirectory(); err != nil {
					t.Fatalf("WriteDirectory page %d: %v", p, err)
				}
			}
		}
	}()

	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open read: %v", err)
	}
	defer tif.Close()

	if !tif.IsBigTIFF() {
		t.Error("IsBigTIFF() = false, want true")
	}
	if tif.NumberOfDirectories() != pages {
		t.Errorf("NumberOfDirectories = %d, want %d", tif.NumberOfDirectories(), pages)
	}

	for p := uint32(0); p < pages; p++ {
		if err := tif.SetDirectory(p); err != nil {
			t.Fatalf("SetDirectory(%d): %v", p, err)
		}
		buf := make([]byte, 16)
		n, err := tif.ReadEncodedStrip(0, buf, -1)
		if err != nil {
			t.Fatalf("ReadEncodedStrip page %d: %v", p, err)
		}
		if n != 16 {
			t.Errorf("Page %d: read %d bytes, want 16", p, n)
		}
		for i := range buf {
			expected := byte(int(p)*16 + i)
			if buf[i] != expected {
				t.Errorf("Page %d byte %d = %d, want %d", p, i, buf[i], expected)
				break
			}
		}
	}
}

func TestCompressWithPredictor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "predictor.tif")

	w, h := uint32(64), uint32(64)
	tif, err := Open(path, OpenWrite)
	if err != nil {
		t.Fatalf("Open write: %v", err)
	}

	tif.SetFieldUint32(TagImageWidth, w)
	tif.SetFieldUint32(TagImageLength, h)
	tif.SetFieldUint16(TagBitsPerSample, 8)
	tif.SetFieldUint16(TagSamplesPerPixel, 1)
	tif.SetFieldUint16(TagCompression, uint16(CompressionDeflate))
	tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
	tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))
	tif.SetFieldUint16(TagPredictor, uint16(PredictorHorizontal))

	for row := uint32(0); row < h; row++ {
		scanline := make([]byte, w)
		for i := range scanline {
			scanline[i] = byte((row + uint32(i)) % 256)
		}
		if err := tif.WriteScanline(scanline, row); err != nil {
			t.Fatalf("WriteScanline %d: %v", row, err)
		}
	}
	tif.Close()

	readTif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open read: %v", err)
	}
	defer readTif.Close()

	comp, _ := readTif.Compression()
	if comp != uint16(CompressionDeflate) {
		t.Errorf("Compression = %d, want %d", comp, CompressionDeflate)
	}

	pred, _ := readTif.Predictor()
	if pred != uint16(PredictorHorizontal) {
		t.Errorf("Predictor = %d, want %d", pred, PredictorHorizontal)
	}

	scanline := make([]byte, readTif.ScanlineSize())
	for row := uint32(0); row < h; row++ {
		if err := readTif.ReadScanline(scanline, row); err != nil {
			t.Fatalf("ReadScanline %d: %v", row, err)
		}
		for i := range scanline {
			expected := byte((row + uint32(i)) % 256)
			if scanline[i] != expected {
				t.Errorf("row %d col %d: got %d, want %d", row, i, scanline[i], expected)
				break
			}
		}
	}
}


func TestMultiStripImage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "multistrip.tif")

	w, h := uint32(16), uint32(100)
	rowsPerStrip := uint32(10)

	tif, err := Open(path, OpenWrite)
	if err != nil {
		t.Fatalf("Open write: %v", err)
	}

	tif.SetFieldUint32(TagImageWidth, w)
	tif.SetFieldUint32(TagImageLength, h)
	tif.SetFieldUint16(TagBitsPerSample, 8)
	tif.SetFieldUint16(TagSamplesPerPixel, 1)
	tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
	tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
	tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))
	tif.SetFieldUint32(TagRowsPerStrip, rowsPerStrip)

	for row := uint32(0); row < h; row++ {
		scanline := make([]byte, w)
		for i := range scanline {
			scanline[i] = byte(row % 256)
		}
		if err := tif.WriteScanline(scanline, row); err != nil {
			t.Fatalf("WriteScanline %d: %v", row, err)
		}
	}
	tif.Close()

	readTif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open read: %v", err)
	}
	defer readTif.Close()

	numStrips := readTif.NumberOfStrips()
	expectedStrips := (h + rowsPerStrip - 1) / rowsPerStrip
	if numStrips != expectedStrips {
		t.Errorf("NumberOfStrips = %d, want %d", numStrips, expectedStrips)
	}

	rps, _ := readTif.RowsPerStrip()
	if rps != rowsPerStrip {
		t.Errorf("RowsPerStrip = %d, want %d", rps, rowsPerStrip)
	}

	buf := make([]byte, readTif.StripSize())
	for strip := uint32(0); strip < numStrips; strip++ {
		n, err := readTif.ReadEncodedStrip(strip, buf, -1)
		if err != nil {
			t.Fatalf("ReadEncodedStrip %d: %v", strip, err)
		}
		if n <= 0 {
			t.Errorf("ReadEncodedStrip %d returned %d bytes", strip, n)
		}
	}
}


func TestEdgeDimensions1x1(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "1x1.tif")

	tif, err := Open(path, OpenWrite)
	if err != nil {
		t.Fatalf("Open write: %v", err)
	}

	tif.SetFieldUint32(TagImageWidth, 1)
	tif.SetFieldUint32(TagImageLength, 1)
	tif.SetFieldUint16(TagBitsPerSample, 8)
	tif.SetFieldUint16(TagSamplesPerPixel, 1)
	tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
	tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
	tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))

	if err := tif.WriteScanline([]byte{42}, 0); err != nil {
		t.Fatalf("WriteScanline: %v", err)
	}
	tif.Close()

	readTif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open read: %v", err)
	}
	defer readTif.Close()

	w, _ := readTif.Width()
	h, _ := readTif.Height()
	if w != 1 || h != 1 {
		t.Errorf("dimensions = %dx%d, want 1x1", w, h)
	}

	scanline := make([]byte, readTif.ScanlineSize())
	if err := readTif.ReadScanline(scanline, 0); err != nil {
		t.Fatalf("ReadScanline: %v", err)
	}
	if scanline[0] != 42 {
		t.Errorf("pixel = %d, want 42", scanline[0])
	}
}


func TestTileReadWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tiled.tif")

	tileW, tileH := uint32(16), uint32(16)
	imgW, imgH := uint32(32), uint32(32)

	tif, err := Open(path, OpenWrite)
	if err != nil {
		t.Fatalf("Open write: %v", err)
	}

	tif.SetFieldUint32(TagImageWidth, imgW)
	tif.SetFieldUint32(TagImageLength, imgH)
	tif.SetFieldUint16(TagBitsPerSample, 8)
	tif.SetFieldUint16(TagSamplesPerPixel, 1)
	tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
	tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
	tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))
	tif.SetFieldUint32(TagTileWidth, tileW)
	tif.SetFieldUint32(TagTileLength, tileH)

	tileSize := int(tileW * tileH)
	tileData := make([]byte, tileSize)

	for tile := uint32(0); tile < tif.NumberOfTiles(); tile++ {
		for i := range tileData {
			tileData[i] = byte((tile + uint32(i)) % 256)
		}
		if _, err := tif.WriteEncodedTile(tile, tileData); err != nil {
			t.Fatalf("WriteEncodedTile %d: %v", tile, err)
		}
	}
	tif.Close()

	readTif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open read: %v", err)
	}
	defer readTif.Close()

	if !readTif.IsTiled() {
		t.Fatal("expected tiled image")
	}

	rtw, _ := readTif.TileWidth()
	rth, _ := readTif.TileLength()
	if rtw != tileW || rth != tileH {
		t.Errorf("tile dimensions = %dx%d, want %dx%d", rtw, rth, tileW, tileH)
	}

	readBuf := make([]byte, readTif.TileSize())
	for tile := uint32(0); tile < readTif.NumberOfTiles(); tile++ {
		n, err := readTif.ReadEncodedTile(tile, readBuf, -1)
		if err != nil {
			t.Fatalf("ReadEncodedTile %d: %v", tile, err)
		}
		if n != tileSize {
			t.Errorf("ReadEncodedTile %d: got %d bytes, want %d", tile, n, tileSize)
		}
		expected := byte(tile % 256)
		if readBuf[0] != expected {
			t.Errorf("tile %d: first byte = %d, want %d", tile, readBuf[0], expected)
		}
	}
}


func TestReadRGBAStrip(t *testing.T) {
	t.Parallel()
	path := filepath.Join("testdata", "rgb-3c-8b.tiff")
	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	if tif.IsTiled() {
		t.Skip("test file is tiled, not stripped")
	}
	w, _ := tif.Width()
	rps, _ := tif.RowsPerStrip()
	if rps == 0 {
		t.Skip("cannot determine rows per strip")
	}

	tests := []struct {
		name string
		fn   func(strip uint32, buf []uint32) error
	}{
		{"Basic", func(strip uint32, buf []uint32) error { return tif.ReadRGBAStrip(strip, buf) }},
		{"Ext", func(strip uint32, buf []uint32) error { return tif.ReadRGBAStripExt(strip, buf, true) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buf := make([]uint32, int(w*rps))
			if err := tc.fn(0, buf); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			hasNonZero := false
			for _, px := range buf {
				if px != 0 {
					hasNonZero = true
					break
				}
			}
			if !hasNonZero {
				t.Error("all RGBA pixels are zero, expected non-zero data")
			}
		})
	}
}


func TestReadRGBATile(t *testing.T) {
	t.Parallel()
	path := filepath.Join("testdata", "quad-tile.jpg.tiff")
	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	tw, _ := tif.TileWidth()
	th, _ := tif.TileLength()

	tests := []struct {
		name string
		fn   func(tile uint32, buf []uint32) error
	}{
		{"Basic", func(tile uint32, buf []uint32) error { return tif.ReadRGBATile(tile, buf) }},
		{"Ext", func(tile uint32, buf []uint32) error { return tif.ReadRGBATileExt(tile, buf, true) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buf := make([]uint32, int(tw*th))
			if err := tc.fn(0, buf); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			hasNonZero := false
			for _, px := range buf {
				if px != 0 {
					hasNonZero = true
					break
				}
			}
			if !hasNonZero {
				t.Error("all RGBA tile pixels are zero")
			}
		})
	}
}


// --- RGBA image ---

func TestRGBAImage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rgba.tif")

	const w, h = 16, 16
	func() {
		tif, err := Open(path, OpenWrite)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer tif.Close()

		tif.SetFieldUint32(TagImageWidth, w)
		tif.SetFieldUint32(TagImageLength, h)
		tif.SetFieldUint16(TagBitsPerSample, 8)
		tif.SetFieldUint16(TagSamplesPerPixel, 3)
		tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
		tif.SetFieldUint16(TagPhotometric, uint16(PhotometricRGB))
		tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))

		scanline := make([]byte, w*3)
		for row := uint32(0); row < h; row++ {
			for col := uint32(0); col < w; col++ {
				scanline[col*3+0] = byte(row * 16)
				scanline[col*3+1] = byte(col * 16)
				scanline[col*3+2] = 128
			}
			if err := tif.WriteScanline(scanline, row); err != nil {
				t.Fatalf("WriteScanline %d: %v", row, err)
			}
		}
	}()

	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open read: %v", err)
	}
	defer tif.Close()

	if err := tif.RGBAImageOK(); err != nil {
		t.Fatalf("RGBAImageOK: %v", err)
	}

	buf := make([]uint32, w*h)
	if err := tif.ReadRGBAImage(buf); err != nil {
		t.Fatalf("ReadRGBAImage: %v", err)
	}

	// Not all pixels can be zero: R=0,G=0,B=128 at row=0,col=0 is non-black.
	zeroCount := 0
	for _, v := range buf {
		if v == 0 {
			zeroCount++
		}
	}
	if zeroCount == w*h {
		t.Error("ReadRGBAImage returned all zeros")
	}
}

// --- EXIF Sub-IFD ---

func TestEXIFSubIFD(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exif_test.tif")

	const w, h uint32 = 8, 8

	// --- Write phase ---
	func() {
		tif, err := Open(path, OpenWrite)
		if err != nil {
			t.Fatalf("Open write: %v", err)
		}
		defer tif.Close()

		// IFD0 tags
		tif.SetFieldUint32(TagImageWidth, w)
		tif.SetFieldUint32(TagImageLength, h)
		tif.SetFieldUint16(TagBitsPerSample, 8)
		tif.SetFieldUint16(TagSamplesPerPixel, 3)
		tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
		tif.SetFieldUint16(TagPhotometric, uint16(PhotometricRGB))
		tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))
		tif.SetFieldString(TagMake, "TestMake")
		tif.SetFieldString(TagModel, "TestModel")

		// Write pixel data
		scanline := make([]byte, w*3)
		for row := uint32(0); row < h; row++ {
			for i := range scanline {
				scanline[i] = byte(row % 256)
			}
			if err := tif.WriteScanline(scanline, row); err != nil {
				t.Fatalf("WriteScanline %d: %v", row, err)
			}
		}

		// Write main IFD before creating EXIF sub-IFD
		if err := tif.WriteDirectory(); err != nil {
			t.Fatalf("WriteDirectory: %v", err)
		}
		if err := tif.SetDirectory(0); err != nil {
			t.Fatalf("SetDirectory(0): %v", err)
		}

		// Create EXIF sub-IFD
		if err := tif.CreateEXIFDirectory(); err != nil {
			t.Fatalf("CreateEXIFDirectory: %v", err)
		}

		// Write scalar EXIF tags (RATIONAL, SETGET_UINT16, SETGET_ASCII)
		if err := tif.SetFieldFloat(TagExifExposureTime, 0.025); err != nil {
			t.Errorf("SetFieldFloat ExposureTime: %v", err)
		}
		if err := tif.SetFieldFloat(TagExifFNumber, 5.6); err != nil {
			t.Errorf("SetFieldFloat FNumber: %v", err)
		}
		if err := tif.SetFieldUint16(TagExifExposureProgram, 1); err != nil {
			t.Errorf("SetFieldUint16 ExposureProgram: %v", err)
		}
		// Zero-value uint16 tags (Flash=0, CustomRendered=0, Sharpness=0)
		if err := tif.SetFieldUint16(TagExifFlash, 0); err != nil {
			t.Errorf("SetFieldUint16 Flash=0: %v", err)
		}
		if err := tif.SetFieldUint16(TagExifCustomRendered, 0); err != nil {
			t.Errorf("SetFieldUint16 CustomRendered=0: %v", err)
		}
		if err := tif.SetFieldUint16(TagExifSharpness, 0); err != nil {
			t.Errorf("SetFieldUint16 Sharpness=0: %v", err)
		}
		// Zero-value float (ExposureCompensation=0)
		if err := tif.SetFieldFloat(TagExifExposureCompensation, 0); err != nil {
			t.Errorf("SetFieldFloat ExposureCompensation=0: %v", err)
		}
		if err := tif.SetFieldFloat(TagExifFocalLength, 35.0); err != nil {
			t.Errorf("SetFieldFloat FocalLength: %v", err)
		}
		if err := tif.SetFieldString(TagExifDateTimeOriginal, "2025:05:16 15:12:13"); err != nil {
			t.Errorf("SetFieldString DateTimeOriginal: %v", err)
		}

		// C16 uint16 array (ISO: SETGET_C16_UINT16)
		if err := tif.SetFieldUint16Slice(TagExifISO, []uint16{100}); err != nil {
			t.Errorf("SetFieldUint16Slice ISO: %v", err)
		}

		// UINT8 single byte (SceneType: SETGET_UINT8)
		if err := tif.SetFieldUint8(TagExifSceneType, 1); err != nil {
			t.Errorf("SetFieldUint8 SceneType: %v", err)
		}

		// C0 float array (LensSpecification: SETGET_C0_FLOAT, 4 floats)
		if err := tif.SetFieldC0FloatSlice(TagExifLensInfo, []float64{24.0, 70.0, 4.0, 4.0}); err != nil {
			t.Errorf("SetFieldC0FloatSlice LensInfo: %v", err)
		}

		// MakerNote (SETGET_C16_UINT8)
		makerNote := []byte{0x54, 0x45, 0x53, 0x54} // "TEST"
		if err := tif.SetFieldByteSlice(TagExifMakerNote, makerNote); err != nil {
			t.Errorf("SetFieldByteSlice MakerNote: %v", err)
		}

		// Write custom directory and get offset
		exifOffset, err := tif.WriteCustomDirectory()
		if err != nil {
			t.Fatalf("WriteCustomDirectory: %v", err)
		}

		// Return to main IFD and set EXIFIFD pointer
		if err := tif.SetDirectory(0); err != nil {
			t.Fatalf("SetDirectory(0): %v", err)
		}
		if err := tif.SetFieldUint64(TagEXIFIFD, exifOffset); err != nil {
			t.Fatalf("SetFieldUint64 EXIFIFD: %v", err)
		}
		if err := tif.WriteDirectory(); err != nil {
			t.Fatalf("WriteDirectory: %v", err)
		}
	}()

	// --- Read phase ---
	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open read: %v", err)
	}
	defer tif.Close()

	// Verify IFD0 tags
	if make, _ := tif.GetFieldString(TagMake); make != "TestMake" {
		t.Errorf("Make = %q, want %q", make, "TestMake")
	}
	if model, _ := tif.GetFieldString(TagModel); model != "TestModel" {
		t.Errorf("Model = %q, want %q", model, "TestModel")
	}

	// Navigate to EXIF Sub-IFD
	exifOffset, err := tif.GetFieldUint64(TagEXIFIFD)
	if err != nil {
		t.Fatalf("GetFieldUint64 EXIFIFD: %v", err)
	}
	if exifOffset == 0 {
		t.Fatal("EXIFIFD offset is 0, EXIF Sub-IFD not linked")
	}

	if err := tif.ReadEXIFDirectory(exifOffset); err != nil {
		t.Fatalf("ReadEXIFDirectory: %v", err)
	}

	// Verify RATIONAL EXIF tags (stored as float by libtiff)
	if et, err := tif.GetFieldFloat(TagExifExposureTime); err != nil || math.Abs(et-0.025) > 1e-6 {
		t.Errorf("ExposureTime = %v (err=%v), want ~0.025", et, err)
	}
	if fn, err := tif.GetFieldFloat(TagExifFNumber); err != nil || math.Abs(fn-5.6) > 0.01 {
		t.Errorf("FNumber = %v (err=%v), want ~5.6", fn, err)
	}
	if fl, err := tif.GetFieldFloat(TagExifFocalLength); err != nil || math.Abs(fl-35.0) > 0.01 {
		t.Errorf("FocalLength = %v (err=%v), want ~35.0", fl, err)
	}

	// Verify scalar uint16 tags (including zero values)
	if ep, err := tif.GetFieldUint16(TagExifExposureProgram); err != nil || ep != 1 {
		t.Errorf("ExposureProgram = %d (err=%v), want 1", ep, err)
	}
	if flash, err := tif.GetFieldUint16(TagExifFlash); err != nil || flash != 0 {
		t.Errorf("Flash = %d (err=%v), want 0", flash, err)
	}
	if cr, err := tif.GetFieldUint16(TagExifCustomRendered); err != nil || cr != 0 {
		t.Errorf("CustomRendered = %d (err=%v), want 0", cr, err)
	}
	if sh, err := tif.GetFieldUint16(TagExifSharpness); err != nil || sh != 0 {
		t.Errorf("Sharpness = %d (err=%v), want 0", sh, err)
	}

	// Verify zero-value float (RATIONAL precision)
	if ec, err := tif.GetFieldFloat(TagExifExposureCompensation); err != nil || math.Abs(ec) > 1e-6 {
		t.Errorf("ExposureCompensation = %v (err=%v), want ~0", ec, err)
	}

	// Verify string tag
	if dt, err := tif.GetFieldString(TagExifDateTimeOriginal); err != nil || dt != "2025:05:16 15:12:13" {
		t.Errorf("DateTimeOriginal = %q (err=%v), want %q", dt, err, "2025:05:16 15:12:13")
	}

	// Verify ISO (C16 uint16)
	if isoSlice, err := tif.GetFieldUint16Slice(TagExifISO); err != nil || len(isoSlice) != 1 || isoSlice[0] != 100 {
		t.Errorf("ISO = %v (err=%v), want [100]", isoSlice, err)
	}

	// Verify SceneType (uint8)
	if st, err := tif.GetFieldUint8(TagExifSceneType); err != nil || st != 1 {
		t.Errorf("SceneType = %d (err=%v), want 1", st, err)
	}
}

// --- GPS Sub-IFD ---

func TestGPSSubIFD(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gps_test.tif")

	const w, h uint32 = 4, 4

	// --- Write phase (following libtiff official custom_dir_EXIF_231.c flow) ---
	func() {
		tif, err := Open(path, OpenWrite)
		if err != nil {
			t.Fatalf("Open write: %v", err)
		}
		defer tif.Close()

		// IFD0 tags
		tif.SetFieldUint32(TagImageWidth, w)
		tif.SetFieldUint32(TagImageLength, h)
		tif.SetFieldUint16(TagBitsPerSample, 8)
		tif.SetFieldUint16(TagSamplesPerPixel, 3)
		tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
		tif.SetFieldUint16(TagPhotometric, uint16(PhotometricRGB))
		tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))

		// Reserve dummy GPS IFD pointer (per libtiff docs)
		tif.SetFieldUint64(TagGPSIFD, 0)

		// Write pixel data
		scanline := make([]byte, w*3)
		for row := uint32(0); row < h; row++ {
			for i := range scanline {
				scanline[i] = byte(row * 50)
			}
			if err := tif.WriteScanline(scanline, row); err != nil {
				t.Fatalf("WriteScanline %d: %v", row, err)
			}
		}

		// Save main IFD to file
		if err := tif.WriteDirectory(); err != nil {
			t.Fatalf("WriteDirectory IFD0: %v", err)
		}

		// Reload IFD0 and create GPS directory
		if err := tif.SetDirectory(0); err != nil {
			t.Fatalf("SetDirectory(0): %v", err)
		}
		if err := tif.CreateGPSDirectory(); err != nil {
			t.Fatalf("CreateGPSDirectory: %v", err)
		}

		// Write GPS tags (GPSTAG IDs: 0=VersionID, 1=LatitudeRef, 2=Latitude)
		gpsVersion := []byte{2, 2, 0, 1}
		if err := tif.SetFieldC0ByteSlice(0, gpsVersion); err != nil {
			t.Fatalf("SetField GPSVersionID: %v", err)
		}
		if err := tif.SetFieldString(1, "N"); err != nil {
			t.Fatalf("SetField LatitudeRef: %v", err)
		}
		// Latitude: RATIONAL[3], SETGET_C0_DOUBLE
		if err := tif.SetFieldC0DoubleSlice(2, []float64{30.0, 15.0, 0.0}); err != nil {
			t.Fatalf("SetField Latitude: %v", err)
		}

		// Write GPS custom directory
		gpsOffset, err := tif.WriteCustomDirectory()
		if err != nil {
			t.Fatalf("WriteCustomDirectory GPS: %v", err)
		}
		t.Logf("GPS Sub-IFD offset: %d", gpsOffset)

		// Return to main IFD and set GPS pointer
		if err := tif.SetDirectory(0); err != nil {
			t.Fatalf("SetDirectory(0) after GPS: %v", err)
		}
		if err := tif.SetFieldUint64(TagGPSIFD, gpsOffset); err != nil {
			t.Fatalf("SetField GPSIFD: %v", err)
		}
		if err := tif.WriteDirectory(); err != nil {
			t.Fatalf("WriteDirectory with GPS pointer: %v", err)
		}
	}()

	// --- Read phase ---
	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open read: %v", err)
	}
	defer tif.Close()

	wv, _ := tif.Width()
	hv, _ := tif.Height()
	if wv != w || hv != h {
		t.Errorf("dimensions = %dx%d, want %dx%d", wv, hv, w, h)
	}

	gpsOffset, err := tif.GetFieldUint64(TagGPSIFD)
	if err != nil {
		t.Fatalf("GetFieldUint64 GPSIFD: %v", err)
	}
	if gpsOffset == 0 {
		t.Fatal("GPSIFD offset is 0, GPS Sub-IFD not linked")
	}
	t.Logf("Read back GPS IFD offset: %d", gpsOffset)
}

// --- Fixture files (table-driven) ---

func TestFixtures(t *testing.T) {
	tests := []struct {
		name  string
		file  string
		check func(t *testing.T, tf *TIFF)
	}{
		{
			"Grayscale8bit", "minisblack-1c-8b.tiff",
			func(t *testing.T, tf *TIFF) {
				wv, _ := tf.Width()
				hv, _ := tf.Height()
				if wv == 0 || hv == 0 {
					t.Error("expected non-zero dimensions")
				}
				if v, err := tf.BitsPerSample(); err != nil {
					t.Fatalf("BitsPerSample: %v", err)
				} else if v != 8 {
					t.Errorf("BitsPerSample = %d, want 8", v)
				}
				if v, err := tf.SamplesPerPixel(); err != nil {
					t.Fatalf("SamplesPerPixel: %v", err)
				} else if v != 1 {
					t.Errorf("SamplesPerPixel = %d, want 1", v)
				}
				if v, err := tf.Photometric(); err != nil {
					t.Fatalf("Photometric: %v", err)
				} else if v != uint16(PhotometricMinIsBlack) {
					t.Errorf("Photometric = %d, want %d", v, PhotometricMinIsBlack)
				}
				if tf.IsTiled() {
					t.Error("strip-based image should not report as tiled")
				}
				buf := make([]byte, tf.ScanlineSize())
				if err := tf.ReadScanline(buf, 0); err != nil {
					t.Fatalf("ReadScanline 0: %v", err)
				}
			},
		},
		{
			"Grayscale16bit", "minisblack-1c-16b.tiff",
			func(t *testing.T, tf *TIFF) {
				if v, err := tf.BitsPerSample(); err != nil {
					t.Fatalf("BitsPerSample: %v", err)
				} else if v != 16 {
					t.Errorf("BitsPerSample = %d, want 16", v)
				}
				wv, _ := tf.Width()
				expected := int(wv) * 2
				if tf.ScanlineSize() != expected {
					t.Errorf("ScanlineSize = %d, want %d", tf.ScanlineSize(), expected)
				}
				buf := make([]byte, tf.ScanlineSize())
				if err := tf.ReadScanline(buf, 0); err != nil {
					t.Fatalf("ReadScanline 0: %v", err)
				}
			},
		},
		{
			"RGB8bit", "rgb-3c-8b.tiff",
			func(t *testing.T, tf *TIFF) {
				if v, err := tf.SamplesPerPixel(); err != nil {
					t.Fatalf("SamplesPerPixel: %v", err)
				} else if v != 3 {
					t.Errorf("SamplesPerPixel = %d, want 3", v)
				}
				if v, err := tf.Photometric(); err != nil {
					t.Fatalf("Photometric: %v", err)
				} else if v != uint16(PhotometricRGB) {
					t.Errorf("Photometric = %d, want %d (RGB)", v, PhotometricRGB)
				}
				if v, err := tf.BitsPerSample(); err != nil {
					t.Fatalf("BitsPerSample: %v", err)
				} else if v != 8 {
					t.Errorf("BitsPerSample = %d, want 8", v)
				}
				buf := make([]byte, tf.ScanlineSize())
				if err := tf.ReadScanline(buf, 0); err != nil {
					t.Fatalf("ReadScanline 0: %v", err)
				}
			},
		},
		{
			"RGB16bit", "rgb-3c-16b.tiff",
			func(t *testing.T, tf *TIFF) {
				if v, err := tf.BitsPerSample(); err != nil {
					t.Fatalf("BitsPerSample: %v", err)
				} else if v != 16 {
					t.Errorf("BitsPerSample = %d, want 16", v)
				}
				if v, err := tf.SamplesPerPixel(); err != nil {
					t.Fatalf("SamplesPerPixel: %v", err)
				} else if v != 3 {
					t.Errorf("SamplesPerPixel = %d, want 3", v)
				}
				wv, _ := tf.Width()
				expected := int(wv) * 3 * 2
				if tf.ScanlineSize() != expected {
					t.Errorf("ScanlineSize = %d, want %d", tf.ScanlineSize(), expected)
				}
			},
		},
		{
			"Bilevel", "miniswhite-1c-1b.tiff",
			func(t *testing.T, tf *TIFF) {
				if v, err := tf.BitsPerSample(); err != nil {
					t.Fatalf("BitsPerSample: %v", err)
				} else if v != 1 {
					t.Errorf("BitsPerSample = %d, want 1", v)
				}
				if v, err := tf.Photometric(); err != nil {
					t.Fatalf("Photometric: %v", err)
				} else if v != uint16(PhotometricMinIsWhite) {
					t.Errorf("Photometric = %d, want %d", v, PhotometricMinIsWhite)
				}
			},
		},
		{
			"Palette", "palette-1c-8b.tiff",
			func(t *testing.T, tf *TIFF) {
				if v, err := tf.Photometric(); err != nil {
					t.Fatalf("Photometric: %v", err)
				} else if v != uint16(PhotometricPalette) {
					t.Errorf("Photometric = %d, want %d (Palette)", v, PhotometricPalette)
				}
				if v, err := tf.BitsPerSample(); err != nil {
					t.Fatalf("BitsPerSample: %v", err)
				} else if v != 8 {
					t.Errorf("BitsPerSample = %d, want 8", v)
				}
			},
		},
		{
			"LZWSingleStrip", "lzw-single-strip.tiff",
			func(t *testing.T, tf *TIFF) {
				if v, err := tf.Compression(); err != nil {
					t.Fatalf("Compression: %v", err)
				} else if v != uint16(CompressionLZW) {
					t.Errorf("Compression = %d, want %d (LZW)", v, CompressionLZW)
				}
				stripSize := tf.StripSize()
				buf := make([]byte, stripSize)
				for strip := uint32(0); strip < tf.NumberOfStrips(); strip++ {
					n, err := tf.ReadEncodedStrip(strip, buf, -1)
					if err != nil {
						t.Fatalf("ReadEncodedStrip %d: %v", strip, err)
					}
					if n <= 0 {
						t.Errorf("Strip %d: read %d bytes", strip, n)
					}
				}
			},
		},
		{
			"LZWCompat", "quad-lzw-compat.tiff",
			func(t *testing.T, tf *TIFF) {
				if v, err := tf.Compression(); err != nil {
					t.Fatalf("Compression: %v", err)
				} else if v != uint16(CompressionLZW) {
					t.Errorf("Compression = %d, want %d", v, CompressionLZW)
				}
				buf := make([]byte, tf.StripSize())
				for strip := uint32(0); strip < tf.NumberOfStrips(); strip++ {
					if _, err := tf.ReadEncodedStrip(strip, buf, -1); err != nil {
						t.Fatalf("ReadEncodedStrip %d: %v", strip, err)
					}
				}
			},
		},
		{
			"Tiled", "quad-tile.jpg.tiff",
			func(t *testing.T, tf *TIFF) {
				if !tf.IsTiled() {
					t.Fatal("expected tiled image")
				}
				tw, _ := tf.TileWidth()
				tl, _ := tf.TileLength()
				if tw == 0 || tl == 0 {
					t.Errorf("TileWidth=%d, TileLength=%d, expected non-zero", tw, tl)
				}
				tileSize := tf.TileSize()
				if tileSize <= 0 {
					t.Errorf("TileSize = %d, expected > 0", tileSize)
				}
				numTiles := tf.NumberOfTiles()
				if numTiles == 0 {
					t.Error("NumberOfTiles = 0, expected > 0")
				}
			},
		},
		{
			"AlphaChannel", "minisblack-2c-8b-alpha.tiff",
			func(t *testing.T, tf *TIFF) {
				if v, err := tf.SamplesPerPixel(); err != nil {
					t.Fatalf("SamplesPerPixel: %v", err)
				} else if v != 2 {
					t.Errorf("SamplesPerPixel = %d, want 2 (grayscale + alpha)", v)
				}
				if v, err := tf.BitsPerSample(); err != nil {
					t.Fatalf("BitsPerSample: %v", err)
				} else if v != 8 {
					t.Errorf("BitsPerSample = %d, want 8", v)
				}
			},
		},
		{
			"TwoIFDs", "test_two_ifds.tif",
			func(t *testing.T, tf *TIFF) {
				count := tf.NumberOfDirectories()
				if count != 2 {
					t.Fatalf("NumberOfDirectories = %d, want 2", count)
				}
				if !tf.ReadDirectory() {
					t.Fatal("ReadDirectory() returned false, expected second IFD")
				}
				if tf.CurrentDirectory() != 1 {
					t.Errorf("CurrentDirectory = %d, want 1", tf.CurrentDirectory())
				}
				if tf.ReadDirectory() {
					t.Error("ReadDirectory() should return false after second IFD")
				}
				if err := tf.SetDirectory(0); err != nil {
					t.Fatalf("SetDirectory(0): %v", err)
				}
				if tf.CurrentDirectory() != 0 {
					t.Errorf("CurrentDirectory = %d, want 0", tf.CurrentDirectory())
				}
			},
		},
		{
			"RGBARead", "rgb-3c-8b.tiff",
			func(t *testing.T, tf *TIFF) {
				if err := tf.RGBAImageOK(); err != nil {
					t.Fatalf("RGBAImageOK: %v", err)
				}
				w, _ := tf.Width()
				h, _ := tf.Height()
				buf := make([]uint32, w*h)
				if err := tf.ReadRGBAImage(buf); err != nil {
					t.Fatalf("ReadRGBAImage: %v", err)
				}
				if buf[0] == 0 {
					t.Error("first RGBA pixel is all-zero, expected color data")
				}
			},
		},
		{
			"32bppNone", "32bpp-None.tiff",
			func(t *testing.T, tf *TIFF) {
				// 32bpp-None is actually 8-bit per channel RGBA (4 samples * 8 bits = 32bpp).
				if v, err := tf.BitsPerSample(); err != nil {
					t.Fatalf("BitsPerSample: %v", err)
				} else if v != 8 {
					t.Errorf("BitsPerSample = %d, want 8", v)
				}
				if v, err := tf.SamplesPerPixel(); err != nil {
					t.Fatalf("SamplesPerPixel: %v", err)
				} else if v != 4 {
					t.Errorf("SamplesPerPixel = %d, want 4", v)
				}
			},
		},
		{
			"DeflateLastStrip", "deflate-last-strip-extra-data.tiff",
			func(t *testing.T, tf *TIFF) {
				if v, err := tf.Compression(); err != nil {
					t.Fatalf("Compression: %v", err)
				} else if v != uint16(CompressionDeflate) {
					t.Errorf("Compression = %d, want %d", v, CompressionDeflate)
				}
				buf := make([]byte, tf.StripSize())
				for strip := uint32(0); strip < tf.NumberOfStrips(); strip++ {
					if _, err := tf.ReadEncodedStrip(strip, buf, -1); err != nil {
						t.Fatalf("ReadEncodedStrip %d: %v", strip, err)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tf := openFixture(t, tt.file)
			defer tf.Close()
			tt.check(t, tf)
		})
	}
}

// --- Tests for API improvement changes ---

func TestErrClosedSentinel(t *testing.T) {
	tif, err := Open(filepath.Join(t.TempDir(), "test.tif"), OpenWrite)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	tif.Close()

	w, err := tif.Width()
	if !errors.Is(err, ErrClosed) {
		t.Errorf("Width after Close: err = %v, want errors.Is(err, ErrClosed)", err)
	}
	if w != 0 {
		t.Errorf("Width after Close: w = %d, want 0", w)
	}
}

func TestIsOpen(t *testing.T) {
	tif, err := Open(filepath.Join(t.TempDir(), "test.tif"), OpenWrite)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if !tif.IsOpen() {
		t.Error("IsOpen before Close: want true")
	}
	tif.Close()
	if tif.IsOpen() {
		t.Error("IsOpen after Close: want false")
	}
}

func TestGetFieldAnyUint32(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.tif")
	createTestTIFF(t, path, 4, 4)

	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	val, err := tif.GetFieldAny(TagImageWidth)
	if err != nil {
		t.Fatalf("GetFieldAny(ImageWidth): %v", err)
	}
	v, ok := val.(uint32)
	if !ok {
		t.Fatalf("GetFieldAny(ImageWidth): type = %T, want uint32", val)
	}
	if v != 4 {
		t.Errorf("GetFieldAny(ImageWidth) = %d, want 4", v)
	}
}

func TestGetFieldAnyString(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.tif")
	createTestTIFF(t, path, 4, 4)

	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	val, err := tif.GetFieldAny(TagSoftware)
	if err != nil {
		t.Fatalf("GetFieldAny(Software): %v", err)
	}
	v, ok := val.(string)
	if !ok {
		t.Fatalf("GetFieldAny(Software): type = %T, want string", val)
	}
	if v != "golibtiff test" {
		t.Errorf("GetFieldAny(Software) = %q, want %q", v, "golibtiff test")
	}
}

func TestGetFieldAnyFieldNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.tif")
	createTestTIFF(t, path, 4, 4)

	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	_, err = tif.GetFieldAny(Tag(0xFFFE)) // non-existent tag
	if err == nil {
		t.Error("GetFieldAny(non-existent): want error, got nil")
	}
}

func TestSetFieldAnyInt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.tif")

	tif, err := Open(path, OpenWrite)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	tif.SetFieldUint32(TagImageWidth, 1)
	tif.SetFieldUint32(TagImageLength, 1)
	tif.SetFieldUint16(TagBitsPerSample, 8)
	tif.SetFieldUint16(TagSamplesPerPixel, 1)
	tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
	tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
	tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))

	// SetFieldAny with int16 (FillOrder, tag 266)
	if err := tif.SetFieldAny(TagFillOrder, int16(1)); err != nil {
		t.Errorf("SetFieldAny(int16): %v", err)
	}

	// SetFieldAny with int32 (TileWidth, tag 322)
	if err := tif.SetFieldAny(TagTileWidth, int32(256)); err != nil {
		t.Errorf("SetFieldAny(int32): %v", err)
	}

	tif.Close()
}
