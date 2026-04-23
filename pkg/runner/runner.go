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
	"time"

	"github.com/google/uuid"

	"open-make-tiff/pkg/dngconverter"
	"open-make-tiff/pkg/exiftool"
	"open-make-tiff/pkg/golibraw"
	"open-make-tiff/pkg/golibtiff"
	"open-make-tiff/pkg/icc"
)

var ErrDstFileExists = errors.New("destination file already exists")

var baseRawOpts = []golibraw.Option{
	golibraw.WithUserMul(1, 1, 1, 1),
	golibraw.WithOutputColorSpace(golibraw.ColorSpaceRaw),
	golibraw.WithFlip(golibraw.FlipNone),
	golibraw.WithHighlightMode(golibraw.HighlightUnclip),
	golibraw.With16BitOutput(),
	golibraw.WithNoAutoBrightness(),
	golibraw.WithGamma(1.0, 1.0),
	golibraw.WithAdjustMaxThreshold(0),
	golibraw.WithEmbeddedColorMatrix(false),
}

type DecodeType int

const (
	DecodeDirect DecodeType = iota
	DecodeDNG
	DecodeTIFF
)

type Config struct {
	EnableAdobeDNGConverter bool
	EnableSubfolder         bool
	EnableCompression       bool
	Profile                 string
	KeepLogFiles            bool
	KeepIntermediateFiles   bool
}

type ConvertEnv struct {
	SrcPath       string
	DstDir        string
	DngIntPrePath string
	DngIntPath    string
	TiffIntPath   string
}

// decodedImage holds decoded pixel data.
// Unlike golibraw.ProcessedImage (whose Width/Height are uint16 per LibRaw C API),
// decodedImage uses uint32 for dimensions to support arbitrarily large TIFF images.
type decodedImage struct {
	DecodeType DecodeType
	Width      uint32
	Height     uint32
	Colors     uint16
	Bits       uint16
	Data       []byte
	CamMul     [4]float32
}

type Option func(*Runner)

type Runner struct {
	cfg                       Config
	logger                    *slog.Logger
	keepLogFiles              bool
	keepIntermediateFiles     bool
	et                        *exiftool.Exiftool
	dngConverterExecutable    string
	dngConverterExecutableSet bool
}

func WithExiftool(et *exiftool.Exiftool) Option {
	return func(r *Runner) {
		r.et = et
	}
}

func WithDNGConverterExecutable(path string) Option {
	return func(r *Runner) {
		r.dngConverterExecutable = path
		r.dngConverterExecutableSet = true
	}
}

func New(cfg Config, opts ...Option) *Runner {
	r := &Runner{
		cfg:                   cfg,
		logger:                slog.New(slog.NewTextHandler(os.Stdout, nil)),
		keepLogFiles:          cfg.KeepLogFiles,
		keepIntermediateFiles: cfg.KeepIntermediateFiles,
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

	dstPath := filepath.Join(dstDir, fmt.Sprintf("%s.tiff", name))
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
		if !r.keepIntermediateFiles {
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

		logPath = filepath.Join(dstDir, fmt.Sprintf("%s_%s.log", name, token))
		dngIntPrePath = filepath.Join(dstDir, fmt.Sprintf("%s_%s.int_pre.dng", name, token))
		dngIntPath = filepath.Join(dstDir, fmt.Sprintf("%s_%s.int.dng", name, token))
		tiffIntPath = filepath.Join(dstDir, fmt.Sprintf("%s_%s.int.tiff", name, token))

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
		if returnErr == nil && !r.keepLogFiles {
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

	img, err := r.decode(ctx, env)
	if err != nil {
		returnErr = err
		return returnErr
	}

	if writeErr := r.writeMemImageToTIFF(env.TiffIntPath, img); writeErr != nil {
		returnErr = writeErr
		return returnErr
	}

	if r.et != nil {
		if metaErr := r.writeMetadataExiftool(env.TiffIntPath, env, img); metaErr != nil {
			returnErr = metaErr
			return returnErr
		}
	}

	if err := os.Rename(tiffIntPath, dstPath); err != nil {
		returnErr = err
		return returnErr
	}

	return nil
}

// probeFile detects whether srcPath is a readable TIFF that LibRaw cannot handle.
// Returns (false, nil) if the file appears to be RAW (LibRaw can open it).
// Returns (true, nil) if LibRaw cannot open it but libtiff can (plain TIFF).
// Returns (false, err) only if LibRaw init itself fails — a fatal condition.
//
// When both LibRaw and libtiff fail to open the file, it returns (false, nil)
// and logs a warning, allowing the caller to fall through to decodeDirect
// for a final attempt with full options (DNG SDK, RawSpeed, etc.).
//
// LibRaw is probed first because many RAW formats (CR2, NEF, ARW, DNG, etc.)
// are TIFF containers — only LibRaw can distinguish them from plain TIFF.
func (r *Runner) probeFile(srcPath string) (isTIFF bool, err error) {
	rp, probeErr := golibraw.New()
	if probeErr != nil {
		return false, fmt.Errorf("golibraw init failed: %w", probeErr)
	}
	librawOK := rp.OpenFile(srcPath) == nil
	rp.Close()

	if librawOK {
		r.logger.Info("detected RAW format", "path", srcPath)
		return false, nil
	}

	// LibRaw cannot open it — check if it is a readable TIFF.
	tf, tiffErr := golibtiff.Open(srcPath, golibtiff.OpenRead)
	if tiffErr != nil {
		r.logger.Warn("file not recognized as RAW or TIFF by probe, will attempt direct decode", "path", srcPath)
		return false, nil
	}
	tf.Close()

	r.logger.Info("detected TIFF format", "path", srcPath)
	return true, nil
}

func (r *Runner) decode(ctx context.Context, env ConvertEnv) (*decodedImage, error) {
	isTIFF, probeErr := r.probeFile(env.SrcPath)
	if probeErr != nil {
		return nil, probeErr
	}
	if isTIFF {
		img, err := r.decodeTIFF(env.SrcPath)
		if err != nil {
			return nil, err
		}
		img.DecodeType = DecodeTIFF
		return img, nil
	}

	useDNG := r.cfg.EnableAdobeDNGConverter
	if useDNG {
		var execPath string
		if r.dngConverterExecutableSet {
			execPath = r.dngConverterExecutable
		} else {
			execPath = dngconverter.GetDefaultExecutablePath()
		}
		if _, err := os.Stat(execPath); err != nil {
			r.logger.Warn("DNG Converter not available, using direct path", "error", err)
			useDNG = false
		}
	}

	if useDNG {
		img, err := r.decodeWithDNG(ctx, env)
		if err == nil {
			img.DecodeType = DecodeDNG
			return img, nil
		}
		r.logger.Warn("DNG converter path failed, falling back to direct", "error", err)
	}

	img, err := r.decodeDirect(ctx, env)
	if err != nil {
		return nil, err
	}
	img.DecodeType = DecodeDirect
	return img, nil
}

func (r *Runner) decodeWithDNG(ctx context.Context, env ConvertEnv) (*decodedImage, error) {
	start := time.Now()
	defer func() { r.logger.Info("decode DNG", "time", time.Since(start).Seconds()) }()

	dngOpts1 := []dngconverter.Option{
		dngconverter.WithUncompressed(true),
		dngconverter.WithPreviewSize(dngconverter.PreviewNone),
		dngconverter.WithCameraRawCompat(dngconverter.CameraRaw54),
		dngconverter.WithOutputDir(env.DstDir),
		dngconverter.WithOutputFilename(filepath.Base(env.DngIntPrePath)),
		dngconverter.WithLogger(r.logger),
	}
	if r.dngConverterExecutableSet {
		dngOpts1 = append(dngOpts1, dngconverter.WithExecutable(r.dngConverterExecutable))
	}

	dngConv1, err := dngconverter.New(dngOpts1...)
	if err != nil {
		return nil, fmt.Errorf("dng converter (raw): %w", err)
	}

	now := time.Now()
	if err := dngConv1.Convert(ctx, env.SrcPath); err != nil {
		return nil, fmt.Errorf("dng converter (raw) convert: %w", err)
	}
	r.logger.Info("dng converter (raw)", "time", time.Since(now).Seconds())

	dngOpts2 := []dngconverter.Option{
		dngconverter.WithUncompressed(true),
		dngconverter.WithLinear(true),
		dngconverter.WithPreviewSize(dngconverter.PreviewNone),
		dngconverter.WithDNGVersion(dngconverter.DNG11),
		dngconverter.WithOutputDir(env.DstDir),
		dngconverter.WithOutputFilename(filepath.Base(env.DngIntPath)),
		dngconverter.WithLogger(r.logger),
	}
	if r.dngConverterExecutableSet {
		dngOpts2 = append(dngOpts2, dngconverter.WithExecutable(r.dngConverterExecutable))
	}

	dngConv2, err := dngconverter.New(dngOpts2...)
	if err != nil {
		_ = os.Remove(env.DngIntPrePath)
		return nil, fmt.Errorf("dng converter (linear): %w", err)
	}

	now = time.Now()
	if err := dngConv2.Convert(ctx, env.DngIntPrePath); err != nil {
		_ = os.Remove(env.DngIntPrePath)
		return nil, fmt.Errorf("dng converter (linear) convert: %w", err)
	}
	r.logger.Info("dng converter (linear)", "time", time.Since(now).Seconds())
	if !r.keepIntermediateFiles {
		_ = os.Remove(env.DngIntPrePath)
	}

	rp, err := golibraw.New(baseRawOpts...)
	if err != nil {
		return nil, fmt.Errorf("decodeWithDNG: init raw processor: %w", err)
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
		return nil, fmt.Errorf("decodeWithDNG: open %s: %w", env.DngIntPath, err)
	}

	if _, err := rp.AdjustToRawInsetCrop(golibraw.InsetCropAllMask, 0.0); err != nil {
		return nil, fmt.Errorf("decodeWithDNG: adjust crop: %w", err)
	}

	if err := rp.Unpack(); err != nil {
		return nil, fmt.Errorf("decodeWithDNG: unpack: %w", err)
	}
	if err := rp.Process(); err != nil {
		return nil, fmt.Errorf("decodeWithDNG: process: %w", err)
	}
	raw, err := rp.MakeMemImage()
	if err != nil {
		return nil, fmt.Errorf("decodeWithDNG: make mem image: %w", err)
	}
	r.logger.Info("golibraw (DNG)", "time", time.Since(now).Seconds())

	cd, cdErr := rp.GetColorData()
	var camMul [4]float32
	if cdErr == nil {
		camMul = cd.CamMul
	} else {
		r.logger.Debug("GetColorData failed in decodeWithDNG", "err", cdErr)
	}

	return &decodedImage{
		Width: uint32(raw.Width), Height: uint32(raw.Height),
		Colors: uint16(raw.Colors), Bits: uint16(raw.Bits),
		Data: raw.Data, CamMul: camMul,
	}, nil
}

func (r *Runner) decodeDirect(ctx context.Context, env ConvertEnv) (*decodedImage, error) {
	now := time.Now()
	defer func() { r.logger.Info("decode direct", "time", time.Since(now).Seconds()) }()

	rp, err := golibraw.New(append(baseRawOpts,
		golibraw.WithDNGSDK(golibraw.DNGSDKDefault|golibraw.DNGSDKXTrans),
		golibraw.WithUseRawSpeed(golibraw.RawSpeedV3Use),
		golibraw.WithRawOptions(
			golibraw.RawOptDNGAddEnhanced|
				golibraw.RawOptDNGPreferLargestImage|
				golibraw.RawOptDNGAllowSizeChange|
				golibraw.RawOptDNGStage2IfPresent|
				golibraw.RawOptDNGStage3IfPresent,
		),
	)...)
	if err != nil {
		return nil, fmt.Errorf("decodeDirect: init raw processor: %w", err)
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
		return nil, fmt.Errorf("decodeDirect: enable DNG SDK: %w", err)
	}

	if err := rp.OpenFile(env.SrcPath); err != nil {
		return nil, fmt.Errorf("decodeDirect: open %s: %w", env.SrcPath, err)
	}
	if err := rp.Unpack(); err != nil {
		return nil, fmt.Errorf("decodeDirect: unpack: %w", err)
	}
	if err := rp.Process(); err != nil {
		return nil, fmt.Errorf("decodeDirect: process: %w", err)
	}
	raw, err := rp.MakeMemImage()
	if err != nil {
		return nil, fmt.Errorf("decodeDirect: make mem image: %w", err)
	}

	cd, cdErr := rp.GetColorData()
	var camMul [4]float32
	if cdErr == nil {
		camMul = cd.CamMul
	} else {
		r.logger.Debug("GetColorData failed in decodeDirect", "err", cdErr)
	}

	return &decodedImage{
		Width: uint32(raw.Width), Height: uint32(raw.Height),
		Colors: uint16(raw.Colors), Bits: uint16(raw.Bits),
		Data: raw.Data, CamMul: camMul,
	}, nil
}

// decodeTIFF reads all pixel data from a TIFF file into a contiguous buffer.
// Supports both tiled and stripped layouts.
func (r *Runner) decodeTIFF(srcPath string) (*decodedImage, error) {
	now := time.Now()
	defer func() { r.logger.Info("decode tiff", "time", time.Since(now).Seconds()) }()

	src, err := golibtiff.Open(srcPath, golibtiff.OpenRead)
	if err != nil {
		return nil, fmt.Errorf("decodeTIFF: open: %w", err)
	}
	defer src.Close()

	width, err := src.GetFieldUint32(golibtiff.TagImageWidth)
	if err != nil {
		return nil, fmt.Errorf("decodeTIFF: missing ImageWidth: %w", err)
	}
	height, err := src.GetFieldUint32(golibtiff.TagImageLength)
	if err != nil {
		return nil, fmt.Errorf("decodeTIFF: missing ImageLength: %w", err)
	}
	colors, err := src.GetFieldUint16(golibtiff.TagSamplesPerPixel)
	if err != nil {
		return nil, fmt.Errorf("decodeTIFF: missing SamplesPerPixel: %w", err)
	}
	bits, err := src.GetFieldUint16(golibtiff.TagBitsPerSample)
	if err != nil {
		return nil, fmt.Errorf("decodeTIFF: missing BitsPerSample: %w", err)
	}

	scanline := int64(width) * int64(colors) * int64(bits/8)
	data := make([]byte, int64(height)*scanline)

	if src.IsTiled() {
		tileSize := src.TileSize()
		tileBuf := make([]byte, tileSize)
		tileWidth, _ := src.GetFieldUint32(golibtiff.TagTileWidth)
		tileLength, _ := src.GetFieldUint32(golibtiff.TagTileLength)
		if tileWidth == 0 || tileLength == 0 {
			return nil, fmt.Errorf("decodeTIFF: invalid tile dimensions")
		}
		tilesAcross := (width + tileWidth - 1) / tileWidth
		for tile := uint32(0); tile < src.NumberOfTiles(); tile++ {
			_, err := src.ReadEncodedTile(tile, tileBuf, -1)
			if err != nil {
				return nil, fmt.Errorf("decodeTIFF: read tile %d: %w", tile, err)
			}
			tileRow := (tile / tilesAcross) * tileLength
			tileCol := (tile % tilesAcross) * tileWidth
			tileScanline := int64(tileWidth) * int64(colors) * int64(bits/8)
			actualTileRows := tileLength
			if tileRow+tileLength > height {
				actualTileRows = height - tileRow
			}
			for tr := uint32(0); tr < actualTileRows; tr++ {
				srcOff := int64(tr) * tileScanline
				dstOff := int64(tileRow+tr)*scanline + int64(tileCol)*int64(colors)*int64(bits/8)
				copySize := tileScanline
				if tileCol+tileWidth > width {
					copySize = int64(width-tileCol) * int64(colors) * int64(bits/8)
				}
				copy(data[dstOff:], tileBuf[srcOff:srcOff+copySize])
			}
		}
	} else {
		offset := int64(0)
		for strip := uint32(0); strip < src.NumberOfStrips(); strip++ {
			n, err := src.ReadEncodedStrip(strip, data[offset:], -1)
			if err != nil {
				return nil, fmt.Errorf("decodeTIFF: read strip %d: %w", strip, err)
			}
			offset += int64(n)
		}
	}

	return &decodedImage{
		Width: width, Height: height,
		Colors: colors, Bits: bits,
		Data: data,
	}, nil
}

func (r *Runner) writeMemImageToTIFF(path string, img *decodedImage) error {
	now := time.Now()
	defer func() { r.logger.Info("write TIFF", "time", time.Since(now).Seconds()) }()

	tf, err := golibtiff.Open(path, golibtiff.OpenWrite)
	if err != nil {
		return err
	}
	defer tf.Close()

	w, h := img.Width, img.Height
	colors, bits := img.Colors, img.Bits
	if err := tf.SetFieldUint32(golibtiff.TagImageWidth, w); err != nil {
		return fmt.Errorf("set ImageWidth: %w", err)
	}
	if err := tf.SetFieldUint32(golibtiff.TagImageLength, h); err != nil {
		return fmt.Errorf("set ImageLength: %w", err)
	}
	if err := tf.SetFieldUint16(golibtiff.TagBitsPerSample, bits); err != nil {
		return fmt.Errorf("set BitsPerSample: %w", err)
	}
	if err := tf.SetFieldUint16(golibtiff.TagSamplesPerPixel, colors); err != nil {
		return fmt.Errorf("set SamplesPerPixel: %w", err)
	}
	if err := tf.SetFieldUint16(golibtiff.TagPhotometric, uint16(golibtiff.PhotometricRGB)); err != nil {
		return fmt.Errorf("set Photometric: %w", err)
	}
	if r.cfg.EnableCompression {
		if err := tf.SetFieldUint16(golibtiff.TagCompression, uint16(golibtiff.CompressionLZW)); err != nil {
			return fmt.Errorf("set Compression: %w", err)
		}
		if err := tf.SetFieldUint16(golibtiff.TagPredictor, uint16(golibtiff.PredictorHorizontal)); err != nil {
			return fmt.Errorf("set Predictor: %w", err)
		}
	}
	if err := tf.SetFieldUint16(golibtiff.TagPlanarConfig, uint16(golibtiff.PlanarConfigContig)); err != nil {
		return fmt.Errorf("set PlanarConfig: %w", err)
	}
	if err := tf.SetFieldUint32(golibtiff.TagRowsPerStrip, h); err != nil {
		return fmt.Errorf("set RowsPerStrip: %w", err)
	}

	// ICC profile.
	if profile, ok := icc.Profiles[r.cfg.Profile]; ok {
		if err := tf.SetFieldByteSlice(golibtiff.TagIccProfile, profile.Data); err != nil {
			return fmt.Errorf("set ICC profile: %w", err)
		}
	}

	// Phase 2: Write pixel scanlines.
	scanline := int64(w) * int64(colors) * int64(bits/8)
	for row := uint32(0); row < h; row++ {
		off := int64(row) * scanline
		if err := tf.WriteScanline(img.Data[off:off+scanline], row); err != nil {
			return fmt.Errorf("write scanline %d: %w", row, err)
		}
	}

	if err := tf.WriteDirectory(); err != nil {
		return fmt.Errorf("write directory: %w", err)
	}

	return nil
}

func (r *Runner) writeMetadataExiftool(tiffPath string, env ConvertEnv, img *decodedImage) error {
	now := time.Now()
	defer func() { r.logger.Info("write metadata", "time", time.Since(now).Seconds()) }()

	rawPath := env.SrcPath
	secondSrcPath := env.SrcPath
	if img.DecodeType == DecodeDNG {
		secondSrcPath = env.DngIntPath
	}

	args := []string{
		"--ICC_Profile",
		"-tagsFromFile", rawPath, "-all", "-XMP:all=", "-all:ImageDescription=",
		"-tagsFromFile", secondSrcPath,
		"-AsShotNeutral", "-UniqueCameraModel", "-LocalizedCameraModel",
		"-XMP-aux:all", "-XMP-exifEX:all", "-XMP-dc:subject",
		"-XMP-lr:HierarchicalSubject", "-XMP-mwg-kw:all",
	}

	if img.DecodeType == DecodeDNG {
		args = append(args, "-XMP-dc:Description<raw-wb: ${AsShotNeutral}")
	} else if img.CamMul != [4]float32{} {
		args = append(args, fmt.Sprintf("-XMP-dc:Description=raw-wb: %g %g %g", img.CamMul[0], img.CamMul[1], img.CamMul[2]))
	}

	args = append(args,
		"-IPTC:all=", "-all:Colorspace=", "-orientation=",
		"-XMP-crs:RAWFileName="+filepath.Base(rawPath),
		"-overwrite_original", tiffPath,
	)

	if err := r.et.ExecuteWrite(args...); err != nil {
		return fmt.Errorf("exiftool metadata copy: %w", err)
	}
	return nil
}
