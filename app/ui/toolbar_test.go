package ui

import (
	"slices"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func toolbarTestApp() *App {
	a := &App{Texts: i18n.Catalog{Language: "en"}, Query: catalog.Query{Sort: "added", Descending: true}, SidebarVisible: true, InspectorVisible: true}
	a.Snapshot = model.Snapshot{
		Assets: []model.Asset{
			{ID: "a", Name: "Alpine", Extension: "jpg", Colors: []model.Color{{Family: "red"}}, CreatedAt: time.Unix(1, 0), TagIDs: []string{"ref"}},
			{ID: "b", Name: "Beach", Extension: "png", Colors: []model.Color{{Family: "blue"}}, CreatedAt: time.Unix(2, 0)},
			{ID: "c", Name: "Canvas", Extension: "JPG", Colors: []model.Color{{Family: "red"}}, CreatedAt: time.Unix(3, 0)},
		}, Tags: []model.Tag{{ID: "ref", Name: "Reference"}},
	}
	a.Index = catalog.New(a.Snapshot)
	a.Refine()
	return a
}

func toolbarTester(a *App) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { a.toolbar(c); ui.Spacer(c) }) }, 584, 640)
}

func toolbarClick(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
}

func TestToolbarSearchAndViewSelection(t *testing.T) {
	a := toolbarTestApp()
	tt := toolbarTester(a)
	toolbarClick(t, tt, a.T("search_assets"))
	tt.Type("alpine")
	if len(a.Visible) != 1 || a.Visible[0].ID != "a" {
		t.Fatalf("搜索结果：%v", a.Visible)
	}
	toolbarClick(t, tt, a.T("view_grid"))
	if a.ViewMode != 1 {
		t.Fatalf("网格视图：%d", a.ViewMode)
	}
	toolbarClick(t, tt, a.T("view_list"))
	if a.ViewMode != 2 {
		t.Fatalf("列表视图：%d", a.ViewMode)
	}
	toolbarClick(t, tt, a.T("toggle_sidebar"))
	toolbarClick(t, tt, a.T("toggle_inspector"))
	if a.SidebarVisible || a.InspectorVisible {
		t.Fatal("折叠按钮未切换状态")
	}
}

func TestToolbarFiltersAndRepeatedSort(t *testing.T) {
	a := toolbarTestApp()
	tt := toolbarTester(a)
	toolbarClick(t, tt, a.T("sort"))
	toolbarClick(t, tt, a.T("sort_name"))
	if a.Query.Sort != "name" || a.Query.Descending || a.Visible[0].ID != "a" {
		t.Fatalf("名称排序：%+v", a.Query)
	}
	toolbarClick(t, tt, a.T("sort"))
	toolbarClick(t, tt, a.T("sort_name"))
	if !a.Query.Descending || a.Visible[0].ID != "c" {
		t.Fatalf("名称反向排序：%+v", a.Query)
	}
	toolbarClick(t, tt, a.T("filter"))
	toolbarClick(t, tt, a.T("color_red"))
	if !slices.Contains(a.Query.ColorFamilies, "red") || len(a.Visible) != 2 {
		t.Fatalf("颜色筛选：%+v", a.Query)
	}
	toolbarClick(t, tt, "Reference")
	if len(a.Visible) != 1 || a.Visible[0].ID != "a" {
		t.Fatalf("标签筛选：%+v", a.Query)
	}
	toolbarClick(t, tt, a.T("clear_filters"))
	if len(a.Visible) != 3 || len(a.Query.ColorFamilies)+len(a.Query.TagIDs)+len(a.Query.Extensions) != 0 {
		t.Fatalf("清除筛选：%+v", a.Query)
	}
	toolbarClick(t, tt, "JPG")
	if len(a.Visible) != 2 {
		t.Fatalf("文件类型筛选：%+v", a.Query)
	}
	toolbarClick(t, tt, "PNG")
	if len(a.Visible) != 3 {
		t.Fatalf("同组文件类型应取并集：%+v", a.Query)
	}
}

func TestToolbarActualTypesAndToggle(t *testing.T) {
	a := toolbarTestApp()
	if got := a.toolbarExtensions(); !slices.Equal(got, []string{"jpg", "png"}) {
		t.Fatalf("文件类型：%v", got)
	}
	values := toolbarToggle(nil, "red", true)
	values = toolbarToggle(values, "red", true)
	if !slices.Equal(values, []string{"red"}) {
		t.Fatalf("重复筛选：%v", values)
	}
	if got := toolbarToggle(values, "red", false); len(got) != 0 {
		t.Fatalf("移除筛选：%v", got)
	}
	if got := toolbarNumber(12000); got != "12,000" {
		t.Fatal(got)
	}
}

func TestToolbarUpdateAppearsOnlyForKnownVersion(t *testing.T) {
	previous := availableUpdateVersion
	t.Cleanup(func() { availableUpdateVersion = previous })
	availableUpdateVersion = ""
	a := toolbarTestApp()
	tt := toolbarTester(a)
	if _, exists := tt.AccessByRole(ui.RoleButton, a.T("update")); exists {
		t.Fatal("没有已知更新时不应显示更新按钮")
	}
	availableUpdateVersion = "2.1.0"
	tt.Frame()
	if _, exists := tt.AccessByRole(ui.RoleButton, a.T("update")); !exists {
		t.Fatal("发现新版本后应显示更新按钮")
	}
}
