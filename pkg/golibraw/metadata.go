package golibraw

/*
#include <libraw/libraw.h>
*/
import "C"

import (
	"time"
)

func (rp *RawProcessor) GetImageSizes() ImageSizes {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.closed || rp.handle == nil {
		return ImageSizes{}
	}

	s := rp.handle.sizes
	return ImageSizes{
		RawHeight:       uint16(s.raw_height),
		RawWidth:        uint16(s.raw_width),
		Height:          uint16(s.height),
		Width:           uint16(s.width),
		TopMargin:       uint16(s.top_margin),
		LeftMargin:      uint16(s.left_margin),
		IHeight:         uint16(s.iheight),
		IWidth:          uint16(s.iwidth),
		Flip:            int(s.flip),
		PixelAspectRatio: float64(s.pixel_aspect),
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
}

func (rp *RawProcessor) GetCameraInfo() CameraInfo {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.closed || rp.handle == nil {
		return CameraInfo{}
	}

	ip := C.libraw_get_iparams(rp.handle)
	return CameraInfo{
		Make:            C.GoString(&ip.make[0]),
		Model:           C.GoString(&ip.model[0]),
		NormalizedMake:  C.GoString(&ip.normalized_make[0]),
		NormalizedModel: C.GoString(&ip.normalized_model[0]),
		Software:        C.GoString(&ip.software[0]),
		RawCount:        uint(ip.raw_count),
		DNGVersion:      uint(ip.dng_version),
		IsFoveon:        ip.is_foveon != 0,
		Colors:          int(ip.colors),
		CDesc:           C.GoString(&ip.cdesc[0]),
		MakerIndex:      uint(ip.maker_index),
	}
}

func (rp *RawProcessor) GetLensInfo() LensInfo {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.closed || rp.handle == nil {
		return LensInfo{}
	}

	li := C.libraw_get_lensinfo(rp.handle)
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
	}
}

func (rp *RawProcessor) GetShootingParams() ShootingParams {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.closed || rp.handle == nil {
		return ShootingParams{}
	}

	ot := C.libraw_get_imgother(rp.handle)
	ts := time.Unix(int64(ot.timestamp), 0)
	return ShootingParams{
		ISOSpeed:  float32(ot.iso_speed),
		Shutter:   float32(ot.shutter),
		Aperture:  float32(ot.aperture),
		FocalLen:  float32(ot.focal_len),
		Timestamp: ts,
		Artist:    C.GoString(&ot.artist[0]),
		Desc:      C.GoString(&ot.desc[0]),
	}
}

func (rp *RawProcessor) GetShootingInfo() ShootingInfo {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.closed || rp.handle == nil {
		return ShootingInfo{}
	}

	si := rp.handle.shootinginfo
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
	}
}

func (rp *RawProcessor) GetMakernotesLens() MakernotesLensInfo {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.closed || rp.handle == nil {
		return MakernotesLensInfo{}
	}

	ml := rp.handle.lens.makernotes
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
	}
}

func (rp *RawProcessor) GetTemperatures() SensorTemperatures {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.closed || rp.handle == nil {
		return SensorTemperatures{}
	}

	c := rp.handle.makernotes.common
	return SensorTemperatures{
		CameraTemperature:      float32(c.CameraTemperature),
		SensorTemperature:      float32(c.SensorTemperature),
		SensorTemperature2:     float32(c.SensorTemperature2),
		LensTemperature:        float32(c.LensTemperature),
		AmbientTemperature:     float32(c.AmbientTemperature),
		BatteryTemperature:     float32(c.BatteryTemperature),
		ExifAmbientTemperature: float32(c.exifAmbientTemperature),
		FlashEC:                float32(c.FlashEC),
		FlashGN:                float32(c.FlashGN),
		RealISO:                float32(c.real_ISO),
		Firmware:               C.GoString(&c.firmware[0]),
	}
}

func (rp *RawProcessor) GetThumbnailInfo() ThumbnailInfo {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.closed || rp.handle == nil {
		return ThumbnailInfo{}
	}

	t := rp.handle.thumbnail
	return ThumbnailInfo{
		Format: ThumbnailFormat(t.tformat),
		Width:  uint16(t.twidth),
		Height: uint16(t.theight),
		Length: uint(t.tlength),
		Colors: int(t.tcolors),
	}
}

func (rp *RawProcessor) GetColorData() ColorData {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.closed || rp.handle == nil {
		return ColorData{}
	}

	c := rp.handle.color
	cd := ColorData{
		Black:                uint(c.black),
		LinearMax:            [4]uint{uint(c.linear_max[0]), uint(c.linear_max[1]), uint(c.linear_max[2]), uint(c.linear_max[3])},
		CamMul:               [4]float32{float32(c.cam_mul[0]), float32(c.cam_mul[1]), float32(c.cam_mul[2]), float32(c.cam_mul[3])},
		PreMul:               [4]float32{float32(c.pre_mul[0]), float32(c.pre_mul[1]), float32(c.pre_mul[2]), float32(c.pre_mul[3])},
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
	return cd
}

func (rp *RawProcessor) GetWBCoeffs() map[WBIndex][4]int {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	result := make(map[WBIndex][4]int)
	if rp.closed || rp.handle == nil {
		return result
	}

	for i := 0; i < 256; i++ {
		if rp.handle.color.WB_Coeffs[i][0] != 0 || rp.handle.color.WB_Coeffs[i][1] != 0 {
			result[WBIndex(i)] = [4]int{
				int(rp.handle.color.WB_Coeffs[i][0]),
				int(rp.handle.color.WB_Coeffs[i][1]),
				int(rp.handle.color.WB_Coeffs[i][2]),
				int(rp.handle.color.WB_Coeffs[i][3]),
			}
		}
	}
	return result
}

func (rp *RawProcessor) GetWBTempCoeffs() []WBTempCoeff {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.closed || rp.handle == nil {
		return nil
	}

	var result []WBTempCoeff
	for i := 0; i < 64; i++ {
		if rp.handle.color.WBCT_Coeffs[i][0] == 0 {
			break
		}
		result = append(result, WBTempCoeff{
			CCT: int(rp.handle.color.WBCT_Coeffs[i][0]),
			Coeffs: [4]float32{
				float32(rp.handle.color.WBCT_Coeffs[i][1]),
				float32(rp.handle.color.WBCT_Coeffs[i][2]),
				float32(rp.handle.color.WBCT_Coeffs[i][3]),
				float32(rp.handle.color.WBCT_Coeffs[i][4]),
			},
		})
	}
	return result
}

func (rp *RawProcessor) GetDNGColor(idx int) DNGColorInfo {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.closed || rp.handle == nil || idx < 0 || idx > 1 {
		return DNGColorInfo{}
	}

	dc := rp.handle.color.dng_color[idx]
	di := DNGColorInfo{
		Illuminant: uint16(dc.illuminant),
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
	return di
}

func (rp *RawProcessor) GetDNGLevels() DNGLevels {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.closed || rp.handle == nil {
		return DNGLevels{}
	}

	dl := rp.handle.color.dng_levels
	return DNGLevels{
		AsShotNeutral: [4]float32{
			float32(dl.asshotneutral[0]),
			float32(dl.asshotneutral[1]),
			float32(dl.asshotneutral[2]),
			float32(dl.asshotneutral[3]),
		},
		BaselineExposure: float32(dl.baseline_exposure),
		AnalogBalance: [4]float32{
			float32(dl.analogbalance[0]),
			float32(dl.analogbalance[1]),
			float32(dl.analogbalance[2]),
			float32(dl.analogbalance[3]),
		},
	}
}

func (rp *RawProcessor) GetICCProfile() []byte {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.closed || rp.handle == nil || rp.handle.color.profile == nil {
		return nil
	}

	length := C.uint(rp.handle.color.profile_length)
	if length == 0 {
		return nil
	}
	return C.GoBytes(rp.handle.color.profile, C.int(length))
}

// AdjustToRawInsetCrop applies the raw inset crop from DNG metadata.
// mask selects which crops to check (InsetCropDefaultMask, InsetCropUserMask, or InsetCropAllMask).
// UserCrop (crop[1]) is preferred over DefaultCrop (crop[0]) when both are valid.
// maxcrop is the minimum fraction of the current width/height the crop must cover (0 = no limit).
func (rp *RawProcessor) AdjustToRawInsetCrop(mask InsetCropMask, maxcrop float32) (InsetCropIndex, error) {
	sizes := rp.GetImageSizes()

	limW := int(float32(sizes.Width) * maxcrop)
	limH := int(float32(sizes.Height) * maxcrop)

	adjIndex := -1
	for i := 1; i >= 0; i-- {
		if mask&(1<<i) == 0 {
			continue
		}
		c := sizes.RawInsetCrops[i]
		if c.Top == 0xffff || c.Left == 0xffff {
			continue
		}
		if int(c.Left)+int(c.Width) > int(sizes.RawWidth) {
			continue
		}
		if int(c.Top)+int(c.Height) > int(sizes.RawHeight) {
			continue
		}
		if int(c.Width) < limW || int(c.Height) < limH {
			continue
		}
		adjIndex = i
		break
	}

	if adjIndex < 0 {
		return InsetCropNone, nil
	}

	c := sizes.RawInsetCrops[adjIndex]
	w := min(int(c.Width), int(sizes.RawWidth)-int(c.Left))
	h := min(int(c.Height), int(sizes.RawHeight)-int(c.Top))

	if err := rp.ApplyOptions(WithCropBox(
		uint(c.Left), uint(c.Top), uint(w), uint(h),
	)); err != nil {
		return InsetCropNone, err
	}
	return InsetCropIndex(adjIndex + 1), nil
}
