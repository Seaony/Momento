package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestInspectorFixScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_INSPECTOR_FIX_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_INSPECTOR_FIX_SCREENSHOTS=1 生成检查器复核截图")
	}
	dir := "/tmp/momento-polish/shots3/inspector-fix"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := newScreenshotFixture(t)
	for _, theme := range []string{"light", "dark"} {
		for _, state := range []string{"filled", "empty"} {
			t.Run(theme+"/"+state, func(t *testing.T) {
				a := fixture.app(t, "tags", "zh-CN", theme)
				a.TagPage = false
				a.Selected.Clear()
				for i := range a.Snapshot.Assets {
					asset := &a.Snapshot.Assets[i]
					if asset.Extension != "jpg" || len(asset.Colors) != 8 {
						continue
					}
					asset.Name = "暖光下的山间晨雾.jpg"
					if state == "filled" {
						asset.Note = "晨光、层叠山色与低饱和色调。"
						asset.SourceURL = "https://example.com/references/landscape/early-morning-light?collection=mountain-study&image=warm-mist-2026"
					}
					a.Selected.Add(asset.ID)
					break
				}
				a.applySnapshot(a.Snapshot)
				a.RevealInspector()
				tt := ui.NewTester(a.View, 1440, 1000)
				tt.SetLocale(a.Texts.Locale())
				tt.SetTitleBar(ui.TitleBar{Height: MainToolbarHeight, Left: 80, Bottom: 37})
				tt.SetWindowActive(true)
				tt.Advance(time.Second)
				writeScreenshot(t, filepath.Join(dir, state+"-zh-CN-"+theme+".png"), tt)
			})
		}
	}
}
