//go:build !teststub

package dialogs

import (
	"context"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/NguyenHien-8/NoteHub/internal/app"
	"github.com/NguyenHien-8/NoteHub/internal/ui/work"
)

func buttonIn(object fyne.CanvasObject, text string) *widget.Button {
	if b, ok := object.(*widget.Button); ok && b.Text == text {
		return b
	}
	var children []fyne.CanvasObject
	if c, ok := object.(*fyne.Container); ok {
		children = c.Objects
	}
	if w, ok := object.(fyne.Widget); ok {
		children = test.WidgetRenderer(w).Objects()
	}
	for _, child := range children {
		if found := buttonIn(child, text); found != nil {
			return found
		}
	}
	return nil
}

func TestReloadPreventsEditsUntilLatestRevisionArrives(t *testing.T) {
	application := test.NewApp()
	defer application.Quit()
	backend, err := app.OpenBackend(context.Background(), app.Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	note, err := backend.Memos.Create(context.Background(), "Original")
	if err != nil {
		t.Fatal(err)
	}
	queue := make(chan func(), 16)
	jobs := work.NewWithDispatcher(func(f func()) { queue <- f })
	defer func() { jobs.Stop(); jobs.Wait() }()
	w := application.NewWindow("test")
	defer w.Close()
	w.Resize(fyne.NewSize(1000, 700))
	m := New(backend, w, jobs, func() {})
	next := func() {
		select {
		case f := <-queue:
			f()
		case <-time.After(5 * time.Second):
			t.Fatal("missing dialog callback")
		}
	}
	m.EditMemo(*note)
	next()
	var entry *widget.Entry
	for e := range m.drafts {
		entry = e
	}
	if entry == nil {
		t.Fatal("editor not opened")
	}
	entry.SetText("My unsaved draft")
	if _, err := backend.Memos.Update(context.Background(), note.ID, note.Revision, "Latest saved version"); err != nil {
		t.Fatal(err)
	}
	save := buttonIn(w.Canvas().Overlays().Top(), "Save")
	if save == nil {
		t.Fatal("missing save action")
	}
	test.Tap(save)
	next()
	if entry.Text != "My unsaved draft" || !m.Dirty() || entry.Disabled() {
		t.Fatal("revision conflict discarded or locked the user's unsaved draft")
	}
	stored, err := backend.Memos.Get(context.Background(), note.ID)
	if err != nil || stored.Content != "Latest saved version" {
		t.Fatal("stale edit overwrote the database")
	}
	reload := buttonIn(w.Canvas().Overlays().Top(), "Reload latest")
	if reload == nil {
		t.Fatal("missing reload action")
	}
	test.Tap(reload)
	confirm := buttonIn(w.Canvas().Overlays().Top(), "Yes")
	if confirm == nil {
		t.Fatal("missing confirmation")
	}
	test.Tap(confirm)
	if !entry.Disabled() {
		t.Fatal("editor accepts new text while reload can overwrite it")
	}
	cancel := buttonIn(w.Canvas().Overlays().Top(), "Cancel")
	if cancel == nil || !cancel.Disabled() {
		t.Fatal("editor can close while reload is pending")
	}
	next()
	if entry.Disabled() || entry.Text != "Latest saved version" || m.Dirty() {
		t.Fatalf("reload did not restore a clean editable latest note: %q", entry.Text)
	}
	test.Tap(cancel)
	if len(m.drafts) != 0 {
		t.Fatal("closed editor left a hidden draft")
	}
}

func TestShareCompletionRefreshesCountsAfterDialogClosed(t *testing.T) {
	application := test.NewApp()
	defer application.Quit()
	backend, err := app.OpenBackend(context.Background(), app.Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	note, err := backend.Memos.Create(context.Background(), "Shared note")
	if err != nil {
		t.Fatal(err)
	}
	queue := make(chan func(), 16)
	jobs := work.NewWithDispatcher(func(f func()) { queue <- f })
	defer func() { jobs.Stop(); jobs.Wait() }()
	w := application.NewWindow("test")
	defer w.Close()
	w.Resize(fyne.NewSize(1000, 700))
	changes := 0
	m := New(backend, w, jobs, func() { changes++ })
	next := func() {
		select {
		case f := <-queue:
			f()
		case <-time.After(5 * time.Second):
			t.Fatal("missing share callback")
		}
	}
	m.ShareMemo(*note)
	next()
	create := buttonIn(w.Canvas().Overlays().Top(), "Create share")
	closeButton := buttonIn(w.Canvas().Overlays().Top(), "Close")
	if create == nil || closeButton == nil {
		t.Fatal("missing share dialog actions")
	}
	test.Tap(create)
	test.Tap(closeButton)
	next()
	counts, err := backend.Timeline.Counts(context.Background())
	if err != nil || counts.Shared != 1 || changes != 1 {
		t.Fatalf("share completion lost refresh: counts=%+v changes=%d err=%v", counts, changes, err)
	}
}
