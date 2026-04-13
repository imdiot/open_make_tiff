package golibtiff

import (
	"os"
	"path/filepath"
	"testing"
)

// --- OpenFromBuffer tests ---

func TestOpenFromBuffer(t *testing.T) {
	path := filepath.Join("testdata", "rgb-3c-8b.tiff")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	tif, err := OpenFromBuffer(data)
	if err != nil {
		t.Fatalf("OpenFromBuffer: %v", err)
	}
	defer tif.Close()

	w, err := tif.Width()
	if err != nil {
		t.Fatalf("Width: %v", err)
	}
	if w != 157 {
		t.Errorf("Width = %d, want 157", w)
	}

	h, err := tif.Height()
	if err != nil {
		t.Fatalf("Height: %v", err)
	}
	if h != 151 {
		t.Errorf("Height = %d, want 151", h)
	}
}

func TestOpenFromBufferEmpty(t *testing.T) {
	_, err := OpenFromBuffer(nil)
	if err == nil {
		t.Fatal("expected error for nil buffer")
	}
	_, err = OpenFromBuffer([]byte{})
	if err == nil {
		t.Fatal("expected error for empty buffer")
	}
}

func TestOpenFromBufferInvalid(t *testing.T) {
	_, err := OpenFromBuffer([]byte("not a tiff file"))
	if err == nil {
		t.Fatal("expected error for invalid TIFF data")
	}
}

// TestOpenFromBufferReadPixels verifies pixel data can be read from a buffer-backed TIFF.
func TestOpenFromBufferReadPixels(t *testing.T) {
	path := filepath.Join("testdata", "rgb-3c-8b.tiff")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	tif, err := OpenFromBuffer(data)
	if err != nil {
		t.Fatalf("OpenFromBuffer: %v", err)
	}
	defer tif.Close()

	w, _ := tif.Width()
	scanline := make([]byte, tif.ScanlineSize())
	if err := tif.ReadScanline(scanline, 0); err != nil {
		t.Fatalf("ReadScanline: %v", err)
	}
	if len(scanline) != int(w*3) {
		t.Errorf("scanline size = %d, want %d", len(scanline), w*3)
	}
}

// TestOpenFromBufferIsolation verifies that modifying the original buffer
// after opening does not affect the TIFF (buffer is copied).
func TestOpenFromBufferIsolation(t *testing.T) {
	path := filepath.Join("testdata", "rgb-3c-8b.tiff")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	original := make([]byte, len(data))
	copy(original, data)

	tif, err := OpenFromBuffer(data)
	if err != nil {
		t.Fatalf("OpenFromBuffer: %v", err)
	}
	defer tif.Close()

	// Corrupt the original buffer
	for i := range data {
		data[i] = 0xFF
	}

	// Should still read correctly from the copy
	w, err := tif.Width()
	if err != nil {
		t.Fatalf("Width after corruption: %v", err)
	}
	if w != 157 {
		t.Errorf("Width = %d after corruption, want 157", w)
	}
}

// --- WriteToBuffer tests ---

func TestWriteToBuffer(t *testing.T) {
	tif, getBuffer, err := WriteToBuffer()
	if err != nil {
		t.Fatalf("WriteToBuffer: %v", err)
	}

	// Write a simple 4x4 grayscale image
	w, h := uint32(4), uint32(4)
	tif.SetFieldUint32(TagImageWidth, w)
	tif.SetFieldUint32(TagImageLength, h)
	tif.SetFieldUint16(TagBitsPerSample, 8)
	tif.SetFieldUint16(TagSamplesPerPixel, 1)
	tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
	tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
	tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))

	for row := uint32(0); row < h; row++ {
		scanline := []byte{byte(row), byte(row), byte(row), byte(row)}
		if err := tif.WriteScanline(scanline, row); err != nil {
			t.Fatalf("WriteScanline %d: %v", row, err)
		}
	}

	if err := tif.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	buf := getBuffer()
	if len(buf) == 0 {
		t.Fatal("buffer is empty after writing")
	}

	// Verify the buffer is a valid TIFF by reading it back
	readTif, err := OpenFromBuffer(buf)
	if err != nil {
		t.Fatalf("OpenFromBuffer of written data: %v", err)
	}
	defer readTif.Close()

	rw, err := readTif.Width()
	if err != nil {
		t.Fatalf("read Width: %v", err)
	}
	if rw != w {
		t.Errorf("read Width = %d, want %d", rw, w)
	}

	rh, err := readTif.Height()
	if err != nil {
		t.Fatalf("read Height: %v", err)
	}
	if rh != h {
		t.Errorf("read Height = %d, want %d", rh, h)
	}

	// Verify pixel data
	scanline := make([]byte, readTif.ScanlineSize())
	for row := uint32(0); row < h; row++ {
		if err := readTif.ReadScanline(scanline, row); err != nil {
			t.Fatalf("ReadScanline %d: %v", row, err)
		}
		expected := byte(row)
		for _, b := range scanline {
			if b != expected {
				t.Errorf("row %d: pixel = %d, want %d", row, b, expected)
				break
			}
		}
	}
}

func TestWriteToBufferWithCompression(t *testing.T) {
	tif, getBuffer, err := WriteToBuffer()
	if err != nil {
		t.Fatalf("WriteToBuffer: %v", err)
	}

	w, h := uint32(8), uint32(8)
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
			scanline[i] = byte(row)
		}
		if err := tif.WriteScanline(scanline, row); err != nil {
			t.Fatalf("WriteScanline %d: %v", row, err)
		}
	}

	if err := tif.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	buf := getBuffer()
	if len(buf) == 0 {
		t.Fatal("buffer is empty")
	}

	// Verify round-trip
	readTif, err := OpenFromBuffer(buf)
	if err != nil {
		t.Fatalf("OpenFromBuffer: %v", err)
	}
	defer readTif.Close()

	rw, _ := readTif.Width()
	rh, _ := readTif.Height()
	if rw != w || rh != h {
		t.Errorf("dimensions = %dx%d, want %dx%d", rw, rh, w, h)
	}

	comp, _ := readTif.Compression()
	if comp != uint16(CompressionDeflate) {
		t.Errorf("Compression = %d, want %d", comp, CompressionDeflate)
	}
}
