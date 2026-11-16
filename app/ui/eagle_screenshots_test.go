package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
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
	states := []string{"single", "multiple", "filter", "sort", "smart", "rename-format", "rename-replace", "subfolders", "recent"}
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
			case "subfolders":
				a.SetScope("folder", "travel")
			case "recent":
				a.SetScope("recent", "")
			}
			tt := ui.NewTester(a.View, 1100, 700)
			tt.SetLocale(a.Texts.Locale())
			tt.SetDark(theme == "dark")
			tt.SetTitleBar(ui.TitleBar{Height: MainToolbarHeight, Left: 80, Bottom: 37})
			tt.SetWindowActive(true)
			tt.Frame()
			tt.Advance(time.Second)
			writeScreenshot(t, filepath.Join(dir, state+"-zh-CN-"+theme+".png"), tt)
		}
	}
}
