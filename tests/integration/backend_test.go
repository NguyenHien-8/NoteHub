//go:build !teststub

package integration

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/NguyenHien-8/NoteHub/internal/domain"
	"github.com/NguyenHien-8/NoteHub/internal/repository"
	"github.com/NguyenHien-8/NoteHub/internal/repository/sqlite"
	"github.com/NguyenHien-8/NoteHub/internal/service"
	"github.com/NguyenHien-8/NoteHub/internal/storage"
)

type fixture struct {
	store       *sqlite.Store
	files       *storage.LocalStore
	memo        *service.MemoService
	attachments *service.AttachmentService
	timeline    *service.TimelineService
	calendar    *service.CalendarService
	search      *service.SearchService
	shares      *service.ShareService
	backup      *service.BackupService
}

func newFixture(t *testing.T, root string) *fixture {
	t.Helper()
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(root, "notehub.db"))
	if err != nil {
		t.Fatal(err)
	}
	files, err := storage.NewLocalStore(filepath.Join(root, "attachments"), 16<<20)
	if err != nil {
		t.Fatal(err)
	}
	opts := service.Options{MaxMemoBytes: 1 << 20, MaxAttachmentBytes: 16 << 20}
	timeline := service.NewTimelineService(store, store, store)
	f := &fixture{store: store, files: files, timeline: timeline}
	f.memo = service.NewMemoService(store, store, store, files, opts)
	f.attachments = service.NewAttachmentService(store, store, files)
	f.calendar = service.NewCalendarService(store, timeline)
	f.search = service.NewSearchService(store, store, store)
	f.shares = service.NewShareService(store, store, store, store, files)
	f.backup = service.NewBackupService(store, store, store, files, opts, "test")
	t.Cleanup(func() { _ = store.Close() })
	return f
}

func TestBackendFlow(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, t.TempDir())

	memo, err := f.memo.Create(ctx, "Ghi chú FPGA #FPGA #Research/FPGA")
	if err != nil {
		t.Fatal(err)
	}
	if len(memo.Tags) != 3 {
		t.Fatalf("tags=%v", memo.Tags)
	}
	att, err := f.attachments.Add(ctx, memo.ID, "test.txt", "text/plain", bytes.NewBufferString("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if att.Size != 5 {
		t.Fatalf("attachment size=%d", att.Size)
	}

	items, _, err := f.timeline.List(ctx, repository.TimelineQuery{Limit: 10, Tags: []string{"FPGA"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || len(items[0].Attachments) != 1 {
		t.Fatalf("timeline=%+v", items)
	}

	found, err := f.search.Search(ctx, repository.SearchQuery{Text: "ghi chu", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 {
		t.Fatalf("accent-insensitive search found %d", len(found))
	}

	day := memo.CreatedAt.In(time.Local).Format("2006-01-02")
	byDay, err := f.calendar.MemosForDate(ctx, day, time.Local, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(byDay) != 1 {
		t.Fatalf("calendar found %d", len(byDay))
	}

	updated, err := f.memo.Update(ctx, memo.ID, memo.Revision, "Updated #FPGA")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.memo.Update(ctx, memo.ID, memo.Revision, "stale update"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected revision conflict, got %v", err)
	}
	if updated.Revision != memo.Revision+1 {
		t.Fatalf("revision=%d", updated.Revision)
	}

	grant, err := f.shares.Create(ctx, memo.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	shared, err := f.shares.Resolve(ctx, grant.Token)
	if err != nil {
		t.Fatal(err)
	}
	if shared.Memo.UID != memo.UID {
		t.Fatalf("shared uid=%s", shared.Memo.UID)
	}
	var hashLen int
	if err := f.store.DB().QueryRowContext(ctx, `SELECT length(token_hash) FROM memo_share WHERE uid=?`, grant.Share.UID).Scan(&hashLen); err != nil {
		t.Fatal(err)
	}
	if hashLen != 32 {
		t.Fatalf("token hash length=%d", hashLen)
	}

	exportPath := filepath.Join(t.TempDir(), "backup.zip")
	if err := f.backup.ExportFile(ctx, exportPath); err != nil {
		t.Fatal(err)
	}
	other := newFixture(t, t.TempDir())
	report, err := other.backup.ImportFile(ctx, exportPath, service.ImportSkip)
	if err != nil {
		t.Fatal(err)
	}
	if report.Created != 1 || len(report.Failures) != 0 {
		t.Fatalf("report=%+v", report)
	}
	imported, err := other.memo.GetByUID(ctx, memo.UID)
	if err != nil {
		t.Fatal(err)
	}
	if len(imported.Attachments) != 1 || imported.Attachments[0].SHA256 != att.SHA256 {
		t.Fatalf("imported=%+v", imported)
	}
}
