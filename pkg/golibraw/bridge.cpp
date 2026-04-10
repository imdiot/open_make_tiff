// golibraw C++ bridge: wraps LibRaw C++ class methods behind C ABI for CGo.

#include <libraw/libraw.h>

extern "C" {

int golibraw_is_fuji_rotated(libraw_data_t* lr) {
    if (!lr) return 0;
    LibRaw* ip = (LibRaw*)(lr->parent_class);
    return ip->is_fuji_rotated();
}

int golibraw_is_sraw(libraw_data_t* lr) {
    if (!lr) return 0;
    LibRaw* ip = (LibRaw*)(lr->parent_class);
    return ip->is_sraw();
}

int golibraw_sraw_midpoint(libraw_data_t* lr) {
    if (!lr) return 0;
    LibRaw* ip = (LibRaw*)(lr->parent_class);
    return ip->sraw_midpoint();
}

int golibraw_is_nikon_sraw(libraw_data_t* lr) {
    if (!lr) return 0;
    LibRaw* ip = (LibRaw*)(lr->parent_class);
    return ip->is_nikon_sraw();
}

int golibraw_is_coolscan_nef(libraw_data_t* lr) {
    if (!lr) return 0;
    LibRaw* ip = (LibRaw*)(lr->parent_class);
    return ip->is_coolscan_nef();
}

int golibraw_is_jpeg_thumb(libraw_data_t* lr) {
    if (!lr) return 0;
    LibRaw* ip = (LibRaw*)(lr->parent_class);
    return ip->is_jpeg_thumb();
}

int golibraw_is_floating_point(libraw_data_t* lr) {
    if (!lr) return 0;
    LibRaw* ip = (LibRaw*)(lr->parent_class);
    return ip->is_floating_point();
}

int golibraw_have_fpdata(libraw_data_t* lr) {
    if (!lr) return 0;
    LibRaw* ip = (LibRaw*)(lr->parent_class);
    return ip->have_fpdata();
}

int golibraw_error_count(libraw_data_t* lr) {
    if (!lr) return 0;
    LibRaw* ip = (LibRaw*)(lr->parent_class);
    return ip->error_count();
}

int golibraw_thumb_ok(libraw_data_t* lr, long long maxsz) {
    if (!lr) return 0;
    LibRaw* ip = (LibRaw*)(lr->parent_class);
    return ip->thumbOK((INT64)maxsz);
}

int golibraw_raw_was_read(libraw_data_t* lr) {
    if (!lr) return 0;
    // raw_was_read() is protected, replicate its logic by checking rawdata pointers
    return lr->rawdata.raw_image || lr->rawdata.color4_image
        || lr->rawdata.color3_image || lr->rawdata.float_image
        || lr->rawdata.float3_image || lr->rawdata.float4_image;
}

// Color filter queries
int golibraw_color(libraw_data_t* lr, int row, int col) {
    if (!lr) return 0;
    LibRaw* ip = (LibRaw*)(lr->parent_class);
    return ip->COLOR(row, col);
}

int golibraw_fc(libraw_data_t* lr, int row, int col) {
    if (!lr) return 0;
    LibRaw* ip = (LibRaw*)(lr->parent_class);
    return ip->FC(row, col);
}

int golibraw_fcol(libraw_data_t* lr, int row, int col) {
    if (!lr) return 0;
    LibRaw* ip = (LibRaw*)(lr->parent_class);
    return ip->fcol(row, col);
}

// Pipeline step methods
int golibraw_adjust_maximum(libraw_data_t* lr) {
    if (!lr) return LIBRAW_INPUT_CLOSED;
    LibRaw* ip = (LibRaw*)(lr->parent_class);
    return ip->adjust_maximum();
}

int golibraw_raw2image_ex(libraw_data_t* lr, int do_subtract_black) {
    if (!lr) return LIBRAW_INPUT_CLOSED;
    LibRaw* ip = (LibRaw*)(lr->parent_class);
    return ip->raw2image_ex(do_subtract_black);
}

void golibraw_convert_float_to_int(libraw_data_t* lr, float dmin, float dmax, float dtarget) {
    if (!lr) return;
    LibRaw* ip = (LibRaw*)(lr->parent_class);
    ip->convertFloatToInt(dmin, dmax, dtarget);
}

// Output optimization
void golibraw_get_mem_image_format(libraw_data_t* lr, int* width, int* height, int* colors, int* bps) {
    if (!lr) return;
    LibRaw* ip = (LibRaw*)(lr->parent_class);
    ip->get_mem_image_format(width, height, colors, bps);
}

int golibraw_copy_mem_image(libraw_data_t* lr, void* scan0, int stride, int bgr) {
    if (!lr) return LIBRAW_INPUT_CLOSED;
    LibRaw* ip = (LibRaw*)(lr->parent_class);
    return ip->copy_mem_image(scan0, stride, bgr);
}

// Utility methods
int golibraw_set_make_from_index(libraw_data_t* lr, unsigned index) {
    if (!lr) return LIBRAW_INPUT_CLOSED;
    LibRaw* ip = (LibRaw*)(lr->parent_class);
    return ip->setMakeFromIndex(index);
}

int golibraw_set_rawspeed_camerafile(libraw_data_t* lr, char* filename) {
    if (!lr) return LIBRAW_INPUT_CLOSED;
    LibRaw* ip = (LibRaw*)(lr->parent_class);
    return ip->set_rawspeed_camerafile(filename);
}

// Fallback when LibRaw was built without USE_RAWSPEED.
// The real implementation lives in rawspeed_glue.cpp under #ifdef USE_RAWSPEED.
// Provide a weak symbol so the linker resolves to us only when the real one is absent.
__attribute__((weak))
int LibRaw::set_rawspeed_camerafile(char*) {
    return LIBRAW_NOT_IMPLEMENTED;
}

} // extern "C"
