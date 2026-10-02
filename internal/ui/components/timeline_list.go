package components

import (
	"image"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
)

// TimelineList renders only the pages its owner has requested from the backend.
type TimelineList struct {
	Object               fyne.CanvasObject
	Scroll               *container.Scroll
	rows                 *fyne.Container
	more                 *widget.Button
	load                 func()
	available, requested bool
}

func NewTimelineList() *TimelineList {
	t := &TimelineList{rows: container.New(layout.NewCustomPaddedVBoxLayout(12))}
	t.Scroll = container.NewVScroll(t.rows)
	t.more = widget.NewButtonWithIcon("Load more", theme.NavigateNextIcon(), t.requestMore)
	t.more.Hide()
	t.Object = container.NewBorder(nil, container.NewCenter(t.more), nil, nil, t.Scroll)
	t.Scroll.OnScrolled = func(offset fyne.Position) {
		viewHeight := t.Scroll.Size().Height
		contentHeight := t.rows.MinSize().Height
		if viewHeight > 0 && contentHeight > viewHeight && offset.Y+viewHeight >= contentHeight-180 {
			t.requestMore()
		}
	}
	t.SetMemos(nil, nil, MemoActions{})
	return t
}

func (t *TimelineList) SetMemos(memos []domain.Memo, thumbs map[int64]image.Image, actions MemoActions) {
	t.rows.Objects = nil
	if len(memos) == 0 {
		heading := widget.NewLabelWithStyle("A little space for your thoughts", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		body := widget.NewLabel("Write your first note above, or try a different filter.")
		body.Wrapping = fyne.TextWrapWord
		body.Alignment = fyne.TextAlignCenter
		body.Importance = widget.LowImportance
		t.rows.Add(Surface(container.NewVBox(heading, body)))
	} else {
		now := time.Now()
		previous := ""
		for _, memo := range memos {
			group := timelineGroup(memo.CreatedAt, now)
			if group != previous {
				title := widget.NewLabelWithStyle(group, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
				date := ""
				if group != "Earlier" {
					date = memo.CreatedAt.In(time.Local).Format("Monday, 2 January 2006")
				}
				subtitle := widget.NewLabel(date)
				subtitle.Importance = widget.LowImportance
				if date == "" {
					subtitle.Hide()
				}
				t.rows.Add(container.NewBorder(nil, nil, nil, subtitle, title))
				previous = group
			}
			t.rows.Add(NewMemoCard(memo, thumbs[memo.ID], actions))
		}
	}
	t.rows.Refresh()
	t.Scroll.Refresh()
}

func (t *TimelineList) SetMore(available bool, load func()) {
	t.available, t.requested, t.load = available, false, load
	t.more.SetText("Load more")
	t.more.Enable()
	if available && load != nil {
		t.more.Show()
	} else {
		t.more.Hide()
	}
}

func (t *TimelineList) requestMore() {
	if !t.available || t.requested || t.load == nil {
		return
	}
	t.requested = true
	t.more.SetText("Loading…")
	t.more.Disable()
	t.load()
}
