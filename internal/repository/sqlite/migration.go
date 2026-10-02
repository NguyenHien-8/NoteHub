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
	{version: 2, name: "memo favorites", sql: `
		ALTER TABLE memo ADD COLUMN favorite INTEGER NOT NULL DEFAULT 0 CHECK (favorite IN (0,1));
		CREATE INDEX idx_memo_favorites ON memo(created_ts DESC,id DESC) WHERE favorite=1;
	`},
	{version: 3, name: "attachment order", sql: `
		ALTER TABLE attachment ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0 CHECK (sort_order >= 0);
		WITH ranked AS (
			SELECT id,ROW_NUMBER() OVER (PARTITION BY memo_id ORDER BY created_ts ASC,id ASC)-1 AS position
			FROM attachment
		)
		UPDATE attachment SET sort_order=(SELECT position FROM ranked WHERE ranked.id=attachment.id);
		CREATE INDEX idx_attachment_order ON attachment(memo_id,sort_order,id);
	`},
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
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		// Check under the IMMEDIATE transaction's writer lock. Concurrent
		// openers must see the committed version before attempting ALTER TABLE.
		var exists int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migration WHERE version=?`, m.version).Scan(&exists); err != nil {
			_ = tx.Rollback()
			return err
		}
		if exists != 0 {
			_ = tx.Rollback()
			continue
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
