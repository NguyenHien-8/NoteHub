package components

import (
	"fmt"
	"hash/fnv"
	"image/color"
	"strings"
	"time"
)

func boundedText(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit-1]) + "…"
}

func memoPreview(content string) (string, string) {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	for i, line := range lines {
		if title := strings.TrimSpace(line); title != "" {
			bodyLines := lines[i+1:]
			if len(bodyLines) > 3 {
				bodyLines = bodyLines[:3]
			}
			return boundedText(title, 90), boundedText(strings.TrimSpace(strings.Join(bodyLines, "\n")), 260)
		}
	}
	return "Attachment note", ""
}

func timelineGroup(created, now time.Time) string {
	local := created.In(now.Location())
	if local.Format("2006-01-02") == now.Format("2006-01-02") {
		return "Today"
	}
	if local.Format("2006-01-02") == now.AddDate(0, 0, -1).Format("2006-01-02") {
		return "Yesterday"
	}
	return "Earlier"
}

func monthCells(year int, month time.Month) []int {
	first := time.Date(year, month, 1, 12, 0, 0, 0, time.Local)
	offset := (int(first.Weekday()) + 6) % 7
	last := first.AddDate(0, 1, -1).Day()
	rows := (offset + last + 6) / 7
	cells := make([]int, rows*7)
	for day := 1; day <= last; day++ {
		cells[offset+day-1] = day
	}
	return cells
}

func tagColor(tag string) color.NRGBA {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(strings.TrimPrefix(tag, "#")))
	palette := []color.NRGBA{
		{224, 236, 255, 255}, {225, 244, 235, 255}, {241, 230, 255, 255},
		{255, 236, 220, 255}, {255, 227, 237, 255}, {220, 242, 247, 255},
	}
	return palette[hash.Sum32()%uint32(len(palette))]
}

func fileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	}
	if size < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(size)/(1024*1024*1024))
}
