// Package notecontent extracts searchable text from the versioned rich-text
// envelope. Legacy Markdown remains byte-for-byte unchanged.
package notecontent

import (
	"golang.org/x/net/html"
	"strings"
)

const RichPrefix = "<!-- NoteHub rich text v1 -->\n"

func Plain(content string) string {
	if !strings.HasPrefix(content, RichPrefix) {
		return content
	}
	root, err := html.Parse(strings.NewReader(strings.TrimPrefix(content, RichPrefix)))
	if err != nil {
		return content
	}
	var out strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "head" || n.Data == "style" || n.Data == "script") {
			return
		}
		if n.Type == html.TextNode {
			out.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if n.Type == html.ElementNode {
			switch n.Data {
			case "p", "div", "br", "li", "h1", "h2", "h3", "tr", "pre":
				out.WriteByte('\n')
			}
		}
	}
	walk(root)
	return strings.TrimSpace(out.String())
}

// TagSource keeps code blocks excluded under the existing Markdown tag rules.
func TagSource(content string) string {
	if !strings.HasPrefix(content, RichPrefix) {
		return content
	}
	root, err := html.Parse(strings.NewReader(strings.TrimPrefix(content, RichPrefix)))
	if err != nil {
		return ""
	}
	var out strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "head", "style", "script", "pre", "code":
				return
			}
		}
		if n.Type == html.TextNode {
			out.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if n.Type == html.ElementNode {
			switch n.Data {
			case "p", "div", "br", "li", "h1", "h2", "h3", "tr":
				out.WriteByte('\n')
			}
		}
	}
	walk(root)
	return out.String()
}
