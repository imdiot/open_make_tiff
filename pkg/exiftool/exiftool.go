package exiftool

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrExecutableNotFound = errors.New("exiftool executable not found")
	ErrProcessClosed      = errors.New("exiftool process is closed")
	ErrProcessKilled      = errors.New("exiftool process killed after timeout")
	ErrVersionMismatch    = errors.New("exiftool version mismatch")
	ErrNoResponse         = errors.New("unexpected EOF from exiftool")
	ErrNoMetadata         = errors.New("no metadata returned")
	ErrContextCanceled    = errors.New("exiftool operation canceled by context")
)

const (
	defaultScanBufSize  = 64 * 1024
	defaultScanBufMax   = 10 * 1024 * 1024
	minExiftoolVersion  = "12.15"
	defaultCloseTimeout = 5 * time.Second
	writeSuccessToken   = "image files updated"
)

var readyToken = []byte("{ready}")

// Exiftool manages a persistent exiftool process (-stay_open mode).
type Exiftool struct {
	mu sync.Mutex

	executable   string
	logger       *slog.Logger
	closeTimeout time.Duration
	lazyInit     bool

	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
	version string
	closed  bool
	done    chan struct{} // nil=not started, open=running, closed=exited

	instanceCtx    context.Context
	cancelInstance context.CancelFunc
}

// Option configures Exiftool behavior.
type Option func(*Exiftool)

// WithExecutable sets the exiftool binary path.
func WithExecutable(path string) Option {
	return func(e *Exiftool) {
		e.executable = path
	}
}

// WithLogger sets the structured logger.
func WithLogger(logger *slog.Logger) Option {
	return func(e *Exiftool) {
		e.logger = logger
	}
}

// WithCloseTimeout overrides the default close timeout (default 5s).
func WithCloseTimeout(d time.Duration) Option {
	return func(e *Exiftool) {
		e.closeTimeout = d
	}
}

// WithLazyInit defers process startup until first use.
func WithLazyInit() Option {
	return func(e *Exiftool) {
		e.lazyInit = true
	}
}

// WithContext binds a context to the Exiftool instance lifecycle.
func WithContext(ctx context.Context) Option {
	return func(e *Exiftool) {
		e.instanceCtx = ctx
	}
}

// GetDefaultExecutablePath resolves the default exiftool path via exec.LookPath.
func GetDefaultExecutablePath() string {
	path, err := exec.LookPath("exiftool")
	if err != nil {
		return ""
	}
	return path
}

// New creates an Exiftool instance, starts a persistent process, and verifies the version.
func New(opts ...Option) (*Exiftool, error) {
	e := &Exiftool{closeTimeout: defaultCloseTimeout}
	for _, o := range opts {
		o(e)
	}

	if e.logger == nil {
		e.logger = slog.Default()
	}

	execPath := cmp.Or(e.executable, GetDefaultExecutablePath())
	if execPath == "" {
		return nil, ErrExecutableNotFound
	}
	if _, err := os.Stat(execPath); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrExecutableNotFound, execPath)
	}
	e.executable = execPath

	ctx := e.instanceCtx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	e.instanceCtx = ctx
	e.cancelInstance = cancel

	if !e.lazyInit {
		if err := e.start(); err != nil {
			return nil, err
		}
	}

	return e, nil
}

// Close shuts down the exiftool process and releases all resources.
//
// Phase 0: Cancel context FIRST (no mutex needed).
// This unblocks executeInner (select on instanceCtx.Done),
// which causes Execute() to release the mutex.
// cancelInstance is set once in New() and CancelFunc is goroutine-safe.
//
// Phase 1: Acquire mutex (executeInner has now returned), set closed flag.
//
// Phase 2: Wait for the process to exit (cmd.Wait closes all pipes).
func (e *Exiftool) Close() error {
	// Phase 0: cancel context FIRST — no mutex needed
	// Breaks the deadlock: executeInner waits for ctx, mutex waits for executeInner
	e.cancelInstance()

	// Phase 1: acquire mutex (executeInner has now returned)
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return nil
	}
	e.closed = true
	done := e.done
	cmd := e.cmd
	e.mu.Unlock()

	// Phase 2: wait for process to exit
	if done != nil {
		select {
		case <-done:
		case <-time.After(e.closeTimeout):
			if cmd != nil && cmd.Process != nil {
				cmd.Process.Kill()
			}
			<-done
			return ErrProcessKilled
		}
	}

	return nil
}

// Version returns the exiftool version string.
// In lazy mode, this triggers process startup if not yet started.
func (e *Exiftool) Version() (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return "", ErrProcessClosed
	}

	if err := e.ensureStarted(); err != nil {
		return "", err
	}

	return e.version, nil
}

// Execute runs arbitrary exiftool commands and returns raw response text.
func (e *Exiftool) Execute(args ...string) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return "", ErrProcessClosed
	}

	if err := e.ensureStarted(); err != nil {
		return "", err
	}

	return e.executeInner(args...)
}

// ExecuteWrite runs an exiftool write command and validates the response.
// Use this for commands that modify files (metadata writes, tag copies, etc.).
func (e *Exiftool) ExecuteWrite(args ...string) error {
	resp, err := e.Execute(args...)
	if err != nil {
		return err
	}
	return handleWriteResponse(resp)
}

// ExecuteWithStdin runs a one-shot process for commands requiring stdin input.
func (e *Exiftool) ExecuteWithStdin(ctx context.Context, stdinData []byte, args ...string) (string, error) {
	e.mu.Lock()
	execPath := e.executable
	e.mu.Unlock()

	cmd := exec.CommandContext(ctx, execPath, args...)
	cmd.SysProcAttr = getSysProcAttr()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("error creating stdin pipe: %w", err)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("error starting command: %w", err)
	}

	// Assign child to OS-level job/process-group for crash protection.
	if pid := cmd.Process.Pid; pid > 0 {
		if err := assignToJob(pid); err != nil {
			e.logger.Warn("failed to assign exiftool to job object", "pid", pid, "error", err)
		}
	}

	if _, err := stdin.Write(stdinData); err != nil {
		stdin.Close()
		return "", fmt.Errorf("error writing to stdin: %w", err)
	}
	stdin.Close()

	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("error executing command: %s: %w", stderr.String(), err)
	}

	return stdout.String(), nil
}

// ReadProperty reads a single tag value (-s3 -<tag>) as plain text.
func (e *Exiftool) ReadProperty(file string, tag string) (string, error) {
	resp, err := e.Execute("-s3", "-"+tag, file)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp), nil
}

// ReadMetadata reads full metadata (-j JSON output) as structured Metadata.
func (e *Exiftool) ReadMetadata(file string) (*Metadata, error) {
	resp, err := e.Execute("-j", file)
	if err != nil {
		return nil, err
	}

	var results []map[string]any
	if err := json.Unmarshal([]byte(resp), &results); err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %w", err)
	}
	if len(results) == 0 {
		return nil, ErrNoMetadata
	}

	return &Metadata{
		File:   file,
		Fields: results[0],
	}, nil
}

// WriteMetadata writes tags to a file.
// tags format: map[tag]value; nil value deletes the tag.
func (e *Exiftool) WriteMetadata(file string, tags map[string]any) error {
	args := []string{"-overwrite_original"}

	for k, v := range tags {
		if v == nil {
			args = append(args, "-"+k+"=")
		} else {
			args = append(args, fmt.Sprintf("-%s=%v", k, v))
		}
	}
	args = append(args, file)

	resp, err := e.Execute(args...)
	if err != nil {
		return err
	}

	return handleWriteResponse(resp)
}

// CopyTags copies specified tags from src to dst.
func (e *Exiftool) CopyTags(src, dst string, tags []string) error {
	args := []string{"-overwrite_original"}
	for _, tag := range tags {
		args = append(args, "-"+tag)
	}
	args = append(args, "-TagsFromFile", src)
	for _, tag := range tags {
		args = append(args, "-"+tag)
	}
	args = append(args, dst)

	resp, err := e.Execute(args...)
	if err != nil {
		return err
	}

	return handleWriteResponse(resp)
}

func (e *Exiftool) start() error {
	args := []string{"-stay_open", "True", "-@", "-"}

	e.cmd = exec.Command(e.executable, args...)
	e.cmd.SysProcAttr = getSysProcAttr()

	// OS-level pipe: has kernel buffer, process exit automatically sends EOF.
	// Unlike io.Pipe (synchronous, unbuffered), writes only block when the
	// kernel buffer is full, and cmd.Wait() is never blocked by an IO goroutine.
	stdoutPipe, err := e.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error creating stdout pipe: %w", err)
	}

	// Capture stderr for diagnostics during startup.
	// bytes.Buffer is synchronous (no deadlock risk unlike io.Pipe).
	// exiftool's Perl startup errors appear here, critical for diagnosing
	// why the process might exit immediately (e.g., antivirus kill).
	var stderrBuf bytes.Buffer
	e.cmd.Stderr = &stderrBuf

	if e.stdin, err = e.cmd.StdinPipe(); err != nil {
		return fmt.Errorf("error piping stdin: %w", err)
	}

	e.scanner = bufio.NewScanner(stdoutPipe)
	buf := make([]byte, defaultScanBufSize)
	e.scanner.Buffer(buf, defaultScanBufMax)
	e.scanner.Split(splitReadyToken)

	if err := e.cmd.Start(); err != nil {
		return fmt.Errorf("error starting exiftool: %w", err)
	}

	// Assign child to OS-level job/process-group for crash protection.
	if pid := e.cmd.Process.Pid; pid > 0 {
		if err := assignToJob(pid); err != nil {
			e.logger.Warn("failed to assign exiftool to job object", "pid", pid, "error", err)
		}
	}

	// Monitor goroutine: sole caller of cmd.Wait.
	// cmd.StdoutPipe() creates no IO goroutine for stdout.
	// stderr writes to bytes.Buffer (synchronous, never blocks).
	// cmd.Wait() only waits for process exit — no deadlock possible.
	e.done = make(chan struct{})
	go func() {
		e.cmd.Wait()
		close(e.done)
	}()

	// Context watcher: kill process immediately when context is canceled.
	go func() {
		select {
		case <-e.done:
			// Process exited on its own.
		case <-e.instanceCtx.Done():
			if e.cmd != nil && e.cmd.Process != nil {
				e.cmd.Process.Kill()
			}
			<-e.done
		}
	}()

	// First start: version check
	if e.version == "" {
		ver, err := e.executeInner("-ver")
		if err != nil {
			e.cmd.Process.Kill()
			<-e.done
			e.reset()
			if stderrOutput := strings.TrimSpace(stderrBuf.String()); stderrOutput != "" {
				return fmt.Errorf("error checking version: %w (stderr: %s)", err, stderrOutput)
			}
			return fmt.Errorf("error checking version: %w", err)
		}
		ver = strings.TrimSpace(ver)
		if err := checkVersion(ver); err != nil {
			e.cmd.Process.Kill()
			<-e.done
			e.reset()
			return fmt.Errorf("%w: %s", err, ver)
		}
		e.version = ver
	}

	return nil
}

// ensureStarted starts the persistent process on first use in lazy mode,
// or restarts it if the process has exited.
// Must be called with e.mu held.
func (e *Exiftool) ensureStarted() error {
	if e.isRunning() {
		return nil
	}
	if e.instanceCtx.Err() != nil {
		return ErrContextCanceled
	}
	if e.done != nil {
		e.reset()
	}
	return e.start()
}

// isRunning checks if the subprocess is alive.
// Must be called with e.mu held.
func (e *Exiftool) isRunning() bool {
	if e.done == nil {
		return false
	}
	select {
	case <-e.done:
		return false
	default:
		return true
	}
}

// reset cleans up resources from a dead process.
// Must be called with e.mu held, after done is closed.
func (e *Exiftool) reset() {
	e.cmd = nil
	e.stdin = nil
	e.scanner = nil
	e.done = nil
}

// executeInner is the lock-free internal execute method.
// It can be interrupted by instanceCtx cancellation, which is critical for
// preventing mutex deadlocks when Close() cancels the context.
func (e *Exiftool) executeInner(args ...string) (string, error) {
	var buf strings.Builder
	for _, arg := range args {
		buf.WriteString(arg)
		buf.WriteByte('\n')
	}
	buf.WriteString("-execute\n")

	if _, err := io.WriteString(e.stdin, buf.String()); err != nil {
		return "", fmt.Errorf("error writing command to stdin: %w", err)
	}

	type scanResult struct {
		text string
		err  error
	}
	resultCh := make(chan scanResult, 1)
	go func() {
		if !e.scanner.Scan() {
			if err := e.scanner.Err(); err != nil {
				resultCh <- scanResult{"", fmt.Errorf("error reading response: %w", err)}
				return
			}
			resultCh <- scanResult{"", ErrNoResponse}
			return
		}
		resultCh <- scanResult{e.scanner.Text(), nil}
	}()

	select {
	case r := <-resultCh:
		return r.text, r.err
	case <-e.instanceCtx.Done():
		return "", ErrContextCanceled
	}
}

func splitReadyToken(data []byte, atEOF bool) (int, []byte, error) {
	idx := bytes.Index(data, readyToken)
	if idx == -1 {
		if atEOF && len(data) > 0 {
			return 0, data, fmt.Errorf("no final token found in output")
		}
		return 0, nil, nil
	}

	// Skip the ready token and any trailing line ending (\r\n or \n)
	end := idx + len(readyToken)
	if end < len(data) && data[end] == '\r' {
		end++
	}
	if end < len(data) && data[end] == '\n' {
		end++
	}

	return end, data[:idx], nil
}

func handleWriteResponse(resp string) error {
	cleaned := strings.TrimSpace(resp)
	if strings.Contains(cleaned, writeSuccessToken) {
		return nil
	}
	if cleaned == "" {
		return nil
	}
	return errors.New(cleaned)
}

func parseVersion(ver string) (major, minor int, err error) {
	parts := strings.SplitN(ver, ".", 2)
	if len(parts) < 1 {
		return 0, 0, fmt.Errorf("invalid version format: %s", ver)
	}

	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid major version: %s", ver)
	}

	if len(parts) > 1 {
		minor, err = strconv.Atoi(parts[1])
		if err != nil {
			return major, 0, nil // some versions only have major
		}
	}

	return major, minor, nil
}

func checkVersion(ver string) error {
	if ver == "" {
		return fmt.Errorf("%w: empty version", ErrVersionMismatch)
	}

	minMajor, minMinor, err := parseVersion(minExiftoolVersion)
	if err != nil {
		return err
	}

	major, minor, err := parseVersion(ver)
	if err != nil {
		return fmt.Errorf("%w: cannot parse version %q", ErrVersionMismatch, ver)
	}

	if major > minMajor || (major == minMajor && minor >= minMinor) {
		return nil
	}

	return fmt.Errorf("%w: got %s, need >= %s", ErrVersionMismatch, ver, minExiftoolVersion)
}
