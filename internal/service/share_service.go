package service

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/NguyenHien-8/NoteHub/internal/domain"
	"github.com/NguyenHien-8/NoteHub/internal/identity"
	"github.com/NguyenHien-8/NoteHub/internal/repository"
	"github.com/NguyenHien-8/NoteHub/internal/storage"
)

type ShareService struct {
	shares      repository.ShareRepository
	memos       repository.MemoRepository
	attachments repository.AttachmentRepository
	files       *storage.LocalStore
	hydrator    hydrator
}

func NewShareService(shares repository.ShareRepository, memos repository.MemoRepository, tags repository.TagRepository, attachments repository.AttachmentRepository, files *storage.LocalStore) *ShareService {
	return &ShareService{shares: shares, memos: memos, attachments: attachments, files: files, hydrator: hydrator{tags: tags, attachments: attachments}}
}

func (s *ShareService) Create(ctx context.Context, memoID int64, expiresAt *time.Time) (*domain.ShareGrant, error) {
	if _, err := s.memos.GetMemoByID(ctx, memoID); err != nil {
		return nil, err
	}
	now := time.Now().UTC().Truncate(time.Second)
	if expiresAt != nil {
		t := expiresAt.UTC().Truncate(time.Second)
		if !t.After(now) {
			return nil, fmt.Errorf("%w: share expiration must be in the future", domain.ErrInvalid)
		}
		expiresAt = &t
	}
	uid, err := identity.NewUID()
	if err != nil {
		return nil, err
	}
	token, err := identity.NewBearerToken()
	if err != nil {
		return nil, err
	}
	hash, err := identity.HashToken(token)
	if err != nil {
		return nil, err
	}
	sh := &domain.Share{UID: uid, MemoID: memoID, CreatedAt: now, ExpiresAt: expiresAt}
	if err := s.shares.CreateShare(ctx, sh, hash); err != nil {
		return nil, err
	}
	return &domain.ShareGrant{Share: *sh, Token: token}, nil
}

func (s *ShareService) Resolve(ctx context.Context, token string) (*domain.SharedMemo, error) {
	hash, err := identity.HashToken(token)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	sh, err := s.shares.ResolveShareByTokenHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if sh.ExpiresAt != nil && !time.Now().UTC().Before(*sh.ExpiresAt) {
		return nil, domain.ErrNotFound
	}
	memo, err := s.memos.GetMemoByID(ctx, sh.MemoID)
	if err != nil {
		return nil, err
	}
	if err := s.hydrator.one(ctx, memo); err != nil {
		return nil, err
	}
	return &domain.SharedMemo{Share: *sh, Memo: *memo}, nil
}

func (s *ShareService) List(ctx context.Context, memoID int64) ([]domain.Share, error) {
	return s.shares.ListSharesByMemoID(ctx, memoID)
}

func (s *ShareService) Revoke(ctx context.Context, shareUID string) error {
	return s.shares.DeleteShareByUID(ctx, shareUID)
}

func (s *ShareService) PurgeExpired(ctx context.Context) (int64, error) {
	return s.shares.DeleteExpiredShares(ctx, time.Now().UTC())
}

// OpenSharedAttachment re-resolves the bearer grant before opening bytes. This
// prevents a caller from taking a valid memo token and guessing attachment IDs
// belonging to another memo.
func (s *ShareService) OpenSharedAttachment(ctx context.Context, token, attachmentUID string) (*os.File, *domain.Attachment, error) {
	shared, err := s.Resolve(ctx, token)
	if err != nil {
		return nil, nil, err
	}
	a, err := s.attachments.GetAttachmentByUID(ctx, attachmentUID)
	if err != nil {
		return nil, nil, err
	}
	if a.MemoID != shared.Memo.ID {
		return nil, nil, domain.ErrNotFound
	}
	f, err := s.files.Open(a.RelativePath)
	if err != nil {
		return nil, nil, err
	}
	return f, a, nil
}
