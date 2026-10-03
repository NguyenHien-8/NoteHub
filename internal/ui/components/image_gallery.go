package components

import (
	"image"
	"image/color"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
)

const (
	galleryGap          float32 = 10
	galleryTileHeight   float32 = 300
	galleryMinWidth     float32 = 160
	galleryColumnWidth  float32 = 230
	galleryFooterHeight float32 = 32
)

// ImageGallery renders every image attachment as a compact visual tile. When a
// reorder callback is supplied, a primary-button drag previews the drop target
// and commits the complete attachment ordering only when the mouse is released.
type ImageGallery struct {
	widget.BaseWidget
	attachments      []domain.Attachment
	thumbnails       map[int64]image.Image
	tiles            []*imageTile
	onOpen           func(domain.Attachment)
	onReorder        func([]int64)
	onDelete         func(domain.Attachment)
	dragSource       *imageTile
	dropTarget       *imageTile
	ghost            *fyne.Container
	ghostImage       *canvas.Image
	ghostShade       *canvas.Rectangle
	ghostBorder      *canvas.Rectangle
	marker           *canvas.Rectangle
	frames           []galleryFrame
	frameWidth       float32
	frameHeight      float32
	reflowAnimations map[*imageTile]*fyne.Animation
}

func NewImageGallery(attachments []domain.Attachment, thumbnails map[int64]image.Image, open func(domain.Attachment), reorder func([]int64)) *ImageGallery {
	g := &ImageGallery{onOpen: open, onReorder: reorder, reflowAnimations: make(map[*imageTile]*fyne.Animation)}
	g.ExtendBaseWidget(g)
	g.ghostImage = canvas.NewImageFromImage(nil)
	g.ghostImage.FillMode = canvas.ImageFillContain
	g.ghostImage.Translucency = 0.08
	g.ghostShade = canvas.NewRectangle(color.NRGBA{A: 40})
	g.ghostShade.CornerRadius = 10
	g.ghostBorder = canvas.NewRectangle(color.NRGBA{R: 238, G: 246, B: 255, A: 245})
	g.ghostBorder.CornerRadius = 8
	g.ghostBorder.StrokeColor, g.ghostBorder.StrokeWidth = primaryBlue, 2
	g.ghost = container.NewWithoutLayout(g.ghostShade, g.ghostBorder, g.ghostImage)
	g.ghost.Hide()
	g.marker = canvas.NewRectangle(primaryBlue)
	g.marker.CornerRadius = 2
	g.marker.Hide()
	g.SetAttachments(attachments, thumbnails)
	return g
}

// SetDelete enables an inline delete action on image tiles. Deletion itself is
// owned by the caller so it can confirm and persist the operation safely.
func (g *ImageGallery) SetDelete(remove func(domain.Attachment)) {
	g.onDelete = remove
	for _, tile := range g.tiles {
		tile.Refresh()
	}
}

// SetReorderGuard lets dialogs temporarily block drag reordering while another
// save/delete operation owns the attachment snapshot. The guard is evaluated
// at drag start and again at drop, preventing optimistic UI changes that cannot
// be persisted.
func (g *ImageGallery) SetReorderGuard(guard func() bool) { g.canReorder = guard }

func (g *ImageGallery) reorderAllowed() bool {
	if g.onReorder == nil || g.reorderPending {
		return false
	}
	return g.canReorder == nil || g.canReorder()
}

// SetAttachments replaces the gallery snapshot without retaining aliases to
// caller slices. It is useful after an attachment is deleted in the editor.
func (g *ImageGallery) SetAttachments(attachments []domain.Attachment, thumbnails map[int64]image.Image) {
	g.stopReflowAnimations()
	g.attachments = append([]domain.Attachment(nil), attachments...)
	g.thumbnails = thumbnails
	g.tiles = nil
	g.frames = nil
	for _, attachment := range g.attachments {
		if !strings.HasPrefix(attachment.MIMEType, "image/") {
			continue
		}
		tile := newImageTile(g, attachment, thumbnails[attachment.ID])
		if g.onReorder == nil {
			tile.object = tile
		} else {
			tile.object = &draggableImageTile{imageTile: tile}
		}
		// Extend the actual widget in the render tree, not the embedded tile.
		// Two renderer identities sharing the same image cause stale geometry.
		tile.ExtendBaseWidget(tile.object.(fyne.Widget))
		g.tiles = append(g.tiles, tile)
	}
	g.dragSource, g.dropTarget = nil, nil
	g.reorderPending = false
	g.ghost.Hide()
	g.marker.Hide()
	g.Refresh()
}

func (g *ImageGallery) CreateRenderer() fyne.WidgetRenderer {
	return &imageGalleryRenderer{gallery: g, objects: g.canvasObjects()}
}

func (g *ImageGallery) MinSize() fyne.Size {
	if len(g.tiles) == 0 {
		return fyne.NewSize(0, 0)
	}
	_, height := g.geometry(g.layoutWidth())
	return fyne.NewSize(galleryMinWidth, height)
}

func (g *ImageGallery) layoutWidth() float32 {
	if g.Size().Width > 0 {
		return g.Size().Width
	}
	return galleryColumnWidth
}

// Height depends on width. Invalidating the canvas when the width changes lets
// enclosing cards and scroll areas remeasure rows after sidebar/window resizing.
func (g *ImageGallery) Resize(size fyne.Size) {
	previous := g.MinSize().Height
	g.BaseWidget.Resize(size)
	if previous != g.MinSize().Height {
		canvas.Refresh(g)
	}
}

type galleryFrame struct {
	position fyne.Position
	size     fyne.Size
}

// Fill every row, including the last, with widths weighted by image aspect.
// Height is bounded so a lone portrait never creates a screen-tall card.
// Contain inside each frame keeps all of the original image visible.
func (g *ImageGallery) geometry(width float32) ([]galleryFrame, float32) {
	count := len(g.tiles)
	if count == 0 {
		return nil, 0
	}
	width = max(1, width)
	if len(g.frames) == count && width == g.frameWidth {
		return g.frames, g.frameHeight
	}
	columns := min(count, max(1, int((width+galleryGap)/(galleryColumnWidth+galleryGap))))
	frames := make([]galleryFrame, count)
	y := float32(0)
	for first := 0; first < count; first += columns {
		end := min(count, first+columns)
		ratios := make([]float32, end-first)
		total := float32(0)
		for i := first; i < end; i++ {
			ratio := float32(4.0 / 3.0)
			if img := g.tiles[i].preview.Image; img != nil && img.Bounds().Dy() > 0 {
				ratio = float32(img.Bounds().Dx()) / float32(img.Bounds().Dy())
			}
			// Extreme panoramas/portraits still get usable frames, with contain
			// preserving their full content inside them.
			ratios[i-first] = min(float32(2.2), max(float32(0.65), ratio))
			total += ratios[i-first]
		}
		usable := max(1, width-float32(end-first-1)*galleryGap)
		previewHeight := min(galleryTileHeight, usable/total)
		height := previewHeight + galleryFooterHeight + 16
		x := float32(0)
		for i := first; i < end; i++ {
			w := usable * ratios[i-first] / total
			if i == end-1 {
				w = width - x // absorb rounding so the final frame meets the edge
			}
			frames[i] = galleryFrame{fyne.NewPos(x, y), fyne.NewSize(w, height)}
			x += w + galleryGap
		}
		y += height + galleryGap
	}
	g.frames, g.frameWidth, g.frameHeight = frames, width, y-galleryGap
	return frames, g.frameHeight
}

func (g *ImageGallery) canvasObjects() []fyne.CanvasObject {
	objects := make([]fyne.CanvasObject, 0, len(g.tiles)+2)
	for _, tile := range g.tiles {
		objects = append(objects, tile.object)
	}
	return append(objects, g.marker, g.ghost)
}

func (g *ImageGallery) targetAt(point fyne.Position) *imageTile {
	if point.X < 0 || point.Y < 0 || point.X >= g.Size().Width || point.Y >= g.Size().Height {
		return nil
	}
	// Hit-test stable layout slots rather than animated tile positions. This
	// prevents a tile that is gliding out of the way from repeatedly stealing
	// the pointer and making drag feedback jitter.
	frames, _ := g.geometry(g.Size().Width)
	for index, frame := range frames {
		position, size := frame.position, frame.size
		if point.X >= position.X && point.X <= position.X+size.Width && point.Y >= position.Y && point.Y <= position.Y+size.Height {
			return g.tiles[index]
		}
	}
	return nil
}

func (g *ImageGallery) beginDrag(source *imageTile, point fyne.Position) {
	if !g.reorderAllowed() || !source.primaryDown {
		return
	}
	if g.dragSource != source {
		g.clearDragFeedback(true)
		g.dragSource = source
		source.dragging, source.suppressTap = true, true
		source.Refresh()
		g.ghostImage.Image = source.preview.Image
		g.ghostImage.Refresh()
		g.ghost.Show()
	}
	target := g.targetAt(point)
	if target == source {
		target = nil
	}
	if g.dropTarget != target {
		if g.dropTarget != nil {
			g.dropTarget.dropTarget = false
			g.dropTarget.Refresh()
		}
		g.dropTarget = target
		if target != nil {
			target.dropTarget = true
			target.Refresh()
		}
		g.animateDropPreview(source, target)
	}
	g.moveGhost(source, point)
	if target == nil {
		g.marker.Hide()
	} else {
		x := target.Position().X
		if target.Position().Y > source.Position().Y || (target.Position().Y == source.Position().Y && x > source.Position().X) {
			x += target.Size().Width - 3
		}
		g.marker.Move(fyne.NewPos(x, target.Position().Y+4))
		g.marker.Resize(fyne.NewSize(3, max(0, target.Size().Height-8)))
		g.marker.Show()
	}
}

func (g *ImageGallery) moveGhost(source *imageTile, point fyne.Position) {
	size := source.Size()
	scale := min(float32(0.72), float32(220)/max(1, size.Width))
	w, h := max(float32(1), size.Width*scale), max(float32(1), (size.Height-galleryFooterHeight)*scale)
	x := max(float32(0), min(g.Size().Width-w-5, point.X-w/2))
	y := max(float32(0), min(g.Size().Height-h-5, point.Y-h/2))
	g.ghostShade.Move(fyne.NewPos(4, 5))
	g.ghostShade.Resize(fyne.NewSize(w, h))
	g.ghostBorder.Resize(fyne.NewSize(w, h))
	g.ghostImage.Move(fyne.NewPos(6, 6))
	g.ghostImage.Resize(fyne.NewSize(max(0, w-12), max(0, h-12)))
	g.ghost.Resize(fyne.NewSize(w+5, h+5))
	g.ghost.Move(fyne.NewPos(x, y))
}

func (g *ImageGallery) endDrag(source *imageTile) {
	if !g.reorderAllowed() {
		g.clearDragFeedback(true)
		return
	}
	if g.dragSource != source || g.dropTarget == nil || g.dropTarget == source {
		g.clearDragFeedback(true)
		return
	}
	sourceIndex, targetIndex := -1, -1
	images := make([]int64, 0, len(g.tiles))
	for index, tile := range g.tiles {
		images = append(images, tile.attachment.ID)
		if tile == source {
			sourceIndex = index
		}
		if tile == g.dropTarget {
			targetIndex = index
		}
	}
	if sourceIndex < 0 || targetIndex < 0 || sourceIndex == targetIndex {
		g.clearDragFeedback(true)
		return
	}
	moved := images[sourceIndex]
	images = append(images[:sourceIndex], images[sourceIndex+1:]...)
	if targetIndex > len(images) {
		targetIndex = len(images)
	}
	images = append(images, 0)
	copy(images[targetIndex+1:], images[targetIndex:])
	images[targetIndex] = moved

	ordered := make([]int64, 0, len(g.attachments))
	imageIndex := 0
	for _, attachment := range g.attachments {
		if strings.HasPrefix(attachment.MIMEType, "image/") {
			ordered = append(ordered, images[imageIndex])
			imageIndex++
		} else {
			ordered = append(ordered, attachment.ID)
		}
	}

	// Keep the already animated preview as an optimistic local ordering. The
	// persistence callback refreshes the timeline on success; callers also
	// refresh/revert on failure. This avoids the old drop -> snap back -> reload
	// sequence while retaining database authority over the final order.
	source.suppressTap = true
	g.reorderPending = true
	g.applyLocalOrder(ordered)
	callback := g.onReorder
	g.clearDragFeedback(false)
	g.animateCanonicalPositions(110 * time.Millisecond)
	if callback != nil {
		callback(ordered)
	}
}

func (g *ImageGallery) animateDropPreview(source, target *imageTile) {
	frames, _ := g.geometry(g.Size().Width)
	if len(frames) != len(g.tiles) {
		return
	}
	order := append([]*imageTile(nil), g.tiles...)
	if target != nil && target != source {
		sourceIndex, targetIndex := -1, -1
		for index, tile := range order {
			if tile == source {
				sourceIndex = index
			}
			if tile == target {
				targetIndex = index
			}
		}
		if sourceIndex >= 0 && targetIndex >= 0 {
			order = append(order[:sourceIndex], order[sourceIndex+1:]...)
			if targetIndex > len(order) {
				targetIndex = len(order)
			}
			order = append(order, nil)
			copy(order[targetIndex+1:], order[targetIndex:])
			order[targetIndex] = source
		}
	}

	destinations := make(map[*imageTile]fyne.Position, len(order))
	for index, tile := range order {
		if tile != source {
			destinations[tile] = frames[index].position
		}
	}
	g.animateTiles(destinations, 135*time.Millisecond)
}

func (g *ImageGallery) animateCanonicalPositions(duration time.Duration) {
	g.frames = nil
	frames, _ := g.geometry(g.Size().Width)
	destinations := make(map[*imageTile]fyne.Position, len(g.tiles))
	for index, tile := range g.tiles {
		destinations[tile] = frames[index].position
	}
	g.animateTiles(destinations, duration)
}

func (g *ImageGallery) animateTiles(destinations map[*imageTile]fyne.Position, duration time.Duration) {
	g.stopReflowAnimations()
	app := fyne.CurrentApp()
	for tile, destination := range destinations {
		start := tile.object.Position()
		if start == destination {
			continue
		}
		if app == nil || app.Driver() == nil || duration <= 0 {
			tile.object.Move(destination)
			continue
		}
		tile := tile
		start := start
		destination := destination
		var animation *fyne.Animation
		animation = fyne.NewAnimation(duration, func(progress float32) {
			x := start.X + (destination.X-start.X)*progress
			y := start.Y + (destination.Y-start.Y)*progress
			tile.object.Move(fyne.NewPos(x, y))
			if progress >= 1 && g.reflowAnimations[tile] == animation {
				delete(g.reflowAnimations, tile)
			}
		})
		animation.Curve = fyne.AnimationEaseOut
		g.reflowAnimations[tile] = animation
		animation.Start()
	}
}

func (g *ImageGallery) stopReflowAnimations() {
	for tile, animation := range g.reflowAnimations {
		animation.Stop()
		delete(g.reflowAnimations, tile)
	}
}

func (g *ImageGallery) applyLocalOrder(ids []int64) {
	byTile := make(map[int64]*imageTile, len(g.tiles))
	for _, tile := range g.tiles {
		byTile[tile.attachment.ID] = tile
	}
	byAttachment := make(map[int64]domain.Attachment, len(g.attachments))
	for _, attachment := range g.attachments {
		byAttachment[attachment.ID] = attachment
	}
	images := make([]*imageTile, 0, len(g.tiles))
	attachments := make([]domain.Attachment, 0, len(ids))
	for position, id := range ids {
		attachment, ok := byAttachment[id]
		if !ok {
			return
		}
		attachment.Position = position
		attachments = append(attachments, attachment)
		if tile := byTile[id]; tile != nil {
			images = append(images, tile)
		}
	}
	if len(images) != len(g.tiles) || len(attachments) != len(g.attachments) {
		return
	}
	g.tiles = images
	g.attachments = attachments
	g.frames = nil
}

func (g *ImageGallery) clearDragFeedback(resetLayout bool) {
	for _, tile := range g.tiles {
		changed := tile.dragging || tile.dropTarget
		tile.dragging, tile.dropTarget = false, false
		if changed {
			tile.Refresh()
		}
	}
	g.dragSource, g.dropTarget = nil, nil
	g.ghost.Hide()
	g.marker.Hide()
	if resetLayout {
		g.animateCanonicalPositions(100 * time.Millisecond)
	}
}

type imageGalleryRenderer struct {
	gallery *ImageGallery
	objects []fyne.CanvasObject
}

func (r *imageGalleryRenderer) Layout(size fyne.Size) {
	frames, _ := r.gallery.geometry(size.Width)
	for index, tile := range r.gallery.tiles {
		tile.object.Move(frames[index].position)
		tile.object.Resize(frames[index].size)
	}
}

func (r *imageGalleryRenderer) MinSize() fyne.Size { return r.gallery.MinSize() }
func (r *imageGalleryRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}
func (r *imageGalleryRenderer) Destroy() {}
func (r *imageGalleryRenderer) Refresh() {
	r.objects = r.gallery.canvasObjects()
	r.Layout(r.gallery.Size())
	for _, object := range r.objects {
		canvas.Refresh(object)
	}
}

type imageTile struct {
	widget.BaseWidget
	gallery     *ImageGallery
	attachment  domain.Attachment
	preview     *canvas.Image
	name        *widget.Label
	object      fyne.CanvasObject
	primaryDown bool
	dragging    bool
	dropTarget  bool
	suppressTap bool
}

func newImageTile(gallery *ImageGallery, attachment domain.Attachment, thumbnail image.Image) *imageTile {
	t := &imageTile{gallery: gallery, attachment: attachment}
	t.preview = canvas.NewImageFromImage(thumbnail)
	t.preview.FillMode = canvas.ImageFillContain
	t.preview.Translucency = 0
	t.name = widget.NewLabel(boundedText(attachment.Filename, 28))
	t.name.Truncation = fyne.TextTruncateEllipsis
	t.name.Importance = widget.LowImportance
	return t
}

func (t *imageTile) CreateRenderer() fyne.WidgetRenderer {
	background := canvas.NewRectangle(color.NRGBA{R: 241, G: 245, B: 249, A: 255})
	background.CornerRadius = 8
	background.StrokeWidth = 1
	background.StrokeColor = color.NRGBA{R: 226, G: 232, B: 240, A: 255}
	placeholder := widget.NewLabelWithStyle("Image preview", fyne.TextAlignCenter, fyne.TextStyle{})
	placeholder.Importance = widget.LowImportance
	remove := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		if t.gallery.onDelete != nil {
			t.gallery.onDelete(t.attachment)
		}
	})
	remove.Importance = widget.LowImportance
	if t.gallery.onDelete == nil {
		remove.Hide()
	}
	r := &imageTileRenderer{tile: t, background: background, placeholder: placeholder, remove: remove, objects: []fyne.CanvasObject{background, placeholder, t.preview, t.name, remove}}
	r.Refresh()
	return r
}

func (t *imageTile) MinSize() fyne.Size { return fyne.NewSize(1, galleryFooterHeight+16) }

func (t *imageTile) Tapped(*fyne.PointEvent) {
	if t.suppressTap {
		t.suppressTap = false
		return
	}
	if t.gallery.onOpen != nil {
		t.gallery.onOpen(t.attachment)
	}
}

func (t *imageTile) MouseDown(event *desktop.MouseEvent) {
	t.primaryDown = event != nil && event.Button == desktop.MouseButtonPrimary
	if t.primaryDown {
		t.suppressTap = false
	}
}
func (t *imageTile) MouseUp(*desktop.MouseEvent) { t.primaryDown = false }

type draggableImageTile struct{ *imageTile }

func (d *draggableImageTile) Cursor() desktop.Cursor { return desktop.PointerCursor }

func (d *draggableImageTile) Dragged(event *fyne.DragEvent) {
	if event == nil || !d.primaryDown {
		return
	}
	position := d.Position()
	point := fyne.NewPos(position.X+event.Position.X, position.Y+event.Position.Y)
	d.gallery.beginDrag(d.imageTile, point)
}
func (d *draggableImageTile) DragEnd() {
	d.gallery.endDrag(d.imageTile)
	d.primaryDown = false
}

type imageTileRenderer struct {
	tile        *imageTile
	background  *canvas.Rectangle
	placeholder *widget.Label
	remove      *widget.Button
	objects     []fyne.CanvasObject
}

func (r *imageTileRenderer) Layout(size fyne.Size) {
	r.background.Resize(size)
	footerHeight := galleryFooterHeight
	previewHeight := max(0, size.Height-footerHeight-4)
	r.previewLayout(size.Width, previewHeight)
	r.tile.name.Move(fyne.NewPos(8, previewHeight+4))
	nameWidth := max(0, size.Width-16)
	if r.tile.gallery.onDelete != nil {
		nameWidth = max(0, nameWidth-30)
		r.remove.Move(fyne.NewPos(size.Width-30, previewHeight+2))
		r.remove.Resize(fyne.NewSize(28, 28))
	}
	r.tile.name.Resize(fyne.NewSize(nameWidth, footerHeight))
}

func (r *imageTileRenderer) previewLayout(width, height float32) {
	if r.tile.preview.Image == nil {
		r.tile.preview.Hide()
		r.placeholder.Show()
		r.placeholder.Move(fyne.NewPos(8, 8))
		r.placeholder.Resize(fyne.NewSize(max(0, width-16), max(0, height-16)))
	} else {
		r.placeholder.Hide()
		r.tile.preview.Show()
		r.tile.preview.Move(fyne.NewPos(6, 6))
		r.tile.preview.Resize(fyne.NewSize(max(0, width-12), max(0, height-12)))
	}
}

func (r *imageTileRenderer) MinSize() fyne.Size           { return r.tile.MinSize() }
func (r *imageTileRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *imageTileRenderer) Destroy()                     {}
func (r *imageTileRenderer) Refresh() {
	variant := fyne.CurrentApp().Settings().ThemeVariant()
	r.background.FillColor = r.tile.Theme().Color(theme.ColorNameInputBackground, variant)
	if r.tile.dragging || r.tile.dropTarget {
		r.background.StrokeWidth = 2
		r.background.StrokeColor = primaryBlue
	} else {
		r.background.StrokeWidth = 1
		r.background.StrokeColor = r.tile.Theme().Color(theme.ColorNameInputBorder, variant)
	}
	if r.tile.gallery.onDelete == nil {
		r.remove.Hide()
	} else {
		r.remove.Show()
	}
	r.background.Refresh()
	if r.tile.dragging {
		r.tile.preview.Translucency = 0.65
	} else {
		r.tile.preview.Translucency = 0
	}
	r.tile.preview.Refresh()
	r.tile.name.Refresh()
	r.Layout(r.tile.Size())
}
