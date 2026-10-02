# NoteHub desktop usability improvements

**Goal:** Fix the reported desktop overflow, unnecessary search scrolling and
refresh flicker; provide compact configurable typography, resizable sidebars
and persistent multi-image galleries with deletion from note dialogs.

**User decisions:** Edit this project directly, no ZIP. Keep the Go/Fyne and
SQLite architecture. Images reorder within a responsive grid, as confirmed
in the conversation. Preserve the logo and brand size.

**Design:** Fit startup bounds to the monitor's working area with DPI/native
chrome accounted for. Use two draggable rails with collapse buttons and saved
widths. Default text becomes 14 logical pixels with compact controls; Settings
supports built-in fonts, a validated custom font and font size. Search remains
single-line with a responsive minimum-height layout and no Ctrl+K label.
Refresh keeps existing content until results arrive and reuses unchanged
cards. All image attachments receive thumbnails; dragging changes order on
drop and saves through a backend service. Note/editor attachment removal asks
for confirmation and immediately updates the note without discarding text.

## Tasks and ownership

- [ ] Root: regression tests and fixes for search sizing, refresh retention,
  thumbnail coverage/cache and smaller typography/font preferences.
- [ ] Viewport worker: DPI-aware startup fitting in platform-specific files;
  resizable/collapsible rails, persistence and boundary tests.
- [ ] Backend worker: additive migration v3; atomic full-set attachment reorder,
  validation, append order and compatible backup round trips.
- [ ] Gallery worker: responsive image tiles, drag/drop feedback, tap behavior,
  and timeline card reuse. Thumbnail maps use attachment IDs everywhere.
- [ ] Root: integrate window/header/settings, gallery/reorder callbacks and
  image/file deletion in both note viewer and editor; preserve unsaved text.
- [ ] Root: run complete tests, vet, native GUI smoke at actual startup sizing,
  inspect screenshots with multiple images, build refreshed Windows executable,
  document results and remaining platform limits.

## Review focus

- High-DPI, small monitors and long search text must not enlarge the window.
- Sidebar drag bounds must leave a usable center and remember collapsed state.
- Stale asynchronous results must not replace a newer query or closed dialog.
- Stale/invalid reorder requests must not lose attachments or change other notes.
- Failed deletion/reorder must retain editor text and visibly report failure.
- Refresh and font/theme changes must retain content and scroll position.

**Validation:** Real SQLite integration tests, Fyne interaction tests, Windows
native smoke, gofmt, `go test -count=1 ./...`, `go vet ./...`, native release build.
macOS/Linux remain native-CI checks; no claim based on Windows alone.
