package screens

import (
	"context"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"strings"
	"sync"

	"github.com/NguyenHien-8/NoteHub/internal/domain"
	"github.com/NguyenHien-8/NoteHub/internal/service"
	"golang.org/x/image/draw"
)

// Previews bounds both decoded image dimensions and retained thumbnail memory.
// All decoding happens inside a screen's worker, never during widget rendering.
type Previews struct {
	attachments *service.AttachmentService
	mu          sync.Mutex
	cache       map[string]image.Image
}

func NewPreviews(s *service.AttachmentService) *Previews {
	return &Previews{attachments: s, cache: make(map[string]image.Image)}
}

func (p *Previews) Image(ctx context.Context, a domain.Attachment) image.Image {
	if !strings.HasPrefix(a.MIMEType, "image/") || ctx.Err() != nil {
		return nil
	}
	key := a.UID + ":" + a.SHA256
	p.mu.Lock()
	cached, ok := p.cache[key]
	p.mu.Unlock()
	if ok {
		return cached
	}
	f, _, err := p.attachments.Open(ctx, a.ID)
	if err != nil {
		return nil
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 12_000_000 {
		return nil
	}
	if _, err = f.Seek(0, 0); err != nil {
		return nil
	}
	src, _, err := image.Decode(f)
	if err != nil || ctx.Err() != nil {
		return nil
	}
	width, height := 300, 200
	if cfg.Width*height > cfg.Height*width {
		height = max(1, cfg.Height*width/cfg.Width)
	} else {
		width = max(1, cfg.Width*height/cfg.Height)
	}
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
	p.mu.Lock()
	if len(p.cache) >= 120 {
		clear(p.cache)
	}
	p.cache[key] = dst
	p.mu.Unlock()
	return dst
}

func (p *Previews) Memos(ctx context.Context, memos []domain.Memo) map[int64]image.Image {
	out := make(map[int64]image.Image)
	for _, m := range memos {
		if ctx.Err() != nil {
			break
		}
		for _, a := range m.Attachments {
			if img := p.Image(ctx, a); img != nil {
				out[m.ID] = img
				break
			}
		}
	}
	return out
}
