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
	a := &App{Preferences: DefaultPreferences(), Texts: i18n.Catalog{Language: "en"}, Query: catalog.Query{Sort: "added", Descending: true}, SidebarVisible: true, InspectorVisible: true, SidebarWidth: 236}
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
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Fill().Children(func() {
			if a.SidebarVisible {
				ui.Box(c).Width(a.SidebarWidth + 1).Shrink(0)
			}
			ui.Column(c).Grow(1).FillHeight().MinWidth(0).Children(func() { a.toolbar(c); ui.Spacer(c) })
			if a.InspectorVisible {
				ui.Box(c).Width(280).Shrink(0)
			}
		})
	}, 1500, 640)
	tt.Frame()
	return tt
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
	a.SidebarVisible = false
	tt.Frame()
	toolbarClick(t, tt, a.T("toggle_sidebar"))
	if !a.SidebarVisible {
		t.Fatal("折叠按钮未切换状态")
	}
}

func TestToolbarIconTooltips(t *testing.T) {
	keys := []string{"toggle_sidebar", "filter", "sort", "view_masonry", "view_grid", "view_list", "import"}
	for _, language := range []string{"en", "zh-CN"} {
		for _, key := range keys {
			t.Run(language+"/"+key, func(t *testing.T) {
				a := toolbarTestApp()
				a.Texts.Language = language
				if key == "toggle_sidebar" {
					a.SidebarVisible = false
				}
				tt := toolbarTester(a)
				label := a.T(key)
				box, ok := tt.Find(label)
				if !ok {
					t.Fatalf("缺少图标 %s", key)
				}
				tt.Move(box.X+box.W/2, box.Y+box.H/2)
				tt.Advance(time.Second)
				if !tt.HasText(label) {
					t.Fatalf("悬停没有显示 %s", label)
				}
			})
		}
	}
}

func TestToolbarColorNamesAndSelection(t *testing.T) {
	for _, language := range []string{"en", "zh-CN"} {
		for _, family := range toolbarFamilies {
			t.Run(language+"/"+family.ID, func(t *testing.T) {
				a := toolbarTestApp()
				a.Texts.Language = language
				tt := toolbarTester(a)
				toolbarClick(t, tt, a.T("filter"))
				label := a.T("color_" + family.ID)
				box, ok := tt.Find(label)
				if !ok {
					t.Fatalf("缺少颜色 %s", family.ID)
				}
				tt.Move(box.X+box.W/2, box.Y+box.H/2)
				tt.Advance(time.Second)
				if !tt.HasText(label) {
					t.Fatalf("悬停没有显示颜色族 %s", label)
				}
				toolbarClick(t, tt, label)
				access, ok := tt.Access(label)
				if !ok || !access.Checked {
					t.Fatalf("颜色族没有报告选中状态：%+v", access)
				}
				toolbarClick(t, tt, label)
				access, ok = tt.Access(label)
				if !ok || access.Checked {
					t.Fatal("第二次点击没有取消选中")
				}
			})
		}
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
}

func TestToolbarUpdateAppearsOnlyForKnownVersion(t *testing.T) {
	previous := updates
	updates = newUpdateChecker(nil, nil, func(fn func()) { fn() })
	t.Cleanup(func() { updates = previous })
	a := toolbarTestApp()
	tt := toolbarTester(a)
	if _, exists := tt.AccessByRole(ui.RoleButton, a.T("update")); exists {
		t.Fatal("没有已知更新时不应显示更新按钮")
	}
	updates.mu.Lock()
	updates.status = UpdateCheckStatus{State: UpdateAvailable, Version: "2.1.0"}
	updates.mu.Unlock()
	tt.Frame()
	if _, exists := tt.AccessByRole(ui.RoleButton, a.T("update")); !exists {
		t.Fatal("发现新版本后应显示更新按钮")
	}
}

func TestNarrowToolbarKeepsThumbnailPopover(t *testing.T) {
	a := toolbarTestApp()
	a.SidebarWidth = 236
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Fill().Children(func() {
			ui.Box(c).Width(236)
			ui.Column(c).Grow(1).FillHeight().MinWidth(0).Children(func() { a.toolbar(c); ui.Spacer(c) })
			ui.Box(c).Width(280)
		})
	}, 1100, 640)
	tt.Frame()
	button, ok := tt.AccessByRole(ui.RoleButton, a.T("thumbnail_size"))
	if !ok || button.Disabled {
		t.Fatal("默认1100窗口没有缩略图入口")
	}
	filter, ok := tt.AccessByRole(ui.RoleButton, a.T("filter"))
	if !ok {
		t.Fatal("缺筛选入口")
	}
	if button.Bounds.W != filter.Bounds.W || button.Bounds.H != filter.Bounds.H {
		t.Fatal("缩略图入口与同排按钮尺寸不一致")
	}
	search, ok := tt.AccessByRole(ui.RoleTextField, a.T("search_assets"))
	if !ok || search.Bounds.W < 96 {
		t.Fatal("缩略图入口挤压了搜索框")
	}
	if _, ok := tt.AccessByRole(ui.RoleSlider, a.T("thumbnail_size")); ok {
		t.Fatal("窄窗口仍显示完整滑块")
	}
	toolbarClick(t, tt, a.T("thumbnail_size"))
	slider, ok := tt.AccessByRole(ui.RoleSlider, a.T("thumbnail_size"))
	if !a.Toolbar.ThumbnailOpen || !ok || slider.Disabled {
		t.Fatal("缩略图浮层未打开")
	}
	before := a.thumbnailScale()
	toolbarClick(t, tt, a.T("thumbnail_larger"))
	if a.thumbnailScale() <= before {
		t.Fatal("浮层中的缩略图按钮没有生效")
	}
	tt.Key(0, ui.KeyEscape)
	if a.Toolbar.ThumbnailOpen {
		t.Fatal("Esc没有关闭缩略图浮层")
	}
	a.Query.Search = "no-result"
	a.Refine()
	tt.Frame()
	search, _ = tt.AccessByRole(ui.RoleTextField, a.T("search_assets"))
	if search.Bounds.W < 96 {
		t.Fatalf("结果提示挤压了窄窗口搜索框：%+v", search.Bounds)
	}
	a.ViewMode = 2
	tt.Frame()
	button, ok = tt.AccessByRole(ui.RoleButton, a.T("thumbnail_size"))
	if !ok || !button.Disabled {
		t.Fatal("列表视图缩略图入口没有禁用")
	}
	a.ViewMode = 0
	tt.SetSize(1500, 640)
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleButton, a.T("thumbnail_size")); ok {
		t.Fatal("宽窗口仍只显示缩略图入口")
	}
	if slider, ok = tt.AccessByRole(ui.RoleSlider, a.T("thumbnail_size")); !ok || slider.Disabled {
		t.Fatal("宽窗口没有完整滑块")
	}
}
