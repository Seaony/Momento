package ui

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// 把图片卡和信息卡按真实布局画在一块模拟桌面上，检查两块卡片的整体效果。
func TestPreviewWindowScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_PREVIEW_WINDOW_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_PREVIEW_WINDOW_SCREENSHOTS=1 输出独立预览窗口截图")
	}
	dir := "/tmp/momento-polish/shots4/preview"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := newScreenshotFixture(t)
	for _, state := range []string{"normal", "hold", "pinned"} {
		for _, theme := range []string{"light", "dark"} {
			a := fixture.app(t, "small", "zh-CN", theme)
			var asset model.Asset
			for _, item := range a.Visible {
				if item.Extension == "jpg" {
					asset = item
					break
				}
			}
			asset.Name = "暖光下的山间晨雾.jpg"
			a.ShowPreview(asset.ID, false)
			w := a.Preview.active
			w.asset = asset
			picture, err := readPreviewPicture(filepath.Join(fixture.root, filepath.FromSlash(asset.RelativePath)), asset.Rotation)
			if err != nil {
				t.Fatal(err)
			}
			w.picture.finishPicture(w.picture.Generation, picture, nil)
			canvas := mygo.Rectangle{Width: 900, Height: 640}
			imageFrame, infoFrame := previewLayout(canvas, mygo.Rectangle{}, asset, a.Preferences.PreviewFit == "fit")
			imageTester := ui.NewTester(func(c *ui.Context) { w.view(c, a) }, imageFrame.Width, imageFrame.Height)
			infoTester := ui.NewTester(func(c *ui.Context) { w.infoView(c, a) }, infoFrame.Width, infoFrame.Height)
			for _, tt := range []*ui.Tester{imageTester, infoTester} {
				tt.SetLocale(a.Texts.Locale())
			}
			start := imageTester.Now()
			w.openedAt = start.Add(-time.Second)
			switch state {
			case "hold":
				w.picture.Held, w.pressedAt = true, start.Add(-1500*time.Millisecond)
				a.Preview.gesture = w
			case "pinned":
				w.pinned, w.pinnedAt = true, start.Add(-time.Second)
			}
			imageTester.Frame()
			infoTester.Frame()
			if state == "pinned" {
				imageTester.Move(float32(imageFrame.Width)/2, float32(imageFrame.Height)/2)
				imageTester.Advance(300 * time.Millisecond)
			}
			out := image.NewRGBA(image.Rect(0, 0, canvas.Width, canvas.Height))
			paintDesktop(out, theme == "dark")
			draw.Draw(out, image.Rect(imageFrame.X, imageFrame.Y, imageFrame.X+imageFrame.Width, imageFrame.Y+imageFrame.Height), imageTester.Image(), image.Point{}, draw.Over)
			if state != "pinned" {
				draw.Draw(out, image.Rect(infoFrame.X, infoFrame.Y, infoFrame.X+infoFrame.Width, infoFrame.Y+infoFrame.Height), infoTester.Image(), image.Point{}, draw.Over)
			}
			writePreviewWindowContact(t, filepath.Join(dir, state+"-"+theme+".png"), out)
		}
	}
}

// 模拟桌面：一张偏暖的渐变，用来看半透明卡片的观感。
func paintDesktop(img *image.RGBA, dark bool) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			t := float64(x+y) / float64(b.Dx()+b.Dy())
			c := color.RGBA{uint8(214 - 60*t), uint8(196 - 40*t), uint8(182 + 20*t), 255}
			if dark {
				c = color.RGBA{uint8(40 + 20*t), uint8(36 + 14*t), uint8(48 + 24*t), 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
}

func writePreviewWindowContact(t *testing.T, path string, img image.Image) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	err = png.Encode(file, img)
	closeErr := file.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
}
