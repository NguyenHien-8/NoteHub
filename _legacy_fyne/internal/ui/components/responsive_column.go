package components

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

// NewResponsiveVBox propagates the available width before measuring each row.
// Galleries and wrapped text can then report their correct height in the same
// layout pass, without painting into the next note after the viewport shrinks.
func NewResponsiveVBox(gap float32, objects ...fyne.CanvasObject) *fyne.Container {
	return container.New(responsiveColumn{gap: gap}, objects...)
}

type responsiveColumn struct{ gap float32 }

func (l responsiveColumn) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	y := float32(0)
	for _, child := range objects {
		if !child.Visible() {
			continue
		}
		child.Resize(fyne.NewSize(size.Width, child.Size().Height))
		height := child.MinSize().Height
		child.Move(fyne.NewPos(0, y))
		child.Resize(fyne.NewSize(size.Width, height))
		y += height + l.gap
	}
}

func (l responsiveColumn) MinSize(objects []fyne.CanvasObject) fyne.Size {
	size := fyne.Size{}
	count := 0
	for _, child := range objects {
		if !child.Visible() {
			continue
		}
		min := child.MinSize()
		size.Width = max(size.Width, min.Width)
		size.Height += min.Height
		count++
	}
	size.Height += float32(max(0, count-1)) * l.gap
	return size
}
