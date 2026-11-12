package ui

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestPreviewScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_PREVIEW_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_PREVIEW_SCREENSHOTS=1 输出预览截图")
	}
	dir := os.Getenv("MOMENTO_PREVIEW_SCREENSHOT_DIR")
	if dir == "" {
		dir = "/tmp/momento-eagle/preview/screenshots"
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	type record struct {
		File, State, Language, Theme string
		Width, Height                int
		Zoom                         float32
	}
	var records []record
	for _, language := range []string{"zh-Hans", "en", "ja"} {
		for _, theme := range []string{"light", "dark"} {
			for _, state := range []string{"small-smart", "small-fit", "large", "transparent-checker", "transparent-solid", "gif-first", "gif-second", "zoom400-smooth", "zoom400-pixelated", "rotation-90", "rotation-270", "gif-rotation-90"} {
				a := previewFixture(288, 192)
				a.Texts.Language = language
				a.Preferences.Theme = theme
				a.Visible[0].Name = state + ".png"
				switch state {
				case "rotation-90", "rotation-270", "gif-rotation-90":
					rotation := 90
					if state == "rotation-270" {
						rotation = 270
					}
					picture, err := readPreviewPicture(writeDirectionalPreview(t, state == "gif-rotation-90"), rotation)
					if err != nil {
						t.Fatal(err)
					}
					if state == "gif-rotation-90" {
						picture, err = picture.animation.next()
						if err != nil {
							t.Fatal(err)
						}
					}
					a.Preview.Bitmap, a.Preview.pixelBitmap = picture.bitmap, picture.pixelBitmap
					a.Visible[0].Rotation = rotation
					a.Preview.targetRotation = rotation
					a.Preferences.PreviewFit = "fit"
				case "small-fit":
					a.Preferences.PreviewFit = "fit"
				case "large":
					a = previewFixture(2400, 1600)
					a.Texts.Language = language
					a.Preferences.Theme = theme
					a.Visible[0].Name = "Large image.jpg"
				case "transparent-checker", "transparent-solid":
					picture := image.NewNRGBA(image.Rect(0, 0, 480, 320))
					for y := 40; y < 280; y++ {
						for x := 60; x < 420; x++ {
							if (x-240)*(x-240)+(y-160)*(y-160) < 110*110 {
								picture.SetNRGBA(x, y, color.NRGBA{R: 80, G: 130, B: 220, A: 190})
							}
						}
					}
					a.Preview.Bitmap, a.Preview.pixelBitmap = previewBitmaps(picture)
					a.Preferences.PreviewChecker = state == "transparent-checker"
				case "gif-first", "gif-second":
					picture, err := readPreviewPicture(writePreviewGIF(t), 0)
					if err != nil {
						t.Fatal(err)
					}
					if state == "gif-second" {
						picture, err = picture.animation.next()
						if err != nil {
							t.Fatal(err)
						}
					}
					a.Preview.Bitmap, a.Preview.pixelBitmap = picture.bitmap, picture.pixelBitmap
					a.Visible[0].Name = "Animated image.gif"
				}
				tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); ui.Box(c).Fill().Background(c.Theme().Background); a.preview(c) }, 1100, 640)
				tt.Advance(5 * time.Millisecond)
				if state == "zoom400-smooth" || state == "zoom400-pixelated" {
					a.Preview.Zoom = 4
					a.Preview.fitMode = false
					a.Preferences.PreviewPixelated = state == "zoom400-pixelated"
					tt.Frame()
				}
				file := state + "-" + language + "-" + theme + ".png"
				f, err := os.Create(filepath.Join(dir, file))
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(f, tt.Image())
				closeErr := f.Close()
				if err != nil {
					t.Fatal(err)
				}
				if closeErr != nil {
					t.Fatal(closeErr)
				}
				records = append(records, record{file, state, language, theme, 1100, 640, a.Preview.Zoom})
			}
		}
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
}
