package components

import (
	"image"
	"reflect"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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
	cards                map[int64]timelineCard
	headers              map[string]fyne.CanvasObject
	empty                fyne.CanvasObject
}

type timelineCard struct {
	memo   domain.Memo
	object fyne.CanvasObject
}

func NewTimelineList() *TimelineList {
	t := &TimelineList{rows: NewResponsiveVBox(10), cards: make(map[int64]timelineCard), headers: make(map[string]fyne.CanvasObject)}
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
	objects := make([]fyne.CanvasObject, 0, len(memos)+3)
	cards := make(map[int64]timelineCard, len(memos))
	if len(memos) == 0 {
		if t.empty == nil {
			heading := widget.NewLabelWithStyle("A little space for your thoughts", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
			body := widget.NewLabel("Write your first note above, or try a different filter.")
			body.Wrapping = fyne.TextWrapWord
			body.Alignment = fyne.TextAlignCenter
			body.Importance = widget.LowImportance
			t.empty = Surface(container.NewVBox(heading, body))
		}
		objects = append(objects, t.empty)
	} else {
		now := time.Now()
		previous := ""
		for _, memo := range memos {
			group := timelineGroup(memo.CreatedAt, now)
			if group != previous {
				date := ""
				if group != "Earlier" {
					date = memo.CreatedAt.In(time.Local).Format("Monday, 2 January 2006")
				}
				key := group + date
				header := t.headers[key]
				if header == nil {
					title := widget.NewLabelWithStyle(group, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
					subtitle := widget.NewLabel(date)
					subtitle.Importance = widget.LowImportance
					if date == "" {
						subtitle.Hide()
					}
					header = container.NewBorder(nil, nil, nil, subtitle, title)
					t.headers[key] = header
				}
				objects = append(objects, header)
				previous = group
			}
			card, exists := t.cards[memo.ID]
			if !exists || !reflect.DeepEqual(card.memo, memo) {
				snapshot := memo
				snapshot.Tags = append([]string(nil), memo.Tags...)
				snapshot.Attachments = append([]domain.Attachment(nil), memo.Attachments...)
				card = timelineCard{memo: snapshot, object: NewMemoCard(memo, thumbs, actions)}
			}
			cards[memo.ID] = card
			objects = append(objects, card.object)
		}
	}
	t.cards = cards
	unchanged := len(objects) == len(t.rows.Objects)
	if unchanged {
		for index, object := range objects {
			if object != t.rows.Objects[index] {
				unchanged = false
				break
			}
		}
	}
	if unchanged {
		return
	}
	t.rows.Objects = objects
	t.rows.Refresh()
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
