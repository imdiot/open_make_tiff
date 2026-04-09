package runner

import (
	"cmp"
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
	"open-make-tiff/pkg/icc"
)

var ErrDstFileExists = errors.New("destination file already exists")

// decodedImage holds pixel data decoded from a TIFF source.
// Unlike golibraw.ProcessedImage (whose Width/Height are uint16 per LibRaw C API),
// decodedImage uses uint32 for dimensions to support arbitrarily large TIFF images.
type decodedImage struct {
	Width  uint32
	Height uint32
	Colors uint16
	Bits   uint16
	Data   []byte
}

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

func WithDNGConverterExecutable(path string) Option {
	return func(r *Runner) {
		r.dngConverterExecutable = path
		r.dngConverterExecutableSet = true
	}
}

type Runner struct {
	cfg                       Config
	logger                    *slog.Logger
	removeIntermediate        bool
	et                        *exiftool.Exiftool
	dngConverterExecutable    string
	dngConverterExecutableSet bool
}

type ConvertEnv struct {
	SrcPath       string
	DstDir        string
	DngIntPrePath string
	DngIntPath    string
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

	var isNonRawFFF bool

	if strings.ToLower(ext) == ".fff" {
		rp, err := golibraw.New()
		if err != nil {
			returnErr = fmt.Errorf("golibraw init failed: %w", err)
			return returnErr
		}
		isNonRawFFF = rp.OpenFile(srcPath) != nil
		rp.Close()

		if isNonRawFFF {
			r.logger.Info("fff file is TIFF-based, using direct TIFF read", "path", srcPath)
		} else {
			r.logger.Info("fff file is RAW, using golibraw", "path", srcPath)
		}
		useDNG = false
	}

	var img *decodedImage
	secondSrc := srcPath

	if isNonRawFFF {
		now := time.Now()
		img, err = decodeTIFF(srcPath)
		if err != nil {
			returnErr = err
			return returnErr
		}
		r.logger.Info("read TIFF as mem image (TIFF-based fff)", "time", time.Since(now).Seconds())
	} else {
		usedDNG := false
		var rawImg *golibraw.ProcessedImage
		if useDNG {
			rawImg, err = r.decodeWithDNG(ctx, env)
			if err == nil {
				usedDNG = true
			} else {
				r.logger.Warn("DNG converter path failed, falling back to direct: " + err.Error())
			}
		}
		if !usedDNG {
			rawImg, err = r.decodeDirect(ctx, env)
			if err != nil {
				returnErr = err
				return returnErr
			}
		} else {
			secondSrc = env.DngIntPath
		}
		img = &decodedImage{
			Width:  uint32(rawImg.Width),
			Height: uint32(rawImg.Height),
			Colors: rawImg.Colors,
			Bits:   rawImg.Bits,
			Data:   rawImg.Data,
		}
	}

	var meta *ExtractedMetadata
	{
		var metaErr error
		meta, metaErr = r.extractMetadata(srcPath, secondSrc, "ColorSpace")
		if metaErr != nil {
			returnErr = fmt.Errorf("extract metadata: %w", metaErr)
			return returnErr
		}
	}

	if writeErr := r.writeMemImageToTIFF(tiffIntPath, img, meta); writeErr != nil {
		returnErr = writeErr
		return returnErr
	}

	if err := os.Rename(tiffIntPath, dstPath); err != nil {
		returnErr = err
		return returnErr
	}

	return nil
}

func (r *Runner) decodeWithDNG(ctx context.Context, env ConvertEnv) (*golibraw.ProcessedImage, error) {
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
	if r.removeIntermediate {
		_ = os.Remove(env.DngIntPrePath)
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
		return nil, err
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
		return nil, err
	}

	if _, err := rp.AdjustToRawInsetCrop(golibraw.InsetCropAllMask, 0.0); err != nil {
		return nil, err
	}

	if err := rp.Unpack(); err != nil {
		return nil, err
	}
	if err := rp.Process(); err != nil {
		return nil, err
	}
	img, err := rp.MakeMemImage()
	if err != nil {
		return nil, err
	}
	r.logger.Info("run golibraw (with DNG)", "time", time.Since(now).Seconds())

	return img, nil
}

func (r *Runner) decodeDirect(ctx context.Context, env ConvertEnv) (*golibraw.ProcessedImage, error) {
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
		return nil, err
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
		return nil, err
	}

	now := time.Now()
	if err := rp.OpenFile(env.SrcPath); err != nil {
		return nil, err
	}
	if err := rp.Unpack(); err != nil {
		return nil, err
	}
	if err := rp.Process(); err != nil {
		return nil, err
	}
	img, err := rp.MakeMemImage()
	if err != nil {
		return nil, err
	}
	r.logger.Info("run golibraw (direct)", "time", time.Since(now).Seconds())

	return img, nil
}

func decodeTIFF(srcPath string) (*decodedImage, error) {
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
	colors, _ := src.GetFieldUint16(golibtiff.TagSamplesPerPixel)
	bits, _ := src.GetFieldUint16(golibtiff.TagBitsPerSample)
	if colors == 0 {
		colors = 3
	}
	if bits == 0 {
		bits = 16
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
		Width:  width,
		Height: height,
		Colors: colors,
		Bits:   bits,
		Data:   data,
	}, nil
}

func (r *Runner) writeMemImageToTIFF(path string, img *decodedImage, meta *ExtractedMetadata) error {
	now := time.Now()

	tf, err := golibtiff.Open(path, golibtiff.OpenWrite)
	if err != nil {
		return err
	}
	defer tf.Close()

	w := img.Width
	h := img.Height
	colors := img.Colors
	bits := img.Bits
	scanline := int64(w) * int64(colors) * int64(bits/8)

	if err := tf.SetFieldUint32(golibtiff.TagImageWidth, w); err != nil {
		return fmt.Errorf("set ImageWidth: %w", err)
	}
	if err := tf.SetFieldUint32(golibtiff.TagImageLength, h); err != nil {
		return fmt.Errorf("set ImageLength: %w", err)
	}
	_ = tf.SetFieldUint16(golibtiff.TagBitsPerSample, bits)
	_ = tf.SetFieldUint16(golibtiff.TagSamplesPerPixel, colors)
	_ = tf.SetFieldUint16(golibtiff.TagPhotometric, golibtiff.PhotometricRGB)
	if r.cfg.EnableCompression {
		if err := tf.SetFieldUint16(golibtiff.TagCompression, golibtiff.CompressionLZW); err != nil {
			return fmt.Errorf("set Compression: %w", err)
		}
		_ = tf.SetFieldUint16(golibtiff.TagPredictor, golibtiff.PredictorHorizontal)
	}
	_ = tf.SetFieldUint16(golibtiff.TagPlanarConfig, golibtiff.PlanarConfigContig)
	_ = tf.SetFieldUint32(golibtiff.TagRowsPerStrip, h)

	if meta != nil {
		if err := writeIFD0Tags(tf, meta, r.cfg); err != nil {
			return fmt.Errorf("write IFD0 tags: %w", err)
		}
		// Reserve dummy offsets for Sub-IFD pointers in the main IFD.
		// This prevents libtiff from rewriting the main IFD to a new location
		// when we later set the real offsets (per libtiff official docs).
		if len(meta.EXIF) > 0 {
			_ = tf.SetFieldUint64(golibtiff.TagEXIFIFD, 0)
		}
		if len(meta.GPS) > 0 {
			_ = tf.SetFieldUint64(golibtiff.TagGPSIFD, 0)
		}
	}

	for row := uint32(0); row < h; row++ {
		off := int64(row) * scanline
		if err := tf.WriteScanline(img.Data[off:off+scanline], row); err != nil {
			return fmt.Errorf("write scanline %d: %w", row, err)
		}
	}

	if err := writeIFDWithOptionalSubIFDs(tf, meta); err != nil {
		return err
	}
	r.logger.Info("write TIFF", "time", time.Since(now).Seconds())
	return nil
}


func writeIFD0Tags(tf *golibtiff.TIFF, meta *ExtractedMetadata, cfg Config) error {
	writeGroup(tf, meta.IFD0, skipIFD0IDs)

	dpi := cmp.Or(float64(cfg.DPI), 300.0)
	if err := tf.SetFieldDouble(golibtiff.TagXResolution, dpi); err != nil {
		return fmt.Errorf("set XResolution: %w", err)
	}
	if err := tf.SetFieldDouble(golibtiff.TagYResolution, dpi); err != nil {
		return fmt.Errorf("set YResolution: %w", err)
	}
	_ = tf.SetFieldUint16(golibtiff.TagResolutionUnit, golibtiff.ResolutionUnitInch)

	if profile, ok := icc.Profiles[cfg.Profile]; ok {
		if err := tf.SetFieldByteSlice(golibtiff.TagIccProfile, profile.Data()); err != nil {
			return fmt.Errorf("set ICC profile: %w", err)
		}
	}
	if len(meta.XMPPacket) > 0 {
		if err := tf.SetFieldByteSlice(golibtiff.TagXMP, meta.XMPPacket); err != nil {
			return fmt.Errorf("set XMP: %w", err)
		}
	}
	return nil
}

func writeIFDWithOptionalSubIFDs(tf *golibtiff.TIFF, meta *ExtractedMetadata) error {
	if meta == nil {
		return tf.WriteDirectory()
	}
	hasEXIF := len(meta.EXIF) > 0
	hasGPS := len(meta.GPS) > 0

	// Step 1: Write main IFD (IFD0) to disk.
	if err := tf.WriteDirectory(); err != nil {
		return fmt.Errorf("write IFD0: %w", err)
	}

	// Step 2: Write EXIF Sub-IFD (if present).
	if hasEXIF {
		if err := tf.SetDirectory(0); err != nil {
			return fmt.Errorf("set dir 0 for EXIF: %w", err)
		}
		if err := tf.CreateEXIFDirectory(); err != nil {
			return fmt.Errorf("create EXIF directory: %w", err)
		}
		writeGroup(tf, meta.EXIF, nil)
		exifOffset, err := tf.WriteCustomDirectory()
		if err != nil {
			return fmt.Errorf("write EXIF custom directory: %w", err)
		}

		// Update EXIF pointer in IFD0.
		if err := tf.SetDirectory(0); err != nil {
			return fmt.Errorf("set dir 0 after EXIF: %w", err)
		}
		if err := tf.SetFieldUint64(golibtiff.TagEXIFIFD, exifOffset); err != nil {
			return fmt.Errorf("set EXIF IFD pointer: %w", err)
		}
		if err := tf.WriteDirectory(); err != nil {
			return fmt.Errorf("write IFD0 with EXIF pointer: %w", err)
		}
	}

	// Step 3: Write GPS Sub-IFD (if present).
	if hasGPS {
		if err := tf.SetDirectory(0); err != nil {
			return fmt.Errorf("set dir 0 for GPS: %w", err)
		}
		if err := tf.CreateGPSDirectory(); err != nil {
			return fmt.Errorf("create GPS directory: %w", err)
		}
		writeGroup(tf, meta.GPS, nil)
		gpsOffset, err := tf.WriteCustomDirectory()
		if err != nil {
			return fmt.Errorf("write GPS custom directory: %w", err)
		}
		if err := tf.SetDirectory(0); err != nil {
			return fmt.Errorf("set dir 0 after GPS: %w", err)
		}
		if err := tf.SetFieldUint64(golibtiff.TagGPSIFD, gpsOffset); err != nil {
			return fmt.Errorf("set GPS IFD pointer: %w", err)
		}
		if err := tf.WriteDirectory(); err != nil {
			return fmt.Errorf("write IFD0 with GPS pointer: %w", err)
		}
	}

	return nil
}
