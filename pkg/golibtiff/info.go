package golibtiff

// Convenience methods for common TIFF tags. These are pure Go wrappers around GetField* methods.

func (t *TIFF) Width() (uint32, error)       { return t.GetFieldUint32(TagImageWidth) }
func (t *TIFF) Height() (uint32, error)      { return t.GetFieldUint32(TagImageLength) }
func (t *TIFF) BitsPerSample() (uint16, error) { return t.GetFieldUint16(TagBitsPerSample) }
func (t *TIFF) SamplesPerPixel() (uint16, error) { return t.GetFieldUint16(TagSamplesPerPixel) }
func (t *TIFF) Compression() (uint16, error) { return t.GetFieldUint16(TagCompression) }
func (t *TIFF) Photometric() (uint16, error) { return t.GetFieldUint16(TagPhotometric) }
func (t *TIFF) PlanarConfig() (uint16, error) { return t.GetFieldUint16(TagPlanarConfig) }
func (t *TIFF) Orientation() (uint16, error) { return t.GetFieldUint16(TagOrientation) }
func (t *TIFF) SampleFormat() (uint16, error) { return t.GetFieldUint16(TagSampleFormat) }
func (t *TIFF) RowsPerStrip() (uint32, error) { return t.GetFieldUint32(TagRowsPerStrip) }
func (t *TIFF) FillOrder() (uint16, error)    { return t.GetFieldUint16(TagFillOrder) }
func (t *TIFF) Predictor() (uint16, error)    { return t.GetFieldUint16(TagPredictor) }
func (t *TIFF) TileWidth() (uint32, error)    { return t.GetFieldUint32(TagTileWidth) }
func (t *TIFF) TileLength() (uint32, error)   { return t.GetFieldUint32(TagTileLength) }

func (t *TIFF) XResolution() (float64, error) { return t.GetFieldFloat(TagXResolution) }
func (t *TIFF) YResolution() (float64, error) { return t.GetFieldFloat(TagYResolution) }

func (t *TIFF) ResolutionUnit() (uint16, error) { return t.GetFieldUint16(TagResolutionUnit) }

// Software returns the TIFFTAG_SOFTWARE string.
func (t *TIFF) Software() (string, error) { return t.GetFieldString(TagSoftware) }

// DateTime returns the TIFFTAG_DATETIME string.
func (t *TIFF) DateTime() (string, error) { return t.GetFieldString(TagDateTime) }

// ImageDescription returns the TIFFTAG_IMAGEDESCRIPTION string.
func (t *TIFF) ImageDescription() (string, error) {
	return t.GetFieldString(TagImageDescription)
}
