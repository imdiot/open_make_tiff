cmake_minimum_required(VERSION 3.15)
project(rawspeed3 CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)

include(GNUInstallDirs)
include(CMakePackageConfigHelpers)

# ============================================================================
# Dependencies
# ============================================================================
find_package(pugixml CONFIG REQUIRED)
find_package(JPEG REQUIRED)
find_package(ZLIB REQUIRED)

# ============================================================================
# Generate rawspeedconfig.h (replaces upstream CMake configure_file)
# ============================================================================
include(CheckSymbolExists)
include(CheckCXXSourceCompiles)

set(RAWSPEED_CACHELINESIZE 64)
set(RAWSPEED_PAGESIZE 4096)

# Check memory alignment functions
unset(HAVE_ALIGNED_ALLOC CACHE)
unset(HAVE_POSIX_MEMALIGN CACHE)
unset(HAVE_MM_MALLOC CACHE)
check_symbol_exists(aligned_alloc "stdlib.h" HAVE_ALIGNED_ALLOC)
if(NOT HAVE_ALIGNED_ALLOC)
  check_symbol_exists(posix_memalign "stdlib.h" HAVE_POSIX_MEMALIGN)
endif()
if(NOT HAVE_POSIX_MEMALIGN)
  check_symbol_exists(_mm_malloc "xmmintrin.h" HAVE_MM_MALLOC)
endif()
check_symbol_exists(_aligned_malloc "malloc.h" HAVE_ALIGNED_MALLOC)
if(HAVE_ALIGNED_MALLOC)
  check_symbol_exists(_aligned_free "malloc.h" HAVE_ALIGNED_FREE)
  if(NOT HAVE_ALIGNED_FREE)
    unset(HAVE_ALIGNED_MALLOC CACHE)
  endif()
endif()

# Check thread_local support
check_cxx_source_compiles("
  thread_local int x = 0;
  int main() { x = 1; return x; }
" HAVE_CXX_THREAD_LOCAL)
if(NOT HAVE_CXX_THREAD_LOCAL)
  check_cxx_source_compiles("
    __thread int x = 0;
    int main() { x = 1; return x; }
  " HAVE_GCC_THREAD_LOCAL)
endif()

# Check HAVE_JPEG_MEM_SRC (libjpeg-turbo extension)
check_symbol_exists(jpeg_mem_src "jpeglib.h" HAVE_JPEG_MEM_SRC)

# Write rawspeedconfig.h
configure_file(
  "${CMAKE_CURRENT_SOURCE_DIR}/cmake/rawspeedconfig.h.in"
  "${CMAKE_CURRENT_BINARY_DIR}/rawspeedconfig.h"
)

# ============================================================================
# RawSpeed library sources
# ============================================================================
set(RAWSPEED_SRC_DIR "${CMAKE_CURRENT_SOURCE_DIR}/src/librawspeed")

file(GLOB_RECURSE RAWSPEED_SRCS "${RAWSPEED_SRC_DIR}/*.cpp")

# Exclude FileReader/FileWriter (not needed for LibRaw integration)
list(FILTER RAWSPEED_SRCS EXCLUDE REGEX "FileReader\\.cpp$")
list(FILTER RAWSPEED_SRCS EXCLUDE REGEX "FileWriter\\.cpp$")
list(FILTER RAWSPEED_SRCS EXCLUDE REGEX "ChecksumFile\\.cpp$")

add_library(rawspeed STATIC ${RAWSPEED_SRCS})

target_include_directories(rawspeed
    PUBLIC
        "$<BUILD_INTERFACE:${CMAKE_CURRENT_BINARY_DIR}>"
        "$<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/src/external>"
    PRIVATE
        "${RAWSPEED_SRC_DIR}"
)

target_compile_definitions(rawspeed PRIVATE
    HAVE_PUGIXML
    HAVE_JPEG
    HAVE_ZLIB
)

if(HAVE_JPEG_MEM_SRC)
  target_compile_definitions(rawspeed PRIVATE HAVE_JPEG_MEM_SRC)
endif()

target_link_libraries(rawspeed
    PUBLIC
        pugixml::pugixml
    PRIVATE
        JPEG::JPEG
        ZLIB::ZLIB
)

# Platform-specific settings
if(APPLE)
  target_compile_definitions(rawspeed PRIVATE qMacOS=1)
  # Ensure correct deployment target for universal builds
  if(NOT CMAKE_OSX_DEPLOYMENT_TARGET)
    set_target_properties(rawspeed PROPERTIES
        COMPILE_FLAGS "-mmacosx-version-min=10.13"
    )
  endif()
elseif(WIN32)
  target_compile_definitions(rawspeed PRIVATE qWinOS=1 NOMINMAX=1)
else()
  target_compile_definitions(rawspeed PRIVATE qLinux=1)
endif()

# Suppress warnings in third-party code
if(MSVC)
  target_compile_options(rawspeed PRIVATE /W0)
else()
  target_compile_options(rawspeed PRIVATE -w)
endif()

# ============================================================================
# Generate cameras.cpp from cameras.xml
# ============================================================================
set(CAMERAS_XML "${CMAKE_CURRENT_SOURCE_DIR}/data/cameras.xml")
set(CAMERAS_CPP "${CMAKE_CURRENT_BINARY_DIR}/cameras.cpp")

# Embed cameras.xml as a C string literal
add_custom_command(
    OUTPUT "${CAMERAS_CPP}"
    COMMAND "${CMAKE_COMMAND}"
        -DINPUT="${CAMERAS_XML}"
        -DOUTPUT="${CAMERAS_CPP}"
        -P "${CMAKE_CURRENT_SOURCE_DIR}/cmake/generate_cameras_cpp.cmake"
    DEPENDS "${CAMERAS_XML}" "${CMAKE_CURRENT_SOURCE_DIR}/cmake/generate_cameras_cpp.cmake"
    COMMENT "Generating cameras.cpp from cameras.xml"
)

# ============================================================================
# C-API wrapper library (rawspeed3)
# ============================================================================
set(CAPI_SRC_DIR "${CMAKE_CURRENT_SOURCE_DIR}/rawspeed3_c_api")

add_library(rawspeed3 STATIC
    "${CAPI_SRC_DIR}/rawspeed3_capi.cpp"
    "${CAMERAS_CPP}"
)

target_include_directories(rawspeed3
    PUBLIC
        "$<BUILD_INTERFACE:${CAPI_SRC_DIR}>"
        "$<INSTALL_INTERFACE:${CMAKE_INSTALL_INCLUDEDIR}>"
    PRIVATE
        "$<BUILD_INTERFACE:${RAWSPEED_SRC_DIR}>"
)

target_link_libraries(rawspeed3
    PUBLIC rawspeed
)

# ============================================================================
# Installation
# ============================================================================
install(FILES "${CAPI_SRC_DIR}/rawspeed3_capi.h"
    DESTINATION "${CMAKE_INSTALL_INCLUDEDIR}"
)

install(TARGETS rawspeed rawspeed3
    EXPORT rawspeed3Targets
    ARCHIVE DESTINATION "${CMAKE_INSTALL_LIBDIR}"
    RUNTIME DESTINATION "${CMAKE_INSTALL_BINDIR}"
)

install(EXPORT rawspeed3Targets
    NAMESPACE rawspeed3::
    DESTINATION "${CMAKE_INSTALL_LIBDIR}/cmake/rawspeed3"
)

configure_package_config_file(
    "${CMAKE_CURRENT_SOURCE_DIR}/cmake/rawspeed3Config.cmake.in"
    "${CMAKE_CURRENT_BINARY_DIR}/rawspeed3Config.cmake"
    INSTALL_DESTINATION "${CMAKE_INSTALL_LIBDIR}/cmake/rawspeed3"
)

write_basic_package_version_file(
    "${CMAKE_CURRENT_BINARY_DIR}/rawspeed3ConfigVersion.cmake"
    VERSION 3.0.0
    COMPATIBILITY SameMajorVersion
)

install(FILES
    "${CMAKE_CURRENT_BINARY_DIR}/rawspeed3Config.cmake"
    "${CMAKE_CURRENT_BINARY_DIR}/rawspeed3ConfigVersion.cmake"
    DESTINATION "${CMAKE_INSTALL_LIBDIR}/cmake/rawspeed3"
)

# ============================================================================
# Pkg-config
# ============================================================================
set(prefix "${CMAKE_INSTALL_PREFIX}")
set(exec_prefix "\${prefix}")
set(includedir "\${prefix}/include")
set(libdir "\${exec_prefix}/lib")
set(VERSION "3.0.0")

configure_file(
    "${CMAKE_CURRENT_SOURCE_DIR}/cmake/rawspeed3.pc.in"
    "${CMAKE_CURRENT_BINARY_DIR}/rawspeed3.pc"
    @ONLY
)

install(FILES "${CMAKE_CURRENT_BINARY_DIR}/rawspeed3.pc"
    DESTINATION "${CMAKE_INSTALL_LIBDIR}/pkgconfig"
)
