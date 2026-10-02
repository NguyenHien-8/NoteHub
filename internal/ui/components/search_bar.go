package components

import (
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func NewSearchBar(onChanged func(string)) *widget.Entry {
	entry := widget.NewEntry()
	entry.Scroll = widget.ScrollHorizontalOnly
	entry.SetPlaceHolder("Tìm kiếm ghi chú, tag, nội dung…")
	entry.Icon = theme.SearchIcon()
	entry.OnChanged = onChanged
	return entry
}
