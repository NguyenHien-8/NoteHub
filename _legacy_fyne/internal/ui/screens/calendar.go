package screens

import (
	"context"
	"fmt"
	"image"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
	"github.com/NguyenHien-8/NoteHub/internal/repository"
	"github.com/NguyenHien-8/NoteHub/internal/ui/components"
	"github.com/NguyenHien-8/NoteHub/internal/ui/work"
)

type CalendarScreen struct {
	Object     fyne.CanvasObject
	env        *Environment
	calendar   *components.Calendar
	month      time.Time
	selected   string
	generation uint64
	list       *components.TimelineList
	caption    *widget.Label
	items      []domain.Memo
	thumbs     map[int64]image.Image
	cursor     *repository.TimelineCursor
	loading    bool
}

func NewCalendar(env *Environment) *CalendarScreen {
	now := time.Now()
	s := &CalendarScreen{env: env, month: time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local), selected: now.Format("2006-01-02"), list: components.NewTimelineList(), caption: widget.NewLabel("")}
	s.calendar = components.NewCalendar(func(date string) { s.selected = date; s.Refresh() }, func(delta int) { s.month = s.month.AddDate(0, delta, 0); s.Refresh() })
	s.Object = container.NewBorder(container.NewVBox(s.calendar.Object, s.caption), nil, nil, nil, s.list.Object)
	return s
}

func (s *CalendarScreen) Refresh() {
	s.generation++
	generation, month := s.generation, s.month
	s.items = nil
	s.thumbs = make(map[int64]image.Image)
	s.cursor = nil
	s.loading = false
	work.Run(s.env.Jobs, func(ctx context.Context) ([]domain.CalendarDay, error) {
		return s.env.Backend.Calendar.Month(ctx, month.Year(), month.Month(), time.Local)
	}, func(days []domain.CalendarDay, err error) {
		if generation != s.generation {
			return
		}
		if err != nil {
			s.env.Error(err)
			return
		}
		s.calendar.SetMonth(month.Year(), month.Month(), days, s.selected)
	})
	s.load()
}

func (s *CalendarScreen) load() {
	if s.loading {
		return
	}
	s.loading = true
	s.caption.SetText("Loading notes…")
	generation, date, cursor := s.generation, s.selected, s.cursor
	work.Run(s.env.Jobs, func(ctx context.Context) (memoPage, error) {
		items, next, err := s.env.Backend.Calendar.PageForDate(ctx, date, time.Local, 40, cursor)
		if err != nil {
			return memoPage{}, err
		}
		return memoPage{items: items, next: next, thumbs: s.env.Previews.Memos(ctx, items)}, nil
	}, func(page memoPage, err error) {
		if generation != s.generation {
			return
		}
		s.loading = false
		if err != nil {
			s.caption.SetText("Could not load notes")
			s.env.Error(err)
			return
		}
		s.items = append(s.items, page.items...)
		s.cursor = page.next
		for id, img := range page.thumbs {
			s.thumbs[id] = img
		}
		s.caption.SetText(fmt.Sprintf("%s · %d notes", date, len(s.items)))
		s.list.SetMemos(s.items, s.thumbs, s.env.Actions)
		s.list.SetMore(page.next != nil, s.load)
	})
}

// Calendar screen.
