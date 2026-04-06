#ifndef GOEXIV2_WRAPPER_H
#define GOEXIV2_WRAPPER_H

#include <stdint.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

// --- Lifecycle ---

// Opens an image by UTF-8 path. Returns opaque handle, or NULL on error.
void* goexiv2_open(const char* path);

// Closes and frees the image handle.
void goexiv2_close(void* img);

// --- Metadata reading ---

// Reads all metadata (EXIF + IPTC + XMP). Returns 0 on success.
int goexiv2_read_metadata(void* img);

// --- Error handling ---

// Returns the last thread-local error message, or NULL if none.
const char* goexiv2_get_last_error(void);

// --- EXIF ---

int   goexiv2_exif_count(void* img);
char* goexiv2_exif_get_key(void* img, int index);
int   goexiv2_exif_has_key(void* img, const char* key);
char* goexiv2_exif_get_string(void* img, const char* key);
int64_t goexiv2_exif_get_long(void* img, const char* key);
double  goexiv2_exif_get_double(void* img, const char* key);
const uint8_t* goexiv2_exif_get_bytes(void* img, const char* key, int* out_len);

// --- IPTC ---

int   goexiv2_iptc_count(void* img);
char* goexiv2_iptc_get_key(void* img, int index);
int   goexiv2_iptc_has_key(void* img, const char* key);
char* goexiv2_iptc_get_string(void* img, const char* key);

// --- XMP ---

int   goexiv2_xmp_count(void* img);
char* goexiv2_xmp_get_key(void* img, int index);
int   goexiv2_xmp_has_key(void* img, const char* key);
char* goexiv2_xmp_get_string(void* img, const char* key);

// Returns the raw XMP packet as a string. Caller frees with goexiv2_free.
char* goexiv2_xmp_packet(void* img);

// --- Utility ---

void        goexiv2_free(void* ptr);
const char* goexiv2_version(void);

#ifdef __cplusplus
}
#endif

#endif
