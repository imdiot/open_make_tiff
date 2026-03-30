package rawidentify

import (
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	_, err := New(WithExecutable("nonexistent-path"))
	if err == nil {
		t.Error("expected error for nonexistent executable")
	}
}

func TestParseBasicOutput(t *testing.T) {
	i := &Identifier{}

	tests := []struct {
		name      string
		filename  string
		output    string
		wantMake  string
		wantModel string
		wantErr   bool
	}{
		{
			name:      "Canon camera",
			filename:  "photo.cr3",
			output:    "photo.cr3 is a Canon EOS R5 image.\n",
			wantMake:  "Canon",
			wantModel: "EOS R5",
			wantErr:   false,
		},
		{
			name:      "Nikon camera",
			filename:  "photo.nef",
			output:    "photo.nef is a Nikon D850 image.\n",
			wantMake:  "Nikon",
			wantModel: "D850",
			wantErr:   false,
		},
		{
			name:     "invalid format",
			filename: "photo.raw",
			output:   "invalid output",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := i.parseBasicOutput(tt.filename, tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseBasicOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if result.Make != tt.wantMake {
					t.Errorf("Make = %v, want %v", result.Make, tt.wantMake)
				}
				if result.Model != tt.wantModel {
					t.Errorf("Model = %v, want %v", result.Model, tt.wantModel)
				}
			}
		})
	}
}

func TestBuildArgs(t *testing.T) {
	i := &Identifier{executable: "raw-identify"}

	tests := []struct {
		name     string
		opts     []Option
		inputs   []string
		wantArgs []string
		wantErr  bool
	}{
		{
			name:     "basic identification",
			inputs:   []string{"photo.raw"},
			wantArgs: []string{"photo.raw"},
		},
		{
			name:     "verbose mode",
			opts:     []Option{WithVerbose(1)},
			inputs:   []string{"photo.raw"},
			wantArgs: []string{"-v", "photo.raw"},
		},
		{
			name:     "white balance",
			opts:     []Option{WithWhiteBalance()},
			inputs:   []string{"photo.raw"},
			wantArgs: []string{"-w", "photo.raw"},
		},
		{
			name:     "unpack function with frame",
			opts:     []Option{WithUnpackFunction(true)},
			inputs:   []string{"photo.raw"},
			wantArgs: []string{"-u", "-f", "photo.raw"},
		},
		{
			name:     "size with half size",
			opts:     []Option{WithSize(true)},
			inputs:   []string{"photo.raw"},
			wantArgs: []string{"-s", "-h", "photo.raw"},
		},
		{
			name:     "embedded color matrix",
			opts:     []Option{WithEmbeddedColorMatrix(true)},
			inputs:   []string{"photo.raw"},
			wantArgs: []string{"+M", "photo.raw"},
		},
		{
			name:     "multiple inputs",
			inputs:   []string{"a.raw", "b.raw"},
			wantArgs: []string{"a.raw", "b.raw"},
		},
		{
			name:     "output file",
			opts:     []Option{WithOutputFile("output.txt")},
			inputs:   []string{"photo.raw"},
			wantArgs: []string{"-o", "output.txt", "photo.raw"},
		},
		{
			name:     "unpack function without frame",
			opts:     []Option{WithUnpackFunction(false)},
			inputs:   []string{"photo.raw"},
			wantArgs: []string{"-u", "photo.raw"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := i.mergeOptions(tt.opts)
			args, err := i.buildArgs(cfg, tt.inputs...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("buildArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !slicesEqual(args, tt.wantArgs) {
				t.Errorf("buildArgs() = %v, want %v", args, tt.wantArgs)
			}
		})
	}
}

func TestValidateOptions(t *testing.T) {
	tests := []struct {
		name        string
		opts        Options
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid options - none",
			opts:    Options{},
			wantErr: false,
		},
		{
			name:    "valid options - verbose",
			opts:    Options{modeSet: true, mode: ModeVerbose, verbose: 1},
			wantErr: false,
		},
		{
			name:    "valid options - unpack with frame",
			opts:    Options{modeSet: true, mode: ModeUnpackFunction, printFrame: true, printFrameSet: true},
			wantErr: false,
		},
		{
			name:    "valid options - size with half size",
			opts:    Options{modeSet: true, mode: ModeSize, halfSize: true, halfSizeSet: true},
			wantErr: false,
		},
		{
			name:    "valid options - embedded color",
			opts:    Options{useEmbeddedColor: true, useEmbeddedColorSet: true},
			wantErr: false,
		},
		{
			name:        "invalid - -f without -u",
			opts:        Options{printFrame: true, printFrameSet: true},
			wantErr:     true,
			errContains: "-f can only be used with -u",
		},
		{
			name:        "invalid - -f with verbose",
			opts:        Options{modeSet: true, mode: ModeVerbose, printFrame: true, printFrameSet: true},
			wantErr:     true,
			errContains: "-f can only be used with -u",
		},
		{
			name:        "invalid - -h without -s",
			opts:        Options{halfSize: true, halfSizeSet: true},
			wantErr:     true,
			errContains: "-h can only be used with -s",
		},
		{
			name:        "invalid - -h with verbose",
			opts:        Options{modeSet: true, mode: ModeVerbose, halfSize: true, halfSizeSet: true},
			wantErr:     true,
			errContains: "-h can only be used with -s",
		},
		{
			name:        "invalid - both -M and +M",
			opts:        Options{useEmbeddedColor: true, useEmbeddedColorSet: true, disableEmbeddedColor: true, disableEmbeddedColorSet: true},
			wantErr:     true,
			errContains: "-M and +M cannot be used together",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOptions(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("validateOptions() error = %v, should contain %s", err, tt.errContains)
				}
			}
		})
	}
}

func TestOutputModeString(t *testing.T) {
	tests := []struct {
		mode OutputMode
		want string
	}{
		{ModeBasic, "basic"},
		{ModeVerbose, "verbose"},
		{ModeWhiteBalance, "whitebalance"},
		{ModeUnpackFunction, "unpack"},
		{ModeSize, "size"},
		{OutputMode(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.want {
				t.Errorf("OutputMode.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
