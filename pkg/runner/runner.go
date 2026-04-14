package runner

import (
	"cmp"
	"context"
	"encoding/hex"
	"encoding/json"
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
	"open-make-tiff/pkg/metadata"
)

var ErrDstFileExists = errors.New("destination file already exists")

// baseRawOpts contains shared LibRaw processing options for consistent RAW decoding.
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

type DecodeType int

const (
	DecodeDirect DecodeType = iota
	DecodeDNG
	DecodeTIFF
)

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

	img, err := r.decode(ctx, env)
	if err != nil {
		returnErr = err
		return returnErr
	}

	secondSrc := env.SrcPath
	if img.DecodeType == DecodeDNG {
		secondSrc = env.DngIntPath
	}

	var meta *metadata.ExtractedMetadata
	{
		var metaErr error
		meta, metaErr = r.extractMetadata(srcPath, secondSrc, "ColorSpace")
		if metaErr != nil {
			returnErr = fmt.Errorf("extract metadata: %w", metaErr)
			return returnErr
		}
	}

	// WB comment: compatible with MakeTiff 2.01 / ColorPerfect workflow.
	// The decode pipeline skips WB correction (WithUserMul 1,1,1,1),
	// so write the original camera WB to XMP dc:Description as "raw-wb: R G B".
	if meta != nil {
		if img.DecodeType == DecodeDNG {
			if ti, ok := meta.IFD0["AsShotNeutral"]; ok && ti.Val != "" {
				meta.XMP["XMP-dc:Description"] = metadata.TagInfo{
					Val: "raw-wb: " + ti.Val,
				}
			}
		} else if img.CamMul != [4]float32{} {
			meta.XMP["XMP-dc:Description"] = metadata.TagInfo{
				Val: fmt.Sprintf("raw-wb: %g %g %g", img.CamMul[0], img.CamMul[1], img.CamMul[2]),
			}
		}
	}

	if writeErr := r.writeMemImageToTIFF(env.TiffIntPath, img, meta); writeErr != nil {
		returnErr = writeErr
		return returnErr
	}

	if err := os.Rename(tiffIntPath, dstPath); err != nil {
		returnErr = err
		return returnErr
	}

	return nil
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
		r.logger.Warn("DNG converter path failed, falling back to direct: " + err.Error())
	}

	img, err := r.decodeDirect(ctx, env)
	if err != nil {
		return nil, err
	}
	img.DecodeType = DecodeDirect
	return img, nil
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
	if r.removeIntermediate {
		_ = os.Remove(env.DngIntPrePath)
	}

	rp, err := golibraw.New(baseRawOpts...)
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
	raw, err := rp.MakeMemImage()
	if err != nil {
		return nil, err
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

	if err := rp.OpenFile(env.SrcPath); err != nil {
		return nil, err
	}
	if err := rp.Unpack(); err != nil {
		return nil, err
	}
	if err := rp.Process(); err != nil {
		return nil, err
	}
	raw, err := rp.MakeMemImage()
	if err != nil {
		return nil, err
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
		Width: width, Height: height,
		Colors: colors, Bits: bits,
		Data: data,
	}, nil
}

func (r *Runner) writeMemImageToTIFF(path string, img *decodedImage, meta *metadata.ExtractedMetadata) error {
	now := time.Now()

	tf, err := golibtiff.Open(path, golibtiff.OpenWrite)
	if err != nil {
		return err
	}
	defer tf.Close()

	// Phase 1: Set image dimension and format tags.
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

	// Phase 2: Write IFD0 metadata and config overrides.
	if meta != nil {
		meta.WriteIFD0(tf)

		dpi := cmp.Or(float64(r.cfg.DPI), 300.0)
		if err := tf.SetFieldDouble(golibtiff.TagXResolution, dpi); err != nil {
			return fmt.Errorf("set XResolution: %w", err)
		}
		if err := tf.SetFieldDouble(golibtiff.TagYResolution, dpi); err != nil {
			return fmt.Errorf("set YResolution: %w", err)
		}
		_ = tf.SetFieldUint16(golibtiff.TagResolutionUnit, uint16(golibtiff.ResolutionUnitInch))
		if profile, ok := icc.Profiles[r.cfg.Profile]; ok {
			if err := tf.SetFieldByteSlice(golibtiff.TagIccProfile, profile.Data()); err != nil {
				return fmt.Errorf("set ICC profile: %w", err)
			}
		}

		if err := meta.ReserveSubIFDs(tf); err != nil {
			return err
		}
	}

	// Phase 3: Write pixel scanlines.
	scanline := int64(w) * int64(colors) * int64(bits/8)
	for row := uint32(0); row < h; row++ {
		off := int64(row) * scanline
		if err := tf.WriteScanline(img.Data[off:off+scanline], row); err != nil {
			return fmt.Errorf("write scanline %d: %w", row, err)
		}
	}

	// Phase 4: Write Sub-IFDs (EXIF, GPS).
	if meta != nil {
		if err := meta.WriteSubIFDs(tf); err != nil {
			return err
		}
	} else {
		if err := tf.WriteDirectory(); err != nil {
			return fmt.Errorf("write directory: %w", err)
		}
	}

	r.logger.Info("write TIFF", "time", time.Since(now).Seconds())
	return nil
}

func (r *Runner) extractMetadata(rawPath, secondSrcPath string, excludeKeys ...string) (*metadata.ExtractedMetadata, error) {
	if r.et == nil {
		return nil, nil
	}

	samePath := strings.EqualFold(filepath.Clean(rawPath), filepath.Clean(secondSrcPath))

	args := []string{
		"-json", "-G1", "-l", "-t", "-b", "-a", "-U", "-ee",
		"-api", "SaveBin=1", "-api", "SaveFormat=1", "-api", "MakerNotes=1",
		"-IFD0:All", "-ExifIFD:All", "-GPS:All",
		"-XMP-aux:All", "-XMP-exifEX:All",
		"-XMP-dc:Subject", "-XMP-lr:HierarchicalSubject", "-XMP-mwg-kw:All",
	}
	args = append(args, rawPath)
	if !samePath {
		args = append(args, secondSrcPath)
	}

	resp, err := r.et.Execute(args...)
	if err != nil {
		return nil, fmt.Errorf("exiftool extract metadata: %w", err)
	}

	var objects []map[string]any
	if err := json.Unmarshal([]byte(resp), &objects); err != nil {
		return nil, fmt.Errorf("json parse: %w", err)
	}
	if len(objects) == 0 {
		return nil, nil
	}

	var rawEM, dngEM *metadata.ExtractedMetadata
	if samePath {
		rawEM = metadata.NewExtractedMetadata(r.logger)
		rawEM.Parse(objects[0])
		dngEM = rawEM
	} else {
		for _, obj := range objects {
			src, _ := obj["SourceFile"].(string)
			em := metadata.NewExtractedMetadata(r.logger)
			em.Parse(obj)
			if strings.EqualFold(filepath.Clean(src), filepath.Clean(rawPath)) {
				rawEM = em
			} else {
				dngEM = em
			}
		}
	}

	if rawEM == nil {
		return nil, nil
	}

	if dngEM != nil {
		for name, ti := range dngEM.IFD0 {
			if metadata.DNGOverrideIDs[ti.TagID()] {
				rawEM.IFD0[name] = ti
			}
		}
		if len(dngEM.XMP) > 0 {
			rawEM.XMP = dngEM.XMP
		}
	}
	rawEM.XMP["XMP-crs:RAWFileName"] = metadata.TagInfo{Val: filepath.Base(rawPath)}

	for _, key := range excludeKeys {
		delete(rawEM.IFD0, key)
		delete(rawEM.EXIF, key)
		delete(rawEM.GPS, key)
	}

	return rawEM, nil
}
