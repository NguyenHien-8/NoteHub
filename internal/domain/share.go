package domain

import "time"

// Share is metadata for a read-only bearer grant. The actual bearer token is
// returned only at creation time; SQLite stores only its SHA-256 digest.
type Share struct {
	ID        int64      `json:"-"`
	UID       string     `json:"uid"`
	MemoID    int64      `json:"-"`
	CreatedAt time.Time  `json:"createdAt"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

type ShareGrant struct {
	Share Share  `json:"share"`
	Token string `json:"token"`
}

type SharedMemo struct {
	Share Share `json:"share"`
	Memo  Memo  `json:"memo"`
}
