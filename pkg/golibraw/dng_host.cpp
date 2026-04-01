// golibraw DNG SDK bridge: creates dng_host via C ABI for CGo.

#include <libraw/libraw.h>

#ifdef USE_DNGSDK
#include <dng_host.h>
#endif

extern "C" {

void* golibraw_create_dng_host() {
#ifdef USE_DNGSDK
    return new dng_host;
#else
    return (void*)0;
#endif
}

void golibraw_destroy_dng_host(void* host) {
#ifdef USE_DNGSDK
    if (host) {
        dng_host* h = static_cast<dng_host*>(host);
        delete h;
    }
#endif
}

// Accesses LibRaw* via parent_class to call set_dng_host.
void golibraw_set_dng_host_for_raw(libraw_data_t* lr, void* host) {
    if (!lr || !host) return;
#ifdef USE_DNGSDK
    LibRaw* ip = (LibRaw*)(lr->parent_class);
    ip->set_dng_host(host);
#endif
}

} // extern "C"
