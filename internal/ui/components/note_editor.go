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

func themedEditorSVG(name, body string) fyne.Resource {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><g fill="#000000" stroke="#000000" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">` + body + `</g></svg>`
	return theme.NewThemedResource(fyne.NewStaticResource("note-editor-"+name+".svg", []byte(svg)))
}

var (
	noteEditorBoldIcon      = themedEditorSVG("bold", `<path stroke="none" d="M7 4h6a4 4 0 0 1 0 8H7zm0 8h7a4 4 0 0 1 0 8H7zm3-5v3h3a1.5 1.5 0 0 0 0-3zm0 6v4h4a2 2 0 0 0 0-4z"/>`)
	noteEditorItalicIcon    = themedEditorSVG("italic", `<path fill="none" d="M10 4h8M6 20h8M14 4L10 20"/>`)
	noteEditorUnderlineIcon = themedEditorSVG("underline", `<path fill="none" d="M7 4v7a5 5 0 0 0 10 0V4M5 21h14"/>`)
	noteEditorStrikeIcon    = themedEditorSVG("strike", `<path fill="none" d="M6 6c1.5-2 3.5-3 6-3 3.2 0 5 1.5 6 3M7 18c1.3 1.8 3 2.7 5.4 2.7 3.4 0 5.6-1.7 5.6-4.1 0-1.5-.8-2.5-2.4-3.2M4 12h16"/>`)
	noteEditorSizeIcon      = themedEditorSVG("size", `<path fill="none" d="M4 7V4h10v3M9 4v16M6 20h6M15 10h5M17.5 10v10M15.5 20h4"/>`)
	noteEditorColorIcon     = themedEditorSVG("color", `<path fill="none" d="M4 19h16M7 16l5-12 5 12M9 12h6"/>`)
	noteEditorHighlightIcon = themedEditorSVG("highlight", `<path fill="none" d="M8 15l7-9 3 3-7 9H8zM6 20h12"/>`)
	noteEditorBulletIcon    = themedEditorSVG("bullets", `<circle stroke="none" cx="5" cy="7" r="1.4"/><circle stroke="none" cx="5" cy="12" r="1.4"/><circle stroke="none" cx="5" cy="17" r="1.4"/><path fill="none" d="M9 7h11M9 12h11M9 17h11"/>`)
	noteEditorNumberIcon    = themedEditorSVG("numbering", `<path fill="none" d="M4 5h2v5M4 10h3M4 14c0-1 3-1.5 3 .4 0 1.4-3 2.5-3 4.6h3M10 7h10M10 12h10M10 17h10"/>`)
	noteEditorIndentIcon    = themedEditorSVG("indent", `<path fill="none" d="M10 6h10M10 11h10M10 16h10M10 21h10M4 9l4 3-4 3z"/>`)
	noteEditorOutdentIcon   = themedEditorSVG("outdent", `<path fill="none" d="M10 6h10M10 11h10M10 16h10M10 21h10M8 9l-4 3 4 3z"/>`)
)

func NewNoteEditor(window fyne.Window) *NoteEditor {
	e := &NoteEditor{Entry: widget.NewMultiLineEntry(), window: window, preview: NewMarkdownView("")}
	// Memo.Content remains one UTF-8, Markdown-compatible source of truth. Plain
	// text is valid Markdown, while toolbar actions add standard Markdown or a
	// small allowlisted set of inline HTML styles for rich-text features that
	// Markdown itself does not define (underline, size, color and highlight).
	e.Entry.Wrapping = fyne.TextWrapOff
	e.Entry.SetPlaceHolder("Write normal text or Markdown…")
	e.Entry.OnChanged = func(value string) {
		if e.tabs != nil && e.tabs.SelectedIndex() == 1 {
			e.preview.SetMarkdown(value)
		}
	}

	source := container.NewTabItem("Editor", e.Entry)
	preview := container.NewTabItem("Preview", container.NewVScroll(e.preview))
	e.tabs = container.NewAppTabs(source, preview)
	e.tabs.OnSelected = func(tab *container.TabItem) {
		if tab == preview {
			e.preview.SetMarkdown(e.Entry.Text)
		}
	}

	toolbar := container.New(layout.NewRowWrapLayoutWithCustomPadding(4, 4))
	button := func(label string, icon fyne.Resource, action func()) {
		b := widget.NewButtonWithIcon(label, icon, func() {
			if !e.busy && !e.closed {
				action()
			}
		})
		b.Importance = widget.LowImportance
		e.controls = append(e.controls, b)
		toolbar.Add(b)
	}
	button("Bold", noteEditorBoldIcon, func() { e.Format("bold", "") })
	button("Italic", noteEditorItalicIcon, func() { e.Format("italic", "") })
	button("Underline", noteEditorUnderlineIcon, func() { e.Format("underline", "") })
	button("Strike", noteEditorStrikeIcon, func() { e.Format("strike", "") })

	size := widget.NewSelect([]string{"10", "12", "14", "16", "18", "20", "24", "28", "32", "40", "48", "64"}, func(value string) {
		if value != "" {
			e.Format("size", value)
		}
	})
	size.PlaceHolder = "Size"
	e.controls = append(e.controls, size)
	toolbar.Add(container.NewBorder(nil, nil, widget.NewIcon(noteEditorSizeIcon), nil, container.NewGridWrap(fyne.NewSize(86, size.MinSize().Height), size)))

	colors := []string{"Black", "White", "Blue", "Green", "Red", "Purple", "Orange", "Yellow"}
	palette := map[string]string{"Black": "#202020", "White": "#ffffff", "Blue": "#2563eb", "Green": "#16803d", "Red": "#dc2626", "Purple": "#9333ea", "Orange": "#f97316", "Yellow": "#fef08a"}
	colorSelect := func(label, action string, icon fyne.Resource) {
		selectColor := widget.NewSelect(colors, func(value string) {
			if code := palette[value]; code != "" {
				e.Format(action, code)
			}
		})
		selectColor.PlaceHolder = label
		e.controls = append(e.controls, selectColor)
		toolbar.Add(container.NewBorder(nil, nil, widget.NewIcon(icon), nil, container.NewGridWrap(fyne.NewSize(116, selectColor.MinSize().Height), selectColor)))
	}
	colorSelect("Text color", "color", noteEditorColorIcon)
	colorSelect("Highlight", "highlight", noteEditorHighlightIcon)

	button("Clear", theme.ContentClearIcon(), func() { e.Format("clear", "") })
	button("Bullets", noteEditorBulletIcon, func() { e.Format("bullet", "") })
	button("Numbering", noteEditorNumberIcon, func() { e.Format("number", "") })
	button("Indent", noteEditorIndentIcon, func() { e.Format("indent", "") })
	button("Outdent", noteEditorOutdentIcon, func() { e.Format("outdent", "") })
	button("Undo", theme.ContentUndoIcon(), func() { e.Entry.Undo(); e.focusSource() })
	button("Redo", theme.ContentRedoIcon(), func() { e.Entry.Redo(); e.focusSource() })

	importButton := widget.NewButtonWithIcon("Import .md", theme.FolderOpenIcon(), e.importMarkdown)
	exportButton := widget.NewButtonWithIcon("Export .md", theme.DocumentSaveIcon(), e.exportMarkdown)
	importButton.Importance, exportButton.Importance = widget.LowImportance, widget.LowImportance
	e.controls = append(e.controls, importButton, exportButton)
	files := container.New(layout.NewRowWrapLayoutWithCustomPadding(4, 4), importButton, exportButton)

	modeHint := widget.NewLabel("Rich-text tools and Markdown share the same note. Select text to format; Preview shows the saved result.")
	modeHint.Importance, modeHint.Wrapping = widget.LowImportance, fyne.TextWrapWord
	e.Object = container.NewBorder(container.NewVBox(toolbar, files, modeHint), nil, nil, nil, e.tabs)
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
	if selected == "" {
		// Match a normal rich-text editor: pressing a formatting tool without a
		// selection arms that style for the next text instead of inserting a
		// visible placeholder word.
		cursor := e.Entry.CursorTextOffset()
		e.paste(open + close)
		e.setCursorOffset(cursor + utf8.RuneCountInString(open))
		e.focusSource()
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

func (e *NoteEditor) setCursorOffset(offset int) {
	runes := []rune(e.Entry.Text)
	offset = max(0, min(offset, len(runes)))
	row, column := 0, 0
	for _, ch := range runes[:offset] {
		if ch == '\n' {
			row++
			column = 0
		} else {
			column++
		}
	}
	e.Entry.CursorRow, e.Entry.CursorColumn = row, column
	e.Entry.Refresh()
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
