# NoteHub Architecture

## Runtime architecture

```text
┌──────────────────────────────────────┐
│ NoteHub.exe                          │
│ C++17 + Qt 6 Widgets                │
│                                      │
│ MainWindow / NoteEditor / Gallery   │
└──────────────────┬───────────────────┘
                   │ QProcess
                   │ JSON lines over stdin/stdout
                   ▼
┌──────────────────────────────────────┐
│ notehub-core.exe                     │
│ Go                                   │
│                                      │
│ IPC → Service → Repository/Storage  │
└──────────────┬───────────────┬───────┘
               │               │
               ▼               ▼
           SQLite + FTS5   Filesystem
```

The Qt process never opens SQLite and never manipulates the attachment store directly. `notehub-core` owns the data directory, migrations, repositories, services, attachment storage, backup and sharing. The desktop process talks to it only through the versioned IPC protocol in `internal/ipc`.

`BackendClient` starts the Go process with `QProcess`, keeps stdout reserved for newline-delimited JSON replies, keeps stderr as diagnostics, rejects malformed/unknown replies and applies request timeouts. On Windows the backend is started with `CREATE_NO_WINDOW` so a console window is not flashed.

## Desktop composition

The current Qt desktop is implemented in `desktop/src`:

- `MainWindow`: application shell, navigation, composer, page routing, filters, settings and note cards.
- `NoteEditor`: rich-text/Markdown editing while preserving untouched Markdown exactly.
- `ImageGallery`: asynchronous image preview loading plus drag-to-reorder.
- `BackendClient`: process lifecycle and IPC transport.

The application uses Qt's `Fusion` style as a deterministic cross-platform base. A single palette + stylesheet is then applied for Light/Dark/System appearance. This prevents Windows native widgets, MinGW widgets and Linux widgets from producing visibly different spacing/borders.

## UX shell algorithm

The main window uses a responsive three-column shell:

```text
header:  sidebar toggle | logo + NoteHub | global search

left rail          center workspace                    right context
──────────         ───────────────────────────         ─────────────
Home               composer                            calendar
Calendar           page heading / clear filters        quick filters
Search             scrollable page content             live counts
Attachments
Tags
Settings
My Tags
```

The left rail has two stable modes instead of continuously changing widths:

- compact: 68 px, icon-only with tooltips;
- expanded: 212 px, icon + text.

The user's compact preference is persisted. Windows narrower than 980 px temporarily use compact mode without overwriting that preference. The right context column is shown at widths of 1180 px and above and hidden below that threshold, so the center workspace remains usable instead of being squeezed.

Navigation icons are drawn with `QPainter` into high-DPI pixmaps at runtime. Each icon has neutral and checked-state pixmaps, avoiding platform-specific legacy `QStyle::StandardPixmap` icons and keeping the visual language consistent on Windows/macOS/Linux.

## Visual hierarchy

The GUI uses four surface levels:

1. application background;
2. tinted navigation rail;
3. neutral workspace / context cards;
4. interactive note/settings/attachment cards.

Borders are deliberately low-contrast and the accent color is reserved for selection, focus, primary actions and calendar activity. Status counts are moved from a permanent status bar into the right-side Quick Filters card, reducing visual noise while keeping All Notes/Favorites/Shared counts visible.

The header uses the embedded `NoteHub.png` resource instead of a platform file icon. The global search field includes a search glyph and keeps `Ctrl+K` as the keyboard shortcut.

## Composer algorithm

The Home composer separates the draft body from secondary actions:

- `Attach files` stages paths only; no file is copied until the memo exists.
- `Add tag` inserts a normalized `#tag` token directly into the draft.
- staged attachments are summarized as a compact count; full filenames remain available as a tooltip.
- `Clear files` appears only when staging is non-empty.
- Save first creates the memo and then imports attachments. A memo therefore stays saved even when one attachment import fails, matching the backend's existing partial-failure semantics.

## Timeline and note cards

Timeline requests remain paged by the backend. The Qt shell keeps a generation number; callbacks from older navigation/search generations are ignored so a slow reply cannot overwrite a newer page.

Each note card has:

- muted timestamp metadata;
- favorite toggle;
- edit action;
- overflow menu for share/delete;
- rendered local note preview;
- image gallery;
- file chips;
- clickable tag chips.

Long text previews are height-bounded to 220 px and do not introduce a nested scrollbar. This keeps the timeline scannable; Edit remains the full-document surface.

Favorite changes are optimistic in the button state but are reverted if the backend returns an error. The button is disabled while the request is in flight, preventing duplicate toggles.

## Image gallery layout algorithm

Image decoding still runs outside the UI thread through `QtConcurrent`. Source images larger than 12 million pixels are rejected for preview decoding, and previews are downscaled before entering the bounded cache.

The previous justified-row algorithm could stretch an incomplete last row to the full note width. Two images could therefore become very large even when earlier rows used four columns. The new algorithm uses a stable responsive grid:

```text
columns = clamp((availableWidth + gap) / 220, 1, 4)
tileWidth = (availableWidth - gap * (columns - 1)) / columns
imageHeight = clamp(tileWidth * 0.68, 130, 205)
```

Every row, including the final incomplete row, uses the same tile width. Images remain aspect-preserving (`KeepAspectRatio`) inside a bounded preview area. The result is more predictable scrolling and avoids giant attachment previews while resizing.

Drag reorder continues to emit the complete attachment ID order. The backend remains authoritative; after a successful reorder the timeline is refreshed from persisted state.

## Attachments page

The attachment manager now uses information rows instead of full-width centered filenames. Each row shows an attachment icon, filename, local path, Open note and Delete actions. Long paths are shown as selectable metadata and retain the complete path as a tooltip.

## Tags page

Tags are rendered as responsive tag cards in a grid instead of full-width buttons. The count stays attached to the tag, and clicking the card returns to Home with that tag filter applied. Empty state messaging explains how tags are created.

## Settings page

Settings are grouped into three cards:

- Appearance: theme, font and text size;
- Data & backup: data folder, ZIP export/import;
- Local sharing: enable toggle and port.

This keeps unrelated controls from appearing as one long form and makes destructive/operational actions easier to distinguish from appearance preferences.

## Theme and typography

`applyAppearance()` owns the application palette and stylesheet. It defines semantic colors for background, surface, border, text, muted text, hover, accent and accent-soft states. System mode follows `QStyleHints::colorScheme()`.

The startup font prefers `Segoe UI Variable Text` on supported Windows systems, falls back to `Segoe UI`, then falls back to Qt's platform font. User-selected font and size settings override the startup choice. No font files are bundled.

## Data/search/backend algorithms kept unchanged

The GUI redesign does not change storage or migrations. Existing algorithms remain backend-owned:

- SQLite migrations and repository interfaces;
- FTS5 search;
- memo revision conflict protection;
- timeline paging;
- calendar aggregation;
- tag parsing;
- SHA-256 attachment handling;
- safe ZIP backup/import;
- hashed share tokens;
- loopback-only optional sharing server.

Therefore existing `notehub.db` and `attachments/` data remain compatible with the redesigned Qt desktop.
