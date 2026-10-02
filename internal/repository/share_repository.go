package repository

import (
	"context"
	"time"

	"github.com/NguyenHien-8/NoteHub/internal/domain"
)

type ShareRepository interface {
	CreateShare(ctx context.Context, share *domain.Share, tokenHash []byte) error
	ResolveShareByTokenHash(ctx context.Context, tokenHash []byte) (*domain.Share, error)
	ListSharesByMemoID(ctx context.Context, memoID int64) ([]domain.Share, error)
	DeleteShareByUID(ctx context.Context, uid string) error
	DeleteExpiredShares(ctx context.Context, now time.Time) (int64, error)
}
