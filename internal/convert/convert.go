// Package convert implements the single-file RAW/TIFF → linear TIFF conversion
// pipeline. It wraps pkg/golibraw, pkg/golibtiff, pkg/exiftool and
// pkg/dngconverter directly: those are the only implementations and CGO is
// mandatory, so no interface abstraction is introduced.
package convert

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

	"open-make-tiff/internal/config"
	"open-make-tiff/pkg/dngconverter"
	"open-make-tiff/pkg/exiftool"
	"open-make-tiff/pkg/golibraw"
	"open-make-tiff/pkg/golibtiff"
	"open-make-tiff/pkg/icc"
)

// ErrDstExists is returned when the destination TIFF already exists.
var ErrDstExists = errors.New("destination file already exists")

// baseRawOpts reproduces the dcraw semantics (-o0 -g1 -H1 -M -r1 -q3) that
// yield a linear, unprocessed TIFF. Must stay 1:1 with the original MakeTiff
// behaviour (see maketiff-2.01-mac-analysis/docs/FIRST_STAGE.md).
var baseRawOpts = []golibraw.Option{
	golibraw.WithUserMul(1, 1, 1, 1),
	golibraw.WithOutputColorSpace(golibraw.ColorSpaceRaw),
	golibraw.WithFlip(golibraw.FlipNone),
	golibraw.WithHighlightMode(golibraw.HighlightUnclip),
	golibraw.With16BitOutput(),
	golibraw.WithNoAutoBrightness(),
	golibraw.WithInterpolationQuality(golibraw.QualityAHD),
	golibraw.WithGamma(1.0, 1.0),
	golibraw.WithAdjustMaxThreshold(0),
	golibraw.WithEmbeddedColorMatrix(false),
}

type decodeSource int

const (
	decodeRawDirect decodeSource = iota
	decodeRawDNG
	decodePlainTIFF
)

type probeFormat int

const (
	probeUnknown probeFormat = iota
	probeRAW
	probePlainTIFF
)

// decodedImage holds decoded pixel data. Dimensions are uint32 (unlike
// golibraw.ProcessedImage's uint16) to support arbitrarily large TIFF images.
type decodedImage struct {
	source decodeSource
	width  uint32
	height uint32
	colors uint16
	bits   uint16
	data   []byte
	camMul [4]float32
}

// Converter runs the conversion pipeline for a single file. Dependencies are
// injected once via New; per-file state lives on the stack of Convert, so a
// single Converter is safe to share across concurrent file conversions.
type Converter struct {
	et      *exiftool.Exiftool // nil = skip metadata
	dngExec string             // macOS shadow-bundle path; "" = use default DNG Converter
}

// Option configures a Converter.
type Option func(*Converter)

// WithExiftool injects the ExifTool handle used to copy metadata. nil disables
// metadata writing.
func WithExiftool(et *exiftool.Exiftool) Option {
	return func(c *Converter) { c.et = et }
}

// WithDNGExecutable sets the Adobe DNG Converter executable path (typically the
// macOS shadow-bundle wrapper). Empty falls back to the platform default.
func WithDNGExecutable(path string) Option {
	return func(c *Converter) { c.dngExec = path }
}

// New creates a Converter with the given options.
func New(opts ...Option) *Converter {
	c := &Converter{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Convert converts srcPath into a linear TIFF next to it (or in a "make_tiff"
// subfolder when cfg.EnableSubfolder). ctx cancellation aborts in-flight
// decoding. The returned error wraps ErrDstExists when the destination exists.
func (c *Converter) Convert(ctx context.Context, srcPath string, cfg *config.Config) error {
	srcPath, err := filepath.Abs(srcPath)
	if err != nil {
		return err
	}

	srcDir := filepath.Dir(srcPath)
	dstDir := srcDir
	if cfg.EnableSubfolder {
		dstDir = filepath.Join(dstDir, "make_tiff")
	}
	name := filepath.Base(srcPath)
	dstPath := filepath.Join(dstDir, fmt.Sprintf("%s.tiff", name))
	if _, err := os.Stat(dstPath); err == nil {
		return fmt.Errorf("%w: %s", ErrDstExists, dstPath)
	}
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	ws := newWorkspace(srcPath, dstDir, name)

	var returnErr error
	defer func() {
		if !cfg.KeepIntermediateFiles {
			ws.cleanupIntermediate()
		}
		if returnErr != nil {
			_ = os.Remove(dstPath)
		}
	}()

	logFile, err := os.Create(ws.logPath)
	if err != nil {
		return err
	}
	log := slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{Level: slog.LevelDebug}))
	defer func() {
		if returnErr != nil {
			log.Error(returnErr.Error())
		}
		_ = logFile.Close()
		if returnErr == nil && !cfg.KeepLogFiles {
			_ = os.Remove(ws.logPath)
		}
	}()

	img, err := c.decode(ctx, ws, cfg, log)
	if err != nil {
		returnErr = err
		return returnErr
	}

	if writeErr := c.writeTIFF(ws.tiffIntPath, img, cfg, log); writeErr != nil {
		returnErr = writeErr
		return returnErr
	}

	if c.et != nil {
		if metaErr := c.writeMetadata(ws.tiffIntPath, ws, img, log); metaErr != nil {
			returnErr = metaErr
			return returnErr
		}
	}

	if err := os.Rename(ws.tiffIntPath, dstPath); err != nil {
		returnErr = err
		return returnErr
	}
	return nil
}

// workspace holds the per-conversion intermediate file paths. A uuid token
// avoids collisions with concurrent or previous runs.
type workspace struct {
	srcPath       string
	dstDir        string
	logPath       string
	dngIntPrePath string
	dngIntPath    string
	tiffIntPath   string
}

func newWorkspace(srcPath, dstDir, name string) *workspace {
	for {
		u := uuid.New()
		token := hex.EncodeToString(u[:])
		ws := &workspace{
			srcPath:       srcPath,
			dstDir:        dstDir,
			logPath:       filepath.Join(dstDir, fmt.Sprintf("%s_%s.log", name, token)),
			dngIntPrePath: filepath.Join(dstDir, fmt.Sprintf("%s_%s.int_pre.dng", name, token)),
			dngIntPath:    filepath.Join(dstDir, fmt.Sprintf("%s_%s.int.dng", name, token)),
			tiffIntPath:   filepath.Join(dstDir, fmt.Sprintf("%s_%s.int.tiff", name, token)),
		}
		conflict := slices.ContainsFunc(
			[]string{ws.logPath, ws.dngIntPrePath, ws.dngIntPath, ws.tiffIntPath},
			func(f string) bool {
				_, err := os.Stat(f)
				return err == nil || !errors.Is(err, os.ErrNotExist)
			},
		)
		if !conflict {
			return ws
		}
	}
}

func (ws *workspace) cleanupIntermediate() {
	for _, f := range []string{ws.dngIntPrePath, ws.dngIntPath, ws.tiffIntPath} {
		if f != "" {
			_ = os.Remove(f)
		}
	}
}

// probe detects whether srcPath is RAW (LibRaw can open it) or a plain TIFF.
// RAW formats are often TIFF containers, so LibRaw is tried first.
func (c *Converter) probe(srcPath string, log *slog.Logger) (probeFormat, error) {
	rp, probeErr := golibraw.New()
	if probeErr != nil {
		return probeUnknown, fmt.Errorf("golibraw init failed: %w", probeErr)
	}
	librawOK := rp.OpenFile(srcPath) == nil
	rp.Close()

	if librawOK {
		log.Info("detected RAW format", "path", srcPath)
		return probeRAW, nil
	}

	tf, tiffErr := golibtiff.Open(srcPath, golibtiff.OpenRead)
	if tiffErr != nil {
		log.Warn("file not recognized as RAW or TIFF by probe, will attempt direct decode", "path", srcPath)
		return probeUnknown, nil
	}
	tf.Close()

	log.Info("detected TIFF format", "path", srcPath)
	return probePlainTIFF, nil
}

func (c *Converter) decode(ctx context.Context, ws *workspace, cfg *config.Config, log *slog.Logger) (*decodedImage, error) {
	f, probeErr := c.probe(ws.srcPath, log)
	if probeErr != nil {
		return nil, probeErr
	}
	if f == probePlainTIFF {
		img, err := c.decodeTIFF(ws.srcPath, log)
		if err != nil {
			return nil, err
		}
		img.source = decodePlainTIFF
		return img, nil
	}

	useDNG := !cfg.DisableAdobeDNGConverter
	if useDNG {
		execPath := c.dngExec
		if execPath == "" {
			execPath = dngconverter.GetDefaultExecutablePath()
		}
		switch {
		case execPath == "":
			useDNG = false
		default:
			if _, err := os.Stat(execPath); err != nil {
				log.Warn("DNG Converter not available, using direct path", "error", err)
				useDNG = false
			}
		}
	}

	if useDNG {
		img, err := c.decodeDNG(ctx, ws, log)
		if err == nil {
			img.source = decodeRawDNG
			return img, nil
		}
		log.Warn("DNG converter path failed, falling back to direct", "error", err)
	}

	img, err := c.decodeDirect(ctx, ws.srcPath, log)
	if err != nil {
		return nil, err
	}
	img.source = decodeRawDirect
	return img, nil
}

func (c *Converter) decodeDirect(ctx context.Context, srcPath string, log *slog.Logger) (*decodedImage, error) {
	start := time.Now()
	defer func() { log.Info("decode direct", "time", time.Since(start).Seconds()) }()

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

	stop := watchCancel(ctx, rp.Cancel)
	defer stop()

	if err := rp.EnableDNGSDK(); err != nil {
		return nil, fmt.Errorf("decodeDirect: enable DNG SDK: %w", err)
	}
	if err := rp.OpenFile(srcPath); err != nil {
		return nil, fmt.Errorf("decodeDirect: open %s: %w", srcPath, err)
	}
	if err := rp.Unpack(); err != nil {
		return nil, fmt.Errorf("decodeDirect: unpack: %w", err)
	}
	if err := rp.Process(); err != nil {
		return nil, fmt.Errorf("decodeDirect: process: %w", err)
	}
	return extractImage(rp, log, "decodeDirect")
}

func (c *Converter) decodeDNG(ctx context.Context, ws *workspace, log *slog.Logger) (*decodedImage, error) {
	start := time.Now()
	defer func() { log.Info("decode DNG", "time", time.Since(start).Seconds()) }()

	// Pass 1: RAW → uninterpolated DNG.
	dngOpts1 := []dngconverter.Option{
		dngconverter.WithUncompressed(true),
		dngconverter.WithPreviewSize(dngconverter.PreviewNone),
		dngconverter.WithCameraRawCompat(dngconverter.CameraRaw54),
		dngconverter.WithOutputDir(ws.dstDir),
		dngconverter.WithOutputFilename(filepath.Base(ws.dngIntPrePath)),
		dngconverter.WithLogger(log),
	}
	if c.dngExec != "" {
		dngOpts1 = append(dngOpts1, dngconverter.WithExecutable(c.dngExec))
	}
	dngConv1, err := dngconverter.New(dngOpts1...)
	if err != nil {
		return nil, fmt.Errorf("dng converter (raw): %w", err)
	}
	now := time.Now()
	if err := dngConv1.Convert(ctx, ws.srcPath); err != nil {
		return nil, fmt.Errorf("dng converter (raw) convert: %w", err)
	}
	log.Info("dng converter (raw)", "time", time.Since(now).Seconds())

	// Pass 2: uninterpolated DNG → linear (demosaiced) DNG.
	dngOpts2 := []dngconverter.Option{
		dngconverter.WithUncompressed(true),
		dngconverter.WithLinear(true),
		dngconverter.WithPreviewSize(dngconverter.PreviewNone),
		dngconverter.WithDNGVersion(dngconverter.DNG11),
		dngconverter.WithOutputDir(ws.dstDir),
		dngconverter.WithOutputFilename(filepath.Base(ws.dngIntPath)),
		dngconverter.WithLogger(log),
	}
	if c.dngExec != "" {
		dngOpts2 = append(dngOpts2, dngconverter.WithExecutable(c.dngExec))
	}
	dngConv2, err := dngconverter.New(dngOpts2...)
	if err != nil {
		_ = os.Remove(ws.dngIntPrePath)
		return nil, fmt.Errorf("dng converter (linear): %w", err)
	}
	now = time.Now()
	if err := dngConv2.Convert(ctx, ws.dngIntPrePath); err != nil {
		_ = os.Remove(ws.dngIntPrePath)
		return nil, fmt.Errorf("dng converter (linear) convert: %w", err)
	}
	log.Info("dng converter (linear)", "time", time.Since(now).Seconds())

	// Extract pixels from the linear DNG via LibRaw.
	rp, err := golibraw.New(baseRawOpts...)
	if err != nil {
		return nil, fmt.Errorf("decodeDNG: init raw processor: %w", err)
	}
	defer rp.Close()

	stop := watchCancel(ctx, rp.Cancel)
	defer stop()

	now = time.Now()
	if err := rp.OpenFile(ws.dngIntPath); err != nil {
		return nil, fmt.Errorf("decodeDNG: open %s: %w", ws.dngIntPath, err)
	}
	if _, err := rp.AdjustToRawInsetCrop(golibraw.InsetCropAllMask, 0.0); err != nil {
		return nil, fmt.Errorf("decodeDNG: adjust crop: %w", err)
	}
	if err := rp.Unpack(); err != nil {
		return nil, fmt.Errorf("decodeDNG: unpack: %w", err)
	}
	if err := rp.Process(); err != nil {
		return nil, fmt.Errorf("decodeDNG: process: %w", err)
	}
	log.Info("golibraw (DNG)", "time", time.Since(now).Seconds())

	return extractImage(rp, log, "decodeDNG")
}

// decodeTIFF reads all pixel data from a TIFF file into a contiguous buffer,
// supporting both tiled and stripped layouts.
func (c *Converter) decodeTIFF(srcPath string, log *slog.Logger) (*decodedImage, error) {
	start := time.Now()
	defer func() { log.Info("decode tiff", "time", time.Since(start).Seconds()) }()

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
		tileBuf := make([]byte, src.TileSize())
		tileWidth, _ := src.GetFieldUint32(golibtiff.TagTileWidth)
		tileLength, _ := src.GetFieldUint32(golibtiff.TagTileLength)
		if tileWidth == 0 || tileLength == 0 {
			return nil, fmt.Errorf("decodeTIFF: invalid tile dimensions")
		}
		tilesAcross := (width + tileWidth - 1) / tileWidth
		for tile := uint32(0); tile < src.NumberOfTiles(); tile++ {
			if _, err := src.ReadEncodedTile(tile, tileBuf, -1); err != nil {
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
		width:  width,
		height: height,
		colors: colors,
		bits:   bits,
		data:   data,
	}, nil
}

func (c *Converter) writeTIFF(path string, img *decodedImage, cfg *config.Config, log *slog.Logger) error {
	start := time.Now()
	defer func() { log.Info("write TIFF", "time", time.Since(start).Seconds()) }()

	tf, err := golibtiff.Open(path, golibtiff.OpenWrite)
	if err != nil {
		return err
	}
	defer tf.Close()

	w, h := img.width, img.height
	colors, bits := img.colors, img.bits
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
	if cfg.EnableCompression {
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

	// ICC profile is embedded verbatim; no colour conversion is performed.
	if profile, ok := icc.Profiles[cfg.ICCProfile]; ok {
		if err := tf.SetFieldByteSlice(golibtiff.TagIccProfile, profile.Data); err != nil {
			return fmt.Errorf("set ICC profile: %w", err)
		}
	}

	scanline := int64(w) * int64(colors) * int64(bits/8)
	for row := uint32(0); row < h; row++ {
		off := int64(row) * scanline
		if err := tf.WriteScanline(img.data[off:off+scanline], row); err != nil {
			return fmt.Errorf("write scanline %d: %w", row, err)
		}
	}

	if err := tf.WriteDirectory(); err != nil {
		return fmt.Errorf("write directory: %w", err)
	}
	return nil
}

func (c *Converter) writeMetadata(tiffPath string, ws *workspace, img *decodedImage, log *slog.Logger) error {
	start := time.Now()
	defer func() { log.Info("write metadata", "time", time.Since(start).Seconds()) }()

	rawPath := ws.srcPath
	secondSrcPath := ws.srcPath
	if img.source == decodeRawDNG {
		secondSrcPath = ws.dngIntPath
	}

	args := []string{
		"--ICC_Profile",
		"-tagsFromFile", rawPath, "-all", "-XMP:all=", "-all:ImageDescription=",
		"-tagsFromFile", secondSrcPath,
		"-AsShotNeutral", "-UniqueCameraModel", "-LocalizedCameraModel",
		"-XMP-aux:all", "-XMP-exifEX:all", "-XMP-dc:subject",
		"-XMP-lr:HierarchicalSubject", "-XMP-mwg-kw:all",
	}

	if img.source == decodeRawDNG {
		args = append(args, "-XMP-dc:Description<raw-wb: ${AsShotNeutral}")
	} else if img.camMul != [4]float32{} {
		args = append(args, fmt.Sprintf("-XMP-dc:Description=raw-wb: %g %g %g", img.camMul[0], img.camMul[1], img.camMul[2]))
	}

	args = append(args,
		"-IPTC:all=", "-all:Colorspace=", "-orientation=",
		"-XMP-crs:RAWFileName="+filepath.Base(rawPath),
		"-overwrite_original", tiffPath,
	)

	if err := c.et.ExecuteWrite(args...); err != nil {
		return fmt.Errorf("exiftool metadata copy: %w", err)
	}
	return nil
}

// watchCancel calls cancel as soon as ctx is done. The returned stop func must
// be called (typically deferred) once the cancellable operation completes, to
// release the watcher goroutine.
func watchCancel(ctx context.Context, cancel func()) (stop func()) {
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			cancel()
		case <-done:
		}
	}()
	return func() { close(done) }
}

// extractImage copies the processed pixels and white-balance multipliers out of
// a RawProcessor. Shared by decodeDirect and decodeDNG.
func extractImage(rp *golibraw.RawProcessor, log *slog.Logger, op string) (*decodedImage, error) {
	w, h, colors, bps := rp.GetMemImageFormat()
	if w == 0 || h == 0 {
		return nil, fmt.Errorf("%s: invalid image format: %dx%d", op, w, h)
	}
	stride := w * colors * (bps / 8)
	buf := make([]byte, stride*h)
	if err := rp.CopyMemImage(buf, stride, false); err != nil {
		return nil, fmt.Errorf("%s: copy mem image: %w", op, err)
	}

	var camMul [4]float32
	if cd, cdErr := rp.GetColorData(); cdErr == nil {
		camMul = cd.CamMul
	} else {
		log.Debug(op+": GetColorData failed", "err", cdErr)
	}

	return &decodedImage{
		width:  uint32(w),
		height: uint32(h),
		colors: uint16(colors),
		bits:   uint16(bps),
		data:   buf,
		camMul: camMul,
	}, nil
}
