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
type Photometric uint16

const (
	PhotometricMinIsWhite Photometric = 0
	PhotometricMinIsBlack Photometric = 1
	PhotometricRGB        Photometric = 2
	PhotometricPalette    Photometric = 3
	PhotometricMask       Photometric = 4
	PhotometricSeparated  Photometric = 5
	PhotometricYCbCr      Photometric = 6
	PhotometricCIELab     Photometric = 8
	PhotometricICCLab     Photometric = 9
	PhotometricITULab     Photometric = 10
	PhotometricLogL       Photometric = 32844
	PhotometricLogLUV     Photometric = 32845
)

// Compression constants.
type Compression uint16

const (
	CompressionNone       Compression = 1
	CompressionCCITTRLE   Compression = 2
	CompressionCCITTFax3  Compression = 3
	CompressionCCITTFax4  Compression = 4
	CompressionLZW        Compression = 5
	CompressionJPEG       Compression = 7
	CompressionDeflate    Compression = 8     // Adobe deflate
	CompressionPackBits   Compression = 32773
	CompressionDeflateOld Compression = 32946 // Deprecated
	CompressionLERC       Compression = 34887
	CompressionLZMA       Compression = 34925
	CompressionZSTD       Compression = 50000
	CompressionWebP       Compression = 50001
)

// Predictor constants.
type Predictor uint16

const (
	PredictorNone          Predictor = 1
	PredictorHorizontal    Predictor = 2
	PredictorFloatingPoint Predictor = 3
)

// Planar configuration constants.
type PlanarConfig uint16

const (
	PlanarConfigContig   PlanarConfig = 1
	PlanarConfigSeparate PlanarConfig = 2
)

// Sample format constants.
type SampleFormat uint16

const (
	SampleFormatUint          SampleFormat = 1
	SampleFormatInt           SampleFormat = 2
	SampleFormatIEEEFP        SampleFormat = 3
	SampleFormatVoid          SampleFormat = 4
	SampleFormatComplexInt    SampleFormat = 5
	SampleFormatComplexIEEEFP SampleFormat = 6
)

// Orientation constants.
type Orientation uint16

const (
	OrientationTopLeft  Orientation = 1
	OrientationTopRight Orientation = 2
	OrientationBotRight Orientation = 3
	OrientationBotLeft  Orientation = 4
	OrientationLeftTop  Orientation = 5
	OrientationRightTop Orientation = 6
	OrientationRightBot Orientation = 7
	OrientationLeftBot  Orientation = 8
)

// Resolution unit constants.
type ResolutionUnit uint16

const (
	ResolutionUnitNone       ResolutionUnit = 1
	ResolutionUnitInch       ResolutionUnit = 2
	ResolutionUnitCentimeter ResolutionUnit = 3
)

// Fill order constants.
type FillOrder uint16

const (
	FillOrderMSB2LSB FillOrder = 1
	FillOrderLSB2MSB FillOrder = 2
)

// Pseudo-tags control codec behavior. They are not written to the TIFF file
// and are passed to SetFieldUint16/SetFieldUint32 before writing data.
const (
	PseudoTagJPEGQuality       Tag = 65537
	PseudoTagJPEGColorMode     Tag = 65538
	PseudoTagJPEGTablesMode    Tag = 65539
	PseudoTagZIPQuality        Tag = 65557
	PseudoTagLZMAPreset        Tag = 65562
	PseudoTagZSTDLevel         Tag = 65564
	PseudoTagLERCVersion       Tag = 65565
	PseudoTagLERCAddCompression Tag = 65566
	PseudoTagLERCMaxZError     Tag = 65567
	PseudoTagWebPLevel         Tag = 65568
	PseudoTagWebPLossless      Tag = 65569
	PseudoTagDeflateSubCodec   Tag = 65570
	PseudoTagWebPLosslessExact Tag = 65571
)

// DataType describes the data type of a tag's values.
type DataType int

const (
	DataTypeByte      DataType = 1  // 8-bit unsigned integer
	DataTypeASCII     DataType = 2  // Null-terminated string
	DataTypeShort     DataType = 3  // 16-bit unsigned integer
	DataTypeLong      DataType = 4  // 32-bit unsigned integer
	DataTypeRational  DataType = 5  // Two 32-bit unsigned (numerator/denominator)
	DataTypeSByte     DataType = 6  // 8-bit signed integer
	DataTypeUndefined DataType = 7  // 8-bit untyped data
	DataTypeSShort    DataType = 8  // 16-bit signed integer
	DataTypeSLong     DataType = 9  // 32-bit signed integer
	DataTypeSRational DataType = 10 // Two 32-bit signed
	DataTypeFloat     DataType = 11 // 32-bit IEEE float
	DataTypeDouble    DataType = 12 // 64-bit IEEE double
	DataTypeIFD       DataType = 13 // 32-bit IFD offset
	DataTypeLong8     DataType = 16 // 64-bit unsigned (BigTIFF)
	DataTypeSLong8    DataType = 17 // 64-bit signed (BigTIFF)
	DataTypeIFD8      DataType = 18 // 64-bit IFD offset (BigTIFF)
)
