package ui

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const previewPeekThreshold = 250 * time.Millisecond
const previewPinThreshold = 3 * time.Second
const previewWindowDuration = 420 * time.Millisecond // 慢到能看清窗口展开的过程

// 预览是一个和侧栏同色的圆角窗口：上面是图片，下面一行靠左的名称和尺寸，右边关闭。
// 留白照旧版 macOS（git 2a13147^ MomentoAssetPreviewOverlay.swift）四周 14；图片最多占屏幕可用区域 85% 宽、85% 高，不再受主窗口大小限制。
const (
	previewPhotoRadius   = 6                                      // 图片圆角
	previewImageInset    = 14                                     // 图片到窗口左、上、右边的留白，底下接底栏
	previewCardRadius    = previewPhotoRadius + previewImageInset // 窗口圆角，和图片圆角同心
	previewWidthPercent  = 85
	previewHeightPercent = 85
	previewFooterMinimum = 52  // 底栏最少两行：上下各 10 + 名称 + 3 + 尺寸
	previewFooterPadding = 10  // 底栏上下内边距
	previewFooterLineGap = 5   // 文件名和下面一行的间距
	previewFooterInset   = 20  // 底栏左右内边距，和窗口圆角一样，文字比图片边再往里收一点
	previewInfoIcon      = 10  // 尺寸、大小、相机前的小图标，和 10 号字一样高
	previewInfoGap       = 3   // 小图标和文字的间距
	previewCloseIcon     = 18  // 关闭按钮的实心小圆
	previewMinimumWidth  = 240 // 太窄时名称没地方放
	previewMinimumHeight = 160
)

type previewWindows struct {
	active       *previewWindow
	pinned       []*previewWindow
	fading       []*previewWindow // 还在播关闭动画的窗口，不再挡着打开新预览
	gesture      *previewWindow
	spaceDown    time.Time
	inputPending bool
}

type previewWindow struct {
	window                    *mygo.Window
	picture                   previewState
	root                      string
	asset                     model.Asset
	items                     []model.Asset
	placeholder               *ui.Bitmap
	lastSelectionID           string
	mainSelection             []string
	pinned, closing, disposed bool
	hasFocus                  bool
	// 只在预览里看的水平翻转和灰度，不改原图，翻页时保留，关掉预览就没了。
	flipped, gray                           bool
	pressedAt, openedAt, closedAt, pinnedAt time.Time
	frame                                   mygo.Rectangle
	closeScale, closeOpacity                float32
	closedHeld                              bool
	closedProgress                          float32
	closedRingOpacity                       float32
	hostAnimating, reduceMotion             bool
	footerFor                               previewFooterKey
	footerHeight                            float32
	decoding                                bool // 后台正在解原图
	video                                   previewVideo
	live                                    previewLive
}

type previewFooterKey struct {
	id, locale string
	width      float32
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
	if w := a.Preview.active; w != nil {
		if !w.closing {
			return
		}
		// 上一个预览还在播关闭动画：让它自己播完，新的直接打开，不用等 420ms。
		a.Preview.fading = append(a.Preview.fading, w)
		a.Preview.active = nil
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
	if w.videoShortcut(c.Shortcut, c.Now()) {
		w.invalidate()
		a.markPreviewInput()
		return
	}
	for _, action := range []string{"preview_zoom_in", "preview_zoom_out", "preview_fit", "preview_actual", "preview_flip", "preview_grayscale"} {
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
		case "preview_flip":
			w.flipped = !w.flipped
		case "preview_grayscale":
			w.gray = !w.gray
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
	windows := append(slices.Clone(a.Preview.pinned), a.Preview.fading...)
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

func (w *previewWindow) focused() bool { return w.hasFocus }

// 用 ShowInactive 显示，不抢主窗口键盘，主窗口才能判断空格是否还按着。
func (w *previewWindow) create(a *App) {
	parent := a.Window.Bounds()
	area := mygo.Screen.DisplayMatching(parent).WorkArea
	w.frame = previewLayout(parent, area, w.asset, a.Preferences.PreviewFit == "fit", a.Texts.Locale(), preferenceZoom(a.Preferences.UIZoom))
	w.hostAnimating = true
	w.window = mygo.NewWindow(mygo.WindowOptions{Title: w.asset.Name, Width: w.frame.Width, Height: w.frame.Height,
		Hidden: true, Frameless: true, Transparent: true, CornerRadius: previewCardRadius, SkipTaskbar: true,
		DisableResize: true, DisableMaximize: true, DisableMinimize: true, DisableFullScreen: true,
		BackgroundColor: "rgba(0,0,0,0)", Content: ui.View(func(c *ui.Context) { w.view(c, a) })})
	w.window.SetMinimumSize(1, 1)
	w.applyFrames(.92, 0)
	w.window.OnClose(func(e *mygo.CloseEvent) { e.PreventDefault(); w.close(a, time.Now()) })
	w.window.OnClosed(func() { w.dispose(); a.removePreviewWindow(w) })
	w.window.OnFocus(func() { w.hasFocus = true; a.previewMenuChanged() })
	w.window.OnBlur(func() { w.hasFocus = false; a.previewMenuChanged() })
	keep := func() {
		if w.pinned && !w.closing && !w.hostAnimating {
			w.frame = w.window.Bounds()
		}
	}
	w.window.OnMove(keep)
	w.window.OnResize(keep)
	w.window.ShowInactive()
}

// 图片按比例放进屏幕可用区域 85% 宽、85% 高的范围（拿不到屏幕时按主窗口），四周加留白、下面接底栏；
// 整个窗口在主窗口中央，不超出屏幕。底栏高度随文件名换行变化，而文件名怎么换行又取决于窗口宽度，所以算两遍。
func previewLayout(parent, area mygo.Rectangle, asset model.Asset, enlarge bool, locale string, zoom float32) mygo.Rectangle {
	// 拿不到屏幕时按主窗口算，并保留最小尺寸；拿得到屏幕时严格按比例，小屏幕也不超出 85%。
	maxW, maxH := max(360, parent.Width*previewWidthPercent/100), max(320, parent.Height*previewHeightPercent/100)
	if area.Width > 0 && area.Height > 0 {
		maxW, maxH = area.Width*previewWidthPercent/100, area.Height*previewHeightPercent/100
	}
	if area.Width > 0 {
		maxW = min(maxW, area.Width-2*previewImageInset)
	}
	iw, ih := assetDisplaySize(asset)
	limit := float32(1)
	if enlarge {
		limit = 2
	}
	footer, width, imageHeight := int(math.Ceil(previewFooterMinimum*float64(zoom))), 0, 0
	for range 2 {
		limitH := maxH
		if area.Height > 0 {
			limitH = min(maxH, area.Height-previewImageInset-footer)
		}
		scale := min(limit, previewFitZoom(float32(iw), float32(ih), float32(maxW), float32(limitH)))
		width = max(previewMinimumWidth, min(maxW, int(float32(iw)*scale))+2*previewImageInset)
		imageHeight = min(limitH, max(previewMinimumHeight, int(float32(ih)*scale)))
		footer = previewFooterPoints(asset, width, locale, zoom)
	}
	height := imageHeight + previewImageInset + footer
	x, y := parent.X+(parent.Width-width)/2, parent.Y+(parent.Height-height)/2
	if area.Width > 0 {
		x = max(area.X, min(x, area.X+area.Width-width))
	}
	if area.Height > 0 {
		y = max(area.Y, min(y, area.Y+area.Height-height))
	}
	return mygo.Rectangle{X: x, Y: y, Width: width, Height: height}
}

// 以窗口中心缩放窗口矩形，做打开、关闭和固定时的弹动。
func scaledAround(r mygo.Rectangle, scale float32) mygo.Rectangle {
	cx, cy := float32(r.X)+float32(r.Width)/2, float32(r.Y)+float32(r.Height)/2
	x := cx + (float32(r.X)-cx)*scale
	y := cy + (float32(r.Y)-cy)*scale
	return mygo.Rectangle{X: int(math.Round(float64(x))), Y: int(math.Round(float64(y))),
		Width: int(math.Round(float64(float32(r.Width) * scale))), Height: int(math.Round(float64(float32(r.Height) * scale)))}
}

// 把当前动画帧的缩放和透明度同步到窗口。
func (w *previewWindow) applyFrames(scale, opacity float32) {
	if w.window == nil {
		return
	}
	w.window.SetBounds(scaledAround(w.frame, scale))
	w.window.SetOpacity(float64(opacity))
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
	w.closedRingOpacity = min(1, max(0, float32(now.Sub(w.pressedAt)-previewPeekThreshold)/float32(120*time.Millisecond)))
	w.closing, w.closedAt = true, now
	w.picture.Held = false
	w.stopVideo()
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
	if w.window != nil {
		w.window.Hide()
		w.window.Destroy()
	}
}
func (w *previewWindow) dispose() {
	w.closeVideo()
	w.closeLive()
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
	a.Preview.fading = slices.DeleteFunc(a.Preview.fading, func(item *previewWindow) bool { return item == w })
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
}
func (w *previewWindow) setAsset(a *App, asset model.Asset) {
	// 换图时文件名行数可能变，窗口跟着增减底栏的高度，图片区域保持原来大小。
	if w.window != nil && w.asset.ID != "" && w.asset.ID != asset.ID {
		locale, zoom := a.Texts.Locale(), preferenceZoom(a.Preferences.UIZoom)
		if delta := previewFooterPoints(asset, w.frame.Width, locale, zoom) - previewFooterPoints(w.asset, w.frame.Width, locale, zoom); delta != 0 {
			w.frame = previewResized(w.frame, delta, mygo.Screen.DisplayMatching(w.frame).WorkArea)
			if !w.hostAnimating {
				w.window.SetBounds(w.frame)
			}
		}
	}
	// 换了素材就关掉旧播放器，不在后台接着出声。
	if asset.ID != w.asset.ID {
		w.closeVideo()
		w.closeLive()
	}
	w.asset = asset
	w.picture.target(w.root, asset.ID, asset.Rotation)
	w.placeholder = nil
	if a.Thumbnails != nil {
		w.placeholder = a.Thumbnails.Get(w.root, asset)
	}
	if w.window != nil {
		w.window.SetTitle(asset.Name)
	}
	w.invalidate()
}
func (w *previewWindow) move(a *App, delta int) {
	w.items = a.previewItems(w)
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

// 翻页时按打开那一刻的顺序走（最近使用排序会随翻页重排，不能跟着变），但去掉之后删掉或被筛掉的图，并换成最新的素材数据。
// 当前图还在主窗口列表里时，只留仍在列表里的；当前图已不在列表里（换了范围），只去掉已不存在或进出回收站的。
func (a *App) previewItems(w *previewWindow) []model.Asset {
	if a.Library == nil || a.Library.Root != w.root {
		return w.items
	}
	visible := make(map[string]model.Asset, len(a.Visible))
	for _, asset := range a.Visible {
		visible[asset.ID] = asset
	}
	_, inView := visible[w.asset.ID]
	items := make([]model.Asset, 0, len(w.items))
	for _, item := range w.items {
		if item.ID == w.asset.ID {
			items = append(items, w.asset)
			continue
		}
		if inView {
			if current, ok := visible[item.ID]; ok {
				items = append(items, current)
			}
		} else if current, ok := a.previewAsset(item.ID); ok && current.Trashed == item.Trashed {
			items = append(items, current)
		}
	}
	return items
}

// 预览窗口的底：和侧栏同色，窗口本身透明只为露出圆角。
func previewCard(c *ui.Context, key any, radius float32) ui.Element {
	return ui.Box(c.Key(key)).Fill().Radius(radius).Clip().Background(paletteFor(c.Theme()).Sidebar)
}

// 预览窗口：上面图片，下面底栏。动画帧也由这个窗口驱动。
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
		w.applyFrames(scale, opacity)
		w.hostAnimating = animating
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
	}
	if w.placeholder == nil && a.Thumbnails != nil {
		w.placeholder = a.Thumbnails.Get(w.root, w.asset)
	}
	p := &w.picture
	// 动图下一帧本来就要重新生成位图，直接改用新设置；后台正在解帧时不动，等它解完。
	if p.animation != nil && !p.animationLoading && p.animation.pixelated != a.Preferences.PreviewPixelated {
		p.animation.pixelated = a.Preferences.PreviewPixelated
	}
	if w.window != nil && w.needsPicture(a) {
		w.load(a)
	}
	w.syncVideo(a, now)
	w.animatePicture(c, a)
	surface := previewCard(c, "preview-window-surface", previewCardRadius)
	if w.pinned {
		surface.DragWindow()
	}
	surface.Children(func() {
		w.viewport(c, a)
		w.footer(c, a, now)
	})
	if w.hasFocus {
		w.handleKeys(c, a, now)
	}
}

// needsPicture 判断要不要在后台解原图：没图时去解；有图但像素方块设置改了，也重新生成，生成好之前先显示旧的。
// 读失败过就不再重试，免得一直读盘。视频由播放器显示，不走图片解码。
func (w *previewWindow) needsPicture(a *App) bool {
	p := &w.picture
	if isVideoAsset(w.asset) || w.decoding {
		return false
	}
	stale := p.Bitmap != nil && p.animation == nil && p.errorText == "" && p.pixelated != a.Preferences.PreviewPixelated
	return p.Bitmap == nil && p.errorText == "" && !p.Loading || stale
}

// 底栏：完整文件名和尺寸靠左两行，右边一个不抢眼的关闭按钮；按住空格时关闭按钮的位置换成小圆环进度。
func (w *previewWindow) footer(c *ui.Context, a *App, now time.Time) {
	pal := paletteFor(c.Theme())
	elapsed := now.Sub(w.pressedAt)
	pinFade := w.pinned && now.Sub(w.pinnedAt) < 120*time.Millisecond
	holding := w.picture.Held && elapsed >= previewPeekThreshold && !w.pinned || w.closing && w.closedHeld && elapsed >= previewPeekThreshold || pinFade
	width, _ := c.Size()
	dimensions, size := previewDetails(w.asset)
	row := ui.Row(c.Key("preview-footer")).Absolute().Left(0).Right(0).Bottom(0).Height(w.footerHeightFor(width, a.Texts.Locale())).
		Padding(0, previewFooterInset).Gap(space12).AlignItems(ui.Center)
	row.Children(func() {
		ui.Column(c.Key("preview-info-text")).Grow(1).Shrink(1).MinWidth(0).Gap(previewFooterLineGap).Children(func() {
			ui.Text(c.Key("preview-name"), previewFileName(w.asset)).FontSize(fontLead).FontWeight(500).TextColor(pal.Text).MaxWidthPercent(100).Selectable()
			if w.picture.errorText != "" && w.picture.Bitmap != nil {
				ui.Text(c, w.picture.errorText).FontSize(fontCaption).TextColor(pal.Danger).SingleLine().MaxWidthPercent(100)
			} else {
				ui.Row(c.Key("preview-meta")).Gap(space12).AlignItems(ui.Center).MaxWidthPercent(100).Children(func() {
					if dimensions != "" {
						previewInfo(c, "maximize-square", dimensions)
					}
					if size != "" {
						previewInfo(c, "diskette", size)
					}
					if isVideoAsset(w.asset) && w.asset.DurationMS > 0 {
						previewInfo(c, "play", videoDuration(assetDuration(w.asset)))
					}

				})
			}
		})
		if holding {
			value := w.holdProgress(now)
			opacity := min(1, max(0, float32(elapsed-previewPeekThreshold)/float32(120*time.Millisecond)))
			if w.closing {
				value, opacity = w.closedProgress, w.closedRingOpacity
			}
			if pinFade {
				value, opacity = 1, 1-float32(now.Sub(w.pinnedAt))/float32(120*time.Millisecond)
			}
			// 占关闭按钮的位置，圆环居中，名称不会因为按住而换位。
			ui.Box(c.Key("preview-hold-progress")).Size(controlSmall, controlSmall).Shrink(0).Justify(ui.Center).AlignItems(ui.Center).
				IgnoreInput(true).Opacity(opacity).Children(func() { previewHoldRing(c, value, pal.Border, pal.Accent) })
		} else if !w.picture.Held && !w.closedHeld {
			if previewCloseButton(c.Key("close-preview"), a.T("close_preview")).Clicked() {
				w.close(a, c.Now())
			}
		}
	})
}

// 按住空格固定时的小圆环：浅色底圈，品牌橘从 12 点顺时针走满。
func previewHoldRing(c *ui.Context, value float32, track, fill ui.Color) {
	const stroke = 2
	ui.Box(c.Key("preview-hold-ring")).Size(iconSmall, iconSmall).Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy, radius := r.X+r.W/2, r.Y+r.H/2, (min(r.W, r.H)-stroke)/2
		arc := func(end float32) [][2]float32 {
			steps := max(2, int(end*48))
			points := make([][2]float32, 0, steps+1)
			for i := 0; i <= steps; i++ {
				angle := float64(end)*2*math.Pi*float64(i)/float64(steps) - math.Pi/2
				points = append(points, [2]float32{cx + radius*float32(math.Cos(angle)), cy + radius*float32(math.Sin(angle))})
			}
			return points
		}
		p.StrokePolyline(arc(1), stroke, track)
		if value > 0 {
			p.StrokePolyline(arc(min(1, value)), stroke, fill)
		}
	})
}

// 完整文件名：素材名加扩展名（导入时名称去掉了最后一个扩展名）。
func previewFileName(asset model.Asset) string {
	if extension := strings.ToLower(strings.TrimPrefix(asset.Extension, ".")); extension != "" {
		return asset.Name + "." + extension
	}
	return asset.Name
}

// 底栏第二行：尺寸和文件大小，各带一个小图标，悬停也不变。
func previewDetails(asset model.Asset) (dimensions, size string) {
	if width, height := assetDisplaySize(asset); width > 0 && height > 0 {
		dimensions = fmt.Sprintf("%d × %d", width, height)
	}
	if asset.Size > 0 {
		size = fileSize(asset.Size)
	}
	return dimensions, size
}

// 量出底栏需要的高度：文件名在扣掉关闭按钮后的宽度里换行，第二行固定一行，至少保持两行高。
func previewFooterHeight(asset model.Asset, width float32, locale string) float32 {
	family := uiFont(locale)
	_, name := ui.ShapeText(previewFileName(asset), ui.Font{Family: family, Size: fontLead, Weight: 500, Lang: locale}, max(1, width-2*previewFooterInset-controlSmall-space12)).Size()
	_, meta := ui.ShapeText("0", ui.Font{Family: family, Size: fontCaption, Lang: locale}, 0).Size()
	return max(previewFooterMinimum, 2*previewFooterPadding+name+previewFooterLineGap+max(meta, previewInfoIcon))
}

// 底栏在窗口里占多少点：界面缩放后内容宽度变窄、文字变大，先按缩放后的宽度排字，再换回窗口的点。
func previewFooterPoints(asset model.Asset, windowWidth int, locale string, zoom float32) int {
	return int(math.Ceil(float64(previewFooterHeight(asset, float32(windowWidth)/zoom, locale) * zoom)))
}

// 底栏变高时窗口往下长；碰到工作区底边就整体上移，不超出屏幕。
func previewResized(r mygo.Rectangle, delta int, area mygo.Rectangle) mygo.Rectangle {
	r.Height += delta
	if area.Height > 0 && r.Y+r.Height > area.Y+area.Height {
		r.Y = max(area.Y, area.Y+area.Height-r.Height)
	}
	return r
}

// 每帧都要用底栏高度，按素材、宽度和语言缓存，不每帧重新排字。
func (w *previewWindow) footerHeightFor(width float32, locale string) float32 {
	key := previewFooterKey{w.asset.ID, locale, width}
	if w.footerFor != key || w.footerHeight == 0 {
		w.footerFor, w.footerHeight = key, previewFooterHeight(w.asset, width, locale)
	}
	return w.footerHeight
}

// 第二行比三级文字再浅一档，只做辅助信息。
func previewCaptionColor(c *ui.Context) ui.Color { return paletteFor(c.Theme()).TextTertiary.Alpha(.8) }

// 底栏信息：一个小图标加一段文字。
func previewInfo(c *ui.Context, icon, text string) ui.Element {
	color := previewCaptionColor(c)
	row := ui.Row(c).Gap(previewInfoGap).AlignItems(ui.Center)
	row.Children(func() {
		appIcon(c, icon).Size(previewInfoIcon, previewInfoIcon).Shrink(0).TextColor(color)
		ui.Text(c, text).FontSize(fontCaption).FontFeatures("tnum").TextColor(color).SingleLine().Shrink(1).MinWidth(0)
	})
	return row
}

// 底栏的关闭按钮：一颗浅灰实心小圆、叉是镂空的，悬停和按下逐级加深；点击区域仍是 28。
func previewCloseButton(c *ui.Context, label string) ui.Element {
	p := paletteFor(c.Theme())
	button := ui.ButtonBase(c).Size(controlSmall, controlSmall).Padding(0).Radius(controlSmall / 2).
		Label(label).Tooltip(label).Cursor(ui.CursorPointer).Justify(ui.Center).AlignItems(ui.Center)
	color := p.TextTertiary.Alpha(.4)
	if button.Pressed() {
		color = p.TextTertiary.Alpha(.85)
	} else if button.Hovered() {
		color = p.TextTertiary.Alpha(.6)
	}
	animateColors(button, ui.Transparent, color)
	button.Children(func() { appIcon(c, "close-circle").Size(previewCloseIcon, previewCloseIcon) })
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
	case w.videoShortcut(c.Shortcut, now):
	case a.previewShortcut(root, "preview_zoom_in"):
		p.zoomAt(1.25, p.viewport.W/2, p.viewport.H/2)
	case a.previewShortcut(root, "preview_zoom_out"):
		p.zoomAt(.8, p.viewport.W/2, p.viewport.H/2)
	case a.previewShortcut(root, "preview_fit"):
		w.fitToWindow()
	case a.previewShortcut(root, "preview_actual"):
		w.actualSize()
	case a.previewShortcut(root, "preview_flip"):
		w.flipped = !w.flipped
	case a.previewShortcut(root, "preview_grayscale"):
		w.gray = !w.gray
	default:
		handled = false
	}
	// 翻页和缩放后重画。
	if handled {
		w.invalidate()
	}
}
