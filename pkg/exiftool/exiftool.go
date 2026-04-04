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
)

const (
	defaultScanBufSize = 64 * 1024
	defaultScanBufMax  = 10 * 1024 * 1024
	minExiftoolVersion = "12.15"

	writeSuccessToken = "image files updated"
)

var readyToken = []byte("{ready}")

// Exiftool manages a persistent exiftool process (-stay_open mode).
type Exiftool struct {
	mu         sync.Mutex
	executable string
	version    string
	defaults   Options
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	scanner    *bufio.Scanner
	closed     bool
	done       chan struct{} // nil=not started, open=running, closed=exited
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
	cfg := defaultOptions()
	for _, opt := range opts {
		opt(&cfg)
	}

	execPath := cmp.Or(cfg.executable, GetDefaultExecutablePath())
	if execPath == "" {
		return nil, ErrExecutableNotFound
	}
	if _, err := os.Stat(execPath); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrExecutableNotFound, execPath)
	}

	cfg.executable = execPath

	e := &Exiftool{
		executable: execPath,
		defaults:   cfg,
	}

	if !cfg.lazyInit {
		if err := e.start(); err != nil {
			return nil, err
		}
	}

	return e, nil
}

func (e *Exiftool) start() error {
	args := []string{"-stay_open", "True", "-@", "-"}

	e.cmd = exec.Command(e.executable, args...)
	e.cmd.SysProcAttr = getSysProcAttr()

	// Merge stdout/stderr to avoid dual-stream read deadlock
	pipeR, pipeW := io.Pipe()
	e.stdout = pipeR
	e.cmd.Stdout = pipeW
	e.cmd.Stderr = pipeW

	var err error
	if e.stdin, err = e.cmd.StdinPipe(); err != nil {
		return fmt.Errorf("error piping stdin: %w", err)
	}

	e.scanner = bufio.NewScanner(pipeR)
	buf := make([]byte, defaultScanBufSize)
	e.scanner.Buffer(buf, defaultScanBufMax)
	e.scanner.Split(splitReadyToken)

	if err := e.cmd.Start(); err != nil {
		return fmt.Errorf("error starting exiftool: %w", err)
	}

	// Monitor goroutine: sole caller of cmd.Wait
	e.done = make(chan struct{})
	go func() {
		e.cmd.Wait()
		close(e.done)
	}()

	// First start: version check
	if e.version == "" {
		ver, err := e.executeInner("-ver")
		if err != nil {
			e.cmd.Process.Kill()
			<-e.done
			e.reset()
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
	e.stdout = nil
	e.scanner = nil
	e.done = nil
}

// ensureStarted starts the persistent process on first use in lazy mode,
// or restarts it if the process has exited.
// Must be called with e.mu held.
func (e *Exiftool) ensureStarted() error {
	if e.isRunning() {
		return nil
	}
	if e.done != nil {
		e.reset()
	}
	return e.start()
}

// Close sends -stay_open False and waits for the process to exit.
func (e *Exiftool) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return nil
	}
	e.closed = true

	if !e.isRunning() {
		return nil
	}

	return e.forceCleanup()
}

func (e *Exiftool) forceCleanup() error {
	var errs []error

	// Best-effort close; process may have already exited
	if e.stdin != nil {
		fmt.Fprintln(e.stdin, "-stay_open")
		fmt.Fprintln(e.stdin, "False")
		fmt.Fprintln(e.stdin, "-execute")
		e.stdin.Close()
	}

	if e.stdout != nil {
		e.stdout.Close()
	}

	if e.done != nil {
		timeout := e.defaults.closeTimeout
		select {
		case <-e.done:
			// Process exited normally
		case <-time.After(timeout):
			if e.cmd != nil && e.cmd.Process != nil {
				e.cmd.Process.Kill()
			}
			<-e.done
			errs = append(errs, ErrProcessKilled)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing exiftool: %w", errors.Join(errs...))
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

// executeInner is the lock-free internal execute method.
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

	if !e.scanner.Scan() {
		if err := e.scanner.Err(); err != nil {
			return "", fmt.Errorf("error reading response: %w", err)
		}
		return "", ErrNoResponse
	}

	resp := e.scanner.Text()

	return resp, nil
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

	var results []map[string]interface{}
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
func (e *Exiftool) WriteMetadata(file string, tags map[string]interface{}) error {
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

// ExecuteWrite runs an exiftool write command and validates the response.
// Use this for commands that modify files (metadata writes, tag copies, etc.).
func (e *Exiftool) ExecuteWrite(args ...string) error {
	resp, err := e.Execute(args...)
	if err != nil {
		return err
	}
	return handleWriteResponse(resp)
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

