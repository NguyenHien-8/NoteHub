//go:build !teststub

package sqlite

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
)

func TestAttachmentMigrationPreservesLegacyOrder(t *testing.T) {
	for _, version := range []int{1, 2} {
		t.Run(string(rune('0'+version)), func(t *testing.T) {
			path := legacyDatabase(t)
			db, err := sql.Open("sqlite", path)
			if err != nil {
				t.Fatal(err)
			}
			if version == 2 {
				if _, err := db.Exec(migrations[1].sql + `INSERT INTO schema_migration(version,name) VALUES(2,'memo favorites');`); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := db.Exec(`INSERT INTO attachment(id,uid,memo_id,created_ts,filename,mime_type,size,sha256,relative_path) VALUES
				(12,'oldest',42,10,'oldest.png','image/png',5,'digest','oldest.png'),
				(4,'tie-before',42,100,'before.png','image/png',5,'digest','before.png'),
				(11,'tie-after',42,100,'after.png','image/png',5,'digest','after.png');`); err != nil {
				t.Fatal(err)
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			for attempt := 0; attempt < 2; attempt++ {
				s, err := Open(context.Background(), path)
				if err != nil {
					t.Fatal(err)
				}
				rows, err := s.DB().Query(`SELECT id,sort_order FROM attachment WHERE memo_id=42 ORDER BY sort_order,id`)
				if err != nil {
					s.Close()
					t.Fatal(err)
				}
				var ids, positions []int64
				for rows.Next() {
					var id, position int64
					if err := rows.Scan(&id, &position); err != nil {
						t.Fatal(err)
					}
					ids = append(ids, id)
					positions = append(positions, position)
				}
				if err := rows.Err(); err != nil {
					t.Fatal(err)
				}
				rows.Close()
				if !reflect.DeepEqual(ids, []int64{12, 4, 9, 11}) || !reflect.DeepEqual(positions, []int64{0, 1, 2, 3}) {
					t.Fatalf("migrated order=%v positions=%v", ids, positions)
				}
				if err := s.Close(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
