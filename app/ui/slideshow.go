package ui

import (
	"fmt"
	"image"
	"log/slog"
	"path/filepath"
	"slices"
	"time"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// 放映间隔可选的秒数；默认 5 秒，选过的记在偏好里。
var slideshowIntervals = []int{3, 5, 10}

const slideshowControlsHide = 2 * time.Second // 指针不动这么久后收起底部控制条

// slideshow 是全屏放映：黑底，一张一张显示当前列表里的图（视频跳过），到时间自动下一张。
type slideshow struct {
	window  *mygo.Window
	root    string
	items   []model.Asset
	index   int
	paused  bool
	shownAt time.Time // 这一张开始显示的时间，到间隔就翻页
	movedAt time.Time // 指针最后一次移动，控制条据此显示和收起
	pointer ui.Point
	// 解好的图按素材 ID 存，只留当前、上一张和下一张。
	bitmaps map[string]*ui.Bitmap
	failed  map[string]bool
	// 正在后台解的那张；一次只解一张，快速翻页时不会堆起一串大图解码。
	decoding string
}

// slideshowItems 是能放映的素材：视频不放。
func slideshowItems(visible []model.Asset) []model.Asset {
	items := make([]model.Asset, 0, len(visible))
	for _, asset := range visible {
		if !isVideoAsset(asset) {
			items = append(items, asset)
		}
	}
	return items
}

// StartSlideshow 从 startID 那张开始放映当前列表；startID 为空或不在列表里时从第一张开始。
func (a *App) StartSlideshow(startID string) {
	if a.Library == nil || a.IsLocked() || a.Slideshow != nil {
		return
	}
	items := slideshowItems(a.Visible)
	if len(items) == 0 {
		a.Notify(a.T("slideshow_empty"))
		return
	}
	show := &slideshow{root: a.Library.Root, items: items, shownAt: time.Now(), movedAt: time.Now(),
		bitmaps: map[string]*ui.Bitmap{}, failed: map[string]bool{}}
	// 起点是视频（不放映）：从它之后的第一张图开始，后面没有图就从头。
	if i := slices.IndexFunc(a.Visible, func(asset model.Asset) bool { return asset.ID == startID }); i >= 0 && isVideoAsset(a.Visible[i]) {
		startID = ""
		for _, asset := range a.Visible[i+1:] {
			if !isVideoAsset(asset) {
				startID = asset.ID
				break
			}
		}
	}
	for i, asset := range items {
		if asset.ID == startID {
			show.index = i
		}
	}
	a.Slideshow = show
	slog.Info("开始放映", "count", len(items), "start", show.index)
	if a.Window == nil {
		return
	}
	show.window = mygo.NewWindow(mygo.WindowOptions{Title: a.T("slideshow"), Width: 1200, Height: 800, Frameless: true,
		BackgroundColor: "#000000", Content: ui.View(func(c *ui.Context) { show.view(c, a) })})
	show.window.OnClosed(func() {
		if a.Slideshow == show {
			a.Slideshow = nil
		}
	})
	show.window.OnLeaveFullScreen(func() { a.StopSlideshow() })
	show.window.SetFullScreen(true)
	show.window.Focus()
}

// StartSlideshowFromSelection 从选中的第一张开始放映，没选中就从第一张开始。
func (a *App) StartSlideshowFromSelection() {
	start := ""
	for _, asset := range a.Visible {
		if a.Selected.Has(asset.ID) {
			start = asset.ID
			break
		}
	}
	a.StartSlideshow(start)
}

// StopSlideshow 结束放映，回到主窗口。
func (a *App) StopSlideshow() {
	show := a.Slideshow
	if show == nil {
		return
	}
	a.Slideshow = nil
	slog.Info("结束放映")
	if show.window != nil {
		show.window.Close()
	}
}

func (a *App) slideshowInterval() time.Duration {
	seconds := a.Preferences.SlideshowSeconds
	if seconds <= 0 {
		seconds = 5
	}
	return time.Duration(seconds) * time.Second
}

// step 往前（-1）或往后（1）翻一张，到头了接着从另一头开始。
func (s *slideshow) step(delta int, now time.Time) {
	s.index = (s.index + delta + len(s.items)) % len(s.items)
	s.shownAt = now
}

// wanted 是现在要留着的图：当前这张、下一张、上一张，按这个顺序解。
func (s *slideshow) wanted() []model.Asset {
	out := make([]model.Asset, 0, 3)
	for _, delta := range []int{0, 1, -1} {
		out = append(out, s.items[(s.index+delta+len(s.items))%len(s.items)])
	}
	return out
}

// load 丢掉不在前后范围里的图，再挑一张还没解的交给后台；解好后回到界面线程放进缓存，内存不随列表变长。
func (s *slideshow) load(a *App) {
	keep := map[string]bool{}
	var next *model.Asset
	for _, asset := range s.wanted() {
		keep[asset.ID] = true
		if next == nil && s.bitmaps[asset.ID] == nil && !s.failed[asset.ID] {
			next = &asset
		}
	}
	for id := range s.bitmaps {
		if !keep[id] {
			delete(s.bitmaps, id)
		}
	}
	if next == nil || s.decoding != "" {
		return
	}
	asset := *next
	s.decoding = asset.ID
	path := filepath.Join(s.root, filepath.FromSlash(asset.RelativePath))
	decode := func() (image.Image, error) {
		img, err := assets.PreviewImage(path)
		if err == nil {
			img, err = assets.RotateImage(img, asset.Rotation)
		}
		return img, err
	}
	if s.window == nil {
		// 离线界面（测试）没有窗口可回调，直接在这一帧里解。
		img, err := decode()
		s.loaded(a, asset.ID, img, err)
		return
	}
	go func() {
		img, err := decode()
		s.window.Update(func() { s.loaded(a, asset.ID, img, err) })
	}()
}

// loaded 收下后台解好的图；放映已经结束，或者这张已经翻出前后范围，就不建位图。
func (s *slideshow) loaded(a *App, id string, img image.Image, err error) {
	if s.decoding == id {
		s.decoding = ""
	}
	if err != nil {
		slog.Warn("放映解图失败", "id", id, "error", err)
		s.failed[id] = true
		return
	}
	if a.Slideshow != s {
		return
	}
	for _, asset := range s.wanted() {
		if asset.ID == id {
			s.bitmaps[id] = previewBitmap(img, false)
			return
		}
	}
}

func (s *slideshow) view(c *ui.Context, a *App) {
	now := c.Now()
	if !s.paused && now.Sub(s.shownAt) >= a.slideshowInterval() {
		s.step(1, now)
	}
	s.load(a)
	s.keys(c, a, now)
	if a.Slideshow != s {
		return
	}
	asset := s.items[s.index]
	root := c.Root().Background(ui.RGBA(0, 0, 0, 1))
	x, y, over := root.PointerPosition()
	if over && (x != s.pointer.X || y != s.pointer.Y) {
		s.pointer, s.movedAt = ui.Point{X: x, Y: y}, now
	}
	controls := now.Sub(s.movedAt) < slideshowControlsHide
	root.Cursor(ui.CursorDefault)
	if !controls {
		root.Cursor(ui.CursorNone)
	}
	root.Children(func() {
		// 换图时新图淡入、旧图淡出。
		ui.Box(c.Key(asset.ID)).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Transition(ui.ElementTransition{Duration: 450 * time.Millisecond, Enter: &ui.Motion{Opacity: 0}, Exit: &ui.Motion{Opacity: 0}}).
			Children(func() {
				switch bitmap := s.bitmaps[asset.ID]; {
				case bitmap != nil:
					ui.Image(c, bitmap).Fill().Fit(ui.Contain)
				case s.failed[asset.ID]:
					ui.Text(c, a.T("slideshow_unreadable", asset.Name)).Absolute().Left(0).Right(0).Top(0).Bottom(0).TextAlign(ui.Center).
						FontSize(fontBody).TextColor(ui.RGBA(255, 255, 255, .6))
				default:
					// 还没解好时先放缩略图，不黑屏。
					if thumb := a.Bitmap(asset); thumb != nil {
						ui.Image(c, thumb).Fill().Fit(ui.Contain)
					}
				}
			})
		if controls {
			s.controls(c, a, now)
		}
	})
	// 到点翻页和收起控制条都要一帧来推动。
	next := a.slideshowInterval() - now.Sub(s.shownAt)
	if controls {
		next = min(next, slideshowControlsHide-now.Sub(s.movedAt))
	}
	if !s.paused || controls {
		c.After(max(16*time.Millisecond, next))
	}
}

// controls 是底部居中的控制条：几分之几、上一张、暂停/继续、下一张、间隔、退出。
func (s *slideshow) controls(c *ui.Context, a *App, now time.Time) {
	white := ui.RGBA(255, 255, 255, .92)
	button := func(key, label, icon string) bool {
		b := ui.ButtonBase(c.Key(key)).Size(32, 32).Radius(radiusControl).Center().Cursor(ui.CursorPointer).Label(label).Tooltip(label)
		if b.Hovered() {
			b.Background(ui.RGBA(255, 255, 255, .14))
		}
		b.Children(func() { appIcon(c, icon).Size(iconMedium, iconMedium).TextColor(white) })
		return b.Clicked()
	}
	ui.Row(c.Key("slideshow-controls")).Absolute().Bottom(32).Left(0).Right(0).Justify(ui.Center).Children(func() {
		ui.Row(c).Height(44).Padding(0, space12).Gap(space4).Radius(22).AlignItems(ui.Center).Background(ui.RGBA(30, 30, 32, .82)).Children(func() {
			ui.Text(c, fmt.Sprintf("%d / %d", s.index+1, len(s.items))).FontFeatures("tnum").FontSize(fontSecondary).TextColor(ui.RGBA(255, 255, 255, .7)).Padding(0, space8)
			if button("previous", a.T("previous_asset"), "caret-left") {
				s.step(-1, now)
			}
			icon, label := "pause", a.T("slideshow_pause")
			if s.paused {
				icon, label = "play", a.T("slideshow_play")
			}
			if button("play", label, icon) {
				s.togglePause(now)
			}
			if button("next", a.T("next_asset"), "caret-right") {
				s.step(1, now)
			}
			ui.Box(c).Size(1, 20).Background(ui.RGBA(255, 255, 255, .18)).Margin(0, space4)
			for _, seconds := range slideshowIntervals {
				label := a.T("slideshow_seconds", seconds)
				chip := ui.ButtonBase(c.Key(fmt.Sprint("interval-", seconds))).Height(28).Padding(0, space8).Radius(14).Center().Cursor(ui.CursorPointer).Label(label)
				on := a.slideshowInterval() == time.Duration(seconds)*time.Second
				switch {
				case on:
					chip.Background(ui.RGBA(255, 255, 255, .22))
				case chip.Hovered():
					chip.Background(ui.RGBA(255, 255, 255, .1))
				}
				chip.Children(func() { ui.Text(c, label).FontFeatures("tnum").FontSize(fontSecondary).TextColor(white) })
				if chip.Clicked() && !on {
					a.Preferences.SlideshowSeconds = seconds
					a.queuePreferences()
				}
			}
			ui.Box(c).Size(1, 20).Background(ui.RGBA(255, 255, 255, .18)).Margin(0, space4)
			if button("exit", a.T("slideshow_exit"), "x") {
				a.StopSlideshow()
			}
		})
	})
}

func (s *slideshow) togglePause(now time.Time) {
	s.paused = !s.paused
	// 继续时这一张重新计时，不会一点继续就马上翻走。
	s.shownAt = now
}

// keys：空格暂停或继续，左右翻页，Esc 退出。
func (s *slideshow) keys(c *ui.Context, a *App, now time.Time) {
	switch {
	case c.Shortcut(0, ui.KeyEscape):
		a.StopSlideshow()
	case c.Shortcut(0, ui.KeySpace):
		// 按住空格时的重复按键不算，不然松手时停没停要看重复了几次。
		if !c.ShortcutRepeated() {
			s.togglePause(now)
		}
	case c.Shortcut(0, ui.KeyLeft):
		s.step(-1, now)
	case c.Shortcut(0, ui.KeyRight):
		s.step(1, now)
	default:
		return
	}
	s.movedAt = now
}
