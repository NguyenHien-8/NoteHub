package repository

import (
	"context"

	"github.com/NguyenHien-8/NoteHub/internal/domain"
)

type AttachmentRepository interface {
	CreateAttachment(ctx context.Context, attachment *domain.Attachment) error
	GetAttachmentByID(ctx context.Context, id int64) (*domain.Attachment, error)
	GetAttachmentByUID(ctx context.Context, uid string) (*domain.Attachment, error)
	ListAttachmentsByMemoIDs(ctx context.Context, memoIDs []int64) (map[int64][]domain.Attachment, error)
	// ReorderAttachments atomically replaces the order of a memo's complete
	// attachment set. Stale membership returns domain.ErrConflict.
	ReorderAttachments(ctx context.Context, memoID int64, orderedIDs []int64) error
	DeleteAttachment(ctx context.Context, id int64) error
	ListAllAttachmentPaths(ctx context.Context) ([]string, error)
	ListAllAttachments(ctx context.Context, limit, offset int) ([]domain.Attachment, error)
}
