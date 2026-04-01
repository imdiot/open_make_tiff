package golibtiff

// Tag represents a TIFF tag identifier.
type Tag uint32

const (
	TagNewSubfileType    Tag = 254
	TagSubfileType       Tag = 255
	TagImageWidth        Tag = 256
	TagImageLength       Tag = 257
	TagBitsPerSample     Tag = 258
	TagCompression       Tag = 259
	TagPhotometric       Tag = 262
	TagThresholding      Tag = 263
	TagFillOrder         Tag = 266
	TagDocumentName      Tag = 269
	TagImageDescription  Tag = 270
	TagMake              Tag = 271
	TagModel             Tag = 272
	TagStripOffsets      Tag = 273
	TagOrientation       Tag = 274
	TagSamplesPerPixel   Tag = 277
	TagRowsPerStrip      Tag = 278
	TagStripByteCounts   Tag = 279
	TagXResolution       Tag = 282
	TagYResolution       Tag = 283
	TagPlanarConfig      Tag = 284
	TagResolutionUnit    Tag = 296
	TagSoftware          Tag = 305
	TagDateTime          Tag = 306
	TagArtist            Tag = 315
	TagPredictor         Tag = 317
	TagColorMap          Tag = 320
	TagTileWidth         Tag = 322
	TagTileLength        Tag = 323
	TagTileOffsets       Tag = 324
	TagTileByteCounts    Tag = 325
	TagSubIFD            Tag = 330
	TagExtraSamples      Tag = 338
	TagSampleFormat      Tag = 339
	TagJPEGTables        Tag = 347
	TagYCbCrSubSampling  Tag = 530
	TagReferenceBlackWhite Tag = 532
	TagCopyright         Tag = 33432
	TagIccProfile        Tag = 34675
	TagEXIFIFD           Tag = 34665
	TagGPSIFD            Tag = 34853
)

// Photometric interpretation constants.
const (
	PhotometricMinIsWhite = 0
	PhotometricMinIsBlack = 1
	PhotometricRGB        = 2
	PhotometricPalette    = 3
	PhotometricMask       = 4
	PhotometricSeparated  = 5
	PhotometricYCbCr      = 6
	PhotometricCIELab     = 8
	PhotometricICCLab     = 9
	PhotometricITULab     = 10
	PhotometricLogL       = 32844
	PhotometricLogLUV     = 32845
)

// Compression constants.
const (
	CompressionNone       = 1
	CompressionCCITTRLE   = 2
	CompressionCCITTFax3  = 3
	CompressionCCITTFax4  = 4
	CompressionLZW        = 5
	CompressionJPEG       = 7
	CompressionDeflate    = 8     // Adobe deflate
	CompressionPackBits   = 32773
	CompressionDeflateOld = 32946 // Deprecated
	CompressionLERC       = 34887
	CompressionLZMA       = 34925
	CompressionZSTD       = 50000
	CompressionWebP       = 50001
)

// Planar configuration constants.
const (
	PlanarConfigContig   = 1
	PlanarConfigSeparate = 2
)

// Sample format constants.
const (
	SampleFormatUInt          = 1
	SampleFormatInt           = 2
	SampleFormatIEEEFP        = 3
	SampleFormatVoid          = 4
	SampleFormatComplexInt    = 5
	SampleFormatComplexIEEEFP = 6
)

// Orientation constants.
const (
	OrientationTopLeft     = 1
	OrientationTopRight    = 2
	OrientationBotRight    = 3
	OrientationBotLeft     = 4
	OrientationLeftTop     = 5
	OrientationRightTop    = 6
	OrientationRightBot    = 7
	OrientationLeftBot     = 8
)

// Resolution unit constants.
const (
	ResolutionUnitNone        = 1
	ResolutionUnitInch        = 2
	ResolutionUnitCentimeter  = 3
)

// Fill order constants.
const (
	FillOrderMSB2LSB = 1
	FillOrderLSB2MSB = 2
)
