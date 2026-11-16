package media

// Linux 等平台没有统一的系统截帧接口，借用户装的 ffmpeg：找不到 ffmpeg 返回 ErrPosterUnavailable。
// macOS、Windows 的系统接口截不了的格式（如 macOS 上的 WebM）也用它兜底。
// ffmpeg 默认会按旋转元数据自动转正（-autorotate 默认开），缩放滤镜拿到的已经是转正后的宽高。

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// stderr 只留最后这么多字节放进错误里，避免坏文件刷出一大屏日志。
const ffmpegStderrLimit = 1024

func ffmpegPoster(ctx context.Context, path string, at time.Duration, maxSize int) (image.Image, error) {
	ffmpeg, err := findFFmpeg()
	if err != nil {
		return nil, fmt.Errorf("%w: ffmpeg not found: %v", ErrPosterUnavailable, err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("media: poster %s: %w", path, err)
	}
	out, err := runFFmpeg(ctx, ffmpeg, abs, at, maxSize)
	// 视频比 at 短时 ffmpeg 正常退出但一帧都不出，这时改取第一帧
	if err == nil && len(out) == 0 && at > 0 {
		out, err = runFFmpeg(ctx, ffmpeg, abs, 0, maxSize)
	}
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("media: poster %s: ffmpeg decoded no video frame", path)
	}
	img, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		return nil, fmt.Errorf("media: poster %s: decode ffmpeg output: %w", path, err)
	}
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba, nil
	}
	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Bounds(), img, img.Bounds().Min, draw.Src)
	return rgba, nil
}

func runFFmpeg(ctx context.Context, ffmpeg, path string, at time.Duration, maxSize int) ([]byte, error) {
	m := strconv.Itoa(maxSize)
	// 宽高各不超过 maxSize 且保持比例：等价于最长边不超过 maxSize；min(iw,…) 保证小图不放大
	scale := "scale='min(iw," + m + ")':'min(ih," + m + ")':force_original_aspect_ratio=decrease"
	cmd := exec.CommandContext(ctx, ffmpeg,
		"-nostdin", "-v", "error",
		// -ss 放在 -i 前面走快速定位，ffmpeg 会从关键帧解到准确时刻
		"-ss", strconv.FormatFloat(at.Seconds(), 'f', 3, 64),
		// 加 file: 前缀，文件名里带冒号时不会被当成别的协议
		"-i", "file:"+path,
		"-frames:v", "1", "-vf", scale,
		"-pix_fmt", "rgb24", "-f", "image2pipe", "-vcodec", "png", "-")
	var stdout bytes.Buffer
	stderr := &tailBuffer{limit: ffmpegStderrLimit}
	cmd.Stdout = &stdout
	cmd.Stderr = stderr
	// 被 ctx 杀掉后最多再等这么久收尾，防止管道没关导致 Wait 卡住
	cmd.WaitDelay = 2 * time.Second
	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		msg := strings.TrimSpace(stderr.String())
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && isMissingDecoder(msg) {
			// 文件没问题，只是这份 ffmpeg 没编进对应解码器
			return nil, fmt.Errorf("%w: ffmpeg %s: %s", ErrPosterUnavailable, path, msg)
		}
		return nil, fmt.Errorf("media: poster %s: ffmpeg: %v: %s", path, err, msg)
	}
	return stdout.Bytes(), nil
}

// isMissingDecoder 认 ffmpeg 新旧版本“没有解码器”的两种说法。
func isMissingDecoder(stderr string) bool {
	return strings.Contains(stderr, "no decoder found") ||
		(strings.Contains(stderr, "Decoder (codec") && strings.Contains(stderr, "not found"))
}

// tailBuffer 只保留写入内容的最后 limit 个字节。
type tailBuffer struct {
	buf   []byte
	limit int
}

func (t *tailBuffer) Write(p []byte) (int, error) {
	t.buf = append(t.buf, p...)
	if over := len(t.buf) - t.limit; over > 0 {
		t.buf = append(t.buf[:0], t.buf[over:]...)
	}
	return len(p), nil
}

func (t *tailBuffer) String() string { return string(t.buf) }

// findFFmpeg 先找 PATH；从访达、程序坞启动的 macOS 应用拿不到终端里的 PATH，再看 Homebrew 的常见位置。
func findFFmpeg() (string, error) {
	path, err := exec.LookPath("ffmpeg")
	if err == nil {
		return path, nil
	}
	for _, candidate := range []string{"/opt/homebrew/bin/ffmpeg", "/usr/local/bin/ffmpeg"} {
		if found, lookErr := exec.LookPath(candidate); lookErr == nil {
			return found, nil
		}
	}
	return "", err
}
