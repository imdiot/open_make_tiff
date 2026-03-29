package runner

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"open-make-tiff/pkg/icc"
	"open-make-tiff/pkg/util"
)

var ErrDstFileExists = errors.New("destination file already exists")

type Config struct {
	EnableAdobeDNGConverter bool
	EnableSubfolder         bool
	EnableCompression       bool
	Profile                 string

	DisableRemoveLog bool
}

type Runner struct {
	cfg    Config
	logger *slog.Logger
}

func New(cfg Config) *Runner {
	return &Runner{
		cfg:    cfg,
		logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}
}

func (r *Runner) Run(ctx context.Context, srcPath string) error {
	var err error
	srcPath, err = filepath.Abs(srcPath)
	if err != nil {
		return err
	}

	srcDir := filepath.Dir(srcPath)
	dstDir := srcDir
	if r.cfg.EnableSubfolder {
		dstDir = filepath.Join(dstDir, "make_tiff")
	}
	name := filepath.Base(srcPath)
	ext := filepath.Ext(srcPath)
	base := name[:len(name)-len(ext)]

	dstPath := filepath.Join(dstDir, fmt.Sprintf("%s.tiff", base))
	if _, err := os.Stat(dstPath); err == nil {
		return fmt.Errorf("%w: %s", ErrDstFileExists, dstPath)
	}

	var (
		token         string
		logPath       string
		dngIntPath    string
		tiffIntPath   string
		tiffFinalPath string
	)

	defer func() {
		for _, f := range []string{dngIntPath, tiffIntPath, tiffFinalPath} {
			if f != "" {
				_ = os.Remove(f)
			}
		}
		if err != nil {
			_ = os.Remove(dstPath)
		}
	}()

	hasNonASCII := runtime.GOOS == "windows" && !isASCII(name)

	for {
		u := uuid.New()
		token = hex.EncodeToString(u[:])

		if hasNonASCII {
			logPath = filepath.Join(dstDir, fmt.Sprintf("omt_%s.log", token))
			dngIntPath = filepath.Join(dstDir, fmt.Sprintf("omt_%s.int.dng", token))
			tiffIntPath = filepath.Join(dstDir, fmt.Sprintf("omt_%s.int.tiff", token))
			tiffFinalPath = filepath.Join(dstDir, fmt.Sprintf("omt_%s.tiff", token))
		} else {
			logPath = filepath.Join(dstDir, fmt.Sprintf("%s_%s.log", base, token))
			dngIntPath = filepath.Join(dstDir, fmt.Sprintf("%s_%s.int.dng", base, token))
			tiffIntPath = filepath.Join(dstDir, fmt.Sprintf("%s_%s.int.tiff", base, token))
			tiffFinalPath = filepath.Join(dstDir, fmt.Sprintf("%s_%s.tiff", base, token))
		}

		conflict := slices.ContainsFunc(
			[]string{logPath, dngIntPath, tiffIntPath, tiffFinalPath},
			func(f string) bool {
				_, err := os.Stat(f)
				return err == nil || !errors.Is(err, os.ErrNotExist)
			},
		)
		if !conflict {
			break
		}
	}

	if err = os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	f, err := os.Create(logPath)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			r.logger.Error(err.Error())
		}
		_ = f.Close()
		if err == nil && !r.cfg.DisableRemoveLog {
			_ = os.Remove(logPath)
		}
	}()
	r.logger = slog.New(slog.NewTextHandler(f, nil))

	r.logger.Info("src", "path", srcPath)
	r.logger.Info("dst tiff", "path", dstPath)
	r.logger.Info("dng int", "path", dngIntPath)
	r.logger.Info("tiff int", "path", tiffIntPath)
	r.logger.Info("tiff final", "path", tiffFinalPath)

	if strings.ToLower(ext) == ".fff" {
		now := time.Now()
		err = r.runTiffcp(ctx, srcPath, tiffIntPath)
		r.logger.Info("runTiffcp", "time", time.Since(now).Seconds())
		if err != nil {
			return err
		}
	} else {
		rawPath := srcPath
		if r.cfg.EnableAdobeDNGConverter && util.EnableAdobeDNGConverter() {
			now := time.Now()
			err = r.runAdobeDNGConverter(ctx, srcPath, dngIntPath)
			r.logger.Info("runAdobeDNGConverter", "time", time.Since(now).Seconds())
			if err != nil {
				r.logger.Warn(err.Error())
				err = nil
			}
			rawPath = dngIntPath
		} else if hasNonASCII {
			now := time.Now()
			err = r.copyFile(srcPath, dngIntPath)
			r.logger.Info("copy raw file", "time", time.Since(now).Seconds())
			if err != nil {
				return err
			}
			rawPath = dngIntPath
		}

		now := time.Now()
		if err = r.runDcrawEmuConvert(ctx, rawPath, tiffIntPath); err != nil {
			return err
		}
		r.logger.Info("runDcrawEmuConvert", "time", time.Since(now).Seconds())
		_ = os.Remove(dngIntPath)

		now = time.Now()
		if err = r.runTiffcp(ctx, tiffIntPath, tiffFinalPath); err != nil {
			return err
		}
		r.logger.Info("runTiffcp", "time", time.Since(now).Seconds())
		_ = os.Remove(tiffIntPath)
	}

	now := time.Now()
	if err = r.runCopyExifAndInsertIccProfile(ctx, srcPath, tiffFinalPath, r.cfg.Profile); err != nil {
		return err
	}
	r.logger.Info("runCopyExifAndInsertIccProfile", "time", time.Since(now).Seconds())

	if err = os.Rename(tiffFinalPath, dstPath); err != nil {
		return err
	}

	return nil
}

func (r *Runner) runTiffcp(ctx context.Context, src string, dst string) error {
	executable, err := util.GetTiffcpExecutable()
	if err != nil {
		return err
	}

	var args []string
	if r.cfg.EnableCompression {
		args = append(args, "-c", "lzw:2")
	}
	args = append(args, "-,=%", fmt.Sprintf("%s%%0", src), dst)
	cmd := exec.CommandContext(ctx, executable, args...)
	r.logger.Info("run tiffcp", "args", cmd.Args)
	cmd.SysProcAttr = util.GetSysProcAttr()
	return cmd.Run()
}

func (r *Runner) runAdobeDNGConverter(ctx context.Context, src string, dst string) error {
	executable := util.GetAdobeDNGConverterExecutable()
	args := []string{
		"-u", "-l", "-p0",
		"-d", filepath.Dir(dst),
		"-o", filepath.Base(dst),
		src,
	}
	cmd := exec.CommandContext(ctx, executable, args...)
	r.logger.Info("run adobe dng converter", "args", cmd.Args)
	cmd.SysProcAttr = util.GetSysProcAttr()
	return cmd.Run()
}

func (r *Runner) runDcrawEmuConvert(ctx context.Context, src string, dst string) error {
	executable, err := util.GetDcrawEmuExecutable()
	if err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = dstFile.Close()
	}()

	args := []string{
		"-T", "-r", "1", "1", "1", "1", "-o", "0", "-t", "0", "-H", "1", "-4", "-Z", "-",
		filepath.Base(src),
	}
	cmd := exec.CommandContext(ctx, executable, args...)
	r.logger.Info("run dcraw_emu", "args", cmd.Args)
	cmd.SysProcAttr = util.GetSysProcAttr()
	cmd.Dir = filepath.Dir(src)
	cmd.Stdout = dstFile
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err = cmd.Run(); err != nil {
		return err
	}
	if stderr.String() != "" {
		return fmt.Errorf(stderr.String())
	}
	return nil
}

func (r *Runner) runCopyExifAndInsertIccProfile(ctx context.Context, src string, dst string, profileName string) error {
	executable, err := util.GetExiftoolExecutable()
	if err != nil {
		return err
	}

	args := []string{"-overwrite_original", "-tagsfromfile", src, "-EXIF:ALL"}
	var stdin bytes.Buffer
	profile, ok := icc.Profiles[profileName]
	if ok {
		args = append(args, "-ICC_Profile<=-", dst)
		stdin.Write(profile.Data())
	} else {
		args = append(args, "-ICC_Profile=", dst)
	}
	cmd := exec.CommandContext(ctx, executable, args...)
	r.logger.Info("run copy exif and insert icc profile", "args", cmd.Args)
	cmd.Stdin = &stdin
	cmd.SysProcAttr = util.GetSysProcAttr()
	return cmd.Run()
}

func isASCII(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func (r *Runner) copyFile(src string, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
