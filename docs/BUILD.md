# Build

Build NoteHub on the operating system you target. Fyne v2.7.4 uses a native
OpenGL/window driver and requires **CGO_ENABLED=1**, a C compiler, and a
graphical desktop for interactive use. SQLite remains `modernc.org/sqlite`.
Go 1.26.x is used by CI; the application module declares its minimum Go version
in `go.mod`.

## Windows

Install Go and a 64-bit MinGW-w64 GCC compiler. With MSYS2 UCRT64:

```bash
pacman -S --needed mingw-w64-ucrt-x86_64-gcc
```

From PowerShell in the repository:

```powershell
./scripts/build-windows.ps1
# Explicit compiler path and release version are optional:
./scripts/build-windows.ps1 -Version 0.2.0 -Compiler 'C:\msys64\ucrt64\bin\gcc.exe'
./dist/windows/NoteHub.exe
# Optional isolated profile, including paths with spaces:
./dist/windows/NoteHub.exe --data-dir 'D:\My Notes\NoteHub'
```

The script enables CGO, checks the compiler, and handles paths containing spaces.
It discovers GCC from `CC`, `PATH`, standard MSYS2/MinGW locations, or
`C:\Qt\Tools\mingw1310_64\bin\gcc.exe`. `CC` or `-Compiler` must name a compiler
executable, without appended arguments. `-OutputDirectory` changes the output
directory (relative paths are resolved from the repository).

For direct development commands, make GCC available in the current shell:

```powershell
$env:PATH = 'C:\msys64\ucrt64\bin;' + $env:PATH
$env:CC = 'gcc'
$env:CGO_ENABLED = '1'
go run ./cmd/notehub
go build -o dist/NoteHub.exe ./cmd/notehub
```

The release script uses `-H windowsgui` so a launched app does not open a console
window. A direct development build retains the console for diagnostics.

## macOS

Install Go and Xcode Command Line Tools:

```bash
xcode-select --install
bash scripts/build-macos.sh 0.2.0
open dist/macos/NoteHub.app
```

The first argument is a version in `x.y.z` format; the optional second argument
is the output directory. The script compiles a native binary, installs the
pinned `fyne.io/tools/cmd/fyne@v1.7.2` CLI into `build/tools/`, packages that
prebuilt binary with the embedded app icon metadata, and creates:

- `dist/macos/NoteHub.app`
- `dist/macos/NoteHub-0.2.0-macos-<arch>.app.zip`
- `dist/macos/NoteHub-0.2.0-macos-<arch>.dmg`

Fyne CLI is installed separately from the app module, so it does not add tool
dependencies to `go.mod`. The DMG includes an Applications shortcut. These
packages do not include Developer ID signing or notarization; those require
the maintainer's Apple credentials and a separate signing process. Build on
each required Mac architecture; the script creates a single native architecture,
not a universal binary.

See the official [Fyne packaging documentation](https://docs.fyne.io/started/packaging/)
and [pinned CLI package implementation](https://github.com/fyne-io/tools/blob/v1.7.2/cmd/fyne/internal/commands/package.go).

## Linux

On Debian/Ubuntu, install development and desktop integration dependencies:

```bash
sudo apt-get update
sudo apt-get install -y gcc pkg-config libgl1-mesa-dev libx11-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libxxf86vm-dev zenity xdg-utils
bash scripts/build-linux.sh 0.2.0
./dist/linux/NoteHub
```

The first argument is an optional version; the second is an optional output
directory. The script checks the OpenGL/X11 development packages through
`pkg-config`, enables CGO, and creates the binary plus
`NoteHub-0.2.0-linux-<arch>.tar.gz`, containing the binary, README, and license.
Extract the archive and run `./NoteHub` from a graphical desktop session.

The binary dynamically links to system graphics libraries. On a runtime-only
Debian/Ubuntu installation, install `libgl1 libx11-6 libxcursor1 libxrandr2
libxinerama1 libxi6 libxxf86vm1 zenity xdg-utils`. Other distributions need the
equivalent packages. Native file pickers use `zenity`; system opening/revealing
uses `xdg-open`. Build on the oldest Linux distribution you intend to support
to avoid requiring a newer libc than that machine provides.

## Verification

Set CGO and configure the compiler as above, then run:

```bash
go mod tidy
go test ./...
go vet ./...
go build ./cmd/notehub
```

Run `gofmt -w` on changed Go files before committing. UI tests use
`fyne.io/fyne/v2/test` without opening desktop windows; native compilation still
requires graphics headers and CGO. To provide a virtual display on Linux:

```bash
sudo apt-get install -y xvfb
xvfb-run -a go test ./...
```

Do not build the desktop app with `-tags teststub` or disable CGO. Clear an
inherited `GOOS` that conflicts with the host platform. Use a native toolchain
matching `GOARCH`; these scripts do not configure cross compilation.

## CI and tagged artifacts

`.github/workflows/test.yml` uses native Windows, macOS, and Ubuntu runners.
Each job checks Go formatting, runs `go mod tidy` and fails on `go.mod`/`go.sum`
changes, runs `go test ./...` and `go vet ./...`, and compiles
`go build ./cmd/notehub`. Linux installs graphics dependencies, Zenity, and Xvfb;
Windows installs GCC through MSYS2. It runs on pushes, pull requests, and manual
dispatches.

`.github/workflows/release.yml` repeats those checks on a pushed `vX.Y.Z` tag,
then calls the native scripts and uploads Actions artifacts:

| Platform | Artifacts |
| --- | --- |
| Windows | `NoteHub.exe` and a versioned ZIP with README and license |
| macOS | `.app.zip` preserving the app bundle, plus `.dmg` |
| Linux | versioned binary `.tar.gz` with README and license |

Artifact filenames include the actual architecture reported by Go. The workflow
uploads build artifacts for 30 days; it does not publish a GitHub Release or
sign installers. Download them from that workflow run's **Artifacts** section.
The scripts inject the requested version using `-X main.version=<version>`;
the source default is 0.2.0. macOS bundle metadata uses the same version.
`--version` prints the embedded version and exits. `--data-dir PATH` selects an
alternative database/attachment root; otherwise the platform defaults described
in the README apply. For packaged macOS apps, pass arguments with
`open dist/macos/NoteHub.app --args --data-dir "$HOME/NoteHub test data"`.

Remote platform validation is confirmed only by successful Actions runs.
Defining these workflows or compiling locally on Windows does not verify a
macOS/Linux release. Windows setup installers, AppImage, and DEB packaging are
future distribution options.
