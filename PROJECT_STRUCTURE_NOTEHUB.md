# NoteHub – Current Project Structure

## Technology stack

NoteHub now runs as two cooperating desktop processes:

- **C++17 + Qt 6 Widgets**: native desktop GUI (`NoteHub.exe`).
- **Go**: domain/service/repository/storage backend (`notehub-core.exe`).
- **SQLite + FTS5**: local database and text search.
- **Filesystem**: attachment storage.
- **JSON Lines over stdin/stdout**: local IPC managed by `QProcess`.

The previous Go/Fyne GUI is kept only under `_legacy_fyne/` for reference and is not part of the production desktop build.

## Runtime dependency direction

```text
Qt GUI
  │
  │ JSON IPC
  ▼
Go IPC server
  ▼
Service
  ▼
Repository / Storage
  ├── SQLite + FTS5
  └── Filesystem
```

The Qt GUI never accesses SQLite directly. The Go backend remains the single owner of persistence and business rules.

## Current source tree

```text
NoteHub/
├── CMakeLists.txt
├── desktop/
│   ├── CMakeLists.txt
│   ├── resources.qrc
│   ├── src/
│   │   ├── main.cpp
│   │   ├── BackendClient.h
│   │   ├── BackendClient.cpp
│   │   ├── MainWindow.h
│   │   ├── MainWindow.cpp
│   │   ├── NoteEditor.h
│   │   ├── NoteEditor.cpp
│   │   ├── ImageGallery.h
│   │   └── ImageGallery.cpp
│   └── tests/
│       ├── FakeCore.cpp
│       └── UiTests.cpp
├── cmd/
│   └── notehub-core/
│       └── main.go
├── internal/
│   ├── app/
│   ├── domain/
│   ├── ipc/
│   ├── service/
│   ├── repository/
│   │   └── sqlite/
│   ├── storage/
│   ├── backup/
│   ├── share/
│   ├── tagparse/
│   ├── notecontent/
│   └── platform/
├── assets/
│   ├── icons/
│   │   ├── NoteHub.png
│   │   └── NoteHub_White.png
│   └── images/
├── scripts/
│   ├── build-windows.ps1
│   ├── deploy-windows.ps1
│   ├── build-linux.sh
│   └── build-macos.sh
├── tests/
│   └── integration/
├── _legacy_fyne/
├── docs/
│   └── ARCHITECTURE.md
├── go.mod
├── go.sum
└── README.md
```

## Desktop responsibilities

### `desktop/src/main.cpp`

Creates `QApplication`, selects the stable Fusion base style, chooses a platform-appropriate default font, parses command-line options and opens `MainWindow`. The default backend is resolved beside the GUI executable as `notehub-core.exe` on Windows.

### `desktop/src/BackendClient.*`

Owns the Go child process and the JSON-lines transport. It validates reply IDs/protocol, keeps stderr diagnostics separate from stdout, enforces request size/time limits and stops the backend cleanly when the GUI exits.

### `desktop/src/MainWindow.*`

Owns the desktop shell and page routing:

- responsive left navigation rail;
- embedded NoteHub logo;
- global `Ctrl+K` search;
- Home composer;
- timeline/Favorites/Shared views;
- Calendar;
- Attachments manager;
- Tags browser;
- Settings;
- right-side calendar + Quick Filters.

The navigation rail is resizable through a narrow `QSplitter` handle (64–280 px). Under 136 px it becomes icon-only; wider rails show icon + text. The last expanded width is persisted and the header toggle jumps between 68 px and that saved expanded width. The right context panel is hidden below 1180 px.

`MainWindow` also owns the application palette/QSS. Standard Qt platform icons were replaced by `QPainter` glyphs served through a palette-aware `QIconEngine`, so icons repaint correctly after Light/Dark/System changes. The calendar receives the same semantic palette explicitly, including its private item viewport and navigation arrows.

### `desktop/src/ImageGallery.*`

Loads previews asynchronously and caches bounded decoded images. Gallery geometry is now a stable grid of one to four columns. Incomplete final rows keep the same tile width as earlier rows, preventing the oversized previews visible in the previous Qt UI.

### `desktop/src/NoteEditor.*`

Provides rich text plus Markdown-source editing. The editor uses palette-aware vector toolbar icons and exposes attachment import as a toolbar action. Clipboard HTML is accepted only when its rendered line structure matches the clipboard text; otherwise multi-line plain text is used to prevent flattening. Rich save performs an HTML line-round-trip guard. The Edit Note dialog no longer contains Reload Latest, a separate attachment list or attachment/remove buttons. Existing revision/conflict semantics are unchanged because saving still goes through the Go backend.

## Go backend responsibilities

### `cmd/notehub-core`

Starts `internal/app.OpenBackend`, then serves the local IPC protocol.

### `internal/ipc`

Maps JSON IPC methods to backend services. stdout is protocol-only; diagnostics go to stderr.

### `internal/app`

Composition root: resolves data directory, opens SQLite, runs migrations, constructs repositories/services/storage and closes resources.

### `internal/domain`

Pure business data types such as Memo, Attachment, Tag, Calendar and Share.

### `internal/service`

Business operations for memo CRUD, attachments, timeline, calendar, search, tags, sharing and backup.

### `internal/repository/sqlite`

SQLite schema, migrations and repository implementation. The UI never imports this package.

### `internal/storage`

Attachment filesystem paths, checksums and safe file operations.

### `internal/backup`

ZIP export/import and safe archive path validation.

### `internal/share`

Optional loopback HTTP sharing server.

## GUI algorithms changed in the current redesign

1. **Draggable shell**: 64–280 px splitter-driven navigation rail; icon-only below 136 px; toggle restores the last expanded width; right panel hides below 1180 px.
2. **Theme-aware icon engine**: `QPainter` glyphs resolve neutral/active/disabled/checked colors from the live palette; Home/Attachments/Settings use explicit house/paperclip/gear shapes.
3. **Calendar theming**: calendar table viewport, navigation arrows, weekdays/weekends, adjacent-month dates and note-day accents are regenerated from semantic theme colors; no transparent black backing store in Light mode.
4. **Visual hierarchy**: app background → rail → workspace/context cards → content cards; accent color only for interaction/focus.
5. **Viewport anchor after edits**: memo refresh records the edited card's viewport Y position and restores the rebuilt card to that position instead of jumping to the first note.
6. **Attachment staging**: shows a compact count plus tooltip rather than wrapping all staged filenames into the composer.
7. **Timeline preview**: note text preview is bounded and captures wheel input at its own top/bottom boundaries so hovering one memo never scrolls adjacent cards.
8. **Gallery grid**: 1–4 columns, equal tile widths across all rows, preview height bounded to 130–205 px; drag target uses the current theme accent.
9. **Optimistic favorite UX**: favorite button changes immediately, is locked while saving and reverts on backend error.
10. **Compact settings controls**: Theme/Font combos have dynamic down/up chevrons, font popup width matches the field, editable font search uses substring completion, and Text size has explicit up/down steppers.
11. **Tags**: card grid instead of full-width centered rows.
12. **Quick Filters**: live note/favorite/shared counts are shown in the right context panel instead of a permanent status bar.
13. **Rich paste/editor toolbar**: modern vector formatting icons, toolbar attachment action, HTML/plain-text line-structure validation and removal of the old attachment/reload controls from Edit Note.

See `docs/ARCHITECTURE.md` for the algorithms and interaction rationale in detail.
