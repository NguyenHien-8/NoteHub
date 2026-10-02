package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/NguyenHien-8/NoteHub/internal/domain"
)

func insertAttachmentTx(ctx context.Context, tx *sql.Tx, a *domain.Attachment) (int64, error) {
	res, err := tx.ExecContext(ctx, `INSERT INTO attachment(uid,memo_id,created_ts,filename,mime_type,size,sha256,relative_path,sort_order)
		VALUES(?,?,?,?,?,?,?,?,?)`, a.UID, a.MemoID, unix(a.CreatedAt), a.Filename, a.MIMEType, a.Size, a.SHA256, a.RelativePath, a.Position)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err == nil {
		a.ID = id
	}
	return id, err
}

func (s *Store) CreateAttachment(ctx context.Context, a *domain.Attachment) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// The IMMEDIATE transaction serializes appends with reorders and other
	// appends, including writes through a second connection to the database.
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sort_order),-1)+1 FROM attachment WHERE memo_id=?`, a.MemoID).Scan(&a.Position); err != nil {
		return err
	}
	if _, err := insertAttachmentTx(ctx, tx, a); err != nil {
		return err
	}
	return tx.Commit()
}

func scanAttachment(scanner interface{ Scan(...any) error }) (*domain.Attachment, error) {
	var a domain.Attachment
	var created int64
	if err := scanner.Scan(&a.ID, &a.UID, &a.MemoID, &created, &a.Filename, &a.MIMEType, &a.Size, &a.SHA256, &a.RelativePath, &a.Position); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	a.CreatedAt = fromUnix(created)
	return &a, nil
}

func (s *Store) GetAttachmentByID(ctx context.Context, id int64) (*domain.Attachment, error) {
	return scanAttachment(s.db.QueryRowContext(ctx, `SELECT id,uid,memo_id,created_ts,filename,mime_type,size,sha256,relative_path,sort_order FROM attachment WHERE id=?`, id))
}

func (s *Store) GetAttachmentByUID(ctx context.Context, uid string) (*domain.Attachment, error) {
	return scanAttachment(s.db.QueryRowContext(ctx, `SELECT id,uid,memo_id,created_ts,filename,mime_type,size,sha256,relative_path,sort_order FROM attachment WHERE uid=?`, uid))
}

func (s *Store) ListAttachmentsByMemoIDs(ctx context.Context, memoIDs []int64) (map[int64][]domain.Attachment, error) {
	return listAttachmentsExec(ctx, s.db, memoIDs)
}

func listAttachmentsTx(ctx context.Context, tx *sql.Tx, memoIDs []int64) (map[int64][]domain.Attachment, error) {
	return listAttachmentsExec(ctx, tx, memoIDs)
}

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func listAttachmentsExec(ctx context.Context, q queryer, memoIDs []int64) (map[int64][]domain.Attachment, error) {
	out := make(map[int64][]domain.Attachment)
	if len(memoIDs) == 0 {
		return out, nil
	}
	ph := make([]string, len(memoIDs))
	args := make([]any, len(memoIDs))
	for i, id := range memoIDs {
		ph[i] = "?"
		args[i] = id
	}
	rows, err := q.QueryContext(ctx, `SELECT id,uid,memo_id,created_ts,filename,mime_type,size,sha256,relative_path,sort_order FROM attachment
		WHERE memo_id IN (`+strings.Join(ph, ",")+`) ORDER BY memo_id ASC,sort_order ASC,id ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		a, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		out[a.MemoID] = append(out[a.MemoID], *a)
	}
	return out, rows.Err()
}

func (s *Store) DeleteAttachment(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM attachment WHERE id=?`, id)
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

func (s *Store) ReorderAttachments(ctx context.Context, memoID int64, orderedIDs []int64) error {
	if memoID <= 0 {
		return fmt.Errorf("%w: memo ID must be positive", domain.ErrInvalid)
	}
	seen := make(map[int64]struct{}, len(orderedIDs))
	for _, id := range orderedIDs {
		if id <= 0 {
			return fmt.Errorf("%w: attachment ID must be positive", domain.ErrInvalid)
		}
		if _, duplicate := seen[id]; duplicate {
			return fmt.Errorf("%w: duplicate attachment ID", domain.ErrInvalid)
		}
		seen[id] = struct{}{}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM memo WHERE id=?`, memoID).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return domain.ErrNotFound
	}
	current, err := listAttachmentsTx(ctx, tx, []int64{memoID})
	if err != nil {
		return err
	}
	if len(current[memoID]) != len(orderedIDs) {
		return domain.ErrConflict
	}
	for _, a := range current[memoID] {
		if _, ok := seen[a.ID]; !ok {
			return domain.ErrConflict
		}
	}
	for position, id := range orderedIDs {
		if _, err := tx.ExecContext(ctx, `UPDATE attachment SET sort_order=? WHERE memo_id=? AND id=?`, position, memoID, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ListAllAttachmentPaths(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT relative_path FROM attachment ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) ListAllAttachments(ctx context.Context, limit, offset int) ([]domain.Attachment, error) {
	limit = clampLimit(limit, 50, 200)
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,uid,memo_id,created_ts,filename,mime_type,size,sha256,relative_path,sort_order
		FROM attachment ORDER BY created_ts DESC,id DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Attachment
	for rows.Next() {
		a, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}
