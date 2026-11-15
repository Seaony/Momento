package ui

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func TestRandomBrowseKeepsAllActiveAssetsAndStableFiltering(t *testing.T) {
	a := toolbarTestApp()
	for i := range 30 {
		a.Snapshot.Assets = append(a.Snapshot.Assets, model.Asset{ID: fmt.Sprint(i), Name: "Extra", Trashed: i%7 == 0})
	}
	a.Index = catalog.New(a.Snapshot)
	a.SetScope("random", "")
	first := make([]string, len(a.Visible))
	for i, asset := range a.Visible {
		if asset.Trashed {
			t.Fatal("随机浏览包含已删除素材")
		}
		first[i] = asset.ID
	}
	if len(first) != a.Index.Counts().All {
		t.Fatal("随机浏览丢失素材")
	}
	a.Refine()
	for i, asset := range a.Visible {
		if first[i] != asset.ID {
			t.Fatal("重新筛选改变了随机顺序")
		}
	}
	a.Query.Search = "Extra"
	a.Refine()
	filtered := make([]string, len(a.Visible))
	for i, asset := range a.Visible {
		filtered[i] = asset.ID
	}
	at := -1
	for _, id := range filtered {
		next := slices.Index(first, id)
		if next <= at {
			t.Fatal("搜索没有保留随机顺序")
		}
		at = next
	}
	previous := a.Browse.RandomOrder
	a.SetScope("random", "")
	if fmt.Sprint(previous) == fmt.Sprint(a.Browse.RandomOrder) {
		t.Fatal("重新点随机浏览没有打乱")
	}
	sort := a.Query.Sort
	a.changeSort("name")
	if a.Query.Sort != sort {
		t.Fatal("随机浏览仍允许排序")
	}
}

func TestThumbnailSizeChangesMasonryAndGrid(t *testing.T) {
	a := toolbarTestApp()
	a.SetThumbnailSize(2)
	if a.thumbnailScale() != 2 {
		t.Fatal("缩略图大小没有应用")
	}
	small := NewMasonryScaled([]float32{1, 1, 1, 1, 1, 1}, 700, .5)
	large := NewMasonryScaled([]float32{1, 1, 1, 1, 1, 1}, 700, 2)
	if len(small.columns) <= len(large.columns) || small.columns[0][0].Width >= large.columns[0][0].Width {
		t.Fatal("瀑布流没有随缩略图大小改列宽")
	}
	a.ViewMode = 2
	a.StepThumbnailSize(-1)
	if a.thumbnailScale() != 2 {
		t.Fatal("列表视图仍允许调整缩略图")
	}
}

func TestSidebarPreferencesHideCountsAndItems(t *testing.T) {
	a := sidebarTestApp()
	a.Preferences.SidebarCounts = false
	a.Preferences.SidebarHidden = []string{"favorites", "unfiled", "untagged", "random", "folders"}
	tt := ui.NewTester(a.sidebar, 236, 800)
	for _, key := range []string{"favorites", "uncategorized", "untagged", "random_browse", "folders"} {
		if slices.Contains(tt.Texts(), a.T(key)) {
			t.Fatalf("隐藏项仍可见：%s", key)
		}
	}
	for _, key := range []string{"all", "trash", "tag_management"} {
		if !tt.HasText(a.T(key)) {
			t.Fatalf("固定项消失：%s", key)
		}
	}
	for _, text := range tt.Texts() {
		if text == "2" || text == "1" {
			t.Fatal("侧栏行仍显示数量")
		}
	}
}

func TestSidebarFolderDoubleClickPreference(t *testing.T) {
	for _, action := range []string{"expand", "rename"} {
		t.Run(action, func(t *testing.T) {
			a := sidebarTestApp()
			a.Preferences.FolderDoubleClick = action
			tt := ui.NewTester(a.sidebar, 236, 800)
			if err := tt.Click("Design references"); err != nil {
				t.Fatal(err)
			}
			if err := tt.Click("Design references"); err != nil {
				t.Fatal(err)
			}
			if action == "expand" && !tt.HasText("UI details") {
				t.Fatal("双击没有展开文件夹")
			}
			if action == "rename" && (!a.Dialog.Folder.Open || a.Dialog.Folder.Name != "Design references" || a.Dialog.Folder.ID == "") {
				t.Fatal("双击没有打开编辑文件夹")
			}
		})
	}
}

func TestBrowseTagShowsItsAssetsAndClearsPreviousScope(t *testing.T) {
	a := toolbarTestApp()
	a.BrowseTag("ref")
	if a.Query.Scope != "tag" || a.Browse.TagID != "ref" || len(a.Visible) != 1 || a.Visible[0].ID != "a" {
		t.Fatal("标签范围不正确")
	}
	a.SetScope("trash", "")
	if a.Query.Scope != "trash" || a.Browse.TagID != "" || a.TagPage {
		t.Fatal("切换回收站后仍保留标签范围")
	}
}

func TestInspectorKeepsDraftWhenImportIsBusy(t *testing.T) {
	a := sidebarTestApp()
	a.Busy = true
	a.Inspector = inspectorState{AssetID: "a", Title: "Landscape", SourceTitle: "Landscape", Note: "Unfinished note", SourceURL: "https://example.com"}
	a.saveInspectorEdits()
	if len(a.Inspector.Pending) != 1 || a.Inspector.Pending[0].ID != "a" || a.Inspector.Pending[0].Note != "Unfinished note" || a.Inspector.Pending[0].Source != "https://example.com" {
		t.Fatal("后台导入时丢掉了待保存文字")
	}
}

func TestThumbnailSliderHiddenInList(t *testing.T) {
	a := toolbarTestApp()
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { a.toolbar(c); ui.Spacer(c) }) }, 1500, 640)
	access, ok := tt.AccessByRole(ui.RoleSlider, a.T("thumbnail_size"))
	if !ok || access.Disabled {
		t.Fatal("宽窗口没有可用的缩略图滑块")
	}
	// 刻度尺：刻度固定，左端最小、右端最大；第 k 档在左边留白之后 k 个档距的位置。
	gap := float32(rulerWidth-2*rulerPad) / float32(len(thumbnailSteps)-1)
	left, y := access.Bounds.X+rulerPad, access.Bounds.Y+access.Bounds.H/2
	stepX := func(k float32) float32 { return left + k*gap }
	// 点哪一档就到哪一档，点在两档之间按近的算。
	for _, step := range []struct {
		at   float32
		want int
	}{{3, 3}, {0.3, 0}, {4, 4}, {1.6, 2}} {
		tt.ClickAt(stepX(step.at), y)
		tt.Frame()
		if got := thumbnailStep(a.thumbnailScale()); got != step.want {
			t.Fatalf("点在第 %v 档处停在第 %d 档，应为 %d", step.at, got, step.want)
		}
	}
	// 从第 2 档拖到第 4 档：拖着时就跟着换档，松手停在第 4 档。
	tt.Advance(time.Second)
	tt.Press(stepX(2), y)
	for x := stepX(2); x <= stepX(4); x += 4 {
		tt.Move(x, y)
		tt.Frame()
		if x >= stepX(3)+2 && x < stepX(3.5)-2 {
			if got := thumbnailStep(a.thumbnailScale()); got != 3 {
				t.Fatalf("拖过第 3 档时停在第 %d 档", got)
			}
		}
	}
	tt.Release(stepX(4), y)
	tt.Frame()
	if got := thumbnailStep(a.thumbnailScale()); got != 4 {
		t.Fatalf("拖到右端后停在第 %d 档，应为 4", got)
	}
	// 拖过左端也只停在最左一档；指针移出刻度区（上下移开）也一直跟着，松手不算点击。
	tt.Advance(time.Second)
	tt.Press(stepX(4), y)
	tt.Move(stepX(4), y+200)
	tt.Frame()
	for x := stepX(4); x >= stepX(-1); x -= 6 {
		tt.Move(x, y+200)
		tt.Frame()
	}
	tt.Release(stepX(-1), y+200)
	tt.Frame()
	if got := thumbnailStep(a.thumbnailScale()); got != 0 {
		t.Fatalf("拖过左端后停在第 %d 档，应为 0", got)
	}
	// 点过刻度后方向键直接能调。
	tt.Advance(time.Second)
	tt.ClickAt(stepX(2), y)
	tt.Frame()
	tt.Key(0, ui.KeyLeft)
	tt.Frame()
	if got := thumbnailStep(a.thumbnailScale()); got != 1 {
		t.Fatalf("点刻度后按左方向键停在第 %d 档，应为 1", got)
	}
	a.ViewMode = 2
	tt.Frame()
	tt.Advance(time.Second)
	if _, ok := tt.AccessByRole(ui.RoleSlider, a.T("thumbnail_size")); ok {
		t.Fatal("列表视图仍显示缩略图滑块")
	}
}

func TestInspectorFailedDraftIsRetainedAndRestored(t *testing.T) {
	a := sidebarTestApp()
	a.Inspector = inspectorState{LibraryID: "test", Pending: []inspectorEdit{{ID: "a", Note: "Saved nowhere yet", Source: "https://example.com/draft", Details: true, Failed: true}}}
	a.Selected.Add("b")
	tt := ui.NewTester(a.inspector, 280, 1000)
	if !a.InspectorEditsPending() || !a.InspectorEditsFailed() {
		t.Fatal("保存失败后没有保留草稿")
	}
	a.ContinueInspectorEdits()
	if a.Busy {
		t.Fatal("界面每帧自动重试了失败写入")
	}
	a.Selected.Clear()
	a.Selected.Add("a")
	tt.Frame()
	if a.Inspector.Note != "Saved nowhere yet" || a.Inspector.SourceURL != "https://example.com/draft" {
		t.Fatal("重新选择素材没有恢复失败草稿")
	}
}

// 缩略图只停在几个档位：任意值吸附到最近一档，菜单和快捷键一次走一档，两端不越界。
func TestThumbnailSizeSnapsToSteps(t *testing.T) {
	a := toolbarTestApp()
	a.SetThumbnailSize(1.3)
	if a.thumbnailScale() != 1.5 {
		t.Fatalf("1.3 没有吸附到 1.5：%v", a.thumbnailScale())
	}
	a.StepThumbnailSize(-1)
	if a.thumbnailScale() != 1 {
		t.Fatalf("往小一档应是 1：%v", a.thumbnailScale())
	}
	for range 10 {
		a.StepThumbnailSize(1)
	}
	if a.thumbnailScale() != 2 {
		t.Fatalf("最大只到 2：%v", a.thumbnailScale())
	}
	a.Preferences.ThumbnailSize = .6
	if a.thumbnailScale() != .5 {
		t.Fatalf("旧偏好 0.6 应显示为 0.5 档：%v", a.thumbnailScale())
	}
}
