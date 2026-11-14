package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func TestPreviewRound3LocalizedMetadata(t *testing.T) {
	for _, language := range []string{"en", "ja"} {
		for _, width := range []int{300, 338} {
			t.Run(fmt.Sprintf("%s/%d", language, width), func(t *testing.T) {
				a := previewFixture(64, 48)
				a.Texts.Language, a.Preferences.Language = language, language
				w := a.Preview.active
				var measured float32
				metadata := "64 × 48 · 100% · 1 / 1"
				tt := ui.NewTester(func(c *ui.Context) {
					w.view(c, a)
					measured, _ = c.MeasureText(0, ui.Span{Text: metadata, Font: monoFont(c), Size: fontNumber})
				}, width, 220)
				tt.SetLocale(a.Texts.Locale())
				bounds, ok := tt.Find(metadata)
				if !ok {
					t.Fatal("没有显示尺寸、比例和位置")
				}
				if measured > bounds.W+.1 {
					t.Fatalf("数字被截断：文字%.1f，可用%.1f", measured, bounds.W)
				}
				for _, item := range tt.AccessAll() {
					if item.Bounds.W > 0 && item.Bounds.H > 0 && (item.Bounds.X < 0 || item.Bounds.Y < 0 || item.Bounds.X+item.Bounds.W > float32(width)+.1 || item.Bounds.Y+item.Bounds.H > 220.1) {
						t.Fatalf("界面超出窗口：%+v", item)
					}
				}
			})
		}
	}
}

func TestPreviewRound3Screenshots(t *testing.T) {
	if os.Getenv("MOMENTO_PREVIEW_ROUND3_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_PREVIEW_ROUND3_SCREENSHOTS=1 输出第三轮预览截图")
	}
	dir := "/tmp/momento-polish/shots3/preview"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := newScreenshotFixture(t)
	for _, theme := range []string{"light", "dark"} {
		a := fixture.app(t, "small", "zh-CN", theme)
		var asset model.Asset
		for _, candidate := range a.Visible {
			if candidate.Extension == "jpg" {
				asset = candidate
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
		w.openedAt = time.Now().Add(-time.Second)
		frame := previewWindowBounds(mygo.Rectangle{Width: 1100, Height: 800}, mygo.Rectangle{Width: 1440, Height: 900}, asset, a.Preferences.PreviewFit == "fit")
		tt := ui.NewTester(func(c *ui.Context) { w.view(c, a) }, frame.Width, frame.Height)
		tt.SetLocale(a.Texts.Locale())
		tt.Advance(300 * time.Millisecond)
		writeScreenshot(t, filepath.Join(dir, "preview-"+theme+".png"), tt)
	}
}
