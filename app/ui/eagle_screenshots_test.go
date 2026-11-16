package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// 第八阶段（Eagle 第一批）改到的界面：评分、筛选和排序、智能条件、批量重命名、子文件夹卡片、最近使用、检查器路径。
func TestEagleFeatureScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_STAGE8_SHOTS") != "1" {
		t.Skip("设置 MOMENTO_STAGE8_SHOTS=1 生成第八阶段截图")
	}
	dir := "/tmp/momento-stage8"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := newScreenshotFixture(t)
	states := []string{"single", "multiple", "filter", "sort", "smart", "rename-format", "rename-replace", "subfolders", "recent", "smart-more", "preview-flip", "similar", "similar-progress", "export-resize", "export-invalid", "annotate-edit", "annotate-hover", "inspector-annotations"}
	if only := os.Getenv("MOMENTO_STAGE8_ONLY"); only != "" {
		states = strings.Split(only, ",")
	}
	for _, theme := range []string{"light", "dark"} {
		for _, state := range states {
			a := fixture.app(t, "small", "zh-CN", theme)
			snapshot := a.Snapshot
			snapshot.Assets = append([]model.Asset(nil), snapshot.Assets...)
			for i := range snapshot.Assets {
				snapshot.Assets[i].Rating = []int{5, 3, 0, 4, 0, 1, 3}[i%7]
			}
			snapshot.Folders = []model.Folder{
				{ID: "travel", Name: "旅行", Icon: "camera"},
				{ID: "japan", Name: "日本", ParentID: "travel", SortIndex: 0},
				{ID: "iceland", Name: "冰岛", ParentID: "travel", SortIndex: 1},
				{ID: "city", Name: "城市夜景", ParentID: "travel", SortIndex: 2},
				{ID: "empty", Name: "待整理", ParentID: "travel", SortIndex: 3},
				{ID: "kyoto", Name: "京都", ParentID: "japan"},
			}
			snapshot.Tags = []model.Tag{{ID: "mountain", Name: "山"}, {ID: "night", Name: "夜景"}}
			for i := range snapshot.Assets {
				asset := &snapshot.Assets[i]
				asset.FolderIDs = [][]string{{"travel"}, {"japan"}, {"kyoto"}, {"iceland"}, {"city"}}[i%5]
				asset.TagIDs = [][]string{{"mountain"}, {"night"}, nil}[i%3]
			}
			a.applySnapshot(snapshot)
			a.InspectorVisible = state == "single" || state == "multiple"
			switch state {
			case "single":
				a.Selected.Add(a.Visible[0].ID)
			case "multiple":
				for _, asset := range a.Visible[:3] {
					a.Selected.Add(asset.ID)
				}
			case "filter":
				a.Toolbar.FilterOpen = true
				a.Query.Ratings = []int{5}
				a.Refine()
			case "sort":
				a.Query.Sort, a.Query.Descending = "rating", true
				a.Refine()
				a.Toolbar.SortOpen = true
			case "smart":
				a.Preferences.SmartFolders = true
				a.EditSmartFolder("")
				a.GBrowsing.SmartName = "精选"
				a.GBrowsing.SmartRules = []model.SmartRule{{Field: "rating", Operator: "gte", Value: "4"}}
			case "rename-format", "rename-replace":
				ids := make([]string, 0, 5)
				for _, asset := range a.Visible[:5] {
					ids = append(ids, asset.ID)
				}
				a.OpenRename(ids)
				a.Rename.Template = "旅行-{n} {date}"
				if state == "rename-replace" {
					a.Rename.Mode, a.Rename.Find, a.Rename.Regex = 1, "([0-9]+", true
				}
			case "smart-more":
				a.Preferences.SmartFolders = true
				a.EditSmartFolder("")
				a.GBrowsing.SmartName = "竖版海报"
				a.GBrowsing.SmartRules = []model.SmartRule{{Field: "height", Operator: "between", Value: "1200,4000"}, {Field: "source", Operator: "not_contains", Value: "pinterest"}}
			case "similar":
				v := a.Visible
				a.Similar = similarState{Open: true, Groups: []similarGroup{
					{IDs: []string{v[0].ID, v[1].ID, v[2].ID}, Keep: v[1].ID},
					{IDs: []string{v[3].ID, v[4].ID}, Keep: v[3].ID},
				}}
			case "export-resize", "export-invalid":
				a.Selected.Add(a.Visible[0].ID)
				a.Preferences.ExportPresets = []ExportPreset{
					{Name: "网页 2048", ExportOptions: model.ExportOptions{Format: model.ExportJPEG, Quality: .8, MaxEdge: 2048}},
					{Name: "原图 PNG", ExportOptions: model.ExportOptions{Format: model.ExportPNG}},
				}
				// 跳过打开时的重置，直接摆成点了第一个预设的样子。
				a.ExportOpen, a.Dialog.ExportWasOpen = true, true
				a.applyExportPreset(a.Preferences.ExportPresets[0])
				if state == "export-invalid" {
					a.Dialog.MaxEdge = "0"
				}
			case "similar-progress":
				a.Progress = &model.ImportProgress{Stage: "similar", Processed: 1234, Total: 5000}
			case "subfolders":
				a.SetScope("folder", "travel")
			case "recent":
				a.SetScope("recent", "")
			}
			if strings.HasPrefix(state, "annotate") || state == "inspector-annotations" {
				snapshot := a.Snapshot
				snapshot.Assets = append([]model.Asset(nil), snapshot.Assets...)
				for i := range snapshot.Assets {
					if snapshot.Assets[i].ID == a.Visible[0].ID {
						snapshot.Assets[i].Annotations = []model.Annotation{
							{ID: "n1", X: .08, Y: .12, W: .34, H: .3, Text: "标题字太细，换中黑"},
							{ID: "n2", X: .5, Y: .55, W: .4, H: .3, Text: "这块留白可以再大一点"},
						}
					}
				}
				a.applySnapshot(snapshot)
				if state == "inspector-annotations" {
					a.InspectorVisible = true
					a.Selected.Add(a.Visible[0].ID)
				}
			}
			width, height := 1100, 700
			if state == "inspector-annotations" {
				height = 1150 // 批注在检查器下半部分
			}
			view := a.View
			if state == "preview-flip" || strings.HasPrefix(state, "annotate") {
				asset := a.Visible[0]
				photo, _, _, _, err := assets.PrepareImage(filepath.Join(fixture.root, filepath.FromSlash(asset.RelativePath)))
				if err != nil {
					t.Fatal(err)
				}
				a.ShowPreview(asset.ID, false)
				w := a.Preview.active
				w.picture.Bitmap = ui.NewBitmap(photo)
				w.flipped, w.gray = state == "preview-flip", state == "preview-flip"
				if state == "annotate-edit" {
					w.notes.On = true
					w.notes.edit = &annotationEdit{Rect: model.Annotation{X: .55, Y: .1, W: .35, H: .28}, Text: "换成品牌紫"}
				}
				frame := previewLayout(mygo.Rectangle{Width: 900, Height: 640}, mygo.Rectangle{}, asset, a.Preferences.PreviewFit == "fit", a.Texts.Locale(), 1)
				width, height = frame.Width, frame.Height
				view = func(c *ui.Context) { w.view(c, a) }
				w.openedAt = time.Now().Add(-time.Second)
			}
			tt := ui.NewTester(view, width, height)
			tt.SetLocale(a.Texts.Locale())
			tt.SetDark(theme == "dark")
			if state != "preview-flip" {
				tt.SetTitleBar(ui.TitleBar{Height: MainToolbarHeight, Left: 80, Bottom: 37})
			} else {
				a.Preview.active.openedAt = tt.Now().Add(-time.Second)
			}
			tt.SetWindowActive(true)
			tt.Frame()
			tt.Advance(time.Second)
			if state == "annotate-hover" {
				// 指针停在第二个框上，框下面显示它的文字。
				photo := a.Preview.active.picture.imageRect(a.Preview.active.picture.viewport)
				tt.Move(photo.X+photo.W*.7, photo.Y+photo.H*.7)
				tt.Frame()
			}
			writeScreenshot(t, filepath.Join(dir, state+"-zh-CN-"+theme+".png"), tt)
		}
	}
}
