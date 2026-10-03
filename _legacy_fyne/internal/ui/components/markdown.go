package components

import (
	"fmt"
	"html"
	"image/color"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	nethtml "golang.org/x/net/html"
)

const maxMarkdownBytes = 2 << 20

// ReadMarkdown accepts UTF-8 Markdown and bounds imports before allocating the
// entire file. Source is stored directly in the existing memo content field.
func ReadMarkdown(reader io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(reader, maxMarkdownBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > maxMarkdownBytes {
		return "", fmt.Errorf("Markdown files must be 2 MiB or smaller")
	}
	if !utf8.Valid(data) || strings.ContainsRune(string(data), 0) {
		return "", fmt.Errorf("choose a UTF-8 Markdown text file")
	}
	return strings.ReplaceAll(strings.TrimPrefix(string(data), "\ufeff"), "\r\n", "\n"), nil
}

type markdownStyle struct {
	text                   fyne.TextStyle
	size                   float32
	foreground, background color.Color
	underline, strike      bool
}

type markdownRun struct {
	text  string
	style markdownStyle
}

// parseNoteMarkdown deliberately renders images as alt text, and links as
// styled text. Rendering a note never fetches files, URLs, scripts, or images.
func parseNoteMarkdown(content string) []markdownRun {
	source := []byte(content)
	doc := goldmark.New(goldmark.WithExtensions(extension.Strikethrough)).Parser().Parse(text.NewReader(source))
	var runs []markdownRun
	add := func(value string, style markdownStyle) {
		if value != "" {
			runs = append(runs, markdownRun{value, style})
		}
	}
	newline := func() {
		if len(runs) > 0 && !strings.HasSuffix(runs[len(runs)-1].text, "\n") {
			add("\n", markdownStyle{})
		}
	}
	var visit func(ast.Node, markdownStyle, int)
	visit = func(node ast.Node, style markdownStyle, depth int) {
		children := func(childStyle markdownStyle) {
			// HTML spans affect subsequent siblings, while Markdown emphasis can
			// still nest within them. Closing tags restore the previous style.
			current := childStyle
			var stack []htmlStyleFrame
			for child := node.FirstChild(); child != nil; child = child.NextSibling() {
				if raw, ok := child.(*ast.RawHTML); ok {
					var b strings.Builder
					for i := 0; i < raw.Segments.Len(); i++ {
						seg := raw.Segments.At(i)
						b.Write(seg.Value(source))
					}
					readStyledHTML(b.String(), &current, &stack, add)
					continue
				}
				visit(child, current, depth)
			}
		}
		switch n := node.(type) {
		case *ast.Document:
			children(style)
		case *ast.Paragraph, *ast.TextBlock:
			children(style)
			newline()
		case *ast.Heading:
			style.text.Bold = true
			style.size = map[int]float32{1: 28, 2: 23, 3: 20, 4: 18, 5: 16, 6: 14}[n.Level]
			children(style)
			newline()
		case *ast.Text:
			add(html.UnescapeString(string(util.UnescapePunctuations(n.Value(source)))), style)
			if n.SoftLineBreak() || n.HardLineBreak() {
				add("\n", style)
			}
		case *ast.String:
			add(string(n.Value), style)
		case *ast.Emphasis:
			if n.Level == 2 {
				style.text.Bold = true
			} else {
				style.text.Italic = true
			}
			children(style)
		case *extast.Strikethrough:
			style.strike = true
			children(style)
		case *ast.CodeSpan:
			style.text.Monospace = true
			var b strings.Builder
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if t, ok := c.(*ast.Text); ok {
					b.Write(t.Value(source))
				}
			}
			add(strings.ReplaceAll(b.String(), "\n", " "), style)
		case *ast.CodeBlock, *ast.FencedCodeBlock:
			style.text.Monospace = true
			for i := 0; i < node.Lines().Len(); i++ {
				line := node.Lines().At(i)
				add(string(line.Value(source)), style)
			}
			newline()
		case *ast.List:
			number := n.Start
			for child := n.FirstChild(); child != nil; child = child.NextSibling() {
				marker := "• "
				if n.IsOrdered() {
					marker = strconv.Itoa(number) + ". "
					number++
				}
				add(strings.Repeat("    ", depth)+marker, style)
				visit(child, style, depth+1)
				newline()
			}
		case *ast.ListItem:
			children(style)
		case *ast.Blockquote:
			style.text.Italic = true
			children(style)
			newline()
		case *ast.Link, *ast.AutoLink:
			style.underline = true
			if link, ok := node.(*ast.AutoLink); ok {
				add(string(link.Label(source)), style)
			} else {
				children(style)
			}
		case *ast.Image:
			add("[Image: ", style)
			children(style)
			add("]", style)
		case *ast.ThematicBreak:
			add("────────────\n", style)
		case *ast.HTMLBlock:
			// Only the same small formatting allowlist is interpreted in HTML
			// blocks; arbitrary CSS and executable attributes are ignored.
			var b strings.Builder
			for i := 0; i < n.Lines().Len(); i++ {
				line := n.Lines().At(i)
				b.Write(line.Value(source))
			}
			if n.HasClosure() {
				b.Write(n.ClosureLine.Value(source))
			}
			var stack []htmlStyleFrame
			readStyledHTML(b.String(), &style, &stack, add)
			newline()
		default:
			children(style)
		}
	}
	visit(doc, markdownStyle{}, 0)
	return runs
}

type htmlStyleFrame struct {
	tag   string
	style markdownStyle
}

func readStyledHTML(value string, style *markdownStyle, stack *[]htmlStyleFrame, add func(string, markdownStyle)) {
	tokens := nethtml.NewTokenizer(strings.NewReader(value))
	for {
		kind := tokens.Next()
		if kind == nethtml.ErrorToken {
			return
		}
		token := tokens.Token()
		switch kind {
		case nethtml.TextToken:
			add(token.Data, *style)
		case nethtml.StartTagToken, nethtml.SelfClosingTagToken:
			if token.Data == "br" {
				add("\n", *style)
				continue
			}
			switch token.Data {
			case "span", "u", "s", "del", "strike", "b", "strong", "i", "em", "mark":
				*stack = append(*stack, htmlStyleFrame{token.Data, *style})
			default:
				continue
			}
			switch token.Data {
			case "u":
				style.underline = true
			case "s", "del", "strike":
				style.strike = true
			case "b", "strong":
				style.text.Bold = true
			case "i", "em":
				style.text.Italic = true
			case "mark":
				style.background = color.NRGBA{255, 235, 110, 255}
			}
			for _, attribute := range token.Attr {
				if attribute.Key != "style" {
					continue
				}
				for _, rule := range strings.Split(attribute.Val, ";") {
					pair := strings.SplitN(rule, ":", 2)
					if len(pair) != 2 {
						continue
					}
					key, val := strings.ToLower(strings.TrimSpace(pair[0])), strings.TrimSpace(pair[1])
					switch key {
					case "color":
						style.foreground = markdownColor(val)
					case "background-color":
						style.background = markdownColor(val)
					case "font-size":
						if size, err := strconv.ParseFloat(strings.TrimSuffix(val, "px"), 32); err == nil && size >= 8 && size <= 72 {
							style.size = float32(size)
						}
					}
				}
			}
		case nethtml.EndTagToken:
			for i := len(*stack) - 1; i >= 0; i-- {
				if (*stack)[i].tag == token.Data {
					*style = (*stack)[i].style
					*stack = (*stack)[:i]
					break
				}
			}
		}
	}
}

func markdownColor(value string) color.Color {
	value = strings.TrimPrefix(value, "#")
	if len(value) != 6 {
		return nil
	}
	n, err := strconv.ParseUint(value, 16, 24)
	if err != nil {
		return nil
	}
	return color.NRGBA{uint8(n >> 16), uint8(n >> 8), uint8(n), 255}
}

// MarkdownPlainText is used for compact card titles and clear-format actions.
// Unlike regex stripping, parsing preserves literal code and escaped punctuation.
func MarkdownPlainText(content string) string {
	var out strings.Builder
	for _, run := range parseNoteMarkdown(content) {
		out.WriteString(run.text)
	}
	return strings.TrimRight(out.String(), "\n")
}
