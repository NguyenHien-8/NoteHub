package components

import (
	"fmt"
	"image/color"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
)

type Calendar struct {
	Object     fyne.CanvasObject
	label      *widget.Label
	grid       *fyne.Container
	onDay      func(string)
	dayButtons map[int]*widget.Button
}

func NewCalendar(onDay func(string), onMonth func(int)) *Calendar {
	c := &Calendar{label: widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), grid: container.NewGridWithColumns(7), onDay: onDay}
	c.label.Truncation = fyne.TextTruncateEllipsis
	previous := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		if onMonth != nil {
			onMonth(-1)
		}
	})
	next := widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
		if onMonth != nil {
			onMonth(1)
		}
	})
	previous.Importance, next.Importance = widget.LowImportance, widget.LowImportance
	nav := container.NewHBox(previous, next)
	header := container.NewBorder(nil, nil, nil, nav, c.label)
	week := container.NewGridWithColumns(7)
	for _, day := range []string{"T2", "T3", "T4", "T5", "T6", "T7", "CN"} {
		label := widget.NewLabelWithStyle(day, fyne.TextAlignCenter, fyne.TextStyle{})
		label.Importance = widget.LowImportance
		week.Add(container.NewThemeOverride(label, scopedTheme{compact: true, background: primaryBlue, foreground: color.White}))
	}
	c.Object = Surface(container.New(layout.NewCustomPaddedVBoxLayout(8), header, week, c.grid))
	return c
}

func (c *Calendar) SetMonth(year int, month time.Month, days []domain.CalendarDay, selected string) {
	c.label.SetText(time.Date(year, month, 1, 12, 0, 0, 0, time.Local).Format("January 2006"))
	counts := make(map[string]int, len(days))
	for _, day := range days {
		counts[day.Date] = day.Count
	}
	c.grid.Objects = nil
	c.dayButtons = make(map[int]*widget.Button)
	for _, day := range monthCells(year, month) {
		if day == 0 {
			c.grid.Add(layout.NewSpacer())
			continue
		}
		date := fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		button := widget.NewButton(strconv.Itoa(day), func() {
			if c.onDay != nil {
				c.onDay(date)
			}
		})
		button.Importance = widget.LowImportance
		th := scopedTheme{background: primaryBlue, foreground: color.White, compact: true}
		if date == selected {
			button.Importance = widget.HighImportance
		}
		c.dayButtons[day] = button
		marker := " "
		if count := counts[date]; count > 0 {
			marker = "•"
			if count > 1 {
				marker = fmt.Sprintf("• %d", count)
			}
		}
		dot := widget.NewLabelWithStyle(marker, fyne.TextAlignCenter, fyne.TextStyle{})
		dot.Importance = widget.HighImportance
		dot.SizeName = theme.SizeNameCaptionText
		cell := container.New(layout.NewCustomPaddedVBoxLayout(0), container.NewThemeOverride(button, th), dot)
		c.grid.Add(cell)
	}
	c.grid.Refresh()
}
