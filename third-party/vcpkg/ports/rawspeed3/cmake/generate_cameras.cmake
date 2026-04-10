# generate_cameras.cmake
# Converts cameras.xml into a C string literal (cameras.cpp)
# Uses execute_process to call rsxml2c.sh for reliable conversion

execute_process(
    COMMAND "${CMAKE_COMMAND}" -E env
        bash "${RSXML2C_SH}"
        "<" "${CAMERAS_XML}"
    OUTPUT_FILE "${CAMERAS_CPP}"
    RESULT_VARIABLE EXIT_CODE
    OUTPUT_VARIABLE OUTPUT
    ERROR_VARIABLE ERROR_OUTPUT
)

if(NOT EXIT_CODE EQUAL 0)
    message(FATAL_ERROR "Failed to generate cameras.cpp: ${ERROR_OUTPUT}")
endif()
