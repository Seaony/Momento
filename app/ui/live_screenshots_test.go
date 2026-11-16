package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// 实况照片相关界面：网格和列表标记、筛选里的「实况」、导出对话框的勾选、预览里的「实况」、智能文件夹的时长条件。
func TestLivePhotoScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_STAGE4_SHOTS") != "1" {
		t.Skip("设置 MOMENTO_STAGE4_SHOTS=1 生成实况照片截图")
	}
	dir := "/tmp/momento-stage4/live"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := newScreenshotFixture(t)
	for _, theme := range []string{"light", "dark"} {
		for _, state := range []string{"grid", "list", "filter", "export", "smart", "preview"} {
			a := fixture.app(t, "small", "zh-CN", theme)
			a.InspectorVisible = false
			snapshot := a.Snapshot
			snapshot.Assets = append([]model.Asset(nil), snapshot.Assets...)
			var live []string
			for _, i := range []int{1, 4, len(snapshot.Assets) - 2} {
				asset := &snapshot.Assets[i]
				asset.Kind, asset.DurationMS, asset.MotionPath = model.KindLive, 1600, "assets/live.mov"
				live = append(live, asset.ID)
			}
			snapshot.Assets[0].Kind, snapshot.Assets[0].DurationMS, snapshot.Assets[0].Extension = model.KindVideo, 12_000, "mp4"
			a.applySnapshot(snapshot)
			width, height := 1100, 700
			view := a.View
			switch state {
			case "list":
				a.ViewMode = 2
			case "filter":
				a.Toolbar.FilterOpen = true
			case "export":
				a.Selected.Add(live[0])
				a.Selected.Add(live[1])
				a.ExportOpen = true
			case "smart":
				a.Preferences.SmartFolders = true
				a.EditSmartFolder("")
				a.GBrowsing.SmartName = "短视频和实况"
				a.GBrowsing.SmartRules = []model.SmartRule{{Field: "duration", Operator: "lte", Value: "10"}}
			case "preview":
				var asset model.Asset
				for _, item := range a.Visible {
					if item.ID == live[0] {
						asset = item
					}
				}
				photo, _, _, _, err := assets.PrepareImage(filepath.Join(fixture.root, filepath.FromSlash(asset.RelativePath)))
				if err != nil {
					t.Fatal(err)
				}
				a.ShowPreview(asset.ID, false)
				w := a.Preview.active
				w.picture.Bitmap = ui.NewBitmap(photo)
				frame := previewLayout(mygo.Rectangle{Width: 900, Height: 640}, mygo.Rectangle{}, asset, a.Preferences.PreviewFit == "fit", a.Texts.Locale(), 1)
				width, height = frame.Width, frame.Height
				view = func(c *ui.Context) { w.view(c, a) }
				w.openedAt = time.Now().Add(-time.Second)
			}
			tt := ui.NewTester(view, width, height)
			tt.SetLocale(a.Texts.Locale())
			tt.SetDark(theme == "dark")
			if state != "preview" {
				tt.SetTitleBar(ui.TitleBar{Height: MainToolbarHeight, Left: 80, Bottom: 37})
			} else {
				a.Preview.active.openedAt = tt.Now().Add(-time.Second)
			}
			tt.SetWindowActive(true)
			tt.Frame()
			tt.Advance(time.Second)
			writeScreenshot(t, filepath.Join(dir, state+"-zh-CN-"+theme+".png"), tt)
		}
	}
}
