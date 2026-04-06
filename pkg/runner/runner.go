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
	DPI                     int
}

type Option func(*Runner)

func WithRemoveIntermediate() Option {
	return func(r *Runner) {
		r.removeIntermediate = true
	}
}

func WithExiftool(et *exiftool.Exiftool) Option {
	return func(r *Runner) {
		r.et = et
	}
}

type Runner struct {
	cfg                Config
	logger             *slog.Logger
	removeIntermediate bool
	et                 *exiftool.Exiftool
}

type ConvertEnv struct {
	SrcPath       string
	DstDir        string
	DngIntPrePath string
	DngIntPath    string
	TiffIntPath   string
}

type MetadataConfig struct {
	RawPath       string
	SecondSrcPath string
	TiffPath      string
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
		dngIntPrePath string
		dngIntPath    string
		tiffIntPath   string
	)

	defer func() {
		if r.removeIntermediate {
			for _, f := range []string{dngIntPrePath, dngIntPath, tiffIntPath} {
				if f != "" {
					_ = os.Remove(f)
				}
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
		dngIntPrePath = filepath.Join(dstDir, fmt.Sprintf("%s_%s.int_pre.dng", base, token))
		dngIntPath = filepath.Join(dstDir, fmt.Sprintf("%s_%s.int.dng", base, token))
		tiffIntPath = filepath.Join(dstDir, fmt.Sprintf("%s_%s.int.tiff", base, token))

		conflict := slices.ContainsFunc(
			[]string{logPath, dngIntPrePath, dngIntPath, tiffIntPath},
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
		if returnErr == nil && r.removeIntermediate {
			_ = os.Remove(logPath)
		}
	}()
	r.logger = slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelDebug}))

	env := ConvertEnv{
		SrcPath:       srcPath,
		DstDir:        dstDir,
		DngIntPrePath: dngIntPrePath,
		DngIntPath:    dngIntPath,
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

	// Non-raw FFF: tiffcopy already wrote the TIFF, still need exiftool rebuild
	if isNonRawFFF {
		now := time.Now()
		if err := r.rebuildMetadata(MetadataConfig{
			RawPath:       srcPath,
			SecondSrcPath: srcPath,
			TiffPath:      tiffIntPath,
		}); err != nil {
			returnErr = err
			return returnErr
		}
		r.logger.Info("rebuild metadata", "time", time.Since(now).Seconds())

		if err := os.Rename(tiffIntPath, dstPath); err != nil {
			returnErr = err
			return returnErr
		}
		return nil
	}

	// Extract metadata before conversion (read-only ExifTool call)
	// For direct path: metadata is extracted from srcPath itself
	var meta *ExtractedMetadata
	if !useDNG {
		meta, err = r.extractMetadata(srcPath, srcPath)
		if err != nil {
			r.logger.Warn("extract metadata failed, proceeding without metadata", "error", err)
		}
	}

	if useDNG {
		if err := r.convertTiffWithDNG(ctx, env, meta); err != nil {
			r.logger.Warn("DNG converter path failed, falling back to direct: " + err.Error())
			useDNG = false
			// Re-extract metadata for direct path (secondSrcPath = srcPath)
			if meta, err = r.extractMetadata(srcPath, srcPath); err != nil {
				r.logger.Warn("extract metadata retry failed", "error", err)
			}
		}
	}

	if !useDNG {
		if err := r.convertTiffDirect(ctx, env, meta); err != nil {
			returnErr = err
			return returnErr
		}
	}

	if err := os.Rename(tiffIntPath, dstPath); err != nil {
		returnErr = err
		return returnErr
	}

	return nil
}

func (r *Runner) convertTiffWithDNG(ctx context.Context, env ConvertEnv, meta *ExtractedMetadata) error {
	dngConv1, err := dngconverter.New(
		dngconverter.WithUncompressed(true),
		dngconverter.WithPreviewSize(dngconverter.PreviewNone),
		dngconverter.WithCameraRawCompat(dngconverter.CameraRaw54),
		dngconverter.WithOutputDir(env.DstDir),
		dngconverter.WithOutputFilename(filepath.Base(env.DngIntPrePath)),
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
		dngconverter.WithOutputFilename(filepath.Base(env.DngIntPath)),
		dngconverter.WithLogger(r.logger),
	)
	if err != nil {
		_ = os.Remove(env.DngIntPrePath)
		return fmt.Errorf("dng converter (linear): %w", err)
	}

	now = time.Now()
	if err := dngConv2.Convert(ctx, env.DngIntPrePath); err != nil {
		_ = os.Remove(env.DngIntPrePath)
		return fmt.Errorf("dng converter (linear) convert: %w", err)
	}
	r.logger.Info("dng converter (linear)", "time", time.Since(now).Seconds())
	if r.removeIntermediate {
		_ = os.Remove(env.DngIntPrePath)
	}

	// Extract metadata after DNG conversion (DNG file now exists)
	if r.et != nil {
		m, err := r.extractMetadata(env.SrcPath, env.DngIntPath)
		if err != nil {
			r.logger.Warn("extract metadata failed, proceeding without metadata", "error", err)
		} else {
			meta = m
		}
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

	cancelDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			rp.Cancel()
		case <-cancelDone:
		}
	}()
	defer close(cancelDone)

	now = time.Now()
	if err := rp.OpenFile(env.DngIntPath); err != nil {
		return err
	}

	// Apply crop from DNG DefaultCrop/UserCrop metadata
	if _, err := rp.AdjustToRawInsetCrop(golibraw.InsetCropAllMask, 0.0); err != nil {
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
	if err := r.writeMemImageToTIFF(env.TiffIntPath, img, meta); err != nil {
		return err
	}
	r.logger.Info("run golibraw (with DNG)", "time", time.Since(now).Seconds())

	return nil
}

func (r *Runner) convertTiffDirect(ctx context.Context, env ConvertEnv, meta *ExtractedMetadata) error {
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
		golibraw.WithRawOptions(
			golibraw.RawOptDNGAddEnhanced|
				golibraw.RawOptDNGPreferLargestImage|
				golibraw.RawOptDNGAllowSizeChange|
				golibraw.RawOptDNGStage2IfPresent|
				golibraw.RawOptDNGStage3IfPresent,
		),
	)
	if err != nil {
		return err
	}
	defer rp.Close()

	cancelDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			rp.Cancel()
		case <-cancelDone:
		}
	}()
	defer close(cancelDone)

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
	if err := r.writeMemImageToTIFF(env.TiffIntPath, img, meta); err != nil {
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

func (r *Runner) rebuildMetadata(mc MetadataConfig) error {
	if r.et == nil {
		return errors.New("exiftool not available")
	}

	dpi := r.cfg.DPI
	if dpi == 0 {
		dpi = 300
	}

	args := []string{"-overwrite_original"}

	args = append(args, "-all=")
	args = append(args, "-TagsFromFile", mc.RawPath, "-All")
	args = append(args, "-XMP:all=", "-All:ImageDescription=")

	// Override DNG-specific tags from the intermediate DNG for accurate color metadata
	args = append(args, "-TagsFromFile", mc.SecondSrcPath,
		"-AsShotNeutral",
		"-UniqueCameraModel",
		"-LocalizedCameraModel",
		"-XMP-aux:All",
		"-XMP-exifEX:All",
		"-XMP-dc:Subject",
		"-XMP-lr:HierarchicalSubject",
		"-XMP-mwg-kw:All",
	)
	args = append(args, "-IPTC:all=", "-ICC_Profile:all=", "-All:Colorspace=")

	if profile, ok := icc.Profiles[r.cfg.Profile]; ok {
		iccPath := mc.TiffPath + ".icc"
		if err := os.WriteFile(iccPath, profile.Data(), 0644); err != nil {
			return fmt.Errorf("write icc profile: %w", err)
		}
		defer os.Remove(iccPath)
		args = append(args, "-ICC_Profile<="+iccPath)
	}

	args = append(args, fmt.Sprintf("-XMP-crs:RAWFileName=%s", filepath.Base(mc.RawPath)))
	args = append(args,
		fmt.Sprintf("-Xresolution=%d", dpi),
		fmt.Sprintf("-Yresolution=%d", dpi),
		"-ResolutionUnit=inches",
	)

	// Reset orientation to avoid incorrect rotation from source
	args = append(args, "-Orientation=")

	args = append(args, mc.TiffPath)

	r.logger.Info("rebuild metadata", "args", args)
	return r.et.ExecuteWrite(args...)
}

func (r *Runner) writeMemImageToTIFF(path string, img *golibraw.ProcessedImage, meta *ExtractedMetadata) error {
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

	if meta != nil {
		writeIFD0Tags(tf, meta, r.cfg)
	}

	for row := uint32(0); row < h; row++ {
		off := int64(row) * scanline
		if err := tf.WriteScanline(img.Data[off:off+scanline], row); err != nil {
			return fmt.Errorf("write scanline %d: %w", row, err)
		}
	}

	// Write EXIF Sub-IFD if metadata contains EXIF tags.
	// Use WriteDirectory (not CheckpointDirectory) to flush strip data to disk,
	// otherwise the in-memory strip buffer is lost during the EXIF Sub-IFD flow.
	if meta != nil && meta.hasEXIF() {
		if err := tf.WriteDirectory(); err != nil {
			return fmt.Errorf("write main IFD: %w", err)
		}
		if err := tf.SetDirectory(0); err != nil {
			return fmt.Errorf("set directory 0: %w", err)
		}
		if err := writeEXIFSubIFD(tf, meta); err != nil {
			return fmt.Errorf("write EXIF sub-IFD: %w", err)
		}
		return nil // writeEXIFSubIFD already calls WriteDirectory
	}

	if err := tf.WriteDirectory(); err != nil {
		return fmt.Errorf("write main IFD: %w", err)
	}

	return nil
}

func writeIFD0Tags(tf *golibtiff.TIFF, meta *ExtractedMetadata, cfg Config) {
	writeTagMap(tf, meta.RawTags, ifd0Mappings)
	if len(meta.DNGTags) > 0 {
		writeTagMap(tf, meta.DNGTags, dngOverrideMappings)
	}

	dpi := float64(cfg.DPI)
	if dpi == 0 {
		dpi = 300
	}
	tf.SetFieldFloat(golibtiff.TagXResolution, dpi)
	tf.SetFieldFloat(golibtiff.TagYResolution, dpi)
	tf.SetFieldUint16(golibtiff.TagResolutionUnit, golibtiff.ResolutionUnitInch)

	if profile, ok := icc.Profiles[cfg.Profile]; ok {
		tf.SetFieldByteSlice(golibtiff.TagIccProfile, profile.Data())
	}
	if len(meta.XMP) > 0 {
		tf.SetFieldByteSlice(golibtiff.TagXMP, meta.XMP)
	}
}

func writeEXIFSubIFD(tf *golibtiff.TIFF, meta *ExtractedMetadata) error {
	if err := tf.CreateEXIFDirectory(); err != nil {
		return fmt.Errorf("create EXIF directory: %w", err)
	}

	writeTagMap(tf, meta.RawTags, exifMappings)

	// MakerNotes (base64 decoded)
	if mn := getMakerNotes(meta.RawTags); len(mn) > 0 {
		tf.SetFieldByteSlice(golibtiff.TagExifMakerNote, mn)
	}

	offset, err := tf.WriteCustomDirectory()
	if err != nil {
		return fmt.Errorf("write custom directory: %w", err)
	}
	if err := tf.SetDirectory(0); err != nil {
		return fmt.Errorf("set directory 0: %w", err)
	}
	if err := tf.SetFieldUint64(golibtiff.TagEXIFIFD, offset); err != nil {
		return fmt.Errorf("set EXIFIFD pointer: %w", err)
	}
	if err := tf.WriteDirectory(); err != nil {
		return fmt.Errorf("rewrite main IFD: %w", err)
	}
	return nil
}
