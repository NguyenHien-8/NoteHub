#!/usr/bin/env bash
set -euo pipefail

version="${1:-0.2.0}"
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo 'Version must be x.y.z' >&2; exit 1; }
[[ "$(uname -s)" == Linux ]] || { echo 'Run this script on Linux.' >&2; exit 1; }
[[ -z "${GOOS:-}" || "$GOOS" == linux ]] || { echo 'Clear conflicting GOOS before building.' >&2; exit 1; }
repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
output_dir="${2:-$repo_root/dist/linux}"
mkdir -p -- "$output_dir"
output_dir="$(cd -- "$output_dir" && pwd)"
for tool in go "${CC:-gcc}" pkg-config tar; do
    command -v "$tool" >/dev/null || { echo "Missing build tool: $tool (see docs/BUILD.md)" >&2; exit 1; }
done
pkg-config --exists gl x11 xcursor xrandr xinerama xi xxf86vm || {
    echo 'Missing OpenGL/X11 development libraries. See docs/BUILD.md.' >&2; exit 1;
}
export CGO_ENABLED=1
compiler="$(command -v "${CC:-gcc}")"
export PATH="$(dirname -- "$compiler"):$PATH"
export CC="$(basename -- "$compiler")"
cd -- "$repo_root"
go build -trimpath -ldflags "-s -w -X main.version=$version" -o "$output_dir/NoteHub" ./cmd/notehub
arch="$(go env GOARCH)"
cp -- README.md "$output_dir/README.md"
cp -- LICENSE "$output_dir/LICENSE"
tar -czf "$output_dir/NoteHub-$version-linux-$arch.tar.gz" -C "$output_dir" NoteHub README.md LICENSE
echo "Built $output_dir/NoteHub and NoteHub-$version-linux-$arch.tar.gz"
