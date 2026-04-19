package golibraw

/*
// Callback trampoline functions exported to C.
// Each trampoline uses a sync.Map registry keyed by the processor pointer
// (passed through LibRaw's datap parameter) to dispatch to per-instance callbacks.
*/
import "C"
import (
	"sync"
	"unsafe"
)

// callbackEntry holds per-processor callback handlers.
type callbackEntry struct {
	dataError  DataErrorHandler
	exif       EXIFParseHandler
	makernotes MakernotesParseHandler
}

// callbackRegistry maps unsafe.Pointer(datap) → *callbackEntry.
var callbackRegistry sync.Map

// registerCallback stores callbacks for a given key and returns the key.
func registerCallback(key unsafe.Pointer, entry *callbackEntry) {
	callbackRegistry.Store(key, entry)
}

// unregisterCallback removes callbacks for a given key.
func unregisterCallback(key unsafe.Pointer) {
	callbackRegistry.Delete(key)
}

//export golibraw_data_error_trampoline
//
// golibraw_data_error_trampoline is called by LibRaw on data errors.
func golibraw_data_error_trampoline(data unsafe.Pointer, file *C.char, offset C.longlong) {
	if data == nil {
		return
	}
	v, ok := callbackRegistry.Load(data)
	if !ok {
		return
	}
	entry := v.(*callbackEntry)
	if entry.dataError != nil {
		f := ""
		if file != nil {
			f = cGoString(file)
		}
		entry.dataError(f, int64(offset))
	}
}

//export golibraw_exif_trampoline
//
// golibraw_exif_trampoline is called by LibRaw for each EXIF tag.
func golibraw_exif_trampoline(context unsafe.Pointer, tag C.int, typ C.int, length C.int, order C.uint, ifp unsafe.Pointer, base C.longlong) {
	if context == nil {
		return
	}
	v, ok := callbackRegistry.Load(context)
	if !ok {
		return
	}
	entry := v.(*callbackEntry)
	if entry.exif != nil {
		entry.exif(int(tag), int(typ), int(length), uint(order), int64(base))
	}
}

//export golibraw_makernotes_trampoline
//
// golibraw_makernotes_trampoline is called by LibRaw for each makernotes IFD.
func golibraw_makernotes_trampoline(context unsafe.Pointer, tag C.int, typ C.int, length C.int, order C.uint, ifp unsafe.Pointer, base C.longlong) {
	if context == nil {
		return
	}
	v, ok := callbackRegistry.Load(context)
	if !ok {
		return
	}
	entry := v.(*callbackEntry)
	if entry.makernotes != nil {
		entry.makernotes(int(tag), int(typ), int(length), uint(order), int64(base))
	}
}
