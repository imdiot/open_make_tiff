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

// TagInfo represents a single tag from exiftool's structured JSON output (-l flag).
type TagInfo struct {
	Val   string `json:"val"`              // Display value; binary tags become "base64:..."
	Num   any    `json:"num,omitempty"`     // Numeric value (float64 or string)
	Rat   string `json:"rat,omitempty"`     // Raw RATIONAL: "1/15", "54/1 5938/100 0/1"
	Hex   string `json:"hex,omitempty"`     // Raw bytes hex (large data truncated to [...])
	Fmt   string `json:"fmt,omitempty"`     // exiftool format: int16u/rational64u/string/undef etc.
	ID    any    `json:"id"`                // Tag ID (int or "0x0112")
	Table string `json:"table"`             // Source table (Exif::Main, GPS::Main etc.)
}

// ExtractedMetadata holds merged metadata ready for TIFF writing.
// Merging (Raw + DNG override) is done during extraction, not during write.
type ExtractedMetadata struct {
	IFD0      map[string]TagInfo // IFD0 tags (Raw base + DNG override for specific tags)
	EXIF      map[string]TagInfo // EXIF Sub-IFD tags (from Raw)
	GPS       map[string]TagInfo // GPS Sub-IFD tags (from Raw)
	XMPPacket []byte             // Constructed XMP packet from DNG's XMP tags
}

// fileMetadata holds parsed metadata for a single file, grouped by IFD.
type fileMetadata struct {
	IFD0 map[string]TagInfo
	EXIF map[string]TagInfo
	GPS  map[string]TagInfo
	XMP  map[string]TagInfo
}

// --- DNG override tag IDs ---
// These IFD0 tags from DNG (secondSrcPath) override the ones from Raw.
var dngOverrideIDs = []uint32{
	50708, // UniqueCameraModel
	50709, // LocalizedCameraModel
	50728, // AsShotNeutral
}

// --- IFD0 skip list ---
// Tags managed by image data or configuration, not written from metadata.
var skipIFD0IDs = map[uint32]bool{
	254: true,                        // NewSubfileType — output is full-res, not thumbnail
	256: true, 257: true, 258: true, // ImageWidth/Height/BitsPerSample
	259: true, 262: true, 277: true, // Compression/Photometric/SamplesPerPixel
	278: true, 279: true, 317: true, // RowsPerStrip/StripOffsets/Predictor
	339: true, // SampleFormat
	282: true, 283: true, 296: true, // XResolution/YResolution/ResolutionUnit → config override
	34665: true, // ExifIFD pointer
	34853: true, // GPSInfoIFD pointer
	34675: true, // ICC Profile → config override
	700: true,   // XMP → packet logic
}

// --- Extraction ---

func (r *Runner) extractMetadata(rawPath, secondSrcPath string, excludeKeys ...string) (*ExtractedMetadata, error) {
	if r.et == nil {
		return nil, nil
	}

	samePath := pathEqual(rawPath, secondSrcPath)

	args := []string{
		"-json", "-G1", "-l", "-t", "-b", "-a", "-U", "-ee",
		"-api", "SaveBin=1", "-api", "SaveFormat=1", "-api", "MakerNotes=2",
		"-IFD0:All", "-ExifIFD:All", "-GPS:All",
		"-XMP-aux:All", "-XMP-exifEX:All",
		"-XMP-dc:Subject", "-XMP-lr:HierarchicalSubject", "-XMP-mwg-kw:All",
	}
	args = append(args, rawPath)
	if !samePath {
		args = append(args, secondSrcPath)
	}

	resp, err := r.et.Execute(args...)
	if err != nil {
		return nil, fmt.Errorf("exiftool extract metadata: %w", err)
	}

	var objects []map[string]any
	if err := json.Unmarshal([]byte(resp), &objects); err != nil {
		return nil, fmt.Errorf("json parse: %w", err)
	}
	if len(objects) == 0 {
		return nil, nil
	}

	var rawFM, dngFM *fileMetadata
	if samePath {
		fm := parseFileMetadata(objects[0])
		rawFM = fm
		dngFM = fm
	} else {
		for _, obj := range objects {
			src, _ := obj["SourceFile"].(string)
			fm := parseFileMetadata(obj)
			if pathEqual(src, rawPath) {
				rawFM = fm
			} else {
				dngFM = fm
			}
		}
	}

	if rawFM == nil {
		return nil, nil
	}

	em := &ExtractedMetadata{
		IFD0: rawFM.IFD0,
		EXIF: rawFM.EXIF,
		GPS:  rawFM.GPS,
	}

	// DNG override: replace specific IFD0 tags from DNG metadata
	if dngFM != nil {
		idMap := make(map[uint32]string, len(dngFM.IFD0))
		for name, ti := range dngFM.IFD0 {
			if id := parseTagID(ti.ID); id != 0 {
				idMap[id] = name
			}
		}
		for _, id := range dngOverrideIDs {
			if name, ok := idMap[id]; ok {
				em.IFD0[name] = dngFM.IFD0[name]
			}
		}
	}

	// Exclude specified tag keys
	for _, key := range excludeKeys {
		delete(em.IFD0, key)
		delete(em.EXIF, key)
	}

	// Build XMP packet from DNG's XMP tags
	if dngFM != nil && len(dngFM.XMP) > 0 {
		em.XMPPacket = buildXMPacket(dngFM.XMP, filepath.Base(rawPath))
	}

	return em, nil
}

// --- Parsing ---

func pathEqual(a, b string) bool {
	return a == b || strings.EqualFold(a, b) ||
		strings.EqualFold(strings.ReplaceAll(a, `\`, `/`), strings.ReplaceAll(b, `\`, `/`))
}

// splitGroupKey extracts group prefix and tag name from a -G1 prefixed key.
// "IFD0:Make" → ("IFD0", "Make")
// "XMP-aux:Lens" → ("XMP-aux", "Lens")
func splitGroupKey(key string) (group, name string) {
	idx := strings.IndexByte(key, ':')
	if idx < 0 {
		return "", key
	}
	return key[:idx], key[idx+1:]
}

func parseFileMetadata(obj map[string]any) *fileMetadata {
	fm := &fileMetadata{
		IFD0: make(map[string]TagInfo),
		EXIF: make(map[string]TagInfo),
		GPS:  make(map[string]TagInfo),
		XMP:  make(map[string]TagInfo),
	}
	for key, raw := range obj {
		if key == "SourceFile" {
			continue
		}
		group, name := splitGroupKey(key)
		if name == "" {
			continue
		}
		ti := parseTagInfo(raw)
		// Tags from non-standard tables (e.g. PanasonicRaw::Main) may reuse
		// standard tag IDs for different purposes — filter them out early.
		if !standardTables[ti.Table] {
			continue
		}
		switch group {
		case "IFD0", "SubIFD":
			fm.IFD0[name] = ti
		case "ExifIFD":
			fm.EXIF[name] = ti
		case "GPS":
			fm.GPS[name] = ti
		default:
			if strings.HasPrefix(group, "XMP") {
				fm.XMP[name] = ti
			}
		}
	}
	return fm
}

// parseTagInfo converts a raw JSON value to TagInfo.
// Handles both structured objects (-l flag) and simple values.
func parseTagInfo(v any) TagInfo {
	switch val := v.(type) {
	case map[string]any:
		ti := TagInfo{}
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
		return ti
	case string:
		return TagInfo{Val: val}
	default:
		return TagInfo{Val: fmt.Sprintf("%v", v)}
	}
}

// --- Tag ID parsing ---

func parseTagID(v any) uint32 {
	switch val := v.(type) {
	case float64:
		return uint32(val)
	case string:
		if strings.HasPrefix(val, "0x") || strings.HasPrefix(val, "0X") {
			if u, err := strconv.ParseUint(val[2:], 16, 32); err == nil {
				return uint32(u)
			}
		}
		if u, err := strconv.ParseUint(val, 10, 32); err == nil {
			return uint32(u)
		}
	}
	return 0
}

// --- Value conversion helpers ---

func numToFloat(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(val), 64)
		return f
	}
	return 0
}

func numToUint16(v any) uint16 {
	switch val := v.(type) {
	case float64:
		return uint16(val)
	case string:
		u, _ := strconv.ParseUint(strings.TrimSpace(val), 10, 16)
		return uint16(u)
	}
	return 0
}

func numToUint32(v any) uint32 {
	switch val := v.(type) {
	case float64:
		return uint32(val)
	case string:
		u, _ := strconv.ParseUint(strings.TrimSpace(val), 10, 32)
		return uint32(u)
	}
	return 0
}

func numToUint8(v any) uint8 {
	switch val := v.(type) {
	case float64:
		return uint8(val)
	case string:
		u, _ := strconv.ParseUint(strings.TrimSpace(val), 10, 8)
		return uint8(u)
	}
	return 0
}

// tagUint16 extracts a uint16 from TagInfo, trying Num first, then Hex, then Val.
func tagUint16(ti TagInfo) uint16 {
	if ti.Num != nil {
		return numToUint16(ti.Num)
	}
	if data := decodeHex(ti.Hex); len(data) >= 2 {
		return uint16(data[0]) | uint16(data[1])<<8
	}
	return numToUint16(ti.Val)
}

// tagUint8 extracts a uint8 from TagInfo: Num > Hex > Val.
func tagUint8(ti TagInfo) uint8 {
	if ti.Num != nil {
		return numToUint8(ti.Num)
	}
	if data := decodeHex(ti.Hex); len(data) >= 1 {
		return data[0]
	}
	return numToUint8(ti.Val)
}

// tagUint32 extracts a uint32 from TagInfo: Num > Hex > Val.
func tagUint32(ti TagInfo) uint32 {
	if ti.Num != nil {
		return numToUint32(ti.Num)
	}
	if data := decodeHex(ti.Hex); len(data) >= 4 {
		return uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
	}
	return numToUint32(ti.Val)
}

// tagFloat extracts a float64 from TagInfo: Num > Hex (RATIONAL decode) > Val.
func tagFloat(ti TagInfo) float64 {
	if ti.Num != nil {
		return numToFloat(ti.Num)
	}
	if ti.Hex != "" {
		if data := decodeHex(ti.Hex); len(data) == 8 {
			num := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
			den := uint32(data[4]) | uint32(data[5])<<8 | uint32(data[6])<<16 | uint32(data[7])<<24
			if den != 0 {
				return float64(num) / float64(den)
			}
		}
	}
	return numToFloat(ti.Val)
}

// --- RATIONAL parsing ---

// parseRational parses a rat string like "1/15" or "54/1 5938/100 0/1"
// into a slice of float64 values preserving exact division.
func parseRational(ratStr string) []float64 {
	var vals []float64
	for part := range strings.FieldsSeq(ratStr) {
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
	return vals
}

// --- Binary decoding ---

// decodeBinary decodes binary data from base64 (Val) or hex (Hex).
// Prefers base64 as it contains complete data; hex may be truncated for large blobs.
func decodeBinary(ti TagInfo) []byte {
	if rest, ok := strings.CutPrefix(ti.Val, "base64:"); ok {
		if d, err := base64.StdEncoding.DecodeString(rest); err == nil {
			return d
		}
	}
	return decodeHex(ti.Hex)
}

// decodeHex parses a hex string like "43 61 6e 6f 6e 00".
// Returns nil if the string contains [...] (truncated data).
func decodeHex(hexStr string) []byte {
	if hexStr == "" || strings.Contains(hexStr, "[...]") {
		return nil
	}
	parts := strings.Fields(hexStr)
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

// --- Auto-type writing ---

// standardTables lists exiftool source tables whose tag semantics align with
// libtiff's standard TIFF/EXIF field definitions. Tags from other tables
// (e.g. PanasonicRaw::Main) may reuse standard tag IDs for different meanings
// and are filtered out during metadata parsing.
// Source: exiftool ExifTool.pm:8922 — SHORT_NAME = package minus "Image::ExifTool::"
var standardTables = map[string]bool{
	"Exif::Main": true, // IFD0 + ExifIFD + InteropIFD + SubIFD
	"GPS::Main":  true, // GPS
}

// writeTag writes a single tag to the TIFF based on libtiff field_type and writecount.
func writeTag(tf *golibtiff.TIFF, ti TagInfo, skipIDs map[uint32]bool) {
	id := parseTagID(ti.ID)
	if skipIDs != nil && skipIDs[id] {
		return
	}
	if ti.Rat == "0/0" {
		return
	}
	tag := golibtiff.Tag(id)

	ft := tf.GetFieldType(tag)
	if ft < 0 {
		return // unknown tag
	}

	// RATIONAL via rat field takes priority
	if ti.Rat != "" {
		writeRationalTag(tf, tag, ti)
		return
	}

	wc := tf.FieldWriteCount(tag)

	switch ft {
	case 3, 8: // SHORT, SSHORT
		if wc == 1 {
			tf.SetFieldUint16(tag, tagUint16(ti))
		} else if wc < 0 {
			// Variable-count SHORT array (e.g. ISO tag 34855: SETGET_C16_UINT16)
			if v := tagUint16(ti); v != 0 {
				tf.SetFieldUint16Slice(tag, []uint16{v})
			}
		}
	case 4, 9: // LONG, SLONG
		if wc == 1 {
			tf.SetFieldUint32(tag, tagUint32(ti))
		}
	case 1, 6: // BYTE, SBYTE
		if wc == 1 {
			tf.SetFieldUint8(tag, tagUint8(ti))
		} else if data := decodeBinary(ti); len(data) > 0 {
			if !tf.FieldPassCount(tag) {
				tf.SetFieldC0ByteSlice(tag, data)
			} else {
				tf.SetFieldByteSlice(tag, data)
			}
		}
	case 7: // UNDEFINED
		if wc == 1 {
			if data := decodeBinary(ti); len(data) > 0 {
				tf.SetFieldUint8(tag, data[0])
			}
		} else if data := decodeBinary(ti); len(data) > 0 {
			if !tf.FieldPassCount(tag) {
				tf.SetFieldC0ByteSlice(tag, data)
			} else {
				tf.SetFieldByteSlice(tag, data)
			}
		}
	case 2: // ASCII
		// Prefer ti.Num (raw value) over ti.Val (display value).
		// GPS Ref tags: Num="N", Val="North" — EXIF spec requires the raw form.
		if s, ok := ti.Num.(string); ok && s != "" {
			tf.SetFieldString(tag, s)
		} else if ti.Val != "" {
			tf.SetFieldString(tag, ti.Val)
		}
	case 5, 10: // RATIONAL, SRATIONAL (no rat field — single value only)
		if wc == 1 {
			tf.SetFieldDouble(tag, tagFloat(ti))
		}
	case 11, 12: // FLOAT, DOUBLE
		if wc == 1 {
			tf.SetFieldDouble(tag, tagFloat(ti))
		}
	case 13: // IFD
	}
}

// writeRationalTag writes a RATIONAL/SRATIONAL tag using double-precision.
func writeRationalTag(tf *golibtiff.TIFF, tag golibtiff.Tag, ti TagInfo) {
	vals := parseRational(ti.Rat)
	switch len(vals) {
	case 0:
		return
	case 1:
		tf.SetFieldDouble(tag, vals[0])
	default:
		sz := tf.FieldSetGetSize(tag)
		if !tf.FieldPassCount(tag) {
			if sz == 4 {
				tf.SetFieldC0FloatSlice(tag, vals)
			} else {
				tf.SetFieldC0DoubleSlice(tag, vals)
			}
		} else {
			if sz == 4 {
				tf.SetFieldFloatSlice(tag, vals)
			} else {
				tf.SetFieldDoubleSlice(tag, vals)
			}
		}
	}
}

// writeGroup writes all tags in a group to the current TIFF directory.
func writeGroup(tf *golibtiff.TIFF, tags map[string]TagInfo, skip map[uint32]bool) {
	for _, ti := range tags {
		writeTag(tf, ti, skip)
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

// xmpTagMap maps ExifTool JSON tag names to XMP prefix and array flag.
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

func buildXMPacket(xmpValues map[string]TagInfo, rawFileName string) []byte {
	type elem struct {
		prefix  string
		key     string
		value   string
		isArray bool
	}

	var elements []elem
	usedNS := make(map[string]string)
	for name, ti := range xmpValues {
		info, ok := xmpTagMap[name]
		if !ok || ti.Val == "" {
			continue
		}
		elements = append(elements, elem{
			prefix: info.prefix, key: name,
			value: ti.Val, isArray: info.isArray,
		})
		usedNS[info.prefix] = xmpNS[info.prefix]
	}
	if len(elements) == 0 && rawFileName == "" {
		return nil
	}
	if rawFileName != "" {
		usedNS["crs"] = xmpNS["crs"]
	}

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
			writeXMLElement(enc, name, e.value)
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

func writeXMPArray(enc *xml.Encoder, name, value string) {
	var items []string
	for part := range strings.SplitSeq(value, ", ") {
		if part != "" {
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
