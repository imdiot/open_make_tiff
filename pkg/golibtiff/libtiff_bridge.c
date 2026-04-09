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
