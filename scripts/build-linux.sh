#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${VERSION:-0.3.0}"
DIST="$ROOT/dist/linux"
BUILD="$ROOT/build/qt-linux"
mkdir -p "$DIST"

echo "[1/3] Building Go backend"
go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o "$DIST/notehub-backend" "$ROOT/cmd/notehub-backend"

echo "[2/3] Building Qt frontend"
cmake -S "$ROOT" -B "$BUILD" -DCMAKE_BUILD_TYPE=Release ${QT_ROOT:+-DCMAKE_PREFIX_PATH="$QT_ROOT"}
cmake --build "$BUILD" --parallel
cp "$BUILD/frontend/NoteHub" "$DIST/NoteHub"
chmod +x "$DIST/NoteHub" "$DIST/notehub-backend"

echo "[3/3] Packaging"
cp "$ROOT/assets/icons/NoteHub.png" "$DIST/NoteHub.png"
tar -C "$DIST" -czf "$ROOT/dist/NoteHub-linux-x86_64.tar.gz" NoteHub notehub-backend NoteHub.png
echo "Done: $DIST/NoteHub"
