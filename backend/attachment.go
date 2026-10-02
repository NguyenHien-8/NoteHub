package backend

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Service) AddAttachmentFromFile(ctx context.Context, memoID int64, sourcePath string) (*Attachment, error) {
	f, err := os.Open(sourcePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	name := filepath.Base(sourcePath)
	mimeType := mime.TypeByExtension(strings.ToLower(filepath.Ext(name)))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	return s.AddAttachment(ctx, memoID, name, mimeType, f)
}

func (s *Service) AddAttachment(ctx context.Context, memoID int64, filename, mimeType string, r io.Reader) (*Attachment, error) {
	if memoID <= 0 {
		return nil, errors.New("invalid memo id")
	}
	memo, err := s.GetMemo(ctx, memoID)
	if err != nil {
		return nil, err
	}
	uid, err := newUID()
	if err != nil {
		return nil, err
	}
	filename = filepath.Base(strings.TrimSpace(filename))
	if filename == "." || filename == "" {
		filename = "attachment.bin"
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	ext := filepath.Ext(filename)
	relative := filepath.Join(memo.UID, uid+ext)
	full, err := ensureInside(s.attachmentRoot, relative)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return nil, err
	}
	out, err := os.OpenFile(full, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	hash := sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(out, hash), r)
	closeErr := out.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(full)
		if copyErr != nil {
			return nil, copyErr
		}
		return nil, closeErr
	}
	now := time.Now().UTC().Truncate(time.Second)
	result, err := s.db.ExecContext(ctx, `INSERT INTO attachment(uid,memo_id,created_ts,filename,mime_type,size,sha256,relative_path) VALUES(?,?,?,?,?,?,?,?)`,
		uid, memoID, unix(now), filename, mimeType, size, hex.EncodeToString(hash.Sum(nil)), filepath.ToSlash(relative))
	if err != nil {
		_ = os.Remove(full)
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Attachment{ID: id, UID: uid, MemoID: memoID, CreatedAt: now, Filename: filename, MIMEType: mimeType, Size: size, SHA256: hex.EncodeToString(hash.Sum(nil)), RelativePath: filepath.ToSlash(relative)}, nil
}

func (s *Service) ListAttachments(ctx context.Context, memoID int64) ([]Attachment, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,uid,memo_id,created_ts,filename,mime_type,size,sha256,relative_path FROM attachment WHERE memo_id=? ORDER BY created_ts ASC,id ASC`, memoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Attachment
	for rows.Next() {
		var a Attachment
		var ts int64
		if err := rows.Scan(&a.ID, &a.UID, &a.MemoID, &ts, &a.Filename, &a.MIMEType, &a.Size, &a.SHA256, &a.RelativePath); err != nil {
			return nil, err
		}
		a.CreatedAt = fromUnix(ts)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Service) AttachmentPath(ctx context.Context, attachmentID int64) (string, error) {
	var relative string
	if err := s.db.QueryRowContext(ctx, `SELECT relative_path FROM attachment WHERE id=?`, attachmentID).Scan(&relative); err != nil {
		return "", err
	}
	return ensureInside(s.attachmentRoot, filepath.FromSlash(relative))
}

func (s *Service) DeleteAttachment(ctx context.Context, attachmentID int64) error {
	var relative string
	if err := s.db.QueryRowContext(ctx, `SELECT relative_path FROM attachment WHERE id=?`, attachmentID).Scan(&relative); err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM attachment WHERE id=?`, attachmentID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	full, err := ensureInside(s.attachmentRoot, filepath.FromSlash(relative))
	if err == nil {
		_ = os.Remove(full)
	}
	return nil
}

func (s *Service) copyAttachmentTo(ctx context.Context, attachment Attachment, dst io.Writer) error {
	path, err := ensureInside(s.attachmentRoot, filepath.FromSlash(attachment.RelativePath))
	if err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open attachment %s: %w", attachment.UID, err)
	}
	defer f.Close()
	_, err = io.Copy(dst, f)
	return err
}
