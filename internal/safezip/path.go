package safezip

import (
	"errors"
	"path"
	"strings"
)

func ValidateName(name string) error {
	if name == "" || strings.ContainsRune(name, '\\') {
		return errors.New("empty or backslash path")
	}
	if strings.HasPrefix(name, "/") {
		return errors.New("absolute path")
	}
	clean := path.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return errors.New("path traversal")
	}
	if clean != name && clean+"/" != name {
		return errors.New("non-canonical path")
	}
	return nil
}
