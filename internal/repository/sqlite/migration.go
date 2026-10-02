package sqlite

import (
	"context"
	"fmt"
)

type migration struct {
	version int
	name    string
	sql     string
}

var migrations = []migration{
	{version: 1, name: "initial desktop backend", sql: schemaV1},
}

func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migration (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_ts INTEGER NOT NULL DEFAULT (unixepoch())
	)`); err != nil {
		return err
	}
	for _, m := range migrations {
		var exists int
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migration WHERE version=?`, m.version).Scan(&exists); err != nil {
			return err
		}
		if exists != 0 {
			continue
		}
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, m.sql); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %d (%s): %w", m.version, m.name, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migration(version,name) VALUES(?,?)`, m.version, m.name); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return nil
}
