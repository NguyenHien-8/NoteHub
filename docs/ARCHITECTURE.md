# NoteHub Architecture

```text
Desktop UI (future Fyne)
        │
        ▼
internal/service
        │
        ├───────────────┬────────────────┐
        ▼               ▼                ▼
repository interfaces   storage          backup/share
        │               │
        ▼               ▼
SQLite + FTS5       local attachments
```

The UI must not issue SQL or manipulate attachment paths directly.
`internal/app.OpenBackend` is the composition root used by the future GUI.

See `BACKEND_FROM_MEMOS.md` for the Memos algorithm selection rationale.
