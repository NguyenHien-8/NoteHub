package components

import (
	"image/png"
	"os"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
)

func TestTemporaryPreview(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	test.ApplyTheme(t, theme.LightTheme())
	s := NewSidebar(nil, nil, nil)
	s.SetTags([]domain.TagCount{{Tag: "FPGA", Count: 4}, {Tag: "Research/ESP32", Count: 12}, {Tag: "Study", Count: 3}})
	c := NewComposer(nil, nil, nil)
	tl := NewTimelineList()
	tl.SetMemos([]domain.Memo{{ID: 1, Content: "Kiểm tra FPGA Cyclone IV\nKiểm tra clock và timing constraint trong thiết kế mới.", CreatedAt: time.Now(), Tags: []string{"FPGA", "Research"}}, {ID: 2, Content: "Ghi chú cho tuần mới\nĐọc tài liệu và chuẩn bị các thí nghiệm tiếp theo.\nNhững ý tưởng nhỏ sẽ giúp công việc rõ ràng hơn.", CreatedAt: time.Now(), Tags: []string{"Study"}}}, nil, MemoActions{})
	cal := NewCalendar(nil, nil)
	cal.SetMonth(2026, time.October, []domain.CalendarDay{{Date: "2026-10-02", Count: 2}}, "2026-10-02")
	left := container.NewGridWrap(fyne.NewSize(240, 770), s.Object)
	right := container.NewGridWrap(fyne.NewSize(280, 480), cal.Object)
	center := container.NewBorder(c.Object, nil, nil, nil, tl.Object)
	content := container.New(layout.NewCustomPaddedLayout(20, 20, 20, 20), container.NewBorder(container.NewBorder(nil, nil, widget.NewLabelWithStyle("NoteHub", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), nil, NewSearchBar(nil)), nil, left, right, center))
	w := test.NewWindow(content)
	defer w.Close()
	w.Resize(fyne.NewSize(1450, 900))
	f, err := os.Create("component-preview.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err = png.Encode(f, w.Canvas().Capture()); err != nil {
		t.Fatal(err)
	}
}
