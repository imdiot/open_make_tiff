package exiftool

import (
	"bufio"
	"context"
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

func exiftoolAvailable(t testing.TB) {
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

// newTestInstance creates a started Exiftool with auto-cleanup.
func newTestInstance(t *testing.T, opts ...Option) *Exiftool {
	t.Helper()
	exiftoolAvailable(t)
	e, err := New(opts...)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { e.Close() })
	return e
}

// newLazyInstance creates a lazy-init Exiftool with auto-cleanup.
func newLazyInstance(t *testing.T, opts ...Option) *Exiftool {
	t.Helper()
	exiftoolAvailable(t)
	opts = append([]Option{WithLazyInit()}, opts...)
	e, err := New(opts...)
	if err != nil {
		t.Fatalf("New(WithLazyInit) error = %v", err)
	}
	t.Cleanup(func() { e.Close() })
	return e
}

// ---------------------------------------------------------------------------
// Unit tests (no exiftool process needed)
// ---------------------------------------------------------------------------

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

func TestWithOptions(t *testing.T) {
	e := &Exiftool{closeTimeout: defaultCloseTimeout}
	if e.lazyInit {
		t.Error("default lazyInit should be false")
	}
	if e.closeTimeout != 5*time.Second {
		t.Errorf("default closeTimeout = %v, want 5s", e.closeTimeout)
	}

	WithLazyInit()(e)
	if !e.lazyInit {
		t.Error("WithLazyInit() did not set lazyInit to true")
	}

	WithCloseTimeout(10 * time.Second)(e)
	if e.closeTimeout != 10*time.Second {
		t.Errorf("custom closeTimeout = %v, want 10s", e.closeTimeout)
	}
}

func TestNewInvalidPath(t *testing.T) {
	_, err := New(WithExecutable("/nonexistent/exiftool"))
	if !errors.Is(err, ErrExecutableNotFound) {
		t.Errorf("New with invalid path error = %v, want ErrExecutableNotFound", err)
	}
}

func TestGetDefaultExecutablePath(t *testing.T) {
	path := GetDefaultExecutablePath()
	t.Logf("default exiftool path: %q", path)
}

// ---------------------------------------------------------------------------
// Lifecycle tests
// ---------------------------------------------------------------------------

func TestNewAndClose(t *testing.T) {
	e := newTestInstance(t)

	if _, err := e.Version(); err != nil {
		t.Errorf("Version() error = %v", err)
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

func TestDoubleClose(t *testing.T) {
	_ = newTestInstance(t)
	// t.Cleanup will also call Close, verifying double-close is safe
}

func TestNewWithCustomExecutable(t *testing.T) {
	path, err := exec.LookPath("exiftool")
	if err != nil {
		t.Skip("exiftool not in PATH")
	}

	newTestInstance(t, WithExecutable(path))
}

// ---------------------------------------------------------------------------
// Execute / Read / Write
// ---------------------------------------------------------------------------

func TestExecute(t *testing.T) {
	e := newTestInstance(t)

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

func TestReadProperty(t *testing.T) {
	e := newTestInstance(t)

	model, err := e.ReadProperty(testFile("ExifTool.jpg"), "Model")
	if err != nil {
		t.Fatalf("ReadProperty error = %v", err)
	}
	t.Logf("ExifTool.jpg Model: %q", model)
}

func TestReadMetadata(t *testing.T) {
	e := newTestInstance(t)

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

func TestWriteMetadata(t *testing.T) {
	e := newTestInstance(t)
	dst := copyToTmp(t, "ExifTool.jpg")

	err := e.WriteMetadata(dst, map[string]interface{}{
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

func TestWriteMetadataDelete(t *testing.T) {
	e := newTestInstance(t)
	dst := copyToTmp(t, "ExifTool.jpg")

	if err := e.WriteMetadata(dst, map[string]interface{}{"Comment": "temporary"}); err != nil {
		t.Fatalf("WriteMetadata error = %v", err)
	}

	// Delete via nil value
	if err := e.WriteMetadata(dst, map[string]interface{}{"Comment": nil}); err != nil {
		t.Fatalf("WriteMetadata delete error = %v", err)
	}
}

func TestCopyTags(t *testing.T) {
	e := newTestInstance(t)
	src := copyToTmp(t, "ExifTool.jpg")
	dst := copyToTmp(t, "Canon.jpg")

	err := e.WriteMetadata(src, map[string]interface{}{
		"Comment": "source comment for copy test",
	})
	if err != nil {
		t.Fatalf("WriteMetadata src error = %v", err)
	}

	if err := e.CopyTags(src, dst, []string{"Comment"}); err != nil {
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

// ---------------------------------------------------------------------------
// Format-specific metadata reads
// ---------------------------------------------------------------------------

func TestReadFormats(t *testing.T) {
	e := newTestInstance(t)
	tests := []struct {
		name    string
		file    string
		wantKey string
	}{
		{"GPS", "GPS.jpg", "GPSLatitude"},
		{"Canon", "Canon.jpg", "Make"},
		{"DNG", "DNG.dng", ""},
		{"MP3", "MP3.mp3", ""},
		{"QuickTime", "QuickTime.mov", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md, err := e.ReadMetadata(testFile(tt.file))
			if err != nil {
				t.Fatalf("ReadMetadata error = %v", err)
			}
			if len(md.Fields) == 0 {
				t.Errorf("%s: metadata is empty", tt.name)
			}
			if tt.wantKey != "" {
				if _, err := md.GetString(tt.wantKey); err != nil {
					t.Errorf("%s: key %q not found: %v", tt.name, tt.wantKey, err)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Stdin / ICC
// ---------------------------------------------------------------------------

func TestExecuteWithStdin(t *testing.T) {
	e := newTestInstance(t)
	dst := copyToTmp(t, "ExifTool.jpg")

	iccData, err := os.ReadFile(testFile("ICC_Profile.icc"))
	if err != nil {
		t.Fatalf("failed to read ICC profile: %v", err)
	}

	result, err := e.ExecuteWithStdin(t.Context(), iccData, "-ICC_Profile<=-", "-overwrite_original", dst)
	if err != nil {
		t.Fatalf("ExecuteWithStdin error = %v", err)
	}
	t.Logf("ExecuteWithStdin result: %q", result)
}

func TestICCWriteViaPath(t *testing.T) {
	e := newTestInstance(t)
	dst := copyToTmp(t, "ExifTool.jpg")
	iccPath := testFile("ICC_Profile.icc")

	resp, err := e.Execute("-ICC_Profile<="+iccPath, "-overwrite_original", dst)
	if err != nil {
		t.Fatalf("ICC write via path error = %v", err)
	}
	t.Logf("ICC write via path response: %q", resp)
}

// ---------------------------------------------------------------------------
// Concurrency
// ---------------------------------------------------------------------------

func TestConcurrentExecute(t *testing.T) {
	e := newTestInstance(t)

	const n = 10
	errCh := make(chan error, n)
	for range n {
		go func() {
			_, err := e.Execute("-ver")
			errCh <- err
		}()
	}

	for range n {
		if err := <-errCh; err != nil {
			t.Errorf("concurrent Execute error: %v", err)
		}
	}
}

// ---------------------------------------------------------------------------
// Lazy init
// ---------------------------------------------------------------------------

func TestLazyInitCloseWithoutUse(t *testing.T) {
	exiftoolAvailable(t)
	e, err := New(WithLazyInit())
	if err != nil {
		t.Fatalf("New(WithLazyInit) error = %v", err)
	}
	if err := e.Close(); err != nil {
		t.Errorf("Close() without use error = %v", err)
	}
}

func TestLazyInitExecuteTriggersStart(t *testing.T) {
	e := newLazyInstance(t)

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
		t.Fatalf("New(WithLazyInit) error = %v", err)
	}
	e.Close()

	_, err = e.Execute("-ver")
	if !errors.Is(err, ErrProcessClosed) {
		t.Errorf("Execute after close error = %v, want ErrProcessClosed", err)
	}
}

func TestLazyInitVersion(t *testing.T) {
	e := newLazyInstance(t)

	ver, err := e.Version()
	if err != nil {
		t.Fatalf("Version() error = %v", err)
	}
	if ver == "" {
		t.Error("Version() returned empty")
	}
}

func TestLazyInitVersionAfterClose(t *testing.T) {
	exiftoolAvailable(t)
	e, err := New(WithLazyInit())
	if err != nil {
		t.Fatalf("New(WithLazyInit) error = %v", err)
	}
	e.Close()

	_, err = e.Version()
	if !errors.Is(err, ErrProcessClosed) {
		t.Errorf("Version after close error = %v, want ErrProcessClosed", err)
	}
}

func TestLazyInitReadMetadata(t *testing.T) {
	e := newLazyInstance(t)

	md, err := e.ReadMetadata(testFile("ExifTool.jpg"))
	if err != nil {
		t.Fatalf("ReadMetadata error = %v", err)
	}
	if len(md.Fields) == 0 {
		t.Error("Metadata.Fields is empty")
	}
}

func TestLazyInitWriteMetadata(t *testing.T) {
	e := newLazyInstance(t)
	dst := copyToTmp(t, "ExifTool.jpg")

	err := e.WriteMetadata(dst, map[string]interface{}{"Comment": "lazy write test"})
	if err != nil {
		t.Fatalf("WriteMetadata error = %v", err)
	}
}

func TestLazyInitConcurrentStart(t *testing.T) {
	e := newLazyInstance(t)

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

// ---------------------------------------------------------------------------
// Context lifecycle
// ---------------------------------------------------------------------------

func TestContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	e := newTestInstance(t, WithContext(ctx))

	ver, err := e.Version()
	if err != nil {
		t.Fatalf("Version() before cancel error = %v", err)
	}
	t.Logf("version before cancel: %s", ver)

	// Cancel the context
	cancel()

	// Give the context watcher goroutine time to react
	time.Sleep(50 * time.Millisecond)

	// Execute should fail after context cancel
	_, err = e.Execute("-ver")
	if err == nil {
		t.Error("Execute after context cancel should return error")
	}
	t.Logf("Execute after cancel error (expected): %v", err)
}

func TestCloseWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	e := newTestInstance(t, WithContext(ctx))

	// Close should work fine with context
	if err := e.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Double close should be fine
	if err := e.Close(); err != nil {
		t.Errorf("second Close() error = %v", err)
	}
}

func TestContextCancelThenClose(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	e := newTestInstance(t, WithContext(ctx))

	cancel()
	time.Sleep(50 * time.Millisecond)

	if err := e.Close(); err != nil {
		t.Errorf("Close() after context cancel error = %v", err)
	}
}

func TestLazyInitWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	e := newLazyInstance(t, WithContext(ctx))

	ver, err := e.Version()
	if err != nil {
		t.Fatalf("Version() error = %v", err)
	}
	t.Logf("lazy + context version: %s", ver)
}

func TestLazyInitContextCanceledBeforeUse(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	e := newLazyInstance(t, WithContext(ctx))

	cancel()
	time.Sleep(50 * time.Millisecond)

	_, err := e.Execute("-ver")
	if err == nil {
		t.Error("Execute after context cancel should return error")
	}
	t.Logf("Execute after cancel error (expected): %v", err)
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkBitsPerSample_Persistent(b *testing.B) {
	exiftoolAvailable(b)

	e, err := New()
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	defer e.Close()

	file := testFile("Canon.jpg")
	b.ResetTimer()
	for range b.N {
		_, err := e.ReadProperty(file, "BitsPerSample")
		if err != nil {
			b.Fatalf("ReadProperty error = %v", err)
		}
	}
}

func BenchmarkBitsPerSample_OneShot(b *testing.B) {
	exiftoolAvailable(b)

	file := testFile("Canon.jpg")
	b.ResetTimer()
	for range b.N {
		out, err := exec.Command("exiftool", "-s3", "-BitsPerSample", file).Output()
		if err != nil {
			b.Fatalf("exec error = %v", err)
		}
		_ = strings.TrimSpace(string(out))
	}
}

