package golibraw

/*
#include <libraw/libraw.h>
*/
import "C"

func cMat3x3(a [3][3]C.float) [3][3]float32 {
	return [3][3]float32{
		{float32(a[0][0]), float32(a[0][1]), float32(a[0][2])},
		{float32(a[1][0]), float32(a[1][1]), float32(a[1][2])},
		{float32(a[2][0]), float32(a[2][1]), float32(a[2][2])},
	}
}

func cMat4x3(a [4][3]C.double) [4][3]float64 {
	return [4][3]float64{
		{float64(a[0][0]), float64(a[0][1]), float64(a[0][2])},
		{float64(a[1][0]), float64(a[1][1]), float64(a[1][2])},
		{float64(a[2][0]), float64(a[2][1]), float64(a[2][2])},
		{float64(a[3][0]), float64(a[3][1]), float64(a[3][2])},
	}
}

func (rp *RawProcessor) GetCanonMakernotes() (CanonMakernotes, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if err := rp.ensureOpen(); err != nil {
		return CanonMakernotes{}, err
	}
	c := rp.res.handle.makernotes.canon
	return CanonMakernotes{
		ColorDataVer:          int32(c.ColorDataVer),
		ColorDataSubVer:       int32(c.ColorDataSubVer),
		SpecularWhiteLevel:    int32(c.SpecularWhiteLevel),
		NormalWhiteLevel:      int32(c.NormalWhiteLevel),
		ChannelBlackLevel:     [4]int32{int32(c.ChannelBlackLevel[0]), int32(c.ChannelBlackLevel[1]), int32(c.ChannelBlackLevel[2]), int32(c.ChannelBlackLevel[3])},
		AverageBlackLevel:     int32(c.AverageBlackLevel),
		Multishot:             [4]uint32{uint32(c.multishot[0]), uint32(c.multishot[1]), uint32(c.multishot[2]), uint32(c.multishot[3])},
		MeteringMode:          int16(c.MeteringMode),
		SpotMeteringMode:      int16(c.SpotMeteringMode),
		FlashMeteringMode:     byte(c.FlashMeteringMode),
		FlashExposureLock:     int16(c.FlashExposureLock),
		ExposureMode:          int16(c.ExposureMode),
		AESetting:             int16(c.AESetting),
		ImageStabilization:    int16(c.ImageStabilization),
		FlashMode:             int16(c.FlashMode),
		FlashActivity:         int16(c.FlashActivity),
		FlashBits:             int16(c.FlashBits),
		ManualFlashOutput:     int16(c.ManualFlashOutput),
		FlashOutput:           int16(c.FlashOutput),
		FlashGuideNumber:      int16(c.FlashGuideNumber),
		ContinuousDrive:       int16(c.ContinuousDrive),
		SensorWidth:           int16(c.SensorWidth),
		SensorHeight:          int16(c.SensorHeight),
		AFMicroAdjMode:        int(c.AFMicroAdjMode),
		AFMicroAdjValue:       float32(c.AFMicroAdjValue),
		MakernotesFlip:        int16(c.MakernotesFlip),
		AutoRotateMode:        int16(c.AutoRotateMode),
		RecordMode:            int16(c.RecordMode),
		SRAWQuality:           int16(c.SRAWQuality),
		WBI:                   uint(c.wbi),
		RFLensID:              int16(c.RF_lensID),
		AutoLightingOptimizer: int(c.AutoLightingOptimizer),
		HighlightTonePriority: int(c.HighlightTonePriority),
		Quality:               int16(c.Quality),
		CanonLog:              int(c.CanonLog),
		DefaultCropAbsolute: Area{
			Top: int16(c.DefaultCropAbsolute.t), Left: int16(c.DefaultCropAbsolute.l),
			Bottom: int16(c.DefaultCropAbsolute.b), Right: int16(c.DefaultCropAbsolute.r),
		},
		RecommendedImageArea: Area{
			Top: int16(c.RecommendedImageArea.t), Left: int16(c.RecommendedImageArea.l),
			Bottom: int16(c.RecommendedImageArea.b), Right: int16(c.RecommendedImageArea.r),
		},
		LeftOpticalBlack: Area{
			Top: int16(c.LeftOpticalBlack.t), Left: int16(c.LeftOpticalBlack.l),
			Bottom: int16(c.LeftOpticalBlack.b), Right: int16(c.LeftOpticalBlack.r),
		},
		UpperOpticalBlack: Area{
			Top: int16(c.UpperOpticalBlack.t), Left: int16(c.UpperOpticalBlack.l),
			Bottom: int16(c.UpperOpticalBlack.b), Right: int16(c.UpperOpticalBlack.r),
		},
		ActiveArea: Area{
			Top: int16(c.ActiveArea.t), Left: int16(c.ActiveArea.l),
			Bottom: int16(c.ActiveArea.b), Right: int16(c.ActiveArea.r),
		},
		ISOGain: [2]int16{int16(c.ISOgain[0]), int16(c.ISOgain[1])},
	}, nil
}

func (rp *RawProcessor) GetNikonMakernotes() (NikonMakernotes, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if err := rp.ensureOpen(); err != nil {
		return NikonMakernotes{}, err
	}
	c := rp.res.handle.makernotes.nikon
	return NikonMakernotes{
		ExposureBracketValue:      float64(c.ExposureBracketValue),
		ActiveDLighting:           uint16(c.ActiveDLighting),
		ShootingMode:              uint16(c.ShootingMode),
		ImageStabilization:        [7]byte{byte(c.ImageStabilization[0]), byte(c.ImageStabilization[1]), byte(c.ImageStabilization[2]), byte(c.ImageStabilization[3]), byte(c.ImageStabilization[4]), byte(c.ImageStabilization[5]), byte(c.ImageStabilization[6])},
		VibrationReduction:        byte(c.VibrationReduction),
		VRMode:                    byte(c.VRMode),
		FlashSetting:              C.GoStringN(&c.FlashSetting[0], 13),
		FlashType:                 C.GoStringN(&c.FlashType[0], 20),
		FlashExposureCompensation: [4]byte{byte(c.FlashExposureCompensation[0]), byte(c.FlashExposureCompensation[1]), byte(c.FlashExposureCompensation[2]), byte(c.FlashExposureCompensation[3])},
		ExternalFlashExposureComp: [4]byte{byte(c.ExternalFlashExposureComp[0]), byte(c.ExternalFlashExposureComp[1]), byte(c.ExternalFlashExposureComp[2]), byte(c.ExternalFlashExposureComp[3])},
		FlashExposureBracketValue: [4]byte{byte(c.FlashExposureBracketValue[0]), byte(c.FlashExposureBracketValue[1]), byte(c.FlashExposureBracketValue[2]), byte(c.FlashExposureBracketValue[3])},
		FlashMode:                 byte(c.FlashMode),
		FlashExposureCompensation2: int8(c.FlashExposureCompensation2),
		FlashExposureCompensation3: int8(c.FlashExposureCompensation3),
		FlashExposureCompensation4: int8(c.FlashExposureCompensation4),
		FlashSource:               byte(c.FlashSource),
		FlashFirmware:             [2]byte{byte(c.FlashFirmware[0]), byte(c.FlashFirmware[1])},
		ExternalFlashFlags:        byte(c.ExternalFlashFlags),
		FlashControlCommanderMode: byte(c.FlashControlCommanderMode),
		FlashOutputAndCompensation: byte(c.FlashOutputAndCompensation),
		FlashFocalLength:          byte(c.FlashFocalLength),
		FlashGNDistance:           byte(c.FlashGNDistance),
		FlashGroupControlMode:     [4]byte{byte(c.FlashGroupControlMode[0]), byte(c.FlashGroupControlMode[1]), byte(c.FlashGroupControlMode[2]), byte(c.FlashGroupControlMode[3])},
		FlashGroupOutputComp:      [4]byte{byte(c.FlashGroupOutputAndCompensation[0]), byte(c.FlashGroupOutputAndCompensation[1]), byte(c.FlashGroupOutputAndCompensation[2]), byte(c.FlashGroupOutputAndCompensation[3])},
		FlashColorFilter:          byte(c.FlashColorFilter),
		NEFCompression:            uint16(c.NEFCompression),
		ExposureMode:              int(c.ExposureMode),
		ExposureProgram:           int(c.ExposureProgram),
		NMEshots:                  int(c.nMEshots),
		MEgainOn:                  int(c.MEgainOn),
		MEWB:                      [4]float64{float64(c.ME_WB[0]), float64(c.ME_WB[1]), float64(c.ME_WB[2]), float64(c.ME_WB[3])},
		AFFineTune:                byte(c.AFFineTune),
		AFFineTuneIndex:           byte(c.AFFineTuneIndex),
		AFFineTuneAdj:             int8(c.AFFineTuneAdj),
		LensDataVersion:           uint(c.LensDataVersion),
		FlashInfoVersion:          uint(c.FlashInfoVersion),
		ColorBalanceVersion:       uint(c.ColorBalanceVersion),
		Key:                       byte(c.key),
		NEFBitDepth:               [4]uint16{uint16(c.NEFBitDepth[0]), uint16(c.NEFBitDepth[1]), uint16(c.NEFBitDepth[2]), uint16(c.NEFBitDepth[3])},
		HighSpeedCropFormat:       uint16(c.HighSpeedCropFormat),
		SensorWidth:               uint16(c.SensorWidth),
		SensorHeight:              uint16(c.SensorHeight),
		ActiveDL:                  uint16(c.Active_D_Lighting),
		PictureControlVersion:     uint(c.PictureControlVersion),
		PictureControlName:        C.GoStringN(&c.PictureControlName[0], 20),
		PictureControlBase:        C.GoStringN(&c.PictureControlBase[0], 20),
		ShotInfoVersion:           uint(c.ShotInfoVersion),
		ShotInfoFirmware:          C.GoStringN(&c.ShotInfoFirmware[0], 9),
		MakernotesFlip:            int16(c.MakernotesFlip),
		RollAngle:                 float64(c.RollAngle),
		PitchAngle:                float64(c.PitchAngle),
		YawAngle:                  float64(c.YawAngle),
		SensorHighSpeedCrop: SensorHighSpeedCrop{
			Left:   uint16(c.SensorHighSpeedCrop.cleft),
			Top:    uint16(c.SensorHighSpeedCrop.ctop),
			Width:  uint16(c.SensorHighSpeedCrop.cwidth),
			Height: uint16(c.SensorHighSpeedCrop.cheight),
		},
	}, nil
}

func (rp *RawProcessor) GetFujiMakernotes() (FujiMakernotes, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if err := rp.ensureOpen(); err != nil {
		return FujiMakernotes{}, err
	}
	c := rp.res.handle.makernotes.fuji
	return FujiMakernotes{
		ExpoMidPointShift:       float32(c.ExpoMidPointShift),
		DynamicRange:            uint16(c.DynamicRange),
		FilmMode:                uint16(c.FilmMode),
		DynamicRangeSetting:     uint16(c.DynamicRangeSetting),
		DevelopmentDynamicRange: uint16(c.DevelopmentDynamicRange),
		AutoDynamicRange:        uint16(c.AutoDynamicRange),
		DRangePriority:          uint16(c.DRangePriority),
		DRangePriorityAuto:      uint16(c.DRangePriorityAuto),
		DRangePriorityFixed:     uint16(c.DRangePriorityFixed),
		FujiModel:               C.GoString(&c.FujiModel[0]),
		FujiModel2:              C.GoString(&c.FujiModel2[0]),
		BrightnessCompensation:  float32(c.BrightnessCompensation),
		FocusMode:               uint16(c.FocusMode),
		AFMode:                  uint16(c.AFMode),
		FocusPixel:              [2]uint16{uint16(c.FocusPixel[0]), uint16(c.FocusPixel[1])},
		PrioritySettings:        uint16(c.PrioritySettings),
		FocusSettings:           uint(c.FocusSettings),
		AFCSettings:             uint(c.AF_C_Settings),
		FocusWarning:            uint16(c.FocusWarning),
		ImageStabilization:      [3]uint16{uint16(c.ImageStabilization[0]), uint16(c.ImageStabilization[1]), uint16(c.ImageStabilization[2])},
		FlashMode:               uint16(c.FlashMode),
		WBPreset:                uint16(c.WB_Preset),
		ShutterType:             uint16(c.ShutterType),
		ExrMode:                 uint16(c.ExrMode),
		Macro:                   uint16(c.Macro),
		Rating:                  uint(c.Rating),
		CropMode:                uint16(c.CropMode),
		SerialSignature:         C.GoString(&c.SerialSignature[0]),
		SensorID:                C.GoString(&c.SensorID[0]),
		RAFVersion:              C.GoString(&c.RAFVersion[0]),
		RAFDataGeneration:       int(c.RAFDataGeneration),
		RAFDataVersion:          uint16(c.RAFDataVersion),
		IsTSNERDTS:              int(c.isTSNERDTS),
		DriveMode:               int16(c.DriveMode),
		BlackLevel: [9]uint16{uint16(c.BlackLevel[0]), uint16(c.BlackLevel[1]), uint16(c.BlackLevel[2]),
			uint16(c.BlackLevel[3]), uint16(c.BlackLevel[4]), uint16(c.BlackLevel[5]),
			uint16(c.BlackLevel[6]), uint16(c.BlackLevel[7]), uint16(c.BlackLevel[8])},
		AutoBracketing:  int(c.AutoBracketing),
		SequenceNumber:  int(c.SequenceNumber),
		SeriesLength:    int(c.SeriesLength),
		PixelShiftOffset: [2]float32{float32(c.PixelShiftOffset[0]), float32(c.PixelShiftOffset[1])},
		ImageCount:      int(c.ImageCount),
		RAFDataImageSizeTable: func() [32]uint {
			var t [32]uint
			for i := range 32 {
				t[i] = uint(c.RAFData_ImageSizeTable[i])
			}
			return t
		}(),
	}, nil
}

func (rp *RawProcessor) GetOlympusMakernotes() (OlympusMakernotes, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if err := rp.ensureOpen(); err != nil {
		return OlympusMakernotes{}, err
	}
	c := rp.res.handle.makernotes.olympus
	return OlympusMakernotes{
		CameraType2:       C.GoStringN(&c.CameraType2[0], 6),
		ValidBits:         uint16(c.ValidBits),
		TagX640:           uint(c.tagX640),
		TagX641:           uint(c.tagX641),
		TagX642:           uint(c.tagX642),
		TagX643:           uint(c.tagX643),
		TagX644:           uint(c.tagX644),
		TagX645:           uint(c.tagX645),
		TagX646:           uint(c.tagX646),
		TagX647:           uint(c.tagX647),
		TagX648:           uint(c.tagX648),
		TagX649:           uint(c.tagX649),
		TagX650:           uint(c.tagX650),
		TagX651:           uint(c.tagX651),
		TagX652:           uint(c.tagX652),
		TagX653:           uint(c.tagX653),
		SensorCalibration: [2]int{int(c.SensorCalibration[0]), int(c.SensorCalibration[1])},
		DriveMode:         [5]uint16{uint16(c.DriveMode[0]), uint16(c.DriveMode[1]), uint16(c.DriveMode[2]), uint16(c.DriveMode[3]), uint16(c.DriveMode[4])},
		ColorSpace:        uint16(c.ColorSpace),
		FocusMode:         [2]uint16{uint16(c.FocusMode[0]), uint16(c.FocusMode[1])},
		AutoFocus:         uint16(c.AutoFocus),
		AFPoint:           uint16(c.AFPoint),
		AFAreas: func() [64]uint {
			var a [64]uint
			for i := range 64 {
				a[i] = uint(c.AFAreas[i])
			}
			return a
		}(),
		AFPointSelected: [5]float64{float64(c.AFPointSelected[0]), float64(c.AFPointSelected[1]), float64(c.AFPointSelected[2]), float64(c.AFPointSelected[3]), float64(c.AFPointSelected[4])},
		AFResult:        uint16(c.AFResult),
		AFFineTune:      byte(c.AFFineTune),
		AFFineTuneAdj:   [3]int16{int16(c.AFFineTuneAdj[0]), int16(c.AFFineTuneAdj[1]), int16(c.AFFineTuneAdj[2])},
		SpecialMode: func() [3]uint {
			var s [3]uint
			for i := range 3 {
				s[i] = uint(c.SpecialMode[i])
			}
			return s
		}(),
		ZoomStepCount:     uint16(c.ZoomStepCount),
		FocusStepCount:    uint16(c.FocusStepCount),
		FocusStepInfinity: uint16(c.FocusStepInfinity),
		FocusStepNear:     uint16(c.FocusStepNear),
		FocusDistance:      float64(c.FocusDistance),
		AspectFrame:       [4]uint16{uint16(c.AspectFrame[0]), uint16(c.AspectFrame[1]), uint16(c.AspectFrame[2]), uint16(c.AspectFrame[3])},
		StackedImage:      [2]uint{uint(c.StackedImage[0]), uint(c.StackedImage[1])},
		IsLiveND:          byte(c.isLiveND),
		LiveNDfactor:      uint(c.LiveNDfactor),
		PanoramaMode:      uint16(c.Panorama_mode),
		PanoramaFrameNum:  uint16(c.Panorama_frameNum),
	}, nil
}

func (rp *RawProcessor) GetSonyMakernotes() (SonyMakernotes, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if err := rp.ensureOpen(); err != nil {
		return SonyMakernotes{}, err
	}
	c := rp.res.handle.makernotes.sony
	return SonyMakernotes{
		CameraType:                   uint16(c.CameraType),
		Sony0x9400Version:            byte(c.Sony0x9400_version),
		Sony0x9400ReleaseMode2:       byte(c.Sony0x9400_ReleaseMode2),
		Sony0x9400SequenceImageNumber: uint(c.Sony0x9400_SequenceImageNumber),
		Sony0x9400SequenceLength1:     byte(c.Sony0x9400_SequenceLength1),
		Sony0x9400SequenceFileNumber:  uint(c.Sony0x9400_SequenceFileNumber),
		Sony0x9400SequenceLength2:     byte(c.Sony0x9400_SequenceLength2),
		AFAreaModeSetting:            byte(c.AFAreaModeSetting),
		AFAreaMode:                   uint16(c.AFAreaMode),
		FlexibleSpotPosition:         [2]uint16{uint16(c.FlexibleSpotPosition[0]), uint16(c.FlexibleSpotPosition[1])},
		AFPointSelected:              byte(c.AFPointSelected),
		AFPointSelected0x201e:        byte(c.AFPointSelected_0x201e),
		NAFPointsUsed:                int16(c.nAFPointsUsed),
		AFPointsUsed:                 [10]byte{byte(c.AFPointsUsed[0]), byte(c.AFPointsUsed[1]), byte(c.AFPointsUsed[2]), byte(c.AFPointsUsed[3]), byte(c.AFPointsUsed[4]), byte(c.AFPointsUsed[5]), byte(c.AFPointsUsed[6]), byte(c.AFPointsUsed[7]), byte(c.AFPointsUsed[8]), byte(c.AFPointsUsed[9])},
		AFTracking:                   byte(c.AFTracking),
		AFType:                       byte(c.AFType),
		FocusLocation:                [4]uint16{uint16(c.FocusLocation[0]), uint16(c.FocusLocation[1]), uint16(c.FocusLocation[2]), uint16(c.FocusLocation[3])},
		FocusPosition:                uint16(c.FocusPosition),
		AFMicroAdjValue:              int8(c.AFMicroAdjValue),
		AFMicroAdjOn:                 int8(c.AFMicroAdjOn),
		AFMicroAdjRegisteredLenses:   byte(c.AFMicroAdjRegisteredLenses),
		VariableLowPassFilter:        uint16(c.VariableLowPassFilter),
		LongExposureNoiseReduction:   uint(c.LongExposureNoiseReduction),
		HighISONoiseReduction:         uint16(c.HighISONoiseReduction),
		HDR:                          [2]uint16{uint16(c.HDR[0]), uint16(c.HDR[1])},
		Group2010:                    uint16(c.group2010),
		Group9050:                    uint16(c.group9050),
		RealISOOffset:                uint16(c.real_iso_offset),
		MeteringModeOffset:           uint16(c.MeteringMode_offset),
		ExposureProgramOffset:        uint16(c.ExposureProgram_offset),
		ReleaseMode2Offset:           uint16(c.ReleaseMode2_offset),
		MinoltaCamID:                 uint(c.MinoltaCamID),
		Firmware:                     float32(c.firmware),
		ImageCount3Offset:            uint16(c.ImageCount3_offset),
		ImageCount3:                  uint(c.ImageCount3),
		ElectronicFrontCurtainShutter: uint(c.ElectronicFrontCurtainShutter),
		MeteringMode2:                uint16(c.MeteringMode2),
		SonyDateTime:                 C.GoStringN(&c.SonyDateTime[0], 20),
		ShotNumberSincePowerUp:       uint(c.ShotNumberSincePowerUp),
		PixelShiftGroupPrefix:        uint16(c.PixelShiftGroupPrefix),
		PixelShiftGroupID:            uint(c.PixelShiftGroupID),
		NShotsInPixelShiftGroup:      byte(c.nShotsInPixelShiftGroup),
		NumInPixelShiftGroup:         byte(c.numInPixelShiftGroup),
		PrdImageHeight:               uint16(c.prd_ImageHeight),
		PrdImageWidth:                uint16(c.prd_ImageWidth),
		PrdTotalBPS:                  uint16(c.prd_Total_bps),
		PrdActiveBPS:                 uint16(c.prd_Active_bps),
		PrdStorageMethod:             uint16(c.prd_StorageMethod),
		PrdBayerPattern:              uint16(c.prd_BayerPattern),
		SonyRawFileType:              uint16(c.SonyRawFileType),
		RAWFileType:                  uint16(c.RAWFileType),
		RawSizeType:                  uint16(c.RawSizeType),
		Quality:                      uint(c.Quality),
		FileFormat:                   uint16(c.FileFormat),
		MetaVersion:                  C.GoStringN(&c.MetaVersion[0], 16),
		AspectRatio:                  float32(c.AspectRatio),
	}, nil
}

func (rp *RawProcessor) GetKodakMakernotes() (KodakMakernotes, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if err := rp.ensureOpen(); err != nil {
		return KodakMakernotes{}, err
	}
	c := rp.res.handle.makernotes.kodak
	return KodakMakernotes{
		BlackLevelTop:    uint16(c.BlackLevelTop),
		BlackLevelBottom: uint16(c.BlackLevelBottom),
		OffsetLeft:       int16(c.offset_left),
		OffsetTop:        int16(c.offset_top),
		ClipBlack:        uint16(c.clipBlack),
		ClipWhite:        uint16(c.clipWhite),
		ROMMCamDaylight:  cMat3x3(c.romm_camDaylight),
		ROMMCamTungsten:  cMat3x3(c.romm_camTungsten),
		ROMMCamFluorescent: cMat3x3(c.romm_camFluorescent),
		ROMMCamFlash:     cMat3x3(c.romm_camFlash),
		ROMMCamCustom:    cMat3x3(c.romm_camCustom),
		ROMMCamAuto:      cMat3x3(c.romm_camAuto),
		Val018percent:      uint16(c.val018percent),
		Val100percent:      uint16(c.val100percent),
		Val170percent:      uint16(c.val170percent),
		MakerNoteKodak8a:   int16(c.MakerNoteKodak8a),
		ISOCalibrationGain: float32(c.ISOCalibrationGain),
		AnalogISO:          float32(c.AnalogISO),
	}, nil
}

func (rp *RawProcessor) GetPanasonicMakernotes() (PanasonicMakernotes, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if err := rp.ensureOpen(); err != nil {
		return PanasonicMakernotes{}, err
	}
	c := rp.res.handle.makernotes.panasonic
	return PanasonicMakernotes{
		Compression:   uint16(c.Compression),
		BlackLevelDim: uint16(c.BlackLevelDim),
		BlackLevel:    [8]float32{float32(c.BlackLevel[0]), float32(c.BlackLevel[1]), float32(c.BlackLevel[2]), float32(c.BlackLevel[3]), float32(c.BlackLevel[4]), float32(c.BlackLevel[5]), float32(c.BlackLevel[6]), float32(c.BlackLevel[7])},
		Multishot:     uint(c.Multishot),
		Gamma:         float32(c.gamma),
		HighISOMultiplier: [3]int{int(c.HighISOMultiplier[0]), int(c.HighISOMultiplier[1]), int(c.HighISOMultiplier[2])},
		FocusStepNear: int16(c.FocusStepNear),
		FocusStepCount: int16(c.FocusStepCount),
		ZoomPosition:  uint(c.ZoomPosition),
		LensManufacturer: uint(c.LensManufacturer),
	}, nil
}

func (rp *RawProcessor) GetPentaxMakernotes() (PentaxMakernotes, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if err := rp.ensureOpen(); err != nil {
		return PentaxMakernotes{}, err
	}
	c := rp.res.handle.makernotes.pentax
	return PentaxMakernotes{
		DriveMode:             [4]byte{byte(c.DriveMode[0]), byte(c.DriveMode[1]), byte(c.DriveMode[2]), byte(c.DriveMode[3])},
		FocusMode:             [2]uint16{uint16(c.FocusMode[0]), uint16(c.FocusMode[1])},
		AFPointSelected:       [2]uint16{uint16(c.AFPointSelected[0]), uint16(c.AFPointSelected[1])},
		AFPointSelectedArea:   uint16(c.AFPointSelected_Area),
		AFPointsInFocusVersion: int(c.AFPointsInFocus_version),
		AFPointsInFocus:       uint(c.AFPointsInFocus),
		FocusPosition:         uint16(c.FocusPosition),
		DynamicRangeExpansion: [4]byte{byte(c.DynamicRangeExpansion[0]), byte(c.DynamicRangeExpansion[1]), byte(c.DynamicRangeExpansion[2]), byte(c.DynamicRangeExpansion[3])},
		AFAdjustment:          int16(c.AFAdjustment),
		AFPointMode:           byte(c.AFPointMode),
		MultiExposure:         byte(c.MultiExposure),
		Quality:               uint16(c.Quality),
	}, nil
}

func (rp *RawProcessor) GetPhaseOneMakernotes() (PhaseOneMakernotes, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if err := rp.ensureOpen(); err != nil {
		return PhaseOneMakernotes{}, err
	}
	c := rp.res.handle.makernotes.phaseone
	return PhaseOneMakernotes{
		Software:       C.GoString(&c.Software[0]),
		SystemType:     C.GoString(&c.SystemType[0]),
		FirmwareString: C.GoString(&c.FirmwareString[0]),
		SystemModel:    C.GoString(&c.SystemModel[0]),
	}, nil
}

func (rp *RawProcessor) GetRicohMakernotes() (RicohMakernotes, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if err := rp.ensureOpen(); err != nil {
		return RicohMakernotes{}, err
	}
	c := rp.res.handle.makernotes.ricoh
	return RicohMakernotes{
		AFStatus:           uint16(c.AFStatus),
		AFAreaXPosition:    [2]uint{uint(c.AFAreaXPosition[0]), uint(c.AFAreaXPosition[1])},
		AFAreaYPosition:    [2]uint{uint(c.AFAreaYPosition[0]), uint(c.AFAreaYPosition[1])},
		AFAreaMode:         uint16(c.AFAreaMode),
		SensorWidth:        uint(c.SensorWidth),
		SensorHeight:       uint(c.SensorHeight),
		CroppedImageWidth:  uint(c.CroppedImageWidth),
		CroppedImageHeight: uint(c.CroppedImageHeight),
		WideAdapter:        uint16(c.WideAdapter),
		CropMode:           uint16(c.CropMode),
		NDFilter:           uint16(c.NDFilter),
		AutoBracketing:     uint16(c.AutoBracketing),
		MacroMode:          uint16(c.MacroMode),
		FlashMode:          uint16(c.FlashMode),
		FlashExposureComp:  float64(c.FlashExposureComp),
		ManualFlashOutput:  float64(c.ManualFlashOutput),
	}, nil
}

func (rp *RawProcessor) GetSamsungMakernotes() (SamsungMakernotes, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if err := rp.ensureOpen(); err != nil {
		return SamsungMakernotes{}, err
	}
	c := rp.res.handle.makernotes.samsung
	return SamsungMakernotes{
		ImageSizeFull: [4]uint{uint(c.ImageSizeFull[0]), uint(c.ImageSizeFull[1]), uint(c.ImageSizeFull[2]), uint(c.ImageSizeFull[3])},
		ImageSizeCrop: [4]uint{uint(c.ImageSizeCrop[0]), uint(c.ImageSizeCrop[1]), uint(c.ImageSizeCrop[2]), uint(c.ImageSizeCrop[3])},
		ColorSpace:    [2]int{int(c.ColorSpace[0]), int(c.ColorSpace[1])},
		Key: func() [11]uint {
			var k [11]uint
			for i := range 11 {
				k[i] = uint(c.key[i])
			}
			return k
		}(),
		DigitalGain:  float64(c.DigitalGain),
		DeviceType:   int(c.DeviceType),
		LensFirmware: C.GoString(&c.LensFirmware[0]),
	}, nil
}

func (rp *RawProcessor) GetHasselbladMakernotes() (HasselbladMakernotes, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if err := rp.ensureOpen(); err != nil {
		return HasselbladMakernotes{}, err
	}
	c := rp.res.handle.makernotes.hasselblad
	return HasselbladMakernotes{
		BaseISO:                  int(c.BaseISO),
		Gain:                     float64(c.Gain),
		Sensor:                   C.GoString(&c.Sensor[0]),
		SensorUnit:               C.GoString(&c.SensorUnit[0]),
		HostBody:                 C.GoString(&c.HostBody[0]),
		SensorCode:               int(c.SensorCode),
		SensorSubCode:            int(c.SensorSubCode),
		CoatingCode:              int(c.CoatingCode),
		Uncropped:                int(c.uncropped),
		CaptureSequenceInitiator: C.GoString(&c.CaptureSequenceInitiator[0]),
		SensorUnitConnector:      C.GoString(&c.SensorUnitConnector[0]),
		Format:                   int(c.format),
		NIFDCM:                   [2]int{int(c.nIFD_CM[0]), int(c.nIFD_CM[1])},
		RecommendedCrop:          [2]int{int(c.RecommendedCrop[0]), int(c.RecommendedCrop[1])},
		MNColorMatrix:            cMat4x3(c.mnColorMatrix),
	}, nil
}
