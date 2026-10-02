# NoteHub Database

Primary database: SQLite through `modernc.org/sqlite` (pure Go).

Main tables:

- `memo`
- `memo_tag`
- `attachment`
- `memo_share`
- `schema_migration`
- `memo_fts` (FTS5 virtual table)

Important settings:

- WAL journal mode
- 10-second busy timeout
- foreign keys enabled
- synchronous NORMAL
- mmap disabled
- IMMEDIATE write transactions

Attachments are stored on the filesystem and referenced by a safe relative path.
