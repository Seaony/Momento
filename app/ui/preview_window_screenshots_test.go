package ui

import (
	"image"
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

func TestPreviewWindowScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_PREVIEW_WINDOW_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_PREVIEW_WINDOW_SCREENSHOTS=1 输出独立预览窗口截图")
	}
	dir := "/tmp/momento-polish/shots2/preview-window"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := newScreenshotFixture(t)
	makeApp := func(theme string) *App {
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
		return a
	}
	for _, theme := range []string{"light", "dark"} {
		a := makeApp(theme)
		w := a.Preview.active
		w.openedAt = time.Now().Add(-time.Second)
		r := previewWindowBounds(mygo.Rectangle{Width: 1100, Height: 800}, mygo.Rectangle{Width: 1440, Height: 900}, w.asset, a.Preferences.PreviewFit == "fit", len(w.items) > 1)
		tt := ui.NewTester(func(c *ui.Context) { w.view(c, a) }, r.Width, r.Height)
		tt.SetLocale(a.Texts.Locale())
		tt.Advance(300 * time.Millisecond)
		writeScreenshot(t, filepath.Join(dir, "preview-"+theme+".png"), tt)
	}
	for _, kind := range []string{"quick", "peek", "pin"} {
		a := makeApp("light")
		w := a.Preview.active
		r := previewWindowBounds(mygo.Rectangle{Width: 1100, Height: 800}, mygo.Rectangle{Width: 1440, Height: 900}, w.asset, a.Preferences.PreviewFit == "fit", len(w.items) > 1)
		tt := ui.NewTester(func(c *ui.Context) { w.view(c, a) }, r.Width, r.Height)
		start := tt.Now()
		w.openedAt, w.pressedAt = start, start
		w.picture.Held = true
		a.Preview.gesture = w
		duration := previewWindowDuration
		if kind == "peek" {
			w.openedAt = start.Add(-time.Second)
			w.pressedAt = start.Add(-1200 * time.Millisecond)
			w.closeScale, w.closeOpacity = w.visual(start)
			w.closedHeld, w.closedProgress, w.closedHintOpacity = true, w.holdProgress(start), 1
			w.closing, w.closedAt = true, start
			w.picture.Held = false
		}
		if kind == "pin" {
			duration = previewPinThreshold + previewWindowDuration
		}
		var frames []*image.RGBA
		for _, fraction := range []float64{0, .35, .70, 1} {
			target := start.Add(time.Duration(float64(duration) * fraction))
			if kind == "quick" && w.picture.Held && target.Sub(start) >= 100*time.Millisecond {
				releaseAt := start.Add(100 * time.Millisecond)
				tt.Advance(releaseAt.Sub(tt.Now()))
				w.release(a, releaseAt)
			}
			tt.Advance(target.Sub(tt.Now()))
			scale, opacity := w.visual(tt.Now())
			width, height := int(float32(r.Width)*scale), int(float32(r.Height)*scale)
			tt.SetSize(width, height)
			tt.Frame()
			if kind == "pin" && fraction == 1 {
				tt.Move(float32(width)/2, float32(height)/2)
				tt.Frame()
			}
			bitmap := ui.NewBitmap(tt.Image())
			frame := ui.NewTester(func(c *ui.Context) {
				c.SetTheme(momentoTheme("light", false))
				c.Root().Background(c.Theme().Background)
				ui.Textf(c, "%s · %.0f%% · %dms", map[string]string{"quick": "快按保留", "peek": "偷看松开", "pin": "按满固定"}[kind], fraction*100, target.Sub(start).Milliseconds()).Absolute().Left(20).Top(12).FontSize(13).TextColor(c.Theme().TextMuted)
				ui.Box(c).Absolute().Left(float32(748-width)/2).Top(44+float32(580-height)/2).Size(float32(width), float32(height)).Radius(12).Opacity(opacity).Shadow(0, 8, 24, 0, ui.Hex("#211A16").Alpha(.2)).Children(func() { ui.Image(c, bitmap).Fill() })
			}, 748, 640)
			img := image.NewRGBA(frame.Image().Bounds())
			draw.Draw(img, img.Bounds(), frame.Image(), image.Point{}, draw.Src)
			frames = append(frames, img)
		}
		contact := image.NewRGBA(image.Rect(0, 0, 1496, 1280))
		for i, frame := range frames {
			r := image.Rect((i%2)*748, (i/2)*640, (i%2+1)*748, (i/2+1)*640)
			draw.Draw(contact, r, frame, image.Point{}, draw.Src)
		}
		writePreviewWindowContact(t, filepath.Join(dir, kind+".png"), contact)
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
