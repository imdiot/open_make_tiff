vcpkg_download_distfile(ARCHIVE
    URLS "file://${CMAKE_CURRENT_LIST_DIR}/dng_sdk_1_7_1_2502_20260303.zip"
    FILENAME "dng_sdk_1_7_1_2502_20260303.zip"
    SHA512 cb94f7a58258bdf7e5a0e2a879fefbe567766abc2b3fda162e598371f799471134a2278f0fe21a1d631f5624acdd9a64deb93840c5a2a68283edcde7b44248e2
)

vcpkg_extract_source_archive(
    SOURCE_PATH
    ARCHIVE "${ARCHIVE}"
    SOURCE_BASE "1.7.1"
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

# On MinGW, __stdcall on virtual member functions breaks vtable matching (MSVC ignores it for members)
if(VCPKG_CMAKE_SYSTEM_NAME STREQUAL "MinGW")
    file(READ "${SOURCE_PATH}/xmp/toolkit/public/include/XMP_Environment.h" _env_content)
    string(REPLACE "#define APICALL __stdcall" "#define APICALL" _env_content "${_env_content}")
    file(WRITE "${SOURCE_PATH}/xmp/toolkit/public/include/XMP_Environment.h" "${_env_content}")

    # Fix XMPCommonDefines.h: _MSC_VER is undefined on MinGW, causing
    # "#if _MSC_VER <= 1600" to evaluate true (0 <= 1600), incorrectly
    # selecting std::tr1 path. Insert MinGW detection before the _MSC_VER check.
    file(READ "${SOURCE_PATH}/xmp/toolkit/public/include/XMPCommon/XMPCommonDefines.h" _defines_content)
    string(REPLACE
        "	#if _MSC_VER <= 1600"
        "	#if defined(__MINGW32__) || defined(__MINGW64__)\n\t\t#define SUPPORT_STD_ATOMIC_IMPLEMENTATION 1\n\t\t#define SUPPORT_SHARED_POINTERS_IN_TR1 0\n\t\t#define SUPPORT_SHARED_POINTERS_IN_STD 1\n\t#elif _MSC_VER <= 1600"
        _defines_content "${_defines_content}")
    file(WRITE "${SOURCE_PATH}/xmp/toolkit/public/include/XMPCommon/XMPCommonDefines.h" "${_defines_content}")

    # Fix SuppressSAL.h: SAL annotations are only suppressed for non-Windows,
    # but MinGW is Windows without MSVC's SAL headers. Extend the guard.
    file(READ "${SOURCE_PATH}/xmp/toolkit/source/SuppressSAL.h" _sal_content)
    string(REPLACE
        "#if !defined(_WIN32) && !defined(_WIN64)"
        "#if (!defined(_WIN32) && !defined(_WIN64)) || defined(__MINGW32__)"
        _sal_content "${_sal_content}")
    file(WRITE "${SOURCE_PATH}/xmp/toolkit/source/SuppressSAL.h" "${_sal_content}")

    # Fix dng_pthread.h: MinGW has native pthreads, but the DNG SDK's qWinOS
    # check causes it to use its own fake pthreads implementation (designed for MSVC).
    # Change the guard so MinGW uses system pthreads like other POSIX platforms.
    file(READ "${SOURCE_PATH}/dng_sdk/source/dng_pthread.h" _pthread_content)
    string(REPLACE
        "#if !qWinOS"
        "#if !qWinOS || defined(__MINGW32__)"
        _pthread_content "${_pthread_content}")
    file(WRITE "${SOURCE_PATH}/dng_sdk/source/dng_pthread.h" "${_pthread_content}")
endif()

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
    string(APPEND _DNG_SYSLIBS " -lpthread -lws2_32")
    string(APPEND _XMP_SYSLIBS " -lole32 -lshell32 -luuid")
endif()

file(WRITE "${PKGCONFIG_DIR}/dng.pc" "prefix=\${pcfiledir}/../..
exec_prefix=\${prefix}
libdir=\${exec_prefix}/lib
includedir=\${prefix}/include

Name: dng
Description: Adobe DNG SDK
Version: 1.7.1
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
Version: 1.7.1
Libs: \"-L\${libdir}\" -lxmp${_XMP_SYSLIBS}
Requires: expat zlib
Cflags: \"-I\${includedir}\"
")

vcpkg_install_copyright(FILE_LIST "${SOURCE_PATH}/LICENSE.txt")

file(INSTALL "${CMAKE_CURRENT_LIST_DIR}/usage"
     DESTINATION "${CURRENT_PACKAGES_DIR}/share/${PORT}")
