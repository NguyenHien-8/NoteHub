package components

import (
	"image"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/NguyenHien-8/NoteHub/internal/domain"
)

const (
	galleryGap             float32 = 10
	galleryTileHeight      float32 = 300
	galleryMinWidth        float32 = 300
	galleryMaxContentWidth float32 = 960
	gallerySingleMaxWidth  float32 = 720
	galleryFooterHeight    float32 = 32
)

// ImageGallery renders every image attachment as a compact visual tile. When a
// reorder callback is supplied, a primary-button drag previews the drop target
// and commits the complete attachment ordering only when the mouse is released.
type ImageGallery struct {
	widget.BaseWidget
	attachments []domain.Attachment
	thumbnails  map[int64]image.Image
	tiles       []*imageTile
	onOpen      func(domain.Attachment)
	onReorder   func([]int64)
	onDelete    func(domain.Attachment)
	dragSource  *imageTile
	dropTarget  *imageTile
}

func NewImageGallery(attachments []domain.Attachment, thumbnails map[int64]image.Image, open func(domain.Attachment), reorder func([]int64)) *ImageGallery {
	g := &ImageGallery{onOpen: open, onReorder: reorder}
	g.ExtendBaseWidget(g)
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

// SetAttachments replaces the gallery snapshot without retaining aliases to
// caller slices. It is useful after an attachment is deleted in the editor.
func (g *ImageGallery) SetAttachments(attachments []domain.Attachment, thumbnails map[int64]image.Image) {
	g.attachments = append([]domain.Attachment(nil), attachments...)
	g.thumbnails = thumbnails
	g.tiles = nil
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
		g.tiles = append(g.tiles, tile)
	}
	g.dragSource, g.dropTarget = nil, nil
	g.Refresh()
}

func (g *ImageGallery) CreateRenderer() fyne.WidgetRenderer {
	return &imageGalleryRenderer{gallery: g, objects: g.canvasObjects()}
}

func (g *ImageGallery) MinSize() fyne.Size {
	if len(g.tiles) == 0 {
		return fyne.NewSize(0, 0)
	}
	columns := 1
	if len(g.tiles) > 1 {
		columns = 2
	}
	rows := (len(g.tiles) + columns - 1) / columns
	return fyne.NewSize(galleryMinWidth, float32(rows)*galleryTileHeight+float32(max(0, rows-1))*galleryGap)
}

func (g *ImageGallery) canvasObjects() []fyne.CanvasObject {
	objects := make([]fyne.CanvasObject, 0, len(g.tiles))
	for _, tile := range g.tiles {
		objects = append(objects, tile.object)
	}
	return objects
}

func (g *ImageGallery) targetAt(point fyne.Position) *imageTile {
	if point.X < 0 || point.Y < 0 || point.X >= g.Size().Width || point.Y >= g.Size().Height {
		return nil
	}
	for _, tile := range g.tiles {
		position, size := tile.Position(), tile.Size()
		if point.X >= position.X && point.X <= position.X+size.Width && point.Y >= position.Y && point.Y <= position.Y+size.Height {
			return tile
		}
	}
	return nil
}

func (g *ImageGallery) beginDrag(source *imageTile, point fyne.Position) {
	if g.onReorder == nil || !source.primaryDown {
		return
	}
	target := g.targetAt(point)
	if target == nil || target == source {
		g.clearDragFeedback()
		return
	}
	g.dragSource, g.dropTarget = source, target
	source.dragging = true
	target.dropTarget = true
	source.Refresh()
	target.Refresh()
}

func (g *ImageGallery) endDrag(source *imageTile) {
	if g.dragSource != source || g.dropTarget == nil || g.dropTarget == source {
		g.clearDragFeedback()
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
		g.clearDragFeedback()
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
	source.suppressTap = true
	callback := g.onReorder
	g.clearDragFeedback()
	if callback != nil {
		callback(ordered)
	}
}

func (g *ImageGallery) clearDragFeedback() {
	for _, tile := range g.tiles {
		changed := tile.dragging || tile.dropTarget
		tile.dragging, tile.dropTarget = false, false
		if changed {
			tile.Refresh()
		}
	}
	g.dragSource, g.dropTarget = nil, nil
}

type imageGalleryRenderer struct {
	gallery *ImageGallery
	objects []fyne.CanvasObject
}

func (r *imageGalleryRenderer) Layout(size fyne.Size) {
	count := len(r.gallery.tiles)
	if count == 0 {
		return
	}
	columns := 1
	contentWidth := min(size.Width, gallerySingleMaxWidth)
	if count > 1 {
		columns = 2
		contentWidth = min(size.Width, galleryMaxContentWidth)
	}
	width := contentWidth
	if columns == 2 {
		width = (contentWidth - galleryGap) / 2
	}
	xOffset := max(float32(0), (size.Width-contentWidth)/2)
	for index, tile := range r.gallery.tiles {
		column := index % columns
		row := index / columns
		x := xOffset + float32(column)*(width+galleryGap)
		y := float32(row) * (galleryTileHeight + galleryGap)
		tile.object.Move(fyne.NewPos(x, y))
		tile.object.Resize(fyne.NewSize(width, galleryTileHeight))
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
	t.ExtendBaseWidget(t)
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

func (t *imageTile) MinSize() fyne.Size { return fyne.NewSize(145, galleryTileHeight) }

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
	r.tile.preview.Refresh()
	r.tile.name.Refresh()
	r.Layout(r.tile.Size())
}
