vcpkg_from_github(
    OUT_SOURCE_PATH SOURCE_PATH
    REPO darktable-org/rawspeed
    REF de70ef5fbc62cde91009c8cff7a206272abe631e
    SHA512 0305bb219f7dc98b99f75039086495ddab1230b12f77e2759772531d95dbdd02ac36dc3f0e84ea1631a3ee13c1c9dbaed3bfad632ff383ebd62d36427da727bb
    PATCHES
        patches/01.CameraMeta-extensibility.patch
        patches/02.Makernotes-processing.patch
        patches/03.remove-limits-and-logging.patch
        patches/04.clang-cl-compatibility.patch
        patches/05.no-phase-one-correction.patch
)

# Replace upstream CMake with our custom build
file(COPY "${CMAKE_CURRENT_LIST_DIR}/CMakeLists.txt" DESTINATION "${SOURCE_PATH}")
file(COPY "${CMAKE_CURRENT_LIST_DIR}/cmake" DESTINATION "${SOURCE_PATH}")
file(COPY "${CMAKE_CURRENT_LIST_DIR}/rawspeed3_c_api" DESTINATION "${SOURCE_PATH}")

vcpkg_cmake_configure(
    SOURCE_PATH "${SOURCE_PATH}"
)

vcpkg_cmake_install()
vcpkg_cmake_config_fixup(CONFIG_PATH "lib/cmake/rawspeed3")
vcpkg_fixup_pkgconfig()

file(REMOVE_RECURSE
    "${CURRENT_PACKAGES_DIR}/debug/include"
    "${CURRENT_PACKAGES_DIR}/debug/share"
)

vcpkg_install_copyright(FILE_LIST "${SOURCE_PATH}/LICENSE")
file(INSTALL "${CMAKE_CURRENT_LIST_DIR}/usage"
    DESTINATION "${CURRENT_PACKAGES_DIR}/share/${PORT}")
