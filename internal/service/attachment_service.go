package service

import (
	"bufio"
	"context"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/NguyenHien-8/NoteHub/internal/domain"
	"github.com/NguyenHien-8/NoteHub/internal/identity"
	"github.com/NguyenHien-8/NoteHub/internal/repository"
	"github.com/NguyenHien-8/NoteHub/internal/storage"
)

type AttachmentService struct {
	memos       repository.MemoRepository
	attachments repository.AttachmentRepository
	files       *storage.LocalStore
}

func NewAttachmentService(memos repository.MemoRepository, attachments repository.AttachmentRepository, files *storage.LocalStore) *AttachmentService {
	return &AttachmentService{memos: memos, attachments: attachments, files: files}
}

func (s *AttachmentService) AddFromFile(ctx context.Context, memoID int64, sourcePath string) (*domain.Attachment, error) {
	f, err := os.Open(sourcePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return s.Add(ctx, memoID, filepath.Base(sourcePath), "", f)
}

func (s *AttachmentService) Add(ctx context.Context, memoID int64, filename, mimeType string, r io.Reader) (*domain.Attachment, error) {
	memo, err := s.memos.GetMemoByID(ctx, memoID)
	if err != nil {
		return nil, err
	}
	filename = sanitizeDisplayFilename(filename)
	reader := bufio.NewReader(r)
	if strings.TrimSpace(mimeType) == "" {
		mimeType = mime.TypeByExtension(strings.ToLower(filepath.Ext(filename)))
		if mimeType == "" {
			peek, _ := reader.Peek(512)
			if len(peek) != 0 {
				mimeType = http.DetectContentType(peek)
			}
		}
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	uid, err := identity.NewUID()
	if err != nil {
		return nil, err
	}
	saved, err := s.files.Save(memo.UID, uid, filename, reader)
	if err != nil {
		return nil, err
	}
	a := &domain.Attachment{
		UID: uid, MemoID: memoID, CreatedAt: time.Now().UTC().Truncate(time.Second),
		Filename: filename, MIMEType: mimeType, Size: saved.Size, SHA256: saved.SHA256, RelativePath: saved.RelativePath,
	}
	if err := s.attachments.CreateAttachment(ctx, a); err != nil {
		_ = s.files.Delete(saved.RelativePath)
		return nil, err
	}
	return a, nil
}

func (s *AttachmentService) List(ctx context.Context, memoID int64) ([]domain.Attachment, error) {
	m, err := s.attachments.ListAttachmentsByMemoIDs(ctx, []int64{memoID})
	if err != nil {
		return nil, err
	}
	return m[memoID], nil
}

func (s *AttachmentService) ListAll(ctx context.Context, limit, offset int) ([]domain.Attachment, error) {
	return s.attachments.ListAllAttachments(ctx, limit, offset)
}

// Reorder saves a complete attachment ordering for a memo. Callers must include
// non-image attachments too; a stale attachment set returns domain.ErrConflict.
func (s *AttachmentService) Reorder(ctx context.Context, memoID int64, orderedIDs []int64) error {
	return s.attachments.ReorderAttachments(ctx, memoID, orderedIDs)
}

func (s *AttachmentService) Open(ctx context.Context, attachmentID int64) (*os.File, *domain.Attachment, error) {
	a, err := s.attachments.GetAttachmentByID(ctx, attachmentID)
	if err != nil {
		return nil, nil, err
	}
	f, err := s.files.Open(a.RelativePath)
	if err != nil {
		return nil, nil, err
	}
	return f, a, nil
}

func (s *AttachmentService) Path(ctx context.Context, attachmentID int64) (string, error) {
	a, err := s.attachments.GetAttachmentByID(ctx, attachmentID)
	if err != nil {
		return "", err
	}
	return s.files.Path(a.RelativePath)
}

func (s *AttachmentService) Delete(ctx context.Context, attachmentID int64) error {
	a, err := s.attachments.GetAttachmentByID(ctx, attachmentID)
	if err != nil {
		return err
	}
	if err := s.attachments.DeleteAttachment(ctx, attachmentID); err != nil {
		return err
	}
	// Metadata is already committed. A failed filesystem cleanup only leaves an
	// inaccessible orphan and must not make callers retry a destructive DB op.
	_ = s.files.Delete(a.RelativePath)
	return nil
}

func sanitizeDisplayFilename(filename string) string {
	filename = strings.TrimSpace(strings.ReplaceAll(filename, `\`, `/`))
	filename = path.Base(filename)
	if filename == "." || filename == "/" || filename == "" {
		return "attachment.bin"
	}
	// Keep user-visible Unicode, but remove control characters and path-ish NUL.
	filename = strings.Map(func(r rune) rune {
		if r == 0 || r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, filename)
	if filename == "" {
		return "attachment.bin"
	}
	return filename
}
