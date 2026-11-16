package assets

import (
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/Seaony/Momento/app/internal/i18n"
)

// 缩略图不透明的存 JPEG，带透明的存 PNG。旧图库只有 PNG，读的时候两种都认，优先 JPEG。
var thumbnailExtensions = []string{".jpg", ".png"}

// thumbnailBase 返回不带后缀的缩略图相对路径，顺便校验素材 ID，防止拼出图库外的路径。
func thumbnailBase(id string) (string, error) {
	if len(id) != 64 || strings.ContainsAny(id, "/\\") {
		return "", i18n.NewError("error_asset_id", id)
	}
	for _, r := range id {
		if !strings.ContainsRune("0123456789abcdef", r) {
			return "", i18n.NewError("error_asset_id", id)
		}
	}
	return filepath.Join("thumbnails", id), nil
}

// ThumbnailPath 返回素材缩略图在图库里实际存在的完整路径，.jpg 和 .png 都认，优先 .jpg。
// 两种都没有时返回的错误满足 errors.Is(err, os.ErrNotExist)；ID 不合法或读目录出错时返回对应错误。
func ThumbnailPath(root, id string) (string, error) {
	base, err := thumbnailBase(id)
	if err != nil {
		return "", err
	}
	var missing error
	for _, ext := range thumbnailExtensions {
		path := filepath.Join(root, base+ext)
		_, err := os.Stat(path)
		if err == nil {
			return path, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		missing = err
	}
	return "", missing
}

// WriteThumbnail 把图缩到最长边 512，原子写到 base 加 .jpg（不透明，质量 85）或 .png（带透明），
// 再删掉同一张的另一种格式，返回写好的完整路径。base 不带后缀。
func WriteThumbnail(base string, img image.Image) (string, error) {
	if err := os.MkdirAll(filepath.Dir(base), 0700); err != nil {
		return "", err
	}
	img = resized(img, thumbnailSize)
	ext := ".png"
	if opaque(img) {
		ext = ".jpg"
	}
	f, err := os.CreateTemp(filepath.Dir(base), ".thumbnail-*"+ext)
	if err != nil {
		return "", err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if ext == ".jpg" {
		// 不透明的 NRGBA 和 RGBA 字节一样，换成 RGBA 让编码器走快速路径。
		if n, ok := img.(*image.NRGBA); ok {
			img = &image.RGBA{Pix: n.Pix, Stride: n.Stride, Rect: n.Rect}
		}
		err = jpeg.Encode(f, img, &jpeg.Options{Quality: 85})
	} else {
		err = png.Encode(f, img)
	}
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", err
	}
	path := base + ext
	if err := os.Rename(tmp, path); err != nil {
		return "", err
	}
	return path, removeOtherThumbnails(base, ext)
}

// removeOtherThumbnails 删掉同一张缩略图除 keep 以外的格式，刷新后不留两份。
func removeOtherThumbnails(base, keep string) error {
	var err error
	for _, ext := range thumbnailExtensions {
		if ext != keep {
			err = errors.Join(err, removeIfExists(base+ext))
		}
	}
	return err
}

func opaque(img image.Image) bool {
	if o, ok := img.(interface{ Opaque() bool }); ok {
		return o.Opaque()
	}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a != 0xffff {
				return false
			}
		}
	}
	return true
}
