package ui

import (
	"log/slog"
	"path/filepath"
	"time"

	"github.com/egoist/mygo/ui"
)

// previewLive 是预览窗口里实况照片的那段视频：悬停或点「实况」时在照片上播一遍，播完回到照片。
type previewLive struct {
	player  *ui.VideoPlayer
	id      string
	hovered bool // 上一帧鼠标在「实况」上，刚移上去那一下才开始播
}

// closeLive 关掉实况视频，回到照片。换素材、窗口关闭时调。
func (w *previewWindow) closeLive() {
	if w.live.player != nil {
		w.live.player.Close()
	}
	// 保留「鼠标在不在『实况』上」：播完时鼠标还停着，不能当成刚移上去又播一遍。
	w.live = previewLive{hovered: w.live.hovered}
}

// playLive 从头播一遍当前实况照片的视频，带声音（窗口里点过静音就静音），和系统照片一样。
func (w *previewWindow) playLive(a *App) {
	w.closeLive()
	path := filepath.Join(w.root, filepath.FromSlash(w.asset.MotionPath))
	player, err := openPreviewVideo(path)
	if err != nil {
		slog.Error("实况照片的视频打不开", "asset", w.asset.ID, "error", err)
		a.ShowError(err)
		return
	}
	w.live.player, w.live.id = player.SetLoop(false).SetMuted(w.video.muted), w.asset.ID
}

// liveOverlay 在照片上叠实况视频，并在照片左上角放「实况」：移上去或点一下就播。photo 是照片当前在视口里的位置和大小。
func (w *previewWindow) liveOverlay(c *ui.Context, a *App, photo ui.Rect) {
	l, pal := &w.live, paletteFor(c.Theme())
	if l.id != "" && l.id != w.asset.ID {
		w.closeLive()
	}
	if player := l.player; player != nil {
		switch err := player.Err(); {
		case err != nil:
			slog.Error("实况照片的视频播放出错", "asset", w.asset.ID, "error", err)
			a.ShowError(err)
			w.closeLive()
		case !player.Playing():
			// 实况视频没有暂停按钮，不在播了就是播完了。
			w.closeLive()
		}
	}
	if l.player != nil {
		// 视频和照片同样大小、同一位置，不挡缩放和拖动；播放中隔一会儿看一眼播完没有。
		ui.Video(c.Key("preview-live-video"), l.player).Absolute().Left(photo.X).Top(photo.Y).Size(photo.W, photo.H).
			Fit(ui.Contain).Radius(previewPhotoRadius).PassThrough()
		c.After(100 * time.Millisecond)
	}
	left := max(photo.X, 0) + space12
	top := max(photo.Y, 0) + space12
	badge := ui.ButtonBase(c.Key("preview-live")).Label(a.T("live_photo_play")).Absolute().Left(left).Top(top).Height(controlSmall-4).
		Padding(0, space8).Gap(space4).AlignItems(ui.Center).Radius(radiusControl).Cursor(ui.CursorPointer)
	background := pal.Sidebar.Alpha(.9)
	if badge.Pressed() {
		background = pal.Pressed
	} else if badge.Hovered() {
		background = pal.Hover
	}
	animateColors(elevated(badge, c.Theme()), background, pal.Text)
	badge.Children(func() {
		appIcon(c, "live-photo").Size(iconSmall, iconSmall)
		ui.Text(c, a.T("live_photo")).FontSize(fontSecondary).FontWeight(500).SingleLine()
	})
	hovered := badge.Hovered()
	if hovered && !l.hovered || badge.Clicked() {
		w.playLive(a)
	}
	l.hovered = hovered
}
