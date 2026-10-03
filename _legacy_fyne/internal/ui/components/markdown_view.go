package components

import (
	"strings"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// MarkdownView renders local text only. A small canvas renderer supports the
// extended formatting that Fyne's Markdown RichText does not support yet.
type MarkdownView struct {
	widget.BaseWidget
	runs     []markdownRun
	MaxLines int // zero displays the entire document
}

func NewMarkdownView(source string) *MarkdownView {
	v := &MarkdownView{runs: parseNoteMarkdown(source)}
	v.ExtendBaseWidget(v)
	return v
}

func (v *MarkdownView) SetMarkdown(source string) { v.runs = parseNoteMarkdown(source); v.Refresh() }

func (v *MarkdownView) CreateRenderer() fyne.WidgetRenderer {
	r := &markdownRenderer{view: v}
	r.Refresh()
	return r
}

type markdownRenderer struct {
	view    *MarkdownView
	objects []fyne.CanvasObject
	height  float32
}

func (r *markdownRenderer) Destroy()                     {}
func (r *markdownRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *markdownRenderer) MinSize() fyne.Size {
	width := r.view.Size().Width
	if width <= 0 {
		width = 400
	}
	_, height := r.flow(width, false)
	return fyne.NewSize(1, height)
}
func (r *markdownRenderer) Layout(size fyne.Size) { r.objects, r.height = r.flow(size.Width, true) }
func (r *markdownRenderer) Refresh()              { r.Layout(r.view.Size()); canvas.Refresh(r.view) }

func (r *markdownRenderer) flow(width float32, draw bool) ([]fyne.CanvasObject, float32) {
	if width <= 0 {
		width = 400
	}
	defaultSize := theme.SizeForWidget(theme.SizeNameText, r.view)
	defaultHeight := fyne.MeasureText("M", defaultSize, fyne.TextStyle{}).Height
	gap := float32(3)
	x, y, lineHeight := float32(0), float32(0), defaultHeight
	line, stopped := 1, false
	var objects []fyne.CanvasObject
	newLine := func() {
		if r.view.MaxLines > 0 && line >= r.view.MaxLines {
			stopped = true
			return
		}
		y += lineHeight + gap
		x = 0
		lineHeight = defaultHeight
		line++
	}
	for runIndex, run := range r.view.runs {
		if stopped {
			break
		}
		size := run.style.size
		if size == 0 {
			size = defaultSize
		}
		foreground := run.style.foreground
		if foreground == nil {
			foreground = theme.ColorForWidget(theme.ColorNameForeground, r.view)
		}
		for _, token := range markdownTokens(run.text) {
			if stopped {
				break
			}
			if token == "\n" {
				// The parser's final block delimiter is not a visible blank row.
				if runIndex != len(r.view.runs)-1 || !strings.HasSuffix(run.text, token) {
					newLine()
				}
				continue
			}
			remaining := []rune(strings.ReplaceAll(token, "\t", "    "))
			for len(remaining) > 0 && !stopped {
				part := string(remaining)
				measured := fyne.MeasureText(part, size, run.style.text)
				if x > 0 && x+measured.Width > width {
					newLine()
					if stopped {
						break
					}
				}
				take := len(remaining)
				if measured.Width > width {
					// Split long URLs/code words too, without slicing UTF-8 bytes.
					low, high := 1, len(remaining)
					for low < high {
						mid := (low + high + 1) / 2
						if fyne.MeasureText(string(remaining[:mid]), size, run.style.text).Width <= width {
							low = mid
						} else {
							high = mid - 1
						}
					}
					take = low
					part = string(remaining[:take])
					measured = fyne.MeasureText(part, size, run.style.text)
				}
				if draw {
					if run.style.background != nil {
						background := canvas.NewRectangle(run.style.background)
						background.Move(fyne.NewPos(x, y))
						background.Resize(measured)
						objects = append(objects, background)
					}
					text := canvas.NewText(part, foreground)
					text.TextSize, text.TextStyle = size, run.style.text
					text.Move(fyne.NewPos(x, y))
					text.Resize(measured)
					objects = append(objects, text)
					for _, decoration := range []struct {
						show     bool
						fraction float32
					}{{run.style.underline, .89}, {run.style.strike, .54}} {
						if !decoration.show {
							continue
						}
						stroke := canvas.NewLine(foreground)
						stroke.StrokeWidth = 1
						stroke.Position1 = fyne.NewPos(x, y+measured.Height*decoration.fraction)
						stroke.Position2 = fyne.NewPos(x+measured.Width, y+measured.Height*decoration.fraction)
						objects = append(objects, stroke)
					}
				}
				x += measured.Width
				lineHeight = max(lineHeight, measured.Height)
				remaining = remaining[take:]
				if len(remaining) > 0 {
					newLine()
				}
			}
		}
	}
	return objects, y + lineHeight
}

func markdownTokens(text string) []string {
	var tokens []string
	var current []rune
	flush := func() {
		if len(current) > 0 {
			tokens = append(tokens, string(current))
			current = nil
		}
	}
	space := false
	for _, ch := range text {
		if ch == '\n' {
			flush()
			tokens = append(tokens, "\n")
			continue
		}
		nextSpace := unicode.IsSpace(ch)
		if len(current) > 0 && nextSpace != space {
			flush()
		}
		current = append(current, ch)
		space = nextSpace
	}
	flush()
	return tokens
}
