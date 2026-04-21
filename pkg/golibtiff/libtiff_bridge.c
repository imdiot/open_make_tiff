#include "libtiff_bridge.h"
#include <stdlib.h>
#include <string.h>
#include <stdarg.h>

// Thread-local fallback for the open phase (before TIFF handle exists).
static __thread char openPhaseErrMsg[1024] = {0};
static __thread int openPhaseHasErr = 0;

// Per-handle error handler. If tif is non-NULL, writes to clientinfo;
// otherwise falls back to thread-local for the open phase.
static int perHandleErrorHandler(TIFF *tif, void *user_data, const char *module, const char *fmt, va_list ap) {
    (void)user_data;
    (void)module;
    if (tif != NULL) {
        ErrorState *state = (ErrorState *)TIFFGetClientInfo(tif, "golibtiff_err");
        if (state) {
            vsnprintf(state->msg, sizeof(state->msg), fmt, ap);
            state->has_err = 1;
            return 0;
        }
    }
    // Fallback: open phase or clientinfo not yet attached.
    vsnprintf(openPhaseErrMsg, sizeof(openPhaseErrMsg), fmt, ap);
    openPhaseHasErr = 1;
    return 0;
}

int getPerHandleErrorHandler(TIFFErrorHandlerExtR *out) {
    *out = perHandleErrorHandler;
    return 1;
}

static ErrorState *errorStateNew(void) {
    ErrorState *s = (ErrorState *)malloc(sizeof(ErrorState));
    if (s) { s->msg[0] = '\0'; s->has_err = 0; }
    return s;
}

void attachErrorState(TIFF *tif) {
    ErrorState *s = errorStateNew();
    if (s) TIFFSetClientInfo(tif, s, "golibtiff_err");
}

void detachErrorState(TIFF *tif) {
    ErrorState *s = (ErrorState *)TIFFGetClientInfo(tif, "golibtiff_err");
    if (s) {
        free(s);
        TIFFSetClientInfo(tif, NULL, "golibtiff_err");
    }
}

void clearHandleError(TIFF *tif) {
    ErrorState *state = (ErrorState *)TIFFGetClientInfo(tif, "golibtiff_err");
    if (state) { state->has_err = 0; state->msg[0] = '\0'; }
    openPhaseHasErr = 0;
    openPhaseErrMsg[0] = '\0';
}

int hasHandleError(TIFF *tif) {
    ErrorState *state = (ErrorState *)TIFFGetClientInfo(tif, "golibtiff_err");
    if (state && state->has_err) return 1;
    return openPhaseHasErr;
}

const char *getHandleError(TIFF *tif) {
    ErrorState *state = (ErrorState *)TIFFGetClientInfo(tif, "golibtiff_err");
    if (state && state->has_err) return state->msg;
    if (openPhaseHasErr) return openPhaseErrMsg;
    return "";
}

void clearOpenPhaseError(void) { openPhaseHasErr = 0; openPhaseErrMsg[0] = '\0'; }
int hasOpenPhaseError(void) { return openPhaseHasErr; }
const char *getOpenPhaseError(void) { return openPhaseErrMsg; }

// Typed getters (avoid variadic TIFFGetField from Go).
int tiffGetFieldU16(TIFF *t, uint32_t tag, uint16_t *v) { return TIFFGetField(t, tag, v); }
int tiffGetFieldU32(TIFF *t, uint32_t tag, uint32_t *v) { return TIFFGetField(t, tag, v); }
int tiffGetFieldFloat(TIFF *t, uint32_t tag, float *v) { return TIFFGetField(t, tag, v); }
int tiffGetFieldDouble(TIFF *t, uint32_t tag, double *v) { return TIFFGetField(t, tag, v); }
int tiffGetFieldString(TIFF *t, uint32_t tag, const char **v) { return TIFFGetField(t, tag, v); }
int tiffGetFieldU16Array(TIFF *t, uint32_t tag, uint16_t **v, uint16_t *c) { return TIFFGetField(t, tag, c, v); }
int tiffGetFieldU32Array(TIFF *t, uint32_t tag, uint32_t **v, uint32_t *c) { return TIFFGetField(t, tag, c, v); }
int tiffGetFieldU8(TIFF *t, uint32_t tag, uint8_t *v) { return TIFFGetField(t, tag, v); }
int tiffGetFieldU64(TIFF *t, uint32_t tag, uint64_t *v) { return TIFFGetField(t, tag, v); }
int tiffGetFieldS8(TIFF *t, uint32_t tag, int8_t *v) { return TIFFGetField(t, tag, v); }
int tiffGetFieldS16(TIFF *t, uint32_t tag, int16_t *v) { return TIFFGetField(t, tag, v); }
int tiffGetFieldS32(TIFF *t, uint32_t tag, int32_t *v) { return TIFFGetField(t, tag, v); }
int tiffGetFieldS64(TIFF *t, uint32_t tag, int64_t *v) { return TIFFGetField(t, tag, v); }
int tiffReadEXIFDirectory(TIFF *t, uint64_t off) { return TIFFReadEXIFDirectory(t, (toff_t)off); }

// Typed setters (avoid variadic TIFFSetField from Go).
int tiffSetFieldU16(TIFF *t, uint32_t tag, uint16_t v) { return TIFFSetField(t, tag, v); }
int tiffSetFieldU32(TIFF *t, uint32_t tag, uint32_t v) { return TIFFSetField(t, tag, v); }
int tiffSetFieldFloat(TIFF *t, uint32_t tag, float v) { return TIFFSetField(t, tag, v); }
int tiffSetFieldString(TIFF *t, uint32_t tag, const char *v) { return TIFFSetField(t, tag, v); }
int tiffSetFieldU16Array(TIFF *t, uint32_t tag, uint16_t c, uint16_t *v) { return TIFFSetField(t, tag, c, v); }
int tiffSetFieldU32Array(TIFF *t, uint32_t tag, uint32_t c, uint32_t *v) { return TIFFSetField(t, tag, c, v); }

int tiffSetFieldS8(TIFF *t, uint32_t tag, int8_t v) { return TIFFSetField(t, tag, v); }
int tiffSetFieldS16(TIFF *t, uint32_t tag, int16_t v) { return TIFFSetField(t, tag, v); }
int tiffSetFieldS32(TIFF *t, uint32_t tag, int32_t v) { return TIFFSetField(t, tag, v); }
int tiffSetFieldS64(TIFF *t, uint32_t tag, int64_t v) { return TIFFSetField(t, tag, v); }

int tiffReadRGBAImage(TIFF *t, uint32_t w, uint32_t h, uint32_t *buf) {
    return TIFFReadRGBAImage(t, w, h, buf, 0);
}
tmsize_t tiffReadEncodedTile(TIFF *t, uint32_t tile, void *buf, tmsize_t size) {
    return TIFFReadEncodedTile(t, tile, buf, size);
}
tmsize_t tiffWriteEncodedTile(TIFF *t, uint32_t tile, void *buf, tmsize_t size) {
    return TIFFWriteEncodedTile(t, tile, buf, size);
}

int tiffSetFieldByteSlice(TIFF *t, uint32_t tag, uint32_t c, uint8_t *v) {
    return TIFFSetField(t, tag, c, v);
}
int tiffSetFieldC0ByteSlice(TIFF *t, uint32_t tag, uint8_t *v) {
    return TIFFSetField(t, tag, v);
}
int tiffSetFieldC0U16(TIFF *t, uint32_t tag, uint16_t *v) {
    return TIFFSetField(t, tag, v);
}
int tiffSetFieldC0U32(TIFF *t, uint32_t tag, uint32_t *v) {
    return TIFFSetField(t, tag, v);
}
int tiffCreateEXIFDirectory(TIFF *t) {
    return TIFFCreateEXIFDirectory(t);
}
int tiffWriteCustomDirectory(TIFF *t, uint64_t *offset) {
    return TIFFWriteCustomDirectory(t, offset);
}
int tiffSetFieldFloatSlice(TIFF *t, uint32_t tag, int c, float *v) {
    return TIFFSetField(t, tag, c, v);
}
int tiffSetFieldU64(TIFF *t, uint32_t tag, uint64_t v) {
    return TIFFSetField(t, tag, v);
}
int tiffCheckpointDirectory(TIFF *t) {
    return TIFFCheckpointDirectory(t);
}
int tiffSetFieldU8(TIFF *t, uint32_t tag, uint8_t v) {
    return TIFFSetField(t, tag, v);
}
int tiffSetFieldC0Float(TIFF *t, uint32_t tag, float *v) {
    return TIFFSetField(t, tag, v);
}
int tiffSetFieldDouble(TIFF *t, uint32_t tag, double v) {
    return TIFFSetField(t, tag, v);
}
int tiffSetFieldDoubleSlice(TIFF *t, uint32_t tag, int c, double *v) {
    return TIFFSetField(t, tag, c, v);
}
int tiffSetFieldC0Double(TIFF *t, uint32_t tag, double *v) {
    return TIFFSetField(t, tag, v);
}
int tiffCreateGPSDirectory(TIFF *t) {
    return TIFFCreateGPSDirectory(t);
}
int tiffIsFieldKnown(TIFF *t, uint32_t tag) {
    return TIFFFieldWithTag(t, tag) != NULL;
}
int tiffGetFieldType(TIFF *t, uint32_t tag) {
    const TIFFField *f = TIFFFieldWithTag(t, tag);
    return f ? (int)TIFFFieldDataType(f) : -1;
}
int tiffFieldPassCount(TIFF *t, uint32_t tag) {
    const TIFFField *f = TIFFFieldWithTag(t, tag);
    return f ? (int)TIFFFieldPassCount(f) : -1;
}
int tiffFieldWriteCount(TIFF *t, uint32_t tag) {
    const TIFFField *f = TIFFFieldWithTag(t, tag);
    return f ? (int)TIFFFieldWriteCount(f) : 0;
}
int tiffFieldSetGetSize(TIFF *t, uint32_t tag) {
    const TIFFField *f = TIFFFieldWithTag(t, tag);
    return f ? TIFFFieldSetGetSize(f) : -1;
}
int tiffGetFieldByteSlice(TIFF *t, uint32_t tag, uint8_t **v, uint32_t *c) {
    return TIFFGetField(t, tag, c, v);
}
int tiffUnsetField(TIFF *t, uint32_t tag) { return TIFFUnsetField(t, tag); }
int tiffReadRGBAStrip(TIFF *t, uint32_t strip, uint32_t *buf) {
    return TIFFReadRGBAStrip(t, strip, buf);
}
int tiffReadRGBATile(TIFF *t, uint32_t tile, uint32_t *buf) {
    return TIFFReadRGBATile(t, tile, 0, buf);
}
int tiffReadGPSDirectory(TIFF *t, uint64_t off) {
    return TIFFReadGPSDirectory(t, (toff_t)off);
}
int tiffCreateDirectory(TIFF *t) { return TIFFCreateDirectory(t); }
int tiffRewriteDirectory(TIFF *t) { return TIFFRewriteDirectory(t); }
int tiffUnlinkDirectory(TIFF *t, uint16_t d) {
    return TIFFUnlinkDirectory(t, (tdir_t)d);
}
int tiffGetFieldDefaultedU16(TIFF *t, uint32_t tag, uint16_t *v) {
    return TIFFGetFieldDefaulted(t, tag, v);
}
int tiffGetFieldDefaultedU32(TIFF *t, uint32_t tag, uint32_t *v) {
    return TIFFGetFieldDefaulted(t, tag, v);
}
int tiffGetFieldDefaultedFloat(TIFF *t, uint32_t tag, float *v) {
    return TIFFGetFieldDefaulted(t, tag, v);
}
int tiffGetFieldDefaultedString(TIFF *t, uint32_t tag, const char **v) {
    return TIFFGetFieldDefaulted(t, tag, v);
}
uint64_t tiffGetStrileOffset(TIFF *t, uint32_t s) {
    return TIFFGetStrileOffset(t, s);
}
uint64_t tiffGetStrileByteCount(TIFF *t, uint32_t s) {
    return TIFFGetStrileByteCount(t, s);
}
uint64_t tiffGetStrileOffsetWithErr(TIFF *t, uint32_t s, int *e) {
    return TIFFGetStrileOffsetWithErr(t, s, e);
}
uint64_t tiffGetStrileByteCountWithErr(TIFF *t, uint32_t s, int *e) {
    return TIFFGetStrileByteCountWithErr(t, s, e);
}
int tiffReadFromUserBuffer(TIFF *t, uint32_t strile, void *in, tmsize_t insz, void *out, tmsize_t outsz) {
    return TIFFReadFromUserBuffer(t, strile, in, insz, out, outsz);
}
int tiffReadRGBAImageOriented(TIFF *t, uint32_t w, uint32_t h, uint32_t *buf, int orient, int stop) {
    return TIFFReadRGBAImageOriented(t, w, h, buf, orient, stop);
}
int tiffReadRGBAStripExt(TIFF *t, uint32_t strip, uint32_t *buf, int stop) {
    return TIFFReadRGBAStripExt(t, strip, buf, stop);
}
int tiffReadRGBATileExt(TIFF *t, uint32_t tw, uint32_t th, uint32_t *buf, int stop) {
    return TIFFReadRGBATileExt(t, tw, th, buf, stop);
}
uint64_t tiffCurrentDirOffset(TIFF *t) { return TIFFCurrentDirOffset(t); }
void tiffDefaultTileSize(TIFF *t, uint32_t *tw, uint32_t *th) { TIFFDefaultTileSize(t, tw, th); }
