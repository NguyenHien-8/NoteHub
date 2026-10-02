package components

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func NewSearchBar(onChanged func(string)) *widget.Entry {
	entry := widget.NewEntry()
	// Single-line search never needs an embedded scroll bar. Fyne still keeps
	// the caret visible while editing, without rendering a vertical thumb when
	// the field is empty.
	entry.Scroll = fyne.ScrollNone
	entry.SetPlaceHolder("Tìm kiếm ghi chú, tag, nội dung…")
	entry.Icon = theme.SearchIcon()
	entry.OnChanged = onChanged
	return entry
}
