package tiffcopy

import (
	"fmt"

	"open-make-tiff/pkg/golibtiff"
)

type copyConfig struct {
	compression uint16
	predictor   uint16
}

// Option configures the TIFF copy operation.
type Option func(*copyConfig)

// WithLZWCompression enables LZW compression with the given predictor value.
// A predictor of 2 enables horizontal differencing (recommended for most images).
func WithLZWCompression(predictor uint16) Option {
	return func(c *copyConfig) {
		c.compression = golibtiff.CompressionLZW
		c.predictor = predictor
	}
}

// WithDeflateCompression enables Deflate (zlib) compression with the given preset (1-9)
// and predictor value.
func WithDeflateCompression(preset int, predictor uint16) Option {
	return func(c *copyConfig) {
		c.compression = golibtiff.CompressionDeflate
		c.predictor = predictor
	}
}

// Copy copies a TIFF file from srcPath to dstPath, optionally changing compression.
// All pages (IFDs) in the source file are copied. Pixel data is read decompressed and
// re-encoded with the target compression, preserving all image metadata tags.
func Copy(srcPath, dstPath string, opts ...Option) error {
	cfg := copyConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	src, err := golibtiff.Open(srcPath, golibtiff.OpenRead)
	if err != nil {
		return fmt.Errorf("tiffcopy: open source: %w", err)
	}
	defer src.Close()

	dst, err := golibtiff.Open(dstPath, golibtiff.OpenWrite)
	if err != nil {
		return fmt.Errorf("tiffcopy: open dest: %w", err)
	}
	defer dst.Close()

	numDirs := src.NumberOfDirectories()
	for i := uint32(0); i < numDirs; i++ {
		if err := src.SetDirectory(i); err != nil {
			return fmt.Errorf("tiffcopy: set source directory %d: %w", i, err)
		}
		if err := copyIFD(src, dst, cfg); err != nil {
			return fmt.Errorf("tiffcopy: copy IFD %d: %w", i, err)
		}
		if i < numDirs-1 {
			if err := dst.WriteDirectory(); err != nil {
				return fmt.Errorf("tiffcopy: write directory %d: %w", i, err)
			}
		}
	}

	return nil
}

func copyIFD(src, dst *golibtiff.TIFF, cfg copyConfig) error {
	if err := copyTags(src, dst, cfg); err != nil {
		return err
	}

	if src.IsTiled() {
		return copyTileData(src, dst)
	}
	return copyStripData(src, dst)
}

func copyTags(src, dst *golibtiff.TIFF, cfg copyConfig) error {
	width, _ := src.GetFieldUint32(golibtiff.TagImageWidth)
	height, _ := src.GetFieldUint32(golibtiff.TagImageLength)
	spp, _ := src.GetFieldUint16(golibtiff.TagSamplesPerPixel)
	photo, _ := src.GetFieldUint16(golibtiff.TagPhotometric)
	planar, _ := src.GetFieldUint16(golibtiff.TagPlanarConfig)
	orientation, _ := src.GetFieldUint16(golibtiff.TagOrientation)

	// BitsPerSample may be a single value or an array
	bpsSlice, err := src.GetFieldUint16Slice(golibtiff.TagBitsPerSample)
	if err != nil || len(bpsSlice) == 0 {
		bps, _ := src.GetFieldUint16(golibtiff.TagBitsPerSample)
		bpsSlice = []uint16{bps}
	}

	// SampleFormat may be a single value or an array
	sfSlice, err := src.GetFieldUint16Slice(golibtiff.TagSampleFormat)
	if err != nil || len(sfSlice) == 0 {
		sf, _ := src.GetFieldUint16(golibtiff.TagSampleFormat)
		sfSlice = []uint16{sf}
	}

	// Core dimensions
	if err := dst.SetFieldUint32(golibtiff.TagImageWidth, width); err != nil {
		return fmt.Errorf("set width: %w", err)
	}
	if err := dst.SetFieldUint32(golibtiff.TagImageLength, height); err != nil {
		return fmt.Errorf("set height: %w", err)
	}
	if err := dst.SetFieldUint16(golibtiff.TagSamplesPerPixel, spp); err != nil {
		return fmt.Errorf("set samples per pixel: %w", err)
	}
	if len(bpsSlice) == 1 {
		if err := dst.SetFieldUint16(golibtiff.TagBitsPerSample, bpsSlice[0]); err != nil {
			return fmt.Errorf("set bits per sample: %w", err)
		}
	} else {
		if err := dst.SetFieldUint16Slice(golibtiff.TagBitsPerSample, bpsSlice); err != nil {
			return fmt.Errorf("set bits per sample: %w", err)
		}
	}
	if err := dst.SetFieldUint16(golibtiff.TagPhotometric, photo); err != nil {
		return fmt.Errorf("set photometric: %w", err)
	}
	if err := dst.SetFieldUint16(golibtiff.TagPlanarConfig, planar); err != nil {
		return fmt.Errorf("set planar config: %w", err)
	}

	if len(sfSlice) == 1 {
		_ = dst.SetFieldUint16(golibtiff.TagSampleFormat, sfSlice[0])
	} else if len(sfSlice) > 1 {
		_ = dst.SetFieldUint16Slice(golibtiff.TagSampleFormat, sfSlice)
	}

	// Compression
	compression := cfg.compression
	if compression == 0 {
		compression, _ = src.GetFieldUint16(golibtiff.TagCompression)
	}
	if err := dst.SetFieldUint16(golibtiff.TagCompression, compression); err != nil {
		return fmt.Errorf("set compression: %w", err)
	}

	// Predictor (only relevant for LZW/Deflate)
	if cfg.predictor != 0 {
		_ = dst.SetFieldUint16(golibtiff.TagPredictor, cfg.predictor)
	} else {
		pred, err := src.GetFieldUint16(golibtiff.TagPredictor)
		if err == nil {
			_ = dst.SetFieldUint16(golibtiff.TagPredictor, pred)
		}
	}

	// Layout: strip or tile
	if src.IsTiled() {
		tw, _ := src.GetFieldUint32(golibtiff.TagTileWidth)
		tl, _ := src.GetFieldUint32(golibtiff.TagTileLength)
		_ = dst.SetFieldUint32(golibtiff.TagTileWidth, tw)
		_ = dst.SetFieldUint32(golibtiff.TagTileLength, tl)
	} else {
		rps, err := src.GetFieldUint32(golibtiff.TagRowsPerStrip)
		if err == nil {
			_ = dst.SetFieldUint32(golibtiff.TagRowsPerStrip, rps)
		}
	}

	// Optional tags
	if orientation > 0 {
		_ = dst.SetFieldUint16(golibtiff.TagOrientation, orientation)
	}
	copyOptionalTag(src, dst, golibtiff.TagExtraSamples, true)
	copyOptionalTag(src, dst, golibtiff.TagResolutionUnit, false)
	copyOptionalFloatTag(src, dst, golibtiff.TagXResolution)
	copyOptionalFloatTag(src, dst, golibtiff.TagYResolution)
	copyOptionalStringTag(src, dst, golibtiff.TagSoftware)
	copyOptionalStringTag(src, dst, golibtiff.TagDateTime)
	copyOptionalStringTag(src, dst, golibtiff.TagImageDescription)
	copyOptionalStringTag(src, dst, golibtiff.TagArtist)
	copyOptionalStringTag(src, dst, golibtiff.TagDocumentName)
	copyOptionalStringTag(src, dst, golibtiff.TagCopyright)
	copyOptionalUint32Tag(src, dst, golibtiff.TagNewSubfileType)

	return nil
}

func copyOptionalTag(src, dst *golibtiff.TIFF, tag golibtiff.Tag, isSlice bool) {
	if isSlice {
		val, err := src.GetFieldUint16Slice(tag)
		if err == nil && len(val) > 0 {
			_ = dst.SetFieldUint16Slice(tag, val)
		}
	} else {
		val, err := src.GetFieldUint16(tag)
		if err == nil {
			_ = dst.SetFieldUint16(tag, val)
		}
	}
}

func copyOptionalFloatTag(src, dst *golibtiff.TIFF, tag golibtiff.Tag) {
	val, err := src.GetFieldFloat(tag)
	if err == nil {
		_ = dst.SetFieldFloat(tag, val)
	}
}

func copyOptionalStringTag(src, dst *golibtiff.TIFF, tag golibtiff.Tag) {
	val, err := src.GetFieldString(tag)
	if err == nil && val != "" {
		_ = dst.SetFieldString(tag, val)
	}
}

func copyOptionalUint32Tag(src, dst *golibtiff.TIFF, tag golibtiff.Tag) {
	val, err := src.GetFieldUint32(tag)
	if err == nil {
		_ = dst.SetFieldUint32(tag, val)
	}
}

func copyStripData(src, dst *golibtiff.TIFF) error {
	numStrips := src.NumberOfStrips()
	stripSize := src.StripSize()
	if stripSize <= 0 || numStrips == 0 {
		return fmt.Errorf("tiffcopy: invalid strip layout (strips=%d, size=%d)", numStrips, stripSize)
	}

	buf := make([]byte, stripSize)
	for strip := uint32(0); strip < numStrips; strip++ {
		n, err := src.ReadEncodedStrip(strip, buf, -1)
		if err != nil {
			return fmt.Errorf("tiffcopy: read strip %d: %w", strip, err)
		}
		if _, err := dst.WriteEncodedStrip(strip, buf[:n]); err != nil {
			return fmt.Errorf("tiffcopy: write strip %d: %w", strip, err)
		}
	}
	return nil
}

func copyTileData(src, dst *golibtiff.TIFF) error {
	numTiles := src.NumberOfTiles()
	tileSize := src.TileSize()
	if tileSize <= 0 || numTiles == 0 {
		return fmt.Errorf("tiffcopy: invalid tile layout (tiles=%d, size=%d)", numTiles, tileSize)
	}

	buf := make([]byte, tileSize)
	for tile := uint32(0); tile < numTiles; tile++ {
		n, err := src.ReadEncodedTile(tile, buf, -1)
		if err != nil {
			return fmt.Errorf("tiffcopy: read tile %d: %w", tile, err)
		}
		if _, err := dst.WriteEncodedTile(tile, buf[:n]); err != nil {
			return fmt.Errorf("tiffcopy: write tile %d: %w", tile, err)
		}
	}
	return nil
}
