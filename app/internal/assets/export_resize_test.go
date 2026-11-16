package assets

import (
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
)

// 转格式导出时按最长边缩小、保持比例，先转向再缩；本来就小的不放大；原格式不能改尺寸，要报错而不是悄悄导原图。
func TestExportMaxEdge(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	dir := t.TempDir()
	write := func(name string, w, h int) string {
		path := filepath.Join(dir, name)
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := jpeg.Encode(f, gradientImage(w, h, w < 500), nil); err != nil {
			t.Fatal(err)
		}
		return path
	}
	if _, err := s.Import(ctx, model.ImportRequest{Paths: []string{write("big.jpg", 1200, 900), write("small.jpg", 300, 200)}}, nil); err != nil {
		t.Fatal(err)
	}
	snapshot, err := lib.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]model.Asset{}
	for _, asset := range snapshot.Assets {
		byName[asset.Name] = asset
	}
	big, small := byName["big"], byName["small"]
	size := func(path string) (int, int) {
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		config, _, err := image.DecodeConfig(f)
		if err != nil {
			t.Fatal(err)
		}
		return config.Width, config.Height
	}
	for _, format := range []model.ExportFormat{model.ExportJPEG, model.ExportPNG} {
		paths, err := s.ExportWithOptions(ctx, []model.Asset{big, small}, t.TempDir(), model.ExportOptions{Format: format, MaxEdge: 600})
		if err != nil {
			t.Fatal(err)
		}
		if w, h := size(paths[0]); w != 600 || h != 450 {
			t.Fatalf("%s 大图应缩到 600×450，实际 %d×%d", format, w, h)
		}
		if w, h := size(paths[1]); w != 300 || h != 200 {
			t.Fatalf("%s 小图不该放大，实际 %d×%d", format, w, h)
		}
	}
	rotated := big
	rotated.Rotation = 90
	paths, err := s.ExportWithOptions(ctx, []model.Asset{rotated}, t.TempDir(), model.ExportOptions{Format: model.ExportPNG, MaxEdge: 400})
	if err != nil {
		t.Fatal(err)
	}
	if w, h := size(paths[0]); w != 300 || h != 400 {
		t.Fatalf("转过 90° 的图应先转向再缩到 300×400，实际 %d×%d", w, h)
	}
	for _, options := range []model.ExportOptions{{Format: model.ExportOriginal, MaxEdge: 600}, {Format: model.ExportJPEG, MaxEdge: -1}} {
		_, err := s.ExportWithOptions(ctx, []model.Asset{big}, t.TempDir(), options)
		var localized *i18n.LocalizedError
		if !errors.As(err, &localized) || localized.Key != "error_export_max_edge" {
			t.Fatalf("%+v 应报 error_export_max_edge，实际 %v", options, err)
		}
	}
}

// 缩小时半透明像素的颜色不能被压糙，16 位的图缩完还是 16 位。
func TestFitLongestEdgeKeepsPrecision(t *testing.T) {
	translucent := image.NewNRGBA(image.Rect(0, 0, 100, 80))
	deep := image.NewNRGBA64(image.Rect(0, 0, 100, 80))
	for y := range 80 {
		for x := range 100 {
			translucent.SetNRGBA(x, y, color.NRGBA{200, 100, 50, 10})
			deep.SetNRGBA64(x, y, color.NRGBA64{51234, 1234, 60001, 65535})
		}
	}
	small := fitLongestEdge(translucent, 50)
	if got := color.NRGBAModel.Convert(small.At(10, 10)).(color.NRGBA); got != (color.NRGBA{200, 100, 50, 10}) {
		t.Fatalf("半透明像素颜色变了：%v", got)
	}
	if small.Bounds().Dx() != 50 || small.Bounds().Dy() != 40 {
		t.Fatalf("尺寸不对：%v", small.Bounds())
	}
	if got := color.NRGBA64Model.Convert(fitLongestEdge(deep, 50).At(10, 10)).(color.NRGBA64); got != (color.NRGBA64{51234, 1234, 60001, 65535}) {
		t.Fatalf("16 位颜色被压成 8 位：%v", got)
	}
}
