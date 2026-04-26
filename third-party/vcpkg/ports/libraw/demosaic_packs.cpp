/* Demosaic pack integration bridge for LibRaw 0.22.x */
#include "../../internal/dcraw_defs.h"

#define CLASS LibRaw::

/* Compatibility: merror() was removed in LibRaw 0.22 */
#undef merror
#define merror(ptr, where) do { if (!(ptr)) throw LIBRAW_EXCEPTION_ALLOC; } while(0)

/* Compatibility: static class members accessed as bare names by pack code */
#define xyz_rgb  LibRaw_constants::xyz_rgb
#define d65_white LibRaw_constants::d65_white

/* Compatibility: es_med_passes was removed from output_params in LibRaw 0.22.
   Map to med_passes — both control median filter iterations but originally had
   independent defaults (es_med_passes=1, med_passes=0). */
#define es_med_passes med_passes

#ifdef LIBRAW_DEMOSAIC_PACK_GPL2
/* Modified AHD (quality 5) */
#include "gpl2/ahd_interpolate_mod.c"
#undef TS
/* LMMSE (quality 9) */
#include "gpl2/lmmse_interpolate.c"
#undef PIX_SORT
/* AFD (quality 6) */
#include "gpl2/ahd_partial_interpolate.c"
#undef TS
#include "gpl2/afd_interpolate_pl.c"
#undef PIX_SORT
/* VCD + post-processing (quality 7, 8) */
#include "gpl2/refinement.c"
#include "gpl2/es_median_filter.c"
#undef PIX_SORT
#include "gpl2/median_filter_new.c"
#undef PIX_SORT
#include "gpl2/vcd_interpolate.c"
#undef PIX_SORT
#else
/* Stub fallback — fall back to AHD when packs are disabled */
void CLASS ahd_interpolate_mod() { ahd_interpolate(); }
void CLASS ahd_partial_interpolate(int) { ahd_interpolate(); }
void CLASS afd_interpolate_pl(int, int) { ahd_interpolate(); }
void CLASS vcd_interpolate(int) { ahd_interpolate(); }
void CLASS lmmse_interpolate(int) { ahd_interpolate(); }
void CLASS es_median_filter() {}
void CLASS median_filter_new() {}
void CLASS refinement() {}
#endif

#ifdef LIBRAW_DEMOSAIC_PACK_GPL3
/* AMaZE (quality 10) — redefines SQR/LIM/ULIM (identical to defines.h) */
#undef SQR
#undef LIM
#undef ULIM
#include "gpl3/amaze_demosaic_RT.cc"
#undef TS
#else
void CLASS amaze_demosaic_RT() { ahd_interpolate(); }
#endif

/* Cleanup compatibility macros */
#undef merror
#undef xyz_rgb
#undef d65_white
#undef es_med_passes
