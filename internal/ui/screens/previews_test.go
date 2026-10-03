package screens

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/NguyenHien-8/NoteHub/internal/app"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
)

func TestMemoPreviewsIncludeEveryImageByAttachmentID(t *testing.T) {
	ctx := context.Background()
	b, err := app.OpenBackend(ctx, app.Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	m, err := b.Memos.Create(ctx, "Gallery")
	if err != nil {
		t.Fatal(err)
	}
	var data bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 20, 10))
	img.Set(0, 0, color.White)
	if err := png.Encode(&data, img); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"first.png", "second.png", "third.png"} {
		a, err := b.Attachments.Add(ctx, m.ID, name, "image/png", bytes.NewReader(data.Bytes()))
		if err != nil {
			t.Fatal(err)
		}
		m.Attachments = append(m.Attachments, *a)
	}
	previews := NewPreviews(b.Attachments)
	got := previews.Memos(ctx, []domain.Memo{*m})
	if len(got) != 3 {
		t.Fatalf("only %d of 3 images have previews", len(got))
	}
	for _, a := range m.Attachments {
		if got[a.ID] == nil {
			t.Fatalf("missing image %d", a.ID)
		}
	}
}
