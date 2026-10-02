//go:build darwin

package platform

import (
	"os"
	"path/filepath"
)

func defaultDataDir(appName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Application Support", appName), nil
}
