package binary

import (
	"context"
	"open-make-tiff/pkg/util"
	"testing"
)

func TestBinary(t *testing.T) {
	cfg := Config{
		Executable: util.GetAdobeDNGConverterExecutable(),
		Dir:        "",
		Args: []string{
			"-c", "-u", "-l", "-p0",
			"-d", "D:\\Work\\open-make-tiff\\test",
			"-o", "IMG_0021_test.dng",
			"D:\\Work\\open-make-tiff\\test\\IMG_0021.CR3",
		},
		Env: nil,
	}
	b := New(cfg)
	out, err := b.Run(context.Background())
	if err != nil {
		t.Error(out, err)
	}
}
