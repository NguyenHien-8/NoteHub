package components

import (
	"image/color"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
)

func TestComposerKeepsStagingUntilSuccessAndCopiesPaths(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	var calls int
	var saved []string
	c := NewComposer(func(content string, paths []string) {
		calls++
		saved = paths
		if content != "Ghi chú" {
			t.Errorf("content = %q", content)
		}
	}, nil, nil)
	test.Tap(c.save)
	if calls != 0 {
		t.Fatal("blank memo submitted")
	}
	c.Entry.SetText("Ghi chú")
	c.AddPaths([]string{"/tmp/a.txt", "/tmp/a.txt", "/tmp/b.pdf"})
	c.SetBusy(true)
	test.Tap(c.save)
	if calls != 0 {
		t.Fatal("busy composer submitted")
	}
	c.SetBusy(false)
	test.Tap(c.save)
	if calls != 1 || len(saved) != 2 || c.Entry.Text != "Ghi chú" || len(c.Paths()) != 2 {
		t.Fatal("failed save lost staged content")
	}
	saved[0] = "changed"
	if c.Paths()[0] != "/tmp/a.txt" {
		t.Fatal("callback aliases staging")
	}
	paths := c.Paths()
	paths[0] = "changed"
	if c.Paths()[0] != "/tmp/a.txt" {
		t.Fatal("Paths aliases staging")
	}
	c.Reset()
	if c.Entry.Text != "" || len(c.Paths()) != 0 {
		t.Fatal("successful save reset failed")
	}
}

func TestComposerAllowsAttachmentOnlyNote(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	var submitted bool
	c := NewComposer(func(content string, paths []string) { submitted = content == "" && len(paths) == 1 }, nil, nil)
	c.AddPaths([]string{"/tmp/photo.png"})
	test.Tap(c.save)
	if !submitted {
		t.Fatal("attachment-only note was blocked")
	}
}

func TestSidebarMenuOrderSelectionAndNavigation(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	var navigated string
	s := NewSidebar(func(id string) { navigated = id }, nil, nil)
	for index, name := range []string{"Home", "Calendar", "Search", "Attachments", "Tags", "Settings"} {
		id := strings.ToLower(name)
		if s.buttons[id].Text != name {
			t.Fatalf("menu %d = %q", index, s.buttons[id].Text)
		}
		test.Tap(s.buttons[id])
		if navigated != id {
			t.Fatalf("menu dispatched %q", navigated)
		}
	}
	s.SetSelected("Calendar")
	if s.buttons["calendar"].Importance != widget.HighImportance || s.buttons["home"].Importance != widget.LowImportance {
		t.Fatal("selection did not follow navigation")
	}
	s.SetTags([]domain.TagCount{{Tag: "Research/FPGA", Count: 12}})
	rows := s.tags.Content.(*fyne.Container)
	if len(rows.Objects) != 1 {
		t.Fatal("real tags were not displayed")
	}
}

func TestTimelineRequestsNextPageOnceUntilOwnerResponds(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	timeline := NewTimelineList()
	var calls int
	timeline.SetMore(true, func() { calls++ })
	test.Tap(timeline.more)
	test.Tap(timeline.more)
	if calls != 1 {
		t.Fatalf("duplicate paging requests = %d", calls)
	}
	timeline.SetMore(true, func() { calls++ })
	test.Tap(timeline.more)
	if calls != 2 {
		t.Fatal("next page never became available")
	}
	timeline.SetMore(false, func() { calls++ })
	test.Tap(timeline.more)
	if calls != 2 {
		t.Fatal("exhausted timeline requested another page")
	}
}

func TestCalendarFitsRightRailAndSurfaceFollowsTheme(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	test.ApplyTheme(t, theme.LightTheme())
	c := NewCalendar(nil, nil)
	c.SetMonth(2026, time.September, nil, "")
	if width := c.Object.MinSize().Width; width > 280 {
		t.Fatalf("calendar minimum width %.0f exceeds the right rail", width)
	}
	s := Surface(widget.NewLabel("theme")).(*surface)
	r := test.WidgetRenderer(s).(*surfaceRenderer)
	if r.bg.FillColor != color.White {
		t.Fatal("light card is not white")
	}
	test.ApplyTheme(t, theme.DarkTheme())
	s.Refresh()
	if r.bg.FillColor == color.White {
		t.Fatal("dark card stayed white")
	}
}

func TestAttachmentActionsDispatchFromVisibleButtons(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	var called string
	card := NewAttachmentCard(domain.Attachment{Filename: "研究.pdf", MIMEType: "application/pdf", Size: 4096}, nil, func() { called = "open" }, func() { called = "reveal" }, func() { called = "delete" }, func() { called = "memo" })
	buttons := map[string]*widget.Button{}
	var visit func(fyne.CanvasObject)
	visit = func(object fyne.CanvasObject) {
		switch value := object.(type) {
		case *widget.Button:
			buttons[value.Text] = value
		case *fyne.Container:
			for _, child := range value.Objects {
				visit(child)
			}
		case *surface:
			visit(value.content)
		case *container.ThemeOverride:
			visit(value.Content)
		}
	}
	visit(card)
	for _, check := range []struct{ label, want string }{{"Open", "open"}, {"Reveal", "reveal"}, {"Delete", "delete"}, {"Go to Memo", "memo"}} {
		button := buttons[check.label]
		if button == nil {
			t.Fatalf("missing action %s", check.label)
		}
		test.Tap(button)
		if called != check.want {
			t.Fatalf("%s dispatched %q", check.label, called)
		}
	}
}

func TestComposerTagInsertionAndAttachmentRemoval(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	c := NewComposer(nil, nil, nil)
	c.Entry.SetText("Đang nghiên cứu")
	c.InsertTag("#Research/FPGA")
	if !strings.Contains(c.Entry.Text, "#Research/FPGA") || strings.Contains(c.Entry.Text, "##") {
		t.Fatalf("tag = %q", c.Entry.Text)
	}
	c.AddPaths([]string{"/tmp/a.txt"})
	row := c.staged.Objects[0].(*fyne.Container)
	test.Tap(row.Objects[1].(*widget.Button))
	if len(c.Paths()) != 0 {
		t.Fatal("attachment was not removed")
	}
}

func TestMemoMenuAndCalendarDispatchRealValues(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	memo := domain.Memo{ID: 42, Content: "Original content", Revision: 7}
	var got domain.Memo
	menu := memoMenu(memo, MemoActions{Edit: func(m domain.Memo) { got = m }})
	menu.Items[0].Action()
	if got.ID != 42 || got.Revision != 7 || got.Content != memo.Content {
		t.Fatalf("action lost memo: %#v", got)
	}
	var date string
	c := NewCalendar(func(value string) { date = value }, nil)
	c.SetMonth(2026, time.March, []domain.CalendarDay{{Date: "2026-03-01", Count: 2}}, "2026-03-01")
	test.Tap(c.dayButtons[1])
	if date != "2026-03-01" {
		t.Fatalf("calendar callback = %q", date)
	}
}
