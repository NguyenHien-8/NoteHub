package backend

import (
	"context"
	"fmt"
	"time"
)

func (s *Service) CalendarMonth(ctx context.Context, year int, month time.Month, loc *time.Location) ([]CalendarDay, error) {
	if loc == nil {
		loc = time.Local
	}
	start := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	end := start.AddDate(0, 1, 0)
	rows, err := s.db.QueryContext(ctx, `SELECT created_ts FROM memo WHERE created_ts>=? AND created_ts<? ORDER BY created_ts ASC`, unix(start), unix(end))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var ts int64
		if err := rows.Scan(&ts); err != nil {
			return nil, err
		}
		day := time.Unix(ts, 0).In(loc).Format("2006-01-02")
		counts[day]++
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var out []CalendarDay
	for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		if counts[key] > 0 {
			out = append(out, CalendarDay{Date: key, Count: counts[key]})
		}
	}
	return out, nil
}

func (s *Service) MemosForDate(ctx context.Context, date string, loc *time.Location) ([]Memo, error) {
	if loc == nil {
		loc = time.Local
	}
	day, err := time.ParseInLocation("2006-01-02", date, loc)
	if err != nil {
		return nil, fmt.Errorf("invalid date %q: %w", date, err)
	}
	end := day.AddDate(0, 0, 1)
	return s.ListTimeline(ctx, TimelineQuery{From: &day, To: &end, Limit: 500})
}
