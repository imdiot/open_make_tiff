$ErrorActionPreference = "Stop"

# Read productVersion from wails.json (ConvertFrom-Json — no reliance on make's
# grep/sed shell). Invoked by the Makefile `package-windows` target from the
# project root, so wails.json is in the CWD.
$version = (Get-Content "wails.json" -Raw | ConvertFrom-Json).info.productVersion

$binDir = "build/bin"
$exe = Join-Path $binDir "OpenMakeTiff.exe"
$thirdParty = Join-Path $binDir "third-party"
if (-not (Test-Path $exe)) {
    throw "Build output not found: $exe (run 'make build-windows' first)"
}
if (-not (Test-Path $thirdParty)) {
    throw "third-party/ not found: $thirdParty (postBuildHook 'copy-third-party.ps1' failed?)"
}

# Compress from build/bin so OpenMakeTiff.exe and third-party/ are siblings at
# the zip root — the runtime resolves exiftool via filepath.Dir(self)/third-party/.
Push-Location $binDir
try {
    # Remove prior zip first; -Force also overwrites, explicit delete avoids the
    # append quirk in older PowerShell.
    Remove-Item "OpenMakeTiff-*-windows-x64.zip" -Force -ErrorAction SilentlyContinue
    $out = "OpenMakeTiff-$version-windows-x64.zip"
    Compress-Archive -Path "OpenMakeTiff.exe", "third-party" -DestinationPath $out -Force
    Write-Host "packaged: $binDir/$out"
} finally {
    Pop-Location
}
