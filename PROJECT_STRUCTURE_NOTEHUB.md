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

The redesign uses fixed responsive breakpoints rather than continuously shrinking controls. The left rail is 68 px compact or 212 px expanded. The right context panel is hidden below 1180 px. This preserves center workspace width and avoids clipped labels.

`MainWindow` also owns the application palette/QSS. Standard Qt platform icons were replaced by high-DPI line icons painted with `QPainter`, so Windows no longer shows a mixture of old Explorer-style folder/file icons inside the modern shell.

### `desktop/src/ImageGallery.*`

Loads previews asynchronously and caches bounded decoded images. Gallery geometry is now a stable grid of one to four columns. Incomplete final rows keep the same tile width as earlier rows, preventing the oversized previews visible in the previous Qt UI.

### `desktop/src/NoteEditor.*`

Provides rich text plus Markdown-source editing. The editor uses the same application theme, a fixed non-floating toolbar and document-style tabs. Existing revision/conflict semantics are unchanged because saving still goes through the Go backend.

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

1. **Responsive shell**: compact rail under 980 px or by user preference; right panel under 1180 px; center workspace receives remaining width.
2. **Stable icon system**: `QPainter` renders normal/selected high-DPI pixmaps instead of platform standard icons.
3. **Visual hierarchy**: app background → rail → workspace/context cards → content cards; accent color only for interaction/focus.
4. **Attachment staging**: shows a compact count plus tooltip rather than wrapping all staged filenames into the composer.
5. **Timeline preview**: note text preview is bounded to keep scanning predictable; full editing stays in the editor dialog.
6. **Gallery grid**: 1–4 columns, equal tile widths across all rows, preview height bounded to 130–205 px.
7. **Optimistic favorite UX**: favorite button changes immediately, is locked while saving and reverts on backend error.
8. **Tags**: card grid instead of full-width centered rows.
9. **Settings**: grouped by Appearance, Data & backup and Local sharing.
10. **Quick Filters**: live note/favorite/shared counts are shown in the right context panel instead of a permanent status bar.

See `docs/ARCHITECTURE.md` for the algorithms and interaction rationale in detail.
