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
`Desktop.SetTypography` persists one global family + logical-pixel text size.
`internal/ui/typography` resolves only font files already installed by the host
OS, caches loaded resources, falls back to the Fyne system font when a face is
missing, and never ships third-party font binaries.

The desktop chrome is a responsive `Rails` widget. Left/right widths are saved
only after drag end; drag motion changes geometry without refreshing complete
widget subtrees. The resize hit area is transparent and sits in the gap between
rounded panels. Side panels use a soft tinted surface while the center workspace
uses a neutral surface. Automatic rail suppression on narrow windows is not
persisted as a user collapse choice. The left rail retains a compact icon menu
instead of disappearing. On Windows, `Desktop.Show` asks the OS to maximize
the live native window once; the initial fitted client size remains its restore
size. The OS handles monitor work-area and DPI boundaries.

Timeline uses 40-note keyset pages. Calendar boundaries use local calendar
arithmetic, including DST. Search has a 300 ms debounce. Image previews are
decoded off the UI thread, capped at 12 million source pixels, resized to at
most 640 × 420 while preserving aspect ratio, and retained in a bounded cache.
Gallery tiles use `canvas.ImageFillContain`. A width-dependent layout fills
each row with aspect-weighted frames, adapting the column count and bounded
height. Geometry is cached until width or attachment data changes.
`NewResponsiveVBox` propagates width before measuring child heights, so gallery
rows cannot overlap the next card during resize. A draggable tile has exactly
one Fyne renderer identity, including when wrapped with drag behavior.
An in-gallery floating preview follows pointer motion without rebuilding the
gallery or changing attachment order. Ordering is committed only on drag release
through the attachment service; failed writes leave the saved order intact.

The note editor stores Markdown directly in `Memo.Content`, retaining revision
checks and the existing draft lifecycle. Toolbar edits produce Markdown or
allowlisted inline HTML style spans. The local renderer supports those styles
without fetching URLs or executing HTML. Markdown import/export operates on
the draft; only Save updates the memo. No database migration is required.

Migration v2 adds favorites without modifying v1. Counts use distinct active
shared memos. Backup records carry an optional favorite flag compatible with
existing version 1 archives; bearer tokens are never exported. Migration checks
execute inside IMMEDIATE writer transactions to serialize concurrent starts.

The optional HTTP server binds to loopback only and starts solely from an
explicit Settings action. No saved preference automatically opens a port.

See `BACKEND_FROM_MEMOS.md` for the Memos algorithm selection rationale.
