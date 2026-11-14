package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestDesignBaseScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_DESIGN_BASE_SCREENSHOTS") != "1" {
		t.Skip("显式生成第三轮基础色板和字体截图")
	}
	dir := "/tmp/momento-polish/shots3/base"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := newScreenshotFixture(t)
	for _, theme := range []string{"light", "dark"} {
		a := fixture.app(t, "small", "zh-CN", theme)
		a.Selected.Clear()
		a.InspectorVisible = false
		tt := ui.NewTester(a.View, 1440, 900)
		tt.SetLocale(a.Texts.Locale())
		tt.SetTitleBar(ui.TitleBar{Height: MainToolbarHeight, Left: 80, Bottom: 37})
		tt.Advance(time.Second)
		writeScreenshot(t, filepath.Join(dir, theme+".png"), tt)
	}
}
