package assets

import (
	"context"
	"errors"
	"image"
	"testing"
	"testing/synctest"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/media"
	"github.com/Seaony/Momento/app/internal/model"
)

func TestSystemFullDecodeSharesBudgetAndCancelsWhileWaiting(t *testing.T) {
	previousSize, previousDecode := systemImageSize, systemDecodeImage
	t.Cleanup(func() { systemImageSize, systemDecodeImage = previousSize, previousDecode })
	systemImageSize = func(string) (int, int, error) { return 6000, 4000, nil }
	systemDecodeImage = func(string) (image.Image, error) {
		if want := int64(6000 * 4000 * 6); decodeBudget.used != want {
			t.Errorf("系统解码占用 %d 字节，预期 %d 字节", decodeBudget.used, want)
		}
		return nil, media.ErrUnsupported
	}
	path := mediaFixture("image_sample.psd")
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		release, err := decodeBudget.reserve(ctx, decodeBudget.limit)
		if err != nil {
			t.Fatal(err)
		}
		defer release()
		result := make(chan error, 1)
		go func() {
			_, _, _, _, err := prepareImage(ctx, path)
			result <- err
		}()
		synctest.Wait()
		cancel()
		if err := <-result; !errors.Is(err, context.Canceled) {
			t.Fatalf("等待额度时取消，得到 %v", err)
		}
	})
	if _, _, _, _, err := PrepareImage(path); !errors.Is(err, media.ErrUnsupported) {
		t.Fatalf("系统解码失败原因丢失：%v", err)
	}
	if decodeBudget.used != 0 {
		t.Fatalf("系统解码失败后还有 %d 字节未归还", decodeBudget.used)
	}
}

// 系统给不了缩略图（Windows 没装解码器、Linux）：照样导入原文件，用印着格式名的通用封面，宽高记 0，不算颜色。
func TestImportSystemFormatWithoutThumbnailUsesFormatPoster(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	previousSize, previousThumb := systemImageSize, systemImageThumbnail
	systemImageSize = func(string) (int, int, error) { return 0, 0, media.ErrUnsupported }
	systemImageThumbnail = func(string, int) (image.Image, error) { return nil, media.ErrPosterUnavailable }
	t.Cleanup(func() { systemImageSize, systemImageThumbnail = previousSize, previousThumb })
	result, err := s.Import(ctx, model.ImportRequest{Paths: []string{mediaFixture("image_sample.psd")}}, nil)
	if err != nil || result.Imported != 1 {
		t.Fatalf("没能导入：%+v %v", result, err)
	}
	asset := snapshot(t, ctx, lib).Assets[0]
	exif, err := lib.AssetEXIF(ctx, asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	if asset.Width != 0 || asset.Height != 0 || len(asset.Colors) != 0 || exif["posterUnavailable"] != "1" || asset.MediaType != "image/vnd.adobe.photoshop" {
		t.Fatalf("通用封面的素材不对：%+v %v", asset, exif)
	}
	thumbnailFile(t, lib.Root, asset.ID)
}

// 系统解码报错（文件坏了）只让这一张失败。
func TestImportBrokenSystemFormatFailsOnlyThatFile(t *testing.T) {
	ctx, _, s := testLibrary(t)
	previous := systemImageThumbnail
	systemImageThumbnail = func(string, int) (image.Image, error) { return nil, errors.New("decoder exploded") }
	t.Cleanup(func() { systemImageThumbnail = previous })
	result, err := s.Import(ctx, model.ImportRequest{Paths: []string{mediaFixture("image_sample.psd"), "testdata/sample.png"}}, nil)
	if err != nil || result.Imported != 1 || len(result.Errors) != 1 {
		t.Fatalf("坏文件影响了别的：%+v %v", result, err)
	}
}

// 通用封面上要真的印出字，不是一块纯色。
func TestFormatPosterHasLabel(t *testing.T) {
	poster := formatPoster("psd").(*image.RGBA)
	colors := map[[4]uint8]bool{}
	for i := 0; i < len(poster.Pix); i += 4 {
		colors[[4]uint8(poster.Pix[i:i+4])] = true
	}
	if len(colors) < 2 || poster.Bounds().Dx() != thumbnailSize {
		t.Fatalf("通用封面没有印格式名：%d 种颜色", len(colors))
	}
}

// 解不了原图的电脑（Windows、Linux）：转格式导出明确报错，不拿缩略图冒充；预览退而用系统缩略图。
func TestSystemFormatWithoutFullDecode(t *testing.T) {
	previousDecode, previousPreview := systemDecodeImage, systemPreviewImage
	systemDecodeImage = func(string) (image.Image, error) { return nil, media.ErrUnsupported }
	preview := image.NewRGBA(image.Rect(0, 0, 40, 30))
	systemPreviewImage = func(string) (image.Image, error) {
		if decodeBudget.used == 0 {
			t.Error("系统预览解码时没有占用预算")
		}
		return preview, nil
	}
	t.Cleanup(func() { systemDecodeImage, systemPreviewImage = previousDecode, previousPreview })
	path := mediaFixture("image_sample.psd")
	_, _, _, _, err := PrepareImage(path)
	var localized *i18n.LocalizedError
	if !errors.As(err, &localized) || localized.Key != "error_system_decode_unavailable" {
		t.Fatalf("应明确报解不了原图：%v", err)
	}
	if got, err := PreviewImage(path); err != nil || got != preview {
		t.Fatalf("预览没有退而用系统缩略图：%v", err)
	}
	if decodeBudget.used != 0 {
		t.Fatalf("系统预览结束后还有 %d 字节未归还", decodeBudget.used)
	}
}
