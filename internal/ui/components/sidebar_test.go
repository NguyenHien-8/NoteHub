package components

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
)

func TestSidebarCompactNavigationPreservesActionsAndSelection(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	var navigated string
	s := NewSidebar(func(page string) { navigated = page }, nil, nil)
	s.SetCompact(true)
	s.Object.Resize(fyne.NewSize(44, 220))
	for _, id := range []string{"home", "calendar", "search", "attachments", "tags", "settings"} {
		button := s.buttons[id]
		if button.Text != "" || button.Icon == nil || !button.Visible() || button.Hint == "" {
			t.Fatalf("compact action %s lost its icon or label", id)
		}
		test.Tap(button)
		if navigated != id { t.Fatalf("%s navigated to %s", id, navigated) }
	}
	s.SetSelected("settings")
	if s.buttons["settings"].Importance != widget.HighImportance { t.Fatal("compact selection is missing") }
	if !s.compactTags.Visible() || s.scroll.Content.MinSize().Height <= s.scroll.Size().Height {
		t.Fatal("short sidebar must scroll all navigation actions including My Tags")
	}
	s.SetCompact(false)
	if s.buttons["settings"].Text != "Settings" || s.buttons["settings"].Importance != widget.HighImportance {
		t.Fatal("expansion lost button labels or selection")
	}
}

func TestCompactMyTagsUsesCurrentTagsAndAddAction(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	var selected string
	added := false
	s := NewSidebar(nil, func(tag string) { selected = tag }, func() { added = true })
	w := test.NewWindow(s.Object)
	defer w.Close()
	w.Resize(fyne.NewSize(640, 400))
	s.SetCompact(true)
	s.SetTags([]domain.TagCount{{Tag: "Research", Count: 3}})
	test.Tap(s.compactTags)
	if s.tagPopup == nil || !s.tagPopup.Visible() { t.Fatal("My Tags did not open") }
	rows := s.popupTags.Objects[0].(*fyne.Container)
	row := rows.Objects[0].(*fyne.Container)
	for _, object := range row.Objects {
		if button, ok := object.(*widget.Button); ok { test.Tap(button) }
	}
	if selected != "Research" || s.tagPopup.Visible() { t.Fatal("tag selection did not filter and close the popup") }
	test.Tap(s.compactTags)
	test.Tap(s.popupAdd)
	if !added || s.tagPopup.Visible() { t.Fatal("compact Add Tag action did not run and close") }
}
