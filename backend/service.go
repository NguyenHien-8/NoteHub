package backend

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Service is the pure-Go backend intended to be called by a desktop UI.
// It owns SQLite and local attachment storage; it has no web UI dependency.
type Service struct {
	db             *sql.DB
	dataDir        string
	attachmentRoot string
}

func Open(ctx context.Context, cfg Config) (*Service, error) {
	cfg = cfg.withDefaults()
	if cfg.DataDir == "" {
		return nil, errors.New("DataDir is required")
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	if err := os.MkdirAll(cfg.attachmentRoot(), 0o755); err != nil {
		return nil, fmt.Errorf("create attachment dir: %w", err)
	}

	dsn := filepath.Clean(cfg.dbPath()) + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=mmap_size(0)&_txlock=immediate"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if _, err := db.ExecContext(ctx, schemaSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize schema: %w", err)
	}
	return &Service{db: db, dataDir: cfg.DataDir, attachmentRoot: cfg.attachmentRoot()}, nil
}

func (s *Service) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func unix(t time.Time) int64     { return t.UTC().Unix() }
func fromUnix(v int64) time.Time { return time.Unix(v, 0).UTC() }
