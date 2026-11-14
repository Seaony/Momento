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
	Open, Held                      bool
	Bitmap                          *ui.Bitmap
	Loading                         bool
	Generation                      uint64
	Zoom                            float32
	Pan                             ui.Point
	targetRoot, targetID, errorText string
	targetRotation                  int
	pixelBitmap                     *ui.Bitmap
	viewport                        ui.Rect
	fitZoom                         float32
	initialized, fitMode            bool
	animation                       *previewAnimation
	animationLoading                bool
	nextFrameAt, wheelAt            time.Time
	slideDirection                  float32
	slideFrames                     int
}

type previewAnimation struct {
	decoder  ui.VideoDecoder
	delay    time.Duration
	rotation int
}
type previewPicture struct {
	bitmap, pixelBitmap *ui.Bitmap
	animation           *previewAnimation
}

func (p *previewState) target(root, id string, rotation int) {
	if root == "" {
		p.slideDirection, p.slideFrames = 0, 0
	}
	if p.targetRoot == root && p.targetID == id && p.targetRotation == rotation {
		return
	}
	p.Generation++
	p.targetRoot, p.targetID, p.targetRotation = root, id, rotation
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
func readPreviewPicture(path string, rotation int) (previewPicture, error) {
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
		img, err := assets.RotateImage(frame.Image, rotation)
		if err != nil {
			return previewPicture{}, err
		}
		smooth, pixelated := previewBitmaps(img)
		return previewPicture{smooth, pixelated, &previewAnimation{decoder, frame.Duration, rotation}}, nil
	}
	img, _, _, _, err := assets.PrepareImage(path)
	if err != nil {
		return previewPicture{}, err
	}
	img, err = assets.RotateImage(img, rotation)
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
	img, err := assets.RotateImage(frame.Image, animation.rotation)
	if err != nil {
		return previewPicture{}, err
	}
	smooth, pixelated := previewBitmaps(img)
	return previewPicture{smooth, pixelated, &previewAnimation{animation.decoder, frame.Duration, animation.rotation}}, nil
}

// 图片放在图片卡里，四周留 previewPadding；可缩放、拖动，翻页时左右滑入。
func (w *previewWindow) viewport(c *ui.Context, a *App) {
	p, asset := &w.picture, w.asset
	width, height := c.Size()
	inset := float32(previewPadding)
	p.viewport = ui.Rect{X: inset, Y: inset, W: width - 2*inset, H: height - 2*inset}
	viewport := ui.Box(c.Key("preview-image")).Absolute().Left(inset - 1).Top(inset - 1).Right(inset - 1).Bottom(inset - 1)
	if !p.Open {
		viewport.PassThrough().AccessHidden(true)
		return
	}
	viewport.Label(asset.Name).HandleInput(func(ev ui.InputEvent) bool { return w.input(a, ev) })
	if p.slideDirection != 0 {
		viewport.ChildExit(ui.Motion{X: -64 * p.slideDirection, Scale: .98})
	}
	bitmap := p.Bitmap
	if bitmap == nil && p.errorText == "" {
		bitmap = w.placeholder
	}
	if bitmap != nil {
		bw, bh := bitmap.Size()
		if p.Bitmap == nil && asset.Width > 0 && asset.Height > 0 {
			bw, bh = assetDisplaySize(asset)
		}
		oldFit := p.fitZoom
		p.fitZoom = min(2, previewFitZoom(float32(bw), float32(bh), p.viewport.W, p.viewport.H))
		if !p.initialized {
			p.fitMode = w.pinned || a.Preferences.PreviewFit == "fit"
			p.Zoom = p.fitZoom
			if !p.fitMode {
				p.Zoom = min(1, p.Zoom)
			}
			p.initialized = true
		} else if oldFit != p.fitZoom && p.fitMode {
			p.Zoom, p.Pan = p.fitZoom, ui.Point{}
		}
		p.clampPan()
		viewport.Children(func() {
			if a.Preferences.PreviewPixelated && p.pixelBitmap != nil {
				bitmap = p.pixelBitmap
			}
			photo := previewPhoto(c.Key(struct{ ID string }{asset.ID}), bitmap, previewPhotoRadius, a.Preferences.PreviewChecker, ui.Contain).
				Absolute().Label(asset.Name)
			photoWidth, photoHeight := float32(bw)*p.Zoom, float32(bh)*p.Zoom
			photo.Left((p.viewport.W-photoWidth)/2+p.Pan.X).Top((p.viewport.H-photoHeight)/2+p.Pan.Y).Size(photoWidth, photoHeight)
			transition := ui.ElementTransition{Animation: &motionSpring, Colors: true, ScaleContent: true,
				Enter: &ui.Motion{Opacity: 1}, Exit: &ui.Motion{Scale: .94}}
			if p.slideDirection != 0 {
				transition.Enter = &ui.Motion{X: 64 * p.slideDirection, Scale: .98}
				transition.Exit = &ui.Motion{X: -64 * p.slideDirection, Scale: .98}
				p.slideFrames++
				if p.slideFrames > 1 && !photo.Transitioning() {
					p.slideDirection, p.slideFrames = 0, 0
					c.Invalidate()
				}
			}
			photo.Transition(transition)
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
				w.actualSize()
			} else {
				w.fitToWindow()
			}
		}
	} else if p.errorText != "" {
		viewport.Center().Children(func() {
			ui.Column(c).MaxWidth(previewInfoWidth).Padding(space16).Gap(space8).Children(func() {
				ui.Text(c, a.T("preview_failed")).FontSize(fontBody).FontWeight(600).TextColor(paletteFor(c.Theme()).Text)
				ui.Text(c, p.errorText).FontSize(fontSecondary).TextColor(paletteFor(c.Theme()).TextSecondary)
			})
		})
	}
	if p.Loading && bitmap == nil {
		ui.Text(c, a.T("preview_loading")).Absolute().Top(height/2).Left(width/2).TranslatePercent(-50, -50).
			FontSize(fontSecondary).TextColor(paletteFor(c.Theme()).TextSecondary)
	}
}

func previewPhoto(c *ui.Context, bitmap *ui.Bitmap, radius float32, checker bool, fit ui.Fit) ui.Element {
	photo := ui.Box(c).Radius(radius).Clip().Role(ui.RoleImage).Background(paletteFor(c.Theme()).Subtle)
	if checker {
		theme := c.Theme()
		photo.Draw(func(painter *ui.Painter, r ui.Rect) {
			painter.Clip(r, radius, func() { drawPreviewChecker(painter, r, r, theme) })
		})
	}
	photo.Children(func() { ui.Image(c, bitmap).Fill().Fit(fit).Radius(radius) })
	return photo
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
func (w *previewWindow) input(a *App, ev ui.InputEvent) bool {
	p := &w.picture
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
			w.move(a, delta)
			p.wheelAt = ev.Time
		}
		return true
	}
	return false
}
func (w *previewWindow) fitToWindow() {
	p := &w.picture
	if p.Bitmap != nil {
		p.Zoom, p.Pan, p.fitMode = p.fitZoom, ui.Point{}, true
	}
}
func (w *previewWindow) actualSize() {
	p := &w.picture
	if p.Bitmap != nil {
		p.Zoom, p.Pan, p.fitMode = 1, ui.Point{}, false
	}
}
func (a *App) PreviewZoomIn() {
	if w := a.previewTarget(); w != nil {
		p := &w.picture
		p.zoomAt(1.25, p.viewport.W/2, p.viewport.H/2)
		w.invalidate()
	}
}
func (a *App) PreviewZoomOut() {
	if w := a.previewTarget(); w != nil {
		p := &w.picture
		p.zoomAt(.8, p.viewport.W/2, p.viewport.H/2)
		w.invalidate()
	}
}
func (a *App) PreviewFitToWindow() {
	if w := a.previewTarget(); w != nil {
		w.fitToWindow()
		w.invalidate()
	}
}
func (a *App) PreviewActualSize() {
	if w := a.previewTarget(); w != nil {
		w.actualSize()
		w.invalidate()
	}
}
func drawPreviewChecker(p *ui.Painter, viewport, image ui.Rect, theme *ui.Theme) {
	left, top := max(viewport.X, image.X), max(viewport.Y, image.Y)
	right, bottom := min(viewport.X+viewport.W, image.X+image.W), min(viewport.Y+viewport.H, image.Y+image.H)
	if right <= left || bottom <= top {
		return
	}
	const size = float32(space12)
	palette := paletteFor(theme)
	for y := top; y < bottom; y += size {
		for x := left; x < right; x += size {
			color := palette.Subtle
			if (int((x-left)/size)+int((y-top)/size))%2 == 1 {
				color = palette.Hover
			}
			p.Fill(ui.Rect{X: x, Y: y, W: min(size, right-x), H: min(size, bottom-y)}, color, 0)
		}
	}
}

func (a *App) previewAsset(id string) (model.Asset, bool) {
	if asset, ok := a.assetByID[id]; ok {
		return asset, true
	}
	for _, asset := range a.Visible {
		if asset.ID == id {
			return asset, true
		}
	}
	return model.Asset{}, false
}
func (w *previewWindow) load(a *App) {
	p := &w.picture
	p.Loading = true
	generation, asset, root := p.Generation, w.asset, w.root
	go func() {
		picture, err := readPreviewPicture(filepath.Join(root, filepath.FromSlash(asset.RelativePath)), asset.Rotation)
		if err != nil {
			slog.Error("原图预览加载失败", "asset", asset.ID, "error", err)
		}
		w.update(func() {
			p.finishPicture(generation, picture, err)
			if generation == p.Generation && p.Open && err != nil {
				p.errorText = a.Texts.Error(err)
			}
		})
	}()
}
func (w *previewWindow) animatePicture(c *ui.Context, a *App) {
	p := &w.picture
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
		w.update(func() {
			if generation != p.Generation || !p.Open {
				return
			}
			p.animationLoading = false
			if err != nil {
				p.animation = nil
				p.errorText = a.Texts.Error(err)
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
