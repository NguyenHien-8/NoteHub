package components

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// NoteEditor keeps a regular Entry as the draft's source of truth. The persisted
// value is readable Markdown, with inline HTML for styles Markdown cannot encode.
type NoteEditor struct {
	Object       fyne.CanvasObject
	Entry        *widget.Entry
	preview      *MarkdownView
	tabs         *container.AppTabs
	controls     []fyne.Disableable
	window       fyne.Window
	busy, closed bool
}

func NewNoteEditor(window fyne.Window) *NoteEditor {
	e := &NoteEditor{Entry: widget.NewMultiLineEntry(), window: window, preview: NewMarkdownView("")}
	// Source lines stay physical lines, so lists and indentation are predictable.
	// The rendered preview wraps to the available width.
	e.Entry.Wrapping = fyne.TextWrapOff
	e.Entry.SetPlaceHolder("Write Markdown here, or import a .md file…")
	e.Entry.OnChanged = func(value string) {
		if e.tabs != nil && e.tabs.SelectedIndex() == 1 {
			e.preview.SetMarkdown(value)
		}
	}
	source := container.NewTabItem("Markdown source", e.Entry)
	preview := container.NewTabItem("Rendered preview", container.NewVScroll(e.preview))
	e.tabs = container.NewAppTabs(source, preview)
	e.tabs.OnSelected = func(tab *container.TabItem) {
		if tab == preview {
			e.preview.SetMarkdown(e.Entry.Text)
		}
	}
	toolbar := container.New(layout.NewRowWrapLayoutWithCustomPadding(3, 3))
	button := func(label string, action func()) {
		b := widget.NewButton(label, func() {
			if !e.busy && !e.closed {
				action()
			}
		})
		b.Importance = widget.LowImportance
		e.controls = append(e.controls, b)
		toolbar.Add(b)
	}
	for _, item := range []struct{ label, action string }{{"Bold", "bold"}, {"Italic", "italic"}, {"Underline", "underline"}, {"Strike", "strike"}} {
		action := item.action
		button(item.label, func() { e.Format(action, "") })
	}
	size := widget.NewSelect([]string{"10", "12", "14", "16", "18", "24", "32", "48"}, func(value string) {
		if value != "" {
			e.Format("size", value)
		}
	})
	size.PlaceHolder = "Font size"
	e.controls = append(e.controls, size)
	toolbar.Add(size)
	colors := []string{"Black", "White", "Blue", "Green", "Red", "Purple", "Orange", "Yellow"}
	palette := map[string]string{"Black": "#202020", "White": "#ffffff", "Blue": "#2563eb", "Green": "#16803d", "Red": "#dc2626", "Purple": "#9333ea", "Orange": "#f97316", "Yellow": "#fef08a"}
	for _, choice := range []struct{ label, action string }{{"Text color", "color"}, {"Highlight", "highlight"}} {
		action := choice.action
		selectColor := widget.NewSelect(colors, func(value string) {
			if code := palette[value]; code != "" {
				e.Format(action, code)
			}
		})
		selectColor.PlaceHolder = choice.label
		e.controls = append(e.controls, selectColor)
		toolbar.Add(selectColor)
	}
	for _, item := range []struct{ label, action string }{{"Clear", "clear"}, {"• List", "bullet"}, {"1. List", "number"}, {"Indent", "indent"}, {"Outdent", "outdent"}} {
		action := item.action
		button(item.label, func() { e.Format(action, "") })
	}
	button("Undo", func() { e.Entry.Undo(); e.focusSource() })
	button("Redo", func() { e.Entry.Redo(); e.focusSource() })
	importButton := widget.NewButtonWithIcon("Import .md", theme.FolderOpenIcon(), e.importMarkdown)
	exportButton := widget.NewButtonWithIcon("Export .md", theme.DocumentSaveIcon(), e.exportMarkdown)
	e.controls = append(e.controls, importButton, exportButton)
	files := container.New(layout.NewRowWrapLayoutWithCustomPadding(3, 3), importButton, exportButton)
	hint := widget.NewLabel("Select text to format; lists and indentation apply to selected lines.")
	hint.Importance, hint.Wrapping = widget.LowImportance, fyne.TextWrapWord
	e.Object = container.NewBorder(container.NewVBox(toolbar, files, hint), nil, nil, nil, e.tabs)
	return e
}

func (e *NoteEditor) SetText(text string) { e.Entry.SetText(text); e.preview.SetMarkdown(text) }
func (e *NoteEditor) Close()              { e.closed = true; e.SetBusy(true) }
func (e *NoteEditor) SetBusy(busy bool) {
	e.busy = busy
	for _, control := range e.controls {
		if busy {
			control.Disable()
		} else {
			control.Enable()
		}
	}
	if busy {
		e.Entry.Disable()
	} else {
		e.Entry.Enable()
	}
}
func (e *NoteEditor) focusSource() {
	e.tabs.SelectIndex(0)
	if e.window != nil {
		e.window.Canvas().Focus(e.Entry)
	}
}

type noteClipboard string

func (c *noteClipboard) Content() string         { return string(*c) }
func (c *noteClipboard) SetContent(value string) { *c = noteClipboard(value) }

func (e *NoteEditor) paste(text string) {
	clipboard := noteClipboard(text)
	e.Entry.TypedShortcut(&fyne.ShortcutPaste{Clipboard: &clipboard})
}

// Format inserts through Entry's editing API so toolbar changes participate in
// native undo/redo and OnChanged, without touching the system clipboard.
func (e *NoteEditor) Format(action, value string) {
	if e.busy || e.closed || e.Entry.Disabled() {
		return
	}
	switch action {
	case "bullet", "number", "indent", "outdent":
		e.formatLines(action)
		return
	case "clear":
		if selected := e.Entry.SelectedText(); selected != "" {
			e.paste(MarkdownPlainText(selected))
		} else {
			e.formatLines(action)
		}
		e.focusSource()
		return
	}
	selected := e.Entry.SelectedText()
	if selected == "" {
		selected = "text"
	}
	var open, close string
	switch action {
	case "bold":
		open, close = "**", "**"
	case "italic":
		open, close = "*", "*"
	case "underline":
		open, close = "<u>", "</u>"
	case "strike":
		open, close = "~~", "~~"
	case "size":
		size, err := strconv.Atoi(value)
		if err != nil || size < 8 || size > 72 {
			return
		}
		open, close = fmt.Sprintf("<span style=\"font-size:%dpx\">", size), "</span>"
	case "color", "highlight":
		if !strings.HasPrefix(value, "#") || markdownColor(value) == nil {
			return
		}
		property := "color"
		if action == "highlight" {
			property = "background-color"
		}
		open, close = fmt.Sprintf("<span style=\"%s:%s\">", property, value), "</span>"
	default:
		return
	}
	// A second application to a selected formatted fragment removes that style.
	if strings.HasPrefix(selected, open) && strings.HasSuffix(selected, close) && len(selected) >= len(open)+len(close) {
		e.paste(selected[len(open) : len(selected)-len(close)])
	} else {
		e.paste(open + selected + close)
	}
	e.focusSource()
}

var noteListMarker = regexp.MustCompile(`^(?:[-+*]|[0-9]+[.)])\s+`)

func (e *NoteEditor) formatLines(action string) {
	runes := []rune(e.Entry.Text)
	selectedLength := utf8.RuneCountInString(e.Entry.SelectedText())
	// Collapsing left gives the real selection start, even for a backwards
	// selection or repeated text. CursorTextOffset also understands wrapped rows.
	if selectedLength > 0 {
		e.Entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyLeft})
	}
	start := e.Entry.CursorTextOffset()
	end := min(start+selectedLength, len(runes))
	for start > 0 && runes[start-1] != '\n' {
		start--
	}
	if end > start && end > 0 && runes[end-1] == '\n' {
		end--
	} else {
		for end < len(runes) && runes[end] != '\n' {
			end++
		}
	}
	lines := strings.Split(string(runes[start:end]), "\n")
	for i, line := range lines {
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		body := strings.TrimPrefix(line, indent)
		switch action {
		case "bullet":
			lines[i] = indent + "- " + noteListMarker.ReplaceAllString(body, "")
		case "number":
			lines[i] = indent + strconv.Itoa(i+1) + ". " + noteListMarker.ReplaceAllString(body, "")
		case "indent":
			lines[i] = "    " + line
		case "outdent":
			if strings.HasPrefix(line, "\t") {
				lines[i] = line[1:]
			} else {
				lines[i] = line[min(4, len(indent)):]
			}
		case "clear":
			lines[i] = MarkdownPlainText(line)
		}
	}
	replacement := strings.Join(lines, "\n")
	updated := string(runes[:start]) + replacement + string(runes[end:])
	if updated == e.Entry.Text {
		e.focusSource()
		return
	}
	e.Entry.TypedShortcut(&fyne.ShortcutSelectAll{})
	e.paste(updated)
	// Source is intentionally unwrapped, so restoring its physical row/column is
	// well-defined. Never interpret a wrapped Entry row as a source line.
	prefix := strings.Split(string(runes[:start])+replacement, "\n")
	e.Entry.CursorRow, e.Entry.CursorColumn = len(prefix)-1, utf8.RuneCountInString(prefix[len(prefix)-1])
	e.Entry.Refresh()
	e.focusSource()
}

func (e *NoteEditor) importMarkdown() {
	if e.busy || e.closed || e.window == nil {
		return
	}
	picker := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if reader != nil {
			defer reader.Close()
		}
		if e.busy || e.closed {
			return
		}
		if err != nil {
			dialog.ShowError(err, e.window)
			return
		}
		if reader == nil {
			return
		}
		content, err := ReadMarkdown(reader)
		if err != nil {
			dialog.ShowError(err, e.window)
			return
		}
		apply := func() {
			if !e.busy && !e.closed {
				e.Entry.TypedShortcut(&fyne.ShortcutSelectAll{})
				e.paste(content)
				e.focusSource()
			}
		}
		if e.Entry.Text == "" {
			apply()
			return
		}
		dialog.ShowConfirm("Replace editor content?", "Import this Markdown file into the editor? Save the note to keep the imported content.", func(ok bool) {
			if ok {
				apply()
			}
		}, e.window)
	}, e.window)
	picker.SetFilter(storage.NewExtensionFileFilter([]string{".md", ".markdown"}))
	picker.Show()
}

func (e *NoteEditor) exportMarkdown() {
	if e.busy || e.closed || e.window == nil {
		return
	}
	content := e.Entry.Text
	picker := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			if !e.closed {
				dialog.ShowError(err, e.window)
			}
			return
		}
		if writer == nil {
			return
		}
		_, writeErr := writer.Write([]byte(content))
		closeErr := writer.Close()
		if writeErr == nil {
			writeErr = closeErr
		}
		if writeErr != nil && !e.closed {
			dialog.ShowError(writeErr, e.window)
		}
	}, e.window)
	picker.SetFileName("note.md")
	picker.SetFilter(storage.NewExtensionFileFilter([]string{".md"}))
	picker.Show()
}
