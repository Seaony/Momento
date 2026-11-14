package ui

import (
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

type previewWindows struct {
	active       *previewWindow
	pinned       []*previewWindow
	gesture      *previewWindow
	spaceDown    time.Time
	inputPending bool
}

type previewWindow struct {
	window                                       *mygo.Window
	picture                                      previewState
	root                                         string
	asset                                        model.Asset
	items                                        []model.Asset
	placeholder                                  *ui.Bitmap
	operationError                               string
	lastSelectionID                              string
	mainSelection                                []string
	pinned, focused, closing, disposed, rotating bool
	pressedAt, openedAt, closedAt, pinnedAt      time.Time
	frame                                        mygo.Rectangle
	closeScale, closeOpacity                     float32
	closedHeld                                   bool
	closedProgress                               float32
	closedHintOpacity                            float32
	hostAnimating, reduceMotion                  bool
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
		if w.focused && !w.closing && !w.disposed {
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
	if w.window != nil {
		w.window.SetAlwaysOnTop(true)
		w.window.SetMinimumSize(300, 220)
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
		if w.window != nil {
			w.window.Hide()
			w.window.Destroy()
		}
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

func (w *previewWindow) create(a *App) {
	parent := a.Window.Bounds()
	area := mygo.Screen.DisplayMatching(parent).WorkArea
	w.frame = previewWindowBounds(parent, area, w.asset, a.Preferences.PreviewFit == "fit")
	w.hostAnimating = true
	w.window = mygo.NewWindow(mygo.WindowOptions{Title: w.asset.Name, Width: w.frame.Width, Height: w.frame.Height,
		Hidden: true, Frameless: true, Transparent: true, CornerRadius: 12, SkipTaskbar: true,
		DisableResize: true, DisableMaximize: true, DisableMinimize: true, DisableFullScreen: true,
		Vibrancy: mygo.VibrancyGlass, VibrancyTint: "light-dark(rgba(255,250,245,0.28), rgba(33,26,22,0.35))",
		BackgroundColor: "rgba(0,0,0,0)", Content: ui.View(func(c *ui.Context) { w.view(c, a) })})
	w.window.SetMinimumSize(160, 120)
	w.window.SetBounds(scaledPreviewBounds(w.frame, .92))
	w.window.SetOpacity(0)
	w.window.OnClose(func(e *mygo.CloseEvent) { e.PreventDefault(); w.close(a, time.Now()) })
	w.window.OnClosed(func() { w.dispose(); a.removePreviewWindow(w) })
	w.window.OnFocus(func() { w.focused = true; a.previewMenuChanged() })
	w.window.OnBlur(func() { w.focused = false; a.previewMenuChanged() })
	w.window.OnMove(func() {
		if w.pinned && !w.closing && !w.hostAnimating {
			w.frame = w.window.Bounds()
		}
	})
	w.window.OnResize(func() {
		if w.pinned && !w.closing && !w.hostAnimating {
			w.frame = w.window.Bounds()
		}
	})
	w.window.ShowInactive()
}
func previewWindowBounds(parent, area mygo.Rectangle, asset model.Asset, enlarge bool) mygo.Rectangle {
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
	scale := min(limit, previewFitZoom(float32(iw), float32(ih), float32(maxW-50), float32(maxH-114)))
	width, height := min(maxW, max(300, int(float32(iw)*scale)+50)), min(maxH, max(220, int(float32(ih)*scale)+114))
	x, y := parent.X+(parent.Width-width)/2, parent.Y+(parent.Height-height)/2
	if area.Width > 0 {
		x = max(area.X, min(x, area.X+area.Width-width))
	}
	if area.Height > 0 {
		y = max(area.Y, min(y, area.Y+area.Height-height))
	}
	return mygo.Rectangle{X: x, Y: y, Width: width, Height: height}
}
func scaledPreviewBounds(r mygo.Rectangle, scale float32) mygo.Rectangle {
	width, height := int(math.Round(float64(float32(r.Width)*scale))), int(math.Round(float64(float32(r.Height)*scale)))
	return mygo.Rectangle{X: r.X + (r.Width-width)/2, Y: r.Y + (r.Height-height)/2, Width: width, Height: height}
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
		if w.window != nil {
			w.window.Hide()
			w.window.Destroy()
		}
		a.removePreviewWindow(w)
	}
	a.previewMenuChanged()
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
	if w.window != nil && !w.disposed {
		w.window.Update(func() {})
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
		w.invalidate()
	}
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
		w.window.SetBounds(scaledPreviewBounds(w.frame, scale))
		w.window.SetOpacity(float64(opacity))
		w.hostAnimating = animating
	}

	if w.closing && (w.reduceMotion || now.Sub(w.closedAt) >= previewWindowDuration) {
		w.dispose()
		if w.window != nil {
			w.window.Destroy()
		}
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
	}
	if w.placeholder == nil && a.Thumbnails != nil {
		w.placeholder = a.Thumbnails.Get(w.root, w.asset)
	}
	p := &w.picture
	if p.Bitmap == nil && p.errorText == "" && !p.Loading && w.window != nil {
		w.load(a)
	}
	w.animatePicture(c, a)
	surface := ui.Box(c.Key("preview-window-surface")).Fill().Radius(12).Clip().Material(frostedSurface{tint: c.Theme().Background.Alpha(.72), blur: 24}).Border(1, c.Theme().Border)
	if w.pinned {
		surface.DragWindow()
	}
	surface.Children(func() {
		w.viewport(c, a)
		w.controls(c, a, surface.Hovered())
		if w.operationError != "" {
			ui.Column(c).Absolute().Top(64).Left(16).Right(16).Padding(8).Radius(8).Material(panelMaterial(c.Theme())).Children(func() { ui.Text(c, w.operationError).FontSize(12).TextColor(c.Theme().Danger) })
		}

		elapsed := now.Sub(w.pressedAt)
		pinFade := w.pinned && now.Sub(w.pinnedAt) < 120*time.Millisecond
		showHold := w.picture.Held && elapsed >= previewPeekThreshold && !w.pinned || w.closing && w.closedHeld && elapsed >= previewPeekThreshold || pinFade
		if showHold {
			value := w.holdProgress(now)
			opacity := min(1, max(0, float32(elapsed-previewPeekThreshold)/float32(120*time.Millisecond)))
			if w.closing {
				value = w.closedProgress
				opacity = w.closedHintOpacity
			}
			if pinFade {
				value = 1
				opacity = 1 - float32(now.Sub(w.pinnedAt))/float32(120*time.Millisecond)
			}
			progress := ui.Column(c.Key("preview-hold-progress")).Absolute().Bottom(12).Left(24).Right(24).Gap(8).AlignItems(ui.Center).IgnoreInput(true).Opacity(opacity)
			progress.Children(func() {
				ui.Text(c, a.T("preview_hold_to_pin")).FontSize(12).TextColor(c.Theme().TextMuted).SingleLine()
				ui.Box(c).Width(180).Height(3).Radius(2).Background(c.Theme().Border).Children(func() { ui.Box(c).WidthPercent(value * 100).Height(3).Radius(2).Background(c.Theme().Accent) })
			})
		}

	})
	if (w.focused || w.pinned) && (c.Shortcut(0, ui.KeyEscape) || c.Shortcut(ui.Cmd, ui.KeyW) || !w.pinned && previewSpaceShortcut(c)) {
		w.close(a, now)
	}

	if w.focused && a.Settings.RecordingShortcut == "" {
		root := c.Root()
		if a.previewShortcut(root, "preview_previous") {
			w.move(a, -1)
		}
		if a.previewShortcut(root, "preview_next") {
			w.move(a, 1)
		}
		if a.previewShortcut(root, "preview_zoom_in") {
			p.zoomAt(1.25, p.viewport.W/2, p.viewport.H/2)
		}
		if a.previewShortcut(root, "preview_zoom_out") {
			p.zoomAt(.8, p.viewport.W/2, p.viewport.H/2)
		}
		if a.previewShortcut(root, "preview_fit") {
			w.fitToWindow()
		}
		if a.previewShortcut(root, "preview_actual") {
			w.actualSize()
		}
	}

}
func (w *previewWindow) controls(c *ui.Context, a *App, hovered bool) {
	_, height := c.Size()
	p := &w.picture
	index := slices.IndexFunc(w.items, func(asset model.Asset) bool { return asset.ID == w.asset.ID })
	right := float32(16)
	if w.pinned {
		right = 48
	}
	ui.Row(c.Key("preview-controls")).Absolute().Top(12).Left(16).Right(right).Height(40).Padding(4, 8).Gap(4).AlignItems(ui.Center).Radius(20).Material(panelMaterial(c.Theme())).Border(1, c.Theme().Border).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
			ui.Text(c, w.asset.Name).FontSize(13).Bold().TextColor(titleColor(c.Theme())).SingleLine().Tooltip(w.asset.Name)
			iw, ih := assetDisplaySize(w.asset)
			ui.Textf(c, "%d × %d · %.0f%% · %s", iw, ih, p.Zoom*100, a.T("preview_position", index+1, len(w.items))).FontSize(12).TextColor(c.Theme().TextMuted).SingleLine()
		})
		for _, entry := range []struct {
			name, key string
			delta     int
		}{{"rotate-left", "preview_rotate_left", -90}, {"rotate-right", "preview_rotate_right", 90}} {
			if previewSymbolButton(c.Key(entry.key), entry.name, a.T(entry.key), 32, w.rotating || a.Busy).Material(panelMaterial(c.Theme())).Border(1, c.Theme().Border).Clicked() {
				a.rotatePreviewWindow(w, entry.delta)
			}
		}
		if !w.picture.Held && !w.closedHeld && !w.pinned {
			if previewSymbolButton(c.Key("close-preview"), "close", a.T("close_preview"), 32, false).Material(panelMaterial(c.Theme())).Border(1, c.Theme().Border).Clicked() {
				w.close(a, c.Now())
			}
		}
	})
	if len(w.items) > 1 {
		if previewSymbolButton(c.Key("previous-preview"), "chevron-left", a.T("previous_asset"), 36, index <= 0).Absolute().Left(16).Top(height/2).TranslatePercent(0, -50).Material(panelMaterial(c.Theme())).Border(1, c.Theme().Border).Clicked() {
			w.move(a, -1)
		}
		if previewSymbolButton(c.Key("next-preview"), "chevron-right", a.T("next_asset"), 36, index >= len(w.items)-1).Absolute().Right(16).Top(height/2).TranslatePercent(0, -50).Material(panelMaterial(c.Theme())).Border(1, c.Theme().Border).Clicked() {
			w.move(a, 1)
		}
	}
	if w.pinned {
		close := previewSymbolButton(c.Key("close-pinned-preview"), "close", a.T("close_preview"), 28, false).Absolute().Right(12).Top(12).Material(panelMaterial(c.Theme())).Border(1, c.Theme().Border)
		visible := hovered
		close.Opacity(motionTarget(visible)).IgnoreInput(!visible)
		if close.Clicked() {
			w.close(a, c.Now())
		}
	}
}
