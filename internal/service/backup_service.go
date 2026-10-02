package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/NguyenHien-8/NoteHub/internal/backup"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
	"github.com/NguyenHien-8/NoteHub/internal/identity"
	"github.com/NguyenHien-8/NoteHub/internal/repository"
	"github.com/NguyenHien-8/NoteHub/internal/storage"
	"github.com/NguyenHien-8/NoteHub/internal/tagparse"
)

type ImportPolicy int

const (
	ImportSkip ImportPolicy = iota
	ImportReplace
	ImportDuplicate
)

type ImportFailure struct{ UID, Message string }
type ImportReport struct {
	Created    int
	Replaced   int
	Duplicated int
	Skipped    int
	Failures   []ImportFailure
}

type BackupService struct {
	memos       repository.MemoRepository
	tags        repository.TagRepository
	attachments repository.AttachmentRepository
	files       *storage.LocalStore
	hydrator    hydrator
	options     Options
	version     string
}

func NewBackupService(memos repository.MemoRepository, tags repository.TagRepository, attachments repository.AttachmentRepository, files *storage.LocalStore, options Options, version string) *BackupService {
	if version == "" {
		version = "dev"
	}
	return &BackupService{memos: memos, tags: tags, attachments: attachments, files: files, hydrator: hydrator{tags: tags, attachments: attachments}, options: options.withDefaults(), version: version}
}

func (s *BackupService) ExportFile(ctx context.Context, destination string) error {
	dir := filepath.Dir(destination)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".notehub-export-*.zip")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		_ = tmp.Close()
		if !ok {
			_ = os.Remove(tmpName)
		}
	}()
	if err := s.Export(ctx, tmp); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if _, err := os.Stat(destination); err == nil {
		return fmt.Errorf("destination already exists: %s", destination)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(tmpName, destination); err != nil {
		return err
	}
	ok = true
	return nil
}

func (s *BackupService) Export(ctx context.Context, out io.Writer) error {
	memos, err := s.memos.ListAllMemos(ctx)
	if err != nil {
		return err
	}
	if err := s.hydrator.hydrate(ctx, memos); err != nil {
		return err
	}
	attachmentCount := 0
	for _, m := range memos {
		attachmentCount += len(m.Attachments)
	}
	exportTime := time.Now().UTC().Truncate(time.Second)
	w := backup.NewWriter(out, exportTime)
	if err := w.WriteManifest(backup.Manifest{
		Format: backup.Format, FormatVersion: backup.FormatVersion,
		Generator:  backup.Generator{Name: "NoteHub", Version: s.version},
		ExportTime: backup.FormatTime(exportTime),
		Counts:     backup.Counts{Memos: len(memos), Attachments: attachmentCount},
	}); err != nil {
		return err
	}
	closed := false
	defer func() {
		if !closed {
			_ = w.Close()
		}
	}()
	for _, memo := range memos {
		rec := backup.MemoRecord{UID: memo.UID, CreateTime: backup.FormatTime(memo.CreatedAt), UpdateTime: backup.FormatTime(memo.UpdatedAt), ContentPath: backup.ContentPath(memo.UID), Favorite: memo.Favorite}
		for _, a := range memo.Attachments {
			f, err := s.files.Open(a.RelativePath)
			if err != nil {
				return fmt.Errorf("open attachment %s: %w", a.UID, err)
			}
			entryPath := backup.AttachmentPath(a.UID, a.Filename)
			digest, size, writeErr := w.WriteAttachment(entryPath, f)
			_ = f.Close()
			if writeErr != nil {
				return writeErr
			}
			if digest != a.SHA256 || size != a.Size {
				return fmt.Errorf("attachment %s failed integrity check during export", a.UID)
			}
			rec.Attachments = append(rec.Attachments, backup.AttachmentRecord{UID: a.UID, Filename: a.Filename, Type: a.MIMEType, Size: size, SHA256: digest, CreateTime: backup.FormatTime(a.CreatedAt), Path: entryPath})
		}
		if err := w.WriteMemo(rec, []byte(memo.Content)); err != nil {
			return err
		}
	}
	if err := w.Close(); err != nil {
		return err
	}
	closed = true
	return nil
}

func (s *BackupService) ImportFile(ctx context.Context, archivePath string, policy ImportPolicy) (*ImportReport, error) {
	if policy < ImportSkip || policy > ImportDuplicate {
		return nil, fmt.Errorf("%w: unknown import policy", domain.ErrInvalid)
	}
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	const maxImportArchiveBytes int64 = 4 << 30
	if st.Size() < 0 || st.Size() > maxImportArchiveBytes {
		return nil, fmt.Errorf("import archive exceeds %d bytes", maxImportArchiveBytes)
	}
	archive, err := backup.Read(f, st.Size())
	if err != nil {
		return nil, err
	}
	report := &ImportReport{}
	for _, rec := range archive.Memos {
		if err := s.importOne(ctx, archive, rec, policy, report); err != nil {
			report.Failures = append(report.Failures, ImportFailure{UID: rec.UID, Message: err.Error()})
		}
	}
	return report, nil
}

func (s *BackupService) importOne(ctx context.Context, archive *backup.Archive, rec backup.MemoRecord, policy ImportPolicy, report *ImportReport) error {
	contentBytes, err := archive.Content(rec)
	if err != nil {
		return err
	}
	content := string(contentBytes)
	if err := validateContent(content, s.options.MaxMemoBytes); err != nil {
		return err
	}
	created, err := backup.ParseTime(rec.CreateTime)
	if err != nil {
		return err
	}
	updated, err := backup.ParseTime(rec.UpdateTime)
	if err != nil {
		return err
	}

	existing, err := s.memos.GetMemoByUID(ctx, rec.UID)
	exists := err == nil
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	if exists && policy == ImportSkip {
		report.Skipped++
		return nil
	}

	targetUID := rec.UID
	mode := "create"
	if exists {
		switch policy {
		case ImportReplace:
			mode = "replace"
		case ImportDuplicate:
			targetUID, err = identity.NewUID()
			if err != nil {
				return err
			}
			mode = "duplicate"
		}
	}
	memo := &domain.Memo{UID: targetUID, Content: content, CreatedAt: created, UpdatedAt: updated, Revision: 1, Favorite: rec.Favorite}
	tags := tagparse.Extract(content)
	newAttachments := make([]domain.Attachment, 0, len(rec.Attachments))
	cleanup := func() {
		for _, a := range newAttachments {
			_ = s.files.Delete(a.RelativePath)
		}
	}
	for _, ar := range rec.Attachments {
		r, err := archive.OpenAttachment(ar, s.options.MaxAttachmentBytes)
		if err != nil {
			cleanup()
			return err
		}
		newUID, err := identity.NewUID()
		if err != nil {
			_ = r.Close()
			cleanup()
			return err
		}
		saved, saveErr := s.files.Save(targetUID, newUID, ar.Filename, r)
		_ = r.Close()
		if saveErr != nil {
			cleanup()
			return saveErr
		}
		if saved.Size != ar.Size || saved.SHA256 != ar.SHA256 {
			_ = s.files.Delete(saved.RelativePath)
			cleanup()
			return fmt.Errorf("attachment %s integrity mismatch while importing", ar.UID)
		}
		atTime, err := backup.ParseTime(ar.CreateTime)
		if err != nil {
			_ = s.files.Delete(saved.RelativePath)
			cleanup()
			return err
		}
		newAttachments = append(newAttachments, domain.Attachment{UID: newUID, CreatedAt: atTime, Filename: ar.Filename, MIMEType: ar.Type, Size: saved.Size, SHA256: saved.SHA256, RelativePath: saved.RelativePath})
	}

	switch mode {
	case "replace":
		old, err := s.memos.ReplaceImportedMemo(ctx, existing.ID, memo, tags, newAttachments)
		if err != nil {
			cleanup()
			return err
		}
		for _, a := range old {
			_ = s.files.Delete(a.RelativePath)
		}
		report.Replaced++
	case "duplicate":
		if err := s.memos.CreateImportedMemo(ctx, memo, tags, newAttachments); err != nil {
			cleanup()
			return err
		}
		report.Duplicated++
	default:
		if err := s.memos.CreateImportedMemo(ctx, memo, tags, newAttachments); err != nil {
			cleanup()
			return err
		}
		report.Created++
	}
	return nil
}
