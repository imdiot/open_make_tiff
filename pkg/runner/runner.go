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

	"open-make-tiff/pkg/dcrawemu"
	"open-make-tiff/pkg/dngconverter"
	"open-make-tiff/pkg/icc"
	"open-make-tiff/pkg/tiffcp"
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
	srcPath, err := filepath.Abs(srcPath)
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

	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	var returnErr error
	var (
		token         string
		logPath       string
		dngIntPath    string
		dngLinearPath string
		tiffIntPath   string
		tiffFinalPath string
	)

	defer func() {
		for _, f := range []string{dngIntPath, dngLinearPath, tiffIntPath, tiffFinalPath} {
			if f != "" {
				_ = os.Remove(f)
			}
		}
		if returnErr != nil {
			_ = os.Remove(dstPath)
		}
	}()

	hasNonASCII := runtime.GOOS == "windows" && !isASCII(name)

	for {
		u := uuid.New()
		token = hex.EncodeToString(u[:])

		prefix := base
		if hasNonASCII {
			prefix = "omt"
		}
		logPath = filepath.Join(dstDir, fmt.Sprintf("%s_%s.log", prefix, token))
		dngIntPath = filepath.Join(dstDir, fmt.Sprintf("%s_%s.int.dng", prefix, token))
		dngLinearPath = filepath.Join(dstDir, fmt.Sprintf("%s_%s.linear.dng", prefix, token))
		tiffIntPath = filepath.Join(dstDir, fmt.Sprintf("%s_%s.int.tiff", prefix, token))
		tiffFinalPath = filepath.Join(dstDir, fmt.Sprintf("%s_%s.tiff", prefix, token))

		conflict := slices.ContainsFunc(
			[]string{logPath, dngIntPath, dngLinearPath, tiffIntPath, tiffFinalPath},
			func(f string) bool {
				_, err := os.Stat(f)
				return err == nil || !errors.Is(err, os.ErrNotExist)
			},
		)
		if !conflict {
			break
		}
	}

	f, err := os.Create(logPath)
	if err != nil {
		return err
	}

	defer func() {
		if returnErr != nil {
			r.logger.Error(returnErr.Error())
		}
		_ = f.Close()
		if returnErr == nil && !r.cfg.DisableRemoveLog {
			_ = os.Remove(logPath)
		}
	}()
	r.logger = slog.New(slog.NewTextHandler(f, nil))

	r.logger.Info("src", "path", srcPath)
	r.logger.Info("dst tiff", "path", dstPath)
	r.logger.Info("dng int", "path", dngIntPath)
	r.logger.Info("dng linear", "path", dngLinearPath)
	r.logger.Info("tiff int", "path", tiffIntPath)
	r.logger.Info("tiff final", "path", tiffFinalPath)

	tiffcpExec, err := util.GetTiffcpExecutable()
	if err != nil {
		returnErr = err
		return returnErr
	}
	tiffcpOpts := []tiffcp.Option{
		tiffcp.WithExecutable(tiffcpExec),
		tiffcp.WithCommaSeparator("%"),
		tiffcp.WithFormatSpecifier("%0"),
	}
	if r.cfg.EnableCompression {
		tiffcpOpts = append(tiffcpOpts, tiffcp.WithLZWCompression(2))
	}

	if strings.ToLower(ext) == ".fff" {
		now := time.Now()
		tiffConv, err := tiffcp.New(tiffcpOpts...)
		if err != nil {
			returnErr = err
			return returnErr
		}
		err = tiffConv.Convert(ctx, srcPath, tiffIntPath)
		r.logger.Info("run tiffcp", "time", time.Since(now).Seconds())
		if err != nil {
			returnErr = err
			return returnErr
		}
	} else {
		rawPath := srcPath
		var err error
		if r.cfg.EnableAdobeDNGConverter && util.EnableAdobeDNGConverter() {
			now := time.Now()
			var dngConv2 *dngconverter.Converter
			dngConv1, err := dngconverter.New(
				dngconverter.WithUncompressed(true),
				dngconverter.WithPreviewSize(dngconverter.PreviewNone),
				dngconverter.WithCameraRawCompat(dngconverter.CameraRaw54),
				dngconverter.WithOutputDir(dstDir),
				dngconverter.WithOutputFilename(filepath.Base(dngIntPath)),
				dngconverter.WithLogger(r.logger),
			)
			if err != nil {
				r.logger.Warn("create raw converter failed: " + err.Error())
				goto afterDNG
			}

			err = dngConv1.Convert(ctx, srcPath)
			r.logger.Info("dng converter (raw)", "time", time.Since(now).Seconds())
			if err != nil {
				r.logger.Warn("raw stage failed: " + err.Error())
				goto afterDNG
			}

			now = time.Now()
			dngConv2, err = dngconverter.New(
				dngconverter.WithUncompressed(true),
				dngconverter.WithLinear(true),
				dngconverter.WithPreviewSize(dngconverter.PreviewNone),
				dngconverter.WithDNGVersion(dngconverter.DNG11),
				dngconverter.WithOutputDir(dstDir),
				dngconverter.WithOutputFilename(filepath.Base(dngLinearPath)),
				dngconverter.WithLogger(r.logger),
			)
			if err != nil {
				r.logger.Warn("create linear converter failed: " + err.Error())
				_ = os.Remove(dngIntPath)
				goto afterDNG
			}

			err = dngConv2.Convert(ctx, dngIntPath)
			r.logger.Info("dng converter (linear)", "time", time.Since(now).Seconds())
			if err != nil {
				r.logger.Warn("linear stage failed: " + err.Error())
				_ = os.Remove(dngIntPath)
				goto afterDNG
			}

			_ = os.Remove(dngIntPath)
			rawPath = dngLinearPath
		afterDNG:
		} else if hasNonASCII {
			now := time.Now()
			if err := r.copyFile(srcPath, dngIntPath); err != nil {
				returnErr = err
				return returnErr
			}
			r.logger.Info("copy raw file", "time", time.Since(now).Seconds())
			rawPath = dngIntPath
		}

		dcrawExec, err := util.GetDcrawEmuExecutable()
		if err != nil {
			returnErr = err
			return returnErr
		}

		now := time.Now()
		var stderr bytes.Buffer
		dcrawOpts := []dcrawemu.Option{
			dcrawemu.WithExecutable(dcrawExec),
			dcrawemu.WithTIFFOutput(),
			dcrawemu.WithCustomWhiteBalance(1, 1, 1, 1),
			dcrawemu.WithOutputColorSpace(dcrawemu.ColorSpaceRaw),
			dcrawemu.WithFlip(dcrawemu.FlipNone),
			dcrawemu.WithHighlightMode(dcrawemu.HighlightUnclip),
			dcrawemu.WithLinear16Bit(),
			dcrawemu.WithOutputFile(filepath.Base(tiffIntPath)),
			dcrawemu.WithWorkingDir(dstDir),
			dcrawemu.WithStderr(&stderr),
			dcrawemu.WithCheckStderr(true),
		}

		dcrawConv, err := dcrawemu.New(dcrawOpts...)
		if err != nil {
			returnErr = err
			return returnErr
		}

		if err := dcrawConv.Convert(ctx, rawPath); err != nil {
			returnErr = err
			return returnErr
		}
		r.logger.Info("run dcraw_emu", "time", time.Since(now).Seconds())

		now = time.Now()
		tiffConv, err := tiffcp.New(tiffcpOpts...)
		if err != nil {
			returnErr = err
			return returnErr
		}
		if err := tiffConv.Convert(ctx, tiffIntPath, tiffFinalPath); err != nil {
			returnErr = err
			return returnErr
		}
		r.logger.Info("run tiffcp", "time", time.Since(now).Seconds())
		_ = os.Remove(tiffIntPath)
	}

	now := time.Now()
	if err := r.runCopyExifAndInsertIccProfile(ctx, srcPath, tiffFinalPath, r.cfg.Profile); err != nil {
		returnErr = err
		return returnErr
	}
	r.logger.Info("copy exif and insert icc profile", "time", time.Since(now).Seconds())

	if err := os.Rename(tiffFinalPath, dstPath); err != nil {
		returnErr = err
		return returnErr
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
