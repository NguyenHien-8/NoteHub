package sqlite

import (
	"context"
	"strings"

	"github.com/NguyenHien-8/NoteHub/internal/domain"
)

func (s *Store) TagsByMemoIDs(ctx context.Context, memoIDs []int64) (map[int64][]string, error) {
	out := make(map[int64][]string)
	if len(memoIDs) == 0 {
		return out, nil
	}
	ph := make([]string, len(memoIDs))
	args := make([]any, len(memoIDs))
	for i, id := range memoIDs {
		ph[i] = "?"
		args[i] = id
	}
	rows, err := s.db.QueryContext(ctx, `SELECT memo_id,tag FROM memo_tag WHERE memo_id IN (`+strings.Join(ph, ",")+`) ORDER BY memo_id ASC,tag_norm ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var tag string
		if err := rows.Scan(&id, &tag); err != nil {
			return nil, err
		}
		out[id] = append(out[id], tag)
	}
	return out, rows.Err()
}

func (s *Store) ListTagCounts(ctx context.Context) ([]domain.TagCount, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT MIN(tag),COUNT(*) FROM memo_tag GROUP BY tag_norm ORDER BY tag_norm ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.TagCount
	for rows.Next() {
		var item domain.TagCount
		if err := rows.Scan(&item.Tag, &item.Count); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
