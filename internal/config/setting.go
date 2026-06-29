package config

import (
	"cmp"
	"fmt"
	"slices"

	"open-make-tiff/pkg/dngconverter"
	"open-make-tiff/pkg/icc"
)

// WorkerNumOption is a selectable worker-count entry shown in the GUI dropdown.
type WorkerNumOption struct {
	Value int    `json:"value"`
	Label string `json:"label"`
}

// ProfileOption is a selectable ICC profile entry shown in the GUI dropdown.
type ProfileOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Setting is the aggregated option payload bound to the frontend. JSON tags
// must stay aligned with frontend wailsjs/go/models.ts.
type Setting struct {
	WorkerNums              []*WorkerNumOption `json:"worker_nums"`
	Profiles                []*ProfileOption   `json:"profiles"`
	EnableAdobeDNGConverter bool               `json:"enable_adobe_dng_converter"`
}

// NewSetting builds the selectable option lists: worker counts 1..MaxWorkers,
// ICC profiles (a leading "none" then embedded profiles sorted by value), and
// whether the Adobe DNG Converter is available on this machine.
func NewSetting() *Setting {
	s := &Setting{
		WorkerNums:              make([]*WorkerNumOption, 0),
		Profiles:                make([]*ProfileOption, 0),
		EnableAdobeDNGConverter: func() bool { _, err := dngconverter.New(); return err == nil }(),
	}
	for i := 1; i <= MaxWorkers(); i++ {
		s.WorkerNums = append(s.WorkerNums, &WorkerNumOption{Value: i, Label: fmt.Sprintf("%d", i)})
	}
	s.Profiles = append(s.Profiles, &ProfileOption{Value: "", Label: "none"})
	for k, v := range icc.Profiles {
		s.Profiles = append(s.Profiles, &ProfileOption{Value: k, Label: v.Name})
	}
	slices.SortStableFunc(s.Profiles, func(a, b *ProfileOption) int { return cmp.Compare(a.Value, b.Value) })
	return s
}
