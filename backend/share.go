package backend

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

func (s *Service) CreateShare(ctx context.Context, memoID int64, expiresAt *time.Time) (*Share, error) {
	if _, err := s.GetMemo(ctx, memoID); err != nil {
		return nil, err
	}
	token, err := newUID()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Truncate(time.Second)
	var expires any
	if expiresAt != nil {
		v := expiresAt.UTC().Truncate(time.Second)
		expiresAt = &v
		expires = unix(v)
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO memo_share(token,memo_id,created_ts,expires_ts) VALUES(?,?,?,?)`, token, memoID, unix(now), expires)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return &Share{ID: id, Token: token, MemoID: memoID, CreatedAt: now, ExpiresAt: expiresAt}, nil
}

func (s *Service) RevokeShare(ctx context.Context, token string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM memo_share WHERE token=?`, token)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Service) ResolveShare(ctx context.Context, token string) (*SharedMemo, error) {
	var sh Share
	var created int64
	var expires sql.NullInt64
	if err := s.db.QueryRowContext(ctx, `SELECT id,token,memo_id,created_ts,expires_ts FROM memo_share WHERE token=?`, token).
		Scan(&sh.ID, &sh.Token, &sh.MemoID, &created, &expires); err != nil {
		return nil, err
	}
	sh.CreatedAt = fromUnix(created)
	if expires.Valid {
		t := fromUnix(expires.Int64)
		sh.ExpiresAt = &t
		if time.Now().UTC().After(t) {
			return nil, errors.New("share expired")
		}
	}
	memo, err := s.GetMemo(ctx, sh.MemoID)
	if err != nil {
		return nil, err
	}
	return &SharedMemo{Share: sh, Memo: *memo}, nil
}
