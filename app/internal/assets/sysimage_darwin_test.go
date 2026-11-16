//go:build darwin

package assets

import (
	"errors"
	"image"
	"testing"

	"github.com/Seaony/Momento/app/internal/model"
)

// macOS 上 PSD、JPEG 2000、TGA、EXR 走系统解码：真实尺寸、缩略图和导出 PNG 都要对。
func TestImportSystemFormatsOnMac(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	var paths []string
	for _, ext := range []string{"psd", "jp2", "tga", "exr"} {
		paths = append(paths, mediaFixture("image_sample."+ext))
	}
	result, err := s.Import(ctx, model.ImportRequest{Paths: paths}, nil)
	if err != nil || result.Imported != 4 || len(result.Errors) != 0 {
		t.Fatalf("导入结果不对：%+v %v", result, err)
	}
	for _, asset := range snapshot(t, ctx, lib).Assets {
		if asset.Width != 96 || asset.Height != 64 || asset.Kind != model.KindImage {
			t.Fatalf("%s 尺寸不对：%+v", asset.Extension, asset)
		}
		thumbnailFile(t, lib.Root, asset.ID)
		written, err := s.Export(ctx, []model.Asset{asset}, t.TempDir(), model.ExportPNG, .9)
		if err != nil || len(written) != 1 {
			t.Fatalf("%s 导出 PNG 失败：%v", asset.Extension, err)
		}
	}
}

// macOS 上系统解不了的 HEIC（比如老系统不认的编码）退回内置解码器，不算失败。
func TestHEICFallsBackToBuiltInDecoder(t *testing.T) {
	previous := systemImageThumbnail
	systemImageThumbnail = func(string, int) (image.Image, error) { return nil, errors.New("not supported by this macOS") }
	t.Cleanup(func() { systemImageThumbnail = previous })
	prepared, err := prepareThumbnail(t.Context(), "testdata/sample.heic")
	if err != nil || prepared.thumbnail == nil || prepared.width <= 0 {
		t.Fatalf("没有退回内置解码器：%v", err)
	}
}
