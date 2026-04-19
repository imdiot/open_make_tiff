package golibraw

/*
#include <libraw/libraw.h>

extern int golibraw_is_fuji_rotated(libraw_data_t* lr);
extern int golibraw_is_sraw(libraw_data_t* lr);
extern int golibraw_sraw_midpoint(libraw_data_t* lr);
extern int golibraw_is_nikon_sraw(libraw_data_t* lr);
extern int golibraw_is_coolscan_nef(libraw_data_t* lr);
extern int golibraw_is_jpeg_thumb(libraw_data_t* lr);
extern int golibraw_is_floating_point(libraw_data_t* lr);
extern int golibraw_have_fpdata(libraw_data_t* lr);
extern int golibraw_error_count(libraw_data_t* lr);
extern int golibraw_thumb_ok(libraw_data_t* lr, long long maxsz);
extern int golibraw_raw_was_read(libraw_data_t* lr);
extern int golibraw_color(libraw_data_t* lr, int row, int col);
extern int golibraw_fc(libraw_data_t* lr, int row, int col);
extern int golibraw_fcol(libraw_data_t* lr, int row, int col);
*/
import "C"

// IsFujiRotated returns whether the image needs Fuji rotation.
func (rp *RawProcessor) IsFujiRotated() bool {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if !rp.isOpen() {
		return false
	}
	return C.golibraw_is_fuji_rotated(rp.res.handle) != 0
}

// IsSRAW returns whether the image is a half-resolution sRAW.
func (rp *RawProcessor) IsSRAW() bool {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if !rp.isOpen() {
		return false
	}
	return C.golibraw_is_sraw(rp.res.handle) != 0
}

// SRAWMidpoint returns the sRAW midpoint value.
func (rp *RawProcessor) SRAWMidpoint() int {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if !rp.isOpen() {
		return 0
	}
	return int(C.golibraw_sraw_midpoint(rp.res.handle))
}

// IsNikonSRAW returns whether the image is a Nikon sRAW.
func (rp *RawProcessor) IsNikonSRAW() bool {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if !rp.isOpen() {
		return false
	}
	return C.golibraw_is_nikon_sraw(rp.res.handle) != 0
}

// IsCoolscanNEF returns whether the image is a CoolScan scanner NEF.
func (rp *RawProcessor) IsCoolscanNEF() bool {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if !rp.isOpen() {
		return false
	}
	return C.golibraw_is_coolscan_nef(rp.res.handle) != 0
}

// IsJPEGThumb returns whether the thumbnail is JPEG format.
func (rp *RawProcessor) IsJPEGThumb() bool {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if !rp.isOpen() {
		return false
	}
	return C.golibraw_is_jpeg_thumb(rp.res.handle) != 0
}

// IsFloatingPoint returns whether the image is floating-point RAW.
func (rp *RawProcessor) IsFloatingPoint() bool {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if !rp.isOpen() {
		return false
	}
	return C.golibraw_is_floating_point(rp.res.handle) != 0
}

// HaveFPData returns whether floating-point data is available.
func (rp *RawProcessor) HaveFPData() bool {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if !rp.isOpen() {
		return false
	}
	return C.golibraw_have_fpdata(rp.res.handle) != 0
}

// ErrorCount returns the data error count.
func (rp *RawProcessor) ErrorCount() int {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if !rp.isOpen() {
		return 0
	}
	return int(C.golibraw_error_count(rp.res.handle))
}

// ThumbOK returns whether the thumbnail is valid.
// maxSize limits maximum thumbnail size (-1 = no limit).
func (rp *RawProcessor) ThumbOK(maxSize int64) bool {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if !rp.isOpen() {
		return false
	}
	return C.golibraw_thumb_ok(rp.res.handle, C.longlong(maxSize)) != 0
}

// RawWasRead returns whether RAW data has been read.
func (rp *RawProcessor) RawWasRead() bool {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if !rp.isOpen() {
		return false
	}
	return C.golibraw_raw_was_read(rp.res.handle) != 0
}

// Color returns the color channel at (row, col), handling Bayer/X-Trans/Fuji layouts.
func (rp *RawProcessor) Color(row, col int) int {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if !rp.isOpen() {
		return 0
	}
	return int(C.golibraw_color(rp.res.handle, C.int(row), C.int(col)))
}

// FC returns the fast Bayer color channel query at (row, col).
func (rp *RawProcessor) FC(row, col int) int {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if !rp.isOpen() {
		return 0
	}
	return int(C.golibraw_fc(rp.res.handle, C.int(row), C.int(col)))
}

// FCol returns the color channel at (row, col), supporting X-Trans CFA.
func (rp *RawProcessor) FCol(row, col int) int {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if !rp.isOpen() {
		return 0
	}
	return int(C.golibraw_fcol(rp.res.handle, C.int(row), C.int(col)))
}

// ProgressFlags returns the processing progress flags bitmask.
func (rp *RawProcessor) ProgressFlags() uint {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if !rp.isOpen() {
		return 0
	}
	return uint(rp.res.handle.progress_flags)
}

// ProcessWarnings returns the processing warnings bitmask.
func (rp *RawProcessor) ProcessWarnings() uint {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if !rp.isOpen() {
		return 0
	}
	return uint(rp.res.handle.process_warnings)
}
