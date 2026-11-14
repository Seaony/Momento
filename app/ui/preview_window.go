package ui

import (
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const previewPeekThreshold = 250 * time.Millisecond
const previewPinThreshold = 3 * time.Second
const previewWindowDuration = 250 * time.Millisecond

// 预览照旧版 macOS（git 2a13147^ MomentoAssetPreviewOverlay.swift）：上面图片卡，下面单独一块信息卡。
const (
	previewPadding       = 14                    // 图片卡内边距
	previewCardRadius    = 30                    // 图片卡圆角
	previewPhotoRadius   = 12                    // 图片圆角
	previewInfoRadius    = 16                    // 信息卡圆角
	previewInfoHeight    = 52                    // 上下各 10 + 名称 + 3 + 尺寸
	previewInfoGap       = 12                    // 图片卡和信息卡之间
	previewInfoWidth     = 560                   // 信息卡固定宽度，屏幕放不下时才缩
	previewInfoInset     = 14                    // 信息卡左右内边距，关闭按钮贴着它
	previewInfoSide      = previewInfoInset + 48 // 文字两侧留给关闭按钮的位置，保证居中
	previewMinimumWidth  = 200
	previewMinimumHeight = 160
)

type previewWindows struct {
	active       *previewWindow
	pinned       []*previewWindow
	gesture      *previewWindow
	spaceDown    time.Time
	inputPending bool
}

type previewWindow struct {
	window, info                               *mygo.Window
	picture                                    previewState
	root                                       string
	asset                                      model.Asset
	items                                      []model.Asset
	placeholder                                *ui.Bitmap
	lastSelectionID                            string
	mainSelection                              []string
	pinned, closing, disposed                  bool
	imageFocused, infoFocused                  bool
	pressedAt, openedAt, closedAt, pinnedAt    time.Time
	frame, infoFrame                           mygo.Rectangle
	closeScale, closeOpacity                   float32
	closedHeld                                 bool
	closedProgress                             float32
	closedHintOpacity                          float32
	hostAnimating, infoAnimating, reduceMotion bool
}

func (a *App) PreviewOpen() bool { return a.previewTarget() != nil }
func (a *App) PreviewAssetID() string {
	if w := a.previewTarget(); w != nil {
		return w.asset.ID
	}
	return ""
}
func (a *App) PreviewHeld() bool { return a.Preview.gesture != nil }
func (a *App) previewTarget() *previewWindow {
	for _, w := range a.Preview.pinned {
		if w.focused() && !w.closing && !w.disposed {
			return w
		}
	}
	if w := a.Preview.active; w != nil && !w.closing && !w.disposed {
		return w
	}
	return nil
}

func (a *App) ShowPreview(id string, held bool) {
	if a.IsLocked() || a.Library == nil || a.Preview.gesture != nil {
		return
	}
	asset, ok := a.previewAsset(id)
	if !ok {
		return
	}
	if a.Preview.active != nil {
		return
	}
	now := time.Now()
	selection, preferred, _ := a.previewMainSelection("")
	w := &previewWindow{root: a.Library.Root, asset: asset, items: append([]model.Asset(nil), a.Visible...), openedAt: now, lastSelectionID: preferred.ID, mainSelection: selection}
	w.picture.Open = true
	if held {
		w.pressedAt = a.Preview.spaceDown
		if w.pressedAt.IsZero() {
			w.pressedAt = now
		}
		w.picture.Held = true
		a.Preview.gesture = w
	}
	a.Preview.active = w
	w.setAsset(a, asset)
	a.MarkAssetsUsed([]string{id})
	if a.Window != nil {
		w.create(a)
	}
}

func (a *App) ClosePreview() {
	if w := a.Preview.active; w != nil {
		w.close(a, time.Now())
	}
}

func (a *App) ObservePreviewInput(ev ui.InputEvent) {
	if ev.Key != ui.KeySpace {
		return
	}
	if ev.Kind == ui.InputKeyDown && !ev.Repeat {
		a.Preview.spaceDown = ev.Time
	}
	if ev.Kind == ui.InputKeyUp {
		if w := a.Preview.gesture; w != nil {
			w.release(a, ev.Time)
			a.Preview.gesture = nil
		}
		a.Preview.spaceDown = time.Time{}
	}
}

func (a *App) markPreviewInput()         { a.Preview.inputPending = true }
func (a *App) PreviewInputPending() bool { return a.Preview.inputPending }
func (a *App) TakePreviewInput() bool {
	pending := a.Preview.inputPending
	a.Preview.inputPending = false
	return pending
}

// 没有元素焦点时也读取窗口真实按键状态；KeyUp会自动请求这一帧。
func (a *App) ReconcilePreviewKeys(c *ui.Context) {
	held := c.KeyDown(ui.KeySpace)
	released := false
	if !held {
		if w := a.Preview.gesture; w != nil {
			w.release(a, c.Now())
			a.Preview.gesture = nil
			released = true
		}
		a.Preview.spaceDown = time.Time{}
	}
	w := a.Preview.active
	if w == nil || w.closing || a.IsLocked() || c.FocusTakesText() || a.Settings.RecordingShortcut != "" {
		return
	}
	space := previewSpaceShortcut(c)
	repeat := space && c.ShortcutRepeated()
	if c.Shortcut(0, ui.KeyEscape) || space && !repeat && !w.picture.Held && !released {
		w.close(a, c.Now())
		a.markPreviewInput()
		return
	}
	for _, entry := range []struct {
		action string
		delta  int
	}{{"preview_previous", -1}, {"preview_next", 1}} {
		if a.ActionShortcut(c, entry.action) {
			w.move(a, entry.delta)
			a.markPreviewInput()
			return
		}
	}
	for _, action := range []string{"preview_zoom_in", "preview_zoom_out", "preview_fit", "preview_actual"} {
		mods, key, ok := parseShortcut(a.Shortcut(action))
		if !ok {
			continue
		}
		pressed := c.Shortcut(mods, key)
		if action == "preview_zoom_in" && key == ui.KeyEqual {
			pressed = c.Shortcut(mods|ui.Shift, key) || pressed
		}
		if !pressed {
			continue
		}
		switch action {
		case "preview_zoom_in":
			a.PreviewZoomIn()
		case "preview_zoom_out":
			a.PreviewZoomOut()
		case "preview_fit":
			w.fitToWindow()
		case "preview_actual":
			w.actualSize()
		}
		w.invalidate()
		a.markPreviewInput()
		return
	}
}

// 根元素认领有焦点控件的空格，全窗入口处理没有元素焦点的空格。
func previewSpaceShortcut(c *ui.Context) bool {
	local := c.Root().Shortcut(0, ui.KeySpace)
	global := c.Shortcut(0, ui.KeySpace)
	return local || global
}

func (w *previewWindow) release(a *App, now time.Time) {
	if w.disposed || w.closing || !w.picture.Held {
		return
	}
	w.advanceHold(a, now)
	if !w.pinned && now.Sub(w.pressedAt) >= previewPeekThreshold {
		w.close(a, now)
	} else {
		w.picture.Held = false
		w.invalidate()
	}
}
func (w *previewWindow) advanceHold(a *App, now time.Time) {
	if !w.picture.Held || w.pinned || w.closing || now.Sub(w.pressedAt) < previewPinThreshold {
		return
	}
	w.pinned, w.pinnedAt = true, now
	w.picture.fitMode = true
	w.picture.Zoom, w.picture.Pan = w.picture.fitZoom, ui.Point{}
	w.hostAnimating = true
	if a.Preview.active == w {
		a.Preview.active = nil
	}
	a.Preview.pinned = append(a.Preview.pinned, w)
	w.infoAnimating = true
	if w.window != nil {
		w.window.SetAlwaysOnTop(true)
		w.window.SetMinimumSize(previewMinimumWidth, previewMinimumHeight)
		w.window.SetResizable(true)
		w.window.SetMovable(true)
	}
	a.previewMenuChanged()
}
func (w *previewWindow) holdProgress(now time.Time) float32 {
	if !w.picture.Held || w.pinned || now.Sub(w.pressedAt) < previewPeekThreshold {
		return 0
	}
	return min(1, float32(now.Sub(w.pressedAt)-previewPeekThreshold)/float32(previewPinThreshold-previewPeekThreshold))
}

func (a *App) previewMainSelection(previous string) ([]string, model.Asset, bool) {
	selected := a.SelectedAssets()
	ids := make([]string, len(selected))
	for i, asset := range selected {
		ids[i] = asset.ID
	}
	slices.Sort(ids)
	if len(selected) == 0 {
		return ids, model.Asset{}, false
	}
	if a.selectedIndex >= 0 && a.selectedIndex < len(a.Visible) && a.Selected.Has(a.Visible[a.selectedIndex].ID) {
		return ids, a.Visible[a.selectedIndex], true
	}
	if a.Selected.Has(previous) {
		if asset, ok := a.previewAsset(previous); ok {
			return ids, asset, true
		}
	}
	for _, asset := range a.Visible {
		if a.Selected.Has(asset.ID) {
			return ids, asset, true
		}
	}
	return ids, selected[0], true
}

func (a *App) FollowPreviewSelection() {
	w := a.Preview.active
	if w == nil || w.closing {
		return
	}
	selection, asset, found := a.previewMainSelection(w.lastSelectionID)
	if slices.Equal(selection, w.mainSelection) && asset.ID == w.lastSelectionID {
		return
	}
	w.mainSelection, w.lastSelectionID = selection, asset.ID
	if !found {
		w.close(a, time.Now())
		return
	}
	if !slices.ContainsFunc(w.items, func(item model.Asset) bool { return item.ID == asset.ID }) {
		w.items = append([]model.Asset(nil), a.Visible...)
	}
	w.setAsset(a, asset)
}

func (a *App) CancelPreviewGesture() {
	if w := a.Preview.gesture; w != nil {
		if !w.pinned {
			w.close(a, time.Now())
		} else {
			w.picture.Held = false
			w.invalidate()
		}
	}
	a.Preview.gesture = nil
	a.Preview.spaceDown = time.Time{}
}
func (a *App) CloseAllPreviewWindows() {
	windows := append([]*previewWindow(nil), a.Preview.pinned...)
	if a.Preview.active != nil {
		windows = append(windows, a.Preview.active)
	}
	a.Preview = previewWindows{}
	for _, w := range windows {
		w.dispose()
		w.destroyWindows()
	}
	a.previewMenuChanged()
}
func (a *App) InvalidatePreviewWindows() {
	for _, w := range a.Preview.pinned {
		w.invalidate()
	}
	if w := a.Preview.active; w != nil {
		w.invalidate()
	}
}
func (a *App) previewMenuChanged() {
	if a.MenuChanged != nil {
		a.MenuChanged()
	}
	if a.Window != nil {
		a.Window.Update(func() {})
	}
}

func (w *previewWindow) focused() bool { return w.imageFocused || w.infoFocused }

// 两个窗口都用 ShowInactive 显示，不抢主窗口键盘，主窗口才能判断空格是否还按着。
func (w *previewWindow) create(a *App) {
	parent := a.Window.Bounds()
	area := mygo.Screen.DisplayMatching(parent).WorkArea
	w.frame, w.infoFrame = previewLayout(parent, area, w.asset, a.Preferences.PreviewFit == "fit")
	w.hostAnimating = true
	glass := func(frame mygo.Rectangle, radius int, parentWindow *mygo.Window, view func(c *ui.Context)) *mygo.Window {
		return mygo.NewWindow(mygo.WindowOptions{Title: w.asset.Name, Width: frame.Width, Height: frame.Height,
			Hidden: true, Frameless: true, Transparent: true, CornerRadius: radius, SkipTaskbar: true,
			DisableResize: true, DisableMaximize: true, DisableMinimize: true, DisableFullScreen: true,
			Vibrancy: mygo.VibrancyGlass, VibrancyTint: previewGlassTint(), Parent: parentWindow,
			BackgroundColor: "rgba(0,0,0,0)", Content: ui.View(view)})
	}
	w.window = glass(w.frame, previewCardRadius, nil, func(c *ui.Context) { w.view(c, a) })
	w.info = glass(w.infoFrame, previewInfoRadius, w.window, func(c *ui.Context) { w.infoView(c, a) })
	w.window.SetMinimumSize(1, 1)
	w.info.SetMinimumSize(1, 1)
	w.applyFrames(.92, 0, 0)
	w.window.OnClose(func(e *mygo.CloseEvent) { e.PreventDefault(); w.close(a, time.Now()) })
	w.info.OnClose(func(e *mygo.CloseEvent) { e.PreventDefault(); w.close(a, time.Now()) })
	w.window.OnClosed(func() { w.dispose(); a.removePreviewWindow(w) })
	w.window.OnFocus(func() { w.imageFocused = true; a.previewMenuChanged() })
	w.window.OnBlur(func() { w.imageFocused = false; a.previewMenuChanged() })
	w.info.OnFocus(func() { w.infoFocused = true; a.previewMenuChanged() })
	w.info.OnBlur(func() { w.infoFocused = false; a.previewMenuChanged() })
	keep := func() {
		if w.pinned && !w.closing && !w.hostAnimating {
			w.frame = w.window.Bounds()
		}
	}
	w.window.OnMove(keep)
	w.window.OnResize(keep)
	w.window.ShowInactive()
	w.info.ShowInactive()
}

// 图片卡和信息卡作为一组，在主窗口中央；图片按比例放进最多 80% 主窗口的范围。
func previewLayout(parent, area mygo.Rectangle, asset model.Asset, enlarge bool) (image, info mygo.Rectangle) {
	maxW, maxH := max(1, parent.Width*8/10), max(1, parent.Height*8/10)
	if area.Width > 0 {
		maxW = min(maxW, area.Width)
	}
	if area.Height > 0 {
		maxH = min(maxH, area.Height)
	}
	iw, ih := assetDisplaySize(asset)
	limit := float32(1)
	if enlarge {
		limit = 2
	}
	pad := 2 * previewPadding
	maxImageH := maxH - previewInfoGap - previewInfoHeight
	scale := min(limit, previewFitZoom(float32(iw), float32(ih), float32(maxW-pad), float32(maxImageH-pad)))
	width := min(maxW, max(previewMinimumWidth, int(float32(iw)*scale)+pad))
	height := min(maxImageH, max(previewMinimumHeight, int(float32(ih)*scale)+pad))
	infoWidth := min(maxW, previewInfoWidth)
	total := height + previewInfoGap + previewInfoHeight
	x, y := parent.X+(parent.Width-width)/2, parent.Y+(parent.Height-total)/2
	if area.Width > 0 {
		x = max(area.X, min(x, area.X+area.Width-width))
	}
	if area.Height > 0 {
		y = max(area.Y, min(y, area.Y+area.Height-total))
	}
	image = mygo.Rectangle{X: x, Y: y, Width: width, Height: height}
	info = mygo.Rectangle{X: x + (width-infoWidth)/2, Y: y + height + previewInfoGap, Width: infoWidth, Height: previewInfoHeight}
	return
}

// 以 center 为中心缩放一个窗口矩形，两块卡片围绕同一个中心一起放大缩小。
func scaledAround(r mygo.Rectangle, cx, cy, scale float32) mygo.Rectangle {
	x := cx + (float32(r.X)-cx)*scale
	y := cy + (float32(r.Y)-cy)*scale
	return mygo.Rectangle{X: int(math.Round(float64(x))), Y: int(math.Round(float64(y))),
		Width: int(math.Round(float64(float32(r.Width) * scale))), Height: int(math.Round(float64(float32(r.Height) * scale)))}
}

// 把当前动画帧的缩放和透明度同步到两个窗口；固定后图片卡只围绕自己弹一下。
func (w *previewWindow) applyFrames(scale, opacity, infoOpacity float32) {
	if w.window == nil {
		return
	}
	cx, cy := float32(w.frame.X)+float32(w.frame.Width)/2, float32(w.frame.Y)+float32(w.frame.Height)/2
	if !w.pinned {
		top, bottom := w.frame.Y, w.infoFrame.Y+w.infoFrame.Height
		cy = float32(top+bottom) / 2
	}
	w.window.SetBounds(scaledAround(w.frame, cx, cy, scale))
	w.window.SetOpacity(float64(opacity))
	if w.info != nil {
		w.info.SetBounds(scaledAround(w.infoFrame, cx, cy, scale))
		w.info.SetOpacity(float64(infoOpacity))
	}
}

func previewReveal(t float32) float32 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return float32(1 - math.Exp(-7*float64(t))*math.Cos(8*float64(t)))
}
func (w *previewWindow) visual(now time.Time) (scale, opacity float32) {
	if w.closing {
		t := min(1, max(0, float32(now.Sub(w.closedAt))/float32(previewWindowDuration)))
		eased := 1 - (1-t)*(1-t)*(1-t)
		return w.closeScale + (.92-w.closeScale)*eased, w.closeOpacity * (1 - eased)
	}
	reveal := previewReveal(float32(now.Sub(w.openedAt)) / float32(previewWindowDuration))
	scale, opacity = .92+.08*reveal, min(1, max(0, reveal))
	if w.pinned {
		t := min(1, max(0, float32(now.Sub(w.pinnedAt))/float32(previewWindowDuration)))
		scale *= 1 + .02*float32(math.Sin(math.Pi*float64(t)))
	}
	return
}

// 固定后信息卡淡出，只留图片卡。
func (w *previewWindow) infoOpacity(now time.Time, opacity float32) float32 {
	if !w.pinned {
		return opacity
	}
	t := min(1, max(0, float32(now.Sub(w.pinnedAt))/float32(previewWindowDuration)))
	return opacity * (1 - t)
}

func (w *previewWindow) close(a *App, now time.Time) {
	if w.closing || w.disposed {
		return
	}
	w.closeScale, w.closeOpacity = w.visual(now)
	w.closedHeld = w.picture.Held && !w.pinned
	w.closedProgress = w.holdProgress(now)
	w.closedHintOpacity = min(1, max(0, float32(now.Sub(w.pressedAt)-previewPeekThreshold)/float32(120*time.Millisecond)))
	w.closing, w.closedAt = true, now
	w.picture.Held = false
	if w.window != nil {
		w.frame = w.window.Bounds()
		w.closeScale = 1
		w.window.SetMinimumSize(1, 1)
	}
	w.invalidate()
	if w.window == nil || w.reduceMotion {
		w.dispose()
		w.destroyWindows()
		a.removePreviewWindow(w)
	}
	a.previewMenuChanged()
}
func (w *previewWindow) destroyWindows() {
	if w.info != nil {
		w.info.Hide()
		w.info.Destroy()
		w.info = nil
	}
	if w.window != nil {
		w.window.Hide()
		w.window.Destroy()
	}
}
func (w *previewWindow) dispose() {
	w.disposed = true
	w.picture.Open = false
	w.picture.target("", "", 0)
	w.placeholder = nil
}
func (a *App) removePreviewWindow(w *previewWindow) {
	if a.Preview.active == w {
		a.Preview.active = nil
	}
	a.Preview.pinned = slices.DeleteFunc(a.Preview.pinned, func(item *previewWindow) bool { return item == w })
	if a.Preview.gesture == w {
		a.Preview.gesture = nil
	}
	a.previewMenuChanged()
}
func (w *previewWindow) update(fn func()) {
	if w.window != nil {
		w.window.Update(func() {
			if !w.disposed {
				fn()
			}
		})
	}
}
func (w *previewWindow) invalidate() {
	if w.disposed {
		return
	}
	if w.window != nil {
		w.window.Update(func() {})
	}
	if w.info != nil {
		w.info.Update(func() {})
	}
}
func (w *previewWindow) setAsset(a *App, asset model.Asset) {
	w.asset = asset
	w.picture.target(w.root, asset.ID, asset.Rotation)
	w.placeholder = nil
	if a.Thumbnails != nil {
		w.placeholder = a.Thumbnails.Get(w.root, asset)
	}
	if w.window != nil {
		w.window.SetTitle(asset.Name)
	}
	if w.info != nil {
		w.info.SetTitle(asset.Name)
	}
	w.invalidate()
}
func (w *previewWindow) move(a *App, delta int) {
	index := slices.IndexFunc(w.items, func(asset model.Asset) bool { return asset.ID == w.asset.ID })
	if index < 0 || index+delta < 0 || index+delta >= len(w.items) {
		return
	}
	w.picture.slideDirection, w.picture.slideFrames = float32(delta), 0
	w.setAsset(a, w.items[index+delta])
	if a.Library != nil && a.Library.Root == w.root {
		a.MarkAssetsUsed([]string{w.asset.ID})
	}
}

// 两块卡片共用的磨砂底：系统材质上再铺一层很淡的色板色，浅色发白、深色近黑。
func previewCard(c *ui.Context, key any, radius float32) ui.Element {
	p := paletteFor(c.Theme())
	tint := p.Raised.Alpha(materialPanelAlpha)
	if c.Theme().Dark {
		tint = p.Background.Alpha(materialPanelAlpha)
	}
	return ui.Box(c.Key(key)).Fill().Radius(radius).Clip().Background(tint).Border(1, p.BorderSubtle)
}

// 图片卡窗口：只放图片；固定后悬停右上角出现关闭按钮。动画帧由这个窗口驱动。
func (w *previewWindow) view(c *ui.Context, a *App) {
	a.applyTheme(c)
	c.Root().Background(ui.Transparent)
	now := c.Now()
	w.reduceMotion = c.Preferences().ReduceMotion
	w.advanceHold(a, now)
	scale, opacity := w.visual(now)
	animating := now.Sub(w.openedAt) < previewWindowDuration || w.closing || w.pinned && now.Sub(w.pinnedAt) < previewWindowDuration
	if w.reduceMotion {
		scale, opacity = 1, 1
		animating = false
	}
	if w.window != nil && (animating || w.hostAnimating) {
		w.applyFrames(scale, opacity, w.infoOpacity(now, opacity))
		w.hostAnimating = animating
	}
	// 信息卡淡出结束后关掉它，固定窗口只剩图片。
	if w.pinned && w.info != nil && (w.reduceMotion || now.Sub(w.pinnedAt) >= previewWindowDuration) {
		w.info.Hide()
		w.info.Destroy()
		w.info, w.infoFocused, w.infoAnimating = nil, false, false
	}

	if w.closing && (w.reduceMotion || now.Sub(w.closedAt) >= previewWindowDuration) {
		w.dispose()
		w.destroyWindows()
		a.removePreviewWindow(w)
		return
	}
	if animating {
		c.After(16 * time.Millisecond)
	}
	if w.picture.Held && !w.pinned {
		deadline := w.pressedAt.Add(previewPinThreshold)
		if now.Sub(w.pressedAt) < previewPeekThreshold {
			deadline = w.pressedAt.Add(previewPeekThreshold)
		}
		c.After(min(16*time.Millisecond, max(time.Duration(0), deadline.Sub(now))))
		if w.info != nil {
			w.info.Update(func() {})
		}
	}
	if w.placeholder == nil && a.Thumbnails != nil {
		w.placeholder = a.Thumbnails.Get(w.root, w.asset)
	}
	p := &w.picture
	if p.Bitmap == nil && p.errorText == "" && !p.Loading && w.window != nil {
		w.load(a)
	}
	w.animatePicture(c, a)
	surface := previewCard(c, "preview-window-surface", previewCardRadius)
	if w.pinned {
		surface.DragWindow()
	}
	surface.Children(func() {
		w.viewport(c, a)
		if w.pinned {
			close := previewCardButton(c.Key("close-pinned-preview"), "close", a.T("close_preview"), false).
				Absolute().Top(previewPadding).Right(previewPadding).Material(panelMaterial(c.Theme()))
			visible := surface.Hovered()
			close.Opacity(close.AnimateTo("visible", motionTarget(visible), motionColor)).IgnoreInput(!visible)
			if close.Clicked() {
				w.close(a, c.Now())
			}
		}
	})
	if w.imageFocused {
		w.handleKeys(c, a, now)
	}
}

// 信息卡窗口：名称和尺寸居中，右侧圆形关闭按钮；按住空格时第二行换成提示，底部出进度条。
func (w *previewWindow) infoView(c *ui.Context, a *App) {
	a.applyTheme(c)
	c.Root().Background(ui.Transparent)
	now := c.Now()
	pal := paletteFor(c.Theme())
	elapsed := now.Sub(w.pressedAt)
	pinFade := w.pinned && now.Sub(w.pinnedAt) < 120*time.Millisecond
	holding := w.picture.Held && elapsed >= previewPeekThreshold && !w.pinned || w.closing && w.closedHeld && elapsed >= previewPeekThreshold || pinFade
	card := previewCard(c, "preview-info", previewInfoRadius)
	card.Children(func() {
		ui.Column(c.Key("preview-info-text")).Absolute().Left(previewInfoSide).Right(previewInfoSide).Top(0).Bottom(0).
			Justify(ui.Center).AlignItems(ui.Center).Gap(3).Children(func() {
			ui.Text(c, w.asset.Name).FontSize(fontBody).FontWeight(500).TextColor(pal.Text).SingleLine().TextAlign(ui.Center).MaxWidthPercent(100).Tooltip(w.asset.Name)
			if holding {
				ui.Text(c.Key("preview-hold-hint"), a.T("preview_hold_to_pin")).FontSize(fontGroup).TextColor(pal.TextSecondary).SingleLine()
			} else if w.picture.errorText != "" && w.picture.Bitmap != nil {
				ui.Text(c, w.picture.errorText).FontSize(fontGroup).TextColor(pal.Danger).SingleLine()
			} else {
				ui.Text(c.Key("preview-meta"), previewSubtitle(w.asset)).FontSize(fontGroup).FontFeatures("tnum").TextColor(pal.TextSecondary).SingleLine().TextAlign(ui.Center).MaxWidthPercent(100)
			}
		})
		if !w.picture.Held && !w.closedHeld && !w.pinned {
			ui.Row(c.Key("preview-close")).Absolute().Right(previewInfoInset).Top(0).Bottom(0).AlignItems(ui.Center).Children(func() {
				if previewCardButton(c.Key("close-preview"), "close", a.T("close_preview"), false).Clicked() {
					w.close(a, c.Now())
				}
			})
		}
		if holding {
			value := w.holdProgress(now)
			opacity := min(1, max(0, float32(elapsed-previewPeekThreshold)/float32(120*time.Millisecond)))
			if w.closing {
				value, opacity = w.closedProgress, w.closedHintOpacity
			}
			if pinFade {
				value, opacity = 1, 1-float32(now.Sub(w.pinnedAt))/float32(120*time.Millisecond)
			}
			ui.Box(c.Key("preview-hold-progress")).Absolute().Left(previewInfoInset).Right(previewInfoInset).Bottom(space4).Height(space4 / 2).
				Radius(space4 / 4).Background(pal.BorderSubtle).IgnoreInput(true).Opacity(opacity).Children(func() {
				ui.Box(c).WidthPercent(value * 100).FillHeight().Radius(space4 / 4).Background(pal.Accent)
			})
		}
	})
	if w.infoFocused {
		w.handleKeys(c, a, now)
	}
}

// 旧版第二行只有「宽 × 高」，没有尺寸时显示文件名。
func previewSubtitle(asset model.Asset) string {
	if width, height := assetDisplaySize(asset); width > 0 && height > 0 {
		return fmt.Sprintf("%d × %d", width, height)
	}
	return asset.Name
}

// 卡片上的圆形磨砂按钮（旧版的玻璃按钮）：常驻浅底，悬停加深。
func previewCardButton(c *ui.Context, symbol, label string, disabled bool) ui.Element {
	p := paletteFor(c.Theme())
	button := ui.ButtonBase(c).Size(controlSmall, controlSmall).Padding(0).Radius(controlSmall / 2).
		Label(label).Tooltip(label).Disabled(disabled).Cursor(ui.CursorPointer).Justify(ui.Center).AlignItems(ui.Center)
	background, text := p.Hover, p.Text
	if button.Pressed() || button.Hovered() {
		background = p.Pressed
	}
	animateColors(button, background, text)
	button.Children(func() { appIcon(c, symbol).Size(iconSmall, iconSmall) })
	return button
}

// 预览窗口自己拿到键盘时（点过它）处理关闭、翻页和缩放。
func (w *previewWindow) handleKeys(c *ui.Context, a *App, now time.Time) {
	if c.Shortcut(0, ui.KeyEscape) || c.Shortcut(ui.Cmd, ui.KeyW) || !w.pinned && previewSpaceShortcut(c) {
		w.close(a, now)
		return
	}
	if a.Settings.RecordingShortcut != "" {
		return
	}
	p := &w.picture
	root := c.Root()
	handled := true
	switch {
	case a.previewShortcut(root, "preview_previous"):
		w.move(a, -1)
	case a.previewShortcut(root, "preview_next"):
		w.move(a, 1)
	case a.previewShortcut(root, "preview_zoom_in"):
		p.zoomAt(1.25, p.viewport.W/2, p.viewport.H/2)
	case a.previewShortcut(root, "preview_zoom_out"):
		p.zoomAt(.8, p.viewport.W/2, p.viewport.H/2)
	case a.previewShortcut(root, "preview_fit"):
		w.fitToWindow()
	case a.previewShortcut(root, "preview_actual"):
		w.actualSize()
	default:
		handled = false
	}
	// 键盘可能在信息卡窗口上，翻页和缩放改的是图片卡，要让两个窗口都重画。
	if handled {
		w.invalidate()
	}
}
