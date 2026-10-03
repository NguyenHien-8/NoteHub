# Qt / Go desktop migration

The requested target is a native C++17 / Qt 6 Widgets desktop owning one Go
backend child process. Preserve existing SQLite data, attachments, revisions,
favorites, Unicode search, calendar, tags, backups and explicit local sharing.
Carry forward maximized startup, compact icon navigation, responsive image
gallery with native drag feedback, and a fully formatted note editor with
Markdown import/export. Do not build NoteHub.exe during this task.

## Architecture and contract

- `desktop/`: Qt Widgets, QTextEdit, QCalendarWidget, QSettings and QProcess.
- `cmd/notehub-core/`: headless Go entry point; stdin/stdout framed JSON IPC.
- `internal/ipc/`: protocol v1 adapter over existing app/services. Requests
  have string `id`, `method`, object `params`; one UTF-8 JSON record per line.
  Responses echo `id`, `result` or structured `error`. Only stdout is IPC;
  diagnostic logging goes to stderr. Maximum request size is 8 MiB.
- Qt owns the process, starts it by absolute sibling path, performs a version
  handshake, handles fragmented/coalesced responses asynchronously, and closes
  stdin on shutdown. No listening socket is needed for desktop IPC.
- Stable memo/attachment UIDs cross IPC; database IDs remain private. Timeline
  cursors are opaque. Revisions remain optimistic, including on favorite edits.
- Retain raw Markdown notes. Rich editor saves a versioned HTML marker in the
  same content field; extract visible text for tags/FTS. No schema rewrite.
  Export .md warns that unsupported visual formatting is not retained.
- Existing Fyne source (including the interrupted UI work) is preserved under
  `_legacy_fyne/` for reference, excluded from active Go package discovery.

## Implementation sequence

1. Freeze Fyne source; remove UI dependencies from active backend. Add IPC
   tests for framing, malformed requests, CRUD/revision conflict, order, search,
   backup and EOF. Implement the headless process and protocol adapter.
2. Implement Qt process client with bounded buffers, deadlines, failure fanout,
   startup handshake and orderly shutdown. Implement native rich editor and
   responsive gallery; keep UI asynchronous and preserve drafts on failures.
3. Connect Home, Calendar, Search, Attachments, Tags, Settings and sharing to
   real backend methods; use stable UIDs, bounded pages and stale-result guards.
4. Replace build scripts/CMake/CI and document packaging two executables,
   deployment, compatibility, IPC and manual verification.
5. Run Go tests/vet and C++ compiler syntax checks / test-only Qt targets.
   Do not link or package the NoteHub desktop executable. Record precisely
   which native/runtime checks remain unverified without the user's build.

## Review focus

Fragmented JSON, process death during save, stale editor conflicts, closing with
unsaved changes, non-image attachment slots in reordered galleries, existing
data imports, rich text search/tag noise, Unicode paths, and Qt runtime DLLs.

## Execution record

Implementation is in the current workspace, as requested. Previous Fyne
subagents stopped when the user changed the architecture request. Independent
agent review is unavailable because the agents returned an account usage-limit
error; final review will be done directly, with tests and compiler diagnostics.
