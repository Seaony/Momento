package ui

import (
	"fmt"
	"slices"
	"testing"

	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func TestBrowseHistoryBranchesAndIncludesTagPage(t *testing.T) {
	a := sidebarTestApp()
	a.SetScope("favorites", "")
	a.SetScope("folder", "root")
	a.BrowseTag("ref")
	a.NavigateBack()
	if a.Query.FolderID != "root" {
		t.Fatal("标签范围没有保存历史")
	}
	a.ShowTagsPage()
	if !a.TagPage || !a.CanNavigateBack() || a.CanNavigateForward() {
		t.Fatal("标签页没有进入历史")
	}
	a.NavigateBack()
	if a.TagPage || a.Query.FolderID != "root" {
		t.Fatal("后退没有恢复文件夹范围")
	}
	a.NavigateBack()
	if a.Query.Scope != "favorites" {
		t.Fatal("后退没有恢复收藏范围")
	}
	a.NavigateForward()
	if a.Query.FolderID != "root" {
		t.Fatal("前进没有恢复文件夹范围")
	}
	a.SetScope("trash", "")
	if a.CanNavigateForward() {
		t.Fatal("切换新范围后仍保留旧的前进记录")
	}
	a.NavigateBack()
	if a.Query.FolderID != "root" {
		t.Fatal("新范围没有保存来源")
	}
}

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

func TestScopeSummaryMatchesSidebarCountsWithSearch(t *testing.T) {
	a := sidebarTestApp()
	for i := range a.Snapshot.Assets {
		a.Snapshot.Assets[i].Size = int64((i + 1) * 100)
	}
	a.Index = catalog.New(a.Snapshot)
	for _, tc := range []struct {
		scope, folder string
		count         int
		size          int64
	}{
		{"all", "", 2, 300}, {"favorites", "", 1, 100}, {"folder", "root", 1, 200}, {"trash", "", 1, 300}, {"random", "", 2, 300},
	} {
		a.SetScope(tc.scope, tc.folder)
		a.Query.Search = "missing"
		a.Refine()
		_, count, size := a.scopeSummary()
		if count != tc.count || size != tc.size || len(a.Visible) != 0 {
			t.Fatalf("范围合计不符：%s %d %d", tc.scope, count, size)
		}
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
	a.ChangeThumbnailSize(-.1)
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
		if tt.HasText(a.T(key)) {
			t.Fatalf("隐藏项仍可见：%s", key)
		}
	}
	for _, key := range []string{"all", "trash", "tag_management"} {
		if !tt.HasText(a.T(key)) {
			t.Fatalf("固定项消失：%s", key)
		}
	}
	if tt.HasText("2") || tt.HasText("1") {
		t.Fatal("侧栏仍显示数量")
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
			if action == "rename" && (!a.PromptOpen || a.PromptValue != "Design references") {
				t.Fatal("双击没有打开重命名")
			}
		})
	}
}

func TestInspectorNotesAndSourceCanBeEdited(t *testing.T) {
	a := inspectorTestApp(model.Asset{ID: "a", Name: "Reference", Note: "Original", SourceURL: "https://example.com"})
	tt := ui.NewTester(a.inspector, 280, 1000)
	field, ok := tt.AccessByRole(ui.RoleTextField, a.T("asset_note"))
	if !ok {
		t.Fatal("没有备注输入框")
	}
	tt.ClickAt(field.Bounds.X+field.Bounds.W/2, field.Bounds.Y+field.Bounds.H/2)
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("Two lines\nSecond line")
	if a.Inspector.Note != "Two lines\nSecond line" {
		t.Fatalf("备注没有保留多行：%q focus=%v", a.Inspector.Note, tt.Focused(a.T("asset_note")))
	}
	field, ok = tt.AccessByRole(ui.RoleTextField, a.T("source_url"))
	if !ok {
		t.Fatal("没有来源输入框")
	}
	tt.ClickAt(field.Bounds.X+field.Bounds.W/2, field.Bounds.Y+field.Bounds.H/2)
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyBackspace)
	if a.Inspector.SourceURL != "" {
		t.Fatal("来源链接不能清空")
	}
	if _, ok := tt.Find(a.T("open_source_url")); ok {
		t.Fatal("空链接仍显示打开按钮")
	}
}

func TestBrowseTagShowsItsAssetsAndHistory(t *testing.T) {
	a := toolbarTestApp()
	a.BrowseTag("ref")
	name, count, _ := a.scopeSummary()
	if name != "Reference" || count != 1 || len(a.Visible) != 1 || a.Visible[0].ID != "a" {
		t.Fatal("标签范围不正确")
	}
	a.SetScope("trash", "")
	a.NavigateBack()
	if a.Query.Scope != "tag" || a.Browse.TagID != "ref" || len(a.Visible) != 1 {
		t.Fatal("历史没有恢复标签范围")
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

func TestThumbnailSliderIsDisabledInList(t *testing.T) {
	a := toolbarTestApp()
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { a.toolbar(c); ui.Spacer(c) }) }, 1500, 640)
	access, ok := tt.AccessByRole(ui.RoleSlider, a.T("thumbnail_size"))
	if !ok || access.Disabled {
		t.Fatal("宽窗口没有可用的缩略图滑块")
	}
	tt.ClickAt(access.Bounds.X+access.Bounds.W/2, access.Bounds.Y+access.Bounds.H/2)
	if a.thumbnailScale() == 1 {
		t.Fatal("滑块没有改变缩略图大小")
	}
	a.ViewMode = 2
	tt.Frame()
	access, ok = tt.AccessByRole(ui.RoleSlider, a.T("thumbnail_size"))
	if !ok || !access.Disabled {
		t.Fatal("列表视图滑块没有禁用")
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
