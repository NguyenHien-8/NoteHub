package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"example.com/memosdesktop/backend/backend"
)

func main() {
	dataDir := filepath.Join(os.TempDir(), "timeline-notes-demo")
	ctx := context.Background()
	svc, err := backend.Open(ctx, backend.Config{DataDir: dataDir})
	if err != nil {
		log.Fatal(err)
	}
	defer svc.Close()

	memo, err := svc.CreateMemo(ctx, "Ghi chú đầu tiên #demo #project/memos")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created memo %d %s tags=%v\n", memo.ID, memo.UID, memo.Tags)
}
