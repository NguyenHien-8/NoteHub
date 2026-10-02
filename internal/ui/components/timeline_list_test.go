package components

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
)

func TestTimelineRefreshAndPagingPreserveExistingCardsAndScroll(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	timeline := NewTimelineList()
	now := time.Now()
	memos := []domain.Memo{{ID: 1, Content: "First", CreatedAt: now}, {ID: 2, Content: "Second", CreatedAt: now}}
	timeline.SetMemos(memos, nil, MemoActions{})
	window := test.NewWindow(timeline.Object)
	defer window.Close()
	window.Resize(fyne.NewSize(420, 160))
	first, second := timeline.rows.Objects[1], timeline.rows.Objects[2]
	timeline.Scroll.Offset = fyne.NewPos(0, 30)
	timeline.SetMemos(memos, nil, MemoActions{})
	if timeline.rows.Objects[1] != first || timeline.rows.Objects[2] != second {
		t.Fatal("unchanged refresh rebuilt cards and lost their UI state")
	}
	if timeline.Scroll.Offset.Y != 30 {
		t.Fatalf("unchanged refresh moved scroll to %v", timeline.Scroll.Offset)
	}
	memos = append(memos, domain.Memo{ID: 3, Content: "Next page", CreatedAt: now})
	timeline.SetMemos(memos, nil, MemoActions{})
	if timeline.rows.Objects[1] != first || timeline.rows.Objects[2] != second || len(timeline.rows.Objects) != 4 {
		t.Fatal("paging replaced existing cards instead of appending")
	}
	if timeline.Scroll.Offset.Y != 30 {
		t.Fatalf("paging moved scroll to %v", timeline.Scroll.Offset)
	}
}

func TestTimelineReplacesOnlyChangedMemoAndSnapshotsAttachmentOrder(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	timeline := NewTimelineList()
	now := time.Now()
	memos := []domain.Memo{
		{ID: 1, Content: "First", CreatedAt: now, Attachments: []domain.Attachment{{ID: 1, Filename: "a.txt"}, {ID: 2, Filename: "b.txt"}}},
		{ID: 2, Content: "Second", CreatedAt: now},
	}
	timeline.SetMemos(memos, nil, MemoActions{})
	first, second := timeline.rows.Objects[1], timeline.rows.Objects[2]
	memos[0].Attachments[0], memos[0].Attachments[1] = memos[0].Attachments[1], memos[0].Attachments[0]
	timeline.SetMemos(memos, nil, MemoActions{})
	if timeline.rows.Objects[1] == first {
		t.Fatal("changed attachment order retained an obsolete card")
	}
	if timeline.rows.Objects[2] != second {
		t.Fatal("changing one memo rebuilt its unchanged neighbor")
	}
}
