package assets

import (
	"image"

	"github.com/Seaony/Momento/app/internal/i18n"
)

// RotateImage 按顺时针旋转，返回新图片，原图保持不动。
// 顺时针 90°、180°、270° 分别等同 EXIF 方向 6、3、8，和转正共用同一套快速路径。
func RotateImage(src image.Image, degrees int) (image.Image, error) {
	rotation, err := normalizedRotation(degrees)
	if err != nil {
		return nil, err
	}
	orientation := map[int]int{90: 6, 180: 3, 270: 8}[rotation]
	if orientation == 0 {
		return src, nil
	}
	return orient(src, orientation), nil
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
