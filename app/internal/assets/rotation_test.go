package assets

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/gen2brain/avif"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
)

func rotationPattern() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(2, 3, 5, 5))
	colors := []color.NRGBA{{R: 255, A: 255}, {G: 255, A: 255}, {B: 255, A: 255}, {R: 255, G: 255, A: 255}, {G: 255, B: 255, A: 255}, {R: 255, B: 255, A: 255}}
	for i, value := range colors {
		img.SetNRGBA(2+i%3, 3+i/3, value)
	}
	return img
}

func equalImage(t *testing.T, actual, expected image.Image) {
	t.Helper()
	if actual.Bounds().Size() != expected.Bounds().Size() {
		t.Fatalf("旋转尺寸错误: %v want %v", actual.Bounds(), expected.Bounds())
	}
	for y := range actual.Bounds().Dy() {
		for x := range actual.Bounds().Dx() {
			got := color.NRGBA64Model.Convert(actual.At(actual.Bounds().Min.X+x, actual.Bounds().Min.Y+y))
			want := color.NRGBA64Model.Convert(expected.At(expected.Bounds().Min.X+x, expected.Bounds().Min.Y+y))
			if got != want {
				t.Fatalf("旋转像素错误 %d,%d: %v want %v", x, y, got, want)
			}
		}
	}
}

func TestRotateImageUsesClockwiseQuarterTurnsAndKeepsSource(t *testing.T) {
	src := rotationPattern()
	before := append([]byte{}, src.Pix...)
	orders := map[int][]int{0: {0, 1, 2, 3, 4, 5}, 90: {3, 0, 4, 1, 5, 2}, 180: {5, 4, 3, 2, 1, 0}, 270: {2, 5, 1, 4, 0, 3}}
	for rotation, order := range orders {
		img, err := RotateImage(src, rotation)
		if err != nil {
			t.Fatal(err)
		}
		size := image.Pt(3, 2)
		if rotation == 90 || rotation == 270 {
			size = image.Pt(2, 3)
		}
		if img.Bounds().Size() != size {
			t.Fatal(img.Bounds())
		}
		for i, index := range order {
			got := img.At(img.Bounds().Min.X+i%size.X, img.Bounds().Min.Y+i/size.X)
			want := src.At(2+index%3, 3+index/3)
			if color.NRGBAModel.Convert(got) != color.NRGBAModel.Convert(want) {
				t.Fatalf("方向错误 %d index %d", rotation, i)
			}
		}
	}
	negative, err := RotateImage(src, -90)
	if err != nil {
		t.Fatal(err)
	}
	positive, _ := RotateImage(src, 270)
	equalImage(t, negative, positive)
	if _, err := RotateImage(src, 45); err == nil {
		t.Fatal("接受了非90度旋转")
	}
	if !bytes.Equal(src.Pix, before) {
		t.Fatal("旋转修改了原图")
	}
	high := image.NewNRGBA64(image.Rect(0, 0, 2, 1))
	high.SetNRGBA64(0, 0, color.NRGBA64{R: 12345, G: 23456, B: 34567, A: 65535})
	turned, err := RotateImage(high, 90)
	if err != nil {
		t.Fatal(err)
	}
	if turned.At(0, 0) != high.At(0, 0) {
		t.Fatal("旋转丢失16位颜色精度")
	}
}

func TestRotationRefreshesThumbnailAndExportsWithoutChangingOriginal(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	source := filepath.Join(t.TempDir(), "pattern.png")
	f, err := os.Create(source)
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(f, rotationPattern()); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Import(ctx, model.ImportRequest{Paths: []string{source}}, nil); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, rotation := range []int{0, 90, 180, 270} {
		asset := snapshot(t, ctx, lib).Assets[0]
		if err := lib.SetRotation(ctx, []string{asset.ID}, rotation); err != nil {
			t.Fatal(err)
		}
		asset = snapshot(t, ctx, lib).Assets[0]
		if err := s.RefreshThumbnails(ctx, []model.Asset{asset}); err != nil {
			t.Fatal(err)
		}
		expected, err := RotateImage(rotationPattern(), rotation)
		if err != nil {
			t.Fatal(err)
		}
		thumbnail, err := Decode(filepath.Join(lib.Root, "thumbnails", asset.ID+".png"))
		if err != nil {
			t.Fatal(err)
		}
		equalImage(t, thumbnail, expected)
		for _, format := range []model.ExportFormat{model.ExportOriginal, model.ExportPNG} {
			paths, err := s.Export(ctx, []model.Asset{asset}, t.TempDir(), format, 0)
			if err != nil {
				t.Fatal(err)
			}
			actual, err := Decode(paths[0])
			if err != nil {
				t.Fatal(err)
			}
			equalImage(t, actual, expected)
			if rotation == 0 && format == model.ExportOriginal {
				data, err := os.ReadFile(paths[0])
				if err != nil || !bytes.Equal(data, original) {
					t.Fatal("未旋转原格式不再保持原文件")
				}
			}
		}
		stored, err := os.ReadFile(filepath.Join(lib.Root, filepath.FromSlash(asset.RelativePath)))
		if err != nil || !bytes.Equal(stored, original) {
			t.Fatal("旋转改动了图库原文件")
		}
	}
}

func TestRotatedOriginalPreservesWritableFormats(t *testing.T) {
	for _, extension := range []string{"jpg", "png", "bmp", "tiff", "avif"} {
		t.Run(extension, func(t *testing.T) {
			ctx, lib, s := testLibrary(t)
			path := filepath.Join(t.TempDir(), "pattern."+extension)
			f, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			source := rotationPattern()
			switch extension {
			case "jpg":
				err = jpeg.Encode(f, source, &jpeg.Options{Quality: 100})
			case "png":
				err = png.Encode(f, source)
			case "bmp":
				err = bmp.Encode(f, source)
			case "tiff":
				err = tiff.Encode(f, source, nil)
			case "avif":
				err = avif.Encode(f, source, avif.Options{Lossless: true, Speed: 10})
			}
			if err != nil {
				f.Close()
				t.Fatal(err)
			}
			if err = f.Close(); err != nil {
				t.Fatal(err)
			}
			if _, err = s.Import(ctx, model.ImportRequest{Paths: []string{path}}, nil); err != nil {
				t.Fatal(err)
			}
			asset := snapshot(t, ctx, lib).Assets[0]
			asset.Rotation = 90
			paths, err := s.Export(ctx, []model.Asset{asset}, t.TempDir(), model.ExportOriginal, 1)
			if err != nil {
				t.Fatal(err)
			}
			if filepath.Ext(paths[0]) != "."+extension {
				t.Fatalf("原格式被更换: %s", paths[0])
			}
			actual, err := Decode(paths[0])
			if err != nil {
				t.Fatal(err)
			}
			original, err := Decode(path)
			if err != nil {
				t.Fatal(err)
			}
			expected, err := RotateImage(original, 90)
			if err != nil {
				t.Fatal(err)
			}
			if extension == "jpg" {
				if actual.Bounds().Size() != expected.Bounds().Size() {
					t.Fatal("JPEG旋转尺寸错误")
				}
			} else {
				equalImage(t, actual, expected)
			}
		})
	}
}

func TestRotatedOriginalGIFPreservesFramesOffsetsAndTiming(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	palette := color.Palette{color.Transparent, color.RGBA{R: 255, A: 255}, color.RGBA{G: 255, A: 255}, color.RGBA{B: 255, A: 255}}
	first := image.NewPaletted(image.Rect(0, 0, 4, 3), palette)
	first.SetColorIndex(0, 0, 1)
	second := image.NewPaletted(image.Rect(1, 1, 4, 2), palette)
	second.SetColorIndex(1, 1, 2)
	second.SetColorIndex(3, 1, 3)
	animation := &gif.GIF{Image: []*image.Paletted{first, second}, Delay: []int{3, 7}, Disposal: []byte{gif.DisposalBackground, gif.DisposalPrevious}, LoopCount: 4, Config: image.Config{ColorModel: palette, Width: 4, Height: 3}, BackgroundIndex: 0}
	source := filepath.Join(t.TempDir(), "animated.gif")
	f, err := os.Create(source)
	if err != nil {
		t.Fatal(err)
	}
	if err = gif.EncodeAll(f, animation); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Import(ctx, model.ImportRequest{Paths: []string{source}}, nil); err != nil {
		t.Fatal(err)
	}
	asset := snapshot(t, ctx, lib).Assets[0]
	asset.Rotation = 90
	paths, err := s.Export(ctx, []model.Asset{asset}, t.TempDir(), model.ExportOriginal, 0)
	if err != nil {
		t.Fatal(err)
	}
	f, err = os.Open(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	got, err := gif.DecodeAll(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Image) != 2 || got.Config.Width != 3 || got.Config.Height != 4 || got.LoopCount != 4 || !reflect.DeepEqual(got.Delay, animation.Delay) || !reflect.DeepEqual(got.Disposal, animation.Disposal) {
		t.Fatalf("GIF动画信息丢失: %+v", got)
	}
	if got.Image[0].ColorIndexAt(2, 0) != 1 || got.Image[1].Bounds() != image.Rect(1, 1, 2, 4) || got.Image[1].ColorIndexAt(1, 1) != 2 || got.Image[1].ColorIndexAt(1, 3) != 3 {
		t.Fatal("GIF帧坐标或颜色错误")
	}
}

func TestRotatedAnimatedAVIFFFailsWithoutDroppingAnimation(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	source := filepath.Join(t.TempDir(), "animated.avif")
	first := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	first.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	second := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	second.SetNRGBA(2, 1, color.NRGBA{G: 255, A: 255})
	animation := &avif.AVIF{Image: []image.Image{first, second}, Delay: []float64{.1, .2}, LoopCount: 3}
	f, err := os.Create(source)
	if err != nil {
		t.Fatal(err)
	}
	if err = avif.EncodeAll(f, animation, avif.Options{Lossless: true, Speed: 10}); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Import(ctx, model.ImportRequest{Paths: []string{source}}, nil); err != nil {
		t.Fatal(err)
	}
	asset := snapshot(t, ctx, lib).Assets[0]
	asset.Rotation = 90
	destination := t.TempDir()
	paths, err := s.Export(ctx, []model.Asset{asset}, destination, model.ExportOriginal, 0)
	if err == nil || len(paths) != 0 {
		t.Fatal("有限动画被悄悄重编码")
	}
	entries, err := os.ReadDir(destination)
	if err != nil || len(entries) != 0 {
		t.Fatal("失败留下输出")
	}
	stored, err := os.ReadFile(filepath.Join(lib.Root, filepath.FromSlash(asset.RelativePath)))
	if err != nil || !bytes.Equal(stored, original) {
		t.Fatal("失败改动了原动画")
	}
	asset.Rotation = 0
	paths, err = s.Export(ctx, []model.Asset{asset}, destination, model.ExportOriginal, 0)
	if err != nil || len(paths) != 1 {
		t.Fatal("未旋转动画无法原样导出")
	}
	exported, err := os.ReadFile(paths[0])
	if err != nil || !bytes.Equal(exported, original) {
		t.Fatal("原样导出改变了动画")
	}
}

func TestRotatedUnwritableOriginalFailsVisiblyAndKeepsSource(t *testing.T) {
	for _, extension := range []string{"webp", "heic"} {
		t.Run(extension, func(t *testing.T) {
			ctx, lib, s := testLibrary(t)
			importFixture(t, ctx, s, extension)
			asset := snapshot(t, ctx, lib).Assets[0]
			asset.Rotation = 90
			destination := t.TempDir()
			paths, err := s.Export(ctx, []model.Asset{asset}, destination, model.ExportOriginal, 0)
			if err == nil || len(paths) != 0 {
				t.Fatal("不支持的旋转悄悄成功")
			}
			entries, err := os.ReadDir(destination)
			if err != nil || len(entries) != 0 {
				t.Fatal("失败留下输出文件")
			}
			paths, err = s.Export(ctx, []model.Asset{asset}, destination, model.ExportPNG, 0)
			if err != nil || len(paths) != 1 {
				t.Fatalf("无法显式转PNG: %v", err)
			}
			actual, err := Decode(paths[0])
			if err != nil {
				t.Fatal(err)
			}
			if actual.Bounds().Dx() != asset.Height || actual.Bounds().Dy() != asset.Width {
				t.Fatal("PNG方向错误")
			}
		})
	}
}

func TestRotatedMultiPageTIFFFailsInsteadOfDroppingPages(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	var buffer bytes.Buffer
	if err := tiff.Encode(&buffer, rotationPattern(), nil); err != nil {
		t.Fatal(err)
	}
	data := buffer.Bytes()
	var order binary.ByteOrder = binary.LittleEndian
	if string(data[:2]) == "MM" {
		order = binary.BigEndian
	}
	root := int(order.Uint32(data[4:8]))
	count := int(order.Uint16(data[root : root+2]))
	order.PutUint32(data[root+2+count*12:], uint32(len(data)))
	data = append(data, make([]byte, 6)...)
	source := filepath.Join(t.TempDir(), "multipage.tiff")
	if err := os.WriteFile(source, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Import(ctx, model.ImportRequest{Paths: []string{source}}, nil); err != nil {
		t.Fatal(err)
	}
	asset := snapshot(t, ctx, lib).Assets[0]
	asset.Rotation = 90
	destination := t.TempDir()
	paths, err := s.Export(ctx, []model.Asset{asset}, destination, model.ExportOriginal, 0)
	if err == nil || len(paths) != 0 {
		t.Fatal("多页TIFF悄悄变为单页")
	}
	entries, err := os.ReadDir(destination)
	if err != nil || len(entries) != 0 {
		t.Fatal("失败留下输出文件")
	}
}

func TestExportComposesFileOrientationWithLibraryRotationOnce(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, rotationPattern()); err != nil {
		t.Fatal(err)
	}
	data := encoded.Bytes()
	metadata := imageTestTIFF()
	chunk := make([]byte, len(metadata)+12)
	binary.BigEndian.PutUint32(chunk, uint32(len(metadata)))
	copy(chunk[4:], "eXIf")
	copy(chunk[8:], metadata)
	binary.BigEndian.PutUint32(chunk[len(chunk)-4:], crc32.ChecksumIEEE(chunk[4:len(chunk)-4]))
	data = append(append(append([]byte{}, data[:33]...), chunk...), data[33:]...)
	source := filepath.Join(t.TempDir(), "oriented.png")
	if err := os.WriteFile(source, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Import(ctx, model.ImportRequest{Paths: []string{source}}, nil); err != nil {
		t.Fatal(err)
	}
	asset := snapshot(t, ctx, lib).Assets[0]
	if asset.Orientation != 6 {
		t.Fatal("测试源文件没有6号方向")
	}
	asset.Rotation = 90
	expected, err := RotateImage(rotationPattern(), 180)
	if err != nil {
		t.Fatal(err)
	}
	for _, format := range []model.ExportFormat{model.ExportOriginal, model.ExportPNG} {
		paths, err := s.Export(ctx, []model.Asset{asset}, t.TempDir(), format, 0)
		if err != nil {
			t.Fatal(err)
		}
		img, err := Decode(paths[0])
		if err != nil {
			t.Fatal(err)
		}
		equalImage(t, img, expected)
	}
}
