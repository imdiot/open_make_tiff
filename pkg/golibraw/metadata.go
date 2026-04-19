package golibraw

/*
#include <libraw/libraw.h>
*/
import "C"

import (
	"time"
	"unsafe"
)

func (rp *RawProcessor) GetImageSizes() (ImageSizes, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return ImageSizes{}, err
	}

	s := rp.res.handle.sizes
	img := ImageSizes{
		RawHeight:        uint16(s.raw_height),
		RawWidth:         uint16(s.raw_width),
		Height:           uint16(s.height),
		Width:            uint16(s.width),
		TopMargin:        uint16(s.top_margin),
		LeftMargin:       uint16(s.left_margin),
		IHeight:          uint16(s.iheight),
		IWidth:           uint16(s.iwidth),
		Flip:             int(s.flip),
		PixelAspectRatio: float64(s.pixel_aspect),
		RawPitch:         uint(s.raw_pitch),
		RawAspect:        uint16(s.raw_aspect),
		RawInsetCrops: [2]RawInsetCrop{
			{
				Left:   uint16(s.raw_inset_crops[0].cleft),
				Top:    uint16(s.raw_inset_crops[0].ctop),
				Width:  uint16(s.raw_inset_crops[0].cwidth),
				Height: uint16(s.raw_inset_crops[0].cheight),
			},
			{
				Left:   uint16(s.raw_inset_crops[1].cleft),
				Top:    uint16(s.raw_inset_crops[1].ctop),
				Width:  uint16(s.raw_inset_crops[1].cwidth),
				Height: uint16(s.raw_inset_crops[1].cheight),
			},
		},
	}
	for i := 0; i < 8; i++ {
		for j := 0; j < 4; j++ {
			img.Mask[i][j] = int(s.mask[i][j])
		}
	}
	return img, nil
}

func (rp *RawProcessor) GetCameraInfo() (CameraInfo, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return CameraInfo{}, err
	}

	ip := C.libraw_get_iparams(rp.res.handle)
	ci := CameraInfo{
		Make:            C.GoString(&ip.make[0]),
		Model:           C.GoString(&ip.model[0]),
		NormalizedMake:  C.GoString(&ip.normalized_make[0]),
		NormalizedModel: C.GoString(&ip.normalized_model[0]),
		Software:        C.GoString(&ip.software[0]),
		RawCount:        uint(ip.raw_count),
		DNGVersion:      uint(ip.dng_version),
		IsFoveon:        ip.is_foveon != 0,
		Colors:          int(ip.colors),
		Filters:         uint(ip.filters),
		CDesc:           C.GoString(&ip.cdesc[0]),
		MakerIndex:      uint(ip.maker_index),
		XMPLen:          uint(ip.xmplen),
	}
	for i := 0; i < 6; i++ {
		for j := 0; j < 6; j++ {
			ci.XTrans[i][j] = int8(ip.xtrans[i][j])
			ci.XTransAbs[i][j] = int8(ip.xtrans_abs[i][j])
		}
	}
	if ip.xmpdata != nil && ip.xmplen > 0 && ip.xmplen <= 0x7FFFFFFF {
		ci.XMPData = C.GoBytes(unsafe.Pointer(ip.xmpdata), C.int(ip.xmplen))
	}
	return ci, nil
}

func (rp *RawProcessor) GetLensInfo() (LensInfo, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return LensInfo{}, err
	}

	li := C.libraw_get_lensinfo(rp.res.handle)
	return LensInfo{
		LensMake:                C.GoString(&li.LensMake[0]),
		Lens:                    C.GoString(&li.Lens[0]),
		LensSerial:              C.GoString(&li.LensSerial[0]),
		MinFocal:                float32(li.MinFocal),
		MaxFocal:                float32(li.MaxFocal),
		MaxAp4MinFocal:          float32(li.MaxAp4MinFocal),
		MaxAp4MaxFocal:          float32(li.MaxAp4MaxFocal),
		CurFocal:                float32(li.makernotes.CurFocal),
		CurAp:                   float32(li.makernotes.CurAp),
		FocalLengthIn35mmFormat: uint16(li.FocalLengthIn35mmFormat),
		InternalLensSerial:      C.GoString(&li.InternalLensSerial[0]),
		EXIFMaxAp:               float32(li.EXIF_MaxAp),
		Nikon: NikonLensInfo{
			EffectiveMaxAp: float32(li.nikon.EffectiveMaxAp),
			LensIDNumber:   byte(li.nikon.LensIDNumber),
			LensFStops:     byte(li.nikon.LensFStops),
			MCUVersion:     byte(li.nikon.MCUVersion),
			LensType:       byte(li.nikon.LensType),
		},
		DNG: DNGLensInfo{
			MinFocal:       float32(li.dng.MinFocal),
			MaxFocal:       float32(li.dng.MaxFocal),
			MaxAp4MinFocal: float32(li.dng.MaxAp4MinFocal),
			MaxAp4MaxFocal: float32(li.dng.MaxAp4MaxFocal),
		},
	}, nil
}

func (rp *RawProcessor) GetShootingParams() (ShootingParams, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return ShootingParams{}, err
	}

	ot := C.libraw_get_imgother(rp.res.handle)
	ts := time.Unix(int64(ot.timestamp), 0)
	return ShootingParams{
		ISOSpeed:      float32(ot.iso_speed),
		Shutter:       float32(ot.shutter),
		Aperture:      float32(ot.aperture),
		FocalLen:      float32(ot.focal_len),
		Timestamp:     ts,
		ShotOrder:     uint(ot.shot_order),
		Artist:        C.GoString(&ot.artist[0]),
		Desc:          C.GoString(&ot.desc[0]),
		AnalogBalance: [4]float32{float32(ot.analogbalance[0]), float32(ot.analogbalance[1]), float32(ot.analogbalance[2]), float32(ot.analogbalance[3])},
	}, nil
}

func (rp *RawProcessor) GetGPS() (GPSInfo, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return GPSInfo{}, err
	}

	gps := rp.res.handle.other.parsed_gps
	return GPSInfo{
		Latitude:     [3]float32{float32(gps.latitude[0]), float32(gps.latitude[1]), float32(gps.latitude[2])},
		Longitude:    [3]float32{float32(gps.longitude[0]), float32(gps.longitude[1]), float32(gps.longitude[2])},
		GPSTimestamp: [3]float32{float32(gps.gpstimestamp[0]), float32(gps.gpstimestamp[1]), float32(gps.gpstimestamp[2])},
		Altitude:     float32(gps.altitude),
		AltRef:       byte(gps.altref),
		LatRef:       byte(gps.latref),
		LongRef:      byte(gps.longref),
		GPSStatus:    byte(gps.gpsstatus),
		GPSParsed:    gps.gpsparsed != 0,
	}, nil
}

func (rp *RawProcessor) GetShootingInfo() (ShootingInfo, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return ShootingInfo{}, err
	}

	si := rp.res.handle.shootinginfo
	return ShootingInfo{
		DriveMode:          int16(si.DriveMode),
		FocusMode:          int16(si.FocusMode),
		MeteringMode:       int16(si.MeteringMode),
		AFPoint:            int16(si.AFPoint),
		ExposureMode:       int16(si.ExposureMode),
		ExposureProgram:    int16(si.ExposureProgram),
		ImageStabilization: int16(si.ImageStabilization),
		BodySerial:         C.GoString(&si.BodySerial[0]),
		InternalBodySerial: C.GoString(&si.InternalBodySerial[0]),
	}, nil
}

func (rp *RawProcessor) GetMakernotesLens() (MakernotesLensInfo, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return MakernotesLensInfo{}, err
	}

	ml := rp.res.handle.lens.makernotes
	return MakernotesLensInfo{
		Lens:                    C.GoString(&ml.Lens[0]),
		LensFormat:              uint16(ml.LensFormat),
		LensMount:               uint16(ml.LensMount),
		CamID:                   uint64(ml.CamID),
		CameraFormat:            uint16(ml.CameraFormat),
		CameraMount:             uint16(ml.CameraMount),
		Body:                    C.GoString(&ml.body[0]),
		FocalType:               FocalType(ml.FocalType),
		LensFeaturesPre:         C.GoString(&ml.LensFeatures_pre[0]),
		LensFeaturesSuf:         C.GoString(&ml.LensFeatures_suf[0]),
		MinFocal:                float32(ml.MinFocal),
		MaxFocal:                float32(ml.MaxFocal),
		MaxAp4MinFocal:          float32(ml.MaxAp4MinFocal),
		MaxAp4MaxFocal:          float32(ml.MaxAp4MaxFocal),
		MinAp4MinFocal:          float32(ml.MinAp4MinFocal),
		MinAp4MaxFocal:          float32(ml.MinAp4MaxFocal),
		MaxAp:                   float32(ml.MaxAp),
		MinAp:                   float32(ml.MinAp),
		CurFocal:                float32(ml.CurFocal),
		CurAp:                   float32(ml.CurAp),
		MaxAp4CurFocal:          float32(ml.MaxAp4CurFocal),
		MinAp4CurFocal:          float32(ml.MinAp4CurFocal),
		MinFocusDistance:        float32(ml.MinFocusDistance),
		FocusRangeIndex:         float32(ml.FocusRangeIndex),
		LensFStops:              float32(ml.LensFStops),
		TeleconverterID:         uint64(ml.TeleconverterID),
		Teleconverter:           C.GoString(&ml.Teleconverter[0]),
		AdapterID:               uint64(ml.AdapterID),
		Adapter:                 C.GoString(&ml.Adapter[0]),
		AttachmentID:            uint64(ml.AttachmentID),
		Attachment:              C.GoString(&ml.Attachment[0]),
		FocalUnits:              uint16(ml.FocalUnits),
		FocalLengthIn35mmFormat: float32(ml.FocalLengthIn35mmFormat),
	}, nil
}

func (rp *RawProcessor) GetTemperatures() (SensorTemperatures, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return SensorTemperatures{}, err
	}

	c := rp.res.handle.makernotes.common
	return SensorTemperatures{
		CameraTemperature:       float32(c.CameraTemperature),
		SensorTemperature:       float32(c.SensorTemperature),
		SensorTemperature2:      float32(c.SensorTemperature2),
		LensTemperature:         float32(c.LensTemperature),
		AmbientTemperature:      float32(c.AmbientTemperature),
		BatteryTemperature:      float32(c.BatteryTemperature),
		ExifAmbientTemperature:  float32(c.exifAmbientTemperature),
		FlashEC:                 float32(c.FlashEC),
		FlashGN:                 float32(c.FlashGN),
		RealISO:                 float32(c.real_ISO),
		Firmware:                C.GoString(&c.firmware[0]),
		ExifHumidity:            float32(c.exifHumidity),
		ExifPressure:            float32(c.exifPressure),
		ExifWaterDepth:          float32(c.exifWaterDepth),
		ExifAcceleration:        float32(c.exifAcceleration),
		ExifCameraElevationAngle: float32(c.exifCameraElevationAngle),
		ExifExposureIndex:       float32(c.exifExposureIndex),
		ColorSpace:              uint16(c.ColorSpace),
		ExposureCalibrationShift: float32(c.ExposureCalibrationShift),
	}, nil
}

func (rp *RawProcessor) GetThumbnailInfo() (ThumbnailInfo, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return ThumbnailInfo{}, err
	}

	t := rp.res.handle.thumbnail
	return ThumbnailInfo{
		Format: ThumbnailFormat(t.tformat),
		Width:  uint16(t.twidth),
		Height: uint16(t.theight),
		Length: uint(t.tlength),
		Colors: int(t.tcolors),
	}, nil
}

func (rp *RawProcessor) GetColorData() (ColorData, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return ColorData{}, err
	}

	c := rp.res.handle.color
	cd := ColorData{
		Black:                uint(c.black),
		Maximum:              uint(c.maximum),
		DataMaximum:          uint(c.data_maximum),
		FMaximum:             float32(c.fmaximum),
		FNorm:                float32(c.fnorm),
		RawBPS:               uint(c.raw_bps),
		FlashUsed:            float32(c.flash_used),
		CanonEV:              float32(c.canon_ev),
		LinearMax:            [4]uint{uint(c.linear_max[0]), uint(c.linear_max[1]), uint(c.linear_max[2]), uint(c.linear_max[3])},
		CamMul:               [4]float32{float32(c.cam_mul[0]), float32(c.cam_mul[1]), float32(c.cam_mul[2]), float32(c.cam_mul[3])},
		PreMul:               [4]float32{float32(c.pre_mul[0]), float32(c.pre_mul[1]), float32(c.pre_mul[2]), float32(c.pre_mul[3])},
		PhaseOneData: PhaseOneData{
			Format:   int(c.phase_one_data.format),
			KeyOff:   int(c.phase_one_data.key_off),
			Tag21a:   int(c.phase_one_data.tag_21a),
			TBlack:   int(c.phase_one_data.t_black),
			SplitCol: int(c.phase_one_data.split_col),
			BlackCol: int(c.phase_one_data.black_col),
			SplitRow: int(c.phase_one_data.split_row),
			BlackRow: int(c.phase_one_data.black_row),
			Tag210:   float32(c.phase_one_data.tag_210),
		},
		UniqueCameraModel:    C.GoString(&c.UniqueCameraModel[0]),
		LocalizedCameraModel: C.GoString(&c.LocalizedCameraModel[0]),
		ImageUniqueID:        C.GoString(&c.ImageUniqueID[0]),
		RawDataUniqueID:      C.GoString(&c.RawDataUniqueID[0]),
		OriginalRawFileName:  C.GoString(&c.OriginalRawFileName[0]),
		Model2:               C.GoString(&c.model2[0]),
		HasICCProfile:        c.profile != nil,
		ICCProfileLength:     uint(c.profile_length),
		ExifColorSpace:       int(c.ExifColorSpace),
		AsShotWBApplied:      c.as_shot_wb_applied != 0,
	}
	for i := 0; i < 4; i++ {
		cd.CBlack[i] = uint(c.cblack[i])
	}
	for i := 0; i < 8; i++ {
		cd.BlackStat[i] = uint(c.black_stat[i])
	}
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			cd.White[i][j] = uint16(c.white[i][j])
		}
	}
	for k := 0; k < 2; k++ {
		for i := 0; i < 9; i++ {
			cd.P1Color[k].ROMMCam[i] = float32(c.P1_color[k].romm_cam[i])
		}
	}
	for i := 0; i < 3; i++ {
		for j := 0; j < 4; j++ {
			cd.CMatrix[i][j] = float32(c.cmatrix[i][j])
			cd.CCM[i][j] = float32(c.ccm[i][j])
			cd.RGBCam[i][j] = float32(c.rgb_cam[i][j])
		}
	}
	for i := 0; i < 4; i++ {
		for j := 0; j < 3; j++ {
			cd.CamXYZ[i][j] = float32(c.cam_xyz[i][j])
		}
	}
	return cd, nil
}

func (rp *RawProcessor) GetWBCoeffs() (map[WBIndex][4]int, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return nil, err
	}

	result := make(map[WBIndex][4]int)
	for i := 0; i < 256; i++ {
		if rp.res.handle.color.WB_Coeffs[i][0] != 0 || rp.res.handle.color.WB_Coeffs[i][1] != 0 {
			result[WBIndex(i)] = [4]int{
				int(rp.res.handle.color.WB_Coeffs[i][0]),
				int(rp.res.handle.color.WB_Coeffs[i][1]),
				int(rp.res.handle.color.WB_Coeffs[i][2]),
				int(rp.res.handle.color.WB_Coeffs[i][3]),
			}
		}
	}
	return result, nil
}

func (rp *RawProcessor) GetWBTempCoeffs() ([]WBTempCoeff, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return nil, err
	}

	var result []WBTempCoeff
	for i := 0; i < 64; i++ {
		if rp.res.handle.color.WBCT_Coeffs[i][0] == 0 {
			break
		}
		result = append(result, WBTempCoeff{
			CCT: int(rp.res.handle.color.WBCT_Coeffs[i][0]),
			Coeffs: [4]float32{
				float32(rp.res.handle.color.WBCT_Coeffs[i][1]),
				float32(rp.res.handle.color.WBCT_Coeffs[i][2]),
				float32(rp.res.handle.color.WBCT_Coeffs[i][3]),
				float32(rp.res.handle.color.WBCT_Coeffs[i][4]),
			},
		})
	}
	return result, nil
}

func (rp *RawProcessor) GetDNGColor(idx int) (DNGColorInfo, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return DNGColorInfo{}, err
	}
	if idx < 0 || idx > 1 {
		return DNGColorInfo{}, ErrInvalidIndex
	}

	dc := rp.res.handle.color.dng_color[idx]
	di := DNGColorInfo{
		ParsedFields: uint(dc.parsedfields),
		Illuminant:   uint16(dc.illuminant),
	}
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			di.Calibration[i][j] = float32(dc.calibration[i][j])
		}
		for j := 0; j < 3; j++ {
			di.ColorMatrix[i][j] = float32(dc.colormatrix[i][j])
		}
	}
	for i := 0; i < 3; i++ {
		for j := 0; j < 4; j++ {
			di.ForwardMatrix[i][j] = float32(dc.forwardmatrix[i][j])
		}
	}
	return di, nil
}

func (rp *RawProcessor) GetDNGLevels() (DNGLevels, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return DNGLevels{}, err
	}

	dl := rp.res.handle.color.dng_levels
	return DNGLevels{
		AsShotNeutral: [4]float32{
			float32(dl.asshotneutral[0]),
			float32(dl.asshotneutral[1]),
			float32(dl.asshotneutral[2]),
			float32(dl.asshotneutral[3]),
		},
		BaselineExposure:  float32(dl.baseline_exposure),
		AnalogBalance: [4]float32{
			float32(dl.analogbalance[0]),
			float32(dl.analogbalance[1]),
			float32(dl.analogbalance[2]),
			float32(dl.analogbalance[3]),
		},
		DngBlack:          uint(dl.dng_black),
		DngFBlack:         float32(dl.dng_fblack),
		DngWhiteLevel: [4]uint{
			uint(dl.dng_whitelevel[0]),
			uint(dl.dng_whitelevel[1]),
			uint(dl.dng_whitelevel[2]),
			uint(dl.dng_whitelevel[3]),
		},
		DefaultCrop: [4]uint16{
			uint16(dl.default_crop[0]),
			uint16(dl.default_crop[1]),
			uint16(dl.default_crop[2]),
			uint16(dl.default_crop[3]),
		},
		UserCrop: [4]float32{
			float32(dl.user_crop[0]),
			float32(dl.user_crop[1]),
			float32(dl.user_crop[2]),
			float32(dl.user_crop[3]),
		},
		PreviewColorSpace:    uint(dl.preview_colorspace),
		LinearResponseLimit:  float32(dl.LinearResponseLimit),
	}, nil
}

func (rp *RawProcessor) GetICCProfile() ([]byte, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return nil, err
	}

	if rp.res.handle.color.profile == nil {
		return nil, nil
	}

	length := C.uint(rp.res.handle.color.profile_length)
	if length == 0 {
		return nil, nil
	}
	return C.GoBytes(rp.res.handle.color.profile, C.int(length)), nil
}

// AdjustToRawInsetCrop applies the raw inset crop from DNG metadata.
// mask selects which crops to check (InsetCropDefaultMask, InsetCropUserMask, or InsetCropAllMask).
// maxcrop is the minimum fraction of the current width/height the crop must cover (0 = no limit).
// Returns true if a crop was applied, false if no valid crop was found.
func (rp *RawProcessor) AdjustToRawInsetCrop(mask InsetCropMask, maxcrop float32) (bool, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if err := rp.ensureOpen(); err != nil {
		return false, err
	}

	rc := C.libraw_adjust_to_raw_inset_crop(rp.res.handle, C.uint(mask), C.float(maxcrop))
	if rc >= C.LIBRAW_SUCCESS {
		return true, nil
	}
	return false, checkError(rc, ErrBadCrop)
}
