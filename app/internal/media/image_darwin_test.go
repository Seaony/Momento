//go:build darwin

package media

import (
	"errors"
	"path/filepath"
	"testing"
)

// 系统能解的格式：读得出尺寸、缩略图按最长边缩小、全图是原尺寸。样本由 sips 从 96×64 的 PNG 转出。
func TestSystemImageFormats(t *testing.T) {
	for _, ext := range []string{"psd", "jp2", "tga", "exr"} {
		path := filepath.Join("testdata", "image_sample."+ext)
		w, h, err := ImageSize(path)
		if err != nil || w != 96 || h != 64 {
			t.Fatalf("%s 尺寸不对：%dx%d %v", ext, w, h, err)
		}
		thumb, err := ImageThumbnail(path, 48)
		if err != nil || thumb.Bounds().Dx() != 48 || thumb.Bounds().Dy() != 32 {
			t.Fatalf("%s 缩略图不对：%v %v", ext, thumb, err)
		}
		full, err := DecodeImage(path)
		if err != nil || full.Bounds().Dx() != 96 || full.Bounds().Dy() != 64 {
			t.Fatalf("%s 全图不对：%v", ext, err)
		}
	}
}

// 不是图片的文件报 ErrUnsupported，调用方据此给出「读不了」。
func TestSystemImageRejectsGarbage(t *testing.T) {
	if _, _, err := ImageSize(filepath.Join("testdata", "image_garbage.psd")); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("乱码文件应报不支持：%v", err)
	}
	if _, err := ImageThumbnail(filepath.Join("testdata", "image_garbage.psd"), 64); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("乱码文件应报不支持：%v", err)
	}
}
