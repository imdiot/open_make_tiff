#ifndef GOEXIV2_WRAPPER_H
#define GOEXIV2_WRAPPER_H

#include <stdint.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

// --- Tag entry ---

typedef struct {
    uint16_t tag;        // tag() — EXIF: IFD tag, IPTC: dataset, XMP: 0
    int      type_id;    // typeId() — value type enum
    int      count;      // count() — number of value components
    int      size;       // size() — byte length
    char*    key;        // key() — standard identifier (e.g. "Exif.Image.Make")
    char*    value;      // toString() — string representation
    uint8_t* raw;        // raw bytes (binary types), or NULL
    int      raw_len;
    int      ifd_id;     // EXIF only: ifdId(), -1 for others
    int      record;     // IPTC only: record(), -1 for others
} GoExiv2Tag;

// --- Tag array (tags + count paired) ---

typedef struct {
    GoExiv2Tag* tags;
    int         count;
} GoExiv2TagArray;

// --- Complete metadata dump ---

typedef struct {
    GoExiv2TagArray exif;
    GoExiv2TagArray iptc;
    GoExiv2TagArray xmp;
    char*           xmp_packet; // NULL when no XMP packet
} GoExiv2Metadata;

// --- Lifecycle ---

void*              goexiv2_open(const char* path);
void               goexiv2_close(void* img);

// --- Metadata reading ---

GoExiv2Metadata*   goexiv2_read_metadata(void* img);
void               goexiv2_metadata_free(GoExiv2Metadata* md);

// --- Error handling ---

const char*        goexiv2_get_last_error(void);

// --- Utility ---

const char*        goexiv2_version(void);

#ifdef __cplusplus
}
#endif

#endif
