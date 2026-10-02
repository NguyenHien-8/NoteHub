package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
	"github.com/NguyenHien-8/NoteHub/internal/ui/components"
	"testing"
	"time"
)

func TestCalendarFitsActualApplicationRail(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	a.Settings().SetTheme(NewTheme("Light"))
	c := components.NewCalendar(nil, nil)
	c.SetMonth(2026, time.October, []domain.CalendarDay{{Date: "2026-10-02", Count: 120}}, "2026-10-02")
	if size := c.Object.MinSize(); size.Width > 256 || size.Height > 390 {
		t.Fatalf("calendar %v exceeds inner right rail (256 wide, 390 high)", size)
	}
}

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
