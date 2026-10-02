package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"github.com/NguyenHien-8/NoteHub/internal/repository"
	"strings"
)

func (d *Desktop) Navigate(page string) {
	page = strings.ToLower(page)
	var object fyne.CanvasObject
	switch page {
	case "home":
		object = d.Home.Object
		d.Home.Refresh()
	case "calendar":
		object = d.Calendar.Object
		d.Calendar.Refresh()
	case "search":
		object = d.Search.Object
		d.Search.Refresh()
	case "attachments":
		object = d.Attachments.Object
		d.Attachments.Refresh()
	case "tags":
		object = d.tags.Object
	case "settings":
		object = d.settings
	default:
		return
	}
	d.current = page
	d.sidebar.SetSelected(page)
	d.center.Objects = []fyne.CanvasObject{object}
	d.center.Refresh()
}

func (d *Desktop) Refresh() {
	d.Home.Refresh()
	switch d.current {
	case "calendar":
		d.Calendar.Refresh()
	case "search":
		d.Search.Refresh()
	case "attachments":
		d.Attachments.Refresh()
	}
	d.refreshMetadata()
}

func (d *Desktop) showFilter(q repository.TimelineQuery) {
	d.selectedDate = ""
	d.Home.Filter(q, "")
	d.current = "home"
	d.sidebar.SetSelected("home")
	d.center.Objects = []fyne.CanvasObject{d.Home.Object}
	d.center.Refresh()
	d.refreshMini()
}
func (d *Desktop) showTag(tag string) { d.showFilter(repository.TimelineQuery{Tags: []string{tag}}) }
func (d *Desktop) showDate(date string) {
	d.selectedDate = date
	d.Home.Filter(repository.TimelineQuery{}, date)
	d.current = "home"
	d.sidebar.SetSelected("home")
	d.center.Objects = []fyne.CanvasObject{d.Home.Object}
	d.center.Refresh()
	d.refreshMini()
}

func (d *Desktop) requestClose() {
	closeWindow := func() { d.Window.SetCloseIntercept(nil); d.Window.Close() }
	if d.Jobs.Busy() {
		dialog.ShowInformation("Please wait", "An operation is still running. Wait for it to finish before closing NoteHub.", d.Window)
		return
	}
	if d.Home.Dirty() || d.manager.Dirty() {
		dialog.ShowConfirm("Discard unsaved changes?", "Your unsaved text and staged files will be discarded.", func(ok bool) {
			if ok {
				closeWindow()
			}
		}, d.Window)
		return
	}
	closeWindow()
}

// Navigation between Timeline, Calendar, Search, Attachments and Settings.
