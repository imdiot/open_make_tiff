package cli

import (
	"os"
	"testing"

	"open-make-tiff/internal/build"
)

// withArgs swaps os.Args for the duration of a test; Run reads os.Args[1:].
func withArgs(t *testing.T, args ...string) {
	t.Helper()
	saved := os.Args
	os.Args = append([]string{"open-make-tiff"}, args...)
	t.Cleanup(func() { os.Args = saved })
}

// TestVersionFlag: -version prints the version and exits 0 without requiring
// input files (checked before the input-count guard).
func TestVersionFlag(t *testing.T) {
	withArgs(t, "-version")
	if code := Run(build.Info{ProductVersion: "test-version"}); code != 0 {
		t.Errorf("-version exit = %d, want 0", code)
	}
}

// TestNoInputFile: no positional args is a usage error (exit 2).
func TestNoInputFile(t *testing.T) {
	withArgs(t)
	if code := Run(build.Info{ProductVersion: "test"}); code != 2 {
		t.Errorf("no-input exit = %d, want 2", code)
	}
}

// TestBadProfile: an unknown -profile is a usage error (exit 2), checked before
// any conversion runs.
func TestBadProfile(t *testing.T) {
	withArgs(t, "-profile=does-not-exist", "some.raw")
	if code := Run(build.Info{ProductVersion: "test"}); code != 2 {
		t.Errorf("bad-profile exit = %d, want 2", code)
	}
}

// TestWorkersTooSmall: -workers < 1 is a usage error (exit 2).
func TestWorkersTooSmall(t *testing.T) {
	withArgs(t, "-workers=0", "some.raw")
	if code := Run(build.Info{ProductVersion: "test"}); code != 2 {
		t.Errorf("workers=0 exit = %d, want 2", code)
	}
}
