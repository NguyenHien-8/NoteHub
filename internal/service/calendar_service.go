package service

import (
	"context"
	"fmt"
	"time"

	"github.com/NguyenHien-8/NoteHub/internal/domain"
	"github.com/NguyenHien-8/NoteHub/internal/repository"
)

type CalendarService struct {
	memos    repository.MemoRepository
	timeline *TimelineService
}

func NewCalendarService(memos repository.MemoRepository, timeline *TimelineService) *CalendarService {
	return &CalendarService{memos: memos, timeline: timeline}
}

func (s *CalendarService) Month(ctx context.Context, year int, month time.Month, loc *time.Location) ([]domain.CalendarDay, error) {
	if loc == nil {
		loc = time.Local
	}
	start := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	end := start.AddDate(0, 1, 0)
	times, err := s.memos.CreatedBetween(ctx, start, end)
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int)
	for _, t := range times {
		counts[t.In(loc).Format("2006-01-02")]++
	}
	out := make([]domain.CalendarDay, 0, len(counts))
	for day := start; day.Before(end); day = day.AddDate(0, 0, 1) {
		key := day.Format("2006-01-02")
		if counts[key] != 0 {
			out = append(out, domain.CalendarDay{Date: key, Count: counts[key]})
		}
	}
	return out, nil
}

func (s *CalendarService) MemosForDate(ctx context.Context, date string, loc *time.Location, limit int) ([]domain.Memo, error) {
	if loc == nil {
		loc = time.Local
	}
	start, err := time.ParseInLocation("2006-01-02", date, loc)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid date %q", domain.ErrInvalid, date)
	}
	end := start.AddDate(0, 0, 1)
	items, _, err := s.timeline.List(ctx, repository.TimelineQuery{From: &start, To: &end, Limit: limit})
	return items, err
}
