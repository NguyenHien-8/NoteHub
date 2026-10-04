# NoteHub

NoteHub is a native desktop note application with a **C++/Qt 6 frontend** and a **Go backend**. Notes, tags, favorites, attachments and backups remain local in SQLite + the application data directory.

## Runtime layout

```text
NoteHub.exe (Qt GUI)
        │
        │ JSON IPC through QProcess stdin/stdout
        ▼
notehub-core.exe (Go)
        │
        ├── SQLite + FTS5
        └── attachments/
```

`NoteHub.exe` and `notehub-core.exe` must stay in the same packaged application directory unless `--backend` is used explicitly.

## Windows build

The current Windows script targets a Qt MinGW kit and Ninja. Example for Qt 6.11.2:

```powershell

Remove-Item ".\build\qt-release" -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item ".\dist\windows" -Recurse -Force -ErrorAction SilentlyContinue
```

```powershell

cd E:\ALL_PROJECTS\GIT_REPOSITORY\OFFICE_EDITOR\NoteHub

.\scripts\build-windows.ps1 `
    -QtRoot "C:\Qt\6.11.2\mingw_64" `
    -Compiler "C:\Qt\Tools\mingw1310_64\bin\g++.exe" `
    -Clean `
    -ForceCloseRunning
```

Run the packaged application from:

```powershell
.\dist\windows\NoteHub.exe
```

Do not distribute only `NoteHub.exe`. The complete `dist\windows` directory contains `notehub-core.exe`, Qt DLLs, platform plugins and the MinGW runtime required by the executable.

## Desktop UX

The Qt shell is intentionally designed as a modern three-area desktop layout:

- draggable compact/expanded navigation rail on the left;
- note workspace in the center;
- contextual calendar + Quick Filters on the right when enough width is available.

The header contains the embedded NoteHub logo and global search. `Ctrl+K` focuses search. The navigation/workspace boundary is a narrow `QSplitter` handle: it stays visually quiet normally, becomes blue on hover/drag and lets the rail be resized continuously between compact and expanded widths. The header sidebar button still toggles quickly between compact and the last expanded width.

The UI uses Qt Fusion as a stable cross-platform base and one semantic Light/Dark/System theme. Navigation/action icons are palette-aware vector-like `QPainter` glyphs, so their neutral/active/disabled colors change immediately with the theme instead of keeping Light-theme pixels.

### Home

The composer supports text, staged attachments and direct tag insertion. Attachment staging displays a small count instead of a long filename list. Save first creates the memo, then imports attachments; partial attachment failures do not discard the saved memo.

Timeline cards contain timestamp, favorite, edit, share/delete menu, rendered note preview, image gallery, file chips and clickable tag chips. Saving or changing attachments on a lower card preserves that memo's viewport position instead of rebuilding the timeline at the first note. Wheel input is contained by the active note card while the pointer remains over it; the text preview scrolls internally and reaching any card boundary does not unexpectedly start scrolling adjacent memo cards.

### Images

Image previews are decoded outside the UI thread and cached. The gallery now uses a predictable 1–4 column grid. An incomplete final row does not stretch its remaining images to the full note width, preventing giant previews. Dragging still changes persisted attachment order.

### Attachments

The attachment page uses compact information rows with filename, local path, Open note and Delete controls. Clicking the filename opens the file in its system application.

### Tags

Tags are displayed as cards with their counts. Selecting a tag returns to Home with that filter applied.

### Calendar and themes

The mini and full calendars are themed explicitly rather than relying on private `QCalendarWidget` defaults. Calendar table backgrounds, date text, weekend/muted dates, note-day accent markers and previous/next icons all follow Light/Dark/System. System mode also reacts to an OS color-scheme change while NoteHub is running.

### Settings

Settings are separated into:

- Appearance;
- Data & backup;
- Local sharing.

Appearance controls stay compact instead of stretching across the page. Theme and font controls render a live down/up chevron depending on popup state; the font popup is constrained to the same width as the Calibri/font field. The text-size control has explicit up/down steppers. The font field remains searchable with installed-font suggestions and substring completion.

### Editing and clipboard fidelity

The Edit Note dialog is intentionally document-focused: the old Reload Latest button, attachment list, Attach files button and Remove selected button are removed. Attachment import is now a paperclip action on the editor toolbar. The toolbar uses palette-aware vector glyphs instead of text-symbol buttons.

Clipboard paste prefers rich HTML when it round-trips the clipboard's visible line structure, preserving Word/ChatGPT formatting such as bold, lists, colors and headings. If a source application's HTML collapses line breaks, NoteHub falls back to its plain-text clipboard payload so commands and multi-line text are never silently flattened. Before rich content is persisted, the editor verifies that Qt's HTML serialization can round-trip the visible line breaks.

## Data directory

Default locations are resolved by the Go backend:

| Platform | Directory |
| --- | --- |
| Windows | `%LOCALAPPDATA%\NoteHub` |
| macOS | `~/Library/Application Support/NoteHub` |
| Linux | `$XDG_DATA_HOME/NoteHub` or `~/.local/share/NoteHub` |

Use an isolated directory with:

```powershell
.\dist\windows\NoteHub.exe --data-dir "D:\My Notes\NoteHub"
```

## Backend guarantees retained by the Qt migration

The GUI redesign does not change the database format. The Go backend still owns:

- additive SQLite migrations;
- memo revision/conflict checks;
- FTS5 search;
- calendar and timeline queries;
- tag parsing;
- attachment checksum/storage;
- safe ZIP backup/import;
- hashed share tokens;
- loopback-only optional sharing.

Existing `notehub.db` and `attachments/` remain compatible.

## Source map

- `desktop/src/MainWindow.*` — desktop shell, navigation, pages and theme.
- `desktop/src/NoteEditor.*` — rich text/Markdown editor.
- `desktop/src/ImageGallery.*` — async gallery preview and reorder.
- `desktop/src/BackendClient.*` — Go process and IPC transport.
- `cmd/notehub-core` — Go backend executable.
- `internal/ipc` — IPC method dispatch.
- `internal/service` — business logic.
- `internal/repository/sqlite` — SQLite implementation.
- `internal/storage` — attachment filesystem.

See [Architecture](docs/ARCHITECTURE.md) and [Project structure](PROJECT_STRUCTURE_NOTEHUB.md) for the current Qt + Go algorithms.

Licensed under [MIT](LICENSE).
