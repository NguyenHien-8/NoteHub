#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${VERSION:-0.3.0}"
DIST="$ROOT/dist/macos"
BUILD="$ROOT/build/qt-macos"
rm -rf "$DIST"
mkdir -p "$DIST"

echo "[1/4] Building Go backend"
go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o "$DIST/notehub-backend" "$ROOT/cmd/notehub-backend"

echo "[2/4] Building Qt frontend"
cmake -S "$ROOT" -B "$BUILD" -DCMAKE_BUILD_TYPE=Release ${QT_ROOT:+-DCMAKE_PREFIX_PATH="$QT_ROOT"}
cmake --build "$BUILD" --parallel
cp -R "$BUILD/frontend/NoteHub.app" "$DIST/NoteHub.app"
cp "$DIST/notehub-backend" "$DIST/NoteHub.app/Contents/MacOS/notehub-backend"
chmod +x "$DIST/NoteHub.app/Contents/MacOS/notehub-backend"

echo "[3/4] Deploying Qt frameworks"
if [[ -n "${QT_ROOT:-}" && -x "$QT_ROOT/bin/macdeployqt" ]]; then
  "$QT_ROOT/bin/macdeployqt" "$DIST/NoteHub.app"
elif command -v macdeployqt >/dev/null 2>&1; then
  macdeployqt "$DIST/NoteHub.app"
else
  echo "warning: macdeployqt not found; app bundle may require a system Qt installation" >&2
fi

echo "[4/4] Packaging"
(cd "$DIST" && ditto -c -k --sequesterRsrc --keepParent NoteHub.app "$ROOT/dist/NoteHub-macos.zip")
echo "Done: $DIST/NoteHub.app"
