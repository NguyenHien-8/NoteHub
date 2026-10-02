package components

import (
	"image"
	"reflect"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
)

func galleryAttachments() []domain.Attachment {
	return []domain.Attachment{
		{ID: 11, Filename: "first.png", MIMEType: "image/png"},
		{ID: 21, Filename: "notes.pdf", MIMEType: "application/pdf"},
		{ID: 12, Filename: "second.jpg", MIMEType: "image/jpeg"},
		{ID: 13, Filename: "third.gif", MIMEType: "image/gif"},
		{ID: 22, Filename: "data.txt", MIMEType: "text/plain"},
	}
}

func TestGalleryShowsEveryImageAndAttachmentKeyedThumbnail(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	thumbnail := image.NewRGBA(image.Rect(0, 0, 40, 90))
	var opened int64
	g := NewImageGallery(galleryAttachments(), map[int64]image.Image{12: thumbnail}, func(a domain.Attachment) { opened = a.ID }, nil)
	g.Resize(fyne.NewSize(360, 310))
	if len(g.tiles) != 3 {
		t.Fatalf("image tiles = %d; unavailable thumbnails must remain visible", len(g.tiles))
	}
	if g.tiles[1].preview.Image != thumbnail || g.tiles[1].preview.FillMode != canvas.ImageFillContain {
		t.Fatal("thumbnail was not selected by attachment ID with preserved aspect ratio")
	}
	for index, name := range []string{"first.png", "second.jpg", "third.gif"} {
		if g.tiles[index].name.Text != name {
			t.Fatalf("tile %d filename = %q", index, g.tiles[index].name.Text)
		}
		if _, draggable := g.tiles[index].object.(fyne.Draggable); draggable {
			t.Fatal("nil reorder should not intercept dragging in editor galleries")
		}
	}
	test.Tap(g.tiles[1])
	if opened != 12 {
		t.Fatalf("tap opened attachment %d", opened)
	}
	if g.tiles[1].Position().Y != 0 || g.tiles[2].Position().Y == 0 || g.MinSize().Width > 360 {
		t.Fatal("gallery did not fit two compact tiles in a narrow column")
	}
}

func TestGalleryDragCommitsOnlyOnDropAndPreservesNonImageSlots(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	var got []int64
	var opens int
	attachments := galleryAttachments()
	g := NewImageGallery(attachments, nil, func(domain.Attachment) { opens++ }, func(ids []int64) { got = ids })
	g.Resize(fyne.NewSize(360, 310))
	first := g.tiles[0]
	drag := first.object.(fyne.Draggable)
	first.MouseDown(&desktop.MouseEvent{Button: desktop.MouseButtonPrimary})
	drag.Dragged(&fyne.DragEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(40, 185)}, Dragged: fyne.NewDelta(0, 180)})
	if got != nil || g.tiles[0] != first {
		t.Fatal("drag persisted or moved attachment order before drop")
	}
	if !first.dragging || !g.tiles[2].dropTarget {
		t.Fatal("drag did not show source and destination feedback")
	}
	drag.DragEnd()
	if !reflect.DeepEqual(got, []int64{12, 21, 13, 11, 22}) {
		t.Fatalf("dropped order = %v", got)
	}
	if first.dragging || g.tiles[0].dropTarget || attachments[0].ID != 11 {
		t.Fatal("drag feedback or input alias survived commit")
	}
	test.Tap(first)
	if opens != 0 {
		t.Fatal("drag completion opened the image")
	}
	first.MouseDown(&desktop.MouseEvent{Button: desktop.MouseButtonPrimary})
	test.Tap(first)
	if opens != 1 {
		t.Fatal("a subsequent click did not open the image")
	}
}

func TestGalleryRejectsNoOpOutsideAndSecondaryDrags(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	for _, scenario := range []struct {
		name string
		button desktop.MouseButton
		position fyne.Position
	}{
		{"same tile", desktop.MouseButtonPrimary, fyne.NewPos(40, 40)},
		{"outside left", desktop.MouseButtonPrimary, fyne.NewPos(-1, 185)},
		{"outside bottom", desktop.MouseButtonPrimary, fyne.NewPos(40, 311)},
		{"right button", desktop.MouseButtonSecondary, fyne.NewPos(40, 185)},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			calls := 0
			g := NewImageGallery(galleryAttachments(), nil, nil, func([]int64) { calls++ })
			g.Resize(fyne.NewSize(360, 310))
			first := g.tiles[0]
			first.MouseDown(&desktop.MouseEvent{Button: scenario.button})
			drag := first.object.(fyne.Draggable)
			drag.Dragged(&fyne.DragEvent{PointEvent: fyne.PointEvent{Position: scenario.position}, Dragged: fyne.NewDelta(0, 180)})
			drag.DragEnd()
			if calls != 0 || g.tiles[0].attachment.ID != 11 {
				t.Fatal("invalid or unchanged drop emitted a reorder")
			}
		})
	}
}

func TestGalleryDropAtLastTileBoundaryAndBackwards(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	var got []int64
	g := NewImageGallery(galleryAttachments(), nil, nil, func(ids []int64) { got = ids })
	g.Resize(fyne.NewSize(360, 310))
	last := g.tiles[2]
	last.MouseDown(&desktop.MouseEvent{Button: desktop.MouseButtonPrimary})
	drag := last.object.(fyne.Draggable)
	drag.Dragged(&fyne.DragEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(0, -last.Position().Y)}, Dragged: fyne.NewDelta(0, -150)})
	drag.DragEnd()
	if !reflect.DeepEqual(got, []int64{13, 21, 11, 12, 22}) {
		t.Fatalf("backward boundary drop = %v", got)
	}
}
