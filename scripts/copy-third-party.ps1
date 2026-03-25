$ErrorActionPreference = "Stop"

Remove-Item -Path ./third-party -Recurse -Force -ErrorAction SilentlyContinue
Copy-Item ../../third-party/windows-x64 -Destination ./third-party -Recurse -Force
