# NoteHub Architecture

```text
Desktop UI (Fyne v2)
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
`internal/app.OpenBackend` is the composition root used by the GUI.

`internal/ui` owns the window, responsive rails and application theme.
Presentation widgets live in `components`; `screens` and `dialogs` call
backend services. The `work.Runner` owns cancellable worker lifetimes and
posts results through `fyne.Do`. Query generations reject outdated results.
The UI waits for workers before closing sharing and SQLite.

`Desktop.SetAppearance` applies the selected theme and refreshes the content
tree so scoped button themes and native text/icon caches repaint together.

Timeline uses 40-note keyset pages. Calendar boundaries use local calendar
arithmetic, including DST. Search has a 300 ms debounce. Image previews are
decoded off the UI thread, capped at 12 million source pixels and retained in
a bounded thumbnail cache.

Migration v2 adds favorites without modifying v1. Counts use distinct active
shared memos. Backup records carry an optional favorite flag compatible with
existing version 1 archives; bearer tokens are never exported. Migration checks
execute inside IMMEDIATE writer transactions to serialize concurrent starts.

The optional HTTP server binds to loopback only and starts solely from an
explicit Settings action. No saved preference automatically opens a port.

See `BACKEND_FROM_MEMOS.md` for the Memos algorithm selection rationale.
