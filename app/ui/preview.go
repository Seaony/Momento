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
	rotationBitmap, rotationPixels  *ui.Bitmap
	rotationFrom, rotationTo        float32
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
	if root != "" && p.targetID == id && p.targetRotation != rotation {
		p.rotationBitmap, p.rotationPixels = p.Bitmap, p.pixelBitmap
		p.rotationFrom = p.rotationTo
		delta := (rotation-p.targetRotation+540)%360 - 180
		p.rotationTo += float32(delta)
	} else {
		p.rotationBitmap, p.rotationPixels = nil, nil
		p.rotationFrom, p.rotationTo = 0, 0
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

func previewSymbolButton(c *ui.Context, symbol, label string, size float32, disabled bool) ui.Element {
	p := paletteFor(c.Theme())
	button := elevated(ui.ButtonBase(c).Size(size, size).Padding(0).Radius(size/2).Material(panelMaterial(c.Theme())), c.Theme()).
		Label(label).Tooltip(label).Disabled(disabled).Cursor(ui.CursorPointer)
	background, text := ui.Transparent, p.TextSecondary
	if button.Pressed() {
		background, text = p.Pressed, p.Text
	} else if button.Hovered() {
		background, text = p.Hover, p.Text
	}
	animateColors(button, background, text)
	button.Children(func() { appIcon(c, symbol).Size(18, 18) })
	return button
}

func (w *previewWindow) viewport(c *ui.Context, a *App) {
	p, asset := &w.picture, w.asset
	width, height := c.Size()
	p.viewport = ui.Rect{X: space24 + 1, Y: space64 + 1, W: width - 2*(space24+1), H: height - space64 - space48 - 2}
	viewport := ui.Box(c.Key("preview-image")).Absolute().Left(space24).Top(space64).Right(space24).Bottom(space48)
	if !p.Open {
		viewport.PassThrough().AccessHidden(true)
		return
	}
	viewport.Label(asset.Name).HandleInput(func(ev ui.InputEvent) bool { return w.input(a, ev) })
	if p.slideDirection != 0 {
		viewport.ChildExit(ui.Motion{X: -64 * p.slideDirection, Scale: .98})
	}
	bitmap, bitmapTurn := p.Bitmap, p.rotationTo
	if p.rotationBitmap != nil {
		bitmap, bitmapTurn = p.rotationBitmap, p.rotationFrom
	}
	if bitmap == nil && p.errorText == "" {
		bitmap = w.placeholder
	}
	if bitmap != nil {
		bw, bh := bitmap.Size()
		if p.Bitmap == nil && p.rotationBitmap == nil && asset.Width > 0 && asset.Height > 0 {
			bw, bh = assetDisplaySize(asset)
		}
		fitWidth, fitHeight := bw, bh
		if p.rotationBitmap != nil {
			if p.Bitmap != nil {
				fitWidth, fitHeight = p.Bitmap.Size()
			} else if int(p.rotationTo-p.rotationFrom)%180 != 0 {
				fitWidth, fitHeight = fitHeight, fitWidth
			}
		}
		oldFit := p.fitZoom
		p.fitZoom = min(2, previewFitZoom(float32(fitWidth), float32(fitHeight), p.viewport.W, p.viewport.H))
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
			if a.Preferences.PreviewPixelated {
				if p.rotationPixels != nil {
					bitmap = p.rotationPixels
				} else if p.pixelBitmap != nil {
					bitmap = p.pixelBitmap
				}
			}
			photo := previewPhoto(c.Key(struct{ ID string }{asset.ID}), bitmap, radiusPanel, a.Preferences.PreviewChecker, ui.Contain).
				Absolute().Label(asset.Name).Shadow(0, space4, space16, 0, paletteFor(c.Theme()).Shadow)
			zoom := p.Zoom
			if p.rotationBitmap != nil {
				zoom = photo.AnimateTo("rotation-scale", p.Zoom, motionSpring)
			} else {
				photo.Jump("rotation-scale", p.Zoom)
			}
			turn := photo.AnimateTo("rotation", p.rotationTo, motionSpring)
			if p.rotationBitmap != nil {
				angle := float64(turn-bitmapTurn) * math.Pi / 180
				cosine, sine := float32(math.Abs(math.Cos(angle))), float32(math.Abs(math.Sin(angle)))
				rotatedWidth := float32(bw)*cosine + float32(bh)*sine
				rotatedHeight := float32(bw)*sine + float32(bh)*cosine
				zoom = min(zoom, previewFitZoom(rotatedWidth, rotatedHeight, p.viewport.W, p.viewport.H))
			}
			photoWidth, photoHeight := float32(bw)*zoom, float32(bh)*zoom
			photo.Left((p.viewport.W-photoWidth)/2+p.Pan.X).Top((p.viewport.H-photoHeight)/2+p.Pan.Y).Size(photoWidth, photoHeight)
			photo.Rotate(turn - bitmapTurn)
			transition := ui.ElementTransition{Animation: &motionSpring, Colors: true, ScaleContent: true,
				Enter: &ui.Motion{Scale: .94}, Exit: &ui.Motion{Scale: .94}}
			if p.slideDirection == 0 {
				transition.Enter = &ui.Motion{Opacity: 1}
			} else {
				transition.Enter = &ui.Motion{X: 64 * p.slideDirection, Scale: .98}
				transition.Exit = &ui.Motion{X: -64 * p.slideDirection, Scale: .98}
				p.slideFrames++
				if p.slideFrames > 1 && !photo.Transitioning() {
					p.slideDirection, p.slideFrames = 0, 0
					c.Invalidate()
				}
			}
			photo.Transition(transition)
			if p.rotationBitmap != nil && p.Bitmap != nil && photo.Settled("rotation") && photo.Settled("rotation-scale") {
				p.rotationBitmap, p.rotationPixels = nil, nil
				c.Invalidate()
			}
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
			ui.Column(c).MaxWidth(560).Padding(space16).Gap(space8).Radius(radiusPanel).Material(panelMaterial(c.Theme())).Children(func() {
				ui.Text(c, a.T("preview_failed")).FontSize(fontBody).FontWeight(600).TextColor(paletteFor(c.Theme()).Text)
				ui.Text(c, p.errorText).FontSize(fontSecondary).TextColor(paletteFor(c.Theme()).TextSecondary)
			})
		})
	}
	if p.Loading {
		ui.Text(c, a.T("preview_loading")).Absolute().Top(space64).Left(width/2).TranslatePercent(-50, 0).
			FontSize(fontSecondary).Padding(space8, space12).Radius(radiusPanel).Material(panelMaterial(c.Theme())).TextColor(paletteFor(c.Theme()).TextSecondary)
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
				w.operationError = p.errorText
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
