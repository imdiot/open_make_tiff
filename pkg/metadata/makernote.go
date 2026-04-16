package metadata

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strings"

	"open-make-tiff/pkg/golibtiff"
)

// typeSizeTable maps TIFF data type IDs to element sizes in bytes.
var typeSizeTable = [256]uint32{
	1: 1, 2: 1, 3: 2, 4: 4, 5: 8, // BYTE, ASCII, SHORT, LONG, RATIONAL
	6: 1, 7: 1, 8: 2, 9: 4, 10: 8, // SBYTE, UNDEFINED, SSHORT, SLONG, SRATIONAL
	11: 4, 12: 8, // FLOAT, DOUBLE
}

type makerNoteKind int

const (
	makerNoteSkip    makerNoteKind = iota // known relative offset or NotIFD, skip
	makerNoteAnalyze                      // needs binary analysis to determine offset type
)

type makerNoteInfo struct {
	kind     makerNoteKind
	ifdStart int
	bo       binary.ByteOrder
}

// detectByteOrder detects byte order from 2-byte prefix: II=LittleEndian, MM=BigEndian.
func detectByteOrder(b []byte) binary.ByteOrder {
	if len(b) >= 2 && b[0] == 'M' && b[1] == 'M' {
		return binary.BigEndian
	}
	return binary.LittleEndian
}

// detectMakerNoteKind identifies MakerNote format and returns analysis info.
// For known relative-offset formats it returns makerNoteSkip.
// For formats needing binary analysis it returns makerNoteAnalyze with ifdStart and bo.
func detectMakerNoteKind(data []byte) makerNoteInfo {
	if len(data) < 18 { // minimum: signature(2) + IFD(2+12+4)
		return makerNoteInfo{kind: makerNoteSkip}
	}

	// Long signatures first (higher priority)
	if len(data) >= 12 {
		sig := string(data[:12])
		switch {
		case sig == "Nikon\x00\x02\x10\x00\x00\x00":
			return makerNoteInfo{kind: makerNoteSkip}
		case sig == "OLYMPUS\x00\x00\x00":
			return makerNoteInfo{kind: makerNoteSkip}
		case sig == "OM SYSTEM\x00\x00\x00":
			return makerNoteInfo{kind: makerNoteSkip}
		}
	}

	if len(data) >= 10 {
		sig := string(data[:10])
		switch {
		case sig == "RICOH\x00II\x00\x00" || sig == "RICOH\x00MM\x00\x00":
			return makerNoteInfo{kind: makerNoteSkip}
		case sig == "PENTAX \x00\x00\x00\x00" || sig == "SAMSUNG\x00\x00\x00\x00":
			return makerNoteInfo{kind: makerNoteSkip}
		case sig == "Apple iOS\x00\x00\x00":
			return makerNoteInfo{kind: makerNoteSkip}
		}
	}

	if len(data) >= 8 {
		sig8 := string(data[:8])
		if sig8 == "FUJIFILM" {
			return makerNoteInfo{kind: makerNoteSkip}
		}
		if sig8 == "IIII.waR" || sig8 == "MMMMRaw." {
			return makerNoteInfo{kind: makerNoteSkip}
		}
	}

	// Sony DSC/CAM — needs analysis
	if len(data) >= 14 {
		sig := string(data[:10])
		if sig == "SONY DSC \x00" || sig == "SONY CAM \x00" {
			minSize := 12 + 2 + 12 + 4
			if len(data) < minSize {
				return makerNoteInfo{kind: makerNoteSkip}
			}
			return makerNoteInfo{
				kind:     makerNoteAnalyze,
				ifdStart: 12,
				bo:       detectByteOrder(data[10:12]),
			}
		}
	}

	// Sony MOBILE — needs analysis
	if len(data) >= 30 { // minSize = 12 + 2 + 12 + 4
		if string(data[:12]) == "SONY MOBILE\x00" {
			return makerNoteInfo{
				kind:     makerNoteAnalyze,
				ifdStart: 12,
				bo:       detectByteOrder(data[12:14]),
			}
		}
	}

	// Panasonic — needs analysis
	if len(data) >= 12 && string(data[:10]) == "Panasonic\x00" {
		minSize := 12 + 2 + 12 + 4
		if len(data) < minSize {
			return makerNoteInfo{kind: makerNoteSkip}
		}
		return makerNoteInfo{
			kind:     makerNoteAnalyze,
			ifdStart: 12,
			bo:       detectByteOrder(data[10:12]),
		}
	}

	// Pentax AOC — needs analysis
	if len(data) >= 6 && string(data[:4]) == "AOC\x00" {
		minSize := 4 + 2 + 12 + 4
		if len(data) < minSize {
			return makerNoteInfo{kind: makerNoteSkip}
		}
		return makerNoteInfo{
			kind:     makerNoteAnalyze,
			ifdStart: 4,
			bo:       detectByteOrder(data[4:6]),
		}
	}

	// Old Olympus / Epson — needs analysis
	if len(data) >= 12 && (string(data[:6]) == "OLYMP\x00" || string(data[:6]) == "EPSON\x00") {
		minSize := 8 + 2 + 12 + 4
		if len(data) < minSize {
			return makerNoteInfo{kind: makerNoteSkip}
		}
		return makerNoteInfo{
			kind:     makerNoteAnalyze,
			ifdStart: 8,
			bo:       detectByteOrder(data[8:10]),
		}
	}

	// Standard Ricoh — needs analysis
	if len(data) >= 12 && string(data[:6]) == "Ricoh\x00" {
		minSize := 8 + 2 + 12 + 4
		if len(data) < minSize {
			return makerNoteInfo{kind: makerNoteSkip}
		}
		return makerNoteInfo{
			kind:     makerNoteAnalyze,
			ifdStart: 8,
			bo:       detectByteOrder(data[8:10]),
		}
	}

	// Bare TIFF header (Canon CR2, Sony5 ARW) — needs analysis
	if len(data) >= 8 {
		bo := detectByteOrder(data)
		magic := bo.Uint16(data[2:4])
		if magic == 0x002a {
			ifdStart := int(bo.Uint32(data[4:8]))
			minSize := ifdStart + 2 + 12 + 4
			if len(data) < minSize {
				return makerNoteInfo{kind: makerNoteSkip}
			}
			return makerNoteInfo{
				kind:     makerNoteAnalyze,
				ifdStart: ifdStart,
				bo:       bo,
			}
		}
	}

	// Bare IFD (no signature, no TIFF header) -- Sony5 and similar formats.
	// Detect by checking if the first 2 bytes form a plausible entry count
	// and subsequent bytes look like valid IFD entries.
	// Some vendors have entries that don't strictly ascend, so tolerate
	// a small fraction of out-of-order tags.
	if len(data) >= 14 {
		for _, tryBO := range []binary.ByteOrder{binary.LittleEndian, binary.BigEndian} {
			entryCount := tryBO.Uint16(data[0:2])
			if entryCount == 0 || entryCount > 500 {
				continue
			}
			ifdEnd := 2 + int(entryCount)*12 + 4
			if ifdEnd > len(data) {
				continue
			}
			prevTag := uint16(0)
			invalidCount := 0
			for i := uint16(0); i < entryCount; i++ {
				off := 2 + int(i)*12
				tag := tryBO.Uint16(data[off : off+2])
				typ := tryBO.Uint16(data[off+2 : off+4])
				if typ == 0 || typ > 12 || tag < prevTag {
					invalidCount++
				}
				prevTag = tag
			}
			if invalidCount < int(entryCount)/5 {
				return makerNoteInfo{
					kind:     makerNoteAnalyze,
					ifdStart: 0,
					bo:       tryBO,
				}
			}
		}
	}

	return makerNoteInfo{kind: makerNoteSkip}
}

// MakerNoteFixup holds offset correction data for absolute-offset MakerNotes.
// Set by ExtractedMetadata.analyzeMakerNote; nil for relative-offset, self-contained, or absent MakerNotes.
type MakerNoteFixup struct {
	BaseOld   uint32           // Inferred original base offset in source file
	BO        binary.ByteOrder // Byte order of MakerNote IFD content
	Pointers  []int            // Byte positions of offset pointers (relative to MakerNote start)
	DataLen   int              // Length of MakerNote binary data
	HasFooter bool             // Whether Canon-style TIFF footer exists
}

// makerNoteLocation holds the file position of MakerNote data in a TIFF file.
type makerNoteLocation struct {
	offset  uint32
	dataLen uint32
}

func findMakerNoteOffset(path string) (makerNoteLocation, error) {
	f, err := os.Open(path)
	if err != nil {
		return makerNoteLocation{}, err
	}
	defer f.Close()

	header := make([]byte, 8)
	if _, err := io.ReadFull(f, header); err != nil {
		return makerNoteLocation{}, fmt.Errorf("read TIFF header: %w", err)
	}
	fileBO := detectByteOrder(header)
	ifd0Offset := fileBO.Uint32(header[4:8])

	if _, err := f.Seek(int64(ifd0Offset), io.SeekStart); err != nil {
		return makerNoteLocation{}, fmt.Errorf("seek IFD0: %w", err)
	}
	var ifd0CountBuf [2]byte
	if _, err := io.ReadFull(f, ifd0CountBuf[:]); err != nil {
		return makerNoteLocation{}, fmt.Errorf("read IFD0 count: %w", err)
	}
	ifd0Count := fileBO.Uint16(ifd0CountBuf[:])
	if ifd0Count > 500 {
		return makerNoteLocation{}, fmt.Errorf("unreasonable IFD0 entry count: %d", ifd0Count)
	}

	ifd0Entries := make([]byte, int(ifd0Count)*12)
	if _, err := io.ReadFull(f, ifd0Entries); err != nil {
		return makerNoteLocation{}, fmt.Errorf("read IFD0 entries: %w", err)
	}

	var exifIFDOffset uint32
	found := false
	for i := 0; i < int(ifd0Count); i++ {
		entry := ifd0Entries[i*12 : (i+1)*12]
		tag := fileBO.Uint16(entry[0:2])
		if tag == uint16(golibtiff.TagEXIFIFD) {
			exifIFDOffset = fileBO.Uint32(entry[8:12])
			found = true
			break
		}
	}
	if !found {
		return makerNoteLocation{}, fmt.Errorf("EXIF IFD tag 34665 not found")
	}

	if _, err := f.Seek(int64(exifIFDOffset), io.SeekStart); err != nil {
		return makerNoteLocation{}, fmt.Errorf("seek EXIF IFD: %w", err)
	}
	var exifCountBuf [2]byte
	if _, err := io.ReadFull(f, exifCountBuf[:]); err != nil {
		return makerNoteLocation{}, fmt.Errorf("read EXIF IFD count: %w", err)
	}
	exifCount := fileBO.Uint16(exifCountBuf[:])
	if exifCount > 500 {
		return makerNoteLocation{}, fmt.Errorf("unreasonable EXIF entry count: %d", exifCount)
	}

	exifEntries := make([]byte, int(exifCount)*12)
	if _, err := io.ReadFull(f, exifEntries); err != nil {
		return makerNoteLocation{}, fmt.Errorf("read EXIF IFD entries: %w", err)
	}

	for i := 0; i < int(exifCount); i++ {
		entry := exifEntries[i*12 : (i+1)*12]
		tag := fileBO.Uint16(entry[0:2])
		if tag == uint16(golibtiff.TagExifMakerNote) {
			count := fileBO.Uint32(entry[4:8])
			return makerNoteLocation{
				offset:  fileBO.Uint32(entry[8:12]),
				dataLen: count,
			}, nil
		}
	}

	return makerNoteLocation{}, fmt.Errorf("MakerNote tag 37500 not found")
}

// AnalyzeMakerNote analyzes MakerNote binary for absolute offset fixup.
// srcPath is the source RAW file used to determine MakerNote's file offset.
// Falls back to offset inference if the source file cannot be parsed.
func (em *ExtractedMetadata) AnalyzeMakerNote(srcPath string) {
	var baseOld uint32
	if loc, err := findMakerNoteOffset(srcPath); err == nil {
		baseOld = loc.offset
	} else {
		em.logger.Debug("find MakerNote offset failed, using inference", "err", err)
	}
	em.analyzeMakerNote(baseOld)
}

// analyzeMakerNote performs binary analysis of MakerNote IFD entries to collect
// absolute offset pointers that need patching after TIFF re-write.
// baseOld: MakerNote's file offset in the source RAW (>0 = from source file, 0 = infer as fallback).
func (em *ExtractedMetadata) analyzeMakerNote(baseOld uint32) {
	var ti TagInfo
	for _, t := range em.EXIF {
		if t.TagID() == golibtiff.TagExifMakerNote {
			ti = t
			break
		}
	}
	if ti.Val == "" {
		return
	}
	data := ti.Binary()
	if len(data) == 0 {
		return
	}

	info := detectMakerNoteKind(data)
	if info.kind == makerNoteSkip {
		return
	}

	bo := info.bo
	ifdStart := info.ifdStart
	dataLen := len(data)

	if ifdStart+2 > dataLen {
		return
	}
	entryCount := bo.Uint16(data[ifdStart : ifdStart+2])
	if entryCount == 0 || entryCount > 500 {
		return
	}
	ifdEnd := ifdStart + 2 + int(entryCount)*12 + 4
	if ifdEnd > dataLen {
		em.logger.Debug("MakerNote IFD overflows data", "ifdEnd", ifdEnd, "dataLen", dataLen)
		return
	}

	// Pass 1: collect explicit offset pointers from main IFD
	ptrSet := make(map[int]struct{})
	minOffset := uint32(math.MaxUint32)

	for i := uint16(0); i < entryCount; i++ {
		entryOff := ifdStart + 2 + int(i)*12
		typ := bo.Uint16(data[entryOff+2 : entryOff+4])
		count := bo.Uint32(data[entryOff+4 : entryOff+8])
		ts := typeSizeTable[typ]
		if ts == 0 {
			continue
		}
		dataSize := count * ts
		if dataSize > 4 {
			ptrPos := entryOff + 8
			ptrVal := bo.Uint32(data[ptrPos : ptrPos+4])
			if ptrVal == 0 {
				continue
			}
			ptrSet[ptrPos] = struct{}{}
			if ptrVal < minOffset {
				minOffset = ptrVal
			}
		}
	}

	if minOffset == math.MaxUint32 {
		return
	}

	// Use provided baseOld when available; fall back to inference from pointer positions.
	if baseOld == 0 {
		baseOld = minOffset - uint32(ifdEnd)
	}
	if baseOld < uint32(ifdEnd) {
		return
	}

	// Validate: skip out-of-bounds pointers individually, abort only if all are invalid.
	dataLenU32 := uint32(dataLen)
	validSet := make(map[int]struct{}, len(ptrSet))
	for ptrPos := range ptrSet {
		val := bo.Uint32(data[ptrPos : ptrPos+4])
		if val >= baseOld && val < baseOld+dataLenU32 {
			validSet[ptrPos] = struct{}{}
		} else {
			em.logger.Debug("MakerNote skip out-of-bounds pointer",
				"ptrPos", ptrPos, "val", val, "baseOld", baseOld)
		}
	}
	ptrSet = validSet
	if len(ptrSet) == 0 {
		return
	}

	// Pass 2: recursive collection of sub-IFD pointers
	visited := map[uint32]bool{}
	var collectIFD func(ifdRelStart uint32)
	collectIFD = func(ifdRelStart uint32) {
		if visited[ifdRelStart] {
			return
		}
		visited[ifdRelStart] = true
		if ifdRelStart+2 > dataLenU32 {
			return
		}
		ec := bo.Uint16(data[ifdRelStart : ifdRelStart+2])
		if ec == 0 || ec > 500 {
			return
		}

		for i := uint16(0); i < ec; i++ {
			entryOff := ifdRelStart + 2 + uint32(i)*12
			if entryOff+12 > dataLenU32 {
				break
			}
			typ := bo.Uint16(data[entryOff+2 : entryOff+4])
			count := bo.Uint32(data[entryOff+4 : entryOff+8])
			ts := typeSizeTable[typ]

			if ts > 0 && count*ts > 4 {
				ptrPos := entryOff + 8
				ptrVal := bo.Uint32(data[ptrPos : ptrPos+4])
				if ptrVal != 0 && ptrVal >= baseOld && ptrVal < baseOld+dataLenU32 {
					ptrSet[int(ptrPos)] = struct{}{}
				}
			}
		}

		// next-IFD pointer at end of IFD
		nextPtrOff := ifdRelStart + 2 + uint32(ec)*12
		if nextPtrOff+4 <= dataLenU32 {
			nextIFD := bo.Uint32(data[nextPtrOff : nextPtrOff+4])
			if nextIFD != 0 {
				relNext := nextIFD - baseOld
				if relNext > 0 && relNext < dataLenU32 {
					ptrSet[int(nextPtrOff)] = struct{}{}
					collectIFD(relNext)
				}
			}
		}
	}

	collectIFD(uint32(ifdStart))

	// Pass 3: Canon TIFF footer (Canon only — avoids false positives on Sony/Panasonic)
	hasFooter := false
	if dataLen >= 8 && isCanonMake(em.IFD0) {
		footerOff := dataLen - 8
		fb := data[footerOff : footerOff+4]
		if (fb[0] == 'I' && fb[1] == 'I' && fb[2] == 0x2a && fb[3] == 0x00) ||
			(fb[0] == 'M' && fb[1] == 'M' && fb[2] == 0x00 && fb[3] == 0x2a) {
			hasFooter = true
			ptrSet[footerOff+4] = struct{}{}
		}
	}

	if len(ptrSet) == 0 {
		return
	}

	pointers := make([]int, 0, len(ptrSet))
	for p := range ptrSet {
		pointers = append(pointers, p)
	}
	sort.Ints(pointers)

	em.MakerNoteFixup = &MakerNoteFixup{
		BaseOld:   baseOld,
		BO:        bo,
		Pointers:  pointers,
		DataLen:   dataLen,
		HasFooter: hasFooter,
	}
}

func isCanonMake(ifd0 map[string]TagInfo) bool {
	if ti, ok := ifd0["Make"]; ok {
		return strings.HasPrefix(ti.Val, "Canon")
	}
	return false
}

// PatchMakerNoteOffsets patches absolute offset pointers in the output TIFF file.
// No-op if MakerNoteFixup is nil or has no pointers.
func (em *ExtractedMetadata) PatchMakerNoteOffsets(path string) error {
	if em.MakerNoteFixup == nil || len(em.MakerNoteFixup.Pointers) == 0 {
		return nil
	}
	fixup := em.MakerNoteFixup

	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("open for patching: %w", err)
	}
	defer f.Close()

	loc, err := findMakerNoteOffset(path)
	if err != nil {
		return fmt.Errorf("find MakerNote in output: %w", err)
	}

	baseNew := loc.offset
	shift := int64(baseNew) - int64(fixup.BaseOld)

	buf := make([]byte, 4)
	for _, ptrPos := range fixup.Pointers {
		fileOffset := int64(baseNew) + int64(ptrPos)
		if _, err := f.Seek(fileOffset, io.SeekStart); err != nil {
			return fmt.Errorf("seek pointer at %d: %w", ptrPos, err)
		}
		if _, err := io.ReadFull(f, buf); err != nil {
			return fmt.Errorf("read pointer at %d: %w", ptrPos, err)
		}
		currentValue := fixup.BO.Uint32(buf)
		newValue := uint32(int64(currentValue) + shift)
		fixup.BO.PutUint32(buf, newValue)
		if _, err := f.Seek(fileOffset, io.SeekStart); err != nil {
			return fmt.Errorf("seek back pointer at %d: %w", ptrPos, err)
		}
		if _, err := f.Write(buf); err != nil {
			return fmt.Errorf("write pointer at %d: %w", ptrPos, err)
		}
	}

	// Canon TIFF footer: last 4 bytes store the MakerNote data start position
	if fixup.HasFooter {
		footerPtrFileOff := int64(baseNew) + int64(fixup.DataLen) - 4
		fixup.BO.PutUint32(buf, baseNew)
		if _, err := f.Seek(footerPtrFileOff, io.SeekStart); err != nil {
			return fmt.Errorf("seek footer at %d: %w", footerPtrFileOff, err)
		}
		if _, err := f.Write(buf); err != nil {
			return fmt.Errorf("write footer: %w", err)
		}
	}

	if err := f.Sync(); err != nil {
		return fmt.Errorf("sync after patch: %w", err)
	}

	return nil
}
