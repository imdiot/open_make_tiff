#ifndef GOLIBTIFF_BRIDGE_H
#define GOLIBTIFF_BRIDGE_H

#include <tiffio.h>
#include <stdint.h>

// Per-handle error state, stored via TIFFSetClientInfo/GetClientInfo.
typedef struct {
    char msg[1024];
    int has_err;
} ErrorState;

void attachErrorState(TIFF *tif);
void detachErrorState(TIFF *tif);
void clearHandleError(TIFF *tif);
int hasHandleError(TIFF *tif);
const char *getHandleError(TIFF *tif);

void clearOpenPhaseError(void);
int hasOpenPhaseError(void);
const char *getOpenPhaseError(void);

int getPerHandleErrorHandler(TIFFErrorHandlerExtR *out);

// Typed getters (avoid variadic TIFFGetField from Go).
int tiffGetFieldU16(TIFF *t, uint32_t tag, uint16_t *v);
int tiffGetFieldU32(TIFF *t, uint32_t tag, uint32_t *v);
int tiffGetFieldFloat(TIFF *t, uint32_t tag, float *v);
int tiffGetFieldDouble(TIFF *t, uint32_t tag, double *v);
int tiffGetFieldString(TIFF *t, uint32_t tag, const char **v);
int tiffGetFieldU16Array(TIFF *t, uint32_t tag, uint16_t **v, uint16_t *c);
int tiffGetFieldU32Array(TIFF *t, uint32_t tag, uint32_t **v, uint32_t *c);
int tiffGetFieldU8(TIFF *t, uint32_t tag, uint8_t *v);
int tiffGetFieldU64(TIFF *t, uint32_t tag, uint64_t *v);
int tiffGetFieldS8(TIFF *t, uint32_t tag, int8_t *v);
int tiffGetFieldS16(TIFF *t, uint32_t tag, int16_t *v);
int tiffGetFieldS32(TIFF *t, uint32_t tag, int32_t *v);
int tiffGetFieldS64(TIFF *t, uint32_t tag, int64_t *v);
int tiffReadEXIFDirectory(TIFF *t, uint64_t off);

// Typed setters (avoid variadic TIFFSetField from Go).
int tiffSetFieldU16(TIFF *t, uint32_t tag, uint16_t v);
int tiffSetFieldU32(TIFF *t, uint32_t tag, uint32_t v);
int tiffSetFieldFloat(TIFF *t, uint32_t tag, float v);
int tiffSetFieldString(TIFF *t, uint32_t tag, const char *v);
int tiffSetFieldU16Array(TIFF *t, uint32_t tag, uint16_t c, uint16_t *v);
int tiffSetFieldU32Array(TIFF *t, uint32_t tag, uint32_t c, uint32_t *v);

int tiffReadRGBAImage(TIFF *t, uint32_t w, uint32_t h, uint32_t *buf);
tmsize_t tiffReadEncodedTile(TIFF *t, uint32_t tile, void *buf, tmsize_t size);
tmsize_t tiffWriteEncodedTile(TIFF *t, uint32_t tile, void *buf, tmsize_t size);

int tiffSetFieldByteSlice(TIFF *t, uint32_t tag, uint32_t c, uint8_t *v);
int tiffSetFieldC0ByteSlice(TIFF *t, uint32_t tag, uint8_t *v);
int tiffSetFieldC0U16(TIFF *t, uint32_t tag, uint16_t *v);
int tiffSetFieldC0U32(TIFF *t, uint32_t tag, uint32_t *v);
int tiffCreateEXIFDirectory(TIFF *t);
int tiffWriteCustomDirectory(TIFF *t, uint64_t *offset);
int tiffSetFieldFloatSlice(TIFF *t, uint32_t tag, int c, float *v);
int tiffSetFieldU64(TIFF *t, uint32_t tag, uint64_t v);
int tiffCheckpointDirectory(TIFF *t);
int tiffSetFieldU8(TIFF *t, uint32_t tag, uint8_t v);
int tiffSetFieldS8(TIFF *t, uint32_t tag, int8_t v);
int tiffSetFieldS16(TIFF *t, uint32_t tag, int16_t v);
int tiffSetFieldS32(TIFF *t, uint32_t tag, int32_t v);
int tiffSetFieldS64(TIFF *t, uint32_t tag, int64_t v);
int tiffSetFieldC0Float(TIFF *t, uint32_t tag, float *v);
int tiffSetFieldDouble(TIFF *t, uint32_t tag, double v);
int tiffSetFieldDoubleSlice(TIFF *t, uint32_t tag, int c, double *v);
int tiffSetFieldC0Double(TIFF *t, uint32_t tag, double *v);
int tiffCreateGPSDirectory(TIFF *t);
int tiffIsFieldKnown(TIFF *t, uint32_t tag);
int tiffGetFieldType(TIFF *t, uint32_t tag);
int tiffFieldPassCount(TIFF *t, uint32_t tag);
int tiffFieldWriteCount(TIFF *t, uint32_t tag);
int tiffFieldSetGetSize(TIFF *t, uint32_t tag);
int tiffGetFieldByteSlice(TIFF *t, uint32_t tag, uint8_t **v, uint32_t *c);
int tiffUnsetField(TIFF *t, uint32_t tag);
int tiffReadRGBAStrip(TIFF *t, uint32_t strip, uint32_t *buf);
int tiffReadRGBATile(TIFF *t, uint32_t tile, uint32_t *buf);
int tiffReadGPSDirectory(TIFF *t, uint64_t off);
int tiffCreateDirectory(TIFF *t);
int tiffRewriteDirectory(TIFF *t);
int tiffUnlinkDirectory(TIFF *t, uint16_t d);
int tiffGetFieldDefaultedU16(TIFF *t, uint32_t tag, uint16_t *v);
int tiffGetFieldDefaultedU32(TIFF *t, uint32_t tag, uint32_t *v);
int tiffGetFieldDefaultedFloat(TIFF *t, uint32_t tag, float *v);
int tiffGetFieldDefaultedString(TIFF *t, uint32_t tag, const char **v);
uint64_t tiffGetStrileOffset(TIFF *t, uint32_t s);
uint64_t tiffGetStrileByteCount(TIFF *t, uint32_t s);
uint64_t tiffGetStrileOffsetWithErr(TIFF *t, uint32_t s, int *e);
uint64_t tiffGetStrileByteCountWithErr(TIFF *t, uint32_t s, int *e);
int tiffReadFromUserBuffer(TIFF *t, uint32_t strile, void *in, tmsize_t insz, void *out, tmsize_t outsz);
int tiffReadRGBAImageOriented(TIFF *t, uint32_t w, uint32_t h, uint32_t *buf, int orient, int stop);
int tiffReadRGBAStripExt(TIFF *t, uint32_t strip, uint32_t *buf, int stop);
int tiffReadRGBATileExt(TIFF *t, uint32_t tw, uint32_t th, uint32_t *buf, int stop);
uint64_t tiffCurrentDirOffset(TIFF *t);
void tiffDefaultTileSize(TIFF *t, uint32_t *tw, uint32_t *th);

#endif
