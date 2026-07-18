param(
    [Parameter(Mandatory)]
    [string]$Ref
)

$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path $PSScriptRoot -Parent
$VcpkgRoot = Join-Path $ProjectRoot "third-party\vcpkg-root"
$ExeFile = Join-Path $VcpkgRoot "vcpkg.exe"
$MarkerFile = Join-Path $VcpkgRoot ".vcpkg-ref"

# Skip if already cloned + bootstrapped with the matching ref
if (Test-Path $MarkerFile) {
    $existingRef = (Get-Content $MarkerFile -Raw).Trim()
    if ($existingRef -eq $Ref -and (Test-Path $ExeFile)) {
        Write-Host "vcpkg ($Ref) already present at $VcpkgRoot"
        exit 0
    }
}

Write-Host "Cloning microsoft/vcpkg @ $Ref into $VcpkgRoot ..."
if (Test-Path $VcpkgRoot) { Remove-Item $VcpkgRoot -Recurse -Force }
git clone --depth 1 --branch $Ref https://github.com/microsoft/vcpkg.git $VcpkgRoot
if ($LASTEXITCODE -ne 0) {
    # $Ref might be a bare commit, not a branch/tag — shallow clone then fetch
    Write-Host "  (trying commit-based fetch...)"
    git clone --depth 1 https://github.com/microsoft/vcpkg.git $VcpkgRoot
    if ($LASTEXITCODE -ne 0) { Write-Error "git clone failed"; exit 1 }
    Push-Location $VcpkgRoot
    git fetch --depth 1 origin $Ref
    if ($LASTEXITCODE -ne 0) { Write-Error "git fetch $Ref failed"; Pop-Location; exit 1 }
    git checkout $Ref
    if ($LASTEXITCODE -ne 0) { Write-Error "git checkout $Ref failed"; Pop-Location; exit 1 }
    Pop-Location
}

Write-Host "Bootstrapping vcpkg-tool ..."
Push-Location $VcpkgRoot
try {
    .\bootstrap-vcpkg.bat -disableMetrics
    if ($LASTEXITCODE -ne 0) { Write-Error "bootstrap-vcpkg failed"; exit 1 }
} finally {
    Pop-Location
}

Set-Content -Path $MarkerFile -Value $Ref
Write-Host "vcpkg ($Ref) ready at $VcpkgRoot"
