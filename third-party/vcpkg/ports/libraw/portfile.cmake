vcpkg_download_distfile(ARCHIVE
    URLS "https://github.com/LibRaw/LibRaw/archive/ae32c08bfe2374c651176ecf32e07a3751697488.zip"
    FILENAME "LibRaw-ae32c08bfe2374c651176ecf32e07a3751697488.zip"
    SHA512 6c4088adf2da4f5771bf5c68d0d23e02223f70c34c8552d6466a4dd55f9a2e582aabfab3c3c13f05b489bdaa9df1a4249cc1202952e24e547de7698c57cfa91c
)

vcpkg_extract_source_archive(
    SOURCE_PATH
    ARCHIVE "${ARCHIVE}"
)

vcpkg_from_github(
    OUT_SOURCE_PATH LIBRAW_CMAKE_SOURCE_PATH
    REPO LibRaw/LibRaw-cmake
    REF eb98e4325aef2ce85d2eb031c2ff18640ca616d3
    SHA512 63e68a4d30286ec3aa97168d46b7a1199268099ae27b61abcc92e93ec30e48d364086227983a1d724415e5f4da44d905422f30192453b95f31040e5f8469c3f9
    HEAD_REF master
    PATCHES
        dependencies.patch
        # Move the non-thread-safe library to manual-link. This is unfortunately needed
        # because otherwise libraries that build on top of libraw have to choose.
        fix-install.patch
)

file(COPY "${LIBRAW_CMAKE_SOURCE_PATH}/CMakeLists.txt" DESTINATION "${SOURCE_PATH}")
file(COPY "${LIBRAW_CMAKE_SOURCE_PATH}/cmake" DESTINATION "${SOURCE_PATH}")


vcpkg_check_features(OUT_FEATURE_OPTIONS FEATURE_OPTIONS
    FEATURES
        openmp      ENABLE_OPENMP
        openmp      CMAKE_REQUIRE_FIND_PACKAGE_OpenMP
        dng-lossy   CMAKE_REQUIRE_FIND_PACKAGE_JPEG
)

vcpkg_cmake_configure(
    SOURCE_PATH "${SOURCE_PATH}"
    OPTIONS
        ${FEATURE_OPTIONS}
        -DENABLE_EXAMPLES=ON
        -DCMAKE_REQUIRE_FIND_PACKAGE_Jasper=1
        -DCMAKE_REQUIRE_FIND_PACKAGE_ZLIB=1
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

configure_file("${CMAKE_CURRENT_LIST_DIR}/vcpkg-cmake-wrapper.cmake" "${CURRENT_PACKAGES_DIR}/share/${PORT}/vcpkg-cmake-wrapper.cmake" @ONLY)
file(INSTALL "${CMAKE_CURRENT_LIST_DIR}/usage" DESTINATION "${CURRENT_PACKAGES_DIR}/share/${PORT}")
vcpkg_install_copyright(FILE_LIST
    "${SOURCE_PATH}/COPYRIGHT"
    "${SOURCE_PATH}/LICENSE.LGPL"
    "${SOURCE_PATH}/LICENSE.CDDL"
)
