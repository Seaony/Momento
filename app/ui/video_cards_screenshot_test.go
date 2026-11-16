package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

// 网格和列表里的视频：右下角播放标记加时长，列表第二行加时长。
func TestVideoCardScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_STAGE4_SHOTS") != "1" {
		t.Skip("设置 MOMENTO_STAGE4_SHOTS=1 生成视频卡片截图")
	}
	dir := "/tmp/momento-stage4/cards"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := newScreenshotFixture(t)
	for _, theme := range []string{"light", "dark"} {
		for _, view := range []string{"grid", "list"} {
			a := fixture.app(t, "small", "zh-CN", theme)
			a.InspectorVisible = false
			snapshot := a.Snapshot
			snapshot.Assets = append([]model.Asset(nil), snapshot.Assets...)
			for i, ms := range map[int]int64{len(snapshot.Assets) - 1: 12_000, len(snapshot.Assets) - 3: 754_000, 0: 3_723_000} {
				snapshot.Assets[i].Kind, snapshot.Assets[i].DurationMS, snapshot.Assets[i].Extension = model.KindVideo, ms, "mp4"
			}
			a.applySnapshot(snapshot)
			if view == "list" {
				a.ViewMode = 2
			}
			tt := ui.NewTester(a.View, 1100, 700)
			tt.SetLocale(a.Texts.Locale())
			tt.SetDark(theme == "dark")
			tt.SetTitleBar(ui.TitleBar{Height: MainToolbarHeight, Left: 80, Bottom: 37})
			tt.SetWindowActive(true)
			tt.Frame()
			tt.Advance(time.Second)
			writeScreenshot(t, filepath.Join(dir, view+"-zh-CN-"+theme+".png"), tt)
		}
	}
}
