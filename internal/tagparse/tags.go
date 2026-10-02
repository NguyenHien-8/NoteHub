package tagparse

import (
	"strings"
	"unicode"
)

// Extract implements the useful part of Memos' tag behavior without pulling in
// the web stack: tags are case-preserving, deduplicated case-insensitively, and
// hierarchical tags are expanded (#work/project -> work, work/project).
//
// Markdown code fences and inline code are ignored because a #token inside code
// should not become application metadata. Escaped hashtags are ignored too.
func Extract(content string) []string {
	var out []string
	seen := make(map[string]struct{})
	lines := strings.Split(content, "\n")
	inFence := false
	fenceChar := rune(0)
	fenceLen := 0

	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if n, ch := fencePrefix(trimmed); n >= 3 {
			if !inFence {
				inFence, fenceChar, fenceLen = true, ch, n
				continue
			}
			if ch == fenceChar && n >= fenceLen {
				inFence = false
				continue
			}
		}
		if inFence {
			continue
		}
		extractLine(line, &out, seen)
	}
	return out
}

func fencePrefix(s string) (int, rune) {
	runes := []rune(s)
	if len(runes) == 0 || (runes[0] != '`' && runes[0] != '~') {
		return 0, 0
	}
	ch := runes[0]
	n := 0
	for n < len(runes) && runes[n] == ch {
		n++
	}
	return n, ch
}

func extractLine(line string, out *[]string, seen map[string]struct{}) {
	runes := maskMarkdownLinks([]rune(line))
	inlineTicks := 0

	for i := 0; i < len(runes); {
		if runes[i] == '`' {
			n := 1
			for i+n < len(runes) && runes[i+n] == '`' {
				n++
			}
			if inlineTicks == 0 {
				inlineTicks = n
			} else if n == inlineTicks {
				inlineTicks = 0
			}
			i += n
			continue
		}
		if inlineTicks != 0 || runes[i] != '#' {
			i++
			continue
		}
		if i > 0 && runes[i-1] == '\\' {
			i++
			continue
		}
		if i > 0 && !isBoundary(runes[i-1]) {
			i++
			continue
		}
		j := i + 1
		if j >= len(runes) || !isTagStart(runes[j]) {
			i++
			continue
		}
		for j < len(runes) && isTagRune(runes[j]) {
			j++
		}
		value := strings.TrimRight(string(runes[i+1:j]), "/-")
		if value != "" {
			appendHierarchy(value, out, seen)
		}
		i = j
	}
}

func maskMarkdownLinks(runes []rune) []rune {
	out := append([]rune(nil), runes...)
	for i := 0; i < len(runes); i++ {
		if runes[i] != '[' || (i > 0 && runes[i-1] == '\\') {
			continue
		}
		closeBracket := -1
		for j := i + 1; j < len(runes); j++ {
			if runes[j] == ']' && (j == 0 || runes[j-1] != '\\') {
				closeBracket = j
				break
			}
		}
		if closeBracket < 0 || closeBracket+1 >= len(runes) {
			continue
		}
		end := -1
		if runes[closeBracket+1] == '(' {
			depth := 1
			for j := closeBracket + 2; j < len(runes); j++ {
				if runes[j] == '\\' {
					j++
					continue
				}
				switch runes[j] {
				case '(':
					depth++
				case ')':
					depth--
					if depth == 0 {
						end = j
						j = len(runes)
					}
				}
			}
		} else if runes[closeBracket+1] == '[' {
			for j := closeBracket + 2; j < len(runes); j++ {
				if runes[j] == ']' && (j == 0 || runes[j-1] != '\\') {
					end = j
					break
				}
			}
		}
		if end < 0 {
			continue
		}
		start := i
		if i > 0 && runes[i-1] == '!' {
			start = i - 1
		}
		for j := start; j <= end; j++ {
			out[j] = ' '
		}
		i = end
	}
	return out
}

func isBoundary(r rune) bool {
	return unicode.IsSpace(r) || strings.ContainsRune("([{<>:;,.!?\"'", r)
}

func isTagStart(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func isTagRune(r rune) bool {
	return isTagStart(r) || r == '-' || r == '/'
}

func appendHierarchy(value string, out *[]string, seen map[string]struct{}) {
	parts := strings.Split(value, "/")
	current := ""
	for _, part := range parts {
		if part == "" {
			break
		}
		if current == "" {
			current = part
		} else {
			current += "/" + part
		}
		key := strings.ToLower(current)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		*out = append(*out, current)
	}
}
