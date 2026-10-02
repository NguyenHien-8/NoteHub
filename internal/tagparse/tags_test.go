package tagparse

import (
	"reflect"
	"testing"
)

func TestExtract(t *testing.T) {
	input := "Hello #FPGA #work/project #tiếngViệt\n`#inline`\n````go\n#fenced\n````\n\\#escaped [#inside](https://x/#url) ![#image](img.png) #FPGA"
	want := []string{"FPGA", "work", "work/project", "tiếngViệt"}
	if got := Extract(input); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
