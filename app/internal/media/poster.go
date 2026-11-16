package media

import (
	"context"
	"errors"
	"fmt"
	"image"
	"os"
	"time"
)

// Poster 取视频 at 时刻的一帧当封面，转正后缩到最长边不超过 maxSize（原图更小不放大）。
//
//   - at <= 0 或视频比 at 短时取第一帧。Windows 用系统缩略图，没法指定时刻，at 不起作用。
//   - 按视频轨的旋转元数据转正，返回 *image.RGBA。
//   - ctx 取消或超时会尽快返回 ctx.Err()，损坏文件不会让调用方一直卡住。
//   - 这台机器没能力截帧（Linux 没装 ffmpeg、macOS 不认 WebM 容器、缺解码器）时返回的错误
//     满足 errors.Is(err, ErrPosterUnavailable)，调用方改用通用视频封面；
//     文件本身坏了返回带原因的普通错误，不会伪装成 ErrPosterUnavailable。
//
// 可以在多个 goroutine 里并发调用。
func Poster(ctx context.Context, path string, at time.Duration, maxSize int) (image.Image, error) {
	if maxSize <= 0 {
		return nil, fmt.Errorf("media: poster maxSize must be positive, got %d", maxSize)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// 先确认文件在，各平台对“文件不存在”的报错五花八门，这里统一成 *PathError
	st, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("media: poster: %w", err)
	}
	if st.IsDir() {
		return nil, fmt.Errorf("media: poster: %s is a directory", path)
	}
	if at < 0 {
		at = 0
	}
	img, err := poster(ctx, path, at, maxSize)
	// 系统接口截不了（如 macOS 不认 WebM）时，装了 ffmpeg 就用它再试一次；Linux 本来就是 ffmpeg，不重复跑。
	if errors.Is(err, ErrPosterUnavailable) && systemPoster && ctx.Err() == nil {
		if fallback, ffErr := ffmpegPoster(ctx, path, at, maxSize); ffErr == nil {
			return fallback, nil
		} else if !errors.Is(ffErr, ErrPosterUnavailable) {
			return nil, ffErr
		}
	}
	return img, err
}

// fitSize 把 w×h 等比缩到最长边不超过 maxSize，原本更小就不动；结果至少 1 像素。
func fitSize(w, h, maxSize int) (int, int) {
	if w <= maxSize && h <= maxSize {
		return w, h
	}
	if w >= h {
		return maxSize, max(1, int(float64(h)*float64(maxSize)/float64(w)+0.5))
	}
	return max(1, int(float64(w)*float64(maxSize)/float64(h)+0.5)), maxSize
}
