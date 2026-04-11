# generate_cameras_cpp.cmake
# Reads an XML file and generates a C source file embedding it as a string literal.
# Usage: cmake -DINPUT=<xml_file> -DOUTPUT=<cpp_file> -P generate_cameras_cpp.cmake

file(READ "${INPUT}" XML_CONTENT)

# Remove carriage returns
string(REPLACE "\r" "" XML_CONTENT "${XML_CONTENT}")

# Escape backslashes first
string(REPLACE "\\" "\\\\" XML_CONTENT "${XML_CONTENT}")

# Escape double quotes
string(REPLACE "\"" "\\\"" XML_CONTENT "${XML_CONTENT}")

# Replace two leading spaces with tab (mimic original sed: s/  /\\t/g)
set(TAB "\t")
string(REPLACE "  " "${TAB}" XML_CONTENT "${XML_CONTENT}")

# Split into lines and wrap each one
string(REPLACE "\n" ";" XML_LINES "${XML_CONTENT}")

set(OUTPUT_LINES "")
foreach(LINE IN LISTS XML_LINES)
    if(NOT LINE STREQUAL "")
        string(APPEND OUTPUT_LINES "\"${LINE}\\n\"\n")
    endif()
endforeach()

set(OUTPUT_CONTENT "const char *_rawspeed3_data_xml=\n${OUTPUT_LINES}\"\\0\";\n")

file(WRITE "${OUTPUT}" "${OUTPUT_CONTENT}")
