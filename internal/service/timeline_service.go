package service

import (
	"context"
	"time"

	"github.com/NguyenHien-8/NoteHub/internal/domain"
	"github.com/NguyenHien-8/NoteHub/internal/repository"
)

type TimelineService struct {
	memos    repository.MemoRepository
	hydrator hydrator
}

func (s *TimelineService) Counts(ctx context.Context) (domain.MemoCounts, error) {
	return s.memos.CountMemos(ctx, time.Now().UTC())
}

func NewTimelineService(memos repository.MemoRepository, tags repository.TagRepository, attachments repository.AttachmentRepository) *TimelineService {
	return &TimelineService{memos: memos, hydrator: hydrator{tags: tags, attachments: attachments}}
}

func (s *TimelineService) List(ctx context.Context, q repository.TimelineQuery) ([]domain.Memo, *repository.TimelineCursor, error) {
	items, next, err := s.memos.ListTimeline(ctx, q)
	if err != nil {
		return nil, nil, err
	}
	if err := s.hydrator.hydrate(ctx, items); err != nil {
		return nil, nil, err
	}
	return items, next, nil
}
