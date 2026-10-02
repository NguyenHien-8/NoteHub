package platform

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type DataPaths struct {
	Root        string
	Database    string
	Attachments string
}

func ResolveDataPaths(appName string) (DataPaths, error) {
	appName = strings.TrimSpace(appName)
	if appName == "" {
		appName = "NoteHub"
	}
	root, err := defaultDataDir(appName)
	if err != nil {
		return DataPaths{}, err
	}
	if root == "" {
		return DataPaths{}, errors.New("empty application data directory")
	}
	p := DataPaths{
		Root:        root,
		Database:    filepath.Join(root, "notehub.db"),
		Attachments: filepath.Join(root, "attachments"),
	}
	if err := os.MkdirAll(p.Attachments, 0o755); err != nil {
		return DataPaths{}, err
	}
	return p, nil
}
