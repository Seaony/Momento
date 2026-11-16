//go:build !darwin && !windows

package media

import (
	"fmt"
	"image"
)

// SystemImageDecoder：Linux 等平台没有系统图片解码接口。
const SystemImageDecoder = false

// ImageSize 在这个平台读不了 RAW、PSD 这类格式。
func ImageSize(path string) (int, int, error) {
	return 0, 0, fmt.Errorf("%w: %s", ErrUnsupported, path)
}

// ImageThumbnail 在这个平台拿不到缩略图，调用方用通用封面。
func ImageThumbnail(path string, _ int) (image.Image, error) {
	return nil, fmt.Errorf("%w: %s", ErrPosterUnavailable, path)
}

// DecodeImage 在这个平台解不了 RAW、PSD 这类格式。
func DecodeImage(path string) (image.Image, error) {
	return nil, fmt.Errorf("%w: %s", ErrUnsupported, path)
}

// PreviewImage 在这个平台也拿不到。
func PreviewImage(path string) (image.Image, error) {
	return DecodeImage(path)
}
