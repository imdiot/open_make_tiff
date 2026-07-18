set(DNG_VERSION "1.7.1")

vcpkg_download_distfile(ARCHIVE
    URLS "file://${CMAKE_CURRENT_LIST_DIR}/dng_sdk_1_7_1_2502_20260303.zip"
    FILENAME "dng_sdk_1_7_1_2502_20260303.zip"
    SHA512 cb94f7a58258bdf7e5a0e2a879fefbe567766abc2b3fda162e598371f799471134a2278f0fe21a1d631f5624acdd9a64deb93840c5a2a68283edcde7b44248e2
)

vcpkg_extract_source_archive(
    SOURCE_PATH
    ARCHIVE "${ARCHIVE}"
    SOURCE_BASE "1.7.1"
    PATCHES
        mingw-xmp-environment.patch
        mingw-xmp-common-defines.patch
        mingw-suppress-sal.patch
        mingw-pthread.patch
)

# Copy CMakeLists.txt
file(COPY "${CMAKE_CURRENT_LIST_DIR}/CMakeLists.txt" DESTINATION "${SOURCE_PATH}")
file(COPY "${CMAKE_CURRENT_LIST_DIR}/cmake" DESTINATION "${SOURCE_PATH}")

vcpkg_check_features(OUT_FEATURE_OPTIONS FEATURE_OPTIONS
    FEATURES
        tools DNG_BUILD_TOOLS
)

vcpkg_cmake_configure(
    SOURCE_PATH "${SOURCE_PATH}"
    OPTIONS
        ${FEATURE_OPTIONS}
)

vcpkg_cmake_install()
vcpkg_cmake_config_fixup(CONFIG_PATH "lib/cmake/adobe-dng-sdk")
vcpkg_copy_pdbs()

if(DNG_BUILD_TOOLS)
    vcpkg_copy_tools(TOOL_NAMES dng_validate AUTO_CLEAN)
endif()

file(REMOVE_RECURSE "${CURRENT_PACKAGES_DIR}/debug/include")

# Generate pkg-config files (upstream CMake does not produce .pc)
set(PKGCONFIG_DIR "${CURRENT_PACKAGES_DIR}/lib/pkgconfig")
file(MAKE_DIRECTORY "${PKGCONFIG_DIR}")

# Platform-specific system libraries
set(_DNG_SYSLIBS "")
set(_XMP_SYSLIBS "")
if(APPLE)
    string(APPEND _DNG_SYSLIBS " -framework CoreFoundation -framework CoreServices")
    string(APPEND _XMP_SYSLIBS " -framework CoreFoundation -framework CoreServices")
elseif(UNIX)
    string(APPEND _DNG_SYSLIBS " -lm -lc++")
    string(APPEND _XMP_SYSLIBS " -lc++")
endif()

# MinGW uses POSIX pthreads and needs ws2_32 for htons/ntohl
if(VCPKG_CMAKE_SYSTEM_NAME STREQUAL "MinGW")
    string(APPEND _DNG_SYSLIBS " -lws2_32")
    string(APPEND _XMP_SYSLIBS " -lole32 -lshell32 -luuid")
endif()

file(WRITE "${PKGCONFIG_DIR}/dng.pc" "prefix=\${pcfiledir}/../..
exec_prefix=\${prefix}
libdir=\${exec_prefix}/lib
includedir=\${prefix}/include

Name: dng
Description: Adobe DNG SDK
Version: ${DNG_VERSION}
Libs: \"-L\${libdir}\" -ldng${_DNG_SYSLIBS}
Requires: xmp libjxl libjxl_threads libjxl_cms libjpeg zlib
Cflags: \"-I\${includedir}\"
")

file(WRITE "${PKGCONFIG_DIR}/xmp.pc" "prefix=\${pcfiledir}/../..
exec_prefix=\${prefix}
libdir=\${exec_prefix}/lib
includedir=\${prefix}/include

Name: xmp
Description: Adobe XMP SDK (part of DNG SDK)
Version: ${DNG_VERSION}
Libs: \"-L\${libdir}\" -lxmp${_XMP_SYSLIBS}
Requires: expat zlib
Cflags: \"-I\${includedir}\"
")

vcpkg_install_copyright(FILE_LIST "${SOURCE_PATH}/LICENSE.txt")

file(INSTALL "${CMAKE_CURRENT_LIST_DIR}/usage"
     DESTINATION "${CURRENT_PACKAGES_DIR}/share/${PORT}")
