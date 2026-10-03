package components

import (
	"fmt"
	"image/color"
	"reflect"
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
	tags     *fyne.Container
	scroll   *container.Scroll
	heading  fyne.CanvasObject
	compactTags *HintButton
	buttons  map[string]*HintButton
	compact  bool
	onTag    func(string)
	onAddTag func()
	selected string
	tagData  []domain.TagCount
	tagPopup *widget.PopUp
	popupTags *fyne.Container
	popupAdd *widget.Button
}

func NewSidebar(onNavigate func(string), onTag func(string), onAddTag func()) *Sidebar {
	s := &Sidebar{menu: container.NewVBox(), tags: container.NewStack(sidebarTags(nil, onTag)), buttons: map[string]*HintButton{}, onTag: onTag, onAddTag: onAddTag}
	for _, item := range []struct {
		name string
		icon fyne.Resource
	}{
		{"Home", theme.HomeIcon()}, {"Calendar", theme.CalendarIcon()}, {"Search", theme.SearchIcon()},
		{"Attachments", theme.MailAttachmentIcon()}, {"Tags", theme.ListIcon()}, {"Settings", theme.SettingsIcon()},
	} {
		id := strings.ToLower(item.name)
		button := NewHintButton(item.name, item.icon, func() {
			if onNavigate != nil {
				onNavigate(id)
			}
		})
		button.Alignment = widget.ButtonAlignLeading
		button.Importance = widget.LowImportance
		s.buttons[id] = button
		// Keep each button in one stable theme container to avoid rebuilding
		// the menu and its theme scopes on every navigation.
		s.menu.Add(container.NewThemeOverride(button, scopedTheme{
			background: color.NRGBA{R: 234, G: 243, B: 255, A: 255}, foreground: primaryBlue,
		}))
	}
	add := widget.NewButtonWithIcon("", theme.ContentAddIcon(), onAddTag)
	add.Importance = widget.LowImportance
	if onAddTag == nil {
		add.Disable()
	}
	s.heading = container.NewBorder(nil, nil, nil, add, widget.NewLabelWithStyle("My Tags", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	s.compactTags = NewHintButton("My Tags", theme.FolderOpenIcon(), s.showTags)
	s.compactTags.SetText("")
	s.compactTags.Importance = widget.LowImportance
	s.compactTags.Hide()
	content := container.New(layout.NewCustomPaddedVBoxLayout(12), s.menu, widget.NewSeparator(), s.heading, s.tags, s.compactTags)
	s.scroll = container.NewVScroll(content)
	s.Object = s.scroll
	s.SetSelected("home")
	return s
}

func (s *Sidebar) SetHintLayer(hints *HintLayer) {
	for _, button := range s.buttons { button.SetHintLayer(hints) }
	s.compactTags.SetHintLayer(hints)
}

func (s *Sidebar) SetCompact(compact bool) {
	if s.compact == compact { return }
	s.compact = compact
	for _, button := range s.buttons {
		button.hideHint()
		if compact {
			button.Alignment = widget.ButtonAlignCenter
			button.SetText("")
		} else {
			button.Alignment = widget.ButtonAlignLeading
			button.SetText(button.Hint)
		}
	}
	if compact {
		s.heading.Hide()
		s.tags.Hide()
		s.compactTags.Show()
	} else {
		s.heading.Show()
		s.tags.Show()
		s.compactTags.Hide()
		if s.tagPopup != nil { s.tagPopup.Hide() }
	}
	s.scroll.Offset = fyne.Position{}
	s.scroll.Refresh()
}

func (s *Sidebar) showTags() {
	canvas := fyne.CurrentApp().Driver().CanvasForObject(s.compactTags)
	if canvas == nil { return }
	if s.tagPopup != nil { s.tagPopup.Hide() }
	selectTag := func(tag string) {
		s.tagPopup.Hide()
		if s.onTag != nil { s.onTag(tag) }
	}
	s.popupTags = container.NewStack(sidebarTags(s.tagData, selectTag))
	s.popupAdd = widget.NewButtonWithIcon("Add tag", theme.ContentAddIcon(), func() {
		s.tagPopup.Hide()
		if s.onAddTag != nil { s.onAddTag() }
	})
	if s.onAddTag == nil { s.popupAdd.Disable() }
	close := widget.NewButtonWithIcon("", theme.CancelIcon(), func() { s.tagPopup.Hide() })
	close.Importance = widget.LowImportance
	heading := container.NewBorder(nil, nil, nil, close, widget.NewLabelWithStyle("My Tags", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	content := container.NewBorder(heading, s.popupAdd, nil, nil, container.NewVScroll(s.popupTags))
	s.tagPopup = widget.NewPopUp(content, canvas)
	s.tagPopup.Resize(fyne.NewSize(min(280, canvas.Size().Width-20), min(360, canvas.Size().Height-20)))
	s.tagPopup.ShowAtRelativePosition(fyne.NewPos(s.compactTags.Size().Width+12, 0), s.compactTags)
}

func (s *Sidebar) SetSelected(selected string) {
	selected = strings.ToLower(selected)
	if selected == s.selected {
		return
	}
	previous := s.selected
	s.selected = selected
	if button := s.buttons[previous]; button != nil {
		button.Importance = widget.LowImportance
		button.Refresh()
	}
	if button := s.buttons[selected]; button != nil {
		button.Importance = widget.HighImportance
		button.Refresh()
	}
}

func (s *Sidebar) SetTags(tags []domain.TagCount) {
	if reflect.DeepEqual(s.tagData, tags) {
		return
	}
	s.tagData = append([]domain.TagCount(nil), tags...)
	s.tags.Objects = []fyne.CanvasObject{sidebarTags(tags, s.onTag)}
	s.tags.Refresh()
	if s.tagPopup != nil && s.tagPopup.Visible() {
		s.popupTags.Objects = []fyne.CanvasObject{sidebarTags(tags, func(tag string) {
			s.tagPopup.Hide()
			if s.onTag != nil { s.onTag(tag) }
		})}
		s.popupTags.Refresh()
	}
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
