# NoteHub

NoteHub is being developed as a cross-platform desktop timeline note application
for Windows, macOS and Linux.

## Current status

This package contains the implemented **Go backend**. The Fyne GUI folders are
still structural placeholders for the next phase.

Implemented backend features:

- Create / read / edit / delete memo
- Optimistic edit revision
- Local file attachments + SHA-256
- Timeline with keyset pagination
- Calendar/date query
- Hierarchical tags
- SQLite FTS5 Unicode search
- Read-only share tokens + optional HTTP handler/server
- Export / Import ZIP with Skip / Replace / Duplicate conflict policies
- SQLite migrations
- Windows/macOS/Linux application-data paths

## Backend startup

```go
backend, err := app.OpenBackend(ctx, app.Config{
    AppName: "NoteHub",
    Version: "0.1.0",
})
if err != nil {
    // handle error
}
defer backend.Close()
```

The future GUI should call `backend.Memos`, `backend.Timeline`,
`backend.Calendar`, `backend.Search`, etc.; it should not open SQLite directly.

## Development

```bash
go mod tidy
go test ./...
```

For offline type-checking only:

```bash
go test -tags teststub ./...
```

Do not use `teststub` for a real build.

See:

- `docs/BACKEND_FROM_MEMOS.md`
- `docs/BACKEND_TEST_REPORT.md`
- `PROJECT_STRUCTURE_NOTEHUB.md`
