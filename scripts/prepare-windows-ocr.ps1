# Build-time preparation only. The installed application never downloads OCR components.
param(
    [ValidateSet('x64', 'arm64')]
    [string]$Architecture = 'x64'
)
$ErrorActionPreference = 'Stop'
$RepoRoot = Split-Path $PSScriptRoot -Parent
$VcpkgRevision = '9e3427bc82738568947beb508e78231f99c04f4c'
$ModelRevision = '87416418657359cb625c412a48b6e1d6d41c29bd'
$Triplet = "$Architecture-windows-static"
$CacheRoot = Join-Path $RepoRoot '.cache/windows-ocr'
$VcpkgRoot = Join-Path $CacheRoot "vcpkg-$VcpkgRevision"
$ModelRoot = Join-Path $RepoRoot 'src-tauri/assets/ocr/tessdata'
$LicenseRoot = Join-Path $RepoRoot 'src-tauri/assets/ocr/licenses'
$BinaryCache = Join-Path $CacheRoot 'binary-cache'
$TripletRoot = Join-Path $CacheRoot 'triplets'
New-Item -ItemType Directory -Force $CacheRoot, $ModelRoot, $LicenseRoot, $BinaryCache, $TripletRoot | Out-Null

function Invoke-Checked([string]$Command, [string[]]$Arguments) {
    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) { throw "$Command failed with exit code $LASTEXITCODE" }
}

if (-not (Test-Path (Join-Path $VcpkgRoot '.git'))) {
    Invoke-Checked git @('init', $VcpkgRoot)
    Invoke-Checked git @('-C', $VcpkgRoot, 'remote', 'add', 'origin', 'https://github.com/microsoft/vcpkg.git')
    Invoke-Checked git @('-C', $VcpkgRoot, 'fetch', '--depth', '1', 'origin', $VcpkgRevision)
    Invoke-Checked git @('-C', $VcpkgRoot, 'checkout', '--detach', 'FETCH_HEAD')
}
$ActualRevision = (& git -C $VcpkgRoot rev-parse HEAD).Trim()
if ($ActualRevision -ne $VcpkgRevision) { throw 'Unexpected vcpkg revision in OCR build cache' }
Invoke-Checked (Join-Path $VcpkgRoot 'bootstrap-vcpkg.bat') @('-disableMetrics')
$env:CARGO_BUILD_TARGET = if ($Architecture -eq 'arm64') { 'aarch64-pc-windows-msvc' } else { 'x86_64-pc-windows-msvc' }
$env:STATIC_VCRUNTIME = 'false'
$env:VCPKG_ROOT = $VcpkgRoot
$env:VCPKGRS_TRIPLET = $Triplet
$env:VCPKG_DEFAULT_BINARY_CACHE = $BinaryCache
Remove-Item Env:VCPKGRS_DYNAMIC -ErrorAction SilentlyContinue
# vcpkg-rs links installed/<triplet>/lib for both Cargo profiles. Build those
# release libraries only; the unused native debug libraries double cold CI time.
$BaseTriplet = Join-Path $VcpkgRoot "triplets/$Triplet.cmake"
if (-not (Test-Path $BaseTriplet)) { $BaseTriplet = Join-Path $VcpkgRoot "triplets/community/$Triplet.cmake" }
if (-not (Test-Path $BaseTriplet)) { throw "Unsupported pinned vcpkg triplet: $Triplet" }
$TripletContent = (Get-Content $BaseTriplet -Raw) + "`nset(VCPKG_BUILD_TYPE release)`n"
Set-Content (Join-Path $TripletRoot "$Triplet.cmake") $TripletContent -Encoding utf8
Invoke-Checked (Join-Path $VcpkgRoot 'vcpkg.exe') @('install', "tesseract:$Triplet", "--overlay-triplets=$TripletRoot", '--disable-metrics')

$Models = @{
    'eng' = '7d4322bd2a7749724879683fc3912cb542f19906c83bcc1a52132556427170b2'
    'chi_sim' = 'a5fcb6f0db1e1d6d8522f39db4e848f05984669172e584e8d76b6b3141e1f730'
}
foreach ($Language in $Models.Keys) {
    $Destination = Join-Path $ModelRoot "$Language.traineddata"
    if (-not (Test-Path $Destination)) {
        $Temporary = "$Destination.download"
        Invoke-WebRequest "https://raw.githubusercontent.com/tesseract-ocr/tessdata_fast/$ModelRevision/$Language.traineddata" -OutFile $Temporary
        $ActualHash = (Get-FileHash $Temporary -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($ActualHash -ne $Models[$Language]) { throw "OCR model checksum mismatch: $Language" }
        Move-Item $Temporary $Destination -Force
    }
    $ActualHash = (Get-FileHash $Destination -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($ActualHash -ne $Models[$Language]) { throw "OCR model checksum mismatch: $Language" }
}
# Keep all redistributable library copyright notices with the installed model assets.
$Share = Join-Path $VcpkgRoot "installed/$Triplet/share"
Get-ChildItem $Share -Directory | ForEach-Object {
    $CopyrightFile = Join-Path $_.FullName 'copyright'
    if (Test-Path $CopyrightFile) { Copy-Item $CopyrightFile (Join-Path $LicenseRoot "$($_.Name).txt") -Force }
}
Invoke-WebRequest "https://raw.githubusercontent.com/tesseract-ocr/tessdata_fast/$ModelRevision/LICENSE" -OutFile (Join-Path $LicenseRoot 'tessdata-fast.txt')
if ($env:GITHUB_ENV) {
    "CARGO_BUILD_TARGET=$env:CARGO_BUILD_TARGET" | Out-File $env:GITHUB_ENV -Append -Encoding utf8
    "STATIC_VCRUNTIME=false" | Out-File $env:GITHUB_ENV -Append -Encoding utf8
    "VCPKG_ROOT=$VcpkgRoot" | Out-File $env:GITHUB_ENV -Append -Encoding utf8
    "VCPKGRS_TRIPLET=$Triplet" | Out-File $env:GITHUB_ENV -Append -Encoding utf8
    "VCPKG_DEFAULT_BINARY_CACHE=$BinaryCache" | Out-File $env:GITHUB_ENV -Append -Encoding utf8
}
Write-Output "Prepared local Windows OCR dependencies for $Triplet (English + Simplified Chinese)."
