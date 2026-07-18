vcpkg_from_github(
    OUT_SOURCE_PATH SOURCE_PATH
    REPO LibRaw/LibRaw
    REF "${VERSION}"
    SHA512 123050ea30366ada37b40e0aee84453f71f10a5e5e39261a1d16b96dc395f85a9ecdfd043d51b4c347a67546affdfa7ca84c10fa84d73b9b4070c074f1d301e8
    HEAD_REF master
)

vcpkg_from_github(
    OUT_SOURCE_PATH LIBRAW_CMAKE_SOURCE_PATH
    REPO LibRaw/LibRaw-cmake
    REF eb98e4325aef2ce85d2eb031c2ff18640ca616d3
    SHA512 63e68a4d30286ec3aa97168d46b7a1199268099ae27b61abcc92e93ec30e48d364086227983a1d724415e5f4da44d905422f30192453b95f31040e5f8469c3f9
    HEAD_REF master
    PATCHES
        dependencies.patch
        dngsdk-support.patch
        # Move the non-thread-safe library to manual-link. This is unfortunately needed
        # because otherwise libraries that build on top of libraw have to choose.
        fix-install.patch
)

# Download RawSpeed v1 source (legacy, from rawspeed master branch)
if("rawspeed" IN_LIST FEATURES)
    vcpkg_from_github(
        OUT_SOURCE_PATH RAWSPEED_V1_SOURCE_PATH
        REPO darktable-org/rawspeed
        REF 0f1d601c3cf6245ba60a7e05ea11cb62c501b3f1
        SHA512 3f9d34b174622daac0066c234cacce400e81efdba28acc4939a3d51ce410c5a1e597f07e7c471d7b672e94c22c10aa540fbb1eb30b725ff29c8db19a268be2c5
        HEAD_REF master
        PATCHES
            rawspeed.cpucount-unix.patch
            rawspeed.cpucount-macos.patch
            rawspeed.samsung-decoder.patch
            rawspeed.mingw-compat.patch
    )
endif()

file(COPY "${LIBRAW_CMAKE_SOURCE_PATH}/CMakeLists.txt" DESTINATION "${SOURCE_PATH}")
file(COPY "${LIBRAW_CMAKE_SOURCE_PATH}/cmake" DESTINATION "${SOURCE_PATH}")

# Copy patched RawSpeed v1 sources into LibRaw source tree
if("rawspeed" IN_LIST FEATURES)
    file(GLOB RAWSPEED_V1_SOURCES
        "${RAWSPEED_V1_SOURCE_PATH}/RawSpeed/*.cpp"
        "${RAWSPEED_V1_SOURCE_PATH}/RawSpeed/*.h"
    )
    file(COPY ${RAWSPEED_V1_SOURCES} DESTINATION "${SOURCE_PATH}/RawSpeed")

    # Create minimal dlldef.h for static builds (win32-dll.patch not needed)
    file(WRITE "${SOURCE_PATH}/RawSpeed/dlldef.h"
        "#ifndef DLLDEF_H\n#define DLLDEF_H\n#define DllDef\n#endif\n")
endif()

# Download LibRaw Demosaic Pack GPL2
if("demosaic-pack-gpl2" IN_LIST FEATURES OR "demosaic-pack-gpl3" IN_LIST FEATURES)
    vcpkg_from_github(
        OUT_SOURCE_PATH DEMOSAIC_PACK_GPL2_SOURCE_PATH
        REPO LibRaw/LibRaw-demosaic-pack-GPL2
        REF 0.18.6
        SHA512 84bf3e1136735bf316810c30a8fd03597737fef11c203bc2b6e3435fb67b821cf7fd7069344a054eac76ce5d49ac311e59464e62e3ce448c80409648726e2ed0
        HEAD_REF master
    )
    file(COPY
        "${DEMOSAIC_PACK_GPL2_SOURCE_PATH}/afd_interpolate_pl.c"
        "${DEMOSAIC_PACK_GPL2_SOURCE_PATH}/ahd_interpolate_mod.c"
        "${DEMOSAIC_PACK_GPL2_SOURCE_PATH}/ahd_partial_interpolate.c"
        "${DEMOSAIC_PACK_GPL2_SOURCE_PATH}/vcd_interpolate.c"
        "${DEMOSAIC_PACK_GPL2_SOURCE_PATH}/lmmse_interpolate.c"
        "${DEMOSAIC_PACK_GPL2_SOURCE_PATH}/refinement.c"
        "${DEMOSAIC_PACK_GPL2_SOURCE_PATH}/es_median_filter.c"
        "${DEMOSAIC_PACK_GPL2_SOURCE_PATH}/median_filter_new.c"
        DESTINATION "${SOURCE_PATH}/src/demosaic/gpl2"
    )
endif()

# Download LibRaw Demosaic Pack GPL3
if("demosaic-pack-gpl3" IN_LIST FEATURES)
    vcpkg_from_github(
        OUT_SOURCE_PATH DEMOSAIC_PACK_GPL3_SOURCE_PATH
        REPO LibRaw/LibRaw-demosaic-pack-GPL3
        REF 0.18.6
        SHA512 0425051c7058992f0058eb5b0a28d65ac8ee7efb2c54d7631bd3d29e297e6d5a87d395916ec7f5becd9863990237b48cc4661012893aa2e1ba2c6833ba3ce0cc
        HEAD_REF master
    )
    file(COPY
        "${DEMOSAIC_PACK_GPL3_SOURCE_PATH}/amaze_demosaic_RT.cc"
        DESTINATION "${SOURCE_PATH}/src/demosaic/gpl3"
    )
endif()

# Add ENABLE_RAWSPEED3 option and support (avoids fragile patch)
file(READ "${SOURCE_PATH}/CMakeLists.txt" CMAKE_CONTENT)
string(REPLACE
    "option(ENABLE_RAWSPEED             \"Build library with extra RawSpeed codec support (default=OFF)\"                OFF)"
    "option(ENABLE_RAWSPEED             \"Build library with extra RawSpeed codec support (default=OFF)\"                OFF)\noption(ENABLE_RAWSPEED3            \"Build library with RawSpeed v3 codec support  (default=OFF)\"                OFF)\noption(ENABLE_DEMOSAIC_PACK_GPL2   \"Build with GPL2 demosaic pack support       (default=OFF)\"                OFF)\noption(ENABLE_DEMOSAIC_PACK_GPL3   \"Build with GPL3 demosaic pack support       (default=OFF)\"                OFF)"
    CMAKE_CONTENT "${CMAKE_CONTENT}"
)
string(REPLACE
    "MACRO_BOOL_TO_01(RAWSPEED_SUPPORT_CAN_BE_COMPILED LIBRAW_USE_RAWSPEED)"
    "MACRO_BOOL_TO_01(RAWSPEED_SUPPORT_CAN_BE_COMPILED LIBRAW_USE_RAWSPEED)\n\n# RawSpeed v3 support\nif(ENABLE_RAWSPEED3)\n    if(NOT TARGET rawspeed3::rawspeed3)\n        find_package(rawspeed3 CONFIG REQUIRED)\n    endif()\n    add_definitions(-DUSE_RAWSPEED3)\n    set(RAWSPEED3_SUPPORT_CAN_BE_COMPILED true)\nendif()\nMACRO_BOOL_TO_01(RAWSPEED3_SUPPORT_CAN_BE_COMPILED LIBRAW_USE_RAWSPEED3)"
    CMAKE_CONTENT "${CMAKE_CONTENT}"
)
# Add rawspeed3 linking to raw and raw_r targets
string(REPLACE
    "if(RAWSPEED_SUPPORT_CAN_BE_COMPILED)\n    target_link_libraries(raw PUBLIC ${LIBXML2_LIBRARIES})\nendif()"
    "if(RAWSPEED_SUPPORT_CAN_BE_COMPILED)\n    target_link_libraries(raw PUBLIC ${LIBXML2_LIBRARIES})\nendif()\n\nif(RAWSPEED3_SUPPORT_CAN_BE_COMPILED)\n    target_link_libraries(raw PUBLIC rawspeed3::rawspeed3)\nendif()"
    CMAKE_CONTENT "${CMAKE_CONTENT}"
)
string(REPLACE
    "if(RAWSPEED_SUPPORT_CAN_BE_COMPILED)\n    target_link_libraries(raw_r PUBLIC ${LIBXML2_LIBRARIES} Threads::Threads)\nendif()"
    "if(RAWSPEED_SUPPORT_CAN_BE_COMPILED)\n    target_link_libraries(raw_r PUBLIC ${LIBXML2_LIBRARIES} Threads::Threads)\nendif()\n\nif(RAWSPEED3_SUPPORT_CAN_BE_COMPILED)\n    target_link_libraries(raw_r PUBLIC rawspeed3::rawspeed3)\nendif()"
    CMAKE_CONTENT "${CMAKE_CONTENT}"
)
# Add demosaic pack compile definitions
string(REPLACE
    "# Flag to add Raspberry Pi RAW support"
    "# Demosaic Pack support\nif(ENABLE_DEMOSAIC_PACK_GPL2)\n    add_definitions(-DLIBRAW_DEMOSAIC_PACK_GPL2)\nendif()\nif(ENABLE_DEMOSAIC_PACK_GPL3)\n    add_definitions(-DLIBRAW_DEMOSAIC_PACK_GPL3)\nendif()\n\n# Flag to add Raspberry Pi RAW support"
    CMAKE_CONTENT "${CMAKE_CONTENT}"
)
file(WRITE "${SOURCE_PATH}/CMakeLists.txt" "${CMAKE_CONTENT}")

# Fix PTHREADS_FOUND bug: find_package(Threads) sets Threads_FOUND, not PTHREADS_FOUND
if("rawspeed" IN_LIST FEATURES)
    file(READ "${SOURCE_PATH}/CMakeLists.txt" CMAKE_CONTENT)
    string(REPLACE "AND PTHREADS_FOUND)" "AND Threads_FOUND)" CMAKE_CONTENT "${CMAKE_CONTENT}")
    string(REPLACE "if(NOT PTHREADS_FOUND)" "if(NOT Threads_FOUND)" CMAKE_CONTENT "${CMAKE_CONTENT}")
    string(REPLACE "include_directories(\${LIBXML2_INCLUDE_DIR} \${PTHREADS_INCLUDE_DIR})"
                   "include_directories(\${LIBXML2_INCLUDE_DIR})" CMAKE_CONTENT "${CMAKE_CONTENT}")
    string(REPLACE "add_definitions(\${LIBXML2_DEFINITIONS} \${PTHREADS_DEFINITIONS})"
                   "add_definitions(\${LIBXML2_DEFINITIONS})" CMAKE_CONTENT "${CMAKE_CONTENT}")
    # Add rawspeed_xmldata.cpp to build (contains embedded cameras.xml)
    string(REPLACE
        "\${RAWSPEED_PATH}/TiffParserOlympus.cpp\n    )"
        "\${RAWSPEED_PATH}/TiffParserOlympus.cpp\n                             \${RAWSPEED_PATH}/rawspeed_xmldata.cpp\n    )"
        CMAKE_CONTENT "${CMAKE_CONTENT}"
    )
    file(WRITE "${SOURCE_PATH}/CMakeLists.txt" "${CMAKE_CONTENT}")
endif()

# Inject LIBRAW_USE_RAWSPEED3 into config header template
file(READ "${SOURCE_PATH}/cmake/data/libraw_config.h.cmake" CONFIG_H_CONTENT)
string(REPLACE
    "#cmakedefine LIBRAW_USE_RAWSPEED 1"
    "#cmakedefine LIBRAW_USE_RAWSPEED 1

/* Define to 1 if LibRaw have been compiled with RawSpeed v3 codec support */
#cmakedefine LIBRAW_USE_RAWSPEED3 1"
    CONFIG_H_CONTENT "${CONFIG_H_CONTENT}"
)
file(WRITE "${SOURCE_PATH}/cmake/data/libraw_config.h.cmake" "${CONFIG_H_CONTENT}")

# Copy demosaic_packs.cpp bridge file
file(COPY "${CMAKE_CURRENT_LIST_DIR}/demosaic_packs.cpp"
     DESTINATION "${SOURCE_PATH}/src/demosaic")

# Patch dcraw_process.cpp — add quality 5-10 dispatch
file(READ "${SOURCE_PATH}/src/postprocessing/dcraw_process.cpp" DCP_CONTENT)
string(REPLACE
    "      else if (quality == 4)\n        dcb(iterations, dcb_enhance);\n\n      else if (quality == 11)"
    "      else if (quality == 4)\n        dcb(iterations, dcb_enhance);\n\n#ifdef LIBRAW_DEMOSAIC_PACK_GPL2\n      else if (quality == 5)\n        ahd_interpolate_mod();\n      else if (quality == 6)\n        afd_interpolate_pl(2, 1);\n      else if (quality == 7)\n        vcd_interpolate(0);\n      else if (quality == 8)\n      {\n        vcd_interpolate(12);\n        refinement();\n        if (O.med_passes > 0)\n        {\n          median_filter_new();\n          es_median_filter();\n        }\n      }\n      else if (quality == 9)\n        lmmse_interpolate(1);\n#endif\n#ifdef LIBRAW_DEMOSAIC_PACK_GPL3\n      else if (quality == 10)\n        amaze_demosaic_RT();\n#endif\n\n      else if (quality == 11)"
    DCP_CONTENT "${DCP_CONTENT}"
)
file(WRITE "${SOURCE_PATH}/src/postprocessing/dcraw_process.cpp" "${DCP_CONTENT}")
# Verify dcraw_process.cpp patch applied
string(FIND "${DCP_CONTENT}" "LIBRAW_DEMOSAIC_PACK_GPL2" _DCP_PATCH_CHECK)
if(_DCP_PATCH_CHECK LESS 0)
    message(FATAL_ERROR "Failed to patch dcraw_process.cpp: quality 5-10 dispatch anchor not found")
endif()

# Patch libraw_internal_funcs.h — add function declarations
file(READ "${SOURCE_PATH}/internal/libraw_internal_funcs.h" IFH_CONTENT)
string(REPLACE
    "\tvoid \tdcb_nyquist();\n#endif"
    "\tvoid \tdcb_nyquist();\n// Demosaic Pack (GPL2)\n\tvoid  \tahd_interpolate_mod();\n\tvoid  \tahd_partial_interpolate(int threshold_value);\n\tvoid  \tafd_interpolate_pl(int afd_passes, int clip_on);\n\tvoid  \tvcd_interpolate(int flags);\n\tvoid  \tlmmse_interpolate(int iterations);\n\tvoid  \tes_median_filter();\n\tvoid  \tmedian_filter_new();\n\tvoid  \trefinement();\n// Demosaic Pack (GPL3)\n\tvoid  \tamaze_demosaic_RT();\n#endif"
    IFH_CONTENT "${IFH_CONTENT}"
)
file(WRITE "${SOURCE_PATH}/internal/libraw_internal_funcs.h" "${IFH_CONTENT}")
# Verify libraw_internal_funcs.h patch applied
string(FIND "${IFH_CONTENT}" "ahd_interpolate_mod" _IFH_PATCH_CHECK)
if(_IFH_PATCH_CHECK LESS 0)
    message(FATAL_ERROR "Failed to patch libraw_internal_funcs.h: declaration anchor not found")
endif()

vcpkg_check_features(OUT_FEATURE_OPTIONS FEATURE_OPTIONS
    FEATURES
        openmp      ENABLE_OPENMP
        openmp      CMAKE_REQUIRE_FIND_PACKAGE_OpenMP
        dng-lossy   CMAKE_REQUIRE_FIND_PACKAGE_JPEG
        dngsdk      ENABLE_DNGSDK
        rawspeed   ENABLE_RAWSPEED
        rawspeed3  ENABLE_RAWSPEED3
        x3ftools   ENABLE_X3FTOOLS
        6by9rpi    ENABLE_6BY9RPI
        demosaic-pack-gpl2 ENABLE_DEMOSAIC_PACK_GPL2
        demosaic-pack-gpl3 ENABLE_DEMOSAIC_PACK_GPL3
)

vcpkg_cmake_configure(
    SOURCE_PATH "${SOURCE_PATH}"
    OPTIONS
        ${FEATURE_OPTIONS}
        -DENABLE_EXAMPLES=OFF
        -DCMAKE_REQUIRE_FIND_PACKAGE_Jasper=1
        -DCMAKE_REQUIRE_FIND_PACKAGE_ZLIB=1
        -DCMAKE_CXX_FLAGS=-D_USE_MATH_DEFINES
    MAYBE_UNUSED_VARIABLES
        CMAKE_REQUIRE_FIND_PACKAGE_OpenMP
)

vcpkg_cmake_install()
vcpkg_copy_pdbs()
vcpkg_cmake_config_fixup(CONFIG_PATH "lib/cmake")
vcpkg_fixup_pkgconfig()

if(VCPKG_LIBRARY_LINKAGE STREQUAL "static")
    vcpkg_replace_string("${CURRENT_PACKAGES_DIR}/include/libraw/libraw_types.h"
        "#ifdef LIBRAW_NODLL" "#if 1"
    )
else()
    vcpkg_replace_string("${CURRENT_PACKAGES_DIR}/include/libraw/libraw_types.h"
        "#ifdef LIBRAW_NODLL" "#if 0"
    )
endif()

file(COPY "${CURRENT_PACKAGES_DIR}/share/cmake/libraw/FindLibRaw.cmake" DESTINATION "${CURRENT_PACKAGES_DIR}/share/${PORT}")
file(REMOVE_RECURSE
    "${CURRENT_PACKAGES_DIR}/debug/include"
    "${CURRENT_PACKAGES_DIR}/debug/share"
    "${CURRENT_PACKAGES_DIR}/share/cmake"
    "${CURRENT_PACKAGES_DIR}/share/doc"
)

# Add direct dependency to .pc when dngsdk feature is enabled
# Transitive deps resolved via dng.pc -> xmp.pc -> libjxl.pc chain
set(_RAW_CFLAGS "")
if("dngsdk" IN_LIST FEATURES)
    set(_RAW_DNG_REQUIRE "dng")
    string(APPEND _RAW_CFLAGS " -DUSE_DNGSDK")
else()
    set(_RAW_DNG_REQUIRE "")
endif()
if("rawspeed3" IN_LIST FEATURES)
    string(APPEND _RAW_CFLAGS " -DUSE_RAWSPEED3 -DUSE_RAWSPEED_BITS")
    string(APPEND _RAW_PC_REQUIRE " rawspeed3")
endif()
if("rawspeed" IN_LIST FEATURES)
    string(APPEND _RAW_CFLAGS " -DUSE_RAWSPEED")
    string(APPEND _RAW_PC_REQUIRE " libxml-2.0")
endif()
foreach(_pc IN ITEMS libraw libraw_r)
    set(_pc_file "${CURRENT_PACKAGES_DIR}/lib/pkgconfig/${_pc}.pc")
    if(EXISTS "${_pc_file}")
        if(_RAW_DNG_REQUIRE OR _RAW_PC_REQUIRE)
            set(_RAW_ALL_REQUIRES "${_RAW_DNG_REQUIRE}${_RAW_PC_REQUIRE} lcms2 zlib libjpeg")
            vcpkg_replace_string("${_pc_file}" "Requires:  lcms2 zlib libjpeg" "Requires: ${_RAW_ALL_REQUIRES}")
        endif()
        if(_RAW_CFLAGS)
            vcpkg_replace_string("${_pc_file}" "Cflags:" "Cflags:${_RAW_CFLAGS}")
        endif()
    endif()
endforeach()

configure_file("${CMAKE_CURRENT_LIST_DIR}/vcpkg-cmake-wrapper.cmake" "${CURRENT_PACKAGES_DIR}/share/${PORT}/vcpkg-cmake-wrapper.cmake" @ONLY)
file(INSTALL "${CMAKE_CURRENT_LIST_DIR}/usage" DESTINATION "${CURRENT_PACKAGES_DIR}/share/${PORT}")
set(COPYRIGHT_FILES
    "${SOURCE_PATH}/COPYRIGHT"
    "${SOURCE_PATH}/LICENSE.LGPL"
    "${SOURCE_PATH}/LICENSE.CDDL"
)
if("demosaic-pack-gpl2" IN_LIST FEATURES)
    list(APPEND COPYRIGHT_FILES "${DEMOSAIC_PACK_GPL2_SOURCE_PATH}/LICENSE.txt")
endif()
if("demosaic-pack-gpl3" IN_LIST FEATURES)
    list(APPEND COPYRIGHT_FILES "${DEMOSAIC_PACK_GPL3_SOURCE_PATH}/LICENSE.txt")
endif()
vcpkg_install_copyright(FILE_LIST ${COPYRIGHT_FILES})
