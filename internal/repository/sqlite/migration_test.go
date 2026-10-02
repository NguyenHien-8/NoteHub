//go:build !teststub

package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"testing"

	"github.com/NguyenHien-8/NoteHub/internal/repository"
)

// Build an actual released v1 database, rather than creating it through Open,
// which always applies the newest schema.
func legacyDatabase(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "legacy.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(schemaV1); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		CREATE TABLE schema_migration(version INTEGER PRIMARY KEY,name TEXT NOT NULL,applied_ts INTEGER NOT NULL DEFAULT (unixepoch()));
		INSERT INTO schema_migration(version,name) VALUES(1,'initial desktop backend');
		INSERT INTO memo(id,uid,content,created_ts,updated_ts,revision) VALUES(42,'legacy-memo','Ghi chú #FPGA',100,200,7);
		INSERT INTO memo_tag(memo_id,tag,tag_norm) VALUES(42,'FPGA','fpga');
		INSERT INTO attachment(id,uid,memo_id,created_ts,filename,mime_type,size,sha256,relative_path)
		VALUES(9,'legacy-attachment',42,100,'hello.txt','text/plain',5,'digest','legacy/file.txt');
		INSERT INTO memo_share(uid,memo_id,token_hash,created_ts) VALUES('legacy-share',42,x'01',100);
	`); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestConcurrentMigrationAppliesV2Once(t *testing.T) {
	ctx := context.Background()
	path := legacyDatabase(t)
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(10000)&_txlock=immediate")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(8)
	s := &Store{db: db}
	start := make(chan struct{})
	errors := make(chan error, 8)
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			errors <- s.Migrate(ctx)
		}()
	}
	close(start)
	workers.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Error(err)
		}
	}
}

func TestMigrationV1RetainsMemoTagsAttachmentsSharesAndFTS(t *testing.T) {
	ctx := context.Background()
	path := legacyDatabase(t)
	for i := 0; i < 2; i++ {
		s, err := Open(ctx, path)
		if err != nil {
			t.Fatal(err)
		}
		m, err := s.GetMemoByID(ctx, 42)
		if err != nil {
			s.Close()
			t.Fatal(err)
		}
		if m.Content != "Ghi chú #FPGA" || m.Revision != 7 || m.Favorite || m.CreatedAt.Unix() != 100 || m.UpdatedAt.Unix() != 200 {
			s.Close()
			t.Fatalf("migration changed memo: %+v", m)
		}
		tags, err := s.TagsByMemoIDs(ctx, []int64{42})
		if err != nil || len(tags[42]) != 1 || tags[42][0] != "FPGA" {
			s.Close()
			t.Fatalf("tags=%v err=%v", tags, err)
		}
		attachments, err := s.ListAllAttachments(ctx, 10, 0)
		if err != nil || len(attachments) != 1 || attachments[0].ID != 9 || attachments[0].RelativePath != "legacy/file.txt" {
			s.Close()
			t.Fatalf("attachments=%v err=%v", attachments, err)
		}
		shares, err := s.ListSharesByMemoID(ctx, 42)
		if err != nil || len(shares) != 1 || shares[0].UID != "legacy-share" {
			s.Close()
			t.Fatalf("shares=%v err=%v", shares, err)
		}
		found, err := s.SearchMemos(ctx, repository.SearchQuery{Text: "ghi chu"})
		if err != nil || len(found) != 1 || found[0].ID != 42 {
			s.Close()
			t.Fatalf("fts=%v err=%v", found, err)
		}
		var version, count int
		if err := s.DB().QueryRowContext(ctx, `SELECT MAX(version),COUNT(*) FROM schema_migration`).Scan(&version, &count); err != nil {
			s.Close()
			t.Fatal(err)
		}
		if version != 2 || count != 2 {
			s.Close()
			t.Fatalf("migration version=%d count=%d", version, count)
		}
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
