#!/usr/bin/env bash
set -euo pipefail
root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
version="${1:-0.3.0}"
output="${2:-$root/dist/linux}"
cmake -S "$root" -B "$root/build/qt-linux" -G Ninja \
  -DCMAKE_BUILD_TYPE=Release -DBUILD_TESTING=OFF -DNOTEHUB_VERSION="$version"
cmake --build "$root/build/qt-linux" --target NoteHub
cmake --install "$root/build/qt-linux" --prefix "$output"
cp "$root/LICENSE" "$output/LICENSE"
printf 'Ready: %s/NoteHub (keep notehub-core and Qt runtime files together)\n' "$output"
