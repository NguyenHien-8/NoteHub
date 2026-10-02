package repository

import (
	"context"
	"time"

	"github.com/NguyenHien-8/NoteHub/internal/domain"
)

type TimelineCursor struct {
	CreatedAt time.Time
	ID        int64
}

type TimelineQuery struct {
	From   *time.Time
	To     *time.Time // exclusive
	Tags   []string
	Limit  int
	Cursor *TimelineCursor
}

type SearchQuery struct {
	Text   string
	Tags   []string
	From   *time.Time
	To     *time.Time // exclusive
	Limit  int
	Offset int
}

// MemoRepository owns atomic memo+tag mutations. Import methods also bind
// attachment metadata in the same SQLite transaction so a backup restore never
// leaves a half-written database record.
type MemoRepository interface {
	CreateMemo(ctx context.Context, memo *domain.Memo, tags []string) error
	CreateImportedMemo(ctx context.Context, memo *domain.Memo, tags []string, attachments []domain.Attachment) error
	GetMemoByID(ctx context.Context, id int64) (*domain.Memo, error)
	GetMemoByUID(ctx context.Context, uid string) (*domain.Memo, error)
	UpdateMemo(ctx context.Context, id, expectedRevision int64, content string, updatedAt time.Time, tags []string) (*domain.Memo, error)
	ReplaceImportedMemo(ctx context.Context, id int64, memo *domain.Memo, tags []string, attachments []domain.Attachment) ([]domain.Attachment, error)
	DeleteMemo(ctx context.Context, id int64) error
	ListTimeline(ctx context.Context, q TimelineQuery) ([]domain.Memo, *TimelineCursor, error)
	SearchMemos(ctx context.Context, q SearchQuery) ([]domain.Memo, error)
	ListAllMemos(ctx context.Context) ([]domain.Memo, error)
	CreatedBetween(ctx context.Context, from, to time.Time) ([]time.Time, error)
}
