package screens

import (
	"context"
	"fmt"
	"image"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
	"github.com/NguyenHien-8/NoteHub/internal/repository"
	"github.com/NguyenHien-8/NoteHub/internal/ui/components"
	"github.com/NguyenHien-8/NoteHub/internal/ui/work"
)

type Timeline struct {
	Object     fyne.CanvasObject
	Composer   *components.Composer
	List       *components.TimelineList
	env        *Environment
	caption    *widget.Label
	progress   *widget.ProgressBarInfinite
	filter     repository.TimelineQuery
	date       string
	items      []domain.Memo
	thumbs     map[int64]image.Image
	cursor     *repository.TimelineCursor
	generation uint64
	loading    bool
	saving     bool
}

func NewTimeline(env *Environment) *Timeline {
	s := &Timeline{env: env, caption: widget.NewLabel("All Notes"), progress: widget.NewProgressBarInfinite(), List: components.NewTimelineList()}
	s.progress.Hide()
	s.Composer = components.NewComposer(s.save, func() { env.PickFiles(s.Composer.AddPaths) }, func() { env.PickTag(s.Composer.InsertTag) })
	clear := widget.NewButton("Clear filters", func() { s.Filter(repository.TimelineQuery{}, "") })
	header := container.NewVBox(s.Composer.Object, container.NewBorder(nil, nil, s.caption, clear), s.progress)
	s.Object = container.NewBorder(header, nil, nil, nil, s.List.Object)
	return s
}

func (s *Timeline) Dirty() bool {
	return strings.TrimSpace(s.Composer.Entry.Text) != "" || len(s.Composer.Paths()) > 0 || s.saving
}
func (s *Timeline) Filter(q repository.TimelineQuery, date string) {
	s.filter = q
	s.date = date
	s.Refresh()
}

func (s *Timeline) Refresh() {
	s.generation++
	s.loading = false
	s.items = nil
	s.thumbs = make(map[int64]image.Image)
	s.cursor = nil
	s.List.SetMemos(nil, nil, s.env.Actions)
	s.List.SetMore(false, nil)
	caption := "All Notes"
	if s.filter.FavoriteOnly {
		caption = "Favorites"
	}
	if s.filter.SharedOnly {
		caption = "Shared"
	}
	if len(s.filter.Tags) > 0 {
		caption = "#" + strings.Join(s.filter.Tags, "  #")
	}
	if s.date != "" {
		caption = s.date
	}
	s.caption.SetText(caption)
	s.load()
}

type memoPage struct {
	items  []domain.Memo
	next   *repository.TimelineCursor
	thumbs map[int64]image.Image
}

func (s *Timeline) load() {
	if s.loading {
		return
	}
	s.loading = true
	s.progress.Show()
	s.progress.Start()
	generation, q, date, cursor := s.generation, s.filter, s.date, s.cursor
	q.Limit = 40
	q.Cursor = cursor
	work.Run(s.env.Jobs, func(ctx context.Context) (memoPage, error) {
		var items []domain.Memo
		var next *repository.TimelineCursor
		var err error
		if date != "" {
			items, next, err = s.env.Backend.Calendar.PageForDate(ctx, date, time.Local, 40, cursor)
		} else {
			items, next, err = s.env.Backend.Timeline.List(ctx, q)
		}
		if err != nil {
			return memoPage{}, err
		}
		return memoPage{items: items, next: next, thumbs: s.env.Previews.Memos(ctx, items)}, nil
	}, func(page memoPage, err error) {
		if generation != s.generation {
			return
		}
		s.loading = false
		s.progress.Stop()
		s.progress.Hide()
		if err != nil {
			s.env.Error(err)
			s.List.SetMore(true, s.load)
			return
		}
		s.items = append(s.items, page.items...)
		s.cursor = page.next
		for id, img := range page.thumbs {
			s.thumbs[id] = img
		}
		s.List.SetMemos(s.items, s.thumbs, s.env.Actions)
		s.List.SetMore(page.next != nil, s.load)
	})
}

type saveResult struct {
	memo   *domain.Memo
	failed []string
}

func (s *Timeline) save(content string, paths []string) {
	if s.saving {
		return
	}
	if strings.TrimSpace(content) == "" && len(paths) == 0 {
		return
	}
	s.saving = true
	s.Composer.SetBusy(true)
	work.Run(s.env.Jobs, func(ctx context.Context) (saveResult, error) {
		m, err := s.env.Backend.Memos.Create(ctx, content)
		if err != nil {
			return saveResult{}, err
		}
		result := saveResult{memo: m}
		for _, path := range paths {
			if _, err := s.env.Backend.Attachments.AddFromFile(ctx, m.ID, path); err != nil {
				result.failed = append(result.failed, fmt.Sprintf("%s: %v", filepath.Base(path), err))
			}
		}
		return result, nil
	}, func(result saveResult, err error) {
		s.saving = false
		s.Composer.SetBusy(false)
		if err != nil {
			s.env.Error(err)
			return
		}
		s.Composer.Reset()
		s.filter = repository.TimelineQuery{}
		s.date = ""
		s.env.Changed()
		if len(result.failed) > 0 {
			dialog.ShowInformation("Note saved; some files were not attached", strings.Join(result.failed, "\n")+"\n\nOpen the note to attach these files again.", s.env.Window)
		}
	})
}

// Timeline screen: editor + chronological memo list.
