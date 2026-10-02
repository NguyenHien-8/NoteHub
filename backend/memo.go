package backend

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"example.com/memosdesktop/backend/internal/tagparse"
)

func (s *Service) CreateMemo(ctx context.Context, content string) (*Memo, error) {
	uid, err := newUID()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Truncate(time.Second)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `INSERT INTO memo(uid, created_ts, updated_ts, content, search_text) VALUES(?,?,?,?,?)`,
		uid, unix(now), unix(now), content, normalizeSearch(content))
	if err != nil {
		return nil, fmt.Errorf("insert memo: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	tags := tagparse.Extract(content)
	if err := replaceTagsTx(ctx, tx, id, tags); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &Memo{ID: id, UID: uid, Content: content, CreatedAt: now, UpdatedAt: now, Tags: tags}, nil
}

func (s *Service) UpdateMemo(ctx context.Context, id int64, content string) (*Memo, error) {
	if id <= 0 {
		return nil, errors.New("invalid memo id")
	}
	now := time.Now().UTC().Truncate(time.Second)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `UPDATE memo SET content=?, search_text=?, updated_ts=? WHERE id=?`,
		content, normalizeSearch(content), unix(now), id)
	if err != nil {
		return nil, err
	}
	if n, _ := result.RowsAffected(); n != 1 {
		return nil, sql.ErrNoRows
	}
	tags := tagparse.Extract(content)
	if err := replaceTagsTx(ctx, tx, id, tags); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetMemo(ctx, id)
}

func (s *Service) DeleteMemo(ctx context.Context, id int64) error {
	if _, err := s.GetMemo(ctx, id); err != nil {
		return err
	}
	attachments, err := s.ListAttachments(ctx, id)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM memo WHERE id=?`, id)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	for _, a := range attachments {
		if p, e := ensureInside(s.attachmentRoot, a.RelativePath); e == nil {
			_ = removeFileAndEmptyParents(p, s.attachmentRoot)
		}
	}
	return nil
}

func (s *Service) GetMemo(ctx context.Context, id int64) (*Memo, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id,uid,created_ts,updated_ts,content FROM memo WHERE id=?`, id)
	memo, err := scanMemo(row)
	if err != nil {
		return nil, err
	}
	if err := s.loadMemoExtras(ctx, memo); err != nil {
		return nil, err
	}
	return memo, nil
}

func (s *Service) GetMemoByUID(ctx context.Context, uid string) (*Memo, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id,uid,created_ts,updated_ts,content FROM memo WHERE uid=?`, uid)
	memo, err := scanMemo(row)
	if err != nil {
		return nil, err
	}
	if err := s.loadMemoExtras(ctx, memo); err != nil {
		return nil, err
	}
	return memo, nil
}

func (s *Service) ListTimeline(ctx context.Context, q TimelineQuery) ([]Memo, error) {
	if q.Limit <= 0 || q.Limit > 500 {
		q.Limit = 50
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	where := []string{"1=1"}
	args := []any{}
	if q.From != nil {
		where = append(where, "m.created_ts >= ?")
		args = append(args, unix(*q.From))
	}
	if q.To != nil {
		where = append(where, "m.created_ts < ?")
		args = append(args, unix(*q.To))
	}
	for _, tag := range q.Tags {
		where = append(where, `EXISTS (SELECT 1 FROM memo_tag mt WHERE mt.memo_id=m.id AND mt.tag_norm=?)`)
		args = append(args, strings.ToLower(strings.TrimPrefix(tag, "#")))
	}
	args = append(args, q.Limit, q.Offset)
	rows, err := s.db.QueryContext(ctx, `SELECT m.id,m.uid,m.created_ts,m.updated_ts,m.content FROM memo m WHERE `+
		strings.Join(where, " AND ")+` ORDER BY m.created_ts DESC,m.id DESC LIMIT ? OFFSET ?`, args...)
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
	// MaxOpenConns(1) is intentional for a desktop SQLite connection. Load
	// composed fields only after closing the list cursor to avoid self-deadlock.
	for i := range list {
		if err := s.loadMemoExtras(ctx, &list[i]); err != nil {
			return nil, err
		}
	}
	return list, nil
}

type rowScanner interface{ Scan(...any) error }

func scanMemo(row rowScanner) (*Memo, error) {
	var m Memo
	var created, updated int64
	if err := row.Scan(&m.ID, &m.UID, &created, &updated, &m.Content); err != nil {
		return nil, err
	}
	m.CreatedAt, m.UpdatedAt = fromUnix(created), fromUnix(updated)
	return &m, nil
}

func (s *Service) loadMemoExtras(ctx context.Context, memo *Memo) error {
	tags, err := s.TagsForMemo(ctx, memo.ID)
	if err != nil {
		return err
	}
	memo.Tags = tags
	attachments, err := s.ListAttachments(ctx, memo.ID)
	if err != nil {
		return err
	}
	memo.Attachments = attachments
	return nil
}

func replaceTagsTx(ctx context.Context, tx *sql.Tx, memoID int64, tags []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM memo_tag WHERE memo_id=?`, memoID); err != nil {
		return err
	}
	for _, tag := range tags {
		if _, err := tx.ExecContext(ctx, `INSERT INTO memo_tag(memo_id,tag,tag_norm) VALUES(?,?,?)`, memoID, tag, strings.ToLower(tag)); err != nil {
			return err
		}
	}
	return nil
}

func removeFileAndEmptyParents(file, stop string) error {
	_ = remove(file)
	for dir := filepathDir(file); dir != stop && dir != "." && dir != string(filepathSeparator()); dir = filepathDir(dir) {
		if err := remove(dir); err != nil {
			break
		}
	}
	return nil
}

// Small wrappers keep memo.go focused and make these operations easy to mock later.
var remove = func(path string) error { return osRemove(path) }
var filepathDir = func(path string) string { return pathDir(path) }
var filepathSeparator = func() rune { return osPathSeparator() }
