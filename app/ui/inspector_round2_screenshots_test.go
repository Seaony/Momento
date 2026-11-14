package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func TestInspectorRound2Screenshots(t *testing.T) {
	if os.Getenv("MOMENTO_INSPECTOR_ROUND2_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_INSPECTOR_ROUND2_SCREENSHOTS=1 生成第二轮检查器与标签截图")
	}
	dir := "/tmp/momento-polish/shots2/inspector"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := newScreenshotFixture(t)
	for _, theme := range []string{"light", "dark"} {
		for _, state := range []string{"single-details", "single-details-bottom", "multiple", "tag-picker", "tags", "tags-empty", "tags-hover"} {
			t.Run(theme+"/"+state, func(t *testing.T) {
				base := state
				if state == "tags-hover" {
					base = "tags"
				}
				if state == "tags-empty" {
					base = "small"
				}
				a := fixture.app(t, base, "zh-CN", theme)
				if state == "tags-empty" {
					a.TagPage = true
				}
				a.RevealInspector()
				tt := ui.NewTester(a.View, 1440, 900)
				tt.SetLocale(a.Texts.Locale())
				tt.SetTitleBar(ui.TitleBar{Height: MainToolbarHeight, Left: 80, Bottom: 37})
				tt.SetWindowActive(true)
				tt.Advance(time.Second)
				if state == "single-details-bottom" {
					tt.Scroll(1340, 800, 0, 2000)
					tt.Advance(time.Second)
				}
				if state == "tags-hover" {
					bounds, ok := tt.Find("Landscape 山水")
					if !ok {
						t.Fatal("没有找到悬停标签")
					}
					tt.Move(bounds.X+bounds.W/2, bounds.Y+bounds.H/2)
					tt.Advance(time.Second)
				}
				writeScreenshot(t, filepath.Join(dir, state+"-zh-CN-"+theme+".png"), tt)
			})
		}
	}
}

func TestTagsRound2MotionFrames(t *testing.T) {
	if os.Getenv("MOMENTO_INSPECTOR_ROUND2_SCREENSHOTS") != "1" {
		t.Skip("显式运行第二轮截图时生成标签动画帧")
	}
	dir := "/tmp/momento-polish/shots2/motion"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	a := tagsTestApp(model.Snapshot{Tags: []model.Tag{{ID: "a", Name: "Alpha"}, {ID: "b", Name: "Beta"}, {ID: "c", Name: "Gamma"}}})
	a.Preferences = DefaultPreferences()
	a.Preferences.Theme = "light"
	a.Texts.Language = "zh-CN"
	tt := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		ui.Column(c).Fill().Children(func() { a.tags(c) })
	}, 720, 480)
	tt.Advance(time.Second)
	a.Snapshot.Tags = []model.Tag{{ID: "a", Name: "Alpha"}, {ID: "c", Name: "Gamma"}, {ID: "d", Name: "Omega"}}
	tt.Frame()
	previous := time.Duration(0)
	for _, frame := range []struct {
		name string
		at   time.Duration
	}{{"0", 0}, {"35", 87 * time.Millisecond}, {"70", 175 * time.Millisecond}, {"100", 500 * time.Millisecond}} {
		tt.Advance(frame.at - previous)
		previous = frame.at
		writeScreenshot(t, filepath.Join(dir, "tags-"+frame.name+".png"), tt)
	}
}
