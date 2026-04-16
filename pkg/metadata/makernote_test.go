package metadata

import (
	"encoding/base64"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"open-make-tiff/pkg/golibtiff"
)

// newEMWithMakerNote creates an ExtractedMetadata with MakerNote binary data
// and calls analyzeMakerNote(0), returning the populated em.
func newEMWithMakerNote(t *testing.T, data []byte) *ExtractedMetadata {
	t.Helper()
	em := NewExtractedMetadata(nil)
	em.EXIF["MakerNote"] = TagInfo{ID: "37500", Val: "base64:" + base64.StdEncoding.EncodeToString(data)}
	em.analyzeMakerNote(0)
	return em
}

// newEMWithMakerNoteAndMake creates an ExtractedMetadata with MakerNote binary data
// and a Make tag in IFD0, then calls analyzeMakerNote(baseOld).
func newEMWithMakerNoteAndMake(t *testing.T, data []byte, makeVal string, baseOld uint32) *ExtractedMetadata {
	t.Helper()
	em := NewExtractedMetadata(nil)
	em.IFD0["Make"] = TagInfo{ID: "271", Val: makeVal}
	em.EXIF["MakerNote"] = TagInfo{ID: "37500", Val: "base64:" + base64.StdEncoding.EncodeToString(data)}
	em.analyzeMakerNote(baseOld)
	return em
}

// --- detectMakerNoteKind tests ---

func TestDetectMakerNoteKind(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		kind makerNoteKind
	}{
		{"Nikon", append([]byte("Nikon\x00\x02\x10\x00\x00\x00"), make([]byte, 100)...), makerNoteSkip},
		{"Fuji", append([]byte("FUJIFILM"), make([]byte, 100)...), makerNoteSkip},
		{"Olympus", append([]byte("OLYMPUS\x00\x00\x00"), make([]byte, 100)...), makerNoteSkip},
		{"OMSystem", append([]byte("OM SYSTEM\x00\x00\x00"), make([]byte, 100)...), makerNoteSkip},
		{"RicohPentaxII", append([]byte("RICOH\x00II\x00\x00"), make([]byte, 100)...), makerNoteSkip},
		{"RicohPentaxMM", append([]byte("RICOH\x00MM\x00\x00"), make([]byte, 100)...), makerNoteSkip},
		{"Pentax", append([]byte("PENTAX \x00\x00\x00\x00"), make([]byte, 100)...), makerNoteSkip},
		{"Samsung", append([]byte("SAMSUNG\x00\x00\x00\x00"), make([]byte, 100)...), makerNoteSkip},
		{"AppleiOS", append([]byte("Apple iOS\x00\x00\x00"), make([]byte, 100)...), makerNoteSkip},
		{"PhaseOneLE", append([]byte("IIII.waR"), make([]byte, 100)...), makerNoteSkip},
		{"PhaseOneBE", append([]byte("MMMMRaw."), make([]byte, 100)...), makerNoteSkip},
		{"SonyDSC", append([]byte("SONY DSC \x00II"), make([]byte, 100)...), makerNoteAnalyze},
		{"SonyCAM", append([]byte("SONY CAM \x00MM"), make([]byte, 100)...), makerNoteAnalyze},
		{"SonyMobile", append([]byte("SONY MOBILE\x00II"), make([]byte, 100)...), makerNoteAnalyze},
		{"Panasonic", append([]byte("Panasonic\x00II"), make([]byte, 100)...), makerNoteAnalyze},
		{"CanonLE", append([]byte{0x49, 0x49, 0x2a, 0x00, 0x08, 0x00, 0x00, 0x00}, make([]byte, 100)...), makerNoteAnalyze},
		{"CanonBE", append([]byte{0x4d, 0x4d, 0x00, 0x2a, 0x00, 0x00, 0x00, 0x08}, make([]byte, 100)...), makerNoteAnalyze},
		{"Empty", nil, makerNoteSkip},
		{"Short", []byte{0x49, 0x49}, makerNoteSkip},
		{"SonyDSC_TooShort", []byte("SONY DSC \x00"), makerNoteSkip},
		{"Unknown", []byte("UNKNOWN_FORMAT"), makerNoteSkip},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := detectMakerNoteKind(tt.data)
			if info.kind != tt.kind {
				t.Errorf("got %v, want %v", info.kind, tt.kind)
			}
		})
	}
}

func TestDetectMakerNoteKind_ByteOrder(t *testing.T) {
	// Sony DSC with LE
	le := append([]byte("SONY DSC \x00II"), make([]byte, 100)...)
	info := detectMakerNoteKind(le)
	if info.bo != binary.LittleEndian {
		t.Errorf("Sony DSC LE: got %v, want LittleEndian", info.bo)
	}
	if info.ifdStart != 12 {
		t.Errorf("Sony DSC ifdStart: got %d, want 12", info.ifdStart)
	}

	// Sony DSC with BE
	be := append([]byte("SONY DSC \x00MM"), make([]byte, 100)...)
	info = detectMakerNoteKind(be)
	if info.bo != binary.BigEndian {
		t.Errorf("Sony DSC BE: got %v, want BigEndian", info.bo)
	}
}

func TestDetectMakerNoteKind_Panasonic(t *testing.T) {
	le := append([]byte("Panasonic\x00II"), make([]byte, 100)...)
	info := detectMakerNoteKind(le)
	if info.kind != makerNoteAnalyze {
		t.Fatalf("got %v, want makerNoteAnalyze", info.kind)
	}
	if info.bo != binary.LittleEndian {
		t.Errorf("Panasonic LE: got %v, want LittleEndian", info.bo)
	}
	if info.ifdStart != 12 {
		t.Errorf("Panasonic ifdStart: got %d, want 12", info.ifdStart)
	}
}

func TestDetectMakerNoteKind_PentaxAOC(t *testing.T) {
	le := append([]byte("AOC\x00II"), make([]byte, 100)...)
	info := detectMakerNoteKind(le)
	if info.kind != makerNoteAnalyze {
		t.Fatalf("got %v, want makerNoteAnalyze", info.kind)
	}
	if info.bo != binary.LittleEndian {
		t.Errorf("Pentax AOC LE: got %v, want LittleEndian", info.bo)
	}
	if info.ifdStart != 4 {
		t.Errorf("Pentax AOC ifdStart: got %d, want 4", info.ifdStart)
	}
}

func TestDetectMakerNoteKind_OldOlympus(t *testing.T) {
	data := append([]byte("OLYMP\x00MM"), make([]byte, 100)...)
	info := detectMakerNoteKind(data)
	if info.kind != makerNoteAnalyze {
		t.Fatalf("got %v, want makerNoteAnalyze", info.kind)
	}
	if info.ifdStart != 8 {
		t.Errorf("Old Olympus ifdStart: got %d, want 8", info.ifdStart)
	}
}

// --- AnalyzeMakerNote tests ---

func TestAnalyzeAbsoluteIFD(t *testing.T) {
	entries := [][4]uint32{{1, 4, 10, 5000}}
	data := buildSonyMakerNote(entries, 0, 50)

	em := newEMWithMakerNote(t, data)
	if em.MakerNoteFixup == nil {
		t.Fatal("expected non-nil fixup for absolute offsets")
	}
	wantBaseOld := uint32(5000) - uint32(12+2+12+4)
	if em.MakerNoteFixup.BaseOld != wantBaseOld {
		t.Errorf("BaseOld = %d, want %d", em.MakerNoteFixup.BaseOld, wantBaseOld)
	}
	if em.MakerNoteFixup.BO != binary.LittleEndian {
		t.Errorf("BO = %v, want LittleEndian", em.MakerNoteFixup.BO)
	}
	if len(em.MakerNoteFixup.Pointers) == 0 {
		t.Error("expected at least one pointer")
	}
}

func TestAnalyzeAbsoluteIFD_ExternalBaseOld(t *testing.T) {
	entries := [][4]uint32{{1, 4, 10, 5310}}
	data := buildSonyMakerNote(entries, 0, 50)

	// Simulate correct baseOld from source file parsing
	em := newEMWithMakerNoteAndMake(t, data, "SONY", 5310)
	if em.MakerNoteFixup == nil {
		t.Fatal("expected non-nil fixup")
	}
	if em.MakerNoteFixup.BaseOld != 5310 {
		t.Errorf("BaseOld = %d, want 5310", em.MakerNoteFixup.BaseOld)
	}
}

func TestAnalyzeRelativeIFD(t *testing.T) {
	entries := [][4]uint32{{1, 4, 10, 30}}
	data := buildSonyMakerNote(entries, 0, 40)

	em := newEMWithMakerNote(t, data)
	if em.MakerNoteFixup != nil {
		t.Errorf("expected nil fixup for relative offsets, got BaseOld=%d", em.MakerNoteFixup.BaseOld)
	}
}

func TestAnalyzeNoPointers(t *testing.T) {
	entries := [][4]uint32{
		{1, 3, 1, 100},
		{2, 4, 1, 200},
	}
	data := buildSonyMakerNote(entries, 0, 0)

	em := newEMWithMakerNote(t, data)
	if em.MakerNoteFixup != nil {
		t.Error("expected nil fixup when no offset pointers exist")
	}
}

func TestAnalyzeEmptyIFD(t *testing.T) {
	entries := [][4]uint32{}
	data := buildSonyMakerNote(entries, 0, 0)

	em := newEMWithMakerNote(t, data)
	if em.MakerNoteFixup != nil {
		t.Error("expected nil for empty IFD")
	}
}

func TestAnalyzeInvalidType(t *testing.T) {
	entries := [][4]uint32{{1, 99, 1, 0}}
	data := buildSonyMakerNote(entries, 0, 0)

	em := newEMWithMakerNote(t, data)
	if em.MakerNoteFixup != nil {
		t.Error("expected nil for unknown type entries")
	}
}

func TestAnalyzeSubIFD(t *testing.T) {
	bo := binary.LittleEndian
	baseOld := uint32(5000)
	ifdEnd := 12 + 2 + 2*12 + 4 // = 42
	subIFDRelStart := uint32(60)

	// Sub-IFD with an entry whose dataSize > 4 and a valid pointer
	subEntries := [][4]uint32{
		{10, 4, 2, baseOld + subIFDRelStart + 2 + 1*12 + 4 + 10},
	}
	subIFD := buildIFD(bo, subEntries, 0)

	mainEntries := [][4]uint32{
		{1, 4, 10, baseOld + uint32(ifdEnd) + 20},
		{2, 4, 1, 0}, // dummy
	}
	extraBytes := int(subIFDRelStart) + len(subIFD) + 40
	// next-IFD pointer points to sub-IFD location
	data := buildSonyMakerNote(mainEntries, baseOld+subIFDRelStart, extraBytes)
	copy(data[subIFDRelStart:], subIFD)

	em := newEMWithMakerNote(t, data)
	if em.MakerNoteFixup == nil {
		t.Fatal("expected non-nil fixup with sub-IFD")
	}
	// Main IFD entry 0 pointer + sub-IFD entry 0 pointer + next-IFD pointer = 3 unique pointers
	if len(em.MakerNoteFixup.Pointers) < 2 {
		t.Errorf("expected at least 2 unique pointers, got %d", len(em.MakerNoteFixup.Pointers))
	}
}

func TestAnalyzeCanonFooter(t *testing.T) {
	bo := binary.LittleEndian
	tiffHeader := make([]byte, 8)
	tiffHeader[0] = 'I'
	tiffHeader[1] = 'I'
	bo.PutUint16(tiffHeader[2:4], 0x002a)
	bo.PutUint32(tiffHeader[4:8], 8)

	entries := [][4]uint32{{1, 4, 10, 5000}}
	ifd := buildIFD(bo, entries, 0)

	data := make([]byte, 8+len(ifd)+40+8)
	copy(data, tiffHeader)
	copy(data[8:], ifd)
	footerOff := len(data) - 8
	data[footerOff] = 'I'
	data[footerOff+1] = 'I'
	data[footerOff+2] = 0x2a
	data[footerOff+3] = 0x00
	bo.PutUint32(data[footerOff+4:footerOff+8], 9999)

	em := newEMWithMakerNoteAndMake(t, data, "Canon", 0)
	if em.MakerNoteFixup == nil {
		t.Fatal("expected non-nil fixup with Canon footer")
	}
	if !em.MakerNoteFixup.HasFooter {
		t.Error("expected HasFooter = true")
	}
}

func TestAnalyzeCanonFooter_NonCanonIgnored(t *testing.T) {
	bo := binary.LittleEndian
	tiffHeader := make([]byte, 8)
	tiffHeader[0] = 'I'
	tiffHeader[1] = 'I'
	bo.PutUint16(tiffHeader[2:4], 0x002a)
	bo.PutUint32(tiffHeader[4:8], 8)

	entries := [][4]uint32{{1, 4, 10, 5000}}
	ifd := buildIFD(bo, entries, 0)

	data := make([]byte, 8+len(ifd)+40+8)
	copy(data, tiffHeader)
	copy(data[8:], ifd)
	footerOff := len(data) - 8
	data[footerOff] = 'I'
	data[footerOff+1] = 'I'
	data[footerOff+2] = 0x2a
	data[footerOff+3] = 0x00
	bo.PutUint32(data[footerOff+4:footerOff+8], 9999)

	// Sony Make — footer should NOT be detected
	em := newEMWithMakerNoteAndMake(t, data, "SONY", 0)
	if em.MakerNoteFixup == nil {
		t.Fatal("expected non-nil fixup (pointer data exists)")
	}
	if em.MakerNoteFixup.HasFooter {
		t.Error("expected HasFooter = false for non-Canon Make")
	}
}

func TestAnalyzeOffsetOutOfBounds_Skipped(t *testing.T) {
	// Two entries: one valid, one out-of-bounds. Valid should be kept.
	entries := [][4]uint32{
		{1, 4, 10, 5000},
		{2, 4, 10, 20000}, // out of bounds
	}
	data := buildSonyMakerNote(entries, 0, 50)

	em := newEMWithMakerNote(t, data)
	if em.MakerNoteFixup == nil {
		t.Fatal("expected non-nil fixup (one valid pointer)")
	}
	if len(em.MakerNoteFixup.Pointers) == 0 {
		t.Error("expected at least one valid pointer to be kept")
	}
}

func TestAnalyzeOffsetOutOfBounds_AllSkipped(t *testing.T) {
	entries := [][4]uint32{{1, 4, 10, 20000}}
	data := buildSonyMakerNote(entries, 0, 0)

	em := newEMWithMakerNote(t, data)
	if em.MakerNoteFixup != nil {
		t.Error("expected nil MakerNoteFixup when all pointers are out of bounds")
	}
}

// --- Cross-vendor analyze tests ---

func TestAnalyzePanasonic(t *testing.T) {
	entries := [][4]uint32{{1, 4, 10, 5000}}
	data := buildPanasonicMakerNote(entries, 0, 50)

	em := newEMWithMakerNote(t, data)
	if em.MakerNoteFixup == nil {
		t.Fatal("expected non-nil fixup for Panasonic absolute offsets")
	}
	if em.MakerNoteFixup.BO != binary.LittleEndian {
		t.Errorf("BO = %v, want LittleEndian", em.MakerNoteFixup.BO)
	}
	if len(em.MakerNoteFixup.Pointers) == 0 {
		t.Error("expected at least one pointer")
	}
}

func TestAnalyzePentaxAOC(t *testing.T) {
	entries := [][4]uint32{{1, 4, 10, 5000}}
	data := buildPentaxAOCMakerNote(entries, 0, 50)

	em := newEMWithMakerNote(t, data)
	if em.MakerNoteFixup == nil {
		t.Fatal("expected non-nil fixup for Pentax AOC absolute offsets")
	}
	if len(em.MakerNoteFixup.Pointers) == 0 {
		t.Error("expected at least one pointer")
	}
}

func TestAnalyzeOldOlympus(t *testing.T) {
	bo := binary.LittleEndian
	entries := [][4]uint32{{1, 4, 10, 5000}}
	data := buildOldOlympusMakerNote(bo, entries, 0, 50)

	em := newEMWithMakerNote(t, data)
	if em.MakerNoteFixup == nil {
		t.Fatal("expected non-nil fixup for Old Olympus absolute offsets")
	}
	if len(em.MakerNoteFixup.Pointers) == 0 {
		t.Error("expected at least one pointer")
	}
}

// --- findMakerNoteOffset tests ---

func TestFindMakerNoteOffset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.tiff")

	makerNote := []byte("SONY DSC \x00\x00\x00" + "TEST_MAKER_NOTE_DATA_FOR_FIND_TEST")
	writeTIFFWithMakerNote(t, path, makerNote)

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	loc, err := findMakerNoteOffset(path)
	if err != nil {
		t.Fatalf("findMakerNoteOffset: %v", err)
	}
	if loc.offset == 0 {
		t.Error("MakerNote offset should not be 0")
	}
	if loc.dataLen != uint32(len(makerNote)) {
		t.Errorf("dataLen = %d, want %d", loc.dataLen, len(makerNote))
	}
}

// --- PatchMakerNoteOffsets tests ---

func TestPatchRoundTrip(t *testing.T) {
	bo := binary.LittleEndian
	baseOld := uint32(10000)

	entries := [][4]uint32{
		{1, 4, 10, baseOld + 32},
		{2, 4, 5, baseOld + 72},
	}
	data := buildSonyMakerNote(entries, 0, 80)

	ifdStart := 12
	ptrPos1 := ifdStart + 2 + 0*12 + 8
	ptrPos2 := ifdStart + 2 + 1*12 + 8

	em := NewExtractedMetadata(nil)
	em.MakerNoteFixup = &MakerNoteFixup{
		BaseOld:  baseOld,
		BO:       bo,
		Pointers: []int{ptrPos1, ptrPos2},
		DataLen:  len(data),
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "patch_test.tiff")
	writeTIFFWithMakerNote(t, path, data)

	if err := em.PatchMakerNoteOffsets(path); err != nil {
		t.Fatalf("PatchMakerNoteOffsets: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	loc, err := findMakerNoteOffset(path)
	if err != nil {
		t.Fatalf("findMakerNoteOffset: %v", err)
	}

	shift := int64(loc.offset) - int64(baseOld)
	buf := make([]byte, 4)

	if _, err := f.Seek(int64(loc.offset)+int64(ptrPos1), 0); err != nil {
		t.Fatalf("Seek: %v", err)
	}
	if _, err := f.Read(buf); err != nil {
		t.Fatalf("Read: %v", err)
	}
	newVal1 := bo.Uint32(buf)
	wantVal1 := uint32(int64(baseOld+32) + shift)
	if newVal1 != wantVal1 {
		t.Errorf("pointer1 = %d, want %d (baseOld+32+shift)", newVal1, wantVal1)
	}

	if _, err := f.Seek(int64(loc.offset)+int64(ptrPos2), 0); err != nil {
		t.Fatalf("Seek: %v", err)
	}
	if _, err := f.Read(buf); err != nil {
		t.Fatalf("Read: %v", err)
	}
	newVal2 := bo.Uint32(buf)
	wantVal2 := uint32(int64(baseOld+72) + shift)
	if newVal2 != wantVal2 {
		t.Errorf("pointer2 = %d, want %d (baseOld+72+shift)", newVal2, wantVal2)
	}

	t.Logf("baseOld=%d, baseNew=%d, shift=%d", baseOld, loc.offset, shift)
}

func TestPatchWithFooter(t *testing.T) {
	bo := binary.LittleEndian
	baseOld := uint32(10000)

	tiffHeader := make([]byte, 8)
	tiffHeader[0] = 'I'
	tiffHeader[1] = 'I'
	bo.PutUint16(tiffHeader[2:4], 0x002a)
	bo.PutUint32(tiffHeader[4:8], 8)

	entries := [][4]uint32{{1, 4, 10, baseOld + 20}}
	ifd := buildIFD(bo, entries, 0)

	data := make([]byte, 8+len(ifd)+40+8)
	copy(data, tiffHeader)
	copy(data[8:], ifd)
	footerOff := len(data) - 8
	data[footerOff] = 'I'
	data[footerOff+1] = 'I'
	data[footerOff+2] = 0x2a
	data[footerOff+3] = 0x00
	bo.PutUint32(data[footerOff+4:footerOff+8], baseOld)

	ptrPos := 8 + 2 + 0*12 + 8

	em := NewExtractedMetadata(nil)
	em.MakerNoteFixup = &MakerNoteFixup{
		BaseOld:   baseOld,
		BO:        bo,
		Pointers:  []int{ptrPos, footerOff + 4},
		DataLen:   len(data),
		HasFooter: true,
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "footer_test.tiff")
	writeTIFFWithMakerNote(t, path, data)

	if err := em.PatchMakerNoteOffsets(path); err != nil {
		t.Fatalf("PatchMakerNoteOffsets: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	loc, err := findMakerNoteOffset(path)
	if err != nil {
		t.Fatalf("findMakerNoteOffset: %v", err)
	}

	footerFileOff := int64(loc.offset) + int64(em.MakerNoteFixup.DataLen) - 4
	buf := make([]byte, 4)
	if _, err := f.Seek(footerFileOff, 0); err != nil {
		t.Fatalf("Seek footer: %v", err)
	}
	if _, err := f.Read(buf); err != nil {
		t.Fatalf("Read footer: %v", err)
	}
	footerVal := bo.Uint32(buf)
	if footerVal != loc.offset {
		t.Errorf("footer = %d, want %d (baseNew)", footerVal, loc.offset)
	}
}

func TestPatchPanasonicRoundTrip(t *testing.T) {
	bo := binary.LittleEndian
	baseOld := uint32(10000)

	entries := [][4]uint32{
		{1, 4, 10, baseOld + 32},
	}
	data := buildPanasonicMakerNote(entries, 0, 80)

	ifdStart := 12
	ptrPos1 := ifdStart + 2 + 0*12 + 8

	em := NewExtractedMetadata(nil)
	em.MakerNoteFixup = &MakerNoteFixup{
		BaseOld:  baseOld,
		BO:       bo,
		Pointers: []int{ptrPos1},
		DataLen:  len(data),
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "panasonic_patch.tiff")
	writeTIFFWithMakerNote(t, path, data)

	if err := em.PatchMakerNoteOffsets(path); err != nil {
		t.Fatalf("PatchMakerNoteOffsets: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	loc, err := findMakerNoteOffset(path)
	if err != nil {
		t.Fatalf("findMakerNoteOffset: %v", err)
	}

	buf := make([]byte, 4)
	if _, err := f.Seek(int64(loc.offset)+int64(ptrPos1), 0); err != nil {
		t.Fatalf("Seek: %v", err)
	}
	if _, err := f.Read(buf); err != nil {
		t.Fatalf("Read: %v", err)
	}
	newVal := bo.Uint32(buf)
	wantVal := uint32(int64(baseOld+32) + int64(loc.offset) - int64(baseOld))
	if newVal != wantVal {
		t.Errorf("pointer = %d, want %d", newVal, wantVal)
	}
}

// --- isCanonMake tests ---

func TestIsCanonMake(t *testing.T) {
	tests := []struct {
		ifd0 map[string]TagInfo
		want bool
	}{
		{map[string]TagInfo{"Make": {Val: "Canon"}}, true},
		{map[string]TagInfo{"Make": {Val: "Canon Inc."}}, true},
		{map[string]TagInfo{"Make": {Val: "SONY"}}, false},
		{map[string]TagInfo{"Make": {Val: "Panasonic"}}, false},
		{map[string]TagInfo{}, false},
		{map[string]TagInfo{"Make": {Val: "NIKON"}}, false},
	}
	for _, tt := range tests {
		got := isCanonMake(tt.ifd0)
		if got != tt.want {
			t.Errorf("isCanonMake(%v) = %v, want %v", tt.ifd0["Make"].Val, got, tt.want)
		}
	}
}

// --- helpers ---

func writeTIFFWithMakerNote(t *testing.T, path string, makerNote []byte) {
	t.Helper()
	tf, err := golibtiff.Open(path, golibtiff.OpenWrite)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer tf.Close()

	setupBasicTIFF(t, tf)
	if err := tf.SetFieldUint64(golibtiff.TagEXIFIFD, 0); err != nil {
		t.Fatalf("Reserve EXIF IFD: %v", err)
	}
	pixel := []byte{0, 0, 0}
	if err := tf.WriteScanline(pixel, 0); err != nil {
		t.Fatalf("WriteScanline: %v", err)
	}
	if err := tf.CheckpointDirectory(); err != nil {
		t.Fatalf("CheckpointDirectory: %v", err)
	}
	if err := tf.CreateEXIFDirectory(); err != nil {
		t.Fatalf("CreateEXIFDirectory: %v", err)
	}
	if err := tf.SetFieldByteSlice(golibtiff.TagExifMakerNote, makerNote); err != nil {
		t.Fatalf("SetFieldByteSlice MakerNote: %v", err)
	}
	exifOffset, err := tf.WriteCustomDirectory()
	if err != nil {
		t.Fatalf("WriteCustomDirectory: %v", err)
	}
	if err := tf.SetDirectory(0); err != nil {
		t.Fatalf("SetDirectory(0): %v", err)
	}
	if err := tf.SetFieldUint64(golibtiff.TagEXIFIFD, exifOffset); err != nil {
		t.Fatalf("SetField EXIFIFD: %v", err)
	}
	if err := tf.WriteDirectory(); err != nil {
		t.Fatalf("WriteDirectory: %v", err)
	}
}

func buildIFD(bo binary.ByteOrder, entries [][4]uint32, nextIFD uint32) []byte {
	entryCount := len(entries)
	size := 2 + entryCount*12 + 4
	buf := make([]byte, size)
	bo.PutUint16(buf[0:2], uint16(entryCount))
	for i, e := range entries {
		off := 2 + i*12
		bo.PutUint16(buf[off:off+2], uint16(e[0]))
		bo.PutUint16(buf[off+2:off+4], uint16(e[1]))
		bo.PutUint32(buf[off+4:off+8], e[2])
		bo.PutUint32(buf[off+8:off+12], e[3])
	}
	bo.PutUint32(buf[2+entryCount*12:], nextIFD)
	return buf
}

func setupBasicTIFF(t *testing.T, tf *golibtiff.TIFF) {
	t.Helper()
	for _, set := range []func() error{
		func() error { return tf.SetFieldUint32(golibtiff.TagImageWidth, 1) },
		func() error { return tf.SetFieldUint32(golibtiff.TagImageLength, 1) },
		func() error { return tf.SetFieldUint16(golibtiff.TagBitsPerSample, 8) },
		func() error { return tf.SetFieldUint16(golibtiff.TagSamplesPerPixel, 3) },
		func() error { return tf.SetFieldUint16(golibtiff.TagPhotometric, uint16(golibtiff.PhotometricRGB)) },
		func() error { return tf.SetFieldUint16(golibtiff.TagPlanarConfig, uint16(golibtiff.PlanarConfigContig)) },
		func() error { return tf.SetFieldUint32(golibtiff.TagRowsPerStrip, 1) },
	} {
		if err := set(); err != nil {
			t.Fatalf("setupBasicTIFF: %v", err)
		}
	}
}

func buildSonyMakerNote(entries [][4]uint32, nextIFD uint32, extraBytes int) []byte {
	bo := binary.LittleEndian
	prefix := []byte("SONY DSC \x00II")
	return buildMakerNoteWithPrefix(bo, prefix, entries, nextIFD, extraBytes, 12)
}

func buildPanasonicMakerNote(entries [][4]uint32, nextIFD uint32, extraBytes int) []byte {
	bo := binary.LittleEndian
	prefix := []byte("Panasonic\x00II")
	return buildMakerNoteWithPrefix(bo, prefix, entries, nextIFD, extraBytes, 12)
}

func buildPentaxAOCMakerNote(entries [][4]uint32, nextIFD uint32, extraBytes int) []byte {
	bo := binary.LittleEndian
	prefix := []byte("AOC\x00II")
	return buildMakerNoteWithPrefix(bo, prefix, entries, nextIFD, extraBytes, 4)
}

func buildOldOlympusMakerNote(bo binary.ByteOrder, entries [][4]uint32, nextIFD uint32, extraBytes int) []byte {
	prefix := make([]byte, 10)
	copy(prefix, "OLYMP\x00")
	if bo == binary.BigEndian {
		prefix[8] = 'M'
		prefix[9] = 'M'
	} else {
		prefix[8] = 'I'
		prefix[9] = 'I'
	}
	return buildMakerNoteWithPrefix(bo, prefix, entries, nextIFD, extraBytes, 8)
}

func buildMakerNoteWithPrefix(bo binary.ByteOrder, prefix []byte, entries [][4]uint32, nextIFD uint32, extraBytes int, ifdStart int) []byte {
	entryCount := len(entries)
	ifdSize := 2 + entryCount*12 + 4
	total := ifdStart + ifdSize + extraBytes
	buf := make([]byte, total)
	copy(buf, prefix)

	bo.PutUint16(buf[ifdStart:ifdStart+2], uint16(entryCount))
	for i, e := range entries {
		off := ifdStart + 2 + i*12
		bo.PutUint16(buf[off:off+2], uint16(e[0]))
		bo.PutUint16(buf[off+2:off+4], uint16(e[1]))
		bo.PutUint32(buf[off+4:off+8], e[2])
		bo.PutUint32(buf[off+8:off+12], e[3])
	}
	bo.PutUint32(buf[ifdStart+2+entryCount*12:], nextIFD)
	return buf
}
