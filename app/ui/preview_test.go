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

func TestPreviewDropsResultsAfterSwitchingAssetOrLibrary(t *testing.T) {
	p := previewState{Open: true}
	bitmap := ui.NewBitmap(image.NewRGBA(image.Rect(0, 0, 4, 8)))
	p.target("first.momento", "one", 0)
	first := p.Generation
	p.Loading = true
	p.target("first.momento", "two", 0)
	p.finish(first, bitmap, nil)
	if p.Bitmap != nil || p.Loading {
		t.Fatal("旧素材的解码结果污染了新素材")
	}
	second := p.Generation
	p.target("second.momento", "two", 0)
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
	p.target("library.momento", "one", 0)
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
	p.target("", "", 0)
	if p.errorText != "" {
		t.Fatal("预览关闭后没有释放错误状态")
	}
}

func readyPreviewApp() *App {
	asset := model.Asset{ID: "one", Name: "Preview original", RelativePath: "assets/one.png", Width: 240, Height: 160}
	a := &App{Library: &library.Library{Root: "library.momento"}, Visible: []model.Asset{asset}, Preferences: DefaultPreferences()}
	w := &previewWindow{root: a.Library.Root, asset: asset, items: append([]model.Asset(nil), a.Visible...)}
	w.picture.Open = true
	w.picture.target(w.root, asset.ID, 0)
	w.picture.Bitmap = ui.NewBitmap(image.NewRGBA(image.Rect(0, 0, 240, 160)))
	a.Preview.active = w
	return a
}
func previewTestView(a *App) func(*ui.Context) {
	return func(c *ui.Context) {
		ui.Box(c).Fill().TabStop(true).AutoFocus().ObserveInput(a.ObservePreviewInput).Children(func() {
			if w := a.Preview.active; w != nil {
				w.view(c, a)
				// 测试里把信息卡叠在底部，同一个 Tester 里能点到翻页和关闭。
				ui.Box(c.Key("test-preview-info")).Absolute().Left(0).Right(0).Bottom(0).Height(previewInfoHeight).Children(func() { w.infoView(c, a) })
			}
		})
		a.ReconcilePreviewKeys(c)
	}
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
			tt := ui.NewTester(previewTestView(a), 1100, 640)
			tt.Advance(5 * time.Millisecond)
			p := &a.Preview.active.picture
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
	tt := ui.NewTester(previewTestView(a), 1000, 640)
	tt.Advance(5 * time.Millisecond)
	p := &a.Preview.active.picture
	cx, cy := p.viewport.X+p.viewport.W/2, p.viewport.Y+p.viewport.H/2
	tt.Pinch(cx, cy, 4)
	if p.Zoom != 4 {
		t.Fatalf("捏合比例%v", p.Zoom)
	}
	if _, found := tt.AccessByRole(ui.RoleButton, a.T("close_preview")); !found {
		t.Fatal("放大图片后关闭控制没有保留")
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
	a.Visible = append(a.Visible, model.Asset{ID: "two"})
	a.Preview.active.items = append([]model.Asset(nil), a.Visible...)
	a.Preferences.PreviewWheel = "zoom"
	tt := ui.NewTester(previewTestView(a), 1100, 640)
	tt.Advance(5 * time.Millisecond)
	p := &a.Preview.active.picture
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
	now := time.Now()
	a.Preview.active.input(a, ui.InputEvent{Kind: ui.InputScroll, DY: 40, Time: now})
	if a.Preview.active.asset.ID != "two" {
		t.Fatal("滚轮没有切到下一张")
	}
	a.Preview.active.input(a, ui.InputEvent{Kind: ui.InputScroll, DY: -40, Time: now.Add(time.Millisecond)})
	if a.Preview.active.asset.ID != "two" {
		t.Fatal("一次手势连续切换了素材")
	}
	a.Preview.active.input(a, ui.InputEvent{Kind: ui.InputScroll, DY: -40, Time: now.Add(time.Second)})
	if a.Preview.active.asset.ID != "one" {
		t.Fatal("滚轮没有切到上一张")
	}
}

func TestPreviewTargetResetsZoomAndIgnoresStaleLoads(t *testing.T) {
	p := previewState{Open: true}
	p.target("a.momento", "one", 0)
	old := p.Generation
	p.Zoom = 4
	p.Pan = ui.Point{X: 40, Y: 20}
	p.initialized = true
	p.target("a.momento", "two", 0)
	p.Loading = true
	p.finish(old, ui.NewBitmap(image.NewRGBA(image.Rect(0, 0, 4, 4))), nil)
	if !p.Loading || p.Bitmap != nil || p.initialized || p.Zoom != 1 || p.Pan != (ui.Point{}) {
		t.Fatal("旧解码干扰了新图片初始化")
	}
}

func TestPreviewGIFFramesLoopAndKeepSamplingVariants(t *testing.T) {
	path := writePreviewGIF(t)
	picture, err := readPreviewPicture(path, 0)
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
	p.target("library", "gif", 0)
	p.finishPicture(p.Generation, picture, nil)
	p.target("library", "another", 0)
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
	a.Preview.active.picture.Bitmap, a.Preview.active.picture.pixelBitmap = previewBitmaps(img)
	tt := ui.NewTester(previewTestView(a), 1100, 640)
	tt.Advance(5 * time.Millisecond)
	a.Preview.active.picture.Zoom = 4
	a.Preview.active.picture.initialized = true
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
	a.Visible[0].Width, a.Visible[0].Height = width, height
	a.Preview.active.asset = a.Visible[0]
	a.Preview.active.items = append([]model.Asset(nil), a.Visible...)
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8((x/9)%2*140 + 40), G: uint8((y/9)%2*100 + 70), B: uint8((x+y)%200 + 30), A: 255})
		}
	}
	a.Preview.active.picture.Bitmap, a.Preview.active.picture.pixelBitmap = previewBitmaps(img)
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
	tt := ui.NewTester(previewTestView(a), 1100, 640)
	tt.Advance(5 * time.Millisecond)
	tt.Key(ui.Cmd, ui.KeyEqual)
	if a.Preview.active.picture.Zoom != 1 {
		t.Fatal("改键后仍响应旧快捷键")
	}
	tt.Key(ui.Alt, ui.KeyZ)
	if a.Preview.active.picture.Zoom != 1.25 {
		t.Fatal("没有响应自定义快捷键")
	}
	a.Settings.RecordingShortcut = "preview_zoom_out"
	tt.Key(ui.Alt, ui.KeyZ)
	if a.Preview.active.picture.Zoom != 1.25 {
		t.Fatal("录制快捷键时仍操作预览")
	}
}
