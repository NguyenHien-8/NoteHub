[CmdletBinding()]
param(
    [ValidatePattern('^\d+\.\d+\.\d+$')]
    [string]$Version = '0.2.0',
    [string]$OutputDirectory = '',
    [string]$Compiler = ''
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$repoRoot = Split-Path -Parent $PSScriptRoot
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw 'Go is required. Install Go and put go.exe on PATH.'
}
if ([Environment]::OSVersion.Platform -ne [PlatformID]::Win32NT) {
    throw 'Run this script on Windows with a native MinGW-w64 GCC toolchain.'
}
if ($env:GOOS -and $env:GOOS -ne 'windows') {
    throw "GOOS=$env:GOOS conflicts with a native Windows build. Clear GOOS first."
}

# Resolve one compiler executable, including paths containing spaces. Put its
# directory on PATH so cgo does not have to parse an unquoted full CC path.
if (-not $Compiler -and $env:CC) { $Compiler = $env:CC }
if (-not $Compiler) {
    $gccCommand = Get-Command gcc -ErrorAction SilentlyContinue
    if ($gccCommand) { $Compiler = $gccCommand.Source }
}
if (-not $Compiler) {
    foreach ($candidate in @(
        'C:\msys64\ucrt64\bin\gcc.exe',
        'C:\msys64\mingw64\bin\gcc.exe',
        'C:\mingw64\bin\gcc.exe',
        'C:\Qt\Tools\mingw1310_64\bin\gcc.exe'
    )) {
        if (Test-Path -LiteralPath $candidate -PathType Leaf) {
            $Compiler = $candidate
            break
        }
    }
}
if (-not $Compiler) {
    throw 'MinGW-w64 GCC is required. Add gcc.exe to PATH or pass -Compiler C:\path\to\gcc.exe.'
}
$compilerCommand = Get-Command $Compiler -ErrorAction SilentlyContinue
if (-not $compilerCommand) { throw "Compiler executable not found: $Compiler" }
$compilerPath = $compilerCommand.Source
$env:PATH = "$(Split-Path -Parent $compilerPath);$env:PATH"
$env:CC = Split-Path -Leaf $compilerPath
$env:CGO_ENABLED = '1'
& $compilerPath --version
if ($LASTEXITCODE -ne 0) { throw 'The C compiler could not run.' }

if (-not $OutputDirectory) { $OutputDirectory = Join-Path $repoRoot 'dist/windows' }
if (-not [IO.Path]::IsPathRooted($OutputDirectory)) {
    $OutputDirectory = Join-Path $repoRoot $OutputDirectory
}
$null = New-Item -ItemType Directory -Force -Path $OutputDirectory
$outputPath = Join-Path $OutputDirectory 'NoteHub.exe'
Push-Location -LiteralPath $repoRoot
try {
    & go build -trimpath -ldflags "-s -w -H windowsgui -X main.version=$Version" -o $outputPath ./cmd/notehub
    if ($LASTEXITCODE -ne 0) { throw "NoteHub Windows build failed (exit $LASTEXITCODE)." }
    if (-not (Test-Path -LiteralPath $outputPath -PathType Leaf)) { throw 'Build did not produce NoteHub.exe.' }
    Write-Host "Built $outputPath (version $Version)"
} finally {
    Pop-Location
}
