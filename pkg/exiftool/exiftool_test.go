package exiftool

import (
	"bufio"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testdataDir = "testdata"

func testFile(name string) string {
	return filepath.Join(testdataDir, name)
}

func exiftoolAvailable(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("exiftool"); err != nil {
		path := GetDefaultExecutablePath()
		if path == "" {
			t.Skip("exiftool not available, skipping integration test")
		}
	}
}

func copyToTmp(t *testing.T, name string) string {
	t.Helper()
	src := testFile(name)
	dst := filepath.Join(t.TempDir(), name)
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("failed to read %s: %v", src, err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		t.Fatalf("failed to write %s: %v", dst, err)
	}
	return dst
}

// --- Unit tests ---

func TestSplitReadyToken(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantOk   bool
		wantVals []string
	}{
		{"single", "hello{ready}\n", true, []string{"hello"}},
		{"multi", "a{ready}\nb{ready}\n", true, []string{"a", "b"}},
		{"empty", "", true, nil},
		{"no token", "hello", false, nil},
		{"empty token", "{ready}\n", true, []string{""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := bufio.NewScanner(strings.NewReader(tt.input))
			sc.Split(splitReadyToken)
			var vals []string
			for sc.Scan() {
				vals = append(vals, sc.Text())
			}
			if (sc.Err() == nil) != tt.wantOk {
				t.Errorf("scan error = %v, wantOk = %v", sc.Err(), tt.wantOk)
			}
			if tt.wantOk {
				if len(vals) != len(tt.wantVals) {
					t.Errorf("got %d values, want %d: %v", len(vals), len(tt.wantVals), vals)
					return
				}
				for i, v := range vals {
					if v != tt.wantVals[i] {
						t.Errorf("val[%d] = %q, want %q", i, v, tt.wantVals[i])
					}
				}
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input   string
		major   int
		minor   int
		wantErr bool
	}{
		{"12.15", 12, 15, false},
		{"13.0", 13, 0, false},
		{"12", 12, 0, false},
		{"", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			major, minor, err := parseVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseVersion(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if major != tt.major || minor != tt.minor {
				t.Errorf("parseVersion(%q) = %d.%d, want %d.%d", tt.input, major, minor, tt.major, tt.minor)
			}
		})
	}
}

func TestCheckVersion(t *testing.T) {
	tests := []struct {
		ver     string
		wantErr bool
	}{
		{"12.15", false},
		{"13.0", false},
		{"12.14", true},
		{"11.99", true},
	}

	for _, tt := range tests {
		t.Run(tt.ver, func(t *testing.T) {
			err := checkVersion(tt.ver)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkVersion(%q) error = %v, wantErr %v", tt.ver, err, tt.wantErr)
			}
		})
	}
}

func TestHandleWriteResponse(t *testing.T) {
	tests := []struct {
		name    string
		resp    string
		wantErr bool
	}{
		{"full success", "1 image files updated\n", false},
		{"success with prefix", "Warning: something\n1 image files updated\n", false},
		{"just error", "Error: No such file", true},
		{"empty response", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handleWriteResponse(tt.resp)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleWriteResponse(%q) err = %v, wantErr %v", tt.resp, err, tt.wantErr)
			}
		})
	}
}

// --- Integration tests ---

func TestNewInvalidPath(t *testing.T) {
	_, err := New(WithExecutable("/nonexistent/exiftool"))
	if !errors.Is(err, ErrExecutableNotFound) {
		t.Errorf("New with invalid path error = %v, want ErrExecutableNotFound", err)
	}
}

func TestNewAndClose(t *testing.T) {
	exiftoolAvailable(t)

	e, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if _, err := e.Version(); err != nil {
		t.Errorf("Version() error = %v", err)
	}

	if err := e.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestExecuteAfterClose(t *testing.T) {
	exiftoolAvailable(t)

	e, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	e.Close()

	_, err = e.Execute("-ver")
	if !errors.Is(err, ErrProcessClosed) {
		t.Errorf("Execute after close error = %v, want ErrProcessClosed", err)
	}
}

func TestExecute(t *testing.T) {
	exiftoolAvailable(t)

	e, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer e.Close()

	resp, err := e.Execute("-ver")
	if err != nil {
		t.Fatalf("Execute(-ver) error = %v", err)
	}

	resp = strings.TrimSpace(resp)
	if resp == "" {
		t.Error("Execute(-ver) returned empty response")
	}
	t.Logf("exiftool version: %s", resp)
}

// --- ExifTool.jpg: general EXIF read/write ---

func TestReadProperty_ExifToolJpg(t *testing.T) {
	exiftoolAvailable(t)

	e, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer e.Close()

	model, err := e.ReadProperty(testFile("ExifTool.jpg"), "Model")
	if err != nil {
		t.Fatalf("ReadProperty error = %v", err)
	}
	t.Logf("ExifTool.jpg Model: %q", model)
}

func TestReadMetadata_ExifToolJpg(t *testing.T) {
	exiftoolAvailable(t)

	e, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer e.Close()

	md, err := e.ReadMetadata(testFile("ExifTool.jpg"))
	if err != nil {
		t.Fatalf("ReadMetadata error = %v", err)
	}

	if md.File != testFile("ExifTool.jpg") {
		t.Errorf("Metadata.File = %q, want %q", md.File, testFile("ExifTool.jpg"))
	}
	if len(md.Fields) == 0 {
		t.Error("Metadata.Fields is empty")
	}

	model, err := md.GetString("Model")
	if err != nil {
		t.Logf("Model not found (may be expected): %v", err)
	} else {
		t.Logf("Model: %s", model)
	}
}

func TestWriteMetadata_ExifToolJpg(t *testing.T) {
	exiftoolAvailable(t)

	e, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer e.Close()

	dst := copyToTmp(t, "ExifTool.jpg")

	err = e.WriteMetadata(dst, map[string]interface{}{
		"Comment": "test comment from exiftool binding",
	})
	if err != nil {
		t.Fatalf("WriteMetadata error = %v", err)
	}

	comment, err := e.ReadProperty(dst, "Comment")
	if err != nil {
		t.Fatalf("ReadProperty after write error = %v", err)
	}
	if comment != "test comment from exiftool binding" {
		t.Errorf("Comment after write = %q, want %q", comment, "test comment from exiftool binding")
	}
}

func TestWriteMetadataDelete_ExifToolJpg(t *testing.T) {
	exiftoolAvailable(t)

	e, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer e.Close()

	dst := copyToTmp(t, "ExifTool.jpg")

	err = e.WriteMetadata(dst, map[string]interface{}{
		"Comment": "temporary",
	})
	if err != nil {
		t.Fatalf("WriteMetadata error = %v", err)
	}

	// Delete via nil value
	err = e.WriteMetadata(dst, map[string]interface{}{
		"Comment": nil,
	})
	if err != nil {
		t.Fatalf("WriteMetadata delete error = %v", err)
	}
}

// --- GPS.jpg: GPS coordinate read ---

func TestReadGPSFromGPSJpg(t *testing.T) {
	exiftoolAvailable(t)

	e, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer e.Close()

	md, err := e.ReadMetadata(testFile("GPS.jpg"))
	if err != nil {
		t.Fatalf("ReadMetadata error = %v", err)
	}

	lat, err := md.GetString("GPSLatitude")
	if err != nil {
		t.Fatalf("GetString(GPSLatitude) error = %v", err)
	}
	if lat == "" {
		t.Error("GPSLatitude is empty")
	}
	t.Logf("GPS.jpg Latitude: %s", lat)

	lon, err := md.GetString("GPSLongitude")
	if err != nil {
		t.Fatalf("GetString(GPSLongitude) error = %v", err)
	}
	if lon == "" {
		t.Error("GPSLongitude is empty")
	}
	t.Logf("GPS.jpg Longitude: %s", lon)
}

// --- Canon.jpg: camera maker info ---

func TestReadCameraInfoFromCanonJpg(t *testing.T) {
	exiftoolAvailable(t)

	e, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer e.Close()

	md, err := e.ReadMetadata(testFile("Canon.jpg"))
	if err != nil {
		t.Fatalf("ReadMetadata error = %v", err)
	}

	make, err := md.GetString("Make")
	if err != nil {
		t.Fatalf("GetString(Make) error = %v", err)
	}
	t.Logf("Canon.jpg Make: %s", make)
}

// --- DNG.dng: RAW metadata ---

func TestReadDNGMetadata(t *testing.T) {
	exiftoolAvailable(t)

	e, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer e.Close()

	md, err := e.ReadMetadata(testFile("DNG.dng"))
	if err != nil {
		t.Fatalf("ReadMetadata error = %v", err)
	}
	if len(md.Fields) == 0 {
		t.Error("DNG metadata is empty")
	}

	make, err := md.GetString("Make")
	if err != nil {
		t.Logf("DNG Make not found: %v", err)
	} else {
		t.Logf("DNG.dng Make: %s", make)
	}
}

// --- MP3.mp3: audio metadata ---

func TestReadMP3Metadata(t *testing.T) {
	exiftoolAvailable(t)

	e, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer e.Close()

	md, err := e.ReadMetadata(testFile("MP3.mp3"))
	if err != nil {
		t.Fatalf("ReadMetadata error = %v", err)
	}
	if len(md.Fields) == 0 {
		t.Error("MP3 metadata is empty")
	}
	t.Logf("MP3.mp3 fields count: %d", len(md.Fields))
}

// --- QuickTime.mov: video metadata ---

func TestReadQuickTimeMetadata(t *testing.T) {
	exiftoolAvailable(t)

	e, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer e.Close()

	md, err := e.ReadMetadata(testFile("QuickTime.mov"))
	if err != nil {
		t.Fatalf("ReadMetadata error = %v", err)
	}
	if len(md.Fields) == 0 {
		t.Error("QuickTime metadata is empty")
	}
	t.Logf("QuickTime.mov fields count: %d", len(md.Fields))
}

// --- CopyTags ---

func TestCopyTags(t *testing.T) {
	exiftoolAvailable(t)

	e, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer e.Close()

	src := copyToTmp(t, "ExifTool.jpg")
	dst := copyToTmp(t, "Canon.jpg")

	err = e.WriteMetadata(src, map[string]interface{}{
		"Comment": "source comment for copy test",
	})
	if err != nil {
		t.Fatalf("WriteMetadata src error = %v", err)
	}

	err = e.CopyTags(src, dst, []string{"Comment"})
	if err != nil {
		t.Fatalf("CopyTags error = %v", err)
	}

	comment, err := e.ReadProperty(dst, "Comment")
	if err != nil {
		t.Fatalf("ReadProperty after copy error = %v", err)
	}
	if comment != "source comment for copy test" {
		t.Errorf("Comment after copy = %q, want %q", comment, "source comment for copy test")
	}
}

// --- ExecuteWithStdin ---

func TestExecuteWithStdin(t *testing.T) {
	exiftoolAvailable(t)

	e, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer e.Close()

	dst := copyToTmp(t, "ExifTool.jpg")
	iccData, err := os.ReadFile(testFile("ICC_Profile.icc"))
	if err != nil {
		t.Fatalf("failed to read ICC profile: %v", err)
	}

	ctx := t.Context()
	result, err := e.ExecuteWithStdin(ctx, iccData, "-ICC_Profile<=-", "-overwrite_original", dst)
	if err != nil {
		t.Fatalf("ExecuteWithStdin error = %v", err)
	}
	t.Logf("ExecuteWithStdin result: %q", result)

	got, err := e.ReadProperty(dst, "ICC_Profile")
	if err != nil {
		t.Logf("ICC_Profile read error (may need -b flag): %v", err)
	} else {
		t.Logf("ICC_Profile written successfully, size: %d chars", len(got))
	}
}

// --- ICC profile write via path ---

func TestICCWriteViaPath(t *testing.T) {
	exiftoolAvailable(t)

	e, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer e.Close()

	dst := copyToTmp(t, "ExifTool.jpg")
	iccPath := testFile("ICC_Profile.icc")

	resp, err := e.Execute("-ICC_Profile<=" + iccPath, "-overwrite_original", dst)
	if err != nil {
		t.Fatalf("ICC write via path error = %v", err)
	}
	t.Logf("ICC write via path response: %q", resp)
}

// --- Concurrent execution ---

func TestConcurrentExecute(t *testing.T) {
	exiftoolAvailable(t)

	e, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer e.Close()

	const n = 10
	errCh := make(chan error, n)
	for i := range n {
		go func() {
			_, err := e.Execute("-ver")
			errCh <- err
		}()
		_ = i
	}

	for range n {
		if err := <-errCh; err != nil {
			t.Errorf("concurrent Execute error: %v", err)
		}
	}
}

// --- Options ---

func TestGetDefaultExecutablePath(t *testing.T) {
	path := GetDefaultExecutablePath()
	t.Logf("default exiftool path: %q", path)
}

func TestWithCloseTimeout(t *testing.T) {
	cfg := defaultOptions()
	if cfg.closeTimeout != 5*time.Second {
		t.Errorf("default closeTimeout = %v, want 5s", cfg.closeTimeout)
	}

	WithCloseTimeout(10 * time.Second)(&cfg)
	if cfg.closeTimeout != 10*time.Second {
		t.Errorf("custom closeTimeout = %v, want 10s", cfg.closeTimeout)
	}
}

func TestNewWithCustomExecutable(t *testing.T) {
	exiftoolAvailable(t)

	path, err := exec.LookPath("exiftool")
	if err != nil {
		t.Skip("exiftool not in PATH")
	}

	e, err := New(WithExecutable(path))
	if err != nil {
		t.Fatalf("New(WithExecutable) error = %v", err)
	}
	defer e.Close()

	if _, err := e.Version(); err != nil {
		t.Errorf("Version() error = %v", err)
	}
}

// --- Lazy init ---

func TestLazyInitOption(t *testing.T) {
	cfg := defaultOptions()
	if cfg.lazyInit {
		t.Error("default lazyInit should be false")
	}

	WithLazyInit()(&cfg)
	if !cfg.lazyInit {
		t.Error("WithLazyInit() did not set lazyInit to true")
	}
}

func TestLazyInitCloseWithoutUse(t *testing.T) {
	exiftoolAvailable(t)

	e, err := New(WithLazyInit())
	if err != nil {
		t.Fatalf("New(WithLazyInit()) error = %v", err)
	}
	if err := e.Close(); err != nil {
		t.Errorf("Close() without use error = %v", err)
	}
}

func TestLazyInitExecuteTriggersStart(t *testing.T) {
	exiftoolAvailable(t)

	e, err := New(WithLazyInit())
	if err != nil {
		t.Fatalf("New(WithLazyInit()) error = %v", err)
	}
	defer e.Close()

	resp, err := e.Execute("-ver")
	if err != nil {
		t.Fatalf("Execute(-ver) error = %v", err)
	}
	if strings.TrimSpace(resp) == "" {
		t.Error("Execute(-ver) returned empty")
	}
}

func TestLazyInitExecuteAfterClose(t *testing.T) {
	exiftoolAvailable(t)

	e, err := New(WithLazyInit())
	if err != nil {
		t.Fatalf("New(WithLazyInit()) error = %v", err)
	}
	e.Close()

	_, err = e.Execute("-ver")
	if !errors.Is(err, ErrProcessClosed) {
		t.Errorf("Execute after close error = %v, want ErrProcessClosed", err)
	}
}

func TestLazyInitVersionTriggersStart(t *testing.T) {
	exiftoolAvailable(t)

	e, err := New(WithLazyInit())
	if err != nil {
		t.Fatalf("New(WithLazyInit()) error = %v", err)
	}
	defer e.Close()

	ver, err := e.Version()
	if err != nil {
		t.Fatalf("Version() error = %v", err)
	}
	if ver == "" {
		t.Error("Version() returned empty, expected lazy start to populate it")
	}
}

func TestLazyInitVersionAfterClose(t *testing.T) {
	exiftoolAvailable(t)

	e, err := New(WithLazyInit())
	if err != nil {
		t.Fatalf("New(WithLazyInit()) error = %v", err)
	}
	e.Close()

	_, err = e.Version()
	if !errors.Is(err, ErrProcessClosed) {
		t.Errorf("Version after close error = %v, want ErrProcessClosed", err)
	}
}

func TestLazyInitReadMetadata(t *testing.T) {
	exiftoolAvailable(t)

	e, err := New(WithLazyInit())
	if err != nil {
		t.Fatalf("New(WithLazyInit()) error = %v", err)
	}
	defer e.Close()

	md, err := e.ReadMetadata(testFile("ExifTool.jpg"))
	if err != nil {
		t.Fatalf("ReadMetadata error = %v", err)
	}
	if len(md.Fields) == 0 {
		t.Error("Metadata.Fields is empty")
	}
}

func TestLazyInitWriteMetadata(t *testing.T) {
	exiftoolAvailable(t)

	e, err := New(WithLazyInit())
	if err != nil {
		t.Fatalf("New(WithLazyInit()) error = %v", err)
	}
	defer e.Close()

	dst := copyToTmp(t, "ExifTool.jpg")
	err = e.WriteMetadata(dst, map[string]interface{}{
		"Comment": "lazy write test",
	})
	if err != nil {
		t.Fatalf("WriteMetadata error = %v", err)
	}
}

func TestLazyInitConcurrentStart(t *testing.T) {
	exiftoolAvailable(t)

	e, err := New(WithLazyInit())
	if err != nil {
		t.Fatalf("New(WithLazyInit()) error = %v", err)
	}
	defer e.Close()

	const n = 10
	errCh := make(chan error, n)
	for range n {
		go func() {
			_, err := e.Execute("-ver")
			errCh <- err
		}()
	}

	for i := range n {
		if err := <-errCh; err != nil {
			t.Errorf("concurrent Execute %d error: %v", i, err)
		}
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
