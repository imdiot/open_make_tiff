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
	TagXMP               Tag = 700 // XMP metadata (BYTE array)

	// DNG tags (IFD0)
	TagUniqueCameraModel    Tag = 50708
	TagLocalizedCameraModel Tag = 50709
	TagAsShotNeutral        Tag = 50728

	// EXIF Sub-IFD tags
	TagExifExposureTime              Tag = 33434
	TagExifFNumber                   Tag = 33437
	TagExifExposureProgram           Tag = 34850
	TagExifISO                       Tag = 34855
	TagExifSensitivityType           Tag = 34864
	TagExifStandardOutputSensitivity Tag = 34865
	TagExifShutterSpeedValue         Tag = 37377
	TagExifApertureValue             Tag = 37378
	TagExifBrightnessValue           Tag = 37379
	TagExifExposureCompensation      Tag = 37380
	TagExifMaxApertureValue          Tag = 37381
	TagExifMeteringMode              Tag = 37383
	TagExifLightSource               Tag = 37384
	TagExifFlash                     Tag = 37385
	TagExifFocalLength               Tag = 37386
	TagExifMakerNote                 Tag = 37500
	TagExifDateTimeOriginal          Tag = 36867
	TagExifCreateDate                Tag = 36868
	TagExifOffsetTime                Tag = 36880
	TagExifOffsetTimeOriginal        Tag = 36881
	TagExifOffsetTimeDigitized       Tag = 36882
	TagExifSensingMethod             Tag = 41495
	TagExifCustomRendered            Tag = 41985
	TagExifExposureMode              Tag = 41986
	TagExifWhiteBalance              Tag = 41987
	TagExifSceneCaptureType          Tag = 41990
	TagExifSharpness                 Tag = 41994
	TagExifSerialNumber              Tag = 42033
	TagExifLensInfo                  Tag = 42034
	TagExifLensMake                  Tag = 42035
	TagExifLensModel                 Tag = 42036
	TagExifLensSerialNumber          Tag = 42037
	TagExifColorSpace                Tag = 40961
	TagExifImageWidth                Tag = 40962
	TagExifImageHeight               Tag = 40963
	TagExifGamma                     Tag = 42240
	TagExifSubjectDistanceRange      Tag = 41996
	TagExifSceneType                 Tag = 41729
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

// Predictor constants.
const (
	PredictorNone          = 1
	PredictorHorizontal    = 2
	PredictorFloatingPoint = 3
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
	ResolutionUnitNone       = 1
	ResolutionUnitInch       = 2
	ResolutionUnitCentimeter = 3
)

// Fill order constants.
const (
	FillOrderMSB2LSB = 1
	FillOrderLSB2MSB = 2
)
