package runner

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"open-make-tiff/pkg/dngconverter"
	"open-make-tiff/pkg/exiftool"
	"open-make-tiff/pkg/golibraw"
	"open-make-tiff/pkg/golibtiff"
	"open-make-tiff/pkg/golibtiff/tiffcopy"
	"open-make-tiff/pkg/icc"
)

var ErrDstFileExists = errors.New("destination file already exists")

type Config struct {
	EnableAdobeDNGConverter bool
	EnableSubfolder         bool
	EnableCompression       bool
	Profile                 string
}

type Option func(*Runner)

func WithDisableRemoveLog() Option {
	return func(r *Runner) {
		r.disableRemoveLog = true
	}
}

func WithExiftool(et *exiftool.Exiftool) Option {
	return func(r *Runner) {
		r.et = et
	}
}

type Runner struct {
	cfg              Config
	logger           *slog.Logger
	disableRemoveLog bool
	et               *exiftool.Exiftool
}

type ConvertEnv struct {
	SrcPath       string
	DstDir        string
	DngIntPath    string
	DngLinearPath string
	TiffIntPath   string
}

func New(cfg Config, opts ...Option) *Runner {
	r := &Runner{
		cfg:    cfg,
		logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
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
		iccPath       string
	)

	defer func() {
		for _, f := range []string{dngIntPath, dngLinearPath, tiffIntPath, iccPath} {
			if f != "" {
				_ = os.Remove(f)
			}
		}
		if returnErr != nil {
			_ = os.Remove(dstPath)
		}
	}()

	for {
		u := uuid.New()
		token = hex.EncodeToString(u[:])

		logPath = filepath.Join(dstDir, fmt.Sprintf("%s_%s.log", base, token))
		dngIntPath = filepath.Join(dstDir, fmt.Sprintf("%s_%s.int.dng", base, token))
		dngLinearPath = filepath.Join(dstDir, fmt.Sprintf("%s_%s.linear.dng", base, token))
		tiffIntPath = filepath.Join(dstDir, fmt.Sprintf("%s_%s.int.tiff", base, token))
		iccPath = filepath.Join(dstDir, fmt.Sprintf("%s_%s.icc", base, token))

		conflict := slices.ContainsFunc(
			[]string{logPath, dngIntPath, dngLinearPath, tiffIntPath, iccPath},
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
		if returnErr == nil && !r.disableRemoveLog {
			_ = os.Remove(logPath)
		}
	}()
	r.logger = slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelDebug}))

	env := ConvertEnv{
		SrcPath:       srcPath,
		DstDir:        dstDir,
		DngIntPath:    dngIntPath,
		DngLinearPath: dngLinearPath,
		TiffIntPath:   tiffIntPath,
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
	if err := r.runCopyExifAndInsertIccProfile(srcPath, tiffIntPath, iccPath); err != nil {
		returnErr = err
		return returnErr
	}
	r.logger.Info("copy exif and insert icc profile", "time", time.Since(now).Seconds())

	if err := os.Rename(tiffIntPath, dstPath); err != nil {
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
	img, err := rp.MakeMemImage()
	if err != nil {
		return err
	}
	if err := r.writeMemImageToTIFF(env.TiffIntPath, img); err != nil {
		return err
	}
	r.logger.Info("run golibraw (with DNG)", "time", time.Since(now).Seconds())

	return nil
}

func (r *Runner) convertTiffDirect(ctx context.Context, env ConvertEnv) error {
	rp, err := golibraw.New(
		golibraw.WithUserMul(1, 1, 1, 1),
		golibraw.WithOutputColorSpace(golibraw.ColorSpaceRaw),
		golibraw.WithFlip(golibraw.FlipNone),
		golibraw.WithHighlightMode(golibraw.HighlightUnclip),
		golibraw.With16BitOutput(),
		golibraw.WithNoAutoBrightness(),
		golibraw.WithGamma(1.0, 1.0),
		golibraw.WithAdjustMaxThreshold(0),
		golibraw.WithEmbeddedColorMatrix(false),
		golibraw.WithDNGSDK(golibraw.DNGSDKDefault|golibraw.DNGSDKXTrans),
		golibraw.WithUseRawSpeed(golibraw.RawSpeedV3Use),
		golibraw.WithRawOptions(golibraw.RawOptDNGAddPreviews|golibraw.RawOptDNGPreferLargestImage),
	)
	if err != nil {
		return err
	}
	defer rp.Close()

	if err := rp.EnableDNGSDK(); err != nil {
		return err
	}

	now := time.Now()
	if err := rp.OpenFile(env.SrcPath); err != nil {
		return err
	}
	if err := rp.Unpack(); err != nil {
		return err
	}
	if err := rp.Process(); err != nil {
		return err
	}
	img, err := rp.MakeMemImage()
	if err != nil {
		return err
	}
	if err := r.writeMemImageToTIFF(env.TiffIntPath, img); err != nil {
		return err
	}
	r.logger.Info("run golibraw (direct)", "time", time.Since(now).Seconds())

	return nil
}

func (r *Runner) convertNonRawFFF(_ context.Context, env ConvertEnv) error {
	var opts []tiffcopy.Option
	if r.cfg.EnableCompression {
		opts = append(opts, tiffcopy.WithLZWCompression(golibtiff.PredictorHorizontal))
	}

	now := time.Now()
	if err := tiffcopy.Copy(env.SrcPath, env.TiffIntPath, opts...); err != nil {
		return err
	}
	r.logger.Info("run tiffcopy (TIFF-based fff)", "time", time.Since(now).Seconds())

	return nil
}

func (r *Runner) runCopyExifAndInsertIccProfile(src string, dst string, iccPath string) error {
	if r.et == nil {
		return errors.New("exiftool not available")
	}

	args := []string{"-overwrite_original", "-tagsfromfile", src, "-EXIF:ALL"}
	profile, ok := icc.Profiles[r.cfg.Profile]
	if ok {
		if err := os.WriteFile(iccPath, profile.Data(), 0644); err != nil {
			return fmt.Errorf("write icc profile: %w", err)
		}
		args = append(args, "-ICC_Profile<="+iccPath, dst)
	} else {
		args = append(args, "-ICC_Profile=", dst)
	}
	r.logger.Info("run copy exif and insert icc profile", "args", args)
	_, err := r.et.Execute(args...)
	return err
}

func (r *Runner) writeMemImageToTIFF(path string, img *golibraw.ProcessedImage) error {
	tf, err := golibtiff.Open(path, golibtiff.OpenWrite)
	if err != nil {
		return err
	}
	defer tf.Close()

	w := uint32(img.Width)
	h := uint32(img.Height)
	colors := uint16(img.Colors)
	bits := uint16(img.Bits)
	scanline := int64(w) * int64(colors) * int64(bits/8)

	tf.SetFieldUint32(golibtiff.TagImageWidth, w)
	tf.SetFieldUint32(golibtiff.TagImageLength, h)
	tf.SetFieldUint16(golibtiff.TagBitsPerSample, bits)
	tf.SetFieldUint16(golibtiff.TagSamplesPerPixel, colors)
	tf.SetFieldUint16(golibtiff.TagPhotometric, golibtiff.PhotometricRGB)
	if r.cfg.EnableCompression {
		tf.SetFieldUint16(golibtiff.TagCompression, golibtiff.CompressionLZW)
		tf.SetFieldUint16(golibtiff.TagPredictor, golibtiff.PredictorHorizontal)
	}
	tf.SetFieldUint16(golibtiff.TagPlanarConfig, golibtiff.PlanarConfigContig)

	for row := uint32(0); row < h; row++ {
		off := int64(row) * scanline
		if err := tf.WriteScanline(img.Data[off:off+scanline], row); err != nil {
			return fmt.Errorf("write scanline %d: %w", row, err)
		}
	}
	return tf.WriteDirectory()
}
