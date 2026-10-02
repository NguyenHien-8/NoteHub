#!/usr/bin/env bash
set -euo pipefail

version="${1:-0.2.0}"
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo 'Version must be x.y.z' >&2; exit 1; }
[[ "$(uname -s)" == Darwin ]] || { echo 'Run this script on macOS.' >&2; exit 1; }
[[ -z "${GOOS:-}" || "$GOOS" == darwin ]] || { echo 'Clear conflicting GOOS before building.' >&2; exit 1; }
repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
output_dir="${2:-$repo_root/dist/macos}"
mkdir -p -- "$output_dir"
output_dir="$(cd -- "$output_dir" && pwd)"
for tool in go clang xcrun hdiutil ditto; do
    command -v "$tool" >/dev/null || { echo "Missing build tool: $tool (install Xcode Command Line Tools)" >&2; exit 1; }
done
xcrun --find clang >/dev/null
export CGO_ENABLED=1
compiler="$(command -v "${CC:-clang}")" || { echo 'Configured C compiler was not found.' >&2; exit 1; }
export PATH="$(dirname -- "$compiler"):$PATH"
export CC="$(basename -- "$compiler")"
cd -- "$repo_root"
go build -trimpath -ldflags "-s -w -X main.version=$version" -o "$output_dir/NoteHub" ./cmd/notehub
arch="$(go env GOARCH)"

# Keep the packaging CLI outside the application's go.mod, pinned rather than
# @latest. It supports the Fyne v2 metadata and prebuilt executable interface.
mkdir -p -- "$repo_root/build/tools"
GOBIN="$repo_root/build/tools" go install fyne.io/tools/cmd/fyne@v1.7.2
cd -- "$output_dir"
"$repo_root/build/tools/fyne" package --os darwin --src "$repo_root/cmd/notehub" \
    --executable "$output_dir/NoteHub" --name NoteHub \
    --icon "$repo_root/assets/icons/NoteHub.png" \
    --app-id io.github.nguyenhien8.notehub --app-version "$version"
test -x "$output_dir/NoteHub.app/Contents/MacOS/NoteHub"
cp -- "$repo_root/LICENSE" "$output_dir/NoteHub.app/Contents/Resources/LICENSE"
ditto -c -k --sequesterRsrc --keepParent "$output_dir/NoteHub.app" "$output_dir/NoteHub-$version-macos-$arch.app.zip"
stage_dir="$(mktemp -d "$output_dir/dmg-stage.XXXXXX")"
trap 'rm -rf -- "$stage_dir"' EXIT
ditto "$output_dir/NoteHub.app" "$stage_dir/NoteHub.app"
ln -s /Applications "$stage_dir/Applications"
hdiutil create -volname NoteHub -srcfolder "$stage_dir" -ov -format UDZO \
    "$output_dir/NoteHub-$version-macos-$arch.dmg"
echo "Built NoteHub.app, app ZIP, and DMG in $output_dir"
