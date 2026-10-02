package backup

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMemoRecordFavoriteIsOptionalAndBackwardCompatible(t *testing.T) {
	var legacy MemoRecord
	if err := json.Unmarshal([]byte(`{"uid":"legacy","contentPath":"memos/legacy.md"}`), &legacy); err != nil {
		t.Fatal(err)
	}
	if legacy.Favorite {
		t.Fatal("legacy record defaulted to favorite")
	}
	encoded, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), `"favorite"`) {
		t.Fatalf("false favorite was not omitted: %s", encoded)
	}
	legacy.Favorite = true
	encoded, err = json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	var restored MemoRecord
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	if !restored.Favorite || FormatVersion != "1.0" {
		t.Fatalf("restored=%+v format=%s", restored, FormatVersion)
	}
}
