//go:build windows

package platform

import (
	"os"
	"path/filepath"
)

func defaultDataDir(appName string) (string, error) {
	if base := os.Getenv("LOCALAPPDATA"); base != "" {
		return filepath.Join(base, appName), nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appName), nil
}
