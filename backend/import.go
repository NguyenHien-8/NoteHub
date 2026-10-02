package backend

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"example.com/memosdesktop/backend/internal/safezip"
	"example.com/memosdesktop/backend/internal/tagparse"
)

const maxImportJSON = 64 << 20 // 64 MiB metadata limit.

func (s *Service) Import(ctx context.Context, zipPath string, policy ImportPolicy) (*ImportReport, error) {
	if policy != ImportSkip && policy != ImportOverwrite {
		return nil, errors.New("policy must be skip or overwrite")
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	entries := map[string]*zip.File{}
	for _, f := range zr.File {
		if err := safezip.ValidateName(f.Name); err != nil {
			return nil, fmt.Errorf("unsafe zip entry %q: %w", f.Name, err)
		}
		entries[f.Name] = f
	}
	manifestFile := entries["manifest.json"]
	memoFile := entries["memos.json"]
	if manifestFile == nil || memoFile == nil {
		return nil, errors.New("archive must contain manifest.json and memos.json")
	}
	var manifest exportManifest
	if err := decodeZipJSON(manifestFile, &manifest); err != nil {
		return nil, err
	}
	if manifest.Format != exportFormat || manifest.Version != exportVersion {
		return nil, fmt.Errorf("unsupported export format %q version %q", manifest.Format, manifest.Version)
	}
	var records []exportMemo
	if err := decodeZipJSON(memoFile, &records); err != nil {
		return nil, err
	}

	report := &ImportReport{}
	for _, rec := range records {
		if strings.TrimSpace(rec.UID) == "" {
			report.Warnings = append(report.Warnings, "skipped memo with empty UID")
			report.Skipped++
			continue
		}
		existing, err := s.GetMemoByUID(ctx, rec.UID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if existing != nil && policy == ImportSkip {
			report.Skipped++
			continue
		}
		memoID, err := s.importMemoRecord(ctx, rec, existing)
		if err != nil {
			return nil, fmt.Errorf("import memo %s: %w", rec.UID, err)
		}
		if existing == nil {
			report.Created++
		} else {
			report.Updated++
			if err := s.deleteAllAttachmentsForMemo(ctx, memoID); err != nil {
				return nil, fmt.Errorf("replace attachments for memo %s: %w", rec.UID, err)
			}
		}
		for _, a := range rec.Attachments {
			zf := entries[a.ArchivePath]
			if zf == nil {
				report.Warnings = append(report.Warnings, fmt.Sprintf("memo %s attachment %s missing", rec.UID, a.UID))
				continue
			}
			created, err := s.importAttachment(ctx, memoID, rec.UID, a, zf)
			if err != nil {
				return nil, fmt.Errorf("import attachment %s: %w", a.UID, err)
			}
			if created {
				report.Attachments++
			}
		}
	}
	return report, nil
}

func (s *Service) importMemoRecord(ctx context.Context, rec exportMemo, existing *Memo) (int64, error) {
	tags := tagparse.Extract(rec.Content)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	var id int64
	if existing == nil {
		result, err := tx.ExecContext(ctx, `INSERT INTO memo(uid,created_ts,updated_ts,content,search_text) VALUES(?,?,?,?,?)`,
			rec.UID, unix(rec.CreatedAt), unix(rec.UpdatedAt), rec.Content, normalizeSearch(rec.Content))
		if err != nil {
			return 0, err
		}
		id, err = result.LastInsertId()
		if err != nil {
			return 0, err
		}
	} else {
		id = existing.ID
		if _, err := tx.ExecContext(ctx, `UPDATE memo SET created_ts=?,updated_ts=?,content=?,search_text=? WHERE id=?`,
			unix(rec.CreatedAt), unix(rec.UpdatedAt), rec.Content, normalizeSearch(rec.Content), id); err != nil {
			return 0, err
		}
	}
	if err := replaceTagsTx(ctx, tx, id, tags); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Service) importAttachment(ctx context.Context, memoID int64, memoUID string, rec exportAttachment, zf *zip.File) (bool, error) {
	var existingID int64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM attachment WHERE uid=?`, rec.UID).Scan(&existingID)
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	rc, err := zf.Open()
	if err != nil {
		return false, err
	}
	defer rc.Close()
	ext := filepath.Ext(rec.Filename)
	relative := filepath.Join(memoUID, rec.UID+ext)
	full, err := ensureInside(s.attachmentRoot, relative)
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return false, err
	}
	out, err := os.OpenFile(full, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return false, err
	}
	h := sha256.New()
	var source io.Reader = rc
	if rec.Size > 0 {
		source = io.LimitReader(rc, rec.Size+1)
	}
	n, copyErr := io.Copy(io.MultiWriter(out, h), source)
	closeErr := out.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(full)
		if copyErr != nil {
			return false, copyErr
		}
		return false, closeErr
	}
	digest := hex.EncodeToString(h.Sum(nil))
	if rec.Size != 0 && rec.Size != n {
		_ = os.Remove(full)
		return false, fmt.Errorf("size mismatch: got %d want %d", n, rec.Size)
	}
	if rec.SHA256 != "" && !strings.EqualFold(rec.SHA256, digest) {
		_ = os.Remove(full)
		return false, errors.New("sha256 mismatch")
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO attachment(uid,memo_id,created_ts,filename,mime_type,size,sha256,relative_path) VALUES(?,?,?,?,?,?,?,?)`,
		rec.UID, memoID, unix(rec.CreatedAt), rec.Filename, rec.MIMEType, n, digest, filepath.ToSlash(relative))
	if err != nil {
		_ = os.Remove(full)
		return false, err
	}
	return true, nil
}

func (s *Service) deleteAllAttachmentsForMemo(ctx context.Context, memoID int64) error {
	attachments, err := s.ListAttachments(ctx, memoID)
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM attachment WHERE memo_id=?`, memoID); err != nil {
		return err
	}
	for _, a := range attachments {
		if p, err := ensureInside(s.attachmentRoot, filepath.FromSlash(a.RelativePath)); err == nil {
			_ = os.Remove(p)
		}
	}
	return nil
}

func decodeZipJSON(f *zip.File, dst any) error {
	r, err := f.Open()
	if err != nil {
		return err
	}
	defer r.Close()
	b, err := readAllLimit(r, maxImportJSON)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, dst); err != nil {
		return fmt.Errorf("decode %s: %w", f.Name, err)
	}
	return nil
}
