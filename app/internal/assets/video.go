package assets

import (
	"context"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/media"
	"github.com/Seaony/Momento/app/internal/model"
)

// posterAt 是取封面的时刻：第 1 秒那一帧比第一帧更有代表性（第一帧常是黑场），不足 1 秒取第一帧。
const posterAt = time.Second

// posterTimeout 限制截一张封面的时间，坏文件不能让整批导入卡住。
const posterTimeout = 20 * time.Second

// genericPosterColor 是取不到封面时（比如 Linux 没装 ffmpeg）通用封面的底色，界面会在上面画视频标记。
var genericPosterColor = color.RGBA{R: 0x2b, G: 0x2b, B: 0x30, A: 0xff}

// genericPosterLabel 是通用封面上格式名的颜色。
var genericPosterLabel = color.RGBA{R: 0xb8, G: 0xb8, B: 0xc0, A: 0xff}

// isVideo 判断按扩展名（不带点、小写）是不是视频文件。
func isVideo(extension string) bool { return media.IsVideoExtension(extension) }

// probeVideo 和 videoPoster 可在测试里替换。
var (
	probeVideo  = media.Probe
	videoPoster = media.Poster
)

// prepareVideoThumbnail 读视频的尺寸、时长和编码信息，截一帧当缩略图。
// 取不到封面但视频本身没坏时，用视频比例的纯色通用封面，并在元数据里记下 posterUnavailable。
func prepareVideoThumbnail(ctx context.Context, path string) (preparedImage, error) {
	info, err := probeVideo(path)
	if err != nil {
		if errors.Is(err, media.ErrUnsupported) {
			return preparedImage{}, i18n.WrapError(err, "error_video_unreadable")
		}
		return preparedImage{}, err
	}
	w, h := info.DisplaySize()
	if w <= 0 || h <= 0 {
		return preparedImage{}, i18n.NewError("error_video_unreadable")
	}
	metadata := map[string]string{
		"pixelWidth":  strconv.Itoa(w),
		"pixelHeight": strconv.Itoa(h),
		"durationMS":  strconv.FormatInt(info.Duration.Milliseconds(), 10),
	}
	if info.Codec != "" {
		metadata["videoCodec"] = info.Codec
	}
	if info.FrameRate > 0 {
		metadata["frameRate"] = strconv.FormatFloat(info.FrameRate, 'f', 2, 64)
	}
	if info.HasAudio {
		metadata["hasAudio"] = "1"
	}
	if stat, statErr := os.Stat(path); statErr == nil {
		metadata["fileModifiedAt"] = stat.ModTime().UTC().Format(time.RFC3339Nano)
	}
	posterCtx, cancel := context.WithTimeout(ctx, posterTimeout)
	defer cancel()
	thumb, err := videoPoster(posterCtx, path, posterAt, thumbnailSize)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return preparedImage{}, ctxErr
	}
	if err != nil {
		if !errors.Is(err, media.ErrPosterUnavailable) {
			return preparedImage{}, i18n.WrapError(err, "error_video_poster")
		}
		slog.Info("这台机器取不到视频封面，用通用封面", "file", filepath.Base(path), "error", err)
		metadata["posterUnavailable"] = "1"
		thumb = genericPoster(w, h)
	}
	return preparedImage{thumbnail: thumb, width: w, height: h, metadata: metadata, orientation: 1, kind: model.KindVideo, durationMS: info.Duration.Milliseconds()}, nil
}

// genericPoster 画一张和视频同比例的纯色图，最长边是缩略图尺寸。
func genericPoster(w, h int) image.Image {
	dw, dh := fitWithin(w, h, thumbnailSize)
	if w < thumbnailSize && h < thumbnailSize {
		// 小视频也画到缩略图尺寸：只是底色，没有清晰度问题。
		scale := float64(thumbnailSize) / float64(max(w, h))
		dw, dh = max(1, int(float64(w)*scale)), max(1, int(float64(h)*scale))
	}
	img := image.NewRGBA(image.Rect(0, 0, dw, dh))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: genericPosterColor}, image.Point{}, draw.Src)
	return img
}

// videoMediaType 返回视频的 MIME 类型。
func videoMediaType(ext string) string {
	switch strings.ToLower(ext) {
	case "mov":
		return "video/quicktime"
	case "m4v":
		return "video/x-m4v"
	default:
		return "video/" + strings.ToLower(ext)
	}
}
