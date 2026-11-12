package assets

import (
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/gen2brain/avif"
	"github.com/gen2brain/heic"
	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// Decode 返回已经转正的第一帧，原文件保持不动。
func Decode(path string) (image.Image, error) {
	img, _, _, _, err := PrepareImage(path)
	return img, err
}

func PrepareImage(path string) (image.Image, map[string]string, int, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, 0, "", err
	}
	defer f.Close()
	var img image.Image
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".heic", ".heif":
		img, err = heic.Decode(f)
	case ".avif":
		img, err = avif.Decode(f, avif.Options{AutoRotate: true})
	default:
		img, _, err = image.Decode(f)
	}
	if err != nil {
		return nil, nil, 0, "", i18n.WrapError(err, "error_decode_image", filepath.Base(path))
	}
	metadata, orientation, profile := readMetadata(path, img)
	// HEIC 和 AVIF 的解码器已应用容器方向，不能再转一次。
	if ext != ".heic" && ext != ".heif" && ext != ".avif" {
		img = orient(img, orientation)
	}
	return img, metadata, orientation, profile, nil
}

func orient(src image.Image, orientation int) image.Image {
	if orientation < 2 || orientation > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dw, dh := w, h
	if orientation >= 5 {
		dw, dh = h, w
	}
	dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	for y := range h {
		for x := range w {
			dx, dy := x, y
			switch orientation {
			case 2:
				dx = w - 1 - x
			case 3:
				dx, dy = w-1-x, h-1-y
			case 4:
				dy = h - 1 - y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = h-1-y, x
			case 7:
				dx, dy = h-1-y, w-1-x
			case 8:
				dx, dy = y, w-1-x
			}
			dst.Set(dx, dy, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func resized(src image.Image, longest int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if max(w, h) <= longest {
		return src
	}
	if w >= h {
		h = max(1, h*longest/w)
		w = longest
	} else {
		w = max(1, w*longest/h)
		h = longest
	}
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Src, nil)
	return dst
}

func WriteThumbnail(dst string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(dst), ".thumbnail-*.png")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err := png.Encode(f, resized(img, 512)); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

func imageColorModel(img image.Image) string {
	switch img.ColorModel() {
	case color.GrayModel, color.Gray16Model:
		return "Gray"
	case color.CMYKModel:
		return "CMYK"
	default:
		return "RGB"
	}
}
