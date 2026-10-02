package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"path/filepath"
)

// Store implements all NoteHub repository interfaces over one SQLite database.
type Store struct {
	db *sql.DB
}

func Open(ctx context.Context, dbPath string) (*Store, error) {
	if dbPath == "" {
		return nil, errors.New("database path is required")
	}
	abs, err := filepath.Abs(dbPath)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return nil, err
	}

	u := &url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}
	q := u.Query()
	// Adopt the SQLite lessons that matter for a desktop Memos-like app:
	// WAL for read/write concurrency, a busy handler instead of immediate BUSY
	// failures, foreign keys for local integrity, no mmap surprises, and
	// IMMEDIATE write transactions to avoid read->write upgrade races.
	q.Add("_pragma", "foreign_keys(ON)")
	q.Add("_pragma", "busy_timeout(10000)")
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "synchronous(NORMAL)")
	q.Add("_pragma", "mmap_size(0)")
	q.Set("_txlock", "immediate")
	u.RawQuery = q.Encode()

	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	s := &Store{db: db}
	if err := s.Migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }
func (s *Store) DB() *sql.DB  { return s.db }
