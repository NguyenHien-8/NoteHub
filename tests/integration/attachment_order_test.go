//go:build !teststub

package integration

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/NguyenHien-8/NoteHub/internal/domain"
)

func addOrderedAttachment(t *testing.T, f *fixture, memoID int64, filename string) *domain.Attachment {
	t.Helper()
	a, err := f.attachments.Add(context.Background(), memoID, filename, "image/png", strings.NewReader(filename))
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func reorderAttachments(t *testing.T, f *fixture, memoID int64, ids []int64) error {
	t.Helper()
	r, ok := any(f.attachments).(interface {
		Reorder(context.Context, int64, []int64) error
	})
	if !ok {
		t.Fatal("attachment service does not support persistent reordering")
	}
	return r.Reorder(context.Background(), memoID, ids)
}

func assertAttachmentIDs(t *testing.T, f *fixture, memoID int64, want []int64) {
	t.Helper()
	got, err := f.attachments.List(context.Background(), memoID)
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]int64, len(got))
	for i, a := range got {
		ids[i] = a.ID
	}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("attachment order=%v, want %v", ids, want)
	}
}

func TestAttachmentOrderSurvivesReopenAndHydration(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	f := newFixture(t, root)
	m := createMemo(t, f, "gallery")
	a := addOrderedAttachment(t, f, m.ID, "first.png")
	b := addOrderedAttachment(t, f, m.ID, "second.png")
	c := addOrderedAttachment(t, f, m.ID, "third.png")
	want := []int64{c.ID, a.ID, b.ID}
	if err := reorderAttachments(t, f, m.ID, want); err != nil {
		t.Fatal(err)
	}
	assertAttachmentIDs(t, f, m.ID, want)
	if err := f.store.Close(); err != nil {
		t.Fatal(err)
	}
	f = newFixture(t, root)
	assertAttachmentIDs(t, f, m.ID, want)
	got, err := f.memo.Get(ctx, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Attachments) != 3 || got.Attachments[0].ID != c.ID || got.Attachments[1].ID != a.ID || got.Attachments[2].ID != b.ID {
		t.Fatalf("hydrated order=%+v", got.Attachments)
	}
}

func TestAttachmentOrderRejectsInvalidSetsWithoutChanges(t *testing.T) {
	f := newFixture(t, t.TempDir())
	m := createMemo(t, f, "gallery")
	a := addOrderedAttachment(t, f, m.ID, "first.png")
	b := addOrderedAttachment(t, f, m.ID, "second.png")
	c := addOrderedAttachment(t, f, m.ID, "third.png")
	other := createMemo(t, f, "other")
	foreign := addOrderedAttachment(t, f, other.ID, "foreign.png")
	original := []int64{a.ID, b.ID, c.ID}
	for _, tc := range []struct {
		name   string
		memoID int64
		ids    []int64
		want   error
	}{
		{"duplicate", m.ID, []int64{b.ID, a.ID, a.ID}, domain.ErrInvalid},
		{"foreign", m.ID, []int64{c.ID, a.ID, foreign.ID}, domain.ErrConflict},
		{"missing", m.ID, []int64{c.ID, a.ID}, domain.ErrConflict},
		{"unknown", m.ID, []int64{c.ID, a.ID, 999999}, domain.ErrConflict},
		{"empty stale set", m.ID, nil, domain.ErrConflict},
		{"zero", m.ID, []int64{c.ID, a.ID, 0}, domain.ErrInvalid},
		{"missing memo", 999999, nil, domain.ErrNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := reorderAttachments(t, f, tc.memoID, tc.ids); !errors.Is(err, tc.want) {
				t.Fatalf("error=%v, want %v", err, tc.want)
			}
			assertAttachmentIDs(t, f, m.ID, original)
			assertAttachmentIDs(t, f, other.ID, []int64{foreign.ID})
		})
	}
	if err := f.attachments.Delete(context.Background(), b.ID); err != nil {
		t.Fatal(err)
	}
	if err := reorderAttachments(t, f, m.ID, original); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("deleted attachment snapshot=%v", err)
	}
	assertAttachmentIDs(t, f, m.ID, []int64{a.ID, c.ID})
	if err := reorderAttachments(t, f, createMemo(t, f, "empty").ID, nil); err != nil {
		t.Fatalf("empty memo reorder=%v", err)
	}
}

func TestAttachmentOrderRollsBackWhenAnUpdateFails(t *testing.T) {
	f := newFixture(t, t.TempDir())
	m := createMemo(t, f, "gallery")
	a := addOrderedAttachment(t, f, m.ID, "first.png")
	b := addOrderedAttachment(t, f, m.ID, "second.png")
	c := addOrderedAttachment(t, f, m.ID, "third.png")
	if _, err := f.store.DB().Exec(`CREATE TRIGGER reject_second_order BEFORE UPDATE ON attachment
		WHEN old.filename='first.png' BEGIN SELECT RAISE(ABORT,'simulated write failure'); END;`); err != nil {
		t.Fatal(err)
	}
	if err := reorderAttachments(t, f, m.ID, []int64{c.ID, a.ID, b.ID}); err == nil {
		t.Fatal("reorder succeeded despite write failure")
	}
	assertAttachmentIDs(t, f, m.ID, []int64{a.ID, b.ID, c.ID})
}

func TestAttachmentOrderAppendsAfterReorderAndDeletion(t *testing.T) {
	f := newFixture(t, t.TempDir())
	m := createMemo(t, f, "gallery")
	a := addOrderedAttachment(t, f, m.ID, "first.png")
	b := addOrderedAttachment(t, f, m.ID, "second.png")
	c := addOrderedAttachment(t, f, m.ID, "third.png")
	if err := reorderAttachments(t, f, m.ID, []int64{c.ID, a.ID, b.ID}); err != nil {
		t.Fatal(err)
	}
	if err := f.attachments.Delete(context.Background(), a.ID); err != nil {
		t.Fatal(err)
	}
	d := addOrderedAttachment(t, f, m.ID, "fourth.png")
	assertAttachmentIDs(t, f, m.ID, []int64{c.ID, b.ID, d.ID})
	if err := reorderAttachments(t, f, m.ID, []int64{b.ID, c.ID}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("snapshot before append=%v", err)
	}
	assertAttachmentIDs(t, f, m.ID, []int64{c.ID, b.ID, d.ID})
}
