package golibtiff

// Convenience methods for common TIFF tags. These are pure Go wrappers around GetField* methods.

func (t *TIFF) Width() uint32    { v, _ := t.GetFieldUint32(TagImageWidth); return v }
func (t *TIFF) Height() uint32   { v, _ := t.GetFieldUint32(TagImageLength); return v }
func (t *TIFF) BitsPerSample() uint16 {
	v, _ := t.GetFieldUint16(TagBitsPerSample)
	return v
}
func (t *TIFF) SamplesPerPixel() uint16 {
	v, _ := t.GetFieldUint16(TagSamplesPerPixel)
	return v
}
func (t *TIFF) Compression() uint16 {
	v, _ := t.GetFieldUint16(TagCompression)
	return v
}
func (t *TIFF) Photometric() uint16 { v, _ := t.GetFieldUint16(TagPhotometric); return v }
func (t *TIFF) PlanarConfig() uint16 {
	v, _ := t.GetFieldUint16(TagPlanarConfig)
	return v
}
func (t *TIFF) Orientation() uint16 {
	v, _ := t.GetFieldUint16(TagOrientation)
	return v
}
func (t *TIFF) SampleFormat() uint16 {
	v, _ := t.GetFieldUint16(TagSampleFormat)
	return v
}
func (t *TIFF) RowsPerStrip() uint32 {
	v, _ := t.GetFieldUint32(TagRowsPerStrip)
	return v
}
func (t *TIFF) FillOrder() uint16 { v, _ := t.GetFieldUint16(TagFillOrder); return v }
func (t *TIFF) Predictor() uint16 { v, _ := t.GetFieldUint16(TagPredictor); return v }
func (t *TIFF) TileWidth() uint32  { v, _ := t.GetFieldUint32(TagTileWidth); return v }
func (t *TIFF) TileLength() uint32 { v, _ := t.GetFieldUint32(TagTileLength); return v }

func (t *TIFF) XResolution() (float64, error) { return t.GetFieldFloat(TagXResolution) }
func (t *TIFF) YResolution() (float64, error) { return t.GetFieldFloat(TagYResolution) }

func (t *TIFF) ResolutionUnit() uint16 {
	v, _ := t.GetFieldUint16(TagResolutionUnit)
	return v
}

// Software returns the TIFFTAG_SOFTWARE string.
func (t *TIFF) Software() (string, error) { return t.GetFieldString(TagSoftware) }

// DateTime returns the TIFFTAG_DATETIME string.
func (t *TIFF) DateTime() (string, error) { return t.GetFieldString(TagDateTime) }

// ImageDescription returns the TIFFTAG_IMAGEDESCRIPTION string.
func (t *TIFF) ImageDescription() (string, error) {
	return t.GetFieldString(TagImageDescription)
}
