package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/NguyenHien-8/NoteHub/internal/domain"
	"github.com/NguyenHien-8/NoteHub/internal/notecontent"
	"github.com/NguyenHien-8/NoteHub/internal/repository"
)

func unix(t time.Time) int64     { return t.UTC().Unix() }
func fromUnix(v int64) time.Time { return time.Unix(v, 0).UTC() }

// The original FTS triggers remain compatible with old databases. Rich-text
// writes replace their index entry with visible text in the same transaction.
func indexRichText(ctx context.Context, tx *sql.Tx, id int64, content string) error {
	if !strings.HasPrefix(content, notecontent.RichPrefix) {
		return nil
	}
	_, err := tx.ExecContext(ctx, `UPDATE memo_fts SET content=? WHERE rowid=?`, notecontent.Plain(content), id)
	return err
}

func scanMemo(scanner interface{ Scan(...any) error }) (*domain.Memo, error) {
	var m domain.Memo
	var created, updated int64
	if err := scanner.Scan(&m.ID, &m.UID, &m.Content, &created, &updated, &m.Revision, &m.Favorite); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	m.CreatedAt = fromUnix(created)
	m.UpdatedAt = fromUnix(updated)
	return &m, nil
}

func (s *Store) CreateMemo(ctx context.Context, memo *domain.Memo, tags []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `INSERT INTO memo(uid,content,created_ts,updated_ts,revision,favorite) VALUES(?,?,?,?,?,?)`,
		memo.UID, memo.Content, unix(memo.CreatedAt), unix(memo.UpdatedAt), max64(memo.Revision, 1), memo.Favorite)
	if err != nil {
		return err
	}
	memo.ID, err = res.LastInsertId()
	if err != nil {
		return err
	}
	memo.Revision = max64(memo.Revision, 1)
	if err := indexRichText(ctx, tx, memo.ID, memo.Content); err != nil {
		return err
	}
	if err := replaceTagsTx(ctx, tx, memo.ID, tags); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) CreateImportedMemo(ctx context.Context, memo *domain.Memo, tags []string, attachments []domain.Attachment) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `INSERT INTO memo(uid,content,created_ts,updated_ts,revision,favorite) VALUES(?,?,?,?,?,?)`,
		memo.UID, memo.Content, unix(memo.CreatedAt), unix(memo.UpdatedAt), max64(memo.Revision, 1), memo.Favorite)
	if err != nil {
		return err
	}
	memo.ID, err = res.LastInsertId()
	if err != nil {
		return err
	}
	memo.Revision = max64(memo.Revision, 1)
	if err := indexRichText(ctx, tx, memo.ID, memo.Content); err != nil {
		return err
	}
	if err := replaceTagsTx(ctx, tx, memo.ID, tags); err != nil {
		return err
	}
	for i := range attachments {
		attachments[i].MemoID = memo.ID
		if _, err := insertAttachmentTx(ctx, tx, &attachments[i]); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) GetMemoByID(ctx context.Context, id int64) (*domain.Memo, error) {
	return scanMemo(s.db.QueryRowContext(ctx, `SELECT id,uid,content,created_ts,updated_ts,revision,favorite FROM memo WHERE id=?`, id))
}

func (s *Store) GetMemoByUID(ctx context.Context, uid string) (*domain.Memo, error) {
	return scanMemo(s.db.QueryRowContext(ctx, `SELECT id,uid,content,created_ts,updated_ts,revision,favorite FROM memo WHERE uid=?`, uid))
}

func (s *Store) UpdateMemo(ctx context.Context, id, expectedRevision int64, content string, updatedAt time.Time, tags []string) (*domain.Memo, error) {
	if expectedRevision < 1 {
		return nil, fmt.Errorf("%w: expected revision is required", domain.ErrInvalid)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `UPDATE memo SET content=?,updated_ts=?,revision=revision+1 WHERE id=? AND revision=?`, content, unix(updatedAt), id, expectedRevision)
	if err != nil {
		return nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n != 1 {
		var exists int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM memo WHERE id=?`, id).Scan(&exists); err != nil {
			return nil, err
		}
		if exists == 0 {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrConflict
	}
	if err := indexRichText(ctx, tx, id, content); err != nil {
		return nil, err
	}
	if err := replaceTagsTx(ctx, tx, id, tags); err != nil {
		return nil, err
	}
	m, err := scanMemo(tx.QueryRowContext(ctx, `SELECT id,uid,content,created_ts,updated_ts,revision,favorite FROM memo WHERE id=?`, id))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *Store) ReplaceImportedMemo(ctx context.Context, id int64, memo *domain.Memo, tags []string, attachments []domain.Attachment) ([]domain.Attachment, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var existingUID string
	if err := tx.QueryRowContext(ctx, `SELECT uid FROM memo WHERE id=?`, id).Scan(&existingUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if memo.UID != existingUID {
		return nil, fmt.Errorf("%w: imported uid does not match replacement target", domain.ErrInvalid)
	}
	old, err := listAttachmentsTx(ctx, tx, []int64{id})
	if err != nil {
		return nil, err
	}
	oldList := old[id]

	if _, err := tx.ExecContext(ctx, `UPDATE memo SET content=?,created_ts=?,updated_ts=?,favorite=?,revision=revision+1 WHERE id=?`,
		memo.Content, unix(memo.CreatedAt), unix(memo.UpdatedAt), memo.Favorite, id); err != nil {
		return nil, err
	}
	if err := indexRichText(ctx, tx, id, memo.Content); err != nil {
		return nil, err
	}
	if err := replaceTagsTx(ctx, tx, id, tags); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM attachment WHERE memo_id=?`, id); err != nil {
		return nil, err
	}
	for i := range attachments {
		attachments[i].MemoID = id
		if _, err := insertAttachmentTx(ctx, tx, &attachments[i]); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return oldList, nil
}

func (s *Store) DeleteMemo(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM memo WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) ListTimeline(ctx context.Context, q repository.TimelineQuery) ([]domain.Memo, *repository.TimelineCursor, error) {
	limit := clampLimit(q.Limit, 50, 200)
	where := []string{"1=1"}
	args := []any{}
	appendTimeRange(&where, &args, q.From, q.To)
	appendTagFilters(&where, &args, q.Tags)
	if q.FavoriteOnly {
		where = append(where, `m.favorite=1`)
	}
	if q.SharedOnly {
		where = append(where, activeSharePredicate)
		args = append(args, unix(time.Now()))
	}
	if q.Cursor != nil {
		where = append(where, `(m.created_ts < ? OR (m.created_ts = ? AND m.id < ?))`)
		ts := unix(q.Cursor.CreatedAt)
		args = append(args, ts, ts, q.Cursor.ID)
	}
	args = append(args, limit+1)
	rows, err := s.db.QueryContext(ctx, `SELECT m.id,m.uid,m.content,m.created_ts,m.updated_ts,m.revision,m.favorite
		FROM memo m WHERE `+strings.Join(where, " AND ")+`
		ORDER BY m.created_ts DESC,m.id DESC LIMIT ?`, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	items, err := scanMemoRows(rows)
	if err != nil {
		return nil, nil, err
	}
	var next *repository.TimelineCursor
	if len(items) > limit {
		items = items[:limit]
		last := items[len(items)-1]
		next = &repository.TimelineCursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}
	return items, next, nil
}

func (s *Store) SearchMemos(ctx context.Context, q repository.SearchQuery) ([]domain.Memo, error) {
	limit := clampLimit(q.Limit, 50, 200)
	if q.Offset < 0 {
		q.Offset = 0
	}
	where := []string{"1=1"}
	args := []any{}
	appendTimeRange(&where, &args, q.From, q.To)
	appendTagFilters(&where, &args, q.Tags)
	text := strings.TrimSpace(q.Text)
	from := `FROM memo m`
	order := `ORDER BY m.created_ts DESC,m.id DESC`
	if text != "" {
		from += ` JOIN memo_fts ON memo_fts.rowid=m.id`
		where = append(where, `memo_fts MATCH ?`)
		args = append(args, buildFTSQuery(text))
		order = `ORDER BY bm25(memo_fts),m.created_ts DESC,m.id DESC`
	}
	args = append(args, limit, q.Offset)
	rows, err := s.db.QueryContext(ctx, `SELECT m.id,m.uid,m.content,m.created_ts,m.updated_ts,m.revision,m.favorite `+from+`
		WHERE `+strings.Join(where, " AND ")+` `+order+` LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMemoRows(rows)
}

func (s *Store) ListAllMemos(ctx context.Context) ([]domain.Memo, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,uid,content,created_ts,updated_ts,revision,favorite FROM memo ORDER BY created_ts ASC,id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMemoRows(rows)
}

func (s *Store) CreatedBetween(ctx context.Context, from, to time.Time) ([]time.Time, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT created_ts FROM memo WHERE created_ts>=? AND created_ts<? ORDER BY created_ts ASC,id ASC`, unix(from), unix(to))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []time.Time
	for rows.Next() {
		var ts int64
		if err := rows.Scan(&ts); err != nil {
			return nil, err
		}
		out = append(out, fromUnix(ts))
	}
	return out, rows.Err()
}

const activeSharePredicate = `EXISTS(SELECT 1 FROM memo_share ms WHERE ms.memo_id=m.id AND (ms.expires_ts IS NULL OR ms.expires_ts>?))`

// CountMemos counts distinct memos; multiple active grants count once.
func (s *Store) CountMemos(ctx context.Context, now time.Time) (domain.MemoCounts, error) {
	var counts domain.MemoCounts
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(m.favorite),0),
		COALESCE(SUM(CASE WHEN `+activeSharePredicate+` THEN 1 ELSE 0 END),0) FROM memo m`, unix(now)).Scan(&counts.All, &counts.Favorites, &counts.Shared)
	return counts, err
}

func (s *Store) SetMemoFavorite(ctx context.Context, id int64, favorite bool, updatedAt time.Time) error {
	res, err := s.db.ExecContext(ctx, `UPDATE memo SET
		revision=revision+CASE WHEN favorite<>? THEN 1 ELSE 0 END,
		updated_ts=CASE WHEN favorite<>? THEN ? ELSE updated_ts END,
		favorite=? WHERE id=?`, favorite, favorite, unix(updatedAt), favorite, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func scanMemoRows(rows *sql.Rows) ([]domain.Memo, error) {
	var out []domain.Memo
	for rows.Next() {
		m, err := scanMemo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

func replaceTagsTx(ctx context.Context, tx *sql.Tx, memoID int64, tags []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM memo_tag WHERE memo_id=?`, memoID); err != nil {
		return err
	}
	seen := make(map[string]struct{})
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		norm := strings.ToLower(tag)
		if _, ok := seen[norm]; ok {
			continue
		}
		seen[norm] = struct{}{}
		if _, err := tx.ExecContext(ctx, `INSERT INTO memo_tag(memo_id,tag,tag_norm) VALUES(?,?,?)`, memoID, tag, norm); err != nil {
			return err
		}
	}
	return nil
}

func appendTimeRange(where *[]string, args *[]any, from, to *time.Time) {
	if from != nil {
		*where = append(*where, `m.created_ts>=?`)
		*args = append(*args, unix(*from))
	}
	if to != nil {
		*where = append(*where, `m.created_ts<?`)
		*args = append(*args, unix(*to))
	}
}

func appendTagFilters(where *[]string, args *[]any, tags []string) {
	seen := make(map[string]struct{})
	for _, tag := range tags {
		norm := strings.ToLower(strings.TrimSpace(tag))
		if norm == "" {
			continue
		}
		if _, ok := seen[norm]; ok {
			continue
		}
		seen[norm] = struct{}{}
		*where = append(*where, `EXISTS(SELECT 1 FROM memo_tag mt WHERE mt.memo_id=m.id AND mt.tag_norm=?)`)
		*args = append(*args, norm)
	}
}

func buildFTSQuery(text string) string {
	terms := strings.Fields(text)
	quoted := make([]string, 0, len(terms))
	for _, term := range terms {
		term = strings.ReplaceAll(term, `"`, `""`)
		if term != "" {
			quoted = append(quoted, `"`+term+`"`)
		}
	}
	if len(quoted) == 0 {
		return `""`
	}
	return strings.Join(quoted, " AND ")
}

func clampLimit(v, def, max int) int {
	if v <= 0 {
		return def
	}
	if v > max {
		return max
	}
	return v
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
