param(
    [Parameter(Mandatory)]
    [string]$Version
)

$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path $PSScriptRoot -Parent
$OutputDir = Join-Path $ProjectRoot "third-party\windows-x64"
$MarkerFile = Join-Path $OutputDir ".exiftool-version"

# Check if already downloaded with correct version
if (Test-Path $MarkerFile) {
    $existingVersion = (Get-Content $MarkerFile -Raw).Trim()
    if ($existingVersion -eq $Version) {
        Write-Host "ExifTool $Version already present for Windows"
        exit 0
    }
}

Write-Host "Downloading ExifTool $Version for Windows..."

# Clean output directory
if (Test-Path $OutputDir) {
    Remove-Item $OutputDir -Recurse -Force
}
New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null

# Download
$url = "https://sourceforge.net/projects/exiftool/files/exiftool-$($Version)_64.zip/download"
$tempZip = Join-Path $env:TEMP "exiftool-win.zip"
$tempExtract = Join-Path $env:TEMP "exiftool-win"

try {
    Write-Host "  Downloading from SourceForge..."
    curl.exe -L -o $tempZip $url
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Download failed with exit code $LASTEXITCODE"
        exit 1
    }
    if (-not (Test-Path $tempZip) -or (Get-Item $tempZip).Length -lt 1MB) {
        Write-Error "Download failed or file too small"
        exit 1
    }

    # Extract
    if (Test-Path $tempExtract) { Remove-Item $tempExtract -Recurse -Force }
    Expand-Archive -Path $tempZip -DestinationPath $tempExtract -Force

    # Move contents up (zip contains exiftool-{version}_64/ subdirectory)
    $subDir = Get-ChildItem -Path $tempExtract -Directory | Select-Object -First 1
    Get-ChildItem -Path $subDir.FullName | Move-Item -Destination $OutputDir -Force

    # Rename exiftool(-k).exe to exiftool.exe
    $oldExe = Join-Path $OutputDir "exiftool(-k).exe"
    $newExe = Join-Path $OutputDir "exiftool.exe"
    if (Test-Path $oldExe) {
        Move-Item $oldExe $newExe -Force
    }

    # Write version marker
    Set-Content -Path $MarkerFile -Value $Version

    Write-Host "ExifTool $Version downloaded for Windows"
}
finally {
    Remove-Item $tempZip -Force -ErrorAction SilentlyContinue
    Remove-Item $tempExtract -Recurse -Force -ErrorAction SilentlyContinue
}
