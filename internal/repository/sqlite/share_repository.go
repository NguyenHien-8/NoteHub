package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/NguyenHien-8/NoteHub/internal/domain"
)

func nullableUnix(t *time.Time) any {
	if t == nil {
		return nil
	}
	return unix(*t)
}

func (s *Store) CreateShare(ctx context.Context, sh *domain.Share, tokenHash []byte) error {
	res, err := s.db.ExecContext(ctx, `INSERT INTO memo_share(uid,memo_id,token_hash,created_ts,expires_ts) VALUES(?,?,?,?,?)`,
		sh.UID, sh.MemoID, tokenHash, unix(sh.CreatedAt), nullableUnix(sh.ExpiresAt))
	if err != nil {
		return err
	}
	sh.ID, err = res.LastInsertId()
	return err
}

func scanShare(scanner interface{ Scan(...any) error }) (*domain.Share, error) {
	var sh domain.Share
	var created int64
	var expires sql.NullInt64
	if err := scanner.Scan(&sh.ID, &sh.UID, &sh.MemoID, &created, &expires); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	sh.CreatedAt = fromUnix(created)
	if expires.Valid {
		t := fromUnix(expires.Int64)
		sh.ExpiresAt = &t
	}
	return &sh, nil
}

func (s *Store) ResolveShareByTokenHash(ctx context.Context, tokenHash []byte) (*domain.Share, error) {
	return scanShare(s.db.QueryRowContext(ctx, `SELECT id,uid,memo_id,created_ts,expires_ts FROM memo_share WHERE token_hash=?`, tokenHash))
}

func (s *Store) ListSharesByMemoID(ctx context.Context, memoID int64) ([]domain.Share, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,uid,memo_id,created_ts,expires_ts FROM memo_share WHERE memo_id=? ORDER BY created_ts DESC,id DESC`, memoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Share
	for rows.Next() {
		sh, err := scanShare(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *sh)
	}
	return out, rows.Err()
}

func (s *Store) DeleteShareByUID(ctx context.Context, uid string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM memo_share WHERE uid=?`, uid)
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

func (s *Store) DeleteExpiredShares(ctx context.Context, now time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM memo_share WHERE expires_ts IS NOT NULL AND expires_ts<=?`, unix(now))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
