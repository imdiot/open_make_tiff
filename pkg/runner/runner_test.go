package runner

import (
	"context"
	"testing"
)

func Test(t *testing.T) {
	r := New(Config{
		EnableAdobeDNGConverter: true,
		EnableSubfolder:         false,
		Profile:                 "sRGB",
	})
	if err := r.Run(context.Background(), "../../test/color_negative.fff"); err != nil {
		t.Error(err)
	}
}
