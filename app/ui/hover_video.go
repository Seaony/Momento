package ui

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Seaony/Momento/app/internal/assets"
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
	id      string
	since   time.Time
	player  *ui.VideoPlayer
	failed  bool    // 这个视频打不开或播放出错，悬停期间不再重试
	opening bool    // 动图正在后台读文件和解码
	lastX   float32 // 上一帧鼠标在卡片里的横坐标，变了才拖进度；-1 表示刚进来
}

// 动图超过这么大就不在网格里悬停播放：GIF 要整个解码进内存。
const maxHoverAnimationBytes = 64 << 20

// hoverAnimation 判断图片是不是可以悬停播放的动图（GIF、APNG），返回它的格式。
func hoverAnimation(asset model.Asset) string {
	switch strings.ToLower(asset.Extension) {
	case "gif":
		return "gif"
	case "png", "apng":
		return "png"
	}
	return ""
}

// openHoverAnimation 在后台读出动图做成播放器；不是动画的 PNG 返回 nil。
func openHoverAnimation(path, format string) (*ui.VideoPlayer, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > maxHoverAnimationBytes {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var decoder ui.VideoDecoder
	if format == "gif" {
		decoder, err = ui.NewGIFDecoder(data)
	} else {
		if animated, _ := assets.IsAnimatedImage(bytes.NewReader(data)); !animated {
			return nil, nil
		}
		decoder, err = ui.NewAPNGDecoder(data)
	}
	if err != nil {
		return nil, err
	}
	return ui.NewVideoPlayer(decoder)
}

// hoverVideoOverlay 在视频（和实况照片）卡片的缩略图上叠播放画面和底部细进度条：静音循环播放，左右移动鼠标就拖动进度。
func (a *App) hoverVideoOverlay(c *ui.Context, card ui.Element, asset model.Asset, hovered bool, radius float32) {
	relative, animation := motionVideo(asset), ""
	if relative == "" {
		// GIF、APNG 动图也悬停播放，播放的是文件本身。
		if animation = hoverAnimation(asset); animation != "" {
			relative = asset.RelativePath
		}
	}
	if !hovered || !a.Preferences.HoverPlayVideos || relative == "" || a.Library == nil {
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
		path := filepath.Join(a.Library.Root, filepath.FromSlash(relative))
		if animation != "" {
			// 动图在后台读和解码，大 GIF 不卡界面；读好时鼠标已经移走就直接关掉。
			if !h.opening {
				h.opening = true
				id := asset.ID
				go func() {
					player, err := openHoverAnimation(path, animation)
					a.Window.Update(func() {
						if a.hoverVideoState.id != id || !a.hoverVideoState.opening {
							if player != nil {
								player.Close()
							}
							return
						}
						a.hoverVideoState.opening = false
						switch {
						case err != nil:
							slog.Warn("悬停播放打不开动图", "id", id, "error", err)
							a.hoverVideoState.failed = true
						case player == nil:
							// 不是动画（普通 PNG）或太大，不播。
							a.hoverVideoState.failed = true
						default:
							player.SetMuted(true).SetLoop(true)
							a.hoverVideoState.player = player
						}
					})
				}()
			}
			return
		}
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
	// 素材存了旋转角度时，画面和缩略图一样转过去：转 90° 或 270° 时先按宽高互换的框铺满，再绕中心转，转完正好盖住卡片。
	video := ui.Video(c.Key("hover-video"), player).Absolute().Fit(ui.Cover).Radius(radius).PassThrough()
	switch rotation := asset.Rotation % 360; rotation {
	case 90, 270:
		box := card.Bounds()
		video.Size(box.H, box.W).Left((box.W - box.H) / 2).Top((box.H - box.W) / 2).Rotate(float32(rotation))
	case 180:
		video.Top(0).Left(0).Right(0).Bottom(0).Rotate(180)
	default:
		video.Top(0).Left(0).Right(0).Bottom(0)
	}
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
