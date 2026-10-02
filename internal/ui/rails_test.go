package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
)

func newTestRails(t *testing.T) (*Rails, *canvas.Rectangle, *canvas.Rectangle, *canvas.Rectangle, fyne.Preferences) {
	t.Helper()
	a := test.NewApp()
	t.Cleanup(a.Quit)
	left, main, right := canvas.NewRectangle(nil), canvas.NewRectangle(nil), canvas.NewRectangle(nil)
	return NewRails(left, main, right, a.Preferences()), left, main, right, a.Preferences()
}

func TestRailsDragBothHandlesPreservesCenterAndSavesAtEnd(t *testing.T) {
	rails, left, main, right, prefs := newTestRails(t)
	rails.Resize(fyne.NewSize(1200, 600))
	renderer := test.WidgetRenderer(rails)
	leftHandle := renderer.Objects()[1].(fyne.Draggable)
	rightHandle := renderer.Objects()[3].(fyne.Draggable)
	if renderer.Objects()[1].(desktop.Cursorable).Cursor() != desktop.HResizeCursor {
		t.Fatal("left divider must communicate horizontal resizing")
	}
	leftHandle.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(80, 0)})
	if left.Size().Width != 330 || main.Size().Width < 400 {
		t.Fatalf("left drag: left=%v center=%v", left.Size(), main.Size())
	}
	if got := prefs.FloatWithFallback("layout.left.width", 250); got != 250 {
		t.Fatalf("drag persisted before release: %v", got)
	}
	leftHandle.DragEnd()
	if got := prefs.Float("layout.left.width"); got != 330 {
		t.Fatalf("released width not persisted: %v", got)
	}
	rightHandle.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(-2000, 0)})
	rightHandle.DragEnd()
	if main.Size().Width < 400 || right.Size().Width > 458 || !right.Visible() {
		t.Fatalf("right drag consumed center or hid itself: main=%v right=%v visible=%v", main.Size(), right.Size(), right.Visible())
	}
	rightHandle.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(2000, 0)})
	rightHandle.DragEnd()
	if right.Size().Width < 272 || main.Size().Width < 400 {
		t.Fatalf("right rail dragged below useful minimum: %v", right.Size())
	}
}

func TestRailsCollapseAndNarrowLayoutRememberUserWidths(t *testing.T) {
	rails, left, main, right, prefs := newTestRails(t)
	rails.Resize(fyne.NewSize(1300, 600))
	handle := test.WidgetRenderer(rails).Objects()[1].(fyne.Draggable)
	handle.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(80, 0)})
	handle.DragEnd()
	rails.Resize(fyne.NewSize(760, 410))
	if right.Visible() || !left.Visible() || main.Size().Width < 400 {
		t.Fatalf("narrow layout: left=%v main=%v right=%v", left.Visible(), main.Size(), right.Visible())
	}
	if rails.RightCollapsed() || prefs.Bool("layout.right.collapsed") {
		t.Fatal("automatic suppression changed manual collapse choice")
	}
	rails.ToggleRight()
	if !right.Visible() || left.Visible() || main.Size().Width < 400 {
		t.Fatal("right toggle must expose the automatically hidden panel at narrow sizes")
	}
	rails.Resize(fyne.NewSize(1300, 600))
	if !left.Visible() || !right.Visible() || left.Size().Width != 330 {
		t.Fatal("wide layout failed to restore preferred widths and panels")
	}
	rails.ToggleLeft()
	rails.ToggleRight()
	if left.Visible() || right.Visible() || main.Size().Width != 1300 {
		t.Fatal("collapsed rails did not release their entire width")
	}
	reopened := NewRails(canvas.NewRectangle(nil), canvas.NewRectangle(nil), canvas.NewRectangle(nil), prefs)
	if !reopened.LeftCollapsed() || !reopened.RightCollapsed() {
		t.Fatal("manual collapse state did not survive reconstruction")
	}
	rails.ToggleLeft()
	if left.Size().Width != 330 {
		t.Fatal("expansion lost saved width")
	}
}

func TestRailsMinSizeIgnoresOversizedChildren(t *testing.T) {
	rails, left, main, right, _ := newTestRails(t)
	for _, child := range []*canvas.Rectangle{left, main, right} {
		child.SetMinSize(fyne.NewSize(3000, 2000))
	}
	if min := rails.MinSize(); min.Width > 760 || min.Height > 340 {
		t.Fatalf("content forced oversized window: %v", min)
	}
	rails.Resize(fyne.NewSize(760, 410))
	if main.Position().X+main.Size().Width > 760 || main.Size().Width < 400 {
		t.Fatalf("center exceeds viewport: pos=%v size=%v", main.Position(), main.Size())
	}
}
