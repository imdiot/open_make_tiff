cmake_minimum_required(VERSION 3.15)
project(adobe-dng-sdk VERSION 1.7.1 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 14)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

include(GNUInstallDirs)
include(CMakePackageConfigHelpers)

# Options
option(DNG_BUILD_TOOLS "Build command-line tools" OFF)

# Find dependencies from vcpkg
find_package(JPEG REQUIRED)
find_package(ZLIB REQUIRED)
find_package(expat CONFIG REQUIRED)

# libjxl - pkg-config with IMPORTED_TARGET handles transitive deps (hwy, brotli, etc.)
find_package(PkgConfig REQUIRED)
pkg_check_modules(LIBJXL REQUIRED libjxl libjxl_threads libjxl_cms IMPORTED_TARGET)

# ============================================================================
# Platform detection
# ============================================================================
if(APPLE)
    add_compile_definitions(
        qMacOS=1 qWinOS=0 qLinux=0
        MAC_ENV=1 XMP_MacBuild=1
    )
elseif(WIN32)
    add_compile_definitions(
        qMacOS=0 qWinOS=1 qLinux=0
        WIN_ENV=1 XMP_WinBuild=1
        NOMINMAX=1  # Prevent Windows min/max macro conflicts
        UNICODE=0   # Disable Unicode to use ANSI APIs
        _UNICODE=0  # Disable Unicode to use ANSI APIs
    )
else()
    add_compile_definitions(
        qMacOS=0 qWinOS=0 qLinux=1
        UNIX_ENV=1 XMP_UNIXBuild=1
    )
endif()

# ============================================================================
# XMP SDK
# ============================================================================
set(XMP_ROOT "${CMAKE_CURRENT_SOURCE_DIR}/xmp/toolkit")
set(XMP_COMMON_DIR "${XMP_ROOT}/XMPCommon")
set(XMP_CORE_DIR "${XMP_ROOT}/XMPCore")
set(XMP_FILES_DIR "${XMP_ROOT}/XMPFiles")
set(XMP_PUBLIC_INCLUDE "${XMP_ROOT}/public/include")

# XMPCommon sources
file(GLOB XMP_COMMON_SOURCES
    "${XMP_COMMON_DIR}/source/*.cpp"
)

# XMPCore sources
file(GLOB XMP_CORE_SOURCES
    "${XMP_CORE_DIR}/source/*.cpp"
)

# XMPFiles sources - following official XMP SDK CMakeListsCommon.txt
file(GLOB XMP_FILES_SOURCES
    "${XMP_FILES_DIR}/source/*.cpp"
    "${XMP_FILES_DIR}/source/FileHandlers/*.cpp"
    "${XMP_FILES_DIR}/source/FormatSupport/*.cpp"
    "${XMP_FILES_DIR}/source/NativeMetadataSupport/*.cpp"
)

# Include subdirectories in FormatSupport (AIFF, WAVE, etc.)
file(GLOB_RECURSE XMP_FORMAT_SUPPORT_SUBDIR_SOURCES
    "${XMP_FILES_DIR}/source/FormatSupport/**/*.cpp"
)
list(APPEND XMP_FILES_SOURCES ${XMP_FORMAT_SUPPORT_SUBDIR_SOURCES})

# PluginHandler sources - need platform-specific handling
file(GLOB XMP_PLUGINHANDLER_SOURCES
    "${XMP_FILES_DIR}/source/PluginHandler/*.cpp"
)
# Remove all platform-specific OS_Utils files first
list(FILTER XMP_PLUGINHANDLER_SOURCES EXCLUDE REGEX ".*OS_Utils_.*\\.cpp$")
# Add back the correct one for the current platform
if(WIN32)
    list(APPEND XMP_PLUGINHANDLER_SOURCES "${XMP_FILES_DIR}/source/PluginHandler/OS_Utils_WIN.cpp")
elseif(APPLE)
    list(APPEND XMP_PLUGINHANDLER_SOURCES "${XMP_FILES_DIR}/source/PluginHandler/OS_Utils_Mac.cpp")
else()
    list(APPEND XMP_PLUGINHANDLER_SOURCES "${XMP_FILES_DIR}/source/PluginHandler/OS_Utils_Linux.cpp")
endif()
list(APPEND XMP_FILES_SOURCES ${XMP_PLUGINHANDLER_SOURCES})

# Additional XMP toolkit sources needed for XMPFiles
file(GLOB XMP_TOOLKIT_SOURCES
    "${XMP_ROOT}/source/*.cpp"
)

# Exclude platform-specific files
if(WIN32)
    list(FILTER XMP_TOOLKIT_SOURCES EXCLUDE REGEX ".*-Unix\\.cpp$")
    list(FILTER XMP_TOOLKIT_SOURCES EXCLUDE REGEX ".*-POSIX\\.cpp$")
    list(FILTER XMP_TOOLKIT_SOURCES EXCLUDE REGEX ".*-Mac\\.cpp$")
elseif(APPLE)
    list(FILTER XMP_TOOLKIT_SOURCES EXCLUDE REGEX ".*-Win\\.cpp$")
    list(FILTER XMP_TOOLKIT_SOURCES EXCLUDE REGEX ".*-Unix\\.cpp$")
else()
    list(FILTER XMP_TOOLKIT_SOURCES EXCLUDE REGEX ".*-Win\\.cpp$")
    list(FILTER XMP_TOOLKIT_SOURCES EXCLUDE REGEX ".*-Mac\\.cpp$")
endif()

list(APPEND XMP_FILES_SOURCES ${XMP_TOOLKIT_SOURCES})

# Add third-party MD5 implementation
file(GLOB XMP_THIRD_PARTY_SOURCES
    "${XMP_ROOT}/third-party/zuid/sources/*.cpp"
)
list(APPEND XMP_FILES_SOURCES ${XMP_THIRD_PARTY_SOURCES})

# XMP library (always static for proper linking)
add_library(xmp STATIC
    ${XMP_COMMON_SOURCES}
    ${XMP_CORE_SOURCES}
    ${XMP_FILES_SOURCES}
)

target_include_directories(xmp
    PRIVATE
        ${XMP_PUBLIC_INCLUDE}
        ${XMP_ROOT}
        ${XMP_COMMON_DIR}/source
        ${XMP_CORE_DIR}/source
        ${XMP_FILES_DIR}/source
        ${XMP_FILES_DIR}/source/FileHandlers
        ${XMP_FILES_DIR}/source/FormatSupport
        ${XMP_FILES_DIR}/source/NativeMetadataSupport
        ${XMP_FILES_DIR}/source/PluginHandler
        "${XMP_ROOT}/source"
        "${XMP_ROOT}/third-party/zlib"
        "${XMP_ROOT}/XMPFilesPlugins/api/source"
)

target_compile_definitions(xmp
    PRIVATE
        XML_STATIC
        TXMP_EXPAT_WRAPPER=1
        BUILDING_XMPCORE_LIB=1
        BUILDING_XMPCORE_AS_STATIC=1
        XMP_COMPONENT_INT_NAMESPACE=AdobeXMPCore_Int
        BUILDING_XMPCOMMON_LIB=1
        BUILDING_XMPCOMMON_AS_STATIC=1
        XMP_COMPONENT_XMPCOMMON=1
        AdobePrivate
        XMP_StaticBuild=1
)

# Link expat and zlib to xmp
target_link_libraries(xmp
    PUBLIC
        expat::expat
        ZLIB::ZLIB
)

# Platform-specific libraries for XMP
if(APPLE)
    target_link_libraries(xmp PUBLIC
        "-framework CoreFoundation"
        "-framework CoreServices"
    )
    # Fix IFF_RIFF vtable issues by ensuring all source files are included
    set_property(SOURCE "${XMP_FILES_DIR}/source/FormatSupport/IFF_RIFF.cpp" PROPERTY COMPILE_FLAGS "-fPIC")
elseif(WIN32)
    target_link_libraries(xmp PUBLIC
        ole32.lib
        shell32.lib
    )
endif()

# ============================================================================
# DNG SDK
# ============================================================================
set(DNG_SOURCE_DIR "${CMAKE_CURRENT_SOURCE_DIR}/dng_sdk/source")

# DNG SDK sources (exclude dng_validate.cpp)
file(GLOB DNG_SOURCES "${DNG_SOURCE_DIR}/*.cpp")
list(REMOVE_ITEM DNG_SOURCES "${DNG_SOURCE_DIR}/dng_validate.cpp")

# MinGW has native pthreads; exclude dng_pthread.cpp (MSVC fake pthreads)
# and provide a minimal dng_pthread_now() implementation instead.
if(MINGW)
    list(REMOVE_ITEM DNG_SOURCES "${DNG_SOURCE_DIR}/dng_pthread.cpp")
    file(WRITE "${CMAKE_CURRENT_BINARY_DIR}/dng_pthread_mingw.cpp"
"#include \"dng_pthread.h\"
#include <sys/time.h>
int dng_pthread_now (struct timespec *now) {
    if (now == NULL) return -1;
    struct timeval tv;
    if (gettimeofday (&tv, NULL) != 0) return errno;
    now->tv_sec  = tv.tv_sec;
    now->tv_nsec = tv.tv_usec * 1000;
    return 0;
}
")
    list(APPEND DNG_SOURCES "${CMAKE_CURRENT_BINARY_DIR}/dng_pthread_mingw.cpp")
endif()

# DNG library
add_library(dng ${DNG_SOURCES})

target_include_directories(dng
    PUBLIC
        $<BUILD_INTERFACE:${DNG_SOURCE_DIR}>
        $<INSTALL_INTERFACE:${CMAKE_INSTALL_INCLUDEDIR}>
    PRIVATE
        ${XMP_PUBLIC_INCLUDE}
        ${XMP_ROOT}
)

target_compile_definitions(dng
    PUBLIC
        qDNGVersion=0x0107001
        qDNGBuild=2502
        qDNGValidateTarget=0
        qDNGSupportJXL=1
    PRIVATE
        qDNGXMPFiles=1
        XMP_StaticBuild=1
)

if(WIN32)
    target_compile_definitions(dng PRIVATE _CRT_SECURE_NO_WARNINGS)
endif()

# Link dependencies to dng
target_link_libraries(dng
    PUBLIC
        xmp
        JPEG::JPEG
        PkgConfig::LIBJXL
        ZLIB::ZLIB
)

# Unix-specific libraries
if(UNIX)
    target_link_libraries(dng PUBLIC m c++)
endif()

# MinGW uses POSIX pthreads and needs ws2_32 for byte-order functions
if(MINGW)
    target_link_libraries(dng PUBLIC pthread ws2_32)
endif()

# Set library properties
set_target_properties(dng PROPERTIES
    VERSION ${PROJECT_VERSION}
    SOVERSION ${PROJECT_VERSION_MAJOR}
    OUTPUT_NAME "dng"
)

# ============================================================================
# dng_validate tool (optional)
# ============================================================================
if(DNG_BUILD_TOOLS)
    add_executable(dng_validate
        "${DNG_SOURCE_DIR}/dng_validate.cpp"
    )
    target_link_libraries(dng_validate PRIVATE dng)
endif()

# ============================================================================
# Installation
# ============================================================================

# Install DNG headers
file(GLOB DNG_HEADERS "${DNG_SOURCE_DIR}/*.h")
install(FILES ${DNG_HEADERS}
    DESTINATION "${CMAKE_INSTALL_INCLUDEDIR}"
)

# Install XMP public headers
file(GLOB_RECURSE XMP_PUBLIC_HEADERS "${XMP_PUBLIC_INCLUDE}/*.h" "${XMP_PUBLIC_INCLUDE}/*.hpp")
install(FILES ${XMP_PUBLIC_HEADERS}
    DESTINATION "${CMAKE_INSTALL_INCLUDEDIR}/xmp"
)

# Install libraries
install(TARGETS dng xmp
    EXPORT adobe-dng-sdkTargets
    RUNTIME DESTINATION ${CMAKE_INSTALL_BINDIR}
    LIBRARY DESTINATION ${CMAKE_INSTALL_LIBDIR}
    ARCHIVE DESTINATION ${CMAKE_INSTALL_LIBDIR}
)

# Install CMake config files
configure_package_config_file(
    "${CMAKE_CURRENT_SOURCE_DIR}/cmake/adobe-dng-sdkConfig.cmake.in"
    "${CMAKE_CURRENT_BINARY_DIR}/adobe-dng-sdkConfig.cmake"
    INSTALL_DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/adobe-dng-sdk
)

write_basic_package_version_file(
    "${CMAKE_CURRENT_BINARY_DIR}/adobe-dng-sdkConfigVersion.cmake"
    VERSION ${PROJECT_VERSION}
    COMPATIBILITY SameMajorVersion
)

install(FILES
    "${CMAKE_CURRENT_BINARY_DIR}/adobe-dng-sdkConfig.cmake"
    "${CMAKE_CURRENT_BINARY_DIR}/adobe-dng-sdkConfigVersion.cmake"
    DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/adobe-dng-sdk
)

install(EXPORT adobe-dng-sdkTargets
    NAMESPACE adobe::
    DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/adobe-dng-sdk
)

# Install dng_validate
if(DNG_BUILD_TOOLS)
    install(TARGETS dng_validate RUNTIME DESTINATION ${CMAKE_INSTALL_BINDIR})
endif()
