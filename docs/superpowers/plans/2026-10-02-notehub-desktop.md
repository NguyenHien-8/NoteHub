# NoteHub Desktop implementation plan

> Execute the supplied desktop specification in this checkout. Use test-driven backend changes and parallel agents only for independent file ownership; integrate and verify the complete application before delivery.

**Goal:** Complete the existing Go backend as a working, persistent native Fyne desktop app matching the supplied three-column reference.

**Spec:** `docs/PROMPT_BUILD_NOTEHUB_DESKTOP.md` (user-supplied requirements).

**Architecture:** Fyne components are presentation-only. Screens and dialogs use `app.Backend` services through cancellable background jobs; widget updates use `fyne.Do`. Existing repositories, storage, search, backup and share algorithms remain in place. Additive migration v2 adds favorites.

**Tech stack:** Go, Fyne v2, modernc SQLite, embedded PNG logo, OS-native file selection.

## Constraints and decisions

- Work directly in the user-requested project; preserve existing documentation and tests.
- Window defaults 1450 × 900, minimum 1100 × 700; left navigation about 250, right rail about 280, collapses on narrow widths.
- Home / Calendar / Search / Attachments / Tags / Settings; no Recent Tags panel or sample data.
- Real service counts, deterministic tag colors, keyset timeline pages of 40, local date boundaries, 300 ms search debounce.
- Revision conflicts retain the editor; failed attachment imports do not undo a saved memo.
- Share server disabled at every startup; explicit user activation on loopback only. Tokens appear only at creation.
- Keep schema v1 unchanged. Preserve favorites in backups using an optional compatible field.
- Root implements app/screens/dialogs; backend agent owns backend changes/tests; component agent owns presentation components. Shared interfaces below prevent file conflicts.

## Review focus

1. Windows drive-letter paths with spaces/Unicode must open SQLite and survive restart.
2. Migration of an existing v1 database must retain memos, tags, attachments, FTS and revisions.
3. Favorite/shared filters must paginate without duplicates; expired/revoked/multiple shares must count each memo correctly.
4. Slow operations must not overwrite newer search/navigation results or mutate widgets from workers.
5. Attachment/save/import failures and closing with unsaved edits must preserve user content and explain partial results.

## Tasks

- [ ] Audit source, reproduce baseline SQLite failure, add regression test and fix its cause.
- [ ] Extend `domain/memo.go`, repository interfaces, SQLite migration/query implementations and services. APIs: `Memo.Favorite bool`; `TimelineQuery.FavoriteOnly/SharedOnly bool`; `MemoService.SetFavorite(ctx,id,bool) error`; `TimelineService.Counts(ctx) (domain.MemoCounts,error)` with `All,Favorites,Shared int`; `AttachmentService.ListAll(ctx,limit,offset) ([]domain.Attachment,error)`; `CalendarService.PageForDate(ctx,date,loc,limit,cursor)` returning timeline page. Preserve backup favorites. Tests cover migration, persistence, filters/counts, expiry, attachment listing and backup round trip.
- [ ] Add pinned Fyne dependency, embed logo in `assets/assets.go`, wire `cmd/notehub/main.go`; preserve safe backend shutdown and startup-error feedback.
- [ ] Implement reusable `internal/ui/components`: cards, editor, timeline, sidebar, tags, calendar and search input. Components accept callbacks and values only. Pure date/title/tag helpers and Fyne widget behavior get tests.
- [ ] Implement `internal/ui/theme.go`, `window.go`, `navigation.go`, asynchronous lifecycle and responsive layout. Wire screen refresh and safe close.
- [ ] Implement `screens/{timeline,calendar,search,attachments,tags,settings}.go`: CRUD, paginated real results, date/tag/favorite/shared filtering, thumbnail preparation off UI thread and empty/loading/error states.
- [ ] Implement `dialogs/{attachment,confirm,share,export,import}.go`, native chooser wrapper in platform, edit conflict handling, share grants/revoke/expiry, backups with policies and report.
- [ ] Test complete desktop behavior with Fyne headless tests and actual native compile/launch where available. Verify no SQL in UI, stale callback guards and clean shutdown.
- [ ] Implement native Windows/macOS/Ubuntu CI and release scripts, build instructions, verification report and file manifest.
- [ ] Run gofmt, `go mod tidy`, `go test ./...`, `go vet ./...`, `go build ./cmd/notehub`; report exact results and unavailable platform checks. Deliver direct project edits. The user's later instruction cancels ZIP creation and delivery.

## Audit baseline

- Existing working tree clean; Go 1.26.2 available, CGO disabled and gcc absent from PATH.
- Entry point/UI/CI/build scripts are placeholders; backend already supplies all main services except favorite/count/attachment-list APIs.
- Initial `go test ./...` fails in `TestBackendFlow` while opening SQLite: `SQL logic error: out of memory (1)`. All existing pure unit packages pass. Diagnose before claiming backend stability.
- Sandbox initially denied Go cache access; escalated test execution succeeded in reaching the actual tests.
