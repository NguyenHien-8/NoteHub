package backend

import (
	"context"
	"strings"
)

func (s *Service) TagsForMemo(ctx context.Context, memoID int64) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT tag FROM memo_tag WHERE memo_id=? ORDER BY tag_norm ASC`, memoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

type TagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

func (s *Service) ListTags(ctx context.Context) ([]TagCount, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT MIN(tag), COUNT(*) FROM memo_tag GROUP BY tag_norm ORDER BY tag_norm ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TagCount
	for rows.Next() {
		var item TagCount
		if err := rows.Scan(&item.Tag, &item.Count); err != nil {
			return nil, err
		}
		item.Tag = strings.TrimSpace(item.Tag)
		out = append(out, item)
	}
	return out, rows.Err()
}
