package service

import (
	"context"

	"github.com/NguyenHien-8/NoteHub/internal/domain"
	"github.com/NguyenHien-8/NoteHub/internal/repository"
)

type SearchService struct {
	memos    repository.MemoRepository
	hydrator hydrator
}

func NewSearchService(memos repository.MemoRepository, tags repository.TagRepository, attachments repository.AttachmentRepository) *SearchService {
	return &SearchService{memos: memos, hydrator: hydrator{tags: tags, attachments: attachments}}
}

func (s *SearchService) Search(ctx context.Context, q repository.SearchQuery) ([]domain.Memo, error) {
	items, err := s.memos.SearchMemos(ctx, q)
	if err != nil {
		return nil, err
	}
	if err := s.hydrator.hydrate(ctx, items); err != nil {
		return nil, err
	}
	return items, nil
}
