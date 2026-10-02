package service

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/NguyenHien-8/NoteHub/internal/domain"
	"github.com/NguyenHien-8/NoteHub/internal/repository"
)

type Options struct {
	MaxMemoBytes       int
	MaxAttachmentBytes int64
}

func (o Options) withDefaults() Options {
	if o.MaxMemoBytes <= 0 {
		o.MaxMemoBytes = 2 << 20
	}
	if o.MaxAttachmentBytes <= 0 {
		o.MaxAttachmentBytes = 1 << 30
	}
	return o
}

func validateContent(content string, maxBytes int) error {
	if !utf8.ValidString(content) {
		return fmt.Errorf("%w: memo content is not valid UTF-8", domain.ErrInvalid)
	}
	if strings.IndexByte(content, 0) >= 0 {
		return fmt.Errorf("%w: memo content contains NUL", domain.ErrInvalid)
	}
	if len(content) > maxBytes {
		return fmt.Errorf("%w: memo content exceeds %d bytes", domain.ErrInvalid, maxBytes)
	}
	return nil
}

type hydrator struct {
	tags        repository.TagRepository
	attachments repository.AttachmentRepository
}

func (h hydrator) hydrate(ctx context.Context, memos []domain.Memo) error {
	if len(memos) == 0 {
		return nil
	}
	ids := make([]int64, len(memos))
	for i := range memos {
		ids[i] = memos[i].ID
	}
	tags, err := h.tags.TagsByMemoIDs(ctx, ids)
	if err != nil {
		return err
	}
	attachments, err := h.attachments.ListAttachmentsByMemoIDs(ctx, ids)
	if err != nil {
		return err
	}
	for i := range memos {
		memos[i].Tags = tags[memos[i].ID]
		memos[i].Attachments = attachments[memos[i].ID]
	}
	return nil
}

func (h hydrator) one(ctx context.Context, memo *domain.Memo) error {
	if memo == nil {
		return domain.ErrNotFound
	}
	items := []domain.Memo{*memo}
	if err := h.hydrate(ctx, items); err != nil {
		return err
	}
	*memo = items[0]
	return nil
}
