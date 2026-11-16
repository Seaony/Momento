//go:build windows

package media

import (
	"fmt"
	"image"
	"path/filepath"
)

// SystemImageDecoder：Windows 只能拿系统缩略图，不能完整解码 RAW、PSD 这类格式。
const SystemImageDecoder = false

// systemPreviewSize 是 Windows 上 RAW、PSD 等格式预览能拿到的最大边长（系统缩略图）。
const systemPreviewSize = 2560

// ImageSize 在 Windows 上读不了，调用方按缩略图比例估算。
func ImageSize(path string) (int, int, error) {
	return 0, 0, fmt.Errorf("%w: %s", ErrUnsupported, path)
}

// ImageThumbnail 用系统缩略图（装了 RAW 图像扩展等解码器才有），拿不到返回 ErrPosterUnavailable。
func ImageThumbnail(path string, maxSize int) (image.Image, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return shellThumbnail(abs, maxSize)
}

// DecodeImage 在 Windows 上做不到完整解码：转格式导出不能拿缩略图冒充原图。
func DecodeImage(path string) (image.Image, error) {
	return nil, fmt.Errorf("%w: %s", ErrUnsupported, path)
}

// PreviewImage 给预览用：Windows 上拿大尺寸的系统缩略图。
func PreviewImage(path string) (image.Image, error) {
	return ImageThumbnail(path, systemPreviewSize)
}
