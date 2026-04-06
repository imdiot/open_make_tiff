// goexiv2 C bridge: wraps exiv2 C++ API behind C ABI for CGo.

#include "wrapper.h"

#include <exiv2/exiv2.hpp>

#include <cstring>
#include <string>
#include <vector>

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
    bool metadataRead;
    std::vector<uint8_t> bytesCache;

    // Key caches: populated once during readMetadata.
    std::vector<std::string> exifKeys;
    std::vector<std::string> iptcKeys;
    std::vector<std::string> xmpKeys;

    GoExiv2Image() : metadataRead(false) {}
};

// Populate a key cache by iterating [begin,end).
template<typename Container>
static void cache_keys(Container& c, std::vector<std::string>& out) {
    out.clear();
    for (auto it = c.begin(); it != c.end(); ++it) {
        try {
            std::string k = it->key();
            if (!k.empty()) {
                out.push_back(k);
            }
        } catch (...) {
            // Skip entries that throw C++ exceptions.
        }
    }
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
    if (g) {
        delete g;
    }
}

// --- Metadata reading ---

int goexiv2_read_metadata(void* p) {
    tl_error[0] = '\0';
    auto* g = static_cast<GoExiv2Image*>(p);
    if (!g || !g->image) {
        set_error("invalid handle");
        return -1;
    }
    try {
        g->image->readMetadata();
        g->metadataRead = true;

        // Pre-cache all keys once.
        cache_keys(g->image->exifData(), g->exifKeys);
        cache_keys(g->image->iptcData(), g->iptcKeys);
        cache_keys(g->image->xmpData(),  g->xmpKeys);

        return 0;
    } catch (Exiv2::Error& e) {
        set_error(e.what());
        return -1;
    } catch (std::exception& e) {
        set_error(e.what());
        return -1;
    }
}

// --- Error handling ---

const char* goexiv2_get_last_error(void) {
    return tl_error[0] ? tl_error : nullptr;
}

// --- EXIF ---

int goexiv2_exif_count(void* p) {
    auto* g = static_cast<GoExiv2Image*>(p);
    if (!g || !g->metadataRead) return 0;
    return static_cast<int>(g->exifKeys.size());
}

char* goexiv2_exif_get_key(void* p, int index) {
    auto* g = static_cast<GoExiv2Image*>(p);
    if (!g || !g->metadataRead) return nullptr;
    if (index < 0 || index >= static_cast<int>(g->exifKeys.size())) return nullptr;
    return strdup(g->exifKeys[index].c_str());
}

int goexiv2_exif_has_key(void* p, const char* key) {
    auto* g = static_cast<GoExiv2Image*>(p);
    if (!g || !g->metadataRead || !key) return 0;
    try {
        auto& exif = g->image->exifData();
        auto it = exif.findKey(Exiv2::ExifKey(key));
        return it != exif.end() ? 1 : 0;
    } catch (...) {
        return 0;
    }
}

char* goexiv2_exif_get_string(void* p, const char* key) {
    auto* g = static_cast<GoExiv2Image*>(p);
    if (!g || !g->metadataRead || !key) return nullptr;
    try {
        auto& exif = g->image->exifData();
        auto it = exif.findKey(Exiv2::ExifKey(key));
        if (it == exif.end()) return nullptr;
        return strdup(it->toString().c_str());
    } catch (...) {
        return nullptr;
    }
}

int64_t goexiv2_exif_get_long(void* p, const char* key) {
    auto* g = static_cast<GoExiv2Image*>(p);
    if (!g || !g->metadataRead || !key) return 0;
    try {
        auto& exif = g->image->exifData();
        auto it = exif.findKey(Exiv2::ExifKey(key));
        if (it == exif.end()) return 0;
        return it->toInt64();
    } catch (...) {
        return 0;
    }
}

double goexiv2_exif_get_double(void* p, const char* key) {
    auto* g = static_cast<GoExiv2Image*>(p);
    if (!g || !g->metadataRead || !key) return 0.0;
    try {
        auto& exif = g->image->exifData();
        auto it = exif.findKey(Exiv2::ExifKey(key));
        if (it == exif.end()) return 0.0;
        return it->toFloat();
    } catch (...) {
        return 0.0;
    }
}

const uint8_t* goexiv2_exif_get_bytes(void* p, const char* key, int* out_len) {
    auto* g = static_cast<GoExiv2Image*>(p);
    if (!g || !g->metadataRead || !key || !out_len) {
        if (out_len) *out_len = 0;
        return nullptr;
    }
    try {
        auto& exif = g->image->exifData();
        auto it = exif.findKey(Exiv2::ExifKey(key));
        if (it == exif.end()) {
            *out_len = 0;
            return nullptr;
        }
        auto& val = it->value();
        auto sz = val.size();
        g->bytesCache.resize(sz);
        val.copy(g->bytesCache.data(), Exiv2::invalidByteOrder);
        *out_len = static_cast<int>(sz);
        return g->bytesCache.data();
    } catch (...) {
        *out_len = 0;
        return nullptr;
    }
}

// --- IPTC ---

int goexiv2_iptc_count(void* p) {
    auto* g = static_cast<GoExiv2Image*>(p);
    if (!g || !g->metadataRead) return 0;
    return static_cast<int>(g->iptcKeys.size());
}

char* goexiv2_iptc_get_key(void* p, int index) {
    auto* g = static_cast<GoExiv2Image*>(p);
    if (!g || !g->metadataRead) return nullptr;
    if (index < 0 || index >= static_cast<int>(g->iptcKeys.size())) return nullptr;
    return strdup(g->iptcKeys[index].c_str());
}

int goexiv2_iptc_has_key(void* p, const char* key) {
    auto* g = static_cast<GoExiv2Image*>(p);
    if (!g || !g->metadataRead || !key) return 0;
    try {
        auto& iptc = g->image->iptcData();
        auto it = iptc.findKey(Exiv2::IptcKey(key));
        return it != iptc.end() ? 1 : 0;
    } catch (...) {
        return 0;
    }
}

char* goexiv2_iptc_get_string(void* p, const char* key) {
    auto* g = static_cast<GoExiv2Image*>(p);
    if (!g || !g->metadataRead || !key) return nullptr;
    try {
        auto& iptc = g->image->iptcData();
        auto it = iptc.findKey(Exiv2::IptcKey(key));
        if (it == iptc.end()) return nullptr;
        return strdup(it->toString().c_str());
    } catch (...) {
        return nullptr;
    }
}

// --- XMP ---

int goexiv2_xmp_count(void* p) {
    auto* g = static_cast<GoExiv2Image*>(p);
    if (!g || !g->metadataRead) return 0;
    return static_cast<int>(g->xmpKeys.size());
}

char* goexiv2_xmp_get_key(void* p, int index) {
    auto* g = static_cast<GoExiv2Image*>(p);
    if (!g || !g->metadataRead) return nullptr;
    if (index < 0 || index >= static_cast<int>(g->xmpKeys.size())) return nullptr;
    return strdup(g->xmpKeys[index].c_str());
}

int goexiv2_xmp_has_key(void* p, const char* key) {
    auto* g = static_cast<GoExiv2Image*>(p);
    if (!g || !g->metadataRead || !key) return 0;
    try {
        auto& xmp = g->image->xmpData();
        auto it = xmp.findKey(Exiv2::XmpKey(key));
        return it != xmp.end() ? 1 : 0;
    } catch (...) {
        return 0;
    }
}

char* goexiv2_xmp_get_string(void* p, const char* key) {
    auto* g = static_cast<GoExiv2Image*>(p);
    if (!g || !g->metadataRead || !key) return nullptr;
    try {
        auto& xmp = g->image->xmpData();
        auto it = xmp.findKey(Exiv2::XmpKey(key));
        if (it == xmp.end()) return nullptr;
        return strdup(it->toString().c_str());
    } catch (...) {
        return nullptr;
    }
}

char* goexiv2_xmp_packet(void* p) {
    auto* g = static_cast<GoExiv2Image*>(p);
    if (!g || !g->metadataRead) return nullptr;
    try {
        auto& packet = g->image->xmpPacket();
        if (packet.empty()) return nullptr;
        return strdup(packet.c_str());
    } catch (...) {
        return nullptr;
    }
}

// --- Utility ---

void goexiv2_free(void* ptr) {
    free(ptr);
}

const char* goexiv2_version(void) {
    return Exiv2::version();
}
