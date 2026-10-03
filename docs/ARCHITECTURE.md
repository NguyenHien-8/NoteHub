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

The left rail is backed by a horizontal `QSplitter` rather than a fixed-width layout. It can be dragged from 64–280 px. Below 136 px it automatically presents icon-only navigation; above that threshold it presents icon + text. The splitter handle is only 7 px wide, stays transparent at rest and uses the theme accent on hover/drag, so the rail frame sits close to the workspace without losing a clear resize affordance.

The current rail width and last expanded width are persisted. The header sidebar button toggles between 68 px and the last expanded width, while manual dragging remains authoritative. The right context column is shown at widths of 1180 px and above and hidden below that threshold, so the center workspace remains usable instead of being squeezed.

Navigation/action icons are vector-like `QPainter` glyphs backed by a palette-aware `QIconEngine`. They are not frozen to Light-theme RGB values: each repaint resolves neutral, hover, disabled and checked colors from the current Qt palette. A theme switch clears the pixmap cache and immediately repaints the existing buttons, so sidebar, note actions, menus and search icons stay legible without recreating the window. Home, Attachments and Settings use explicit house, paperclip and gear geometry instead of ambiguous platform icons.

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

Long text previews are height-bounded to 220 px with a hidden internal scrollbar. `ContainedTextBrowser` always owns/accepts wheel events while the pointer is inside the preview: it scrolls its own document, and when it reaches either boundary the event is not propagated to the outer timeline. A `WheelGuardFrame` catches ignored wheel events from the rest of the same memo card, preventing accidental movement to adjacent memo cards while the pointer is still inside that card. Edit remains the full-document surface.

Favorite changes are optimistic in the button state but are reverted if the backend returns an error. The button is disabled while the request is in flight, preventing duplicate toggles.


### Timeline viewport anchoring

Editing an older memo used to call `refresh()`, rebuild every note card and leave the timeline scroll bar at the beginning. The Qt shell now gives each note card its `memoUid` and accepts an optional refresh anchor.

Before an edit/attachment/reorder refresh:

1. record the edited memo's Y position relative to the scroll viewport;
2. record the current scrollbar value as a fallback;
3. asynchronously reload the page using the existing generation guard;
4. find the rebuilt card with the same `memoUid`;
5. restore the scrollbar so that card returns to the same viewport Y coordinate.

This is more stable than restoring only a raw pixel offset because the edited card or cards above it may change height. New-note creation and explicit navigation still use the normal top-of-view behavior.

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

- Appearance: theme, searchable installed-font picker and text size;
- Data & backup: data folder, ZIP export/import;
- Local sharing: enable toggle and port.

Appearance fields use `QFormLayout::FieldsStayAtSizeHint` plus bounded control widths, so `Light`, a font family and `11 pt` no longer stretch across the full center workspace. Theme/font combo boxes use custom paint-time chevrons: down while closed, up while the popup is open. The font popup is forced to the control width instead of expanding to the widest installed font name. `QFontComboBox` stays editable with case-insensitive `MatchContains` completion. The text-size spin box paints explicit upper/lower chevrons over the native step hit areas.

## Theme, icons and calendar

`applyAppearance()` is the single theme owner. It defines semantic colors for background, surface, border, text, muted/disabled text, hover, accent and accent-soft states and writes them into the application `QPalette` before applying QSS. System mode listens to `QStyleHints::colorSchemeChanged`, so an OS Light/Dark change is reflected while NoteHub is running.

Custom icons no longer embed fixed gray/blue pixmaps. `PaletteIconEngine` resolves its color from the live palette at paint time, and the global pixmap cache is cleared after a theme switch.

`QCalendarWidget` needs additional handling because it owns private navigation buttons and an internal item view. NoteHub therefore:

- replaces previous/next arrows with palette-aware chevrons;
- explicitly applies the app palette to the calendar and its item-view viewport;
- uses a non-transparent table background to avoid the Windows/Fusion black backing-store bug in Light mode;
- regenerates weekday, weekend and adjacent-month date formats from semantic theme colors;
- reapplies the current accent to dates that contain notes.

Both the mini context calendar and the full Calendar page use the same algorithm. Switching theme refreshes existing calendar instances immediately.

The startup font prefers `Segoe UI Variable Text` on supported Windows systems, falls back to `Segoe UI`, then falls back to Qt's platform font. User-selected font and size settings override the startup choice. No font files are bundled.

## Edit Note and clipboard algorithm

`NoteEditor` now uses palette-aware vector toolbar icons for bold, italic, underline, strike, text color, highlight, clear formatting, lists, indentation, undo/redo and attachment. The attachment action emits `attachRequested()`; `MainWindow` performs the existing Go attachment RPC without showing the old attachment list inside the dialog. Reload Latest, Attach files, Remove selected and the attachment preview/list region are removed from the edit dialog, leaving the document as the primary surface.

Paste handling is defensive:

1. read both rich HTML and plain text from `QMimeData`;
2. render the HTML into a temporary `QTextDocument`;
3. compare its newline structure with the clipboard text;
4. if HTML preserves the visible lines, insert HTML and retain rich formatting;
5. otherwise insert normalized plain text so commands/chat text cannot collapse into one line.

When saving rich text, NoteHub serializes the editor document to HTML and immediately round-trips that HTML through another `QTextDocument`. If visible line structure changes, it falls back to a safe plain-document HTML representation rather than persisting a flattened note. Untouched legacy Markdown remains byte-for-byte unchanged.

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
