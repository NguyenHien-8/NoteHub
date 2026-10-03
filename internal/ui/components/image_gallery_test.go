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
	g.Resize(fyne.NewSize(360, 650))
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
	if g.tiles[1].Position().Y == 0 || g.MinSize().Width > 360 {
		t.Fatal("narrow gallery should keep images readable in one column")
	}
	g.Resize(fyne.NewSize(1400, 650))
	last := g.tiles[len(g.tiles)-1]
	if g.tiles[0].Position().X != 0 || last.Position().Y != 0 || last.Position().X+last.Size().Width != 1400 {
		t.Fatalf("wide gallery should use its full row: last pos=%v size=%v", last.Position(), last.Size())
	}
}

// A resize must change the row count and minimum height together, otherwise
// the last row paints over the next note or becomes unreachable by scrolling.
func TestGalleryReflowsAndContainsEveryPreviewAfterRepeatedResize(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	attachments := []domain.Attachment{}
	thumbs := map[int64]image.Image{}
	for i := int64(1); i <= 7; i++ {
		attachments = append(attachments, domain.Attachment{ID: i, Filename: "image.png", MIMEType: "image/png"})
		thumbs[i] = image.NewRGBA(image.Rect(0, 0, int(60*i), 200))
	}
	g := NewImageGallery(attachments, thumbs, nil, func([]int64) {})
	for _, width := range []float32{1400, 320, 900, 460, 1600, 300} {
		g.Resize(fyne.NewSize(width, 2000))
		for _, tile := range g.tiles {
			test.WidgetRenderer(tile.object.(fyne.Widget)).Layout(tile.Size())
			p, s := tile.Position(), tile.Size()
			if p.X < 0 || p.X+s.Width > width+0.01 || p.Y+s.Height > g.MinSize().Height+0.01 {
				t.Fatalf("width %v: tile out of gallery bounds: %v %v, min=%v", width, p, s, g.MinSize())
			}
			p, s = tile.preview.Position(), tile.preview.Size()
			if p.X < 0 || p.X+s.Width > tile.Size().Width || p.Y+s.Height > tile.Size().Height-galleryFooterHeight {
				t.Fatalf("width %v: preview escapes tile: %v %v / %v", width, p, s, tile.Size())
			}
		}
	}
}

func TestGalleryDragStartsOnSourceAndFollowsPointerWithoutSaving(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	calls := 0
	g := NewImageGallery(galleryAttachments(), nil, nil, func([]int64) { calls++ })
	g.Resize(fyne.NewSize(800, 1000))
	first := g.tiles[0]
	first.MouseDown(&desktop.MouseEvent{Button: desktop.MouseButtonPrimary})
	drag := first.object.(fyne.Draggable)
	drag.Dragged(&fyne.DragEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(30, 30)}})
	if !first.dragging || !g.ghost.Visible() || first.preview.Translucency == 0 {
		t.Fatal("drag should be visible before crossing into another image")
	}
	initialGhost := g.ghost.Position()
	drag.Dragged(&fyne.DragEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(160, 160)}})
	if g.ghost.Position() == initialGhost {
		t.Fatal("floating image did not follow the pointer")
	}
	if calls != 0 {
		t.Fatal("pointer movement persisted ordering before release")
	}
	drag.DragEnd()
	if first.dragging || g.ghost.Visible() || g.marker.Visible() || first.preview.Translucency != 0 || calls != 0 {
		t.Fatal("same-image drop should reset drag feedback without saving")
	}
}

func TestGalleryDragCommitsOnlyOnDropAndPreservesNonImageSlots(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	var got []int64
	var opens int
	attachments := galleryAttachments()
	g := NewImageGallery(attachments, nil, func(domain.Attachment) { opens++ }, func(ids []int64) { got = ids })
	g.Resize(fyne.NewSize(560, 1000))
	first := g.tiles[0]
	drag := first.object.(fyne.Draggable)
	first.MouseDown(&desktop.MouseEvent{Button: desktop.MouseButtonPrimary})
	targetPoint := fyne.NewPos(40, g.tiles[2].Position().Y+20)
	drag.Dragged(&fyne.DragEvent{PointEvent: fyne.PointEvent{Position: targetPoint}, Dragged: fyne.NewDelta(0, targetPoint.Y)})
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
		name     string
		button   desktop.MouseButton
		position fyne.Position
	}{
		{"same tile", desktop.MouseButtonPrimary, fyne.NewPos(40, 40)},
		{"outside left", desktop.MouseButtonPrimary, fyne.NewPos(-1, 280)},
		{"outside bottom", desktop.MouseButtonPrimary, fyne.NewPos(40, 651)},
		{"right button", desktop.MouseButtonSecondary, fyne.NewPos(40, 280)},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			calls := 0
			g := NewImageGallery(galleryAttachments(), nil, nil, func([]int64) { calls++ })
			g.Resize(fyne.NewSize(560, 650))
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
	g.Resize(fyne.NewSize(560, 1000))
	last := g.tiles[2]
	last.MouseDown(&desktop.MouseEvent{Button: desktop.MouseButtonPrimary})
	drag := last.object.(fyne.Draggable)
	drag.Dragged(&fyne.DragEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(0, -last.Position().Y)}, Dragged: fyne.NewDelta(0, -150)})
	drag.DragEnd()
	if !reflect.DeepEqual(got, []int64{13, 21, 11, 12, 22}) {
		t.Fatalf("backward boundary drop = %v", got)
	}
}
