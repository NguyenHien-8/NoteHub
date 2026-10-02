package components

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
)

type Sidebar struct {
	Object   fyne.CanvasObject
	menu     *fyne.Container
	tags     *container.Scroll
	buttons  map[string]*widget.Button
	onTag    func(string)
	selected string
}

func NewSidebar(onNavigate func(string), onTag func(string), onAddTag func()) *Sidebar {
	s := &Sidebar{menu: container.NewVBox(), tags: container.NewVScroll(sidebarTags(nil, onTag)), buttons: map[string]*widget.Button{}, onTag: onTag}
	s.tags.SetMinSize(fyne.NewSize(180, 160))
	for _, item := range []struct {
		name string
		icon fyne.Resource
	}{
		{"Home", theme.HomeIcon()}, {"Calendar", theme.CalendarIcon()}, {"Search", theme.SearchIcon()},
		{"Attachments", theme.MailAttachmentIcon()}, {"Tags", theme.ListIcon()}, {"Settings", theme.SettingsIcon()},
	} {
		id := strings.ToLower(item.name)
		button := widget.NewButtonWithIcon(item.name, item.icon, func() {
			if onNavigate != nil {
				onNavigate(id)
			}
		})
		button.Alignment = widget.ButtonAlignLeading
		button.Importance = widget.LowImportance
		s.buttons[id] = button
		// Keep each button in one stable theme container. Reparenting the same
		// widget on every navigation can discard native renderer resources.
		s.menu.Add(container.NewThemeOverride(button, scopedTheme{
			background: color.NRGBA{R: 234, G: 243, B: 255, A: 255}, foreground: primaryBlue,
		}))
	}
	add := widget.NewButtonWithIcon("", theme.ContentAddIcon(), onAddTag)
	add.Importance = widget.LowImportance
	if onAddTag == nil {
		add.Disable()
	}
	heading := container.NewBorder(nil, nil, nil, add, widget.NewLabelWithStyle("My Tags", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	top := container.New(layout.NewCustomPaddedVBoxLayout(16), s.menu, widget.NewSeparator(), heading)
	s.Object = container.NewBorder(top, nil, nil, nil, s.tags)
	s.SetSelected("home")
	return s
}

func (s *Sidebar) SetSelected(selected string) {
	s.selected = strings.ToLower(selected)
	for _, id := range []string{"home", "calendar", "search", "attachments", "tags", "settings"} {
		button := s.buttons[id]
		button.Importance = widget.LowImportance
		if id == s.selected {
			button.Importance = widget.HighImportance
		}
		button.Refresh()
	}
	s.menu.Refresh()
}

func (s *Sidebar) SetTags(tags []domain.TagCount) {
	s.tags.Content = sidebarTags(tags, s.onTag)
	s.tags.Refresh()
}

func sidebarTags(tags []domain.TagCount, onTag func(string)) fyne.CanvasObject {
	if len(tags) == 0 {
		return NewTagList(nil, onTag)
	}
	rows := container.NewVBox()
	for _, tag := range tags {
		name := trimTag(tag.Tag)
		shade := tagColor(name)
		dot := fyne.NewStaticResource("tag-dot.svg", []byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 20 20"><circle cx="10" cy="10" r="4" fill="#%02x%02x%02x"/></svg>`, shade.R-96, shade.G-96, shade.B-96)))
		button := widget.NewButtonWithIcon(boundedText("#"+name, 18), dot, func() {
			if onTag != nil {
				onTag(name)
			}
		})
		button.Importance, button.Alignment = widget.LowImportance, widget.ButtonAlignLeading
		count := widget.NewLabel(fmt.Sprintf("%d", tag.Count))
		count.Importance = widget.LowImportance
		rows.Add(container.NewBorder(nil, nil, nil, count, button))
	}
	return rows
}
