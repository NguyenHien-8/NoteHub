param(
    [string]$QtDir = $env:QT_ROOT,
    [string]$Version = "0.3.0"
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
$Dist = Join-Path $Root "dist/windows"
$Build = Join-Path $Root "build/qt-windows"
New-Item -ItemType Directory -Force -Path $Dist | Out-Null

Write-Host "[1/4] Building Go backend..." -ForegroundColor Cyan
$ldflags = "-s -w -X main.version=$Version"
go build -trimpath -ldflags $ldflags -o (Join-Path $Dist "notehub-backend.exe") ./cmd/notehub-backend

if (-not $QtDir) {
    $candidates = @()
    if (Test-Path "C:\Qt") {
        $candidates += Get-ChildItem "C:\Qt" -Directory -ErrorAction SilentlyContinue |
            Where-Object { $_.Name -match '^6\.' } |
            Sort-Object Name -Descending |
            ForEach-Object {
                Join-Path $_.FullName "msvc2022_64"
                Join-Path $_.FullName "msvc2022_arm64"
                Join-Path $_.FullName "mingw_64"
            }
    }
    $QtDir = $candidates | Where-Object { Test-Path (Join-Path $_ "bin\windeployqt.exe") } | Select-Object -First 1
}
if (-not $QtDir) {
    throw "Qt 6 was not found. Set QT_ROOT, e.g. C:\Qt\6.11.2\msvc2022_64"
}
$QtDir = (Resolve-Path $QtDir).Path
Write-Host "Qt: $QtDir"

$generatorArgs = @()
if ($QtDir -match 'msvc2022_arm64') {
    $generatorArgs = @("-G", "Visual Studio 17 2022", "-A", "ARM64")
} elseif ($QtDir -match 'msvc') {
    $generatorArgs = @("-G", "Visual Studio 17 2022", "-A", "x64")
} elseif ($QtDir -match 'mingw') {
    $ninja = Get-Command ninja.exe -ErrorAction SilentlyContinue
    if (-not $ninja -and (Test-Path "C:\Qt\Tools\Ninja\ninja.exe")) {
        $env:PATH = "C:\Qt\Tools\Ninja;$env:PATH"
        $ninja = Get-Command ninja.exe -ErrorAction SilentlyContinue
    }
    $mingwRoot = Get-ChildItem "C:\Qt\Tools" -Directory -Filter "mingw*" -ErrorAction SilentlyContinue | Sort-Object Name -Descending | Select-Object -First 1
    if ($mingwRoot) { $env:PATH = "$(Join-Path $mingwRoot.FullName 'bin');$env:PATH" }
    if ($ninja) { $generatorArgs = @("-G", "Ninja") }
    else { $generatorArgs = @("-G", "MinGW Makefiles") }
}

Write-Host "[2/4] Configuring Qt frontend..." -ForegroundColor Cyan
cmake -S $Root -B $Build @generatorArgs "-DCMAKE_PREFIX_PATH=$QtDir" "-DCMAKE_BUILD_TYPE=Release"

Write-Host "[3/4] Building Qt frontend..." -ForegroundColor Cyan
cmake --build $Build --config Release --parallel
$exeCandidates = @(
    (Join-Path $Build "frontend/Release/NoteHub.exe"),
    (Join-Path $Build "frontend/NoteHub.exe"),
    (Join-Path $Build "Release/NoteHub.exe")
)
$GuiExe = $exeCandidates | Where-Object { Test-Path $_ } | Select-Object -First 1
if (-not $GuiExe) { throw "NoteHub.exe was not found under $Build" }
Copy-Item $GuiExe (Join-Path $Dist "NoteHub.exe") -Force

Write-Host "[4/4] Deploying Qt runtime..." -ForegroundColor Cyan
$Deploy = Join-Path $QtDir "bin/windeployqt.exe"
& $Deploy --release --no-translations --compiler-runtime (Join-Path $Dist "NoteHub.exe")
Copy-Item (Join-Path $Root "assets/icons/NoteHub.png") (Join-Path $Dist "NoteHub.png") -Force

Write-Host "Done: $Dist\NoteHub.exe" -ForegroundColor Green
