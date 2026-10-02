package tagparse

import (
	"strings"
	"unicode"
)

// Extract returns hashtag names without '#'. Hierarchical tags are expanded:
// #work/project => ["work", "work/project"]. Tags in fenced code blocks and
// inline backticks are ignored. Deduplication is case-insensitive.
func Extract(content string) []string {
	var tags []string
	seen := map[string]struct{}{}
	lines := strings.Split(content, "\n")
	inFence := false
	fenceMark := ""
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			mark := trimmed[:3]
			if !inFence {
				inFence, fenceMark = true, mark
			} else if mark == fenceMark {
				inFence, fenceMark = false, ""
			}
			continue
		}
		if inFence {
			continue
		}
		extractLine(line, &tags, seen)
	}
	return tags
}

func extractLine(line string, tags *[]string, seen map[string]struct{}) {
	r := []rune(line)
	inlineCode := false
	for i := 0; i < len(r); i++ {
		if r[i] == '`' {
			inlineCode = !inlineCode
			continue
		}
		if inlineCode || r[i] != '#' {
			continue
		}
		if i > 0 && r[i-1] == '\\' {
			continue
		}
		if i > 0 && !isBoundary(r[i-1]) {
			continue
		}
		j := i + 1
		if j >= len(r) || !isTagStart(r[j]) {
			continue
		}
		for j < len(r) && isTagRune(r[j]) {
			j++
		}
		value := strings.TrimRight(string(r[i+1:j]), "/-")
		if value == "" {
			continue
		}
		appendHierarchy(value, tags, seen)
		i = j - 1
	}
}

func isBoundary(r rune) bool {
	return unicode.IsSpace(r) || strings.ContainsRune("([{<>:;,.!?", r)
}

func isTagStart(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' }
func isTagRune(r rune) bool  { return isTagStart(r) || r == '-' || r == '/' }

func appendHierarchy(value string, tags *[]string, seen map[string]struct{}) {
	parts := strings.Split(value, "/")
	current := ""
	for _, p := range parts {
		if p == "" {
			break
		}
		if current == "" {
			current = p
		} else {
			current += "/" + p
		}
		key := strings.ToLower(current)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		*tags = append(*tags, current)
	}
}
