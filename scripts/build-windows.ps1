param(
    [string]$Version = '0.3.0',
    [string]$QtRoot = $env:QT_ROOT_DIR,
    [string]$Compiler = 'C:\Qt\Tools\mingw1310_64\bin\g++.exe',
    [string]$OutputDirectory = 'dist\windows'
)
$ErrorActionPreference = 'Stop'
if ($Version -notmatch '^\d+\.\d+\.\d+$') { throw 'Version must be X.Y.Z' }
$repoRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
if (-not $QtRoot) {
    $QtRoot = Get-ChildItem -LiteralPath 'C:\Qt' -Directory -ErrorAction SilentlyContinue |
        Where-Object { $_.Name -match '^6\.\d+\.\d+$' } |
        Sort-Object { [version]$_.Name } -Descending |
        ForEach-Object { Join-Path $_.FullName 'mingw_64' } |
        Where-Object { Test-Path -LiteralPath (Join-Path $_ 'bin\windeployqt.exe') } |
        Select-Object -First 1
}
if (-not $QtRoot -or -not (Test-Path -LiteralPath (Join-Path $QtRoot 'lib\cmake\Qt6\Qt6Config.cmake'))) { throw 'Provide -QtRoot pointing to a Qt 6.5+ MinGW kit.' }
if (-not (Test-Path -LiteralPath $Compiler)) { throw 'Provide -Compiler pointing to g++.exe matching your Qt kit.' }
$cmake = (Get-Command cmake -ErrorAction Stop).Source
$ninja = 'C:\Qt\Tools\Ninja\ninja.exe'
if (-not (Test-Path -LiteralPath $ninja)) { $ninja = (Get-Command ninja -ErrorAction Stop).Source }
$outputPath = [System.IO.Path]::GetFullPath((Join-Path $repoRoot $OutputDirectory))
$buildPath = Join-Path $repoRoot 'build\qt-release'
$previousPath = $env:PATH
try {
    $env:PATH = (Split-Path $Compiler -Parent) + ';' + (Join-Path $QtRoot 'bin') + ';' + $env:PATH
    & $cmake -S $repoRoot -B $buildPath -G Ninja "-DCMAKE_MAKE_PROGRAM=$ninja" "-DCMAKE_CXX_COMPILER=$Compiler" "-DCMAKE_PREFIX_PATH=$QtRoot" "-DNOTEHUB_VERSION=$Version" -DCMAKE_BUILD_TYPE=Release -DBUILD_TESTING=OFF
    if ($LASTEXITCODE -ne 0) { throw 'CMake configuration failed' }
    & $cmake --build $buildPath --target NoteHub
    if ($LASTEXITCODE -ne 0) { throw 'Qt/Go build failed' }
    & $cmake --install $buildPath --prefix $outputPath
    if ($LASTEXITCODE -ne 0) { throw 'Qt runtime deployment failed' }
    Copy-Item -LiteralPath (Join-Path $repoRoot 'LICENSE') -Destination $outputPath
    Write-Host "Ready: $outputPath\NoteHub.exe (keep notehub-core.exe and Qt runtime files alongside it)"
} finally { $env:PATH = $previousPath }
