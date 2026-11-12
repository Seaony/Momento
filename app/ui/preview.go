package ui

import (
	"image"
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

type previewState struct {
	ID                              string
	Open, Held                      bool
	Bitmap                          *ui.Bitmap
	Loading                         bool
	Generation                      uint64
	Zoom                            float32
	Pan                             ui.Point
	targetRoot, targetID, errorText string
	pixelBitmap                     *ui.Bitmap
	viewport                        ui.Rect
	fitZoom                         float32
	initialized, fitMode            bool
	animation                       *previewAnimation
	animationLoading                bool
	nextFrameAt, wheelAt            time.Time
}

type previewAnimation struct {
	decoder ui.VideoDecoder
	delay   time.Duration
}
type previewPicture struct {
	bitmap, pixelBitmap *ui.Bitmap
	animation           *previewAnimation
}

func (p *previewState) target(root, id string) {
	if p.targetRoot == root && p.targetID == id {
		return
	}
	p.Generation++
	p.targetRoot, p.targetID = root, id
	p.Bitmap, p.pixelBitmap, p.animation, p.errorText = nil, nil, nil, ""
	p.Loading, p.animationLoading, p.initialized = false, false, false
	p.Zoom, p.Pan, p.fitZoom = 1, ui.Point{}, 1
	p.nextFrameAt = time.Time{}
	if root == "" {
		p.wheelAt = time.Time{}
	}
}
func (p *previewState) finish(generation uint64, bitmap *ui.Bitmap, err error) {
	if generation != p.Generation || !p.Open {
		return
	}
	p.Loading = false
	if err != nil {
		p.errorText = err.Error()
		return
	}
	p.Bitmap = bitmap
}
func (p *previewState) finishPicture(generation uint64, picture previewPicture, err error) {
	p.finish(generation, picture.bitmap, err)
	if generation == p.Generation && p.Open && err == nil {
		p.pixelBitmap, p.animation = picture.pixelBitmap, picture.animation
	}
}
func previewBitmaps(img image.Image) (*ui.Bitmap, *ui.Bitmap) {
	return ui.NewBitmap(img), ui.NewBitmap(img).Pixelated()
}

// GIF 只保留当前帧的两种采样位图，切到别的素材后不再解码。
func readPreviewPicture(path string) (previewPicture, error) {
	if strings.EqualFold(filepath.Ext(path), ".gif") {
		data, err := os.ReadFile(path)
		if err != nil {
			return previewPicture{}, err
		}
		decoder, err := ui.NewGIFDecoder(data)
		if err != nil {
			return previewPicture{}, err
		}
		frame, err := decoder.Next()
		if err != nil {
			return previewPicture{}, err
		}
		smooth, pixelated := previewBitmaps(frame.Image)
		return previewPicture{smooth, pixelated, &previewAnimation{decoder, frame.Duration}}, nil
	}
	img, _, _, _, err := assets.PrepareImage(path)
	if err != nil {
		return previewPicture{}, err
	}
	smooth, pixelated := previewBitmaps(img)
	return previewPicture{bitmap: smooth, pixelBitmap: pixelated}, nil
}
func (animation *previewAnimation) next() (previewPicture, error) {
	frame, err := animation.decoder.Next()
	if err == io.EOF {
		if err = animation.decoder.Rewind(); err == nil {
			frame, err = animation.decoder.Next()
		}
	}
	if err != nil {
		return previewPicture{}, err
	}
	smooth, pixelated := previewBitmaps(frame.Image)
	return previewPicture{smooth, pixelated, &previewAnimation{animation.decoder, frame.Duration}}, nil
}

func (a *App) preview(c *ui.Context) {
	p := &a.Preview
	if !p.Open {
		p.target("", "")
		return
	}
	if a.Library == nil || p.Held && !c.KeyDown(ui.KeySpace) {
		a.ClosePreview()
		p.target("", "")
		return
	}
	index := previewIndex(a.Visible, p.ID)
	if index < 0 {
		a.ClosePreview()
		p.target("", "")
		return
	}
	asset := a.Visible[index]
	p.target(a.Library.Root, asset.ID)
	if p.Bitmap == nil && p.errorText == "" && !p.Loading {
		a.loadPreview(asset, a.Library.Root)
	}
	a.animatePreview(c)
	width, height := c.Size()
	ui.DialogBase(c.Key("original-preview"), &p.Open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, .78))
		panel.Column().Size(width*.86, height*.86).Padding(12).Gap(12).Background(c.Theme().Background).Radius(12).
			Label(a.T("preview_original")).TabStop(false).AutoFocus()
		ui.Row(c).FillWidth().Gap(12).Children(func() {
			ui.Text(c, asset.Name).Bold().Grow(1).SingleLine()
			ui.Textf(c, "%.0f%%", p.Zoom*100).FontSize(12)
			ui.Text(c, a.T("preview_position", index+1, len(a.Visible))).FontSize(12).TextColor(c.Theme().TextMuted)
			if ui.Button(c.Key("close-preview"), "×").Label(a.T("close_preview")).Size(40, 40).Clicked() {
				a.ClosePreview()
			}
		})
		ui.Row(c).FillWidth().Grow(1).MinHeight(0).Gap(8).Children(func() {
			if ui.Button(c.Key("previous-preview"), "‹").Label(a.T("previous_asset")).Width(40).Height(48).Disabled(index == 0).Clicked() {
				a.movePreview(-1)
			}
			a.previewViewport(c, asset)
			if ui.Button(c.Key("next-preview"), "›").Label(a.T("next_asset")).Width(40).Height(48).Disabled(index == len(a.Visible)-1).Clicked() {
				a.movePreview(1)
			}
		})
		if a.previewShortcut(panel, "preview_previous") {
			a.movePreview(-1)
		}
		if a.previewShortcut(panel, "preview_next") {
			a.movePreview(1)
		}
		if a.previewShortcut(panel, "preview_zoom_in") {
			a.PreviewZoomIn()
		}
		if a.previewShortcut(panel, "preview_zoom_out") {
			a.PreviewZoomOut()
		}
		if a.previewShortcut(panel, "preview_fit") {
			a.PreviewFitToWindow()
		}
		if a.previewShortcut(panel, "preview_actual") {
			a.PreviewActualSize()
		}
	})
	if !p.Open {
		p.target("", "")
	}
}

func (a *App) previewViewport(c *ui.Context, asset model.Asset) {
	p := &a.Preview
	viewport := ui.Box(c.Key("preview-image")).Grow(1).FillHeight().MinWidth(0).MinHeight(0).Clip().Label(asset.Name).
		Background(c.Theme().Surface).HandleInput(a.previewInput)
	bounds := viewport.LayoutBounds()
	if bounds.W > 0 && bounds.H > 0 {
		changed := p.viewport.W != bounds.W || p.viewport.H != bounds.H
		p.viewport = bounds
		if p.Bitmap != nil {
			bw, bh := p.Bitmap.Size()
			p.fitZoom = previewFitZoom(float32(bw), float32(bh), bounds.W, bounds.H)
			if !p.initialized {
				p.fitMode = a.Preferences.PreviewFit == "fit"
				p.Zoom = p.fitZoom
				if !p.fitMode {
					p.Zoom = min(1, p.Zoom)
				}
				p.initialized = true
				c.After(0)
			} else if changed && p.fitMode {
				p.Zoom = p.fitZoom
				p.Pan = ui.Point{}
				c.After(0)
			}
			p.clampPan()
		}
	} else {
		c.After(0)
	}
	switch {
	case p.Bitmap != nil:
		viewport.Draw(func(painter *ui.Painter, r ui.Rect) {
			painter.Clip(r, 0, func() {
				imageRect := p.imageRect(r)
				if a.Preferences.PreviewChecker {
					drawPreviewChecker(painter, r, imageRect, c.Theme())
				}
				bitmap := p.Bitmap
				if a.Preferences.PreviewPixelated && p.pixelBitmap != nil {
					bitmap = p.pixelBitmap
				}
				painter.Image(bitmap, imageRect, ui.Contain)
			})
		})
		if p.canPan() {
			viewport.Cursor(ui.CursorGrab)
			if dx, dy, held := viewport.Dragged(); held {
				viewport.Cursor(ui.CursorGrabbing)
				p.Pan.X += dx
				p.Pan.Y += dy
				p.clampPan()
			}
		}
		if viewport.DoubleClicked() {
			if math.Abs(float64(p.Zoom-p.fitZoom)) < .001 {
				a.PreviewActualSize()
			} else {
				a.PreviewFitToWindow()
			}
		}
	case p.errorText != "":
		viewport.Column().Center().Gap(8).Children(func() {
			ui.Text(c, a.T("preview_failed")).Bold()
			ui.Text(c, p.errorText).TextColor(c.Theme().TextMuted).MaxWidth(560)
		})
	default:
		viewport.Center().Children(func() { ui.Text(c, a.T("preview_loading")).TextColor(c.Theme().TextMuted) })
	}
}
func previewFitZoom(imageWidth, imageHeight, width, height float32) float32 {
	if imageWidth <= 0 || imageHeight <= 0 || width <= 0 || height <= 0 {
		return 1
	}
	return min(width/imageWidth, height/imageHeight)
}
func (p *previewState) imageRect(viewport ui.Rect) ui.Rect {
	if p.Bitmap == nil {
		return ui.Rect{}
	}
	width, height := p.Bitmap.Size()
	w, h := float32(width)*p.Zoom, float32(height)*p.Zoom
	return ui.Rect{X: viewport.X + (viewport.W-w)/2 + p.Pan.X, Y: viewport.Y + (viewport.H-h)/2 + p.Pan.Y, W: w, H: h}
}
func (p *previewState) canPan() bool {
	if p.Bitmap == nil {
		return false
	}
	w, h := p.Bitmap.Size()
	return float32(w)*p.Zoom > p.viewport.W || float32(h)*p.Zoom > p.viewport.H
}
func (p *previewState) clampPan() {
	if p.Bitmap == nil {
		return
	}
	w, h := p.Bitmap.Size()
	x, y := max(0, (float32(w)*p.Zoom-p.viewport.W)/2), max(0, (float32(h)*p.Zoom-p.viewport.H)/2)
	p.Pan.X = max(-x, min(x, p.Pan.X))
	p.Pan.Y = max(-y, min(y, p.Pan.Y))
}
func (p *previewState) zoomAt(scale, x, y float32) {
	if p.Bitmap == nil || !p.initialized || scale <= 0 || math.IsNaN(float64(scale)) || math.IsInf(float64(scale), 0) {
		return
	}
	old := p.Zoom
	p.Zoom = max(.01, min(32, p.Zoom*scale))
	ratio := p.Zoom / old
	p.Pan.X = (p.Pan.X-(x-p.viewport.W/2))*ratio + x - p.viewport.W/2
	p.Pan.Y = (p.Pan.Y-(y-p.viewport.H/2))*ratio + y - p.viewport.H/2
	p.fitMode = false
	p.clampPan()
}
func (a *App) previewInput(ev ui.InputEvent) bool {
	p := &a.Preview
	switch ev.Kind {
	case ui.InputZoom:
		p.zoomAt(ev.Scale, ev.X, ev.Y)
		return true
	case ui.InputScroll:
		if a.Preferences.PreviewWheel == "zoom" {
			if ev.DY != 0 {
				p.zoomAt(float32(math.Exp(float64(-ev.DY)*.005)), ev.X, ev.Y)
			}
		} else if ev.DY != 0 && !ev.Momentum && (p.wheelAt.IsZero() || ev.Time.Sub(p.wheelAt) >= 180*time.Millisecond) {
			delta := 1
			if ev.DY < 0 {
				delta = -1
			}
			a.movePreview(delta)
			p.wheelAt = ev.Time
		}
		return true
	}
	return false
}
func (a *App) PreviewZoomIn()  { p := &a.Preview; p.zoomAt(1.25, p.viewport.W/2, p.viewport.H/2) }
func (a *App) PreviewZoomOut() { p := &a.Preview; p.zoomAt(.8, p.viewport.W/2, p.viewport.H/2) }
func (a *App) PreviewFitToWindow() {
	p := &a.Preview
	if p.Bitmap != nil {
		p.Zoom = p.fitZoom
		p.Pan = ui.Point{}
		p.fitMode = true
	}
}
func (a *App) PreviewActualSize() {
	p := &a.Preview
	if p.Bitmap != nil {
		p.Zoom = 1
		p.Pan = ui.Point{}
		p.fitMode = false
	}
}
func drawPreviewChecker(p *ui.Painter, viewport, image ui.Rect, theme *ui.Theme) {
	left, top := max(viewport.X, image.X), max(viewport.Y, image.Y)
	right, bottom := min(viewport.X+viewport.W, image.X+image.W), min(viewport.Y+viewport.H, image.Y+image.H)
	if right <= left || bottom <= top {
		return
	}
	const size = float32(12)
	for y := top; y < bottom; y += size {
		for x := left; x < right; x += size {
			color := theme.Surface
			if (int((x-left)/size)+int((y-top)/size))%2 == 1 {
				color = theme.SurfacePressed
			}
			p.Fill(ui.Rect{X: x, Y: y, W: min(size, right-x), H: min(size, bottom-y)}, color, 0)
		}
	}
}
func previewIndex(visible []model.Asset, id string) int {
	for i := range visible {
		if visible[i].ID == id {
			return i
		}
	}
	return -1
}
func (a *App) movePreview(delta int) {
	index := previewIndex(a.Visible, a.Preview.ID)
	if index < 0 || index+delta < 0 || index+delta >= len(a.Visible) {
		return
	}
	a.Preview.ID = a.Visible[index+delta].ID
}
func (a *App) loadPreview(asset model.Asset, root string) {
	p := &a.Preview
	p.Loading = true
	generation := p.Generation
	win := a.Window
	path := filepath.Join(root, filepath.FromSlash(asset.RelativePath))
	go func() {
		picture, err := readPreviewPicture(path)
		if err != nil {
			slog.Error("原图预览加载失败", "asset", asset.ID, "error", err)
		}
		win.Update(func() {
			p.finishPicture(generation, picture, err)
			if generation == p.Generation && p.Open && err != nil {
				p.errorText = a.Texts.Error(err)
			}
		})
	}()
}
func (a *App) animatePreview(c *ui.Context) {
	p := &a.Preview
	if p.animation == nil || p.animationLoading || c.Preferences().ReduceMotion {
		return
	}
	if p.nextFrameAt.IsZero() {
		p.nextFrameAt = c.Now().Add(p.animation.delay)
	}
	if remaining := p.nextFrameAt.Sub(c.Now()); remaining > 0 {
		c.After(remaining)
		return
	}
	p.animationLoading = true
	generation, animation, id := p.Generation, p.animation, p.targetID
	go func() {
		picture, err := animation.next()
		if err != nil {
			slog.Error("预览动图播放失败", "asset", id, "error", err)
		}
		a.Window.Update(func() {
			if generation != p.Generation || !p.Open {
				return
			}
			p.animationLoading = false
			if err != nil {
				p.animation = nil
				p.errorText = a.Texts.Error(err)
				a.ShowError(err)
				return
			}
			p.Bitmap, p.pixelBitmap, p.animation = picture.bitmap, picture.pixelBitmap, picture.animation
			p.nextFrameAt = time.Time{}
		})
	}()
}

func (a *App) previewShortcut(panel ui.Element, action string) bool {
	if a.Settings.RecordingShortcut != "" {
		return false
	}
	mods, key, ok := parseShortcut(a.Shortcut(action))
	if !ok {
		return false
	}
	if panel.OverlayShortcut(mods, key) {
		return true
	}
	return action == "preview_zoom_in" && key == ui.KeyEqual && panel.OverlayShortcut(mods|ui.Shift, key)
}
