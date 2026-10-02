# Third-party notice

NoteHub backend design and selected algorithms were adapted from the open-source
Memos project (https://github.com/usememos/memos), licensed under the MIT
License. The upstream license text is preserved in `LICENSE-MEMOS`.

The integration is intentionally selective: NoteHub does not embed the Memos
web frontend, multi-user/server stack, protobuf API, Spaces, reactions, S3
storage, or CEL filter system. The retained ideas were rewritten for a local
cross-platform Go + SQLite desktop architecture.
