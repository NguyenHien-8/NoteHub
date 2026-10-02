package backup

import (
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/NguyenHien-8/NoteHub/internal/identity"
)

const (
	Format         = "notehub-export"
	FormatVersion  = "1.0"
	ManifestEntry  = "manifest.json"
	MemosDir       = "memos/"
	AttachmentsDir = "attachments/"
)

type Generator struct{ Name, Version string }
type Counts struct{ Memos, Attachments int }
type Manifest struct {
	Format        string    `json:"format"`
	FormatVersion string    `json:"formatVersion"`
	Generator     Generator `json:"generator"`
	ExportTime    string    `json:"exportTime"`
	Counts        Counts    `json:"counts"`
}

type MemoRecord struct {
	UID         string             `json:"uid"`
	CreateTime  string             `json:"createTime"`
	UpdateTime  string             `json:"updateTime"`
	ContentPath string             `json:"contentPath"`
	Attachments []AttachmentRecord `json:"attachments,omitempty"`
}

type AttachmentRecord struct {
	UID        string `json:"uid"`
	Filename   string `json:"filename"`
	Type       string `json:"type"`
	Size       int64  `json:"size"`
	SHA256     string `json:"sha256"`
	CreateTime string `json:"createTime"`
	Path       string `json:"path"`
}

func FormatTime(t time.Time) string { return t.UTC().Truncate(time.Second).Format(time.RFC3339) }
func ParseTime(v string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}
func ContentPath(uid string) string { return path.Join(MemosDir, uid+".md") }
func RecordPath(uid string) string  { return path.Join(MemosDir, uid+".json") }
func AttachmentPath(uid, filename string) string {
	return path.Join(AttachmentsDir, uid, SafeFilename(filename))
}

func validateUID(uid string) error {
	if !identity.ValidUID(uid) {
		return fmt.Errorf("invalid uid %q", uid)
	}
	return nil
}

func parseFormatVersion(v string) (int, int, error) {
	majorText, minorText, ok := strings.Cut(v, ".")
	if !ok || majorText == "" || minorText == "" || strings.Contains(minorText, ".") {
		return 0, 0, fmt.Errorf("formatVersion %q is not MAJOR.MINOR", v)
	}
	major, err := strconv.Atoi(majorText)
	if err != nil || major < 0 {
		return 0, 0, fmt.Errorf("invalid major format version %q", v)
	}
	minor, err := strconv.Atoi(minorText)
	if err != nil || minor < 0 {
		return 0, 0, fmt.Errorf("invalid minor format version %q", v)
	}
	return major, minor, nil
}
