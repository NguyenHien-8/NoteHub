package service

import (
	"context"

	"github.com/NguyenHien-8/NoteHub/internal/domain"
	"github.com/NguyenHien-8/NoteHub/internal/repository"
)

type TagService struct{ tags repository.TagRepository }

func NewTagService(tags repository.TagRepository) *TagService { return &TagService{tags: tags} }
func (s *TagService) List(ctx context.Context) ([]domain.TagCount, error) {
	return s.tags.ListTagCounts(ctx)
}
