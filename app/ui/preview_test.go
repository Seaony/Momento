package ui

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func TestPreviewFollowsVisibleOrderWithoutWrapping(t *testing.T) {
	a := &App{Visible: []model.Asset{{ID: "c"}, {ID: "a"}, {ID: "b"}}, Preview: previewState{ID: "a", Open: true}}
	a.movePreview(1)
	if a.Preview.ID != "b" {
		t.Fatalf("下一张 = %q", a.Preview.ID)
	}
	a.movePreview(1)
	if a.Preview.ID != "b" {
		t.Fatal("在列表末尾回到了开头")
	}
	a.movePreview(-1)
	a.movePreview(-1)
	a.movePreview(-1)
	if a.Preview.ID != "c" {
		t.Fatalf("上一张 = %q", a.Preview.ID)
	}
	a.Preview.ID = "filtered-out"
	a.movePreview(1)
	if a.Preview.ID != "filtered-out" {
		t.Fatal("跳到了当前结果以外的素材")
	}
}

func TestPreviewDropsResultsAfterSwitchingAssetOrLibrary(t *testing.T) {
	p := previewState{Open: true}
	bitmap := ui.NewBitmap(image.NewRGBA(image.Rect(0, 0, 4, 8)))
	p.target("first.momento", "one")
	first := p.Generation
	p.Loading = true
	p.target("first.momento", "two")
	p.finish(first, bitmap, nil)
	if p.Bitmap != nil || p.Loading {
		t.Fatal("旧素材的解码结果污染了新素材")
	}
	second := p.Generation
	p.target("second.momento", "two")
	p.finish(second, bitmap, nil)
	if p.Bitmap != nil {
		t.Fatal("另一个图库的图片污染了当前图库")
	}
	p.finish(p.Generation, bitmap, nil)
	if p.Bitmap != bitmap {
		t.Fatal("当前原图没有显示")
	}
}

func TestPreviewFailureVisibleAndCloseDropsPendingResult(t *testing.T) {
	p := previewState{Open: true}
	p.target("library.momento", "one")
	p.Loading = true
	p.finish(p.Generation, nil, errors.New("文件已移走"))
	if p.errorText != "文件已移走" || p.Loading {
		t.Fatal("原图加载失败没有留下可见错误")
	}
	p.Open = false
	p.finish(p.Generation, ui.NewBitmap(image.NewRGBA(image.Rect(0, 0, 4, 4))), nil)
	if p.Bitmap != nil {
		t.Fatal("关闭预览后接受了待处理图片")
	}
	p.target("", "")
	if p.errorText != "" {
		t.Fatal("预览关闭后没有释放错误状态")
	}
}

func TestPreviewEscapeClosesDialog(t *testing.T) {
	a := readyPreviewApp()
	tt := ui.NewTester(a.preview, 1100, 640)
	tt.Advance(5 * time.Millisecond)
	if !tt.HasText("Preview original") {
		if _, ok := tt.Find(a.T("close_preview")); !ok {
			t.Fatal("没有渲染预览浮层")
		}
	}
	tt.Key(0, ui.KeyEscape)
	if a.Preview.Open || a.Preview.Bitmap != nil {
		t.Fatal("Escape 没有关闭预览并释放原图")
	}
}

func TestPreviewSpaceReleaseClosesHeldPreview(t *testing.T) {
	a := readyPreviewApp()
	tt := ui.NewTester(a.preview, 1100, 640)
	tt.Advance(5 * time.Millisecond)
	tt.KeyDown(ui.KeySpace)
	a.Preview.Held = true
	tt.Frame()
	if !a.Preview.Open {
		t.Fatal("空格仍按住时预览已关闭")
	}
	tt.KeyRepeat(0, ui.KeySpace)
	if !a.Preview.Open {
		t.Fatal("空格自动重复触发了关闭按钮")
	}
	tt.KeyUp(ui.KeySpace)
	if a.Preview.Open || a.Preview.Bitmap != nil {
		t.Fatal("松开空格没有关闭预览并释放原图")
	}
}

func readyPreviewApp() *App {
	a := &App{
		Library: &library.Library{Root: "library.momento"},
		Visible: []model.Asset{{ID: "one", Name: "Preview original", RelativePath: "assets/one.png"}},
		Preview: previewState{ID: "one", Open: true},
	}
	a.Preview.target(a.Library.Root, a.Preview.ID)
	a.Preview.Bitmap = ui.NewBitmap(image.NewRGBA(image.Rect(0, 0, 240, 160)))
	return a
}

func TestPreviewSmartFitAndKeyboardZoom(t *testing.T) {
	for _, tc := range []struct {
		name, fit string
		w, h      int
		small     bool
	}{
		{"小图智能", "smart", 288, 192, true}, {"小图适合", "fit", 288, 192, true}, {"大图智能", "smart", 2400, 1600, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := previewFixture(tc.w, tc.h)
			a.Preferences.PreviewFit = tc.fit
			tt := ui.NewTester(a.preview, 1100, 640)
			tt.Advance(5 * time.Millisecond)
			p := &a.Preview
			if !p.initialized || p.fitZoom <= 0 {
				t.Fatal("预览没有按视口初始化")
			}
			expected := p.fitZoom
			if tc.fit == "smart" {
				expected = min(1, expected)
			}
			if p.Zoom != expected {
				t.Fatalf("初始比例%v，预期%v", p.Zoom, expected)
			}
			tt.Key(ui.Cmd, ui.Key1)
			if p.Zoom != 1 {
				t.Fatalf("原始大小比例%v", p.Zoom)
			}
			tt.Key(ui.Cmd, ui.KeyEqual)
			if p.Zoom != 1.25 {
				t.Fatalf("放大比例%v", p.Zoom)
			}
			tt.Key(ui.Cmd, ui.KeyMinus)
			if math.Abs(float64(p.Zoom-1)) > .001 {
				t.Fatalf("缩小比例%v", p.Zoom)
			}
			tt.Key(ui.Cmd, ui.Key0)
			if p.Zoom != p.fitZoom || p.Pan != (ui.Point{}) {
				t.Fatal("适合窗口没有复位")
			}
		})
	}
}

func TestPreviewPinchPanAndDoubleClick(t *testing.T) {
	a := previewFixture(288, 192)
	tt := ui.NewTester(a.preview, 1100, 640)
	tt.Advance(5 * time.Millisecond)
	p := &a.Preview
	cx, cy := p.viewport.X+p.viewport.W/2, p.viewport.Y+p.viewport.H/2
	beforeHeader := tt.Image().RGBAAt(100, 95)
	tt.Pinch(cx, cy, 4)
	if p.Zoom != 4 {
		t.Fatalf("捏合比例%v", p.Zoom)
	}
	if got := tt.Image().RGBAAt(100, 95); got != beforeHeader {
		t.Fatalf("放大图片越过视口遮住了顶部操作栏: %+v", got)
	}

	tt.Press(cx, cy)
	tt.Move(cx+70, cy+30)
	tt.Release(cx+70, cy+30)
	if p.Pan.X != 70 || p.Pan.Y != 30 {
		t.Fatalf("拖动平移%+v", p.Pan)
	}
	tt.Advance(time.Second)
	tt.ClickAt(cx, cy)
	tt.ClickAt(cx, cy)
	if p.Zoom != p.fitZoom || p.Pan != (ui.Point{}) {
		t.Fatalf("双击没有适合窗口：%v，%+v", p.Zoom, p.Pan)
	}
	tt.Advance(time.Second)
	tt.ClickAt(cx, cy)
	tt.ClickAt(cx, cy)
	if p.Zoom != 1 {
		t.Fatalf("第二次双击没有原始大小：%v", p.Zoom)
	}
}

func TestPreviewWheelPreferencesAndBounds(t *testing.T) {
	a := previewFixture(288, 192)
	a.Preferences.PreviewWheel = "zoom"
	tt := ui.NewTester(a.preview, 1100, 640)
	tt.Advance(5 * time.Millisecond)
	p := &a.Preview
	cx, cy := p.viewport.X+p.viewport.W/2, p.viewport.Y+p.viewport.H/2
	tt.Wheel(cx, cy, 0, -40)
	if p.Zoom <= 1 {
		t.Fatal("滚轮缩放没有生效")
	}
	for range 100 {
		a.PreviewZoomIn()
	}
	if p.Zoom != 32 {
		t.Fatalf("放大没有上限：%v", p.Zoom)
	}
	p.Pan = ui.Point{X: 100000, Y: -100000}
	p.clampPan()
	r := p.imageRect(p.viewport)
	if r.X > p.viewport.X || r.Y+r.H < p.viewport.Y+p.viewport.H {
		t.Fatal("平移将图像拖离视口")
	}
	a.PreviewActualSize()
	if p.Pan != (ui.Point{}) {
		t.Fatal("小图原始大小仍有平移")
	}
	a.Preferences.PreviewWheel = "navigate"
	a.Visible = append(a.Visible, model.Asset{ID: "two"})
	now := time.Now()
	a.previewInput(ui.InputEvent{Kind: ui.InputScroll, DY: 40, Time: now})
	if p.ID != "two" {
		t.Fatal("滚轮没有切到下一张")
	}
	a.previewInput(ui.InputEvent{Kind: ui.InputScroll, DY: -40, Time: now.Add(time.Millisecond)})
	if p.ID != "two" {
		t.Fatal("一次手势连续切换了素材")
	}
	a.previewInput(ui.InputEvent{Kind: ui.InputScroll, DY: -40, Time: now.Add(time.Second)})
	if p.ID != "one" {
		t.Fatal("滚轮没有切到上一张")
	}
}

func TestPreviewTargetResetsZoomAndIgnoresStaleLoads(t *testing.T) {
	p := previewState{Open: true}
	p.target("a.momento", "one")
	old := p.Generation
	p.Zoom = 4
	p.Pan = ui.Point{X: 40, Y: 20}
	p.initialized = true
	p.target("a.momento", "two")
	p.Loading = true
	p.finish(old, ui.NewBitmap(image.NewRGBA(image.Rect(0, 0, 4, 4))), nil)
	if !p.Loading || p.Bitmap != nil || p.initialized || p.Zoom != 1 || p.Pan != (ui.Point{}) {
		t.Fatal("旧解码干扰了新图片初始化")
	}
}

func TestPreviewGIFFramesLoopAndKeepSamplingVariants(t *testing.T) {
	path := writePreviewGIF(t)
	picture, err := readPreviewPicture(path)
	if err != nil {
		t.Fatal(err)
	}
	if picture.bitmap == nil || picture.pixelBitmap == nil || picture.animation == nil || picture.animation.delay != 100*time.Millisecond {
		t.Fatal("没有准备GIF首帧")
	}
	first := renderPreviewPicture(picture)
	second, err := picture.animation.next()
	if err != nil {
		t.Fatal(err)
	}
	secondImage := renderPreviewPicture(second)
	if bytes.Equal(first.Pix, secondImage.Pix) {
		t.Fatal("GIF没有切换帧")
	}
	loop, err := second.animation.next()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Pix, renderPreviewPicture(loop).Pix) {
		t.Fatal("GIF没有循环到首帧")
	}
	p := previewState{Open: true}
	p.target("library", "gif")
	p.finishPicture(p.Generation, picture, nil)
	p.target("library", "another")
	if p.animation != nil || p.pixelBitmap != nil {
		t.Fatal("切换素材没有释放动图")
	}
}

func TestPreviewCheckerAndSamplingChangeRenderedImage(t *testing.T) {
	a := previewFixture(288, 192)
	img := image.NewNRGBA(image.Rect(0, 0, 288, 192))
	for y := 48; y < 144; y++ {
		for x := 72; x < 216; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 240, A: 180})
		}
	}
	a.Preview.Bitmap, a.Preview.pixelBitmap = previewBitmaps(img)
	tt := ui.NewTester(a.preview, 1100, 640)
	tt.Advance(5 * time.Millisecond)
	a.Preview.Zoom = 4
	a.Preview.initialized = true
	tt.Frame()
	smooth := append([]byte(nil), tt.Image().Pix...)
	a.Preferences.PreviewPixelated = true
	tt.Frame()
	if bytes.Equal(smooth, tt.Image().Pix) {
		t.Fatal("像素化没有改变放大后的采样")
	}
	checker := append([]byte(nil), tt.Image().Pix...)
	a.Preferences.PreviewChecker = false
	tt.Frame()
	if bytes.Equal(checker, tt.Image().Pix) {
		t.Fatal("关闭棋盘格没有改变透明区域")
	}
}

func previewFixture(width, height int) *App {
	a := readyPreviewApp()
	a.Preferences = DefaultPreferences()
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8((x/9)%2*140 + 40), G: uint8((y/9)%2*100 + 70), B: uint8((x+y)%200 + 30), A: 255})
		}
	}
	a.Preview.Bitmap, a.Preview.pixelBitmap = previewBitmaps(img)
	return a
}

func writePreviewGIF(t *testing.T) string {
	t.Helper()
	palette := color.Palette{color.NRGBA{R: 220, G: 45, B: 60, A: 255}, color.NRGBA{R: 40, G: 100, B: 235, A: 255}}
	first := image.NewPaletted(image.Rect(0, 0, 288, 192), palette)
	second := image.NewPaletted(first.Rect, palette)
	for i := range second.Pix {
		second.Pix[i] = 1
	}
	path := filepath.Join(t.TempDir(), "two-frames.gif")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	err = gif.EncodeAll(file, &gif.GIF{Image: []*image.Paletted{first, second}, Delay: []int{10, 10}, LoopCount: 0})
	closeErr := file.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	return path
}

func renderPreviewPicture(picture previewPicture) *image.RGBA {
	return ui.NewTester(func(c *ui.Context) { ui.Image(c, picture.bitmap).Fill() }, 288, 192).Image()
}

func TestPreviewUsesCustomShortcutAndPausesWhileRecording(t *testing.T) {
	a := previewFixture(288, 192)
	a.Preferences.Shortcuts["preview_zoom_in"] = "Alt+Z"
	tt := ui.NewTester(a.preview, 1100, 640)
	tt.Advance(5 * time.Millisecond)
	tt.Key(ui.Cmd, ui.KeyEqual)
	if a.Preview.Zoom != 1 {
		t.Fatal("改键后仍响应旧快捷键")
	}
	tt.Key(ui.Alt, ui.KeyZ)
	if a.Preview.Zoom != 1.25 {
		t.Fatal("没有响应自定义快捷键")
	}
	a.Settings.RecordingShortcut = "preview_zoom_out"
	tt.Key(ui.Alt, ui.KeyZ)
	if a.Preview.Zoom != 1.25 {
		t.Fatal("录制快捷键时仍操作预览")
	}
}
