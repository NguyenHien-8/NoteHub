param(
    [string]$QtRoot = $env:QT_ROOT_DIR,
    [string]$Compiler = 'C:\Qt\Tools\mingw1310_64\bin\g++.exe',
    [string]$AppDirectory = 'dist\windows'
)

$ErrorActionPreference = 'Stop'

function Invoke-Native {
    param(
        [Parameter(Mandatory = $true)][string]$FilePath,
        [Parameter(ValueFromRemainingArguments = $true)][string[]]$Arguments
    )
    & $FilePath @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed with exit code $LASTEXITCODE`: $FilePath $($Arguments -join ' ')"
    }
}

$repoRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
if (-not $QtRoot) {
    $QtRoot = Get-ChildItem -LiteralPath 'C:\Qt' -Directory -ErrorAction SilentlyContinue |
        Where-Object { $_.Name -match '^6\.\d+\.\d+$' } |
        Sort-Object { [version]$_.Name } -Descending |
        ForEach-Object { Join-Path $_.FullName 'mingw_64' } |
        Where-Object { Test-Path -LiteralPath (Join-Path $_ 'bin\windeployqt.exe') } |
        Select-Object -First 1
}
if (-not $QtRoot) { throw 'Qt MinGW kit not found. Pass -QtRoot.' }
if (-not (Test-Path -LiteralPath $Compiler)) { throw 'MinGW compiler not found. Pass -Compiler.' }

$appPath = [System.IO.Path]::GetFullPath((Join-Path $repoRoot $AppDirectory))
$exe = Join-Path $appPath 'NoteHub.exe'
$core = Join-Path $appPath 'notehub-core.exe'
$windeployqt = Join-Path $QtRoot 'bin\windeployqt.exe'

if (-not (Test-Path -LiteralPath $exe)) {
    throw "NoteHub.exe not found in $appPath"
}
if (-not (Test-Path -LiteralPath $core)) {
    Write-Warning "notehub-core.exe is missing from $appPath. Qt DLL deployment can be repaired, but NoteHub will still need the Go core alongside NoteHub.exe."
}

$previousPath = $env:PATH
try {
    $env:PATH = (Split-Path $Compiler -Parent) + ';' + (Join-Path $QtRoot 'bin') + ';' + $env:PATH
    Invoke-Native $windeployqt --release --compiler-runtime --no-translations $exe
}
finally {
    $env:PATH = $previousPath
}

$required = @(
    'Qt6Core.dll', 'Qt6Gui.dll', 'Qt6Widgets.dll', 'Qt6Concurrent.dll',
    'platforms\qwindows.dll',
    'libgcc_s_seh-1.dll', 'libstdc++-6.dll', 'libwinpthread-1.dll'
)
$missing = $required | Where-Object { -not (Test-Path -LiteralPath (Join-Path $appPath $_)) }
if ($missing) {
    throw "Deployment is still incomplete. Missing: $($missing -join ', ')"
}

Write-Host "Runtime deployment repaired: $exe" -ForegroundColor Green
