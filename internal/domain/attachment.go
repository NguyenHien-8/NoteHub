package domain

import "time"

// Attachment keeps metadata in SQLite while the payload remains on the local
// filesystem. RelativePath is never serialized to shared/export-facing JSON.
type Attachment struct {
	ID           int64     `json:"-"`
	UID          string    `json:"uid"`
	MemoID       int64     `json:"-"`
	Position     int       `json:"position"`
	CreatedAt    time.Time `json:"createdAt"`
	Filename     string    `json:"filename"`
	MIMEType     string    `json:"mimeType"`
	Size         int64     `json:"size"`
	SHA256       string    `json:"sha256"`
	RelativePath string    `json:"-"`
}
