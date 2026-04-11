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

# Add ENABLE_RAWSPEED3 option and support (avoids fragile patch)
file(READ "${SOURCE_PATH}/CMakeLists.txt" CMAKE_CONTENT)
string(REPLACE
    "option(ENABLE_RAWSPEED             \"Build library with extra RawSpeed codec support (default=OFF)\"                OFF)"
    "option(ENABLE_RAWSPEED             \"Build library with extra RawSpeed codec support (default=OFF)\"                OFF)\noption(ENABLE_RAWSPEED3            \"Build library with RawSpeed v3 codec support  (default=OFF)\"                OFF)"
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
    string(APPEND _RAW_PC_REQUIRE " libxml2")
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
vcpkg_install_copyright(FILE_LIST
    "${SOURCE_PATH}/COPYRIGHT"
    "${SOURCE_PATH}/LICENSE.LGPL"
    "${SOURCE_PATH}/LICENSE.CDDL"
)
