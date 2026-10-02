package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/NguyenHien-8/NoteHub/internal/domain"
)

func insertAttachmentTx(ctx context.Context, tx *sql.Tx, a *domain.Attachment) (int64, error) {
	res, err := tx.ExecContext(ctx, `INSERT INTO attachment(uid,memo_id,created_ts,filename,mime_type,size,sha256,relative_path)
		VALUES(?,?,?,?,?,?,?,?)`, a.UID, a.MemoID, unix(a.CreatedAt), a.Filename, a.MIMEType, a.Size, a.SHA256, a.RelativePath)
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
	res, err := s.db.ExecContext(ctx, `INSERT INTO attachment(uid,memo_id,created_ts,filename,mime_type,size,sha256,relative_path)
		VALUES(?,?,?,?,?,?,?,?)`, a.UID, a.MemoID, unix(a.CreatedAt), a.Filename, a.MIMEType, a.Size, a.SHA256, a.RelativePath)
	if err != nil {
		return err
	}
	a.ID, err = res.LastInsertId()
	return err
}

func scanAttachment(scanner interface{ Scan(...any) error }) (*domain.Attachment, error) {
	var a domain.Attachment
	var created int64
	if err := scanner.Scan(&a.ID, &a.UID, &a.MemoID, &created, &a.Filename, &a.MIMEType, &a.Size, &a.SHA256, &a.RelativePath); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	a.CreatedAt = fromUnix(created)
	return &a, nil
}

func (s *Store) GetAttachmentByID(ctx context.Context, id int64) (*domain.Attachment, error) {
	return scanAttachment(s.db.QueryRowContext(ctx, `SELECT id,uid,memo_id,created_ts,filename,mime_type,size,sha256,relative_path FROM attachment WHERE id=?`, id))
}

func (s *Store) GetAttachmentByUID(ctx context.Context, uid string) (*domain.Attachment, error) {
	return scanAttachment(s.db.QueryRowContext(ctx, `SELECT id,uid,memo_id,created_ts,filename,mime_type,size,sha256,relative_path FROM attachment WHERE uid=?`, uid))
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
	rows, err := q.QueryContext(ctx, `SELECT id,uid,memo_id,created_ts,filename,mime_type,size,sha256,relative_path FROM attachment
		WHERE memo_id IN (`+strings.Join(ph, ",")+`) ORDER BY memo_id ASC,created_ts ASC,id ASC`, args...)
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
