package backend

import "time"

type Memo struct {
	ID          int64        `json:"id"`
	UID         string       `json:"uid"`
	Content     string       `json:"content"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
	Tags        []string     `json:"tags"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type Attachment struct {
	ID           int64     `json:"id"`
	UID          string    `json:"uid"`
	MemoID       int64     `json:"memoId"`
	CreatedAt    time.Time `json:"createdAt"`
	Filename     string    `json:"filename"`
	MIMEType     string    `json:"mimeType"`
	Size         int64     `json:"size"`
	SHA256       string    `json:"sha256"`
	RelativePath string    `json:"relativePath"`
}

type Share struct {
	ID        int64      `json:"id"`
	Token     string     `json:"token"`
	MemoID    int64      `json:"memoId"`
	CreatedAt time.Time  `json:"createdAt"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

type SharedMemo struct {
	Share Share `json:"share"`
	Memo  Memo  `json:"memo"`
}

type TimelineQuery struct {
	Limit  int
	Offset int
	From   *time.Time
	To     *time.Time
	Tags   []string
}

type SearchQuery struct {
	Text   string
	Tags   []string
	From   *time.Time
	To     *time.Time
	Limit  int
	Offset int
}

type CalendarDay struct {
	Date  string `json:"date"` // YYYY-MM-DD in the requested location.
	Count int    `json:"count"`
}

type ImportPolicy string

const (
	ImportSkip      ImportPolicy = "skip"
	ImportOverwrite ImportPolicy = "overwrite"
)

type ImportReport struct {
	Created     int      `json:"created"`
	Updated     int      `json:"updated"`
	Skipped     int      `json:"skipped"`
	Attachments int      `json:"attachments"`
	Warnings    []string `json:"warnings,omitempty"`
}
