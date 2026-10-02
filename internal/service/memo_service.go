package service

import (
	"context"
	"time"

	"github.com/NguyenHien-8/NoteHub/internal/domain"
	"github.com/NguyenHien-8/NoteHub/internal/identity"
	"github.com/NguyenHien-8/NoteHub/internal/repository"
	"github.com/NguyenHien-8/NoteHub/internal/storage"
	"github.com/NguyenHien-8/NoteHub/internal/tagparse"
)

type MemoService struct {
	memos       repository.MemoRepository
	attachments repository.AttachmentRepository
	files       *storage.LocalStore
	hydrator    hydrator
	options     Options
}

func NewMemoService(memos repository.MemoRepository, tags repository.TagRepository, attachments repository.AttachmentRepository, files *storage.LocalStore, options Options) *MemoService {
	return &MemoService{memos: memos, attachments: attachments, files: files, hydrator: hydrator{tags: tags, attachments: attachments}, options: options.withDefaults()}
}

func (s *MemoService) Create(ctx context.Context, content string) (*domain.Memo, error) {
	if err := validateContent(content, s.options.MaxMemoBytes); err != nil {
		return nil, err
	}
	uid, err := identity.NewUID()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Truncate(time.Second)
	memo := &domain.Memo{UID: uid, Content: content, CreatedAt: now, UpdatedAt: now, Revision: 1}
	tags := tagparse.Extract(content)
	if err := s.memos.CreateMemo(ctx, memo, tags); err != nil {
		return nil, err
	}
	memo.Tags = tags
	return memo, nil
}

func (s *MemoService) Get(ctx context.Context, id int64) (*domain.Memo, error) {
	memo, err := s.memos.GetMemoByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.hydrator.one(ctx, memo); err != nil {
		return nil, err
	}
	return memo, nil
}

func (s *MemoService) GetByUID(ctx context.Context, uid string) (*domain.Memo, error) {
	memo, err := s.memos.GetMemoByUID(ctx, uid)
	if err != nil {
		return nil, err
	}
	if err := s.hydrator.one(ctx, memo); err != nil {
		return nil, err
	}
	return memo, nil
}

// Update uses optimistic revision checking. A stale editor receives
// domain.ErrConflict instead of silently overwriting a newer change.
func (s *MemoService) Update(ctx context.Context, id, expectedRevision int64, content string) (*domain.Memo, error) {
	if err := validateContent(content, s.options.MaxMemoBytes); err != nil {
		return nil, err
	}
	tags := tagparse.Extract(content)
	memo, err := s.memos.UpdateMemo(ctx, id, expectedRevision, content, time.Now().UTC().Truncate(time.Second), tags)
	if err != nil {
		return nil, err
	}
	if err := s.hydrator.one(ctx, memo); err != nil {
		return nil, err
	}
	return memo, nil
}

func (s *MemoService) Delete(ctx context.Context, id int64) error {
	attachments, err := s.attachments.ListAttachmentsByMemoIDs(ctx, []int64{id})
	if err != nil {
		return err
	}
	if err := s.memos.DeleteMemo(ctx, id); err != nil {
		return err
	}
	// Once the database deletion commits, metadata is gone. Storage cleanup is
	// best-effort: an orphan file is safer than rolling the DB back after bytes
	// have already been deleted.
	for _, a := range attachments[id] {
		_ = s.files.Delete(a.RelativePath)
	}
	return nil
}

// SetFavorite invalidates stale editors when the favorite state changes.
func (s *MemoService) SetFavorite(ctx context.Context, id int64, favorite bool) error {
	return s.memos.SetMemoFavorite(ctx, id, favorite, time.Now().UTC().Truncate(time.Second))
}
