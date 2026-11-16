//go:build !darwin && !windows

package media

import (
	"context"
	"image"
	"time"
)

// systemPoster 表示 poster 用的是系统接口（不是 ffmpeg），截不了时才值得再用 ffmpeg 试。
const systemPoster = false

// Linux 等平台直接用 ffmpeg 截帧。
func poster(ctx context.Context, path string, at time.Duration, maxSize int) (image.Image, error) {
	return ffmpegPoster(ctx, path, at, maxSize)
}
