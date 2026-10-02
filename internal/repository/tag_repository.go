package repository

import (
	"context"

	"github.com/NguyenHien-8/NoteHub/internal/domain"
)

type TagRepository interface {
	TagsByMemoIDs(ctx context.Context, memoIDs []int64) (map[int64][]string, error)
	ListTagCounts(ctx context.Context) ([]domain.TagCount, error)
}
