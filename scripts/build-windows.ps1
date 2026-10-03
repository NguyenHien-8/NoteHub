param(
    [string]$Version = '0.3.0',
    [string]$QtRoot = $env:QT_ROOT_DIR,
    [string]$Compiler = 'C:\Qt\Tools\mingw1310_64\bin\g++.exe',
    [string]$OutputDirectory = 'dist\windows',
    [switch]$Clean,
    [switch]$ForceCloseRunning
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

function Stop-NoteHubProcesses {
    $names = @('NoteHub', 'notehub-core')
    $running = Get-Process -Name $names -ErrorAction SilentlyContinue
    if (-not $running) { return }

    if (-not $ForceCloseRunning) {
        $details = ($running | ForEach-Object { "$($_.ProcessName) (PID $($_.Id))" }) -join ', '
        throw "NoteHub is still running: $details. Close it first, or rebuild with -ForceCloseRunning."
    }

    $running | Stop-Process -Force
    Start-Sleep -Milliseconds 400
}

if ($Version -notmatch '^\d+\.\d+\.\d+$') {
    throw 'Version must be X.Y.Z'
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

if (-not $QtRoot -or -not (Test-Path -LiteralPath (Join-Path $QtRoot 'lib\cmake\Qt6\Qt6Config.cmake'))) {
    throw 'Provide -QtRoot pointing to a Qt 6.5+ MinGW kit, for example C:\Qt\6.11.2\mingw_64.'
}
if (-not (Test-Path -LiteralPath $Compiler)) {
    throw 'Provide -Compiler pointing to the g++.exe that matches the Qt MinGW kit.'
}

$cmake = (Get-Command cmake -ErrorAction Stop).Source
$go = (Get-Command go -ErrorAction Stop).Source
$ninja = 'C:\Qt\Tools\Ninja\ninja.exe'
if (-not (Test-Path -LiteralPath $ninja)) {
    $ninja = (Get-Command ninja -ErrorAction Stop).Source
}
$windeployqt = Join-Path $QtRoot 'bin\windeployqt.exe'
if (-not (Test-Path -LiteralPath $windeployqt)) {
    throw "windeployqt.exe was not found under $QtRoot\bin"
}

$outputPath = [System.IO.Path]::GetFullPath((Join-Path $repoRoot $OutputDirectory))
$buildPath = Join-Path $repoRoot 'build\qt-release-mingw'
$stagePath = Join-Path $repoRoot 'build\package-windows-mingw'

Stop-NoteHubProcesses

if ($Clean) {
    Remove-Item -LiteralPath $buildPath -Recurse -Force -ErrorAction SilentlyContinue
}
Remove-Item -LiteralPath $stagePath -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Path $stagePath -Force | Out-Null

$previousPath = $env:PATH
try {
    # The MinGW runtime directory must be visible to CMake, the linker and windeployqt.
    $env:PATH = (Split-Path $Compiler -Parent) + ';' + (Join-Path $QtRoot 'bin') + ';' + $env:PATH

    Write-Host '[1/5] Checking toolchain...'
    Invoke-Native $go version
    Invoke-Native $cmake --version
    Invoke-Native $ninja --version
    Invoke-Native $Compiler --version

    Write-Host '[2/5] Configuring CMake...'
    Invoke-Native $cmake `
        -S $repoRoot `
        -B $buildPath `
        -G Ninja `
        "-DCMAKE_MAKE_PROGRAM=$ninja" `
        "-DCMAKE_CXX_COMPILER=$Compiler" `
        "-DCMAKE_PREFIX_PATH=$QtRoot" `
        "-DNOTEHUB_VERSION=$Version" `
        -DCMAKE_BUILD_TYPE=Release `
        -DBUILD_TESTING=OFF

    Write-Host '[3/5] Building Qt GUI and Go core...'
    Invoke-Native $cmake --build $buildPath --target NoteHub

    Write-Host '[4/5] Installing and deploying runtime...'
    Invoke-Native $cmake --install $buildPath --prefix $stagePath

    $stageExe = Join-Path $stagePath 'NoteHub.exe'
    if (-not (Test-Path -LiteralPath $stageExe)) {
        throw "CMake install did not produce $stageExe"
    }

    # Run windeployqt explicitly as a second safety net. This is what copies
    # the Qt/runtime DLLs actually required by NoteHub, qwindows.dll and MinGW runtime DLLs.
    Invoke-Native $windeployqt --release --compiler-runtime --no-translations $stageExe

    Copy-Item -LiteralPath (Join-Path $repoRoot 'LICENSE') -Destination $stagePath -Force

    Write-Host '[5/5] Verifying deployable folder...'

    # Verify only files that are unconditionally required by the packaged app.
    # Optional Qt modules must not be hard-coded: windeployqt copies the DLLs
    # actually required by the final executable/runtime dependency graph.
    $required = @(
        'NoteHub.exe',
        'notehub-core.exe',
        'Qt6Core.dll',
        'Qt6Gui.dll',
        'Qt6Widgets.dll',
        'platforms\qwindows.dll',
        'libgcc_s_seh-1.dll',
        'libstdc++-6.dll',
        'libwinpthread-1.dll'
    )

    $missing = New-Object System.Collections.Generic.List[string]
    foreach ($relative in $required) {
        if (-not (Test-Path -LiteralPath (Join-Path $stagePath $relative))) {
            $missing.Add($relative)
        }
    }

    # Inspect the actual PE imports when MinGW objdump is available. This makes
    # validation follow NoteHub.exe instead of guessing that a DLL is required.
    $objdump = Join-Path (Split-Path $Compiler -Parent) 'objdump.exe'
    if (Test-Path -LiteralPath $objdump) {
        $dumpOutput = & $objdump -p $stageExe 2>&1
        if ($LASTEXITCODE -eq 0) {
            $directDependencies = @(
                $dumpOutput |
                    ForEach-Object {
                        if ($_ -match 'DLL Name:\s*(.+?\.dll)\s*$') {
                            $Matches[1].Trim()
                        }
                    } |
                    Where-Object {
                        $_ -match '^Qt6.*\.dll$' -or
                        $_ -match '^lib(gcc|stdc\+\+|winpthread).*\.dll$'
                    } |
                    Sort-Object -Unique
            )

            foreach ($dll in $directDependencies) {
                if (-not (Test-Path -LiteralPath (Join-Path $stagePath $dll))) {
                    $missing.Add($dll)
                }
            }

            if ($directDependencies.Count -gt 0) {
                Write-Host ('Direct deploy dependencies: ' + ($directDependencies -join ', '))
            }
        }
        else {
            Write-Warning 'objdump dependency inspection failed; continuing with baseline package checks.'
        }
    }
    else {
        Write-Warning 'objdump.exe was not found; continuing with baseline package checks.'
    }

    $missing = @($missing | Sort-Object -Unique)
    if ($missing.Count -gt 0) {
        throw "Deployment is incomplete. Missing: $($missing -join ', ')"
    }

    # Publish only after the staged package has passed validation.
    Remove-Item -LiteralPath $outputPath -Recurse -Force -ErrorAction SilentlyContinue
    $outputParent = Split-Path $outputPath -Parent
    New-Item -ItemType Directory -Path $outputParent -Force | Out-Null
    Copy-Item -LiteralPath $stagePath -Destination $outputPath -Recurse -Force

    Write-Host ''
    Write-Host 'Build completed successfully.' -ForegroundColor Green
    Write-Host "Run ONLY this packaged executable:"
    Write-Host "  $outputPath\NoteHub.exe" -ForegroundColor Cyan
    Write-Host ''
    Write-Host 'Do not distribute build\qt-release-mingw\bin\NoteHub.exe by itself.' -ForegroundColor Yellow
}
finally {
    $env:PATH = $previousPath
}
