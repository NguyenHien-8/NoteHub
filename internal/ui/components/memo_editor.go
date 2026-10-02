package components

import (
	"path/filepath"
	"strings"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Composer stages attachments until its owner confirms a successful save.
// All methods must be called on the Fyne application thread.
type Composer struct {
	Object            fyne.CanvasObject
	Entry             *widget.Entry
	staged            *fyne.Container
	paths             []string
	save, attach, tag *widget.Button
	busy              bool
}

func NewComposer(onSave func(string, []string), onAttach, onTag func()) *Composer {
	c := &Composer{Entry: widget.NewMultiLineEntry(), staged: container.NewVBox()}
	c.Entry.SetPlaceHolder("Có suy nghĩ gì...")
	c.Entry.Wrapping = fyne.TextWrapWord
	c.Entry.SetMinRowsVisible(4)
	c.save = widget.NewButton("Save", func() {
		if c.busy || (strings.TrimSpace(c.Entry.Text) == "" && len(c.paths) == 0) || onSave == nil {
			return
		}
		onSave(c.Entry.Text, c.Paths())
	})
	c.save.Importance = widget.HighImportance
	c.attach = widget.NewButtonWithIcon("Đính kèm", theme.MailAttachmentIcon(), func() {
		if !c.busy && onAttach != nil {
			onAttach()
		}
	})
	c.tag = widget.NewButtonWithIcon("Thêm tag", theme.ContentAddIcon(), func() {
		if !c.busy && onTag != nil {
			onTag()
		}
	})
	c.attach.Importance, c.tag.Importance = widget.LowImportance, widget.LowImportance
	actions := container.NewHBox(c.attach, c.tag, layout.NewSpacer(), c.save)
	c.Object = Surface(container.New(layout.NewCustomPaddedVBoxLayout(8), c.Entry, c.staged, actions))
	c.staged.Hide()
	return c
}

func (c *Composer) AddPaths(paths []string) {
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		found := false
		for _, existing := range c.paths {
			if path == existing {
				found = true
				break
			}
		}
		if !found {
			c.paths = append(c.paths, path)
		}
	}
	c.refreshPaths()
}

func (c *Composer) Paths() []string { return append([]string(nil), c.paths...) }

func (c *Composer) Reset() {
	c.Entry.SetText("")
	c.paths = nil
	c.refreshPaths()
}

func (c *Composer) SetBusy(busy bool) {
	c.busy = busy
	if busy {
		c.Entry.Disable()
		c.save.Disable()
		c.attach.Disable()
		c.tag.Disable()
		c.save.SetText("Saving…")
	} else {
		c.Entry.Enable()
		c.save.Enable()
		c.attach.Enable()
		c.tag.Enable()
		c.save.SetText("Save")
	}
	c.refreshPaths()
}

func (c *Composer) InsertTag(tag string) {
	tag = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(tag), "#"))
	if tag == "" || c.busy {
		return
	}
	runes := []rune(c.Entry.Text)
	position, row, column := 0, 0, 0
	for position < len(runes) {
		if row == c.Entry.CursorRow && column >= c.Entry.CursorColumn {
			break
		}
		if runes[position] == '\n' {
			row++
			column = 0
		} else {
			column++
		}
		position++
		if row > c.Entry.CursorRow {
			break
		}
	}
	insert := "#" + tag
	if position > 0 && !unicode.IsSpace(runes[position-1]) {
		insert = " " + insert
	}
	if position < len(runes) && !unicode.IsSpace(runes[position]) {
		insert += " "
	}
	text := string(runes[:position]) + insert + string(runes[position:])
	c.Entry.SetText(text)
	prefix := strings.Split(string(runes[:position])+insert, "\n")
	c.Entry.CursorRow = len(prefix) - 1
	c.Entry.CursorColumn = len([]rune(prefix[len(prefix)-1]))
	c.Entry.Refresh()
}

func (c *Composer) refreshPaths() {
	c.staged.Objects = nil
	for _, path := range c.paths {
		path := path
		filename := widget.NewLabel(boundedText(filepath.Base(path), 48))
		filename.Truncation = fyne.TextTruncateEllipsis
		remove := widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
			if c.busy {
				return
			}
			for i, existing := range c.paths {
				if existing == path {
					c.paths = append(c.paths[:i], c.paths[i+1:]...)
					break
				}
			}
			c.refreshPaths()
		})
		remove.Importance = widget.LowImportance
		if c.busy {
			remove.Disable()
		}
		c.staged.Add(container.NewHBox(filename, remove))
	}
	if len(c.paths) == 0 {
		c.staged.Hide()
	} else {
		c.staged.Show()
	}
	c.staged.Refresh()
}
