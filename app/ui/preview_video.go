package ui

import (
	"log/slog"
	"path/filepath"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/video"
)

// openPreviewVideo 打开预览窗口用的播放器，测试里换成假的。
var openPreviewVideo = video.Open

// openPreviewVideoExternally 用系统默认应用打开播放不了的视频，测试里换成假的。
var openPreviewVideoExternally = mygo.Shell.OpenPath

const (
	previewVideoHideAfter = 2 * time.Second  // 播放中鼠标停这么久，控制条淡出
	previewVideoSeekStep  = 5 * time.Second  // Shift+←/→ 一次跳多少
	previewVideoFrameStep = time.Second / 30 // [ ] 逐帧：播放器给不出帧率，按 30 帧算
	previewVideoBarInset  = space8           // 控制条离视频边缘
	previewVideoBarMin    = 320              // 竖视频太窄时控制条至少这么宽，不挤掉进度条
	previewVideoTrack     = 4                // 进度条轨道粗细
	previewVideoThumb     = 12               // 进度条圆点直径
)

// previewVideo 是预览窗口里当前视频的播放器和控制条状态。一个窗口同时只有一个播放器，换素材、关窗口时都要关掉。
type previewVideo struct {
	player    *ui.VideoPlayer
	id        string // 播放器（或打开失败）对应的素材
	failed    error  // 打不开或播放出错，显示封面和「用默认应用打开」
	closed    bool   // 窗口在播关闭动画：播放器已关、声音已停，只留最后一帧
	muted     bool   // 这个窗口里点过静音，翻到下一个视频也保持
	pointer   ui.Point
	movedAt   time.Time // 鼠标上次在视频上移动、或按过视频快捷键的时间
	scrubbing bool      // 正按着进度条拖
	resume    bool      // 拖之前在播放，松手接着播
	scrub     float64   // 进度条的值（秒）
}

// active 是还能操作的播放器：没打开、出错或窗口正在关时为 nil。
func (v *previewVideo) active() *ui.VideoPlayer {
	if v.player == nil || v.closed || v.failed != nil {
		return nil
	}
	return v.player
}

// syncVideo 让播放器跟着当前素材：是视频就打开（只开一次），不是视频就关掉。只在主线程的 view 里调。
func (w *previewWindow) syncVideo(a *App, now time.Time) {
	v := &w.video
	if w.closing || w.disposed {
		return
	}
	if !isVideoAsset(w.asset) {
		w.closeVideo()
		return
	}
	if v.id == w.asset.ID {
		if player := v.active(); player != nil {
			if err := player.Err(); err != nil {
				slog.Error("预览视频播放出错", "asset", w.asset.ID, "error", err)
				player.Close()
				v.player, v.failed = nil, err
			}
		}
		return
	}
	w.closeVideo()
	v.id, v.movedAt = w.asset.ID, now
	player, err := openPreviewVideo(filepath.Join(w.root, filepath.FromSlash(w.asset.RelativePath)))
	if err != nil {
		slog.Error("预览视频打不开", "asset", w.asset.ID, "error", err)
		v.failed = err
		return
	}
	// 和系统快速查看一样带声音播一遍，不循环，播完停在最后一帧。
	v.player = player.SetLoop(false).SetMuted(v.muted)
}

// closeVideo 关掉播放器并清空状态（保留窗口里的静音选择）。换素材、窗口销毁时调。
func (w *previewWindow) closeVideo() {
	v := &w.video
	if v.player != nil && !v.closed {
		v.player.Close()
	}
	*v = previewVideo{muted: v.muted}
}

// stopVideo 在窗口开始播关闭动画时停掉声音，画面停在当前帧陪窗口淡出。
func (w *previewWindow) stopVideo() {
	w.closeLive()
	v := &w.video
	if v.player != nil && !v.closed {
		v.player.Close()
		v.closed = true
	}
}

func (v *previewVideo) toggle() {
	if player := v.active(); player != nil {
		if player.Playing() {
			player.Pause()
		} else {
			// 播完后再点从头播。
			player.Play()
		}
	}
}

func (v *previewVideo) seekBy(delta time.Duration) {
	player := v.active()
	if player == nil {
		return
	}
	at := max(0, player.Position()+delta)
	if d := player.Duration(); d > 0 {
		at = min(at, d)
	}
	player.Seek(at)
}

func (v *previewVideo) toggleMute() {
	if player := v.active(); player != nil {
		v.muted = !player.Muted()
		player.SetMuted(v.muted)
	}
}

// videoShortcut 处理只在视频上生效的按键：K 播放暂停、Shift+←/→ 跳 5 秒、[ ] 逐帧、M 静音。
// 当前不是视频、播放器不能用或没按这些键时返回 false，图片预览的按键不受影响。
func (w *previewWindow) videoShortcut(shortcut func(ui.Modifiers, ui.Key) bool, now time.Time) bool {
	v := &w.video
	if !isVideoAsset(w.asset) || v.active() == nil {
		return false
	}
	switch {
	case shortcut(0, ui.KeyK):
		v.toggle()
	case shortcut(ui.Shift, ui.KeyLeft):
		v.seekBy(-previewVideoSeekStep)
	case shortcut(ui.Shift, ui.KeyRight):
		v.seekBy(previewVideoSeekStep)
	case shortcut(0, ui.KeyBracketLeft):
		v.player.Pause()
		v.seekBy(-previewVideoFrameStep)
	case shortcut(0, ui.KeyBracketRight):
		v.player.Pause()
		v.seekBy(previewVideoFrameStep)
	case shortcut(0, ui.KeyM):
		v.toggleMute()
	default:
		return false
	}
	// 按了键就把控制条叫出来，看得到进度和状态。
	v.movedAt = now
	return true
}

// videoViewport 画视频：按窗口适应居中，不缩放不平移；播放器还没出画面前先显示封面，播放不了时封面上给出提示和按钮。
func (w *previewWindow) videoViewport(c *ui.Context, a *App, viewport ui.Element) {
	p, v, asset := &w.picture, &w.video, w.asset
	now := c.Now()
	player := v.active()
	shown := v.player
	if v.failed != nil {
		shown = nil
	}
	bw, bh := assetDisplaySize(asset)
	if (bw <= 0 || bh <= 0) && shown != nil {
		bw, bh = shown.Size()
	}
	if (bw <= 0 || bh <= 0) && w.placeholder != nil {
		bw, bh = w.placeholder.Size()
	}
	rect := ui.Rect{W: p.viewport.W, H: p.viewport.H}
	if bw > 0 && bh > 0 {
		zoom := previewFitZoom(float32(bw), float32(bh), p.viewport.W, p.viewport.H)
		if w.pinned || a.Preferences.PreviewFit == "fit" {
			zoom = min(2, zoom)
		} else {
			zoom = min(1, zoom)
		}
		rect.W, rect.H = float32(bw)*zoom, float32(bh)*zoom
		rect.X, rect.Y = (p.viewport.W-rect.W)/2, (p.viewport.H-rect.H)/2
	}
	// 鼠标在视频上动一下就记下来，控制条据此显示或淡出。
	if x, y, over := viewport.PointerPosition(); over {
		if at := (ui.Point{X: x, Y: y}); at != v.pointer {
			v.pointer, v.movedAt = at, now
		}
	}
	viewport.Children(func() {
		frame := ui.Box(c.Key(struct{ Video string }{asset.ID})).Absolute().Left(rect.X).Top(rect.Y).Size(rect.W, rect.H).
			Radius(previewPhotoRadius).Clip().Background(paletteFor(c.Theme()).Subtle)
		w.slideTransition(c, frame)
		if player != nil {
			frame.Cursor(ui.CursorPointer)
			if frame.Clicked() {
				v.toggle()
				v.movedAt = now
			}
		}
		frame.Children(func() {
			if shown != nil {
				ui.Video(c.Key("preview-video"), shown).Fill().Fit(ui.Contain).Label(asset.Name)
			}
			if w.placeholder != nil && (shown == nil || !shown.Ready()) {
				ui.Image(c.Key("preview-video-poster"), w.placeholder).Absolute().Top(0).Left(0).Right(0).Bottom(0).Fit(ui.Contain)
			}
		})
		switch {
		case v.failed != nil:
			w.videoUnplayable(c, a, rect)
		case player != nil:
			w.videoControls(c, a, player, rect, now)
		}
	})
}

// 播放不了：封面上居中一张小卡片，一行说明加一个「用默认应用打开」。
func (w *previewWindow) videoUnplayable(c *ui.Context, a *App, rect ui.Rect) {
	pal := paletteFor(c.Theme())
	box := ui.Box(c.Key("preview-video-unplayable")).Absolute().Left(rect.X).Top(rect.Y).Size(rect.W, rect.H).Center()
	box.Children(func() {
		card := ui.Column(c.Key("preview-video-unplayable-card")).MaxWidth(max(0, rect.W-2*space16)).Padding(space16).Gap(space12).
			AlignItems(ui.Center).Radius(radiusPanel).Background(pal.Sidebar.Alpha(.92))
		elevated(card, c.Theme()).Children(func() {
			ui.Text(c, a.T("video_unplayable")).FontSize(fontBody).FontWeight(500).TextColor(pal.Text).TextAlign(ui.Center).MaxWidthPercent(100)
			if dialogButton(c.Key("preview-video-open-external"), a.T("open_with_default_app"), false).Clicked() {
				path := filepath.Join(w.root, filepath.FromSlash(w.asset.RelativePath))
				if err := openPreviewVideoExternally(path); err != nil {
					slog.Error("用默认应用打开视频失败", "asset", w.asset.ID, "error", err)
					a.ShowError(err)
				}
			}
		})
	})
}

// 视频下沿的控制条：播放/暂停、当前时间 / 总时长、进度条、静音。播放中鼠标停 2 秒淡出，动一下或暂停时出现。
func (w *previewWindow) videoControls(c *ui.Context, a *App, player *ui.VideoPlayer, rect ui.Rect, now time.Time) {
	v, pal := &w.video, paletteFor(c.Theme())
	duration := player.Duration()
	if duration <= 0 {
		duration = assetDuration(w.asset)
	}
	position := player.Position()
	if v.scrubbing {
		position = time.Duration(v.scrub * float64(time.Second))
	}
	if duration > 0 {
		position = min(position, duration)
	}
	height := float32(controlSmall + 2*space4)
	width := min(w.picture.viewport.W-2*previewVideoBarInset, max(rect.W-2*previewVideoBarInset, previewVideoBarMin))
	left := max(previewVideoBarInset, min(rect.X+(rect.W-width)/2, w.picture.viewport.W-previewVideoBarInset-width))
	bar := ui.Row(c.Key("preview-video-controls")).Absolute().Left(left).Top(rect.Y+rect.H-height-previewVideoBarInset).Width(width).Height(height).
		Padding(0, space4).Gap(space4).AlignItems(ui.Center).Radius(radiusControl + space4).Background(pal.Sidebar.Alpha(.9))
	elevated(bar, c.Theme())
	idle := now.Sub(v.movedAt)
	visible := !player.Playing() || v.scrubbing || bar.Hovered() || idle < previewVideoHideAfter
	if visible && player.Playing() && idle < previewVideoHideAfter {
		c.After(previewVideoHideAfter - idle)
	}
	target := float32(0)
	if visible {
		target = 1
	}
	fade := ui.Animation{Duration: 200 * time.Millisecond}
	if c.Preferences().ReduceMotion {
		fade = ui.Animation{Duration: time.Millisecond, NonMotion: true}
	}
	bar.Opacity(bar.AnimateTo("preview-video-controls-opacity", target, fade))
	bar.Children(func() {
		icon, label := "pause", a.T("video_pause")
		if !player.Playing() {
			icon, label = "play", a.T("video_play")
		}
		if previewVideoButton(c.Key("preview-video-play"), label, icon, true).Clicked() {
			v.toggle()
			v.movedAt = now
		}
		ui.Text(c.Key("preview-video-time"), videoDuration(position)+" / "+videoDuration(duration)).FontSize(fontSecondary).FontFeatures("tnum").
			TextColor(pal.TextSecondary).SingleLine().Shrink(0).Padding(0, space4)
		w.videoScrubber(c, a, player, duration)
		icon, label = "volume-loud", a.T("video_mute")
		if player.Muted() {
			icon, label = "volume-cross", a.T("video_unmute")
		}
		if previewVideoButton(c.Key("preview-video-mute"), label, icon, false).Clicked() {
			v.toggleMute()
			v.movedAt = now
		}
	})
}

// 进度条：按住拖动时先暂停、实时跳到指针处，松手按拖之前的状态接着播或停着。
func (w *previewWindow) videoScrubber(c *ui.Context, a *App, player *ui.VideoPlayer, duration time.Duration) {
	v, pal := &w.video, paletteFor(c.Theme())
	if !v.scrubbing {
		v.scrub = player.Position().Seconds()
	}
	end := max(duration.Seconds(), .001)
	v.scrub = min(max(v.scrub, 0), end)
	slider := ui.SliderBase(c.Key("preview-video-position"), &v.scrub, 0, end).Grow(1).Shrink(1).MinWidth(0).Height(controlSmall).
		Padding(0, previewVideoThumb/2).Label(a.T("video_position")).Cursor(ui.CursorPointer)
	pressed := slider.Pressed()
	if pressed && !v.scrubbing {
		v.scrubbing, v.resume = true, player.Playing()
		player.Pause()
	}
	if slider.Changed() {
		player.Seek(time.Duration(v.scrub * float64(time.Second)))
		v.movedAt = c.Now()
	}
	if v.scrubbing && !pressed {
		v.scrubbing = false
		if v.resume {
			player.Play()
		}
	}
	at := float32(v.scrub / end)
	if c.Direction() == ui.RightToLeft {
		at = 1 - at
	}
	thumb := slider.Hovered() || v.scrubbing || slider.Focused()
	track, fill := pal.Text.Alpha(themeValue(c, .14, .2)), pal.Accent
	slider.Draw(func(p *ui.Painter, r ui.Rect) {
		x, y := r.X+previewVideoThumb/2, r.Y+(r.H-previewVideoTrack)/2
		length := max(0, r.W-previewVideoThumb)
		p.Fill(ui.Rect{X: x, Y: y, W: length, H: previewVideoTrack}, track, previewVideoTrack/2)
		played := ui.Rect{X: x, Y: y, W: length * at, H: previewVideoTrack}
		if c.Direction() == ui.RightToLeft {
			played = ui.Rect{X: x + length*at, Y: y, W: length * (1 - at), H: previewVideoTrack}
		}
		p.Fill(played, fill, previewVideoTrack/2)
		if thumb {
			p.Fill(ui.Rect{X: x + length*at - previewVideoThumb/2, Y: r.Y + (r.H-previewVideoThumb)/2, W: previewVideoThumb, H: previewVideoThumb}, fill, previewVideoThumb/2)
		}
	})
}

// 控制条上的图标按钮：和检查器图标按钮同一套悬停、按下底色，图标 16。
func previewVideoButton(c *ui.Context, label, icon string, filled bool) ui.Element {
	p := paletteFor(c.Theme())
	button := ui.ButtonBase(c).Label(label).Tooltip(label).Size(controlSmall, controlSmall).Padding(0).Radius(radiusControl).
		Shrink(0).Justify(ui.Center).AlignItems(ui.Center).Cursor(ui.CursorPointer)
	background := ui.Transparent
	if button.Pressed() {
		background = p.Pressed
	} else if button.Hovered() {
		background = p.Hover
	}
	animateColors(button, background, p.Text)
	button.Children(func() { appIconSelected(c, icon, filled).Size(iconMedium, iconMedium) })
	return button
}

// 翻页时新内容从翻页方向滑入、旧的往反方向滑出；不翻页时淡入、缩小淡出。图片和视频共用。
func (w *previewWindow) slideTransition(c *ui.Context, element ui.Element) {
	p := &w.picture
	transition := ui.ElementTransition{Animation: &motionSpring, Colors: true, ScaleContent: true,
		Enter: &ui.Motion{Opacity: 1}, Exit: &ui.Motion{Scale: .94}}
	if p.slideDirection != 0 {
		transition.Enter = &ui.Motion{X: 64 * p.slideDirection, Scale: .98}
		transition.Exit = &ui.Motion{X: -64 * p.slideDirection, Scale: .98}
		p.slideFrames++
		if p.slideFrames > 1 && !element.Transitioning() {
			p.slideDirection, p.slideFrames = 0, 0
			c.Invalidate()
		}
	}
	element.Transition(transition)
}
