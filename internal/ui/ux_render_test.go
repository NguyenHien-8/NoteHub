//go:build !teststub

package ui

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/NguyenHien-8/NoteHub/internal/app"
	"github.com/NguyenHien-8/NoteHub/internal/ui/work"
)

// Opt-in software-rendered fixtures exercise the actual desktop and database
// using a temporary profile. They never open or modify the user's notes.
func TestDesktopResponsiveGalleryRendering(t *testing.T) {
	output := os.Getenv("NOTEHUB_TEST_SCREENSHOTS")
	if output == "" {
		t.Skip("set NOTEHUB_TEST_SCREENSHOTS to export visual fixtures")
	}
	application := test.NewApp()
	defer application.Quit()
	backend, err := app.OpenBackend(context.Background(), app.Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	note, err := backend.Memos.Create(context.Background(), "Bộ sưu tập ý tưởng\nẢnh dọc, ngang và vuông tự sắp xếp theo chiều rộng. #Project/NoteHub")
	if err != nil {
		t.Fatal(err)
	}
	for i, dimensions := range []image.Point{{180, 300}, {480, 270}, {320, 320}, {400, 280}, {220, 350}, {480, 250}, {360, 280}} {
		img := image.NewRGBA(image.Rect(0, 0, dimensions.X, dimensions.Y))
		for y := 0; y < dimensions.Y; y++ {
			for x := 0; x < dimensions.X; x++ {
				shade := color.RGBA{R: uint8(40 + i*20), G: uint8(130 + x*80/dimensions.X), B: uint8(220 - y*70/dimensions.Y), A: 255}
				if x%60 < 2 || y%60 < 2 {
					shade = color.RGBA{R: 215, G: 235, B: 245, A: 255}
				}
				img.SetRGBA(x, y, shade)
			}
		}
		var data bytes.Buffer
		if err := png.Encode(&data, img); err != nil {
			t.Fatal(err)
		}
		if _, err := backend.Attachments.Add(context.Background(), note.ID, fmt.Sprintf("Study %02d.png", i+1), "image/png", &data); err != nil {
			t.Fatal(err)
		}
	}
	queue := make(chan func(), 256)
	jobs := work.NewWithDispatcher(func(f func()) { queue <- f })
	d := newWindow(application, backend, "test", jobs)
	d.Show()
	defer func() { d.Window.SetCloseIntercept(nil); d.Window.Close(); _ = d.Wait() }()
	flush := func() {
		deadline := time.After(10 * time.Second)
		for jobs.Busy() || len(queue) > 0 {
			select {
			case f := <-queue:
				f()
			case <-time.After(time.Millisecond):
			case <-deadline:
				t.Fatal("UI render jobs timeout")
			}
		}
	}
	flush()
	if err := os.MkdirAll(output, 0o755); err != nil {
		t.Fatal(err)
	}
	capture := func(name string) {
		t.Helper()
		file, err := os.Create(filepath.Join(output, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		if err := png.Encode(file, d.Canvas().Capture()); err != nil {
			t.Fatal(err)
		}
	}
	d.Resize(fyne.NewSize(1450, 900))
	capture("ux-gallery-wide")
	d.rails.ToggleLeft()
	capture("ux-gallery-compact")
	d.Resize(fyne.NewSize(820, 660))
	capture("ux-gallery-narrow")
	d.SetAppearance("Dark")
	capture("ux-gallery-dark")
	d.SetAppearance("Light")
	d.Resize(fyne.NewSize(1200, 820))
	d.manager.EditMemo(*note)
	flush()
	capture("ux-edit-note")
}
