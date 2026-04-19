package dngconverter

import (
	"testing"
)

func TestVersion(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Skip("Adobe DNG Converter not installed")
	}

	ver, err := c.Version()
	if err != nil {
		t.Fatalf("Version() failed: %v", err)
	}
	if ver == "" {
		t.Fatal("Version() returned empty string")
	}
	t.Logf("detected version: %s", ver)

	// Verify cached result matches.
	ver2, err := c.Version()
	if err != nil {
		t.Fatalf("cached Version() failed: %v", err)
	}
	if ver2 != ver {
		t.Errorf("cached version mismatch: %s != %s", ver2, ver)
	}
}
