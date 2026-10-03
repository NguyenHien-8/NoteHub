# Archived Fyne source

This is the source snapshot preserved when the desktop moved to C++ / Qt.
It includes unfinished changes from the earlier UI request. It is reference
material, not an active or validated application target. Its original imports
have intentionally been kept for comparison with Git history.

Go ignores directories beginning with `_` during `go test ./...` and module
dependency discovery. Build the maintained Qt GUI in `desktop/` and the Go
backend in `cmd/notehub-core/`; do not build this archive.
