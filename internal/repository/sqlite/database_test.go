//go:build !teststub

package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenPathWithSpacesUnicodeAndReservedURICharacters(t *testing.T) {
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "Ghi chú #100%")
	path := filepath.Join(dir, "note hub.db")
	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB().ExecContext(ctx, `INSERT INTO memo(uid,content,created_ts,updated_ts) VALUES('persisted','hello',1,1)`); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	s, err = Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var text string
	if err := s.DB().QueryRowContext(ctx, `SELECT content FROM memo WHERE uid='persisted'`).Scan(&text); err != nil {
		t.Fatal(err)
	}
	if text != "hello" {
		t.Fatalf("content=%q", text)
	}
}
