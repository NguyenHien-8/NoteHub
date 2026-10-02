package domain

import "time"

// Memo is the core NoteHub note entity. Database-only IDs are intentionally
// excluded from JSON so a shared memo never leaks local implementation details.
type Memo struct {
	ID          int64        `json:"-"`
	UID         string       `json:"uid"`
	Content     string       `json:"content"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
	Revision    int64        `json:"revision"`
	Tags        []string     `json:"tags,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}
