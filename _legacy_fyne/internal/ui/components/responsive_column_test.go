package components

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestResponsiveCardsDoNotOverlapAfterGalleryResize(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	g := NewImageGallery(galleryAttachments(), nil, nil, nil)
	footer := widget.NewLabel("Tags below all images")
	card := Surface(NewResponsiveVBox(7, g, footer))
	next := widget.NewLabel("Next note")
	rows := NewResponsiveVBox(10, card, next)
	for _, width := range []float32{1400, 360, 1000, 480, 1300} {
		rows.Resize(fyne.NewSize(width, 2000))
		if next.Position().Y < card.Size().Height || footer.Position().Y < g.Position().Y+g.MinSize().Height {
			t.Fatalf("width %v: gallery overlapped footer/next card: gallery %v, footer %v, card %v, next %v", width, g.MinSize(), footer.Position(), card.Size(), next.Position())
		}
		last := g.tiles[len(g.tiles)-1]
		if last.Position().Y+last.Size().Height > g.Size().Height+0.01 {
			t.Fatalf("width %v: last row exceeds allocated gallery height", width)
		}
	}
}
