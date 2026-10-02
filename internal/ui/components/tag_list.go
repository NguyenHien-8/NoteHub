package components

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
)

func NewTagList(tags []domain.TagCount, onTag func(string)) fyne.CanvasObject {
	rows := container.NewVBox()
	if len(tags) == 0 {
		label := widget.NewLabel("Your tags will appear here.\nAdd #tags to a note to organize it.")
		label.Wrapping = fyne.TextWrapWord
		label.Importance = widget.LowImportance
		rows.Add(label)
	}
	for _, tag := range tags {
		count := widget.NewLabel(fmt.Sprintf("%d", tag.Count))
		count.Importance = widget.LowImportance
		rows.Add(container.NewBorder(nil, nil, nil, count, tagButton(tag.Tag, onTag)))
	}
	return rows
}
