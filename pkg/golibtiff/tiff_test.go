package golibtiff

import (
	"os"
	"path/filepath"
	"testing"
)

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
	if err := tif.SetFieldUint16(TagCompression, CompressionNone); err != nil {
		t.Fatalf("SetField compression: %v", err)
	}
	if err := tif.SetFieldUint16(TagPhotometric, PhotometricRGB); err != nil {
		t.Fatalf("SetField photometric: %v", err)
	}
	if err := tif.SetFieldUint16(TagPlanarConfig, PlanarConfigContig); err != nil {
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

	if tif.Width() != w {
		t.Errorf("Width = %d, want %d", tif.Width(), w)
	}
	if tif.Height() != h {
		t.Errorf("Height = %d, want %d", tif.Height(), h)
	}
	if tif.BitsPerSample() != 8 {
		t.Errorf("BitsPerSample = %d, want 8", tif.BitsPerSample())
	}
	if tif.SamplesPerPixel() != 3 {
		t.Errorf("SamplesPerPixel = %d, want 3", tif.SamplesPerPixel())
	}
	if tif.Compression() != CompressionNone {
		t.Errorf("Compression = %d, want %d", tif.Compression(), CompressionNone)
	}
	if tif.Photometric() != PhotometricRGB {
		t.Errorf("Photometric = %d, want %d", tif.Photometric(), PhotometricRGB)
	}
	if tif.PlanarConfig() != PlanarConfigContig {
		t.Errorf("PlanarConfig = %d, want %d", tif.PlanarConfig(), PlanarConfigContig)
	}

	sw, err := tif.Software()
	if err != nil {
		t.Errorf("Software error: %v", err)
	}
	if sw != "golibtiff test" {
		t.Errorf("Software = %q, want %q", sw, "golibtiff test")
	}

	scanlineSize := tif.ScanlineSize()
	if scanlineSize != int64(w*3) {
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

func TestLZWCompression(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lzw.tif")

	const w, h = 128, 64

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
		tif.SetFieldUint16(TagCompression, CompressionLZW)
		tif.SetFieldUint16(TagPhotometric, PhotometricMinIsBlack)
		tif.SetFieldUint16(TagPlanarConfig, PlanarConfigContig)
		tif.SetFieldUint32(TagRowsPerStrip, h)

		data := make([]byte, w*h)
		for i := range data {
			data[i] = byte(i % 256)
		}
		if _, err := tif.WriteEncodedStrip(0, data); err != nil {
			t.Fatalf("WriteEncodedStrip: %v", err)
		}
	}()

	// Read back in a separate scope to ensure file is closed.
	tif2, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open read: %v", err)
	}
	defer tif2.Close()

	if tif2.Compression() != CompressionLZW {
		t.Errorf("Compression = %d, want %d", tif2.Compression(), CompressionLZW)
	}

	buf := make([]byte, w*h)
	n, err := tif2.ReadEncodedStrip(0, buf, -1)
	if err != nil {
		t.Fatalf("ReadEncodedStrip: %v", err)
	}
	if n != w*h {
		t.Errorf("ReadEncodedStrip returned %d bytes, want %d", n, w*h)
	}
	for i := 0; i < w*h; i++ {
		expected := byte(i % 256)
		if buf[i] != expected {
			t.Errorf("byte %d mismatch: got %d, want %d", i, buf[i], expected)
			break
		}
	}
}

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

func TestOpenNonexistent(t *testing.T) {
	_, err := Open("/nonexistent/path/test.tif", OpenRead)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

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
		tif.SetFieldUint16(TagCompression, CompressionNone)
		tif.SetFieldUint16(TagPhotometric, PhotometricMinIsBlack)
		tif.SetFieldUint16(TagPlanarConfig, PlanarConfigContig)
		tif.SetFieldString(TagArtist, "test artist")
		tif.SetFieldString(TagDocumentName, "test doc")
		tif.SetFieldFloat(TagXResolution, 72.0)
		tif.SetFieldFloat(TagYResolution, 72.0)
		tif.SetFieldUint16(TagResolutionUnit, ResolutionUnitInch)

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

func TestClosedHandle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "closed.tif")
	createTestTIFF(t, path, 8, 8)

	tif, _ := Open(path, OpenRead)
	tif.Close()

	if tif.Width() != 0 {
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

// --- Multi-page (multi-IFD) tests, inspired by libtiff test_directory.c ---

const nDirectories = 10

// writeMultiPageTIFF creates a TIFF with nDirectories pages.
// Each page is a single-pixel grayscale image; the pixel value equals the page index.
func writeMultiPageTIFF(t *testing.T, path string) {
	t.Helper()
	tif, err := Open(path, OpenWrite)
	if err != nil {
		t.Fatalf("Open write: %v", err)
	}

	for i := 0; i < nDirectories; i++ {
		tif.SetFieldUint32(TagImageWidth, 1)
		tif.SetFieldUint32(TagImageLength, 1)
		tif.SetFieldUint16(TagBitsPerSample, 8)
		tif.SetFieldUint16(TagSamplesPerPixel, 1)
		tif.SetFieldUint16(TagCompression, CompressionNone)
		tif.SetFieldUint16(TagPhotometric, PhotometricMinIsBlack)
		tif.SetFieldUint16(TagPlanarConfig, PlanarConfigContig)

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

	for i := 0; i < nDirectories; i++ {
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

	page := 0
	buf := make([]byte, 1)
	if err := tif.ReadScanline(buf, 0); err != nil {
		t.Fatalf("ReadScanline page 0: %v", err)
	}
	if buf[0] != 0 {
		t.Errorf("Page 0 pixel = %d, want 0", buf[0])
	}

	for page = 1; page < nDirectories; page++ {
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
		tif.SetFieldUint16(TagCompression, CompressionNone)
		tif.SetFieldUint16(TagPhotometric, PhotometricMinIsBlack)
		tif.SetFieldUint16(TagPlanarConfig, PlanarConfigContig)

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

// --- BigTIFF test ---

func TestBigTIFF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bigtiff.tif")

	tif, err := Open(path, "w8")
	if err != nil {
		t.Fatalf("Open BigTIFF: %v", err)
	}

	tif.SetFieldUint32(TagImageWidth, 8)
	tif.SetFieldUint32(TagImageLength, 8)
	tif.SetFieldUint16(TagBitsPerSample, 8)
	tif.SetFieldUint16(TagSamplesPerPixel, 1)
	tif.SetFieldUint16(TagCompression, CompressionNone)
	tif.SetFieldUint16(TagPhotometric, PhotometricMinIsBlack)
	tif.SetFieldUint16(TagPlanarConfig, PlanarConfigContig)

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
}

// --- RGBA image test ---

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
		tif.SetFieldUint16(TagCompression, CompressionNone)
		tif.SetFieldUint16(TagPhotometric, PhotometricRGB)
		tif.SetFieldUint16(TagPlanarConfig, PlanarConfigContig)

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

// --- Slice field test ---

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
		tif.SetFieldUint16(TagCompression, CompressionNone)
		tif.SetFieldUint16(TagPhotometric, PhotometricMinIsBlack)
		tif.SetFieldUint16(TagPlanarConfig, PlanarConfigContig)

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

// --- Strip read/write test ---

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
		tif.SetFieldUint16(TagCompression, CompressionNone)
		tif.SetFieldUint16(TagPhotometric, PhotometricMinIsBlack)
		tif.SetFieldUint16(TagPlanarConfig, PlanarConfigContig)
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

// --- Deflate (zlib) compression test ---

func TestDeflateCompression(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deflate.tif")

	const w, h = 64, 32

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
		tif.SetFieldUint16(TagCompression, CompressionDeflate)
		tif.SetFieldUint16(TagPhotometric, PhotometricMinIsBlack)
		tif.SetFieldUint16(TagPlanarConfig, PlanarConfigContig)
		tif.SetFieldUint32(TagRowsPerStrip, h)

		data := make([]byte, w*h)
		for i := range data {
			data[i] = byte(i % 256)
		}
		if _, err := tif.WriteEncodedStrip(0, data); err != nil {
			t.Fatalf("WriteEncodedStrip: %v", err)
		}
	}()

	tif2, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open read: %v", err)
	}
	defer tif2.Close()

	if tif2.Compression() != CompressionDeflate {
		t.Errorf("Compression = %d, want %d", tif2.Compression(), CompressionDeflate)
	}

	buf := make([]byte, w*h)
	n, err := tif2.ReadEncodedStrip(0, buf, -1)
	if err != nil {
		t.Fatalf("ReadEncodedStrip: %v", err)
	}
	if n != w*h {
		t.Errorf("ReadEncodedStrip returned %d, want %d", n, w*h)
	}
	for i := range buf {
		if buf[i] != byte(i%256) {
			t.Errorf("byte %d mismatch: got %d, want %d", i, buf[i], byte(i%256))
			break
		}
	}
}

// --- Multi-page BigTIFF test ---

func TestBigTIFFMultiPage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bigtiff_multi.tif")

	const pages = 5

	func() {
		tif, err := Open(path, "w8")
		if err != nil {
			t.Fatalf("Open BigTIFF: %v", err)
		}
		defer tif.Close()

		for p := 0; p < pages; p++ {
			tif.SetFieldUint32(TagImageWidth, 4)
			tif.SetFieldUint32(TagImageLength, 4)
			tif.SetFieldUint16(TagBitsPerSample, 8)
			tif.SetFieldUint16(TagSamplesPerPixel, 1)
			tif.SetFieldUint16(TagCompression, CompressionNone)
			tif.SetFieldUint16(TagPhotometric, PhotometricMinIsBlack)
			tif.SetFieldUint16(TagPlanarConfig, PlanarConfigContig)

			data := make([]byte, 16)
			for i := range data {
				data[i] = byte(int(p)*16 + i)
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

// --- Tests using libtiff test fixture files ---

func openFixture(t *testing.T, name string) *TIFF {
	t.Helper()
	path := filepath.Join("testdata", name)
	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open %s: %v", name, err)
	}
	return tif
}

func TestFixtureGrayscale8bit(t *testing.T) {
	tif := openFixture(t, "minisblack-1c-8b.tiff")
	defer tif.Close()

	if tif.Width() == 0 || tif.Height() == 0 {
		t.Error("expected non-zero dimensions")
	}
	if tif.BitsPerSample() != 8 {
		t.Errorf("BitsPerSample = %d, want 8", tif.BitsPerSample())
	}
	if tif.SamplesPerPixel() != 1 {
		t.Errorf("SamplesPerPixel = %d, want 1", tif.SamplesPerPixel())
	}
	if tif.Photometric() != PhotometricMinIsBlack {
		t.Errorf("Photometric = %d, want %d", tif.Photometric(), PhotometricMinIsBlack)
	}
	if tif.IsTiled() {
		t.Error("strip-based image should not report as tiled")
	}

	buf := make([]byte, tif.ScanlineSize())
	if err := tif.ReadScanline(buf, 0); err != nil {
		t.Fatalf("ReadScanline 0: %v", err)
	}
}

func TestFixtureGrayscale16bit(t *testing.T) {
	tif := openFixture(t, "minisblack-1c-16b.tiff")
	defer tif.Close()

	if tif.BitsPerSample() != 16 {
		t.Errorf("BitsPerSample = %d, want 16", tif.BitsPerSample())
	}
	expected := int64(tif.Width()) * 2
	if tif.ScanlineSize() != expected {
		t.Errorf("ScanlineSize = %d, want %d", tif.ScanlineSize(), expected)
	}

	buf := make([]byte, tif.ScanlineSize())
	if err := tif.ReadScanline(buf, 0); err != nil {
		t.Fatalf("ReadScanline 0: %v", err)
	}
}

func TestFixtureRGB8bit(t *testing.T) {
	tif := openFixture(t, "rgb-3c-8b.tiff")
	defer tif.Close()

	if tif.SamplesPerPixel() != 3 {
		t.Errorf("SamplesPerPixel = %d, want 3", tif.SamplesPerPixel())
	}
	if tif.Photometric() != PhotometricRGB {
		t.Errorf("Photometric = %d, want %d (RGB)", tif.Photometric(), PhotometricRGB)
	}

	if tif.BitsPerSample() != 8 {
		t.Errorf("BitsPerSample = %d, want 8", tif.BitsPerSample())
	}

	buf := make([]byte, tif.ScanlineSize())
	if err := tif.ReadScanline(buf, 0); err != nil {
		t.Fatalf("ReadScanline 0: %v", err)
	}
}

func TestFixtureRGB16bit(t *testing.T) {
	tif := openFixture(t, "rgb-3c-16b.tiff")
	defer tif.Close()

	if tif.BitsPerSample() != 16 {
		t.Errorf("BitsPerSample = %d, want 16", tif.BitsPerSample())
	}
	if tif.SamplesPerPixel() != 3 {
		t.Errorf("SamplesPerPixel = %d, want 3", tif.SamplesPerPixel())
	}

	expected := int64(tif.Width()) * 3 * 2
	if tif.ScanlineSize() != expected {
		t.Errorf("ScanlineSize = %d, want %d", tif.ScanlineSize(), expected)
	}
}

func TestFixtureBilevel(t *testing.T) {
	tif := openFixture(t, "miniswhite-1c-1b.tiff")
	defer tif.Close()

	if tif.BitsPerSample() != 1 {
		t.Errorf("BitsPerSample = %d, want 1", tif.BitsPerSample())
	}
	if tif.Photometric() != PhotometricMinIsWhite {
		t.Errorf("Photometric = %d, want %d", tif.Photometric(), PhotometricMinIsWhite)
	}
}

func TestFixturePalette(t *testing.T) {
	tif := openFixture(t, "palette-1c-8b.tiff")
	defer tif.Close()

	if tif.Photometric() != PhotometricPalette {
		t.Errorf("Photometric = %d, want %d (Palette)", tif.Photometric(), PhotometricPalette)
	}
	if tif.BitsPerSample() != 8 {
		t.Errorf("BitsPerSample = %d, want 8", tif.BitsPerSample())
	}
}

func TestFixtureLZWSingleStrip(t *testing.T) {
	tif := openFixture(t, "lzw-single-strip.tiff")
	defer tif.Close()

	if tif.Compression() != CompressionLZW {
		t.Errorf("Compression = %d, want %d (LZW)", tif.Compression(), CompressionLZW)
	}

	stripSize := tif.StripSize()
	buf := make([]byte, stripSize)
	for strip := uint32(0); strip < tif.NumberOfStrips(); strip++ {
		n, err := tif.ReadEncodedStrip(strip, buf, -1)
		if err != nil {
			t.Fatalf("ReadEncodedStrip %d: %v", strip, err)
		}
		if n <= 0 {
			t.Errorf("Strip %d: read %d bytes", strip, n)
		}
	}
}

func TestFixtureLZWCompat(t *testing.T) {
	tif := openFixture(t, "quad-lzw-compat.tiff")
	defer tif.Close()

	if tif.Compression() != CompressionLZW {
		t.Errorf("Compression = %d, want %d", tif.Compression(), CompressionLZW)
	}

	buf := make([]byte, tif.StripSize())
	for strip := uint32(0); strip < tif.NumberOfStrips(); strip++ {
		_, err := tif.ReadEncodedStrip(strip, buf, -1)
		if err != nil {
			t.Fatalf("ReadEncodedStrip %d: %v", strip, err)
		}
	}
}

func TestFixtureTiled(t *testing.T) {
	tif := openFixture(t, "quad-tile.jpg.tiff")
	defer tif.Close()

	if !tif.IsTiled() {
		t.Fatal("expected tiled image")
	}

	tw := tif.TileWidth()
	tl := tif.TileLength()
	if tw == 0 || tl == 0 {
		t.Errorf("TileWidth=%d, TileLength=%d, expected non-zero", tw, tl)
	}

	tileSize := tif.TileSize()
	if tileSize <= 0 {
		t.Errorf("TileSize = %d, expected > 0", tileSize)
	}

	numTiles := tif.NumberOfTiles()
	if numTiles == 0 {
		t.Error("NumberOfTiles = 0, expected > 0")
	}
}

func TestFixtureAlphaChannel(t *testing.T) {
	tif := openFixture(t, "minisblack-2c-8b-alpha.tiff")
	defer tif.Close()

	if tif.SamplesPerPixel() != 2 {
		t.Errorf("SamplesPerPixel = %d, want 2 (grayscale + alpha)", tif.SamplesPerPixel())
	}

	if tif.BitsPerSample() != 8 {
		t.Errorf("BitsPerSample = %d, want 8", tif.BitsPerSample())
	}
}

func TestFixtureTwoIFDs(t *testing.T) {
	tif := openFixture(t, "test_two_ifds.tif")
	defer tif.Close()

	count := tif.NumberOfDirectories()
	if count != 2 {
		t.Errorf("NumberOfDirectories = %d, want 2", count)
	}

	if !tif.ReadDirectory() {
		t.Fatal("ReadDirectory() returned false, expected second IFD")
	}
	if tif.CurrentDirectory() != 1 {
		t.Errorf("CurrentDirectory = %d, want 1", tif.CurrentDirectory())
	}

	if tif.ReadDirectory() {
		t.Error("ReadDirectory() should return false after second IFD")
	}

	if err := tif.SetDirectory(0); err != nil {
		t.Fatalf("SetDirectory(0): %v", err)
	}
	if tif.CurrentDirectory() != 0 {
		t.Errorf("CurrentDirectory = %d, want 0", tif.CurrentDirectory())
	}
}

func TestFixtureRGBARead(t *testing.T) {
	tif := openFixture(t, "rgb-3c-8b.tiff")
	defer tif.Close()

	if err := tif.RGBAImageOK(); err != nil {
		t.Fatalf("RGBAImageOK: %v", err)
	}

	w, h := tif.Width(), tif.Height()
	buf := make([]uint32, w*h)
	if err := tif.ReadRGBAImage(buf); err != nil {
		t.Fatalf("ReadRGBAImage: %v", err)
	}

	if buf[0] == 0 {
		t.Error("first RGBA pixel is all-zero, expected color data")
	}
}

func TestFixture32bppNone(t *testing.T) {
	tif := openFixture(t, "32bpp-None.tiff")
	defer tif.Close()

	// 32bpp-None is actually 8-bit per channel RGBA (4 samples * 8 bits = 32bpp).
	if tif.BitsPerSample() != 8 {
		t.Errorf("BitsPerSample = %d, want 8", tif.BitsPerSample())
	}
	if tif.SamplesPerPixel() != 4 {
		t.Errorf("SamplesPerPixel = %d, want 4", tif.SamplesPerPixel())
	}
}

func TestFixtureDeflateLastStrip(t *testing.T) {
	tif := openFixture(t, "deflate-last-strip-extra-data.tiff")
	defer tif.Close()

	if tif.Compression() != CompressionDeflate {
		t.Errorf("Compression = %d, want %d", tif.Compression(), CompressionDeflate)
	}

	buf := make([]byte, tif.StripSize())
	for strip := uint32(0); strip < tif.NumberOfStrips(); strip++ {
		_, err := tif.ReadEncodedStrip(strip, buf, -1)
		if err != nil {
			t.Fatalf("ReadEncodedStrip %d: %v", strip, err)
		}
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
