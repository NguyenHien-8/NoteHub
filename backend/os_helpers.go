package backend

import (
	"os"
	"path/filepath"
)

func osRemove(path string) error { return os.Remove(path) }
func pathDir(path string) string { return filepath.Dir(path) }
func osPathSeparator() rune      { return filepath.Separator }
