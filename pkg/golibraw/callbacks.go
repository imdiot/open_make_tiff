package golibraw

/*
#include <libraw/libraw.h>

extern void golibraw_data_error_trampoline(void* data, const char* file, long long offset);
extern void golibraw_exif_trampoline(void* context, int tag, int type, int len, unsigned int ord, void* ifp, long long base);
extern void golibraw_makernotes_trampoline(void* context, int tag, int type, int len, unsigned int ord, void* ifp, long long base);
*/
import "C"
import "unsafe"

// DataErrorHandler is called when a data error is encountered during processing.
type DataErrorHandler func(file string, offset int64)

// EXIFParseHandler is called for each EXIF tag during parsing.
type EXIFParseHandler func(tag, typ, len int, order uint, base int64)

// MakernotesParseHandler is called for each makernotes IFD during parsing.
type MakernotesParseHandler func(tag, typ, len int, order uint, base int64)

// callbackKey is the per-processor key used for callback dispatch.
type callbackKey = unsafe.Pointer

// SetDataErrorHandler sets a callback invoked on data errors during processing.
func (rp *RawProcessor) SetDataErrorHandler(handler DataErrorHandler) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if err := rp.ensureOpen(); err != nil {
		return err
	}

	key := rp.res.cbKey
	var entry *callbackEntry
	if v, ok := callbackRegistry.Load(key); ok {
		entry = v.(*callbackEntry)
		entry.dataError = handler
	} else {
		entry = &callbackEntry{dataError: handler}
		registerCallback(key, entry)
	}

	C.libraw_set_dataerror_handler(rp.res.handle,
		(*[0]byte)(C.golibraw_data_error_trampoline), unsafe.Pointer(key))
	return nil
}

// SetEXIFParseHandler sets a callback invoked for each EXIF tag during parsing.
func (rp *RawProcessor) SetEXIFParseHandler(handler EXIFParseHandler) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if err := rp.ensureOpen(); err != nil {
		return err
	}

	key := rp.res.cbKey
	var entry *callbackEntry
	if v, ok := callbackRegistry.Load(key); ok {
		entry = v.(*callbackEntry)
		entry.exif = handler
	} else {
		entry = &callbackEntry{exif: handler}
		registerCallback(key, entry)
	}

	C.libraw_set_exifparser_handler(rp.res.handle,
		(*[0]byte)(C.golibraw_exif_trampoline), unsafe.Pointer(key))
	return nil
}

// SetMakernotesParseHandler sets a callback invoked for each makernotes IFD during parsing.
func (rp *RawProcessor) SetMakernotesParseHandler(handler MakernotesParseHandler) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if err := rp.ensureOpen(); err != nil {
		return err
	}

	key := rp.res.cbKey
	var entry *callbackEntry
	if v, ok := callbackRegistry.Load(key); ok {
		entry = v.(*callbackEntry)
		entry.makernotes = handler
	} else {
		entry = &callbackEntry{makernotes: handler}
		registerCallback(key, entry)
	}

	C.libraw_set_makernotes_handler(rp.res.handle,
		(*[0]byte)(C.golibraw_makernotes_trampoline), unsafe.Pointer(key))
	return nil
}
