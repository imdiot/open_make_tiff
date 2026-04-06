package runner

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"open-make-tiff/pkg/golibtiff"
)

// tagWriter writes a single tag value to a TIFF IFD.
type tagWriter func(tf *golibtiff.TIFF, tag golibtiff.Tag, v any)

// tagMapping maps an ExifTool JSON key to a libtiff tag + writer function.
type tagMapping struct {
	jsonKey string
	tag     golibtiff.Tag
	write   tagWriter
}

// Predefined writer functions for each TIFF field type.
var (
	writeStr tagWriter = func(tf *golibtiff.TIFF, tag golibtiff.Tag, v any) {
		tf.SetFieldString(tag, valToStr(v))
	}
	writeByteStr tagWriter = func(tf *golibtiff.TIFF, tag golibtiff.Tag, v any) {
		tf.SetFieldByteSlice(tag, []byte(valToStr(v)))
	}
	writeFlt tagWriter = func(tf *golibtiff.TIFF, tag golibtiff.Tag, v any) {
		tf.SetFieldFloat(tag, valToFloat(v))
	}
	writeU16 tagWriter = func(tf *golibtiff.TIFF, tag golibtiff.Tag, v any) {
		tf.SetFieldUint16(tag, valToUint16(v))
	}
	writeU32 tagWriter = func(tf *golibtiff.TIFF, tag golibtiff.Tag, v any) {
		tf.SetFieldUint32(tag, valToUint32(v))
	}
	writeU8 tagWriter = func(tf *golibtiff.TIFF, tag golibtiff.Tag, v any) {
		tf.SetFieldUint8(tag, uint8(valToUint16(v)))
	}
	writeU16Slice tagWriter = func(tf *golibtiff.TIFF, tag golibtiff.Tag, v any) {
		tf.SetFieldUint16Slice(tag, []uint16{valToUint16(v)})
	}
	writeFltSlice tagWriter = func(tf *golibtiff.TIFF, tag golibtiff.Tag, v any) {
		if s := valToFloatSlice(v); len(s) > 0 {
			tf.SetFieldFloatSlice(tag, s)
		}
	}
	writeC0FltSlice tagWriter = func(tf *golibtiff.TIFF, tag golibtiff.Tag, v any) {
		if s := valToFloatSlice(v); len(s) > 0 {
			tf.SetFieldC0FloatSlice(tag, s)
		}
	}
)

// IFD0 tag mappings (written to main IFD).
var ifd0Mappings = []tagMapping{
	{"Make", golibtiff.TagMake, writeStr},
	{"Model", golibtiff.TagModel, writeStr},
	{"Software", golibtiff.TagSoftware, writeStr},
	{"ModifyDate", golibtiff.TagDateTime, writeStr},
	{"Artist", golibtiff.TagArtist, writeStr},
	{"Copyright", golibtiff.TagCopyright, writeStr},
}

// DNG override tag mappings (written to main IFD, from secondSrcPath).
var dngOverrideMappings = []tagMapping{
	{"UniqueCameraModel", golibtiff.TagUniqueCameraModel, writeStr},
	{"LocalizedCameraModel", golibtiff.TagLocalizedCameraModel, writeByteStr},
	{"AsShotNeutral", golibtiff.TagAsShotNeutral, writeFltSlice},
}

// EXIF Sub-IFD tag mappings.
var exifMappings = []tagMapping{
	{"ExposureTime", golibtiff.TagExifExposureTime, writeFlt},
	{"FNumber", golibtiff.TagExifFNumber, writeFlt},
	{"ExposureProgram", golibtiff.TagExifExposureProgram, writeU16},
	{"ISO", golibtiff.TagExifISO, writeU16Slice},
	{"SensitivityType", golibtiff.TagExifSensitivityType, writeU16},
	{"StandardOutputSensitivity", golibtiff.TagExifStandardOutputSensitivity, writeU32},
	{"ShutterSpeedValue", golibtiff.TagExifShutterSpeedValue, writeFlt},
	{"ApertureValue", golibtiff.TagExifApertureValue, writeFlt},
	{"BrightnessValue", golibtiff.TagExifBrightnessValue, writeFlt},
	{"ExposureCompensation", golibtiff.TagExifExposureCompensation, writeFlt},
	{"MaxApertureValue", golibtiff.TagExifMaxApertureValue, writeFlt},
	{"LightSource", golibtiff.TagExifLightSource, writeU16},
	{"Flash", golibtiff.TagExifFlash, writeU16},
	{"FocalLength", golibtiff.TagExifFocalLength, writeFlt},
	{"CreateDate", golibtiff.TagExifCreateDate, writeStr},
	{"DateTimeOriginal", golibtiff.TagExifDateTimeOriginal, writeStr},
	{"OffsetTime", golibtiff.TagExifOffsetTime, writeStr},
	{"OffsetTimeOriginal", golibtiff.TagExifOffsetTimeOriginal, writeStr},
	{"OffsetTimeDigitized", golibtiff.TagExifOffsetTimeDigitized, writeStr},
	{"LensInfo", golibtiff.TagExifLensInfo, writeC0FltSlice},
	{"LensMake", golibtiff.TagExifLensMake, writeStr},
	{"LensModel", golibtiff.TagExifLensModel, writeStr},
	{"LensSerialNumber", golibtiff.TagExifLensSerialNumber, writeStr},
	{"MeteringMode", golibtiff.TagExifMeteringMode, writeU16},
	{"ColorSpace", golibtiff.TagExifColorSpace, writeU16},
	{"ExifImageWidth", golibtiff.TagExifImageWidth, writeU32},
	{"ExifImageHeight", golibtiff.TagExifImageHeight, writeU32},
	{"SceneCaptureType", golibtiff.TagExifSceneCaptureType, writeU16},
	{"Sharpness", golibtiff.TagExifSharpness, writeU16},
	{"SerialNumber", golibtiff.TagExifSerialNumber, writeStr},
	{"ExposureMode", golibtiff.TagExifExposureMode, writeU16},
	{"WhiteBalance", golibtiff.TagExifWhiteBalance, writeU16},
	{"CustomRendered", golibtiff.TagExifCustomRendered, writeU16},
	{"SceneType", golibtiff.TagExifSceneType, writeU8},
	{"SensingMethod", golibtiff.TagExifSensingMethod, writeU16},
	{"SubjectDistanceRange", golibtiff.TagExifSubjectDistanceRange, writeU16},
	{"Gamma", golibtiff.TagExifGamma, writeFlt},
}

// MakerNote JSON key variants per manufacturer.
var makerNoteKeys = []string{
	"MakerNoteFujiFilm", "MakerNoteCanon", "MakerNoteNikon",
	"MakerNoteSony", "MakerNoteSigma", "MakerNotePanasonic",
	"MakerNoteOlympus", "MakerNotePentax", "MakerNoteUnknown",
}

// ExtractedMetadata holds raw JSON objects from ExifTool and a constructed XMP packet.
type ExtractedMetadata struct {
	RawTags map[string]any // from rawPath
	DNGTags map[string]any // from secondSrcPath
	XMP     []byte         // constructed XMP packet
}

// hasEXIF checks if rawTags contains any EXIF-like keys.
func (em *ExtractedMetadata) hasEXIF() bool {
	if em == nil || len(em.RawTags) == 0 {
		return false
	}
	for _, m := range exifMappings {
		if _, ok := em.RawTags[m.jsonKey]; ok {
			return true
		}
	}
	return false
}

// extractMetadata reads metadata from rawPath and secondSrcPath via ExifTool.
// excludeKeys are removed from RawTags after extraction.
func (r *Runner) extractMetadata(rawPath, secondSrcPath string, excludeExifTagKeys ...string) (*ExtractedMetadata, error) {
	if r.et == nil {
		return nil, nil
	}

	args := []string{
		"-json", "-n", "-b",
		"-IFD0:All", "-ExifIFD:All",
		"-XMP-aux:All", "-XMP-exifEX:All",
		"-XMP-dc:Subject", "-XMP-lr:HierarchicalSubject", "-XMP-mwg-kw:All",
		rawPath, secondSrcPath,
	}

	resp, err := r.et.Execute(args...)
	if err != nil {
		return nil, fmt.Errorf("exiftool extract metadata: %w", err)
	}

	em, err := parseExifToolJSON(resp, rawPath, secondSrcPath, filepath.Base(rawPath))
	if err != nil {
		return nil, err
	}
	for _, key := range excludeExifTagKeys {
		delete(em.RawTags, key)
	}
	return em, nil
}

func pathEqual(a, b string) bool {
	return a == b || strings.EqualFold(a, b) ||
		strings.EqualFold(strings.ReplaceAll(a, `\`, `/`), strings.ReplaceAll(b, `\`, `/`))
}

func parseExifToolJSON(jsonStr, rawPath, secondSrcPath, rawFileName string) (*ExtractedMetadata, error) {
	var objects []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &objects); err != nil {
		return nil, fmt.Errorf("json parse: %w", err)
	}

	em := &ExtractedMetadata{}
	samePath := pathEqual(rawPath, secondSrcPath)
	for _, obj := range objects {
		src, _ := obj["SourceFile"].(string)
		if pathEqual(src, rawPath) {
			em.RawTags = obj
		}
		if pathEqual(src, secondSrcPath) {
			if samePath {
				cp := make(map[string]any, len(obj))
				for k, v := range obj {
					cp[k] = v
				}
				em.DNGTags = cp
			} else {
				em.DNGTags = obj
			}
		}
	}

	// Build XMP from DNG tags (secondSrcPath block)
	if len(em.DNGTags) > 0 {
		em.XMP = buildXMPacket(em.DNGTags, rawFileName)
	}

	return em, nil
}

// getMakerNotes decodes base64 MakerNotes from the first matching key.
func getMakerNotes(obj map[string]any) []byte {
	for _, key := range makerNoteKeys {
		if v, ok := obj[key]; ok {
			s := valToStr(v)
			data := s
			if rest, ok := strings.CutPrefix(s, "base64:"); ok {
				data = rest
			}
			if d, err := base64.StdEncoding.DecodeString(data); err == nil {
				return d
			}
		}
	}
	return nil
}

// --- Value conversion helpers ---

func valToStr(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func valToFloat(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(val), 64)
		return f
	}
	return 0
}

func valToUint16(v any) uint16 {
	switch val := v.(type) {
	case float64:
		return uint16(val)
	case string:
		u, _ := strconv.ParseUint(strings.TrimSpace(val), 10, 16)
		return uint16(u)
	}
	return 0
}

func valToUint32(v any) uint32 {
	switch val := v.(type) {
	case float64:
		return uint32(val)
	case string:
		u, _ := strconv.ParseUint(strings.TrimSpace(val), 10, 32)
		return uint32(u)
	}
	return 0
}

func valToFloatSlice(v any) []float64 {
	switch val := v.(type) {
	case string:
		var r []float64
		for p := range strings.FieldsSeq(val) {
			if f, err := strconv.ParseFloat(p, 64); err == nil {
				r = append(r, f)
			}
		}
		return r
	case []any:
		r := make([]float64, 0, len(val))
		for _, item := range val {
			if f, ok := item.(float64); ok {
				r = append(r, f)
			}
		}
		return r
	}
	return nil
}

// writeTagMap writes all mapped tags present in tags to the TIFF.
func writeTagMap(tf *golibtiff.TIFF, tags map[string]any, mappings []tagMapping) {
	for _, m := range mappings {
		if v, ok := tags[m.jsonKey]; ok && !isEmptyValue(v) {
			m.write(tf, m.tag, v)
		}
	}
}

// --- XMP packet construction ---

// XMP namespace prefix → URI.
var xmpNS = map[string]string{
	"aux":    "http://ns.adobe.com/exif/1.0/aux/",
	"exifEX": "http://cipa.jp/exif/1.0/",
	"dc":     "http://purl.org/dc/elements/1.1/",
	"lr":     "http://ns.adobe.com/lightroom/1.0/",
	"mwg-kw": "http://www.metadataworkinggroup.com/schemas/keywords/",
	"crs":    "http://ns.adobe.com/camera-raw-settings/1.0/",
}

// xmpTagMap maps ExifTool JSON keys to XMP prefix + array flag.
var xmpTagMap = map[string]struct {
	prefix  string
	isArray bool
}{
	"Lens":                    {"aux", false},
	"LensID":                  {"aux", false},
	"LensInfo":                {"aux", false},
	"SerialNumber":            {"aux", false},
	"ImageNumber":             {"aux", false},
	"FlashCompensation":       {"aux", false},
	"ApproximateFocusDistance": {"aux", false},
	"Firmware":                {"aux", false},
	"LensModel":               {"exifEX", false},
	"LensMake":                {"exifEX", false},
	"LensSerialNumber":        {"exifEX", false},
	"LensSpecification":       {"exifEX", false},
	"CameraSerialNumber":      {"exifEX", false},
	"Subject":                 {"dc", true},
	"HierarchicalSubject":     {"lr", true},
	"Keywords":                {"mwg-kw", true},
}

func buildXMPacket(xmpValues map[string]any, rawFileName string) []byte {
	type elem struct {
		prefix  string
		key     string
		value   any
		isArray bool
	}

	var elements []elem
	usedNS := make(map[string]string)
	for jsonKey, val := range xmpValues {
		if jsonKey == "SourceFile" {
			continue
		}
		info, ok := xmpTagMap[jsonKey]
		if !ok || isEmptyValue(val) {
			continue
		}
		elements = append(elements, elem{
			prefix: info.prefix, key: jsonKey,
			value: val, isArray: info.isArray,
		})
		usedNS[info.prefix] = xmpNS[info.prefix]
	}
	if len(elements) == 0 && rawFileName == "" {
		return nil
	}
	usedNS["crs"] = xmpNS["crs"]

	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)

	enc.EncodeToken(xml.ProcInst{Target: "xpacket", Inst: []byte("begin=\"\xEF\xBB\xBF\" id=\"W5M0MpCehiHzreSzNTczkc9d\"")})
	enc.EncodeToken(xml.CharData("\n"))

	meta := xml.StartElement{
		Name: xml.Name{Local: "x:xmpmeta"},
		Attr: []xml.Attr{{Name: xml.Name{Local: "xmlns:x"}, Value: "adobe:ns:meta/"}},
	}
	enc.EncodeToken(meta)
	enc.EncodeToken(xml.CharData("\n "))

	rdf := xml.StartElement{
		Name: xml.Name{Local: "rdf:RDF"},
		Attr: []xml.Attr{{Name: xml.Name{Local: "xmlns:rdf"}, Value: "http://www.w3.org/1999/02/22-rdf-syntax-ns#"}},
	}
	enc.EncodeToken(rdf)
	enc.EncodeToken(xml.CharData("\n  "))

	desc := xml.StartElement{
		Name: xml.Name{Local: "rdf:Description"},
		Attr: []xml.Attr{{Name: xml.Name{Local: "rdf:about"}, Value: ""}},
	}
	for prefix, uri := range usedNS {
		desc.Attr = append(desc.Attr, xml.Attr{Name: xml.Name{Local: "xmlns:" + prefix}, Value: uri})
	}
	enc.EncodeToken(desc)

	for _, e := range elements {
		name := e.prefix + ":" + e.key
		if e.isArray {
			writeXMPArray(enc, name, e.value)
		} else {
			writeXMPText(enc, name, e.value)
		}
	}
	if rawFileName != "" {
		writeXMLElement(enc, "crs:RAWFileName", rawFileName)
	}

	enc.EncodeToken(desc.End())
	enc.EncodeToken(xml.CharData("\n  "))
	enc.EncodeToken(rdf.End())
	enc.EncodeToken(xml.CharData("\n "))
	enc.EncodeToken(meta.End())
	enc.EncodeToken(xml.CharData("\n"))
	enc.EncodeToken(xml.ProcInst{Target: "xpacket", Inst: []byte("end=\"w\"")})
	enc.EncodeToken(xml.CharData("\n"))
	enc.Flush()
	return buf.Bytes()
}

func writeXMLElement(enc *xml.Encoder, name, text string) {
	start := xml.StartElement{Name: xml.Name{Local: name}}
	enc.EncodeToken(start)
	enc.EncodeToken(xml.CharData(text))
	enc.EncodeToken(start.End())
}

func writeXMPText(enc *xml.Encoder, name string, value any) {
	writeXMLElement(enc, name, valToStr(value))
}

func writeXMPArray(enc *xml.Encoder, name string, value any) {
	var items []string
	switch v := value.(type) {
	case []any:
		for _, item := range v {
			items = append(items, fmt.Sprintf("%v", item))
		}
	case string:
		for part := range strings.SplitSeq(v, ", ") {
			items = append(items, part)
		}
	}
	if len(items) == 0 {
		return
	}
	tag := xml.StartElement{Name: xml.Name{Local: name}}
	bag := xml.StartElement{Name: xml.Name{Local: "rdf:Bag"}}
	enc.EncodeToken(tag)
	enc.EncodeToken(bag)
	for _, item := range items {
		writeXMLElement(enc, "rdf:li", item)
	}
	enc.EncodeToken(bag.End())
	enc.EncodeToken(tag.End())
}

func isEmptyValue(v any) bool {
	switch val := v.(type) {
	case nil:
		return true
	case string:
		return val == ""
	case []any:
		return len(val) == 0
	default:
		return false
	}
}
