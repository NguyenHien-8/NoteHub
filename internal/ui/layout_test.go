package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"testing"
)

func TestRailsCollapseWithoutShrinkingMainContent(t *testing.T) {
	left, main, right := canvas.NewRectangle(nil), canvas.NewRectangle(nil), canvas.NewRectangle(nil)
	objects := []fyne.CanvasObject{left, main, right}
	layout := railLayout{}
	layout.Layout(objects, fyne.NewSize(1450, 800))
	if !right.Visible() || left.Size().Width != 250 || right.Size().Width != 280 || main.Size().Width < 800 {
		t.Fatalf("wide %v %v %v", left.Size(), main.Size(), right.Size())
	}
	layout.Layout(objects, fyne.NewSize(1100, 628))
	if right.Visible() || main.Size().Width < 800 {
		t.Fatalf("narrow main %v, rail %v", main.Size(), right.Visible())
	}
	layout.Layout(objects, fyne.NewSize(1450, 800))
	if !right.Visible() {
		t.Fatal("right rail did not return")
	}
}
