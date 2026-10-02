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
	DeleteAttachment(ctx context.Context, id int64) error
	ListAllAttachmentPaths(ctx context.Context) ([]string, error)
}
