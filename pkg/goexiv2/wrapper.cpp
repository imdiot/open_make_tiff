// goexiv2 C bridge: wraps exiv2 C++ API behind C ABI for CGo.
// Only exposes lifecycle + one-shot metadata dump.

#include "wrapper.h"

#include <exiv2/exiv2.hpp>

#include <cstdlib>
#include <cstring>
#include <string>

// --- Thread-local error storage ---

static __thread char tl_error[1024] = {0};

static void set_error(const char* msg) {
    if (msg) {
        snprintf(tl_error, sizeof(tl_error), "%s", msg);
    } else {
        tl_error[0] = '\0';
    }
}

// --- Internal image handle ---

struct GoExiv2Image {
    Exiv2::Image::UniquePtr image;
};

// --- Helpers ---

static bool is_binary_type(Exiv2::TypeId tid) {
    return tid == Exiv2::undefined
        || tid == Exiv2::unsignedByte
        || tid == Exiv2::signedByte;
}

// Dump a metadata container into a GoExiv2TagArray.
template<typename Container>
static void dump_tags(Container& c, GoExiv2TagArray* out) {
    out->tags = nullptr;
    out->count = 0;

    int n = 0;
    for (auto it = c.begin(); it != c.end(); ++it) {
        try { if (!it->key().empty()) n++; } catch (...) {}
    }
    if (n == 0) return;

    out->tags = (GoExiv2Tag*)calloc(n, sizeof(GoExiv2Tag));
    if (!out->tags) {
        set_error("out of memory");
        return;
    }
    out->count = n;

    int i = 0;
    for (auto it = c.begin(); it != c.end(); ++it) {
        try {
            auto k = it->key();
            if (k.empty()) continue;
            out->tags[i].tag     = static_cast<uint16_t>(it->tag());
            out->tags[i].type_id = static_cast<int>(it->typeId());
            out->tags[i].count   = static_cast<int>(it->count());
            out->tags[i].size    = static_cast<int>(it->size());
            out->tags[i].key     = strdup(k.c_str());
            out->tags[i].value   = strdup(it->toString().c_str());
            out->tags[i].ifd_id  = -1;
            out->tags[i].record  = -1;

            if (is_binary_type(it->typeId())) {
                auto& val = it->value();
                auto sz = val.size();
                if (sz > 0) {
                    out->tags[i].raw = (uint8_t*)malloc(sz);
                    if (out->tags[i].raw) {
                        val.copy(out->tags[i].raw, Exiv2::invalidByteOrder);
                        out->tags[i].raw_len = static_cast<int>(sz);
                    }
                }
            }
            i++;
        } catch (...) {}
    }
}

static void free_tag_array(GoExiv2TagArray* arr) {
    if (!arr->tags) return;
    for (int i = 0; i < arr->count; i++) {
        free(arr->tags[i].key);
        free(arr->tags[i].value);
        free(arr->tags[i].raw);
    }
    free(arr->tags);
    arr->tags = nullptr;
    arr->count = 0;
}

// --- Lifecycle ---

void* goexiv2_open(const char* path) {
    tl_error[0] = '\0';
    if (!path) {
        set_error("path is NULL");
        return nullptr;
    }
    try {
        auto img = Exiv2::ImageFactory::open(path);
        if (!img) {
            set_error("unknown image type");
            return nullptr;
        }
        auto* g = new GoExiv2Image();
        g->image = std::move(img);
        return g;
    } catch (Exiv2::Error& e) {
        set_error(e.what());
        return nullptr;
    } catch (std::exception& e) {
        set_error(e.what());
        return nullptr;
    }
}

void goexiv2_close(void* p) {
    auto* g = static_cast<GoExiv2Image*>(p);
    delete g;
}

// --- Metadata reading ---

GoExiv2Metadata* goexiv2_read_metadata(void* p) {
    tl_error[0] = '\0';
    auto* g = static_cast<GoExiv2Image*>(p);
    if (!g || !g->image) {
        set_error("invalid handle");
        return nullptr;
    }
    try {
        g->image->readMetadata();
    } catch (Exiv2::Error& e) {
        set_error(e.what());
        return nullptr;
    } catch (std::exception& e) {
        set_error(e.what());
        return nullptr;
    }

    auto* md = (GoExiv2Metadata*)calloc(1, sizeof(GoExiv2Metadata));
    if (!md) {
        set_error("out of memory");
        return nullptr;
    }

    try { dump_tags(g->image->exifData(), &md->exif); } catch (...) {}
    try { dump_tags(g->image->iptcData(), &md->iptc); } catch (...) {}
    try { dump_tags(g->image->xmpData(),  &md->xmp);  } catch (...) {}

    // Fill family-specific raw fields that the template can't access
    try {
        int i = 0;
        for (auto it = g->image->exifData().begin(); it != g->image->exifData().end(); ++it) {
            try { if (!it->key().empty()) {
                md->exif.tags[i].ifd_id = static_cast<int>(((Exiv2::Exifdatum*)&*it)->ifdId());
                i++;
            }} catch (...) { if (!it->key().empty()) i++; }
        }
    } catch (...) {}
    try {
        int i = 0;
        for (auto it = g->image->iptcData().begin(); it != g->image->iptcData().end(); ++it) {
            try { if (!it->key().empty()) {
                md->iptc.tags[i].record = static_cast<int>(((Exiv2::Iptcdatum*)&*it)->record());
                i++;
            }} catch (...) { if (!it->key().empty()) i++; }
        }
    } catch (...) {}

    try {
        auto& packet = g->image->xmpPacket();
        if (!packet.empty()) {
            md->xmp_packet = strdup(packet.c_str());
        }
    } catch (...) {}

    return md;
}

void goexiv2_metadata_free(GoExiv2Metadata* md) {
    if (!md) return;
    free_tag_array(&md->exif);
    free_tag_array(&md->iptc);
    free_tag_array(&md->xmp);
    free(md->xmp_packet);
    free(md);
}

// --- Error handling ---

const char* goexiv2_get_last_error(void) {
    return tl_error[0] ? tl_error : nullptr;
}

// --- Utility ---

const char* goexiv2_version(void) {
    return Exiv2::version();
}
