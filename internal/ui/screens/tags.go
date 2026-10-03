package screens

import (
	"reflect"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
	"github.com/NguyenHien-8/NoteHub/internal/ui/components"
)

type Tags struct {
	Object  fyne.CanvasObject
	content *fyne.Container
	onTag   func(string)
	tags    []domain.TagCount
	loaded  bool
}

func NewTags(onTag func(string)) *Tags {
	s := &Tags{content: container.NewStack(), onTag: onTag}
	s.Object = container.NewBorder(container.NewVBox(widget.NewLabelWithStyle("Tags", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), widget.NewLabel("Add #tags to your notes. Use / to organize related topics.")), nil, nil, nil, container.NewVScroll(s.content))
	return s
}
func (s *Tags) Set(tags []domain.TagCount) {
	if s.loaded && reflect.DeepEqual(s.tags, tags) {
		return
	}
	s.loaded = true
	s.tags = append([]domain.TagCount(nil), tags...)
	s.content.Objects = []fyne.CanvasObject{components.NewTagList(tags, s.onTag)}
	s.content.Refresh()
}
