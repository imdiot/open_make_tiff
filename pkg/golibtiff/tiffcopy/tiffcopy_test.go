package tiffcopy

import (
	"os"
	"path/filepath"
	"testing"

	"open-make-tiff/pkg/golibtiff"
)

func createTestTIFF(t *testing.T, path string, w, h uint32) {
	t.Helper()
	tif, err := golibtiff.Open(path, golibtiff.OpenWrite)
	if err != nil {
		t.Fatalf("Open write: %v", err)
	}
	tif.SetFieldUint32(golibtiff.TagImageWidth, w)
	tif.SetFieldUint32(golibtiff.TagImageLength, h)
	tif.SetFieldUint16(golibtiff.TagBitsPerSample, 8)
	tif.SetFieldUint16(golibtiff.TagSamplesPerPixel, 3)
	tif.SetFieldUint16(golibtiff.TagCompression, golibtiff.CompressionNone)
	tif.SetFieldUint16(golibtiff.TagPhotometric, golibtiff.PhotometricRGB)
	tif.SetFieldUint16(golibtiff.TagPlanarConfig, golibtiff.PlanarConfigContig)

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

func createTestTIFF16bit(t *testing.T, path string, w, h uint32) {
	t.Helper()
	tif, err := golibtiff.Open(path, golibtiff.OpenWrite)
	if err != nil {
		t.Fatalf("Open write: %v", err)
	}
	tif.SetFieldUint32(golibtiff.TagImageWidth, w)
	tif.SetFieldUint32(golibtiff.TagImageLength, h)
	tif.SetFieldUint16(golibtiff.TagBitsPerSample, 16)
	tif.SetFieldUint16(golibtiff.TagSamplesPerPixel, 3)
	tif.SetFieldUint16(golibtiff.TagCompression, golibtiff.CompressionNone)
	tif.SetFieldUint16(golibtiff.TagPhotometric, golibtiff.PhotometricRGB)
	tif.SetFieldUint16(golibtiff.TagPlanarConfig, golibtiff.PlanarConfigContig)

	scanline := make([]byte, w*6)
	for row := uint32(0); row < h; row++ {
		for col := uint32(0); col < w; col++ {
			val := uint16(row*256 + col)
			scanline[col*6+0] = byte(val)
			scanline[col*6+1] = byte(val >> 8)
			scanline[col*6+2] = byte(val)
			scanline[col*6+3] = byte(val >> 8)
			scanline[col*6+4] = byte(val)
			scanline[col*6+5] = byte(val >> 8)
		}
		if err := tif.WriteScanline(scanline, row); err != nil {
			t.Fatalf("WriteScanline %d: %v", row, err)
		}
	}
	tif.Close()
}

func createMultiPageTIFF(t *testing.T, path string, pages int) {
	t.Helper()
	tif, err := golibtiff.Open(path, golibtiff.OpenWrite)
	if err != nil {
		t.Fatalf("Open write: %v", err)
	}

	for p := 0; p < pages; p++ {
		tif.SetFieldUint32(golibtiff.TagImageWidth, 4)
		tif.SetFieldUint32(golibtiff.TagImageLength, 4)
		tif.SetFieldUint16(golibtiff.TagBitsPerSample, 8)
		tif.SetFieldUint16(golibtiff.TagSamplesPerPixel, 1)
		tif.SetFieldUint16(golibtiff.TagCompression, golibtiff.CompressionNone)
		tif.SetFieldUint16(golibtiff.TagPhotometric, golibtiff.PhotometricMinIsBlack)
		tif.SetFieldUint16(golibtiff.TagPlanarConfig, golibtiff.PlanarConfigContig)

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
	tif.Close()
}

func TestCopySinglePage(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.tif")
	dst := filepath.Join(dir, "dst.tif")

	const w, h = 64, 32
	createTestTIFF(t, src, w, h)

	if err := Copy(src, dst); err != nil {
		t.Fatalf("Copy: %v", err)
	}

	tif, err := golibtiff.Open(dst, golibtiff.OpenRead)
	if err != nil {
		t.Fatalf("Open dst: %v", err)
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
	if tif.NumberOfDirectories() != 1 {
		t.Errorf("NumberOfDirectories = %d, want 1", tif.NumberOfDirectories())
	}

	buf := make([]byte, tif.ScanlineSize())
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

func TestCopyMultiPage(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.tif")
	dst := filepath.Join(dir, "dst.tif")

	const pages = 5
	createMultiPageTIFF(t, src, pages)

	if err := Copy(src, dst); err != nil {
		t.Fatalf("Copy: %v", err)
	}

	tif, err := golibtiff.Open(dst, golibtiff.OpenRead)
	if err != nil {
		t.Fatalf("Open dst: %v", err)
	}
	defer tif.Close()

	if tif.NumberOfDirectories() != pages {
		t.Fatalf("NumberOfDirectories = %d, want %d", tif.NumberOfDirectories(), pages)
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

func TestCopyWithLZWCompression(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.tif")
	dst := filepath.Join(dir, "dst.tif")

	const w, h = 128, 64
	createTestTIFF(t, src, w, h)

	if err := Copy(src, dst, WithLZWCompression(golibtiff.PredictorHorizontal)); err != nil {
		t.Fatalf("Copy: %v", err)
	}

	tif, err := golibtiff.Open(dst, golibtiff.OpenRead)
	if err != nil {
		t.Fatalf("Open dst: %v", err)
	}
	defer tif.Close()

	if tif.Compression() != golibtiff.CompressionLZW {
		t.Errorf("Compression = %d, want %d (LZW)", tif.Compression(), golibtiff.CompressionLZW)
	}

	pred, err := tif.GetFieldUint16(golibtiff.TagPredictor)
	if err != nil || pred != 2 {
		t.Errorf("Predictor = %d, err=%v, want 2", pred, err)
	}

	buf := make([]byte, tif.ScanlineSize())
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

func TestCopy16bitRGB(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.tif")
	dst := filepath.Join(dir, "dst.tif")

	const w, h = 32, 16
	createTestTIFF16bit(t, src, w, h)

	if err := Copy(src, dst, WithLZWCompression(golibtiff.PredictorHorizontal)); err != nil {
		t.Fatalf("Copy: %v", err)
	}

	tif, err := golibtiff.Open(dst, golibtiff.OpenRead)
	if err != nil {
		t.Fatalf("Open dst: %v", err)
	}
	defer tif.Close()

	if tif.BitsPerSample() != 16 {
		t.Errorf("BitsPerSample = %d, want 16", tif.BitsPerSample())
	}
	if tif.SamplesPerPixel() != 3 {
		t.Errorf("SamplesPerPixel = %d, want 3", tif.SamplesPerPixel())
	}
	if tif.Compression() != golibtiff.CompressionLZW {
		t.Errorf("Compression = %d, want LZW", tif.Compression())
	}

	buf := make([]byte, tif.ScanlineSize())
	for row := uint32(0); row < h; row++ {
		if err := tif.ReadScanline(buf, row); err != nil {
			t.Fatalf("ReadScanline %d: %v", row, err)
		}
		for col := uint32(0); col < w; col++ {
			expected := uint16(row*256 + col)
			got := uint16(buf[col*6+0]) | uint16(buf[col*6+1])<<8
			if got != expected {
				t.Fatalf("row %d col %d: got %d, want %d", row, col, got, expected)
			}
		}
	}
}

func TestCopyTiledFromFixture(t *testing.T) {
	fixturePath := filepath.Join("..", "testdata", "quad-tile.jpg.tiff")
	if _, err := os.Stat(fixturePath); err != nil {
		t.Skip("fixture not found:", fixturePath)
	}

	dir := t.TempDir()
	dst := filepath.Join(dir, "dst.tif")

	if err := Copy(fixturePath, dst); err != nil {
		t.Fatalf("Copy: %v", err)
	}

	tif, err := golibtiff.Open(dst, golibtiff.OpenRead)
	if err != nil {
		t.Fatalf("Open dst: %v", err)
	}
	defer tif.Close()

	if !tif.IsTiled() {
		t.Error("expected tiled image")
	}
	if tif.Width() == 0 || tif.Height() == 0 {
		t.Error("expected non-zero dimensions")
	}
}

func TestCopyPreservesCompression(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.tif")
	dst := filepath.Join(dir, "dst.tif")

	const w, h = 64, 32
	func() {
		tif, err := golibtiff.Open(src, golibtiff.OpenWrite)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer tif.Close()

		tif.SetFieldUint32(golibtiff.TagImageWidth, w)
		tif.SetFieldUint32(golibtiff.TagImageLength, h)
		tif.SetFieldUint16(golibtiff.TagBitsPerSample, 8)
		tif.SetFieldUint16(golibtiff.TagSamplesPerPixel, 1)
		tif.SetFieldUint16(golibtiff.TagCompression, golibtiff.CompressionLZW)
		tif.SetFieldUint16(golibtiff.TagPhotometric, golibtiff.PhotometricMinIsBlack)
		tif.SetFieldUint16(golibtiff.TagPlanarConfig, golibtiff.PlanarConfigContig)
		tif.SetFieldUint32(golibtiff.TagRowsPerStrip, h)

		data := make([]byte, w*h)
		for i := range data {
			data[i] = byte(i % 256)
		}
		if _, err := tif.WriteEncodedStrip(0, data); err != nil {
			t.Fatalf("WriteEncodedStrip: %v", err)
		}
	}()

	// Copy without specifying compression — should preserve LZW
	if err := Copy(src, dst); err != nil {
		t.Fatalf("Copy: %v", err)
	}

	tif, err := golibtiff.Open(dst, golibtiff.OpenRead)
	if err != nil {
		t.Fatalf("Open dst: %v", err)
	}
	defer tif.Close()

	if tif.Compression() != golibtiff.CompressionLZW {
		t.Errorf("Compression = %d, want %d (LZW preserved)", tif.Compression(), golibtiff.CompressionLZW)
	}

	buf := make([]byte, w*h)
	n, err := tif.ReadEncodedStrip(0, buf, -1)
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

func TestCopyNonexistentSource(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "dst.tif")

	err := Copy("/nonexistent/path/test.tif", dst)
	if err == nil {
		t.Fatal("expected error for nonexistent source")
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
