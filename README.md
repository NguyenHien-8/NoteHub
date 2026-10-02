# NoteHub

NoteHub 0.2.0 is a native Go/Fyne desktop application for Windows, macOS, and
Linux. Write notes in a timeline, attach local files, and find them again with
tags, calendar dates, favorites, or Unicode search. Notes and attachments stay
on your computer in SQLite and the application data directory.

## Start the desktop app

For a built package, open `NoteHub.exe` on Windows, `NoteHub.app` on macOS, or
run `./NoteHub` on Linux. The app opens its own desktop window. The logo is
embedded, so launching from another working directory works.

To build from source, install Go and the native C/OpenGL development tools for
your operating system, then run the appropriate script from this checkout:

```powershell
# Windows PowerShell
./scripts/build-windows.ps1
./dist/windows/NoteHub.exe
```

```bash
# macOS: produces an app bundle, ZIP, and DMG
bash scripts/build-macos.sh
open dist/macos/NoteHub.app

# Linux: produces a binary and tar.gz
bash scripts/build-linux.sh
./dist/linux/NoteHub
```

Linux needs `zenity` for native file selection and `xdg-utils` for opening files
and folders. A graphical desktop session and OpenGL support are required on all
platforms. See [build instructions](docs/BUILD.md) for dependencies, versioned
builds, direct Go commands, and release artifacts.

See the [desktop verification report](docs/DESKTOP_TEST_REPORT.md) for the
commands run, Windows build results, changed files, and platform checks still
awaiting CI verification.

## Use NoteHub

- **Home:** write a note and click **Save**. **Attach** stages files until the
  note is saved (**Đính kèm** in the composer). **Add tag** (**Thêm tag**)
  inserts a hashtag into the text; tags such as
  `#Research/FPGA` support hierarchy. Use **Load more** to browse older notes.
- **Note menu:** edit, favorite, share, or delete a note. Delete asks for
  confirmation. Edits use revisions; a conflict requires reloading instead of
  silently overwriting another change.
- **Calendar:** select a day to view its notes. Clear the date filter to return
  to all notes. The small calendar and quick filters provide the same navigation
  from Home.
- **Search:** press **Ctrl+K** to focus global search, or open Search for text,
  tag, and date filters. Diacritic-insensitive search lets `ghi chu` match
  `Ghi chú`.
- **Attachments:** browse files, open them in the system application, reveal
  their location, delete an attachment, or go to its note. Image files display
  thumbnails; PDF and other files display file cards.
- **Tags:** select a tag to filter Home. **All Notes**, **Favorites**, and
  **Shared** show live counts; Shared counts notes with active, unexpired shares.
- **Settings:** choose System, Light, or Dark appearance; open the data folder;
  export or import a ZIP backup; configure local sharing; and view the version.
  Backup import offers **Skip**, **Replace**, or **Duplicate** for conflicts.

A saved note remains saved if one of its attachments fails to import; the app
reports the failed file. Export backups to a new destination filename. Closing
and reopening the application retains your saved notes, tags, favorites, and
attachments. Existing databases are upgraded through additive migrations.

## Data and sharing

The default data directory is:

| Platform | Directory |
| --- | --- |
| Windows | `%LOCALAPPDATA%\NoteHub` |
| macOS | `~/Library/Application Support/NoteHub` |
| Linux | `$XDG_DATA_HOME/NoteHub`, or `~/.local/share/NoteHub` when unset |

It contains `notehub.db` and `attachments/`. Settings shows the resolved path.
Use Export Backup to obtain a consistent ZIP of notes and attachments.
To use an isolated or portable data directory, launch with
`--data-dir "/path/to/NoteHub data"` (Windows example:
`./dist/windows/NoteHub.exe --data-dir 'D:\My Notes\NoteHub'`). The selected
directory becomes the root for the database and attachments. `--version`
prints the application version without opening the desktop window.

The local share server is **disabled on every launch**. Enable it explicitly
in Settings and choose the port before opening a share link. It binds only to
`127.0.0.1`; links work on the same computer. A share grant can have no expiry,
or expire after 1, 7, or 30 days, and can be revoked. Its raw token is shown only
when created; the database stores its SHA-256 hash. A saved share grant does
not start the server when the app reopens.

## Development and architecture

The GUI calls `internal/app.Backend` services. Services use repositories and
storage; the GUI never accesses SQLite directly. Fyne widget updates return to
the UI thread, while database/file work runs in background jobs.

With native dependencies installed and `CGO_ENABLED=1`:

```bash
go mod tidy
go test ./...
go vet ./...
go build ./cmd/notehub
```

UI tests use Fyne's headless test app. Linux CI also supplies Xvfb for packages
that compile against the native driver. The `teststub` build tag is for legacy
offline backend type checking only; use normal tests and builds for the desktop
application.

[Desktop CI](.github/workflows/test.yml) runs formatting, module cleanliness,
tests, vet, and native compilation on all three operating systems.
[Release artifacts](.github/workflows/release.yml) packages native binaries when
a `vX.Y.Z` tag is pushed. Workflow definitions do not imply that remote jobs
have already run; inspect the Actions results for platform verification.

Existing backend reference documents remain available:

- [Architecture](docs/ARCHITECTURE.md)
- [Backend selection from Memos](docs/BACKEND_FROM_MEMOS.md)
- [Original backend test report](docs/BACKEND_TEST_REPORT.md)
- [Backend change summary](BACKEND_CHANGE_SUMMARY.md)
- [Project structure](PROJECT_STRUCTURE_NOTEHUB.md)
- [Database](docs/DATABASE.md)
- [Desktop implementation specification](docs/PROMPT_BUILD_NOTEHUB_DESKTOP.md)

Licensed under [MIT](LICENSE).
