package ui

import (
	"context"
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/NguyenHien-8/NoteHub/assets"
	"github.com/NguyenHien-8/NoteHub/internal/app"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
	"github.com/NguyenHien-8/NoteHub/internal/platform"
	"github.com/NguyenHien-8/NoteHub/internal/repository"
	"github.com/NguyenHien-8/NoteHub/internal/ui/components"
	"github.com/NguyenHien-8/NoteHub/internal/ui/dialogs"
	"github.com/NguyenHien-8/NoteHub/internal/ui/screens"
	"github.com/NguyenHien-8/NoteHub/internal/ui/work"
)

type Desktop struct {
	fyne.Window
	application            fyne.App
	Backend                *app.Backend
	Jobs                   *work.Runner
	Home                   *screens.Timeline
	Search                 *screens.Search
	Calendar               *screens.CalendarScreen
	Attachments            *screens.Attachments
	GlobalSearch           *widget.Entry
	manager                *dialogs.Manager
	sidebar                *components.Sidebar
	rails                  *Rails
	mini                   *components.Calendar
	tags                   *screens.Tags
	settings               fyne.CanvasObject
	center                 *fyne.Container
	current                string
	month                  time.Time
	selectedDate           string
	all, favorites, shared *widget.Button
	metadataGeneration     uint64
	miniGeneration         uint64
	expiryTicker           *time.Ticker
	stopTick               chan struct{}
	shutdownOnce           sync.Once
	startupOnce            sync.Once
}

func NewWindow(application fyne.App, b *app.Backend, version string) *Desktop {
	return newWindow(application, b, version, work.New())
}

func newWindow(application fyne.App, b *app.Backend, version string, jobs *work.Runner) *Desktop {
	prefs := application.Preferences()
	application.Settings().SetTheme(newConfiguredTheme(
		prefs.StringWithFallback("appearance", "Light"),
		prefs.StringWithFallback("font-family", "System"),
		float32(prefs.FloatWithFallback("font-size", float64(defaultTextSize))),
	))
	w := application.NewWindow("NoteHub")
	w.SetIcon(assets.Logo)
	w.SetPadded(false)
	now := time.Now()
	d := &Desktop{Window: w, application: application, Backend: b, Jobs: jobs, month: time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local), center: container.NewStack(), stopTick: make(chan struct{})}
	previews := screens.NewPreviews(b.Attachments)
	d.manager = dialogs.New(b, w, jobs, d.Refresh)
	d.manager.SetPreviewLoader(previews.Image)
	env := &screens.Environment{Backend: b, Window: w, Jobs: jobs, Previews: previews, Changed: d.Refresh, Tag: d.showTag, Memo: d.manager.ShowMemo, PickFiles: d.manager.PickFiles, PickTag: d.manager.PickTag}
	env.Actions = components.MemoActions{
		Open: func(m domain.Memo) { d.manager.ShowMemo(m.ID) }, Edit: d.manager.EditMemo, Favorite: d.manager.Favorite, Share: d.manager.ShareMemo, Delete: d.manager.DeleteMemo, Tag: d.showTag, Attachment: d.manager.OpenAttachment, Reorder: d.manager.ReorderAttachments,
	}
	d.Home = screens.NewTimeline(env)
	d.Search = screens.NewSearch(env)
	d.Calendar = screens.NewCalendar(env)
	d.Attachments = screens.NewAttachments(env)
	d.tags = screens.NewTags(d.showTag)
	d.settings = screens.NewSettings(env, application, d.manager, version, d.SetAppearance, d.SetTypography)
	d.sidebar = components.NewSidebar(d.Navigate, d.showTag, func() { d.Navigate("home"); d.manager.PickTag(d.Home.Composer.InsertTag) })
	d.mini = components.NewCalendar(d.showDate, func(delta int) { d.month = d.month.AddDate(0, delta, 0); d.refreshMini() })
	d.mini.SetMonth(d.month.Year(), d.month.Month(), nil, "")
	d.all = widget.NewButtonWithIcon("All Notes", theme.DocumentIcon(), func() { d.showFilter(repository.TimelineQuery{}) })
	d.favorites = widget.NewButtonWithIcon("Favorites", favoriteIcon, func() { d.showFilter(repository.TimelineQuery{FavoriteOnly: true}) })
	d.shared = widget.NewButtonWithIcon("Shared", theme.AccountIcon(), func() { d.showFilter(repository.TimelineQuery{SharedOnly: true}) })
	for _, button := range []*widget.Button{d.all, d.favorites, d.shared} {
		button.Alignment = widget.ButtonAlignLeading
		button.Importance = widget.LowImportance
	}
	filters := components.Surface(container.NewVBox(widget.NewLabelWithStyle("Quick Filters", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), d.all, d.favorites, d.shared))
	right := components.SidePanel(container.NewVScroll(container.New(layout.NewCustomPaddedVBoxLayout(14), d.mini.Object, filters)))
	left := components.SidePanel(d.sidebar.Object)
	main := components.WorkspacePanel(d.center)
	d.rails = NewRails(left, main, right, application.Preferences())
	columns := fyne.CanvasObject(container.New(insetLayout{10}, d.rails))
	d.GlobalSearch = components.NewSearchBar(func(text string) {
		// Update the query before navigation. Search.Refresh() cancels the
		// debounce timer, avoiding two back-to-back list rebuilds and flicker.
		d.Search.Query.SetText(text)
		if d.current != "search" {
			d.Navigate("search")
		}
	})
	logo := canvas.NewImageFromResource(assets.Logo)
	logo.FillMode = canvas.ImageFillContain
	logo.SetMinSize(fyne.NewSize(44, 44))
	brand := widget.NewLabelWithStyle("NoteHub", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	brand.SizeName = brandTextSizeName
	search := container.NewGridWrap(fyne.NewSize(380, 40), d.GlobalSearch)
	leftToggle := widget.NewButtonWithIcon("", theme.MenuIcon(), d.rails.ToggleLeft)
	rightToggle := widget.NewButtonWithIcon("", theme.ListIcon(), d.rails.ToggleRight)
	leftToggle.Importance, rightToggle.Importance = widget.LowImportance, widget.LowImportance
	header := container.New(insetLayout{10}, container.NewBorder(nil, nil,
		container.NewHBox(leftToggle, logo, brand),
		container.NewHBox(search, rightToggle),
		layout.NewSpacer()))
	w.SetContent(container.NewBorder(header, nil, nil, nil, columns))
	w.Resize(platform.InitialWindowSize(w.Canvas().Scale()))
	w.CenterOnScreen()
	focus := func(fyne.Shortcut) { w.Canvas().Focus(d.GlobalSearch) }
	w.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyK, Modifier: fyne.KeyModifierControl}, focus)
	w.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyK, Modifier: fyne.KeyModifierSuper}, focus)
	w.SetCloseIntercept(d.requestClose)
	w.SetOnClosed(d.stop)
	d.expiryTicker = time.NewTicker(time.Minute)
	go func() {
		for {
			select {
			case <-d.expiryTicker.C:
				jobs.Post(func() {
					d.refreshMetadata()
					if d.current == "home" {
						d.Home.RefreshActiveShares()
					}
				})
			case <-d.stopTick:
				return
			}
		}
	}()
	d.Navigate("home")
	d.refreshMetadata()
	return d
}

// Show maximizes only the first launch, after Fyne creates the native handle.
// Reopening a hidden window preserves the user's later restored/maximized state.
func (d *Desktop) Show() {
	d.Window.Show()
	d.startupOnce.Do(func() { platform.MaximizeWindow(d.Window) })
}

func (d *Desktop) ShowAndRun() {
	d.Show()
	d.application.Run()
}

// Wait finishes outstanding storage work before the caller closes the backend.
func (d *Desktop) Wait() error {
	d.stop()
	d.Jobs.Wait()
	return d.manager.Close()
}

// Driver quit (including an OS signal) need not call the window's OnClosed.
func (d *Desktop) stop() {
	d.shutdownOnce.Do(func() {
		d.Search.Close()
		d.Jobs.Stop()
		d.expiryTicker.Stop()
		close(d.stopTick)
	})
}

type metadata struct {
	tags   []domain.TagCount
	counts domain.MemoCounts
}

func (d *Desktop) refreshMetadata() {
	d.metadataGeneration++
	generation := d.metadataGeneration
	work.Run(d.Jobs, func(ctx context.Context) (metadata, error) {
		tags, err := d.Backend.Tags.List(ctx)
		if err != nil {
			return metadata{}, err
		}
		counts, err := d.Backend.Timeline.Counts(ctx)
		return metadata{tags: tags, counts: counts}, err
	}, func(result metadata, err error) {
		if generation != d.metadataGeneration {
			return
		}
		if err != nil {
			d.manager.Error(err)
			return
		}
		d.sidebar.SetTags(result.tags)
		d.tags.Set(result.tags)
		setButtonText := func(button *widget.Button, text string) {
			if button.Text != text {
				button.SetText(text)
			}
		}
		setButtonText(d.all, fmt.Sprintf("All Notes        %d", result.counts.All))
		setButtonText(d.favorites, fmt.Sprintf("Favorites        %d", result.counts.Favorites))
		setButtonText(d.shared, fmt.Sprintf("Shared           %d", result.counts.Shared))
	})
	d.refreshMini()
}

func (d *Desktop) refreshMini() {
	d.miniGeneration++
	generation, month := d.miniGeneration, d.month
	work.Run(d.Jobs, func(ctx context.Context) ([]domain.CalendarDay, error) {
		return d.Backend.Calendar.Month(ctx, month.Year(), month.Month(), time.Local)
	}, func(days []domain.CalendarDay, err error) {
		if generation != d.miniGeneration {
			return
		}
		if err != nil {
			d.manager.Error(err)
			return
		}
		d.mini.SetMonth(month.Year(), month.Month(), days, d.selectedDate)
	})
}
