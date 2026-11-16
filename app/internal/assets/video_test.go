package assets

import (
	"bytes"
	"context"
	"errors"
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/media"
	"github.com/Seaony/Momento/app/internal/model"
)

func mediaFixture(name string) string { return filepath.Join("..", "media", "testdata", name) }

// 导入视频：种类、时长、尺寸（按旋转转正）、MIME、缩略图都要对；视频不读关键词也不报错。
func TestImportVideosRecordsKindDurationAndPoster(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	paths := []string{mediaFixture("probe_h264_audio.mp4"), mediaFixture("probe_rot90.mov"), mediaFixture("probe_vp9.webm")}
	result, err := s.Import(ctx, model.ImportRequest{Paths: paths, ImportKeywords: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Imported != len(paths) || len(result.Errors) != 0 {
		t.Fatalf("导入结果不对：%+v", result)
	}
	byName := map[string]model.Asset{}
	for _, asset := range snapshot(t, ctx, lib).Assets {
		byName[asset.OriginalFileName] = asset
	}
	for _, path := range paths {
		asset, ok := byName[filepath.Base(path)]
		if !ok {
			t.Fatalf("%s 没有入库", path)
		}
		info, err := media.Probe(path)
		if err != nil {
			t.Fatal(err)
		}
		w, h := info.DisplaySize()
		if asset.Kind != model.KindVideo || asset.DurationMS != info.Duration.Milliseconds() || asset.DurationMS <= 0 || asset.Width != w || asset.Height != h {
			t.Fatalf("%s 的种类、时长或尺寸不对：%+v，期望 %dx%d %v", path, asset, w, h, info.Duration)
		}
		if asset.MediaType == "" || asset.MediaType[:6] != "video/" {
			t.Fatalf("%s 的 MIME 不对：%s", path, asset.MediaType)
		}
		thumbnailFile(t, lib.Root, asset.ID)
	}
	if got := byName["probe_rot90.mov"]; got.Width >= got.Height {
		t.Fatalf("旋转 90° 的视频应按竖向记尺寸：%dx%d", got.Width, got.Height)
	}
}

// 取不到封面（如 Linux 没装 ffmpeg）时照样导入：用纯色通用封面，不算颜色，元数据里记下来；文件坏了才算失败。
func TestImportVideoWithoutPosterUsesGenericCover(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	previous := videoPoster
	videoPoster = func(context.Context, string, time.Duration, int) (image.Image, error) {
		return nil, media.ErrPosterUnavailable
	}
	t.Cleanup(func() { videoPoster = previous })
	if _, err := s.Import(ctx, model.ImportRequest{Paths: []string{mediaFixture("probe_h264.mp4")}}, nil); err != nil {
		t.Fatal(err)
	}
	assets := snapshot(t, ctx, lib).Assets
	if len(assets) != 1 {
		t.Fatalf("没能导入：%d", len(assets))
	}
	asset := assets[0]
	exif, err := lib.AssetEXIF(ctx, asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	if exif["posterUnavailable"] != "1" || len(asset.Colors) != 0 {
		t.Fatalf("通用封面没有标记或算了颜色：%v %v", exif, asset.Colors)
	}
	thumbnailFile(t, lib.Root, asset.ID)

	videoPoster = func(context.Context, string, time.Duration, int) (image.Image, error) {
		return nil, errors.New("decoder exploded")
	}
	result, err := s.Import(ctx, model.ImportRequest{Paths: []string{mediaFixture("probe_vp8.webm")}}, nil)
	if err != nil || result.Imported != 0 || len(result.Errors) != 1 {
		t.Fatalf("封面截取出错应只让这一个失败：%+v %v", result, err)
	}
}

// 乱码的 .mp4 只让它自己失败，同批图片照常导入。
func TestImportBrokenVideoSkipsOnlyThatFile(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	result, err := s.Import(ctx, model.ImportRequest{Paths: []string{mediaFixture("poster_garbage.mp4"), filepath.Join("testdata", "sample.png")}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Imported != 1 || len(result.Errors) != 1 || len(snapshot(t, ctx, lib).Assets) != 1 {
		t.Fatalf("坏视频影响了别的文件：%+v", result)
	}
}

// 视频选了 JPEG/PNG 导出也原样导出原文件，扩展名不变。
func TestExportVideoAlwaysCopiesOriginal(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	source := mediaFixture("probe_h264.mp4")
	if _, err := s.Import(ctx, model.ImportRequest{Paths: []string{source}}, nil); err != nil {
		t.Fatal(err)
	}
	asset := snapshot(t, ctx, lib).Assets[0]
	dest := t.TempDir()
	paths, err := s.Export(ctx, []model.Asset{asset}, dest, model.ExportJPEG, .9)
	if err != nil || len(paths) != 1 || filepath.Ext(paths[0]) != ".mp4" {
		t.Fatalf("视频导出不对：%v %v", paths, err)
	}
	want, _ := os.ReadFile(source)
	if got, err := os.ReadFile(paths[0]); err != nil || !bytes.Equal(got, want) {
		t.Fatal("导出的视频不是原文件", err)
	}
}

// 单个视频导出到保存框选的路径：名字带着图片后缀时换成视频后缀，内容仍是原视频。
func TestExportSingleVideoKeepsVideoExtension(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	source := mediaFixture("probe_h264.mp4")
	if _, err := s.Import(ctx, model.ImportRequest{Paths: []string{source}}, nil); err != nil {
		t.Fatal(err)
	}
	asset := snapshot(t, ctx, lib).Assets[0]
	paths, err := s.Export(ctx, []model.Asset{asset}, filepath.Join(t.TempDir(), "clip.jpg"), model.ExportJPEG, .9)
	if err != nil || len(paths) != 1 || filepath.Base(paths[0]) != "clip.mp4" {
		t.Fatalf("单个视频导出的文件名不对：%v %v", paths, err)
	}
	want, _ := os.ReadFile(source)
	if got, err := os.ReadFile(paths[0]); err != nil || !bytes.Equal(got, want) {
		t.Fatal("导出的不是原视频", err)
	}
}
