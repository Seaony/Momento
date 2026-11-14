package ui

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func previewLayoutFixture(width, height int, theme string) *App {
	a := previewFixture(width, height)
	a.Texts.Language, a.Preferences.Language, a.Preferences.Theme = "zh-CN", "zh-CN", theme
	w := a.Preview.active
	w.asset.Name = "暖光下的山间晨雾.jpg"
	previous, next := w.asset, w.asset
	previous.ID, next.ID = "previous", "next"
	w.items = []model.Asset{previous, w.asset, next}
	return a
}

func TestPreviewSmallLayoutKeepsNavigationOutsidePicture(t *testing.T) {
	for _, theme := range []string{"light", "dark"} {
		a := previewLayoutFixture(288, 160, theme)
		w := a.Preview.active
		frame := previewWindowBounds(mygo.Rectangle{Width: 1100, Height: 800}, mygo.Rectangle{Width: 1440, Height: 900}, w.asset, false, true)
		tt := ui.NewTester(func(c *ui.Context) { w.view(c, a) }, frame.Width, frame.Height)
		tt.Advance(300 * time.Millisecond)
		photo, ok := tt.AccessByRole(ui.RoleImage, w.asset.Name)
		if !ok {
			t.Fatal("缺少原图")
		}
		if photo.Bounds.X != 64 || photo.Bounds.Y != 72 || photo.Bounds.W != 288 || photo.Bounds.H != 160 {
			t.Fatalf("小图裁切或布局未与窗口同步：%+v", photo.Bounds)
		}
		left, ok := tt.AccessByRole(ui.RoleButton, a.T("previous_asset"))
		if !ok {
			t.Fatal("缺少上一张")
		}
		right, ok := tt.AccessByRole(ui.RoleButton, a.T("next_asset"))
		if !ok {
			t.Fatal("缺少下一张")
		}
		if left.Bounds.X != 16 || right.Bounds.X+right.Bounds.W != float32(frame.Width)-16 || left.Bounds.W != 36 || right.Bounds.W != 36 {
			t.Fatal("翻页按钮尺寸或边距改变")
		}
		if left.Bounds.X+left.Bounds.W+12 != photo.Bounds.X || photo.Bounds.X+photo.Bounds.W+12 != right.Bounds.X {
			t.Fatal("小图翻页按钮没有留在图外12间距")
		}
		if left.Bounds.Y+left.Bounds.H/2 != float32(frame.Height)/2 {
			t.Fatal("翻页按钮没有垂直居中")
		}
		rotation, ok := tt.AccessByRole(ui.RoleButton, a.T("preview_rotate_left"))
		if !ok {
			t.Fatal("缺少旋转按钮")
		}
		capsuleTop := rotation.Bounds.Y - (48-rotation.Bounds.H)/2
		if capsuleTop != 12 || photo.Bounds.Y-(capsuleTop+48) != 12 {
			t.Fatalf("顶部/胶囊到图片不是12：top=%v photo=%v", capsuleTop, photo.Bounds.Y)
		}
	}
}

func TestPreviewLargeAndNarrowLayoutsKeepWholePicture(t *testing.T) {
	for _, tc := range []struct {
		name                                     string
		width, height, windowWidth, windowHeight int
	}{
		{"大图", 1024, 768, 720, 640}, {"窄窗", 288, 160, 300, 220},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := previewLayoutFixture(tc.width, tc.height, "light")
			w := a.Preview.active
			tt := ui.NewTester(func(c *ui.Context) { w.view(c, a) }, tc.windowWidth, tc.windowHeight)
			tt.Advance(300 * time.Millisecond)
			photo, ok := tt.AccessByRole(ui.RoleImage, w.asset.Name)
			if !ok {
				t.Fatal("缺少原图")
			}
			if w.picture.viewport.X != 12 {
				t.Fatal("大图或窄窗仍强留图外按钮空间")
			}
			if math.Abs(float64(photo.Bounds.W/photo.Bounds.H)-float64(tc.width)/float64(tc.height)) > .001 {
				t.Fatal("布局改变了图片比例")
			}
			if photo.Bounds.X < 12 || photo.Bounds.Y < 72 || photo.Bounds.X+photo.Bounds.W > float32(tc.windowWidth)-12+.1 || photo.Bounds.Y+photo.Bounds.H > float32(tc.windowHeight)-48+.1 {
				t.Fatalf("图片被裁切：%+v", photo.Bounds)
			}
			for _, key := range []string{"previous_asset", "next_asset"} {
				button, ok := tt.AccessByRole(ui.RoleButton, a.T(key))
				if !ok || button.Bounds.W != 36 || button.Bounds.X < 0 || button.Bounds.X+button.Bounds.W > float32(tc.windowWidth) {
					t.Fatal("窄窗翻页按钮超出窗口")
				}
			}
		})
	}
}

func TestPreviewMinimumWidthKeepsLongNumericMetadata(t *testing.T) {
	a := previewFixture(4096, 48)
	w := a.Preview.active
	w.asset.Width, w.asset.Height = 4096, 2160
	w.items = make([]model.Asset, 12000)
	w.items[len(w.items)-1] = w.asset
	var measured float32
	text := "4096 × 2160 · 100% · 12000 / 12000"
	tt := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		w.picture.Zoom = 1
		w.controls(c, a, false)
		measured, _ = c.MeasureText(0, ui.Span{Text: text, Font: monoFont(c), Size: fontNumber})
	}, previewMinimumWidth, 220)
	bounds, ok := tt.Find(text)
	if !ok || measured > bounds.W+.1 {
		t.Fatalf("最小窗纯数字行截断：文字%.1f，可用%.1f", measured, bounds.W)
	}
}

func TestPreviewLayoutFixScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_PREVIEW_FIX_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_PREVIEW_FIX_SCREENSHOTS=1 输出预览留白截图")
	}
	dir := "/tmp/momento-polish/shots3/preview-fix"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, theme := range []string{"light", "dark"} {
		a := previewLayoutFixture(288, 160, theme)
		w := a.Preview.active
		frame := previewWindowBounds(mygo.Rectangle{Width: 1100, Height: 800}, mygo.Rectangle{Width: 1440, Height: 900}, w.asset, false, true)
		tt := ui.NewTester(func(c *ui.Context) { w.view(c, a) }, frame.Width, frame.Height)
		tt.SetLocale(a.Texts.Locale())
		tt.Advance(300 * time.Millisecond)
		writeScreenshot(t, filepath.Join(dir, "preview-small-"+theme+".png"), tt)
	}
}
