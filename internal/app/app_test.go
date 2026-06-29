package app

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"open-make-tiff/internal/config"
	"open-make-tiff/internal/convert"
)

// TestConvertEmptyBatch verifies NewApp wiring and the worker-pool/Progress
// plumbing without depending on CGO decoding or ExifTool.
func TestConvertEmptyBatch(t *testing.T) {
	a := NewApp(context.Background(), config.Default(), nil)
	defer a.Close()

	tp := &recordingProgress{done: make(chan struct{})}
	a.Convert(context.Background(), nil, tp)

	select {
	case <-tp.done:
	case <-time.After(time.Second):
		t.Fatal("Done not called")
	}
	if tp.starts != 1 {
		t.Errorf("Start called %d times, want 1", tp.starts)
	}
	if tp.succeeded+tp.skipped+tp.failed != 0 {
		t.Errorf("expected no file events, got succeeded=%d skipped=%d failed=%d", tp.succeeded, tp.skipped, tp.failed)
	}
}

// TestAppConvertIntegration drives the full app.Convert pipeline on a real
// RAW file (OMT_TEST_RAW). Skipped when unset. The converter is rebuilt
// without ExifTool so the test does not depend on a bundled exiftool next to
// the test binary.
func TestAppConvertIntegration(t *testing.T) {
	src := os.Getenv("OMT_TEST_RAW")
	if src == "" {
		t.Skip("OMT_TEST_RAW not set; skipping integration test")
	}

	cfg := &config.Config{
		EnableCompression:        true,
		ICCProfile:               "sRGB",
		DisableAdobeDNGConverter: true,
	}
	cfg.Validate()
	a := NewApp(context.Background(), cfg, nil)
	a.converter = convert.New(convert.WithDNGExecutable(a.dngExec)) // no ExifTool
	defer a.Close()

	tp := &recordingProgress{done: make(chan struct{})}
	a.Convert(context.Background(), []string{src}, tp)

	select {
	case <-tp.done:
	case <-time.After(5 * time.Minute):
		t.Fatal("timeout waiting for conversion")
	}
	if tp.failed != 0 {
		t.Fatalf("conversion failed: succeeded=%d skipped=%d failed=%d", tp.succeeded, tp.skipped, tp.failed)
	}

	out := filepath.Join(filepath.Dir(src), filepath.Base(src)+".tiff")
	defer os.Remove(out)
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("output TIFF not created: %v", err)
	}
}

type recordingProgress struct {
	mu                           sync.Mutex
	starts                       int
	succeeded, skipped, failed   int
	done                         chan struct{}
}

func (r *recordingProgress) Start(int) {
	r.mu.Lock()
	r.starts++
	r.mu.Unlock()
}

func (r *recordingProgress) File(_ string, result Result, _ error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch result {
	case ResultSucceeded:
		r.succeeded++
	case ResultSkipped:
		r.skipped++
	case ResultFailed:
		r.failed++
	}
}

func (r *recordingProgress) Done() { close(r.done) }
