package convert

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"open-make-tiff/internal/config"
	"open-make-tiff/pkg/golibtiff"
)

// TestConvertIntegration runs the full pipeline on a real RAW/TIFF file named
// by OMT_TEST_RAW and checks the output is a readable TIFF. Skipped when the
// env var is unset, so the default test run is never blocked. Metadata is
// skipped (no ExifTool) since this validates decode+encode fidelity only.
func TestConvertIntegration(t *testing.T) {
	src := os.Getenv("OMT_TEST_RAW")
	if src == "" {
		t.Skip("OMT_TEST_RAW not set; skipping integration test")
	}

	// Force the direct (LibRaw) path so the test does not depend on an
	// installed Adobe DNG Converter and stays fast.
	cfg := &config.Config{
		EnableCompression:        true,
		ICCProfile:               "sRGB",
		DisableAdobeDNGConverter: true,
	}
	c := New() // no ExifTool (metadata skipped); default DNG path

	if err := c.Convert(context.Background(), src, cfg); err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	out := filepath.Join(filepath.Dir(src), filepath.Base(src)+".tiff")
	defer os.Remove(out)

	if _, err := os.Stat(out); err != nil {
		t.Fatalf("output TIFF not created: %v", err)
	}

	tf, err := golibtiff.Open(out, golibtiff.OpenRead)
	if err != nil {
		t.Fatalf("output not a readable TIFF: %v", err)
	}
	defer tf.Close()
	if w, err := tf.GetFieldUint32(golibtiff.TagImageWidth); err != nil || w == 0 {
		t.Fatalf("invalid ImageWidth: %d (%v)", w, err)
	}
}
