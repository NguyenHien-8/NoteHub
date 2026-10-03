package components

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestMemoPreviewPreservesUnicodeAndFirstNonemptyLine(t *testing.T) {
	title, body := memoPreview("\n  \nKiểm tra FPGA 🧪\nGhi chú tiếng Việt\nDòng tiếp theo")
	if title != "Kiểm tra FPGA 🧪" || body != "Ghi chú tiếng Việt\nDòng tiếp theo" {
		t.Fatalf("preview = %q / %q", title, body)
	}
	title, body = memoPreview(strings.Repeat("ế", 300))
	if !utf8.ValidString(title) || len([]rune(title)) > 90 || body != "" {
		t.Fatalf("bounded Unicode preview = %q / %q", title, body)
	}
}

func TestTimelineGroupingUsesLocalDatesAcrossDST(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 3, 9, 0, 15, 0, 0, loc)
	for _, tc := range []struct {
		at   time.Time
		want string
	}{
		{now, "Today"},
		{time.Date(2026, 3, 8, 0, 30, 0, 0, loc), "Yesterday"},
		{time.Date(2026, 3, 7, 23, 59, 0, 0, loc), "Earlier"},
	} {
		if got := timelineGroup(tc.at.UTC(), now); got != tc.want {
			t.Fatalf("group = %q, want %q", got, tc.want)
		}
	}
}

func TestMondayFirstCalendarHandlesLeapYearAndSundayStart(t *testing.T) {
	cells := monthCells(2026, time.March)
	if len(cells) != 42 || cells[6] != 1 || cells[36] != 31 {
		t.Fatalf("March grid = %v", cells)
	}
	cells = monthCells(2024, time.February)
	if cells[3] != 1 || cells[31] != 29 || cells[32] != 0 {
		t.Fatalf("leap grid = %v", cells)
	}
}

func TestTagColorsAreStableAndPastel(t *testing.T) {
	first := tagColor("Research/FPGA")
	if first != tagColor("Research/FPGA") || first.A != 255 || first.R < 200 || first.G < 200 || first.B < 200 {
		t.Fatalf("not a stable pastel: %#v", first)
	}
	seen := map[interface{}]bool{}
	for _, name := range []string{"FPGA", "ESP32", "Research", "Study", "Project/NoteHub"} {
		seen[tagColor(name)] = true
	}
	if len(seen) < 3 {
		t.Fatalf("only %d colors", len(seen))
	}
}
