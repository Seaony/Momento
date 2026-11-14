package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func TestPreviewSymbolScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_PREVIEW_SYMBOL_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_PREVIEW_SYMBOL_SCREENSHOTS=1 输出预览符号截图")
	}
	dir := "/tmp/momento-polish/shots/symbol-polish"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, theme := range []string{"light", "dark"} {
		a := previewFixture(960, 640)
		a.Texts.Language, a.Preferences.Theme = "zh-CN", theme
		a.Visible[0].Name = "暖光下的山间晨雾.jpg"
		left, right := a.Visible[0], a.Visible[0]
		left.ID, right.ID = "previous", "next"
		a.Visible = []model.Asset{left, a.Visible[0], right}
		tt := ui.NewTester(func(c *ui.Context) {
			a.applyTheme(c)
			c.Root().Background(c.Theme().Background)
			a.preview(c)
		}, 1440, 900)
		tt.SetLocale(a.Texts.Locale())
		tt.Advance(5 * time.Millisecond)
		writeScreenshot(t, filepath.Join(dir, "preview-"+theme+".png"), tt)
		button, ok := tt.Find(a.T("preview_rotate_left"))
		if !ok {
			t.Fatal("找不到旋转按钮")
		}
		tt.Move(button.X+button.W/2, button.Y+button.H/2)
		writeScreenshot(t, filepath.Join(dir, "preview-hover-"+theme+".png"), tt)
	}
}
