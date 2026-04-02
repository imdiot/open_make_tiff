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

	"open-make-tiff/pkg/dngconverter"
	"open-make-tiff/pkg/golibraw"
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

type ConvertEnv struct {
	SrcPath       string
	DstDir        string
	DngIntPath    string
	DngLinearPath string
	TiffIntPath   string
	HasNonASCII   bool
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
	r.logger = slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelDebug}))

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
		tiffcp.WithLogger(r.logger),
	}
	if r.cfg.EnableCompression {
		tiffcpOpts = append(tiffcpOpts, tiffcp.WithLZWCompression(2))
	}

	env := ConvertEnv{
		SrcPath:       srcPath,
		DstDir:        dstDir,
		DngIntPath:    dngIntPath,
		DngLinearPath: dngLinearPath,
		TiffIntPath:   tiffIntPath,
		HasNonASCII:   hasNonASCII,
	}

	useDNG := r.cfg.EnableAdobeDNGConverter
	var isNonRawFFF bool

	if strings.ToLower(ext) == ".fff" {
		rp, err := golibraw.New()
		if err != nil {
			returnErr = fmt.Errorf("golibraw init failed: %w", err)
			return returnErr
		}

		if err := rp.OpenFile(srcPath); err != nil {
			rp.Close()
			r.logger.Info("fff file is TIFF-based, using tiffcp directly", "path", srcPath)
			if err := r.convertNonRawFFF(ctx, env); err != nil {
				returnErr = err
				return returnErr
			}
			isNonRawFFF = true
		} else {
			rp.Close()
			r.logger.Info("fff file is RAW, using golibraw", "path", srcPath)
			useDNG = false
		}
	}

	if useDNG {
		if err := r.convertTiffWithDNG(ctx, env); err != nil {
			r.logger.Warn("DNG converter path failed, falling back to direct: " + err.Error())
			useDNG = false
		}
	}

	if !useDNG && !isNonRawFFF {
		if err := r.convertTiffDirect(ctx, env); err != nil {
			returnErr = err
			return returnErr
		}
	}

	now := time.Now()
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

	now = time.Now()
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

func (r *Runner) convertTiffWithDNG(ctx context.Context, env ConvertEnv) error {
	dngConv1, err := dngconverter.New(
		dngconverter.WithUncompressed(true),
		dngconverter.WithPreviewSize(dngconverter.PreviewNone),
		dngconverter.WithCameraRawCompat(dngconverter.CameraRaw54),
		dngconverter.WithOutputDir(env.DstDir),
		dngconverter.WithOutputFilename(filepath.Base(env.DngIntPath)),
		dngconverter.WithLogger(r.logger),
	)
	if err != nil {
		return fmt.Errorf("dng converter (raw): %w", err)
	}

	now := time.Now()
	if err := dngConv1.Convert(ctx, env.SrcPath); err != nil {
		return fmt.Errorf("dng converter (raw) convert: %w", err)
	}
	r.logger.Info("dng converter (raw)", "time", time.Since(now).Seconds())

	dngConv2, err := dngconverter.New(
		dngconverter.WithUncompressed(true),
		dngconverter.WithLinear(true),
		dngconverter.WithPreviewSize(dngconverter.PreviewNone),
		dngconverter.WithDNGVersion(dngconverter.DNG11),
		dngconverter.WithOutputDir(env.DstDir),
		dngconverter.WithOutputFilename(filepath.Base(env.DngLinearPath)),
		dngconverter.WithLogger(r.logger),
	)
	if err != nil {
		_ = os.Remove(env.DngIntPath)
		return fmt.Errorf("dng converter (linear): %w", err)
	}

	now = time.Now()
	if err := dngConv2.Convert(ctx, env.DngIntPath); err != nil {
		_ = os.Remove(env.DngIntPath)
		return fmt.Errorf("dng converter (linear) convert: %w", err)
	}
	r.logger.Info("dng converter (linear)", "time", time.Since(now).Seconds())
	_ = os.Remove(env.DngIntPath)
	r.logger.Info("rename linear dng to int", "from", env.DngLinearPath, "to", env.DngIntPath)
	if err := os.Rename(env.DngLinearPath, env.DngIntPath); err != nil {
		return fmt.Errorf("rename linear dng to int: %w", err)
	}

	rp, err := golibraw.New(
		golibraw.WithTIFFOutput(),
		golibraw.WithUserMul(1, 1, 1, 1),
		golibraw.WithOutputColorSpace(golibraw.ColorSpaceRaw),
		golibraw.WithFlip(golibraw.FlipNone),
		golibraw.WithHighlightMode(golibraw.HighlightUnclip),
		golibraw.With16BitOutput(),
		golibraw.WithNoAutoBrightness(),
		golibraw.WithGamma(1.0, 1.0),
		golibraw.WithAdjustMaxThreshold(0),
		golibraw.WithEmbeddedColorMatrix(false),
	)
	if err != nil {
		return err
	}
	defer rp.Close()

	now = time.Now()
	if err := rp.OpenFile(env.DngIntPath); err != nil {
		return err
	}
	if err := rp.Unpack(); err != nil {
		return err
	}
	if err := rp.Process(); err != nil {
		return err
	}
	if err := rp.WritePPMTiff(filepath.Join(env.DstDir, filepath.Base(env.TiffIntPath))); err != nil {
		return err
	}
	r.logger.Info("run golibraw (with DNG)", "time", time.Since(now).Seconds())

	return nil
}

func (r *Runner) convertTiffDirect(ctx context.Context, env ConvertEnv) error {
	srcPath := env.SrcPath
	if env.HasNonASCII {
		now := time.Now()
		if err := r.copyFile(srcPath, env.DngIntPath); err != nil {
			return err
		}
		r.logger.Info("copy raw file (non-ascii)", "time", time.Since(now).Seconds())
		srcPath = env.DngIntPath
	}

	rp, err := golibraw.New(
		golibraw.WithTIFFOutput(),
		golibraw.WithUserMul(1, 1, 1, 1),
		golibraw.WithOutputColorSpace(golibraw.ColorSpaceRaw),
		golibraw.WithFlip(golibraw.FlipNone),
		golibraw.WithHighlightMode(golibraw.HighlightUnclip),
		golibraw.With16BitOutput(),
		golibraw.WithNoAutoBrightness(),
		golibraw.WithGamma(1.0, 1.0),
		golibraw.WithAdjustMaxThreshold(0),
		golibraw.WithEmbeddedColorMatrix(false),
		golibraw.WithDNGSDK(47),
		golibraw.WithUseRawSpeed(256),
		golibraw.WithRawOptions(2560),
	)
	if err != nil {
		return err
	}
	defer rp.Close()

	if err := rp.EnableDNGSDK(); err != nil {
		return err
	}

	now := time.Now()
	if err := rp.OpenFile(srcPath); err != nil {
		return err
	}
	if err := rp.Unpack(); err != nil {
		return err
	}
	if err := rp.Process(); err != nil {
		return err
	}
	if err := rp.WritePPMTiff(filepath.Join(env.DstDir, filepath.Base(env.TiffIntPath))); err != nil {
		return err
	}
	r.logger.Info("run golibraw (direct)", "time", time.Since(now).Seconds())

	return nil
}

func (r *Runner) convertNonRawFFF(ctx context.Context, env ConvertEnv) error {
	inputPath := env.SrcPath
	if env.HasNonASCII {
		now := time.Now()
		if err := r.copyFile(env.SrcPath, env.DngIntPath); err != nil {
			return err
		}
		r.logger.Info("copy fff file (non-ascii)", "time", time.Since(now).Seconds())
		inputPath = env.DngIntPath
	}

	now := time.Now()
	tiffcpExec, err := util.GetTiffcpExecutable()
	if err != nil {
		return err
	}
	tiffConv, err := tiffcp.New(
		tiffcp.WithExecutable(tiffcpExec),
		tiffcp.WithCommaSeparator("%"),
		tiffcp.WithFormatSpecifier("%0"),
		tiffcp.WithLogger(r.logger),
	)
	if err != nil {
		return err
	}
	if err := tiffConv.Convert(ctx, inputPath, env.TiffIntPath); err != nil {
		return err
	}
	r.logger.Info("run tiffcp (TIFF-based fff to tiffIntPath)", "time", time.Since(now).Seconds())

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
