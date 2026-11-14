package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestPreviewLoadsLibraryRotationWithoutChangingOriginal(t *testing.T) {
	path := writeDirectionalPreview(t, false)
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, rotation := range []int{0, 90, 180, 270} {
		picture, err := readPreviewPicture(path, rotation)
		if err != nil {
			t.Fatal(err)
		}
		checkPreviewRotation(t, picture, rotation, 0)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, after) {
		t.Fatal("预览旋转改动了原始文件")
	}
	if _, err = readPreviewPicture(path, 45); err == nil {
		t.Fatal("接受了非法旋转角度")
	}
}

func TestPreviewGIFRotatesEveryFrameAndLoops(t *testing.T) {
	for _, rotation := range []int{90, 180, 270} {
		picture, err := readPreviewPicture(writeDirectionalPreview(t, true), rotation)
		if err != nil {
			t.Fatal(err)
		}
		checkPreviewRotation(t, picture, rotation, 0)
		second, err := picture.animation.next()
		if err != nil {
			t.Fatal(err)
		}
		checkPreviewRotation(t, second, rotation, 1)
		loop, err := second.animation.next()
		if err != nil {
			t.Fatal(err)
		}
		checkPreviewRotation(t, loop, rotation, 0)
		if loop.animation.rotation != rotation {
			t.Fatal("GIF循环丢失了旋转角度")
		}
	}
}

func TestPreviewRotationChangeInvalidatesPendingDecode(t *testing.T) {
	p := previewState{Open: true}
	p.target("library", "one", 0)
	old := p.Generation
	p.Bitmap, p.pixelBitmap = previewBitmaps(image.NewRGBA(image.Rect(0, 0, 120, 80)))
	p.Zoom = 4
	p.initialized = true
	p.Loading = true
	p.target("library", "one", 90)
	p.finish(old, ui.NewBitmap(image.NewRGBA(image.Rect(0, 0, 120, 80))), nil)
	if p.Bitmap != nil || p.pixelBitmap != nil || p.Loading || p.initialized || p.Zoom != 1 {
		t.Fatal("旋转后沿用了旧方向的解码或视图状态")
	}
	if p.targetRotation != 90 {
		t.Fatal("没有记录当前方向")
	}
}

func TestPreviewRotationButtonsDisabledWhileBusy(t *testing.T) {
	a := previewFixture(288, 192)
	a.Busy = true
	tt := ui.NewTester(previewTestView(a), 1100, 640)
	tt.Advance(5 * time.Millisecond)
	for _, key := range []string{"preview_rotate_left", "preview_rotate_right"} {
		label := a.T(key)
		info, ok := tt.AccessByRole(ui.RoleButton, label)
		if !ok || !info.Disabled {
			t.Fatalf("旋转按钮没有按忙碌状态禁用：%s %+v", key, info)
		}
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
		if a.Preview.active.picture.targetRotation != 0 || a.Preview.active.picture.Bitmap == nil {
			t.Fatal("忙碌期间旋转按钮仍然操作了预览")
		}
	}
	a.Busy = false
	tt.Frame()
	for _, key := range []string{"preview_rotate_left", "preview_rotate_right"} {
		info, ok := tt.AccessByRole(ui.RoleButton, a.T(key))
		if !ok || info.Disabled {
			t.Fatalf("空闲后旋转按钮没有恢复：%s %+v", key, info)
		}
	}
}

func checkPreviewRotation(t *testing.T, picture previewPicture, rotation, frame int) {
	t.Helper()
	width, height := 120, 80
	if rotation == 90 || rotation == 270 {
		width, height = height, width
	}
	w, h := picture.bitmap.Size()
	if w != width || h != height {
		t.Fatalf("旋转%d°后尺寸%d×%d，预期%d×%d", rotation, w, h, width, height)
	}
	pw, ph := picture.pixelBitmap.Size()
	if pw != w || ph != h {
		t.Fatal("两种采样的旋转尺寸不同")
	}
	expected := map[int]int{0: 0, 90: 2, 180: 3, 270: 1}[rotation]
	expected = (expected + frame) % 4
	pixels := ui.NewTester(func(c *ui.Context) { ui.Image(c, picture.bitmap).Fill() }, w, h).Image()
	got := pixels.RGBAAt(8, 8)
	palette := previewDirectionalPalette()
	r, g, b, a := palette[expected].RGBA()
	if got != (color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}) {
		t.Fatalf("旋转%d°帧%d方向不对：左上角%+v，预期%v", rotation, frame, got, palette[expected])
	}
}

func previewDirectionalPalette() color.Palette {
	return color.Palette{color.NRGBA{R: 220, G: 40, B: 50, A: 255}, color.NRGBA{R: 40, G: 180, B: 90, A: 255}, color.NRGBA{R: 40, G: 100, B: 235, A: 255}, color.NRGBA{R: 240, G: 190, B: 40, A: 255}}
}

func writeDirectionalPreview(t *testing.T, animated bool) string {
	t.Helper()
	palette := previewDirectionalPalette()
	frames := []*image.Paletted{image.NewPaletted(image.Rect(0, 0, 120, 80), palette), image.NewPaletted(image.Rect(0, 0, 120, 80), palette)}
	for frame, picture := range frames {
		for y := 0; y < 80; y++ {
			for x := 0; x < 120; x++ {
				index := 0
				if x >= 60 {
					index++
				}
				if y >= 40 {
					index += 2
				}
				picture.SetColorIndex(x, y, uint8((index+frame)%4))
			}
		}
	}
	ext := ".png"
	if animated {
		ext = ".gif"
	}
	path := filepath.Join(t.TempDir(), "direction"+ext)
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if animated {
		err = gif.EncodeAll(file, &gif.GIF{Image: frames, Delay: []int{10, 10}, LoopCount: 0})
	} else {
		err = png.Encode(file, frames[0])
	}
	closeErr := file.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	return path
}
