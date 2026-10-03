package typography

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
)

const (
	DefaultTextSize float32 = 14
	MinTextSize     float32 = 10
	MaxTextSize     float32 = 32
)

type fontSpec struct {
	name                              string
	regular, bold, italic, boldItalic []string
}

type fontBundle struct {
	regular, bold, italic, boldItalic fyne.Resource
}

var (
	fontCacheMu sync.Mutex
	fontCache   = make(map[string]*fontBundle)
)

// ClampTextSize keeps global typography in a range that remains usable on
// compact desktop windows while still allowing deliberate large-text modes.
func ClampTextSize(size float32) float32 {
	if size < MinTextSize {
		return MinTextSize
	}
	if size > MaxTextSize {
		return MaxTextSize
	}
	return size
}

// IsKnownFamily validates persisted preference values without depending on
// whether the font happens to be installed on the current machine.
func IsKnownFamily(name string) bool {
	if strings.EqualFold(name, "System") || strings.EqualFold(name, "Monospace") {
		return true
	}
	for _, spec := range platformSpecs() {
		if strings.EqualFold(spec.name, name) {
			return true
		}
	}
	return false
}

// AvailableFamilies returns only selectable families that can be resolved on
// this machine. NoteHub never bundles or redistributes system font files.
func AvailableFamilies() []string {
	families := []string{"System", "Monospace"}
	for _, spec := range platformSpecs() {
		if firstExisting(spec.regular) != "" {
			families = append(families, spec.name)
		}
	}
	return families
}

// Resolve loads an installed font family lazily. Missing style files inherit
// the family's regular face; a completely unavailable family returns nil so
// the caller can fall back to Fyne's system font.
func Resolve(family string, style fyne.TextStyle) fyne.Resource {
	if strings.EqualFold(family, "System") || strings.EqualFold(family, "Monospace") {
		return nil
	}
	key := strings.ToLower(strings.TrimSpace(family))
	fontCacheMu.Lock()
	bundle, ok := fontCache[key]
	fontCacheMu.Unlock()
	if !ok {
		bundle = loadBundle(family)
		fontCacheMu.Lock()
		fontCache[key] = bundle
		fontCacheMu.Unlock()
	}
	if bundle == nil || bundle.regular == nil {
		return nil
	}
	if style.Bold && style.Italic && bundle.boldItalic != nil {
		return bundle.boldItalic
	}
	if style.Bold && bundle.bold != nil {
		return bundle.bold
	}
	if style.Italic && bundle.italic != nil {
		return bundle.italic
	}
	return bundle.regular
}

func loadBundle(family string) *fontBundle {
	var spec *fontSpec
	for _, candidate := range platformSpecs() {
		candidate := candidate
		if strings.EqualFold(candidate.name, family) {
			spec = &candidate
			break
		}
	}
	if spec == nil {
		return nil
	}
	regular := loadFirst(spec.regular)
	if regular == nil {
		return nil
	}
	bold := loadFirst(spec.bold)
	italic := loadFirst(spec.italic)
	boldItalic := loadFirst(spec.boldItalic)
	if bold == nil {
		bold = regular
	}
	if italic == nil {
		italic = regular
	}
	if boldItalic == nil {
		if bold != nil {
			boldItalic = bold
		} else if italic != nil {
			boldItalic = italic
		} else {
			boldItalic = regular
		}
	}
	return &fontBundle{regular: regular, bold: bold, italic: italic, boldItalic: boldItalic}
}

func loadFirst(paths []string) fyne.Resource {
	path := firstExisting(paths)
	if path == "" {
		return nil
	}
	resource, err := fyne.LoadResourceFromPath(path)
	if err != nil {
		return nil
	}
	return resource
}

func firstExisting(paths []string) string {
	for _, path := range paths {
		if path == "" {
			continue
		}
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}

func platformSpecs() []fontSpec {
	switch runtime.GOOS {
	case "windows":
		root := os.Getenv("WINDIR")
		if root == "" {
			root = `C:\Windows`
		}
		fonts := filepath.Join(root, "Fonts")
		p := func(names ...string) []string {
			out := make([]string, 0, len(names))
			for _, name := range names {
				out = append(out, filepath.Join(fonts, name))
			}
			return out
		}
		return []fontSpec{
			{name: "Segoe UI", regular: p("segoeui.ttf"), bold: p("segoeuib.ttf"), italic: p("segoeuii.ttf"), boldItalic: p("segoeuiz.ttf")},
			{name: "Arial", regular: p("arial.ttf"), bold: p("arialbd.ttf"), italic: p("ariali.ttf"), boldItalic: p("arialbi.ttf")},
			{name: "Calibri", regular: p("calibri.ttf"), bold: p("calibrib.ttf"), italic: p("calibrii.ttf"), boldItalic: p("calibriz.ttf")},
			{name: "Times New Roman", regular: p("times.ttf"), bold: p("timesbd.ttf"), italic: p("timesi.ttf"), boldItalic: p("timesbi.ttf")},
			{name: "Georgia", regular: p("georgia.ttf"), bold: p("georgiab.ttf"), italic: p("georgiai.ttf"), boldItalic: p("georgiaz.ttf")},
			{name: "Verdana", regular: p("verdana.ttf"), bold: p("verdanab.ttf"), italic: p("verdanai.ttf"), boldItalic: p("verdanaz.ttf")},
			{name: "Tahoma", regular: p("tahoma.ttf"), bold: p("tahomabd.ttf")},
			{name: "Trebuchet MS", regular: p("trebuc.ttf"), bold: p("trebucbd.ttf"), italic: p("trebucit.ttf"), boldItalic: p("trebucbi.ttf")},
			{name: "Consolas", regular: p("consola.ttf"), bold: p("consolab.ttf"), italic: p("consolai.ttf"), boldItalic: p("consolaz.ttf")},
			{name: "Courier New", regular: p("cour.ttf"), bold: p("courbd.ttf"), italic: p("couri.ttf"), boldItalic: p("courbi.ttf")},
		}
	case "darwin":
		home, _ := os.UserHomeDir()
		dirs := []string{"/System/Library/Fonts", "/Library/Fonts", filepath.Join(home, "Library", "Fonts")}
		p := func(names ...string) []string { return joinCandidates(dirs, names) }
		return []fontSpec{
			{name: "Helvetica Neue", regular: p("HelveticaNeue.ttc", "Helvetica.ttc")},
			{name: "Arial", regular: p("Arial.ttf"), bold: p("Arial Bold.ttf"), italic: p("Arial Italic.ttf"), boldItalic: p("Arial Bold Italic.ttf")},
			{name: "Avenir Next", regular: p("Avenir Next.ttc")},
			{name: "Georgia", regular: p("Georgia.ttf"), bold: p("Georgia Bold.ttf"), italic: p("Georgia Italic.ttf")},
			{name: "Times New Roman", regular: p("Times New Roman.ttf"), bold: p("Times New Roman Bold.ttf"), italic: p("Times New Roman Italic.ttf")},
			{name: "Menlo", regular: p("Menlo.ttc")},
			{name: "Monaco", regular: p("Monaco.ttf")},
		}
	default:
		home, _ := os.UserHomeDir()
		dirs := []string{
			"/usr/share/fonts/truetype/dejavu", "/usr/share/fonts/truetype/liberation2", "/usr/share/fonts/truetype/liberation",
			"/usr/share/fonts/truetype/noto", "/usr/share/fonts/opentype/noto", "/usr/local/share/fonts",
			filepath.Join(home, ".fonts"), filepath.Join(home, ".local", "share", "fonts"),
		}
		p := func(names ...string) []string { return joinCandidates(dirs, names) }
		return []fontSpec{
			{name: "Noto Sans", regular: p("NotoSans-Regular.ttf"), bold: p("NotoSans-Bold.ttf"), italic: p("NotoSans-Italic.ttf"), boldItalic: p("NotoSans-BoldItalic.ttf")},
			{name: "DejaVu Sans", regular: p("DejaVuSans.ttf"), bold: p("DejaVuSans-Bold.ttf"), italic: p("DejaVuSans-Oblique.ttf"), boldItalic: p("DejaVuSans-BoldOblique.ttf")},
			{name: "Liberation Sans", regular: p("LiberationSans-Regular.ttf"), bold: p("LiberationSans-Bold.ttf"), italic: p("LiberationSans-Italic.ttf"), boldItalic: p("LiberationSans-BoldItalic.ttf")},
			{name: "Noto Serif", regular: p("NotoSerif-Regular.ttf"), bold: p("NotoSerif-Bold.ttf"), italic: p("NotoSerif-Italic.ttf"), boldItalic: p("NotoSerif-BoldItalic.ttf")},
			{name: "DejaVu Serif", regular: p("DejaVuSerif.ttf"), bold: p("DejaVuSerif-Bold.ttf"), italic: p("DejaVuSerif-Italic.ttf"), boldItalic: p("DejaVuSerif-BoldItalic.ttf")},
			{name: "Liberation Serif", regular: p("LiberationSerif-Regular.ttf"), bold: p("LiberationSerif-Bold.ttf"), italic: p("LiberationSerif-Italic.ttf"), boldItalic: p("LiberationSerif-BoldItalic.ttf")},
			{name: "DejaVu Sans Mono", regular: p("DejaVuSansMono.ttf"), bold: p("DejaVuSansMono-Bold.ttf"), italic: p("DejaVuSansMono-Oblique.ttf"), boldItalic: p("DejaVuSansMono-BoldOblique.ttf")},
			{name: "Liberation Mono", regular: p("LiberationMono-Regular.ttf"), bold: p("LiberationMono-Bold.ttf"), italic: p("LiberationMono-Italic.ttf"), boldItalic: p("LiberationMono-BoldItalic.ttf")},
		}
	}
}

func joinCandidates(dirs, names []string) []string {
	out := make([]string, 0, len(dirs)*len(names))
	for _, dir := range dirs {
		for _, name := range names {
			out = append(out, filepath.Join(dir, name))
		}
	}
	return out
}
