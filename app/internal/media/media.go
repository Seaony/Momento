// Package media 读视频的基本信息（时长、尺寸、旋转）和截取封面帧，给导入视频和实况照片用。
// 读头信息是纯 Go；截封面用各平台系统接口（macOS AVFoundation、Windows 系统缩略图、Linux ffmpeg），不用 cgo。
// 和 internal 下其他包一样，不引用 mygo 和界面代码，能单独测试。
package media

import (
	"errors"
	"time"
)

// VideoExtensions 是第一期支持导入的视频格式（小写、不带点）。
var VideoExtensions = []string{"mp4", "mov", "m4v", "webm"}

// IsVideoExtension 判断扩展名（可带点、不分大小写）是不是支持的视频格式。
func IsVideoExtension(ext string) bool {
	ext = lowerTrimDot(ext)
	for _, v := range VideoExtensions {
		if v == ext {
			return true
		}
	}
	return false
}

// Info 是从文件头读出的视频信息。
type Info struct {
	// Width、Height 是编码尺寸；Rotation 是播放时要顺时针转的角度（0、90、180、270），
	// 来自 MP4/MOV 视频轨的变换矩阵。显示尺寸用 DisplaySize。
	Width, Height int
	Rotation      int
	Duration      time.Duration
	// Codec 是视频编码的简称（如 avc1、hvc1、vp9、av1），读不到时为空。
	Codec string
	// FrameRate 是平均帧率，读不到时为 0。
	FrameRate float64
	// HasAudio 表示有音频轨。
	HasAudio bool
	// ContentID 是苹果实况照片里视频那半的内容 ID（com.apple.quicktime.content.identifier），没有时为空。
	ContentID string
}

// DisplaySize 返回按 Rotation 转正后的宽高。
func (i Info) DisplaySize() (int, int) {
	if i.Rotation == 90 || i.Rotation == 270 {
		return i.Height, i.Width
	}
	return i.Width, i.Height
}

// ErrUnsupported 表示文件不是能识别的视频容器（或格式损坏到读不出头信息）。
var ErrUnsupported = errors.New("media: unsupported video container")

// ErrPosterUnavailable 表示这台机器上取不到封面（比如 Linux 没装 ffmpeg），调用方改用通用视频封面。
var ErrPosterUnavailable = errors.New("media: video poster unavailable on this system")

func lowerTrimDot(ext string) string {
	b := []byte(ext)
	if len(b) > 0 && b[0] == '.' {
		b = b[1:]
	}
	for i, c := range b {
		if 'A' <= c && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}
	return string(b)
}
