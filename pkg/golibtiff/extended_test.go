package golibtiff

import (
	"bytes"
	"errors"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

// --- Phase 1: Pseudo-Tag constants ---

func TestPseudoTagConstants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tag  Tag
		want uint32
	}{
		{"JPEGQuality", PseudoTagJPEGQuality, 65537},
		{"JPEGColorMode", PseudoTagJPEGColorMode, 65538},
		{"JPEGTablesMode", PseudoTagJPEGTablesMode, 65539},
		{"ZIPQuality", PseudoTagZIPQuality, 65557},
		{"LZMAPreset", PseudoTagLZMAPreset, 65562},
		{"ZSTDLevel", PseudoTagZSTDLevel, 65564},
		{"LERCVersion", PseudoTagLERCVersion, 65565},
		{"LERCAddCompression", PseudoTagLERCAddCompression, 65566},
		{"LERCMaxZError", PseudoTagLERCMaxZError, 65567},
		{"WebPLevel", PseudoTagWebPLevel, 65568},
		{"WebPLossless", PseudoTagWebPLossless, 65569},
		{"DeflateSubCodec", PseudoTagDeflateSubCodec, 65570},
		{"WebPLosslessExact", PseudoTagWebPLosslessExact, 65571},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if uint32(tt.tag) != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.tag, tt.want)
			}
		})
	}
}

func TestDataTypeConstants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		got  int
		want int
	}{
		{"Byte", int(DataTypeByte), 1},
		{"ASCII", int(DataTypeASCII), 2},
		{"Short", int(DataTypeShort), 3},
		{"Long", int(DataTypeLong), 4},
		{"Rational", int(DataTypeRational), 5},
		{"SByte", int(DataTypeSByte), 6},
		{"Undefined", int(DataTypeUndefined), 7},
		{"SShort", int(DataTypeSShort), 8},
		{"SLong", int(DataTypeSLong), 9},
		{"SRational", int(DataTypeSRational), 10},
		{"Float", int(DataTypeFloat), 11},
		{"Double", int(DataTypeDouble), 12},
		{"IFD", int(DataTypeIFD), 13},
		{"Long8", int(DataTypeLong8), 16},
		{"SLong8", int(DataTypeSLong8), 17},
		{"IFD8", int(DataTypeIFD8), 18},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("DataType%s = %d, want %d", tt.name, tt.got, tt.want)
			}
		})
	}
}

// --- GetFieldDouble ---

func TestGetFieldDouble(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "double_test.tif")

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

	// RATIONAL tags are read as float (32-bit) by libtiff, not double.
	testXRes := 300.0
	testYRes := 150.5
	if err := tif.SetFieldFloat(TagXResolution, testXRes); err != nil {
		t.Fatalf("SetFieldFloat XResolution: %v", err)
	}
	tif.SetFieldFloat(TagYResolution, testYRes)

	if err := tif.WriteScanline([]byte{128}, 0); err != nil {
		t.Fatalf("WriteScanline: %v", err)
	}
	tif.Close()

	readTif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open read: %v", err)
	}
	defer readTif.Close()

	// RATIONAL tags should be read via GetFieldFloat (libtiff returns float*)
	xRes, err := readTif.GetFieldFloat(TagXResolution)
	if err != nil {
		t.Fatalf("GetFieldFloat XResolution: %v", err)
	}
	if math.Abs(xRes-testXRes) > 1e-3 {
		t.Errorf("XResolution = %f, want %f", xRes, testXRes)
	}

	yRes, err := readTif.GetFieldFloat(TagYResolution)
	if err != nil {
		t.Fatalf("GetFieldFloat YResolution: %v", err)
	}
	if math.Abs(yRes-testYRes) > 1e-3 {
		t.Errorf("YResolution = %f, want %f", yRes, testYRes)
	}
}

// --- Compression with Predictor ---

// --- Multi-strip image ---

// --- Edge case: 1x1 image ---

// --- Tile read/write ---

// --- Error types ---

func TestErrorTypes(t *testing.T) {
	t.Run("OpenError", func(t *testing.T) {
		_, err := Open("/nonexistent/path.tif", OpenRead)
		var oe *OpenError
		if !errors.As(err, &oe) {
			t.Fatalf("expected OpenError, got %T: %v", err, err)
		}
		if oe.Path != "/nonexistent/path.tif" {
			t.Errorf("OpenError.Path = %q, want %q", oe.Path, "/nonexistent/path.tif")
		}
		if oe.Mode != OpenRead {
			t.Errorf("OpenError.Mode = %q, want %q", oe.Mode, OpenRead)
		}
	})

	t.Run("FieldError", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.tif")
		createTestTIFF(t, path, 4, 4)

		tif, _ := Open(path, OpenRead)
		defer tif.Close()

		_, err := tif.GetFieldUint16(Tag(99999))
		var fe *FieldError
		if !errors.As(err, &fe) {
			t.Fatalf("expected FieldError, got %T: %v", err, err)
		}
		if fe.Op != "get" {
			t.Errorf("FieldError.Op = %q, want %q", fe.Op, "get")
		}
	})

	t.Run("ClosedHandleError", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.tif")
		createTestTIFF(t, path, 4, 4)

		tif, _ := Open(path, OpenRead)
		tif.Close()

		_, err := tif.Width()
		if err == nil {
			t.Fatal("expected error from closed handle")
		}
	})
}

// --- Append mode ---

func TestAppendMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "append.tif")

	// Create a simple single-page TIFF first
	tif, err := Open(path, OpenWrite)
	if err != nil {
		t.Fatalf("Open write: %v", err)
	}
	tif.SetFieldUint32(TagImageWidth, 2)
	tif.SetFieldUint32(TagImageLength, 2)
	tif.SetFieldUint16(TagBitsPerSample, 8)
	tif.SetFieldUint16(TagSamplesPerPixel, 1)
	tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
	tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
	tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))
	tif.WriteScanline([]byte{1, 2}, 0)
	tif.WriteScanline([]byte{3, 4}, 1)
	tif.Close()

	// Open in append mode
	tif, err = Open(path, OpenAppend)
	if err != nil {
		t.Fatalf("Open append: %v", err)
	}

	// Write the first directory and start a new one
	if err := tif.WriteDirectory(); err != nil {
		t.Fatalf("WriteDirectory: %v", err)
	}

	tif.SetFieldUint32(TagImageWidth, 2)
	tif.SetFieldUint32(TagImageLength, 2)
	tif.SetFieldUint16(TagBitsPerSample, 8)
	tif.SetFieldUint16(TagSamplesPerPixel, 1)
	tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
	tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
	tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))

	if err := tif.WriteScanline([]byte{99, 99}, 0); err != nil {
		t.Fatalf("WriteScanline: %v", err)
	}
	if err := tif.WriteScanline([]byte{99, 99}, 1); err != nil {
		t.Fatalf("WriteScanline: %v", err)
	}
	tif.Close()

	readTif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open read: %v", err)
	}
	defer readTif.Close()

	numDirs := readTif.NumberOfDirectories()
	if numDirs < 2 {
		t.Errorf("NumberOfDirectories = %d, want >= 2", numDirs)
	}
}

// --- RATIONAL precision via GetFieldFloat ---

func TestRationalPrecision(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rational.tif")

	res1 := 72.0
	res2 := 300000.0 / 1000.0

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
	tif.SetFieldFloat(TagXResolution, res1)
	tif.SetFieldFloat(TagYResolution, res2)

	if err := tif.WriteScanline([]byte{0}, 0); err != nil {
		t.Fatalf("WriteScanline: %v", err)
	}
	tif.Close()

	readTif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open read: %v", err)
	}
	defer readTif.Close()

	// RATIONAL tags are read as float by libtiff
	xRes, err := readTif.GetFieldFloat(TagXResolution)
	if err != nil {
		t.Fatalf("GetFieldFloat XResolution: %v", err)
	}
	if math.Abs(xRes-res1) > 1e-6 {
		t.Errorf("XResolution = %f, want %f", xRes, res1)
	}

	yRes, err := readTif.GetFieldFloat(TagYResolution)
	if err != nil {
		t.Fatalf("GetFieldFloat YResolution: %v", err)
	}
	if math.Abs(yRes-res2) > 1e-3 {
		t.Errorf("YResolution = %f, want %f (RATIONAL precision lost)", yRes, res2)
	}
}

// --- String tag round-trip ---

func TestStringTagRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "strings.tif")

	testDate := "2025:01:15 12:30:45"
	testDesc := "Test image description"
	testCopyright := "Copyright 2025 Test"
	testArtist := "Test Artist"

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
	tif.SetFieldString(TagDateTime, testDate)
	tif.SetFieldString(TagImageDescription, testDesc)
	tif.SetFieldString(TagCopyright, testCopyright)
	tif.SetFieldString(TagArtist, testArtist)

	if err := tif.WriteScanline([]byte{0}, 0); err != nil {
		t.Fatalf("WriteScanline: %v", err)
	}
	tif.Close()

	readTif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open read: %v", err)
	}
	defer readTif.Close()

	checkString := func(tag Tag, want string) {
		got, err := readTif.GetFieldString(tag)
		if err != nil {
			t.Errorf("GetFieldString(%d): %v", tag, err)
			return
		}
		if got != want {
			t.Errorf("tag %d: got %q, want %q", tag, got, want)
		}
	}

	checkString(TagDateTime, testDate)
	checkString(TagImageDescription, testDesc)
	checkString(TagCopyright, testCopyright)
	checkString(TagArtist, testArtist)
}

// --- Field introspection ---

func TestFieldIntrospection(t *testing.T) {
	tif := openFixture(t, "rgb-3c-8b.tiff")
	defer tif.Close()

	dt := tif.GetFieldType(TagImageWidth)
	if dt != DataTypeLong {
		t.Errorf("GetFieldType(ImageWidth) = %d, want %d", dt, DataTypeLong)
	}

	dt = tif.GetFieldType(TagCompression)
	if dt != DataTypeShort {
		t.Errorf("GetFieldType(Compression) = %d, want %d", dt, DataTypeShort)
	}

	dt = tif.GetFieldType(TagImageDescription)
	if dt != DataTypeASCII {
		t.Errorf("GetFieldType(ImageDescription) = %d, want %d", dt, DataTypeASCII)
	}

	if !tif.IsFieldKnown(TagImageWidth) {
		t.Error("IsFieldKnown(ImageWidth) = false, want true")
	}
	if tif.IsFieldKnown(Tag(99999)) {
		t.Error("IsFieldKnown(99999) = true, want false")
	}

	wc := tif.FieldWriteCount(TagImageWidth)
	if wc != 1 {
		t.Errorf("FieldWriteCount(ImageWidth) = %d, want 1", wc)
	}

	pc := tif.FieldPassCount(TagImageWidth)
	if pc {
		t.Error("FieldPassCount(ImageWidth) = true, want false")
	}
}

// --- ReadRGBAImage buffer validation ---

func TestReadRGBAImageBufferTooSmall(t *testing.T) {
	path := filepath.Join("testdata", "rgb-3c-8b.tiff")
	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	w, _ := tif.Width()
	h, _ := tif.Height()

	// buffer smaller than required
	smallBuf := make([]uint32, int(w)*int(h)-1)
	err = tif.ReadRGBAImage(smallBuf)
	if err == nil {
		t.Fatal("expected error for too-small buffer")
	}

	// exact-size buffer should work
	okBuf := make([]uint32, int(w)*int(h))
	if err := tif.ReadRGBAImage(okBuf); err != nil {
		t.Fatalf("ReadRGBAImage with correct buffer: %v", err)
	}
}

// --- H2: GetFieldByteSlice ---

func TestGetFieldByteSliceRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "xmp_test.tiff")

	tif, err := Open(path, OpenWrite)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	w, h := uint32(4), uint32(4)
	tif.SetFieldUint32(TagImageWidth, w)
	tif.SetFieldUint32(TagImageLength, h)
	tif.SetFieldUint16(TagBitsPerSample, 8)
	tif.SetFieldUint16(TagSamplesPerPixel, 1)
	tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
	tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
	tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))

	// write custom XMP data
	xmpData := []byte("<?xpacket?><x:xmpmeta>test</x:xmpmeta><?xpacket?>")
	tif.SetFieldByteSlice(TagXMP, xmpData)

	for row := range h {
		scanline := make([]byte, w)
		if err := tif.WriteScanline(scanline, row); err != nil {
			t.Fatalf("WriteScanline %d: %v", row, err)
		}
	}

	if err := tif.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// verify XMP data round-trip
	readTif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open for read: %v", err)
	}
	defer readTif.Close()

	got, err := readTif.GetFieldByteSlice(TagXMP)
	if err != nil {
		t.Fatalf("GetFieldByteSlice: %v", err)
	}
	if !bytes.Equal(got, xmpData) {
		t.Errorf("XMP mismatch: got %q, want %q", got, xmpData)
	}
}

// --- H3: UnsetField ---

func TestUnsetField(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unset_test.tiff")

	tif, err := Open(path, OpenWrite)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	w, h := uint32(8), uint32(8)
	tif.SetFieldUint32(TagImageWidth, w)
	tif.SetFieldUint32(TagImageLength, h)
	tif.SetFieldUint16(TagBitsPerSample, 8)
	tif.SetFieldUint16(TagSamplesPerPixel, 1)
	tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
	tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
	tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))

	// set Artist tag
	tif.SetFieldString(TagArtist, "test author")

	// write pixel data
	for row := range h {
		scanline := make([]byte, w)
		if err := tif.WriteScanline(scanline, row); err != nil {
			t.Fatalf("WriteScanline %d: %v", row, err)
		}
	}

	// remove Artist tag
	if err := tif.UnsetField(TagArtist); err != nil {
		t.Fatalf("UnsetField: %v", err)
	}

	if err := tif.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// verify file is readable and Artist is gone
	readTif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open for read: %v", err)
	}
	defer readTif.Close()

	rw, _ := readTif.Width()
	if rw != w {
		t.Errorf("Width = %d, want %d", rw, w)
	}

	val, err := readTif.GetFieldString(TagArtist)
	if err == nil && val != "" {
		t.Errorf("Artist should be unset, got %q", val)
	}
}

// --- H4: ReadRGBAStrip / ReadRGBATile ---

// --- Close concurrency safety ---

func TestCloseConcurrent(t *testing.T) {
	t.Parallel()
	path := filepath.Join("testdata", "rgb-3c-8b.tiff")
	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			_ = tif.Close()
		})
	}
	wg.Wait()
}

// --- C2: ClientIO GC safety ---

func TestClientIOGC(t *testing.T) {
	t.Parallel()
	path := filepath.Join("testdata", "rgb-3c-8b.tiff")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	tif, err := OpenFromBuffer(data)
	if err != nil {
		t.Fatalf("OpenFromBuffer: %v", err)
	}

	// force GC to verify MapFileProc buffer is not collected
	runtime.GC()

	scanline := make([]byte, tif.ScanlineSize())
	if err := tif.ReadScanline(scanline, 0); err != nil {
		t.Fatalf("ReadScanline after GC: %v", err)
	}

	w, _ := tif.Width()
	if len(scanline) != int(w*3) {
		t.Errorf("scanline size = %d, want %d", len(scanline), w*3)
	}

	if err := tif.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// --- ReadEXIFDirectory ---

func TestReadEXIFDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom_dir_test.tiff")
	tif, err := Open(path, OpenWrite)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	tif.SetFieldUint32(TagImageWidth, 8)
	tif.SetFieldUint32(TagImageLength, 8)
	tif.SetFieldUint16(TagBitsPerSample, 8)
	tif.SetFieldUint16(TagSamplesPerPixel, 1)
	tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
	tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
	tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))
	for row := range uint32(8) {
		if err := tif.WriteScanline(make([]byte, 8), row); err != nil {
			t.Fatalf("WriteScanline %d: %v", row, err)
		}
	}
	if err := tif.WriteDirectory(); err != nil {
		t.Fatalf("WriteDirectory: %v", err)
	}
	if err := tif.CreateEXIFDirectory(); err != nil {
		t.Fatalf("CreateEXIFDirectory: %v", err)
	}
	// Use an EXIF-specific tag (ExifDateTimeOriginal = 36867)
	tif.SetFieldString(TagExifDateTimeOriginal, "2024:01:01 12:00:00")
	if err := tif.CheckpointDirectory(); err != nil {
		t.Fatalf("CheckpointDirectory: %v", err)
	}
	subOffset := tif.CurrentDirOffset()
	if err := tif.SetDirectory(0); err != nil {
		t.Fatalf("SetDirectory: %v", err)
	}
	if err := tif.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Read back using ReadEXIFDirectory
	readTif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open for read: %v", err)
	}
	defer readTif.Close()

	if err := readTif.ReadEXIFDirectory(subOffset); err != nil {
		t.Fatalf("ReadEXIFDirectory: %v", err)
	}
	dt, err := readTif.GetFieldString(TagExifDateTimeOriginal)
	if err != nil {
		t.Fatalf("GetFieldString: %v", err)
	}
	if dt != "2024:01:01 12:00:00" {
		t.Errorf("ExifDateTimeOriginal = %q, want %q", dt, "2024:01:01 12:00:00")
	}
}

// --- ReadGPSDirectory ---

func TestReadGPSDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gps_dir_test.tiff")
	tif, err := Open(path, OpenWrite)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	tif.SetFieldUint32(TagImageWidth, 4)
	tif.SetFieldUint32(TagImageLength, 4)
	tif.SetFieldUint16(TagBitsPerSample, 8)
	tif.SetFieldUint16(TagSamplesPerPixel, 1)
	tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
	tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
	tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))
	for row := range uint32(4) {
		scanline := make([]byte, 4)
		if err := tif.WriteScanline(scanline, row); err != nil {
			t.Fatalf("WriteScanline %d: %v", row, err)
		}
	}
	if err := tif.WriteDirectory(); err != nil {
		t.Fatalf("WriteDirectory: %v", err)
	}
	if err := tif.CreateGPSDirectory(); err != nil {
		t.Fatalf("CreateGPSDirectory: %v", err)
	}
	// GPSTAG_LATITUDEREF = 1 (ASCII type in GPS field array)
	tif.SetFieldString(Tag(1), "N")
	if err := tif.CheckpointDirectory(); err != nil {
		t.Fatalf("CheckpointDirectory: %v", err)
	}
	gpsOffset := tif.CurrentDirOffset()
	if err := tif.SetDirectory(0); err != nil {
		t.Fatalf("SetDirectory: %v", err)
	}
	tif.SetFieldUint32(TagGPSIFD, uint32(gpsOffset))
	if err := tif.RewriteDirectory(); err != nil {
		t.Fatalf("RewriteDirectory: %v", err)
	}
	if err := tif.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Read back GPS directory
	readTif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open for read: %v", err)
	}
	defer readTif.Close()

	gpsIFD, err := readTif.GetFieldUint32(TagGPSIFD)
	if err != nil {
		t.Fatalf("GetFieldUint32 GPSIFD: %v", err)
	}
	if err := readTif.ReadGPSDirectory(uint64(gpsIFD)); err != nil {
		t.Fatalf("ReadGPSDirectory: %v", err)
	}
	latRef, err := readTif.GetFieldString(Tag(1))
	if err != nil {
		t.Fatalf("GetFieldString GPS LatitudeRef: %v", err)
	}
	if latRef != "N" {
		t.Errorf("LatitudeRef = %q, want %q", latRef, "N")
	}
}

// --- CreateDirectory ---

func TestCreateDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "create_dir_test.tiff")
	tif, err := Open(path, OpenWrite)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	// Page 1
	tif.SetFieldUint32(TagImageWidth, 4)
	tif.SetFieldUint32(TagImageLength, 4)
	tif.SetFieldUint16(TagBitsPerSample, 8)
	tif.SetFieldUint16(TagSamplesPerPixel, 1)
	tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
	tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
	tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))
	for row := range uint32(4) {
		if err := tif.WriteScanline(make([]byte, 4), row); err != nil {
			t.Fatalf("WriteScanline p1 %d: %v", row, err)
		}
	}
	if err := tif.WriteDirectory(); err != nil {
		t.Fatalf("WriteDirectory: %v", err)
	}
	if err := tif.CreateDirectory(); err != nil {
		t.Fatalf("CreateDirectory: %v", err)
	}
	// Page 2 with full required fields
	tif.SetFieldUint32(TagImageWidth, 2)
	tif.SetFieldUint32(TagImageLength, 2)
	tif.SetFieldUint16(TagBitsPerSample, 8)
	tif.SetFieldUint16(TagSamplesPerPixel, 1)
	tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
	tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
	tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))
	for row := range uint32(2) {
		if err := tif.WriteScanline(make([]byte, 2), row); err != nil {
			t.Fatalf("WriteScanline p2 %d: %v", row, err)
		}
	}
	if err := tif.WriteDirectory(); err != nil {
		t.Fatalf("WriteDirectory p2: %v", err)
	}
	if err := tif.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	readTif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open for read: %v", err)
	}
	defer readTif.Close()
	if readTif.NumberOfDirectories() != 2 {
		t.Errorf("NumberOfDirectories = %d, want 2", readTif.NumberOfDirectories())
	}
}

// --- RewriteDirectory ---

func TestRewriteDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rewrite_test.tiff")
	tif, err := Open(path, OpenWrite)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	tif.SetFieldUint32(TagImageWidth, 4)
	tif.SetFieldUint32(TagImageLength, 4)
	tif.SetFieldUint16(TagBitsPerSample, 8)
	tif.SetFieldUint16(TagSamplesPerPixel, 1)
	tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
	tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
	tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))
	for row := range uint32(4) {
		if err := tif.WriteScanline(make([]byte, 4), row); err != nil {
			t.Fatalf("WriteScanline %d: %v", row, err)
		}
	}
	tif.SetFieldString(TagArtist, "initial")
	if err := tif.WriteDirectory(); err != nil {
		t.Fatalf("WriteDirectory: %v", err)
	}
	// Go back to the first directory, modify a tag, then rewrite
	if err := tif.SetDirectory(0); err != nil {
		t.Fatalf("SetDirectory: %v", err)
	}
	tif.SetFieldString(TagArtist, "rewritten")
	if err := tif.RewriteDirectory(); err != nil {
		t.Fatalf("RewriteDirectory: %v", err)
	}
	if err := tif.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	readTif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open for read: %v", err)
	}
	defer readTif.Close()
	artist, err := readTif.GetFieldString(TagArtist)
	if err != nil {
		t.Fatalf("GetFieldString: %v", err)
	}
	if artist != "rewritten" {
		t.Errorf("Artist = %q, want %q", artist, "rewritten")
	}
}

// --- UnlinkDirectory ---

func TestUnlinkDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unlink_test.tiff")
	tif, err := Open(path, OpenWrite)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	// Page 1
	tif.SetFieldUint32(TagImageWidth, 4)
	tif.SetFieldUint32(TagImageLength, 4)
	tif.SetFieldUint16(TagBitsPerSample, 8)
	tif.SetFieldUint16(TagSamplesPerPixel, 1)
	tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
	tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
	tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))
	for row := range uint32(4) {
		if err := tif.WriteScanline(make([]byte, 4), row); err != nil {
			t.Fatalf("WriteScanline p1 %d: %v", row, err)
		}
	}
	if err := tif.WriteDirectory(); err != nil {
		t.Fatalf("WriteDirectory p1: %v", err)
	}
	// Page 2
	tif.SetFieldUint32(TagImageWidth, 4)
	tif.SetFieldUint32(TagImageLength, 4)
	tif.SetFieldUint16(TagBitsPerSample, 8)
	tif.SetFieldUint16(TagSamplesPerPixel, 1)
	tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
	tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
	tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))
	for row := range uint32(4) {
		if err := tif.WriteScanline(make([]byte, 4), row); err != nil {
			t.Fatalf("WriteScanline p2 %d: %v", row, err)
		}
	}
	if err := tif.WriteDirectory(); err != nil {
		t.Fatalf("WriteDirectory p2: %v", err)
	}
	if err := tif.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Unlink page 1 (index 1, libtiff uses 1-based)
	readTif, err := Open(path, OpenAppend)
	if err != nil {
		t.Fatalf("Open append: %v", err)
	}
	if err := readTif.UnlinkDirectory(1); err != nil {
		t.Fatalf("UnlinkDirectory: %v", err)
	}
	if err := readTif.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	readTif2, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open for read: %v", err)
	}
	defer readTif2.Close()
	if readTif2.NumberOfDirectories() != 1 {
		t.Errorf("NumberOfDirectories = %d, want 1", readTif2.NumberOfDirectories())
	}
}

// --- Tag enumeration ---

func TestTagListEnumeration(t *testing.T) {
	t.Parallel()
	path := filepath.Join("testdata", "rgb-3c-8b.tiff")
	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	count := tif.TagListCount()
	if count <= 0 {
		t.Fatalf("TagListCount = %d, expected > 0", count)
	}

	// Verify iteration works without panic and returns valid tag numbers
	for i := range count {
		tag := tif.TagListEntry(i)
		if tag == 0 {
			t.Errorf("TagListEntry(%d) returned 0", i)
		}
	}
}

// --- GetFieldDefaulted ---

func TestGetFieldDefaulted(t *testing.T) {
	t.Parallel()
	path := filepath.Join("testdata", "rgb-3c-8b.tiff")
	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	// Compression should always have a value (or default to 1)
	comp, err := tif.GetFieldDefaultedUint16(TagCompression)
	if err != nil {
		t.Fatalf("GetFieldDefaultedUint16 Compression: %v", err)
	}
	if comp != uint16(CompressionNone) {
		t.Errorf("Compression = %d, want %d", comp, CompressionNone)
	}
}

// --- Strile access ---

func TestStrileAccess(t *testing.T) {
	t.Parallel()
	path := filepath.Join("testdata", "rgb-3c-8b.tiff")
	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	nStrips := tif.NumberOfStrips()
	if nStrips == 0 {
		t.Fatal("NumberOfStrips = 0")
	}
	for i := range nStrips {
		offset := tif.StrileOffset(i)
		if offset == 0 {
			t.Errorf("StrileOffset(%d) = 0", i)
		}
		count := tif.StrileByteCount(i)
		if count == 0 {
			t.Errorf("StrileByteCount(%d) = 0", i)
		}
	}

	// WithErr variants
	offset, err := tif.StrileOffsetWithErr(0)
	if err != nil {
		t.Fatalf("StrileOffsetWithErr: %v", err)
	}
	if offset == 0 {
		t.Error("StrileOffsetWithErr(0) = 0")
	}
}

// --- ReadRawTile / WriteRawTile ---

func TestReadRawTile(t *testing.T) {
	t.Parallel()
	path := filepath.Join("testdata", "quad-tile.jpg.tiff")
	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	if !tif.IsTiled() {
		t.Skip("not a tiled image")
	}

	rawSize := tif.TileSize()
	buf := make([]byte, rawSize)
	n, err := tif.ReadRawTile(0, buf)
	if err != nil {
		t.Fatalf("ReadRawTile: %v", err)
	}
	if n <= 0 {
		t.Error("ReadRawTile returned 0 bytes")
	}
}

// --- TileRowSize ---

func TestTileRowSize(t *testing.T) {
	t.Parallel()
	path := filepath.Join("testdata", "quad-tile.jpg.tiff")
	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	rowSize := tif.TileRowSize()
	if rowSize <= 0 {
		t.Errorf("TileRowSize = %d, expected > 0", rowSize)
	}
}

// --- ReadFromUserBuffer ---

func TestReadFromUserBuffer(t *testing.T) {
	path := filepath.Join("testdata", "rgb-3c-8b.tiff")
	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	if tif.NumberOfStrips() == 0 {
		t.Fatal("no strips")
	}

	// Read raw strip data
	rawSize := tif.StripSize()
	rawBuf := make([]byte, rawSize)
	n, err := tif.ReadRawStrip(0, rawBuf, rawSize)
	if err != nil {
		t.Fatalf("ReadRawStrip: %v", err)
	}

	// Decode using ReadFromUserBuffer
	outBuf := make([]byte, tif.StripSize())
	if err := tif.ReadFromUserBuffer(0, rawBuf[:n], outBuf); err != nil {
		t.Fatalf("ReadFromUserBuffer: %v", err)
	}
}

// --- ReadRGBAImageOriented ---

func TestReadRGBAImageOriented(t *testing.T) {
	path := filepath.Join("testdata", "rgb-3c-8b.tiff")
	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	w, _ := tif.Width()
	h, _ := tif.Height()
	buf := make([]uint32, int(w)*int(h))

	if err := tif.ReadRGBAImageOriented(buf, OrientationTopLeft, false); err != nil {
		t.Fatalf("ReadRGBAImageOriented: %v", err)
	}

	hasNonZero := false
	for _, px := range buf {
		if px != 0 {
			hasNonZero = true
			break
		}
	}
	if !hasNonZero {
		t.Error("all RGBA pixels are zero")
	}
}

// --- ReadRGBAStripExt / ReadRGBATileExt ---

// --- FlushData ---

func TestFlushData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "flush_data_test.tiff")
	tif, err := Open(path, OpenWrite)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	tif.SetFieldUint32(TagImageWidth, 4)
	tif.SetFieldUint32(TagImageLength, 4)
	tif.SetFieldUint16(TagBitsPerSample, 8)
	tif.SetFieldUint16(TagSamplesPerPixel, 1)
	tif.SetFieldUint16(TagCompression, uint16(CompressionNone))
	tif.SetFieldUint16(TagPhotometric, uint16(PhotometricMinIsBlack))
	tif.SetFieldUint16(TagPlanarConfig, uint16(PlanarConfigContig))
	for row := range uint32(4) {
		if err := tif.WriteScanline(make([]byte, 4), row); err != nil {
			t.Fatalf("WriteScanline %d: %v", row, err)
		}
	}
	if err := tif.FlushData(); err != nil {
		t.Fatalf("FlushData: %v", err)
	}
	if err := tif.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Verify file is valid
	readTif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open for read: %v", err)
	}
	defer readTif.Close()
	w, _ := readTif.Width()
	if w != 4 {
		t.Errorf("Width = %d, want 4", w)
	}
}

// --- Size queries ---

func TestRawStripSize(t *testing.T) {
	t.Parallel()
	path := filepath.Join("testdata", "rgb-3c-8b.tiff")
	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	numStrips := tif.NumberOfStrips()
	for i := range numStrips {
		size := tif.RawStripSize(i)
		if size <= 0 {
			t.Errorf("RawStripSize(%d) = %d, want > 0", i, size)
		}
	}
}

func TestRasterScanlineSize(t *testing.T) {
	t.Parallel()
	path := filepath.Join("testdata", "rgb-3c-8b.tiff")
	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	w, _ := tif.Width()
	spp, _ := tif.SamplesPerPixel()
	bps, _ := tif.BitsPerSample()
	expected := int(w) * int(spp) * int((bps+7)/8)
	rsz := tif.RasterScanlineSize()
	if rsz != expected {
		t.Errorf("RasterScanlineSize = %d, want %d", rsz, expected)
	}
	if rsz != tif.ScanlineSize() {
		t.Errorf("RasterScanlineSize(%d) != ScanlineSize(%d)", rsz, tif.ScanlineSize())
	}
}

func TestVStripSize(t *testing.T) {
	t.Parallel()
	path := filepath.Join("testdata", "rgb-3c-8b.tiff")
	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	rps, _ := tif.RowsPerStrip()
	vsz := tif.VStripSize(rps)
	ssz := tif.StripSize()
	if vsz != ssz {
		t.Errorf("VStripSize(RowsPerStrip) = %d, StripSize = %d", vsz, ssz)
	}
	if tif.VStripSize(1) != tif.ScanlineSize() {
		t.Errorf("VStripSize(1) = %d, ScanlineSize = %d", tif.VStripSize(1), tif.ScanlineSize())
	}
}

// --- Tile coordinate operations ---

func TestComputeStrip(t *testing.T) {
	t.Parallel()
	path := filepath.Join("testdata", "rgb-3c-8b.tiff")
	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	s := tif.ComputeStrip(0, 0)
	if s != 0 {
		t.Errorf("ComputeStrip(0, 0) = %d, want 0", s)
	}
}

func TestComputeTile(t *testing.T) {
	t.Parallel()
	path := filepath.Join("testdata", "quad-tile.jpg.tiff")
	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	if !tif.IsTiled() {
		t.Skip("test file is not tiled")
	}

	tile := tif.ComputeTile(0, 0, 0, 0)
	if tile != 0 {
		t.Errorf("ComputeTile(0,0,0,0) = %d, want 0", tile)
	}
}

func TestReadTileWriteTile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tile_coord_test.tiff")
	tileW, tileH := uint32(16), uint32(16)
	imgW, imgH := uint32(16), uint32(16)

	tif, err := Open(path, OpenWrite)
	if err != nil {
		t.Fatalf("Open: %v", err)
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

	tileData := make([]byte, tileW*tileH)
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

	readBuf := make([]byte, readTif.TileSize())
	n, err := readTif.ReadTile(0, 0, 0, 0, readBuf)
	if err != nil {
		t.Fatalf("ReadTile(0,0,0,0): %v", err)
	}
	if n != int(tileW*tileH) {
		t.Errorf("ReadTile returned %d bytes, want %d", n, tileW*tileH)
	}

	encBuf := make([]byte, readTif.TileSize())
	n2, _ := readTif.ReadEncodedTile(0, encBuf, -1)
	if n != n2 || !bytes.Equal(readBuf[:n], encBuf[:n2]) {
		t.Error("ReadTile and ReadEncodedTile produced different results")
	}
}

func TestVTileSize(t *testing.T) {
	path := filepath.Join("testdata", "quad-tile.jpg.tiff")
	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	if !tif.IsTiled() {
		t.Skip("test file is not tiled")
	}

	th, _ := tif.TileLength()
	if tif.VTileSize(th) != tif.TileSize() {
		t.Errorf("VTileSize(TileLength) = %d, TileSize = %d", tif.VTileSize(th), tif.TileSize())
	}
}

func TestDefaultTileSize(t *testing.T) {
	t.Parallel()
	path := filepath.Join("testdata", "rgb-3c-8b.tiff")
	tif, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tif.Close()

	tw, th := tif.DefaultTileSize()
	if tw == 0 || th == 0 {
		t.Errorf("DefaultTileSize = %dx%d, want non-zero", tw, th)
	}
	if tw%16 != 0 || th%16 != 0 {
		t.Errorf("DefaultTileSize = %dx%d, not multiples of 16", tw, th)
	}
}

func TestSetFieldAny(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "set_any.tif")

	tif, err := Open(path, OpenWrite)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	tests := []struct {
		tag   Tag
		value any
	}{
		{TagImageWidth, uint32(8)},
		{TagImageLength, uint32(4)},
		{TagBitsPerSample, uint16(16)},
		{TagSamplesPerPixel, uint16(3)},
		{TagCompression, uint16(CompressionNone)},
		{TagPhotometric, uint16(PhotometricRGB)},
		{TagPlanarConfig, uint16(PlanarConfigContig)},
	}
	for _, tc := range tests {
		if err := tif.SetFieldAny(tc.tag, tc.value); err != nil {
			t.Fatalf("SetFieldAny(%d): %v", tc.tag, err)
		}
	}
// Write minimal pixel data to satisfy TIFF requirements
	scanline := make([]byte, 8*3*2)
	for row := range uint32(4) {
		tif.WriteScanline(scanline, row)
	}
	tif.Close()

	// Verify round-trip
	tif2, err := Open(path, OpenRead)
	if err != nil {
		t.Fatalf("Open read: %v", err)
	}
	defer tif2.Close()
	if v, err := tif2.GetFieldUint32(TagImageWidth); err != nil || v != 8 {
		t.Errorf("ImageWidth = %d, want 8 (err=%v)", v, err)
	}
	if v, err := tif2.GetFieldUint16(TagBitsPerSample); err != nil || v != 16 {
		t.Errorf("BitsPerSample = %d, want 16 (err=%v)", v, err)
	}
}

func TestGetVersion(t *testing.T) {
	t.Parallel()
	v := GetVersion()
	if v == "" {
		t.Fatal("GetVersion returned empty string")
	}
	if !bytes.Contains([]byte(v), []byte("LIBTIFF")) {
		t.Errorf("GetVersion = %q, want to contain LIBTIFF", v)
	}
}
