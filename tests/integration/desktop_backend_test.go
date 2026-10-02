//go:build !teststub

package integration

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/NguyenHien-8/NoteHub/internal/domain"
	"github.com/NguyenHien-8/NoteHub/internal/repository"
	"github.com/NguyenHien-8/NoteHub/internal/service"
)

func createMemo(t *testing.T, f *fixture, content string) *domain.Memo {
	t.Helper()
	m, err := f.memo.Create(context.Background(), content)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestFavoritePersistenceAndRevision(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	f := newFixture(t, root)
	m := createMemo(t, f, "Ghi chú #FPGA")
	if err := f.memo.SetFavorite(ctx, m.ID, true); err != nil {
		t.Fatal(err)
	}
	got, err := f.memo.Get(ctx, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Favorite || got.Revision != m.Revision+1 {
		t.Fatalf("favorite memo=%+v", got)
	}
	if _, err := f.memo.Update(ctx, m.ID, m.Revision, "stale"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("stale update=%v", err)
	}
	if err := f.memo.SetFavorite(ctx, m.ID, true); err != nil {
		t.Fatal(err)
	}
	gotAgain, err := f.memo.Get(ctx, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotAgain.Revision != got.Revision {
		t.Fatal("idempotent favorite bumped revision")
	}
	updated, err := f.memo.Update(ctx, m.ID, got.Revision, "Updated #FPGA")
	if err != nil || !updated.Favorite {
		t.Fatalf("updated=%+v err=%v", updated, err)
	}
	found, err := f.search.Search(ctx, repository.SearchQuery{Text: "Updated"})
	if err != nil || len(found) != 1 || !found[0].Favorite {
		t.Fatalf("search=%+v err=%v", found, err)
	}
	if err := f.store.Close(); err != nil {
		t.Fatal(err)
	}
	f = newFixture(t, root)
	got, err = f.memo.Get(ctx, m.ID)
	if err != nil || !got.Favorite {
		t.Fatalf("reopened=%+v err=%v", got, err)
	}
	if err := f.memo.SetFavorite(ctx, m.ID, false); err != nil {
		t.Fatal(err)
	}
	got, err = f.memo.Get(ctx, m.ID)
	if err != nil || got.Favorite {
		t.Fatalf("unfavorite=%+v err=%v", got, err)
	}
	if err := f.memo.SetFavorite(ctx, 999999, true); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing memo=%v", err)
	}
}

func TestCountsSharedExpiryAndRevoke(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, t.TempDir())
	counts, err := f.timeline.Counts(ctx)
	if err != nil || counts != (domain.MemoCounts{}) {
		t.Fatalf("empty counts=%+v err=%v", counts, err)
	}
	a := createMemo(t, f, "favorite shared")
	b := createMemo(t, f, "expired")
	createMemo(t, f, "private")
	if err := f.memo.SetFavorite(ctx, a.ID, true); err != nil {
		t.Fatal(err)
	}
	g1, err := f.shares.Create(ctx, a.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(time.Hour)
	g2, err := f.shares.Create(ctx, a.ID, &future)
	if err != nil {
		t.Fatal(err)
	}
	g3, err := f.shares.Create(ctx, b.ID, &future)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.DB().ExecContext(ctx, `UPDATE memo_share SET expires_ts=unixepoch() WHERE uid=?`, g3.Share.UID); err != nil {
		t.Fatal(err)
	}
	assertCounts := func(shared int) {
		t.Helper()
		got, err := f.timeline.Counts(ctx)
		if err != nil || got != (domain.MemoCounts{All: 3, Favorites: 1, Shared: shared}) {
			t.Fatalf("counts=%+v err=%v", got, err)
		}
	}
	assertCounts(1)
	items, _, err := f.timeline.List(ctx, repository.TimelineQuery{SharedOnly: true})
	if err != nil || len(items) != 1 || items[0].ID != a.ID {
		t.Fatalf("shared=%+v err=%v", items, err)
	}
	if err := f.shares.Revoke(ctx, g1.Share.UID); err != nil {
		t.Fatal(err)
	}
	assertCounts(1)
	if err := f.shares.Revoke(ctx, g2.Share.UID); err != nil {
		t.Fatal(err)
	}
	assertCounts(0)
	items, _, err = f.timeline.List(ctx, repository.TimelineQuery{SharedOnly: true})
	if err != nil || len(items) != 0 {
		t.Fatalf("revoked shared=%+v err=%v", items, err)
	}
	if err := f.memo.Delete(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	counts, err = f.timeline.Counts(ctx)
	if err != nil || counts != (domain.MemoCounts{All: 2}) {
		t.Fatalf("deleted counts=%+v err=%v", counts, err)
	}
}

func TestFilteredKeysetPaginationAndCalendar(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, t.TempDir())
	loc := time.FixedZone("UTC+7", 7*60*60)
	day := time.Date(2026, 10, 2, 0, 0, 0, 0, loc)
	var expected []int64
	for i := 0; i < 9; i++ {
		m := createMemo(t, f, "tied #FPGA")
		if _, err := f.store.DB().ExecContext(ctx, `UPDATE memo SET created_ts=? WHERE id=?`, day.Add(time.Hour).Unix(), m.ID); err != nil {
			t.Fatal(err)
		}
		if i%2 == 0 {
			if err := f.memo.SetFavorite(ctx, m.ID, true); err != nil {
				t.Fatal(err)
			}
			if _, err := f.shares.Create(ctx, m.ID, nil); err != nil {
				t.Fatal(err)
			}
			if _, err := f.shares.Create(ctx, m.ID, nil); err != nil {
				t.Fatal(err)
			}
			expected = append([]int64{m.ID}, expected...)
		}
	}
	outside := createMemo(t, f, "outside #FPGA")
	if _, err := f.store.DB().ExecContext(ctx, `UPDATE memo SET created_ts=? WHERE id=?`, day.AddDate(0, 0, 1).Unix(), outside.ID); err != nil {
		t.Fatal(err)
	}
	q := repository.TimelineQuery{FavoriteOnly: true, SharedOnly: true, Tags: []string{"fpga"}, Limit: 2}
	var ids []int64
	for {
		items, next, err := f.timeline.List(ctx, q)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range items {
			ids = append(ids, m.ID)
			if !m.Favorite || len(m.Tags) != 1 {
				t.Fatalf("unhydrated memo=%+v", m)
			}
		}
		if next == nil {
			break
		}
		q.Cursor = next
		if len(ids) > len(expected) {
			t.Fatal("pagination did not terminate")
		}
	}
	if len(ids) != len(expected) {
		t.Fatalf("ids=%v expected=%v", ids, expected)
	}
	for i := range ids {
		if ids[i] != expected[i] {
			t.Fatalf("ids=%v expected=%v", ids, expected)
		}
	}
	var cursor *repository.TimelineCursor
	var count int
	for {
		items, next, err := f.calendar.PageForDate(ctx, "2026-10-02", loc, 2, cursor)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range items {
			if m.ID == outside.ID {
				t.Fatal("next local day included")
			}
		}
		count += len(items)
		if next == nil {
			break
		}
		cursor = next
		if count > 9 {
			t.Fatal("calendar pagination did not terminate")
		}
	}
	if count != 9 {
		t.Fatalf("calendar count=%d", count)
	}
	if _, _, err := f.calendar.PageForDate(ctx, "bad-date", nil, 2, nil); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("invalid date=%v", err)
	}
}

func TestAttachmentListingPagesAndCascade(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, t.TempDir())
	a := createMemo(t, f, "a")
	b := createMemo(t, f, "b")
	var expected []int64
	for _, m := range []*domain.Memo{a, b, a, b, a} {
		att, err := f.attachments.Add(ctx, m.ID, "note.txt", "text/plain", bytes.NewBufferString("hello"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.store.DB().ExecContext(ctx, `UPDATE attachment SET created_ts=1 WHERE id=?`, att.ID); err != nil {
			t.Fatal(err)
		}
		expected = append([]int64{att.ID}, expected...)
	}
	var ids []int64
	for offset := 0; offset < 6; offset += 2 {
		items, err := f.attachments.ListAll(ctx, 2, offset)
		if err != nil {
			t.Fatal(err)
		}
		for _, at := range items {
			ids = append(ids, at.ID)
			if at.Size != 5 || at.Filename != "note.txt" {
				t.Fatalf("attachment=%+v", at)
			}
		}
	}
	if len(ids) != len(expected) {
		t.Fatalf("ids=%v", ids)
	}
	for i := range ids {
		if ids[i] != expected[i] {
			t.Fatalf("ids=%v expected=%v", ids, expected)
		}
	}
	items, err := f.attachments.ListAll(ctx, 0, -1)
	if err != nil || len(items) != 5 {
		t.Fatalf("defaults=%+v err=%v", items, err)
	}
	if err := f.memo.Delete(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	items, err = f.attachments.ListAll(ctx, 10, 0)
	if err != nil || len(items) != 2 {
		t.Fatalf("cascade=%+v err=%v", items, err)
	}
}

func TestBackupFavoritePoliciesAndLegacyRecord(t *testing.T) {
	ctx := context.Background()
	source := newFixture(t, t.TempDir())
	m := createMemo(t, source, "favorite #FPGA")
	if _, err := source.attachments.Add(ctx, m.ID, "exported.txt", "text/plain", bytes.NewBufferString("backup bytes")); err != nil {
		t.Fatal(err)
	}
	if _, err := source.shares.Create(ctx, m.ID, nil); err != nil {
		t.Fatal(err)
	}
	if err := source.memo.SetFavorite(ctx, m.ID, true); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(t.TempDir(), "favorites.zip")
	if err := source.backup.ExportFile(ctx, archivePath); err != nil {
		t.Fatal(err)
	}
	target := newFixture(t, t.TempDir())
	if _, err := target.backup.ImportFile(ctx, archivePath, service.ImportSkip); err != nil {
		t.Fatal(err)
	}
	got, err := target.memo.GetByUID(ctx, m.UID)
	if err != nil || !got.Favorite || len(got.Attachments) != 1 {
		t.Fatalf("restored=%+v err=%v", got, err)
	}
	oldPath, err := target.attachments.Path(ctx, got.Attachments[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	shares, err := target.shares.List(ctx, got.ID)
	if err != nil || len(shares) != 0 {
		t.Fatalf("backup restored share grants: %v err=%v", shares, err)
	}
	if err := target.memo.SetFavorite(ctx, got.ID, false); err != nil {
		t.Fatal(err)
	}
	report, err := target.backup.ImportFile(ctx, archivePath, service.ImportSkip)
	if err != nil || report.Skipped != 1 {
		t.Fatalf("skip=%+v err=%v", report, err)
	}
	got, err = target.memo.GetByUID(ctx, m.UID)
	if err != nil || got.Favorite {
		t.Fatalf("skip overwrote=%+v err=%v", got, err)
	}
	if _, err := os.Stat(oldPath); err != nil {
		t.Fatalf("skip removed attachment: %v", err)
	}
	report, err = target.backup.ImportFile(ctx, archivePath, service.ImportReplace)
	if err != nil || report.Replaced != 1 {
		t.Fatalf("replace=%+v err=%v", report, err)
	}
	got, err = target.memo.GetByUID(ctx, m.UID)
	if err != nil || !got.Favorite {
		t.Fatalf("replace=%+v err=%v", got, err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("replace left previous attachment: %v", err)
	}
	report, err = target.backup.ImportFile(ctx, archivePath, service.ImportDuplicate)
	if err != nil || report.Duplicated != 1 {
		t.Fatalf("duplicate=%+v err=%v", report, err)
	}
	all, _, err := target.timeline.List(ctx, repository.TimelineQuery{FavoriteOnly: true})
	if err != nil || len(all) != 2 || all[0].UID == all[1].UID {
		t.Fatalf("duplicates=%+v err=%v", all, err)
	}
	if len(all[0].Attachments) != 1 || len(all[1].Attachments) != 1 || all[0].Attachments[0].RelativePath == all[1].Attachments[0].RelativePath {
		t.Fatalf("duplicate did not isolate attachments: %+v", all)
	}
	// False is omitted by the writer, reproducing an original v1 record.
	if err := source.memo.SetFavorite(ctx, m.ID, false); err != nil {
		t.Fatal(err)
	}
	legacyPath := filepath.Join(t.TempDir(), "legacy.zip")
	if err := source.backup.ExportFile(ctx, legacyPath); err != nil {
		t.Fatal(err)
	}
	report, err = target.backup.ImportFile(ctx, legacyPath, service.ImportReplace)
	if err != nil || report.Replaced != 1 {
		t.Fatalf("legacy=%+v err=%v", report, err)
	}
	got, err = target.memo.GetByUID(ctx, m.UID)
	if err != nil || got.Favorite {
		t.Fatalf("legacy favorite=%+v err=%v", got, err)
	}
}
