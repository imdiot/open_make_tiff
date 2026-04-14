package metadata

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"open-make-tiff/pkg/golibtiff"
)

var DNGOverrideIDs = map[golibtiff.Tag]bool{
	golibtiff.TagUniqueCameraModel:    true,
	golibtiff.TagLocalizedCameraModel: true,
	golibtiff.TagAsShotNeutral:        true,
}

var skipIFD0IDs = map[golibtiff.Tag]bool{
	golibtiff.TagNewSubfileType:  true,
	golibtiff.TagImageWidth:      true,
	golibtiff.TagImageLength:     true,
	golibtiff.TagBitsPerSample:   true,
	golibtiff.TagSampleFormat:    true,
	golibtiff.TagCompression:     true,
	golibtiff.TagPhotometric:     true,
	golibtiff.TagSamplesPerPixel: true,
	golibtiff.TagPredictor:       true,
	golibtiff.TagRowsPerStrip:    true,
	golibtiff.TagStripByteCounts: true,
	golibtiff.TagXResolution:     true,
	golibtiff.TagYResolution:     true,
	golibtiff.TagResolutionUnit:  true,
	golibtiff.TagEXIFIFD:         true,
	golibtiff.TagGPSIFD:          true,
	golibtiff.TagXMP:             true,
	golibtiff.TagIccProfile:      true,
}

var standardTables = map[string]bool{
	"Exif::Main": true,
	"GPS::Main":  true,
}

type TagInfo struct {
	Val   string `json:"val"`
	Num   any    `json:"num,omitempty"`
	Rat   string `json:"rat,omitempty"`
	Hex   string `json:"hex,omitempty"`
	Fmt   string `json:"fmt,omitempty"`
	ID    any    `json:"id"`
	Table string `json:"table"`
}

func (ti TagInfo) TagID() golibtiff.Tag {
	switch val := ti.ID.(type) {
	case float64:
		return golibtiff.Tag(val)
	case string:
		if strings.HasPrefix(val, "0x") || strings.HasPrefix(val, "0X") {
			if u, err := strconv.ParseUint(val[2:], 16, 32); err == nil {
				return golibtiff.Tag(u)
			}
		}
		if u, err := strconv.ParseUint(val, 10, 32); err == nil {
			return golibtiff.Tag(u)
		}
	}
	return 0
}

func (ti TagInfo) Uint8() uint8 {
	if ti.Num != nil {
		if f, ok := ti.Num.(float64); ok {
			return uint8(f)
		}
		if s, ok := ti.Num.(string); ok {
			u, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 8)
			return uint8(u)
		}
	}
	if data := ti.decodeHex(); len(data) >= 1 {
		return data[0]
	}
	if f, err := strconv.ParseFloat(strings.TrimSpace(ti.Val), 64); err == nil {
		return uint8(f)
	}
	return 0
}

func (ti TagInfo) Uint16() uint16 {
	if ti.Num != nil {
		if f, ok := ti.Num.(float64); ok {
			return uint16(f)
		}
		if s, ok := ti.Num.(string); ok {
			u, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 16)
			return uint16(u)
		}
	}
	if data := ti.decodeHex(); len(data) >= 2 {
		return uint16(data[0]) | uint16(data[1])<<8
	}
	if f, err := strconv.ParseFloat(strings.TrimSpace(ti.Val), 64); err == nil {
		return uint16(f)
	}
	return 0
}

func (ti TagInfo) Uint32() uint32 {
	if ti.Num != nil {
		if f, ok := ti.Num.(float64); ok {
			return uint32(f)
		}
		if s, ok := ti.Num.(string); ok {
			u, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 32)
			return uint32(u)
		}
	}
	if data := ti.decodeHex(); len(data) >= 4 {
		return uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
	}
	if f, err := strconv.ParseFloat(strings.TrimSpace(ti.Val), 64); err == nil {
		return uint32(f)
	}
	return 0
}

func (ti TagInfo) Float() float64 {
	if ti.Num != nil {
		if f, ok := ti.Num.(float64); ok {
			return f
		}
		if s, ok := ti.Num.(string); ok {
			f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
			return f
		}
	}
	if ti.Hex != "" {
		if data := ti.decodeHex(); len(data) == 8 {
			num := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
			den := uint32(data[4]) | uint32(data[5])<<8 | uint32(data[6])<<16 | uint32(data[7])<<24
			if den != 0 {
				return float64(num) / float64(den)
			}
		}
	}
	f, _ := strconv.ParseFloat(strings.TrimSpace(ti.Val), 64)
	return f
}

func (ti TagInfo) ExtractValue(ft golibtiff.DataType, wc int) any {
	if ft == golibtiff.DataTypeRational || ft == golibtiff.DataTypeSRational {
		if wc == 1 {
			if ti.Rat != "" {
				for part := range strings.FieldsSeq(ti.Rat) {
					numStr, denStr, ok := strings.Cut(part, "/")
					if !ok {
						continue
					}
					num, errN := strconv.ParseInt(numStr, 10, 64)
					den, errD := strconv.ParseInt(denStr, 10, 64)
					if errN == nil && errD == nil && den != 0 {
						return float64(num) / float64(den)
					}
				}
			}
			return ti.Float()
		}
		if ti.Rat != "" {
			var vals []float64
			for part := range strings.FieldsSeq(ti.Rat) {
				numStr, denStr, ok := strings.Cut(part, "/")
				if !ok {
					continue
				}
				num, errN := strconv.ParseInt(numStr, 10, 64)
				den, errD := strconv.ParseInt(denStr, 10, 64)
				if errN != nil || errD != nil || den == 0 {
					continue
				}
				vals = append(vals, float64(num)/float64(den))
			}
			if len(vals) > 0 {
				return vals
			}
		}
		if vals := ti.BinaryRationalSlice(); len(vals) > 0 {
			return vals
		}
		return nil
	}
	switch ft {
	case golibtiff.DataTypeShort, golibtiff.DataTypeSShort:
		if wc == 1 {
			return ti.Uint16()
		}
		if data := ti.BinaryUint16Slice(); len(data) > 0 {
			return data
		}
		return nil
	case golibtiff.DataTypeLong, golibtiff.DataTypeSLong:
		if wc == 1 {
			return ti.Uint32()
		}
		if data := ti.BinaryUint32Slice(); len(data) > 0 {
			return data
		}
		return nil
	case golibtiff.DataTypeByte, golibtiff.DataTypeSByte:
		if wc == 1 {
			return ti.Uint8()
		}
		if data := ti.Binary(); len(data) > 0 {
			return data
		}
		return nil
	case golibtiff.DataTypeUndefined:
		if wc == 1 {
			if data := ti.Binary(); len(data) > 0 {
				return data[0]
			}
			return nil
		}
		if data := ti.Binary(); len(data) > 0 {
			return data
		}
		return nil
	case golibtiff.DataTypeASCII:
		if s, ok := ti.Num.(string); ok && s != "" {
			return s
		}
		if ti.Val != "" {
			return ti.Val
		}
		return nil
	case golibtiff.DataTypeFloat, golibtiff.DataTypeDouble:
		return ti.Float()
	}
	return nil
}

func (ti TagInfo) Binary() []byte {
	if rest, ok := strings.CutPrefix(ti.Val, "base64:"); ok {
		if d, err := base64.StdEncoding.DecodeString(rest); err == nil {
			return d
		}
	}
	return ti.decodeHex()
}

func (ti TagInfo) decodeHex() []byte {
	if ti.Hex == "" || strings.Contains(ti.Hex, "[...]") {
		return nil
	}
	parts := strings.Fields(ti.Hex)
	data := make([]byte, 0, len(parts))
	for _, p := range parts {
		if len(p) > 2 {
			continue
		}
		b, err := strconv.ParseUint(p, 16, 8)
		if err != nil {
			return nil
		}
		data = append(data, byte(b))
	}
	return data
}

func (ti TagInfo) BinaryUint16Slice() []uint16 {
	data := ti.Binary()
	if len(data) < 2 || len(data)%2 != 0 {
		return nil
	}
	n := len(data) / 2
	result := make([]uint16, n)
	for i := range result {
		result[i] = uint16(data[2*i]) | uint16(data[2*i+1])<<8
	}
	return result
}

func (ti TagInfo) BinaryUint32Slice() []uint32 {
	data := ti.Binary()
	if len(data) < 4 || len(data)%4 != 0 {
		return nil
	}
	n := len(data) / 4
	result := make([]uint32, n)
	for i := range result {
		result[i] = uint32(data[4*i]) | uint32(data[4*i+1])<<8 |
			uint32(data[4*i+2])<<16 | uint32(data[4*i+3])<<24
	}
	return result
}

func (ti TagInfo) BinaryRationalSlice() []float64 {
	data := ti.Binary()
	if len(data) < 8 || len(data)%8 != 0 {
		return nil
	}
	n := len(data) / 8
	result := make([]float64, 0, n)
	for i := 0; i < n; i++ {
		off := i * 8
		num := uint32(data[off]) | uint32(data[off+1])<<8 | uint32(data[off+2])<<16 | uint32(data[off+3])<<24
		den := uint32(data[off+4]) | uint32(data[off+5])<<8 | uint32(data[off+6])<<16 | uint32(data[off+7])<<24
		if den != 0 {
			result = append(result, float64(num)/float64(den))
		}
	}
	return result
}

// MakerNoteFixup holds offset correction data for absolute-offset MakerNotes.
// Set by ExtractedMetadata.AnalyzeMakerNote; nil for relative-offset, self-contained, or absent MakerNotes.
type MakerNoteFixup struct {
	BaseOld   uint32           // Inferred original base offset in source file
	BO        binary.ByteOrder // Byte order of MakerNote IFD content
	Pointers  []int            // Byte positions of offset pointers (relative to MakerNote start)
	DataLen   int              // Length of MakerNote binary data
	HasFooter bool             // Whether Canon-style TIFF footer exists
}

type ExtractedMetadata struct {
	logger         *slog.Logger
	IFD0           map[string]TagInfo
	EXIF           map[string]TagInfo
	GPS            map[string]TagInfo
	XMP            map[string]TagInfo
	MakerNoteFixup *MakerNoteFixup // nil when no fixup needed
}

func NewExtractedMetadata(logger *slog.Logger) *ExtractedMetadata {
	if logger == nil {
		logger = slog.Default()
	}
	return &ExtractedMetadata{
		logger: logger,
		IFD0:   make(map[string]TagInfo),
		EXIF:   make(map[string]TagInfo),
		GPS:    make(map[string]TagInfo),
		XMP:    make(map[string]TagInfo),
	}
}

func SplitGroupKey(key string) (group, name string) {
	idx := strings.IndexByte(key, ':')
	if idx < 0 {
		return "", key
	}
	return key[:idx], key[idx+1:]
}

func (em *ExtractedMetadata) Parse(obj map[string]any) {
	for key, raw := range obj {
		if key == "SourceFile" {
			continue
		}
		group, name := SplitGroupKey(key)
		if name == "" {
			continue
		}
		var ti TagInfo
		switch val := raw.(type) {
		case map[string]any:
			for k, v := range val {
				switch k {
				case "val":
					if n, ok := v.(float64); ok {
						ti.Val = strconv.FormatFloat(n, 'f', -1, 64)
					} else {
						ti.Val = fmt.Sprintf("%v", v)
					}
				case "num":
					ti.Num = v
				case "rat":
					if s, ok := v.(string); ok {
						ti.Rat = s
					}
				case "hex":
					if s, ok := v.(string); ok {
						ti.Hex = s
					}
				case "fmt":
					if s, ok := v.(string); ok {
						ti.Fmt = s
					}
				case "id":
					ti.ID = v
				case "table":
					if s, ok := v.(string); ok {
						ti.Table = s
					}
				}
			}
		case string:
			ti.Val = val
		default:
			ti.Val = fmt.Sprintf("%v", val)
		}
		if !standardTables[ti.Table] {
			continue
		}
		switch group {
		case "IFD0", "SubIFD":
			em.IFD0[name] = ti
		case "ExifIFD":
			em.EXIF[name] = ti
		case "GPS":
			em.GPS[name] = ti
		default:
			if strings.HasPrefix(group, "XMP") {
				em.XMP[key] = ti
			}
		}
	}

	em.analyzeMakerNote()
}
