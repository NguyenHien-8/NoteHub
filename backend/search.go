package backend

import (
	"context"
	"strings"
)

func (s *Service) Search(ctx context.Context, q SearchQuery) ([]Memo, error) {
	if q.Limit <= 0 || q.Limit > 500 {
		q.Limit = 50
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	where := []string{"1=1"}
	args := []any{}
	if text := normalizeSearch(q.Text); text != "" {
		where = append(where, `m.search_text LIKE ? ESCAPE '\'`)
		args = append(args, "%"+escapeLikeLiteral(text)+"%")
	}
	if q.From != nil {
		where = append(where, "m.created_ts >= ?")
		args = append(args, unix(*q.From))
	}
	if q.To != nil {
		where = append(where, "m.created_ts < ?")
		args = append(args, unix(*q.To))
	}
	for _, tag := range q.Tags {
		norm := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(tag, "#")))
		if norm == "" {
			continue
		}
		where = append(where, `EXISTS (SELECT 1 FROM memo_tag mt WHERE mt.memo_id=m.id AND mt.tag_norm=?)`)
		args = append(args, norm)
	}
	args = append(args, q.Limit, q.Offset)
	query := `SELECT m.id,m.uid,m.created_ts,m.updated_ts,m.content FROM memo m WHERE ` + strings.Join(where, " AND ") +
		` ORDER BY m.created_ts DESC,m.id DESC LIMIT ? OFFSET ?`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	var list []Memo
	for rows.Next() {
		m, err := scanMemo(rows)
		if err != nil {
			_ = rows.Close()
			return nil, err
		}
		list = append(list, *m)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i := range list {
		if err := s.loadMemoExtras(ctx, &list[i]); err != nil {
			return nil, err
		}
	}
	return list, nil
}
