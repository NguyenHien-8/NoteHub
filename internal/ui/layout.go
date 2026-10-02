package ui

import "fyne.io/fyne/v2"

// railLayout reserves stable navigation widths and releases the right rail on
// smaller windows. Its minimum is independent of long note/file contents.
type railLayout struct{}

func (railLayout) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.NewSize(1100, 628) }
func (railLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 3 {
		return
	}
	left, main, right := objects[0], objects[1], objects[2]
	left.Move(fyne.NewPos(0, 0))
	left.Resize(fyne.NewSize(250, size.Height))
	width := size.Width - 250
	if size.Width >= 1280 {
		if !right.Visible() {
			right.Show()
		}
		right.Move(fyne.NewPos(size.Width-280, 0))
		right.Resize(fyne.NewSize(280, size.Height))
		width -= 280
	} else if right.Visible() {
		right.Hide()
	}
	main.Move(fyne.NewPos(250, 0))
	main.Resize(fyne.NewSize(width, size.Height))
}

type insetLayout struct{ padding float32 }

func (l insetLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	size := fyne.NewSize(0, 0)
	for _, o := range objects {
		if o.Visible() {
			size = size.Max(o.MinSize())
		}
	}
	return size.Add(fyne.NewSize(l.padding*2, l.padding*2))
}
func (l insetLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objects {
		if o.Visible() {
			o.Move(fyne.NewPos(l.padding, l.padding))
			o.Resize(fyne.NewSize(max(0, size.Width-2*l.padding), max(0, size.Height-2*l.padding)))
		}
	}
}
