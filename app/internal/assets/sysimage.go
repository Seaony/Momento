package assets

import (
	"context"
	"errors"
	"image"
	"image/draw"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/media"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// 系统解码的几个入口，测试里可以换掉。
var (
	systemImageSize      = media.ImageSize
	systemImageThumbnail = media.ImageThumbnail
	systemDecodeImage    = media.DecodeImage
	systemPreviewImage   = media.PreviewImage
)

// useSystemDecoder 判断这种格式（带点、小写）交给系统解码：RAW、PSD 等只能这样；
// macOS 上 HEIC/AVIF 也走系统，比纯 Go 解码快十几倍。
func useSystemDecoder(ext string) bool {
	if media.IsSystemImageExtension(ext) {
		return true
	}
	return media.SystemImageDecoder && (ext == ".heic" || ext == ".heif" || ext == ".avif")
}

// systemOnly 表示这种格式只有系统能解，系统解不了就没有别的办法。
func systemOnly(ext string) bool { return media.IsSystemImageExtension(ext) }

// tooLarge 按像素上限拒绝，和内置解码器同一个错误。
func tooLarge(path string, w, h int) error {
	if int64(w)*int64(h) > maxImagePixels {
		return i18n.WrapError(i18n.NewError("error_image_too_large", w, h), "error_decode_image", filepath.Base(path))
	}
	return nil
}

// prepareSystemThumbnail 让系统直接出转正后的缩略图，不解全尺寸图。
// 系统给不了缩略图（Linux，或 Windows 没装对应解码器）时导入原文件，用印着格式名的通用封面，元数据记下 posterUnavailable。
func prepareSystemThumbnail(ctx context.Context, path, ext string) (preparedImage, error) {
	w, h, sizeErr := systemImageSize(path)
	if sizeErr == nil {
		if err := tooLarge(path, w, h); err != nil {
			return preparedImage{}, err
		}
	}
	// 系统解码也要占内存（RAW 要整张冲洗），按尺寸排队，和内置解码器共用额度。
	release, err := decodeBudget.admit(ctx, image.Config{Width: w, Height: h}, sizeErr == nil)
	if err != nil {
		return preparedImage{}, err
	}
	thumb, err := systemImageThumbnail(path, thumbnailSize)
	release()
	if err := ctx.Err(); err != nil {
		return preparedImage{}, err
	}
	unavailable := false
	if err != nil {
		if !systemOnly(ext) || !errors.Is(err, media.ErrPosterUnavailable) {
			return preparedImage{}, i18n.WrapError(err, "error_decode_image", filepath.Base(path))
		}
		slog.Info("这台机器取不到这种图片的缩略图，用通用封面", "file", filepath.Base(path), "error", err)
		thumb, unavailable = formatPoster(strings.TrimPrefix(ext, ".")), true
	}
	if sizeErr != nil {
		// 读不出真实尺寸（Windows、Linux）：宽高记 0，界面不显示尺寸，按默认比例排版。
		w, h = 0, 0
	}
	metadata, orientation, profile := readMetadata(path, thumb)
	if w > 0 && h > 0 {
		metadata["pixelWidth"], metadata["pixelHeight"] = strconv.Itoa(w), strconv.Itoa(h)
	} else {
		delete(metadata, "pixelWidth")
		delete(metadata, "pixelHeight")
	}
	if unavailable {
		metadata["posterUnavailable"] = "1"
		delete(metadata, "colorModel")
	}
	return preparedImage{thumbnail: thumb, width: w, height: h, metadata: metadata, orientation: orientation, profile: profile}, nil
}

// decodeSystemImage 让系统解出转正后的全尺寸图，给预览和导出用。
func decodeSystemImage(path string) (image.Image, error) {
	if w, h, err := systemImageSize(path); err == nil {
		if err := tooLarge(path, w, h); err != nil {
			return nil, err
		}
	}
	img, err := systemDecodeImage(path)
	if errors.Is(err, media.ErrUnsupported) && systemOnly(strings.ToLower(filepath.Ext(path))) {
		// 这台电脑解不了原图（Windows、Linux 上的 RAW、PSD 等）：说清楚只能导出原文件，不拿缩略图冒充。
		return nil, i18n.WrapError(err, "error_system_decode_unavailable", strings.ToUpper(strings.TrimPrefix(filepath.Ext(path), ".")))
	}
	if err != nil {
		return nil, i18n.WrapError(err, "error_decode_image", filepath.Base(path))
	}
	return img, nil
}

// PreviewImage 给空格预览用：和 PrepareImage 一样，但这台电脑解不了原图的格式退而用系统给的大缩略图。
func PreviewImage(path string) (image.Image, error) {
	img, _, _, _, err := PrepareImage(path)
	if err == nil || !systemOnly(strings.ToLower(filepath.Ext(path))) {
		return img, err
	}
	preview, previewErr := systemPreviewImage(path)
	if previewErr != nil {
		return nil, err
	}
	return preview, nil
}

// formatPoster 画一张 4:3 的通用封面，正中印着格式名（如 PSD），让人一眼知道这是什么文件。
func formatPoster(format string) image.Image {
	label := strings.ToUpper(format)
	face := basicfont.Face7x13
	// 先在小图上用点阵字画字，再整体放大，字是清楚的方块像素。
	const scale = 8
	small := image.NewRGBA(image.Rect(0, 0, thumbnailSize/scale, thumbnailSize*3/4/scale))
	draw.Draw(small, small.Bounds(), &image.Uniform{C: genericPosterColor}, image.Point{}, draw.Src)
	width := font.MeasureString(face, label).Ceil()
	drawer := font.Drawer{Dst: small, Src: image.NewUniform(genericPosterLabel), Face: face,
		Dot: fixed.P((small.Bounds().Dx()-width)/2, (small.Bounds().Dy()+face.Ascent-face.Descent)/2)}
	drawer.DrawString(label)
	out := image.NewRGBA(image.Rect(0, 0, thumbnailSize, thumbnailSize*3/4))
	xdraw.NearestNeighbor.Scale(out, out.Bounds(), small, small.Bounds(), xdraw.Src, nil)
	return out
}
