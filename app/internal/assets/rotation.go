package assets

import (
	"image"
	"image/color"

	"github.com/Seaony/Momento/app/internal/i18n"
)

// RotateImage 按顺时针旋转，返回新图片，原图保持不动。
func RotateImage(src image.Image, degrees int) (image.Image, error) {
	rotation, err := normalizedRotation(degrees)
	if err != nil {
		return nil, err
	}
	if rotation == 0 {
		return src, nil
	}
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	size := rotatedSize(width, height, rotation)
	var dst interface {
		image.Image
		Set(int, int, color.Color)
	}
	switch src.ColorModel() {
	case color.Gray16Model, color.RGBA64Model, color.NRGBA64Model:
		dst = image.NewNRGBA64(image.Rect(0, 0, size.X, size.Y))
	default:
		dst = image.NewNRGBA(image.Rect(0, 0, size.X, size.Y))
	}
	for y := range height {
		for x := range width {
			point := rotatedPoint(x, y, width, height, rotation)
			dst.Set(point.X, point.Y, src.At(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}
	return dst, nil
}

func normalizedRotation(degrees int) (int, error) {
	if degrees%90 != 0 {
		return 0, i18n.NewError("error_rotation")
	}
	return (degrees%360 + 360) % 360, nil
}

func rotatedSize(width, height, rotation int) image.Point {
	if rotation == 90 || rotation == 270 {
		return image.Pt(height, width)
	}
	return image.Pt(width, height)
}

func rotatedPoint(x, y, width, height, rotation int) image.Point {
	switch rotation {
	case 90:
		return image.Pt(height-1-y, x)
	case 180:
		return image.Pt(width-1-x, height-1-y)
	case 270:
		return image.Pt(y, width-1-x)
	default:
		return image.Pt(x, y)
	}
}
