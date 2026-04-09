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

#endif
