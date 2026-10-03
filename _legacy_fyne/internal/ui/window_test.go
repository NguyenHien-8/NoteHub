//go:build !teststub

package ui

import (
	"context"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/NguyenHien-8/NoteHub/internal/app"
	"github.com/NguyenHien-8/NoteHub/internal/repository"
	"github.com/NguyenHien-8/NoteHub/internal/ui/dialogs"
	"github.com/NguyenHien-8/NoteHub/internal/ui/screens"
	"github.com/NguyenHien-8/NoteHub/internal/ui/work"
)

func TestDesktopWaitCancelsWorkWithoutWindowCloseCallback(t *testing.T) {
	jobs := work.NewWithDispatcher(func(f func()) { f() })
	d := &Desktop{Jobs: jobs, Search: &screens.Search{}, expiryTicker: time.NewTicker(time.Hour), stopTick: make(chan struct{}), manager: dialogs.New(nil, nil, jobs, nil)}
	defer d.expiryTicker.Stop()
	started := make(chan struct{})
	work.Run(jobs, func(ctx context.Context) (struct{}, error) {
		close(started)
		<-ctx.Done()
		return struct{}{}, ctx.Err()
	}, func(struct{}, error) {})
	<-started
	done := make(chan error, 1)
	go func() { done <- d.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		jobs.Stop()
		<-done
		t.Fatal("shutdown waits forever without canceling an open chooser's context")
	}
	if err := d.Wait(); err != nil {
		t.Fatal(err)
	}
}

func walkUI(object fyne.CanvasObject, visit func(fyne.CanvasObject), seen map[fyne.CanvasObject]bool) {
	if object == nil || seen[object] {
		return
	}
	seen[object] = true
	visit(object)
	if c, ok := object.(*fyne.Container); ok {
		for _, child := range c.Objects {
			walkUI(child, visit, seen)
		}
	}
	if w, ok := object.(fyne.Widget); ok {
		for _, child := range test.WidgetRenderer(w).Objects() {
			walkUI(child, visit, seen)
		}
	}
}

func findButton(root fyne.CanvasObject, text string) *widget.Button {
	var result *widget.Button
	walkUI(root, func(o fyne.CanvasObject) {
		if button, ok := o.(*widget.Button); ok && button.Text == text {
			result = button
		}
	}, make(map[fyne.CanvasObject]bool))
	return result
}

func hasLabel(root fyne.CanvasObject, text string) bool {
	found := false
	walkUI(root, func(o fyne.CanvasObject) {
		if label, ok := o.(*widget.Label); ok && label.Text == text {
			found = true
		}
	}, make(map[fyne.CanvasObject]bool))
	return found
}

func TestDesktopCreateFilterNavigateAndReopen(t *testing.T) {
	application := test.NewApp()
	defer application.Quit()
	backend, err := app.OpenBackend(context.Background(), app.Config{DataDir: t.TempDir(), Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	queue := make(chan func(), 256)
	jobs := work.NewWithDispatcher(func(f func()) { queue <- f })
	desktop := newWindow(application, backend, "test", jobs)
	desktop.Show()
	defer func() { desktop.Window.SetCloseIntercept(nil); desktop.Window.Close(); _ = desktop.Wait() }()
	flush := func() {
		timeout := time.NewTimer(10 * time.Second)
		defer timeout.Stop()
		for {
			select {
			case f := <-queue:
				f()
			default:
				if !jobs.Busy() {
					return
				}
			}
			select {
			case f := <-queue:
				f()
			case <-time.After(time.Millisecond):
			case <-timeout.C:
				t.Fatal("UI workers did not settle")
			}
		}
	}
	flush()
	if desktop.manager.SharingURL() != "" {
		t.Fatal("sharing started without consent")
	}
	desktop.Home.Composer.Entry.SetText("Ghi chú FPGA\nNội dung tiếng Việt #Research/FPGA")
	save := findButton(desktop.Home.Composer.Object, "Save")
	if save == nil {
		t.Fatal("no Save action")
	}
	test.Tap(save)
	flush()
	if desktop.Home.Composer.Entry.Text != "" {
		t.Fatal("composer did not clear after success")
	}
	items, _, err := backend.Timeline.List(context.Background(), repository.TimelineQuery{Limit: 50})
	if err != nil || len(items) != 1 {
		t.Fatalf("saved notes %v %v", items, err)
	}
	if !hasLabel(desktop.Home.Object, "Ghi chú FPGA") {
		t.Fatal("new note not shown without restart")
	}
	desktop.manager.Favorite(items[0])
	flush()
	test.Tap(desktop.favorites)
	flush()
	if !hasLabel(desktop.Home.Object, "★ Ghi chú FPGA") {
		t.Fatal("favorite filter lost saved note")
	}
	desktop.showTag("Research/FPGA")
	flush()
	if !hasLabel(desktop.Home.Object, "★ Ghi chú FPGA") {
		t.Fatal("tag filter lost note")
	}
	desktop.Search.Query.SetText("ghi chu")
	desktop.Navigate("search")
	flush()
	if !hasLabel(desktop.Search.Object, "★ Ghi chú FPGA") {
		t.Fatal("accent-insensitive search failed through UI")
	}
	desktop.showDate(items[0].CreatedAt.Local().Format("2006-01-02"))
	flush()
	if !hasLabel(desktop.Home.Object, "★ Ghi chú FPGA") {
		t.Fatal("calendar filter lost note")
	}
	for _, page := range []string{"calendar", "attachments", "tags", "settings", "home"} {
		desktop.Navigate(page)
		flush()
		if desktop.current != page {
			t.Fatal(page)
		}
	}
	desktop.SetAppearance("Dark")
	desktop.Window.Resize(fyne.NewSize(1100, 700))
	flush()
	desktop.SetAppearance("Light")
	desktop.Window.Resize(fyne.NewSize(1450, 900))
	flush()
	// A separate handle reads the committed note and favorite, just as a restart does.
	reopened, err := app.OpenBackend(context.Background(), app.Config{DataDir: backend.Paths.Root})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	note, err := reopened.Memos.Get(context.Background(), items[0].ID)
	if err != nil || !note.Favorite || note.Content != items[0].Content {
		t.Fatalf("reopen lost content: %v %v", note, err)
	}
}

// Screenshot output is opt-in so normal tests never write to the source tree.
func TestDesktopReferenceRendering(t *testing.T) {
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
	for _, content := range []string{
		"Bài tập điều khiển tự động\nThiết kế bộ điều khiển PID và mô phỏng trên MATLAB. #Study",
		"Ý tưởng cải tiến NoteHub\nSQLite lưu trữ cục bộ, tìm kiếm nhanh và sao lưu dữ liệu an toàn. #Project/NoteHub",
		"Tài liệu ESP32-S3 Camera\nTổng hợp tài liệu, sơ đồ chân và các ví dụ sample code. #ESP32",
		"Kiểm tra FPGA Cyclone IV EP4CE10\nĐã nạp code test GPIO, kiểm tra hoạt động của các LED và switch. #FPGA #Research",
	} {
		if _, err := backend.Memos.Create(context.Background(), content); err != nil {
			t.Fatal(err)
		}
	}
	queue := make(chan func(), 256)
	jobs := work.NewWithDispatcher(func(f func()) { queue <- f })
	desktop := newWindow(application, backend, "0.2.0", jobs)
	desktop.Show()
	defer func() { desktop.Window.SetCloseIntercept(nil); desktop.Window.Close(); _ = desktop.Wait() }()
	deadline := time.After(10 * time.Second)
	for jobs.Busy() || len(queue) > 0 {
		select {
		case f := <-queue:
			f()
		case <-time.After(time.Millisecond):
		case <-deadline:
			t.Fatal("render workers timeout")
		}
	}
	if err := os.MkdirAll(output, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"Light", "Dark"} {
		desktop.SetAppearance(mode)
		desktop.Window.Resize(fyne.NewSize(1450, 900))
		f, err := os.Create(filepath.Join(output, "notehub-"+mode+".png"))
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(f, desktop.Window.Canvas().Capture())
		_ = f.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	desktop.SetAppearance("Light")
	desktop.Window.Resize(fyne.NewSize(1100, 700))
	f, err := os.Create(filepath.Join(output, "notehub-narrow.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, desktop.Window.Canvas().Capture()); err != nil {
		t.Fatal(err)
	}
}
