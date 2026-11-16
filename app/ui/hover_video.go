package ui

import (
	"log/slog"
	"path/filepath"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/video"
)

// 鼠标在视频卡片上停这么久才开始播，划过一排卡片时不会每张都打开播放器。
const hoverVideoDelay = 350 * time.Millisecond

// openHoverVideo 打开卡片悬停播放用的播放器，测试里可以换掉。
var openHoverVideo = video.Open

// hoverVideo 是网格里正在悬停播放的那一个视频：同一时间只有一个，鼠标离开就关掉。
type hoverVideo struct {
	id     string
	since  time.Time
	player *ui.VideoPlayer
	failed bool    // 这个视频打不开或播放出错，悬停期间不再重试
	lastX  float32 // 上一帧鼠标在卡片里的横坐标，变了才拖进度；-1 表示刚进来
}

// hoverVideoOverlay 在视频卡片的缩略图上叠播放画面和底部细进度条：静音循环播放，左右移动鼠标就拖动进度。
func (a *App) hoverVideoOverlay(c *ui.Context, card ui.Element, asset model.Asset, hovered bool, radius float32) {
	if !hovered || !a.Preferences.HoverPlayVideos || !isVideoAsset(asset) || a.Library == nil {
		return
	}
	h := &a.hoverVideoState
	if h.id != asset.ID {
		a.closeHoverVideo()
		h.id, h.since, h.lastX = asset.ID, c.Now(), -1
	}
	if h.failed {
		return
	}
	if h.player == nil {
		if wait := hoverVideoDelay - c.Now().Sub(h.since); wait > 0 {
			c.After(wait)
			return
		}
		path := filepath.Join(a.Library.Root, filepath.FromSlash(asset.RelativePath))
		player, err := openHoverVideo(path)
		if err != nil {
			slog.Warn("悬停播放打不开视频", "id", asset.ID, "error", err)
			h.failed = true
			return
		}
		player.SetMuted(true).SetLoop(true)
		h.player = player
	}
	player := h.player
	if err := player.Err(); err != nil {
		slog.Warn("悬停播放出错", "id", asset.ID, "error", err)
		a.closeHoverVideo()
		h.id, h.failed = asset.ID, true
		return
	}
	duration := player.Duration()
	if x, _, over := card.PointerPosition(); over {
		if width := card.Bounds().W; h.lastX >= 0 && x != h.lastX && width > 0 && duration > 0 {
			player.Seek(time.Duration(float64(duration) * float64(min(max(x/width, 0), 1))))
		}
		h.lastX = x
	}
	// 播放器一建好就交给视频控件：系统播放器要被显示才开始播；出画面前是透明的，底下的封面照常露着。
	ui.Video(c.Key("hover-video"), player).Absolute().Top(0).Left(0).Right(0).Bottom(0).Fit(ui.Cover).Radius(radius).PassThrough()
	if duration > 0 {
		progress := float32(float64(player.Position()) / float64(duration))
		width := card.Bounds().W * min(max(progress, 0), 1)
		ui.Box(c.Key("hover-video-progress")).Absolute().Left(0).Bottom(0).Height(2).Width(width).
			Background(ui.RGBA(255, 255, 255, .85)).PassThrough()
	}
}

// closeHoverVideo 关掉悬停播放器；鼠标离开卡片、换图库、关掉设置开关时调用。
func (a *App) closeHoverVideo() {
	h := &a.hoverVideoState
	if h.player != nil {
		h.player.Close()
	}
	*h = hoverVideo{}
}

// finishHoverVideo 在每帧网格画完后调用：这一帧鼠标不在那张卡片上了就关掉播放器。
func (a *App) finishHoverVideo() {
	if h := &a.hoverVideoState; h.id != "" && (h.id != a.hovered || !a.Preferences.HoverPlayVideos) {
		a.closeHoverVideo()
	}
}
