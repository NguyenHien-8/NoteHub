package backend

import (
	"os"
	"path/filepath"
	"runtime"
)

// Config describes where the desktop backend keeps its data.
// DataDir should be an OS-specific application data directory.
type Config struct {
	DataDir string
	DBFile  string
}

func (c Config) withDefaults() Config {
	if c.DBFile == "" {
		c.DBFile = "notes.db"
	}
	return c
}

func (c Config) dbPath() string {
	return filepath.Join(c.DataDir, c.DBFile)
}

func (c Config) attachmentRoot() string {
	return filepath.Join(c.DataDir, "attachments")
}

// DefaultDataDir returns a sensible per-user directory for a desktop app.
// On Windows it prefers LOCALAPPDATA; on macOS/Linux it falls back to
// os.UserConfigDir. Callers may always override this with Config.DataDir.
func DefaultDataDir(appName string) (string, error) {
	if appName == "" {
		appName = "TimelineNotes"
	}
	if runtime.GOOS == "windows" {
		if base := os.Getenv("LOCALAPPDATA"); base != "" {
			return filepath.Join(base, appName), nil
		}
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appName), nil
}
