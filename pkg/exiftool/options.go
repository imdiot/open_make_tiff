package exiftool

import (
	"io"
	"log/slog"
	"time"
)

const defaultCloseTimeout = 100 * time.Millisecond

// Options configures Exiftool behavior.
type Options struct {
	executable   string
	Logger       *slog.Logger
	closeTimeout time.Duration
	stdout       io.Writer
	lazyInit     bool
}

// Option is a functional option for Exiftool.
type Option func(*Options)

// WithExecutable sets the exiftool binary path.
func WithExecutable(path string) Option {
	return func(o *Options) {
		o.executable = path
	}
}

// WithLogger sets the structured logger.
func WithLogger(logger *slog.Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

// WithCloseTimeout overrides the default close timeout (default 5s).
func WithCloseTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.closeTimeout = d
	}
}

// WithStdout sets the stdout writer (for debugging).
func WithStdout(w io.Writer) Option {
	return func(o *Options) {
		o.stdout = w
	}
}

// WithLazyInit defers process startup until first use.
// When enabled, New() validates the executable but does not spawn a process.
// The persistent process starts on the first call to Execute or Version.
func WithLazyInit() Option {
	return func(o *Options) {
		o.lazyInit = true
	}
}

func defaultOptions() Options {
	return Options{
		closeTimeout: defaultCloseTimeout,
	}
}

func (o *Options) logger() *slog.Logger {
	if o.Logger != nil {
		return o.Logger
	}
	return slog.Default()
}
