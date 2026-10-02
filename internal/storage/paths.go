package storage

import (
	"errors"
	"path/filepath"
	"strings"
	"unicode"
)

func ensureInside(root, relative string) (string, error) {
	if filepath.IsAbs(relative) {
		return "", errors.New("absolute storage path is not allowed")
	}
	clean := filepath.Clean(relative)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("storage path escapes attachment root")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	full, err := filepath.Abs(filepath.Join(rootAbs, clean))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootAbs, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("storage path escapes attachment root")
	}
	return full, nil
}

func safeExtension(filename string) string {
	ext := filepath.Ext(filepath.Base(filename))
	if len(ext) > 17 { // '.' plus at most 16 characters.
		return ""
	}
	for i, r := range ext {
		if i == 0 && r == '.' {
			continue
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return ""
		}
	}
	return strings.ToLower(ext)
}
