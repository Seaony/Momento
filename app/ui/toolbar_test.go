package ui

import (
	"bytes"
	"fmt"
	"image"
	"maps"
	"os"
	"path/filepath"
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

// snapshotHasKind 判断整个快照（含回收站）里有没有这种素材。
func snapshotHasKind(a *App, kind string) bool {
	return slices.ContainsFunc(a.Snapshot.Assets, func(asset model.Asset) bool { return asset.Kind == kind })
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
	toolbarClick(t, tt, a.T("view_list"))
	if a.ViewMode != 2 {
		t.Fatalf("列表视图：%d", a.ViewMode)
	}
	toolbarClick(t, tt, a.T("view_masonry"))
	if a.ViewMode != 0 {
		t.Fatalf("瀑布流视图：%d", a.ViewMode)
	}
	a.SidebarVisible = false
	tt.Frame()
	toolbarClick(t, tt, a.T("toggle_sidebar"))
	if !a.SidebarVisible {
		t.Fatal("折叠按钮未切换状态")
	}
}

func TestToolbarIconTooltips(t *testing.T) {
	keys := []string{"toggle_sidebar", "filter", "sort", "view_masonry", "view_list", "import"}
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
				// 悬停提示和读屏名称都带当前位置里这个颜色的素材数。
				label := a.T("color_count", a.T("color_"+family.ID), map[string]int{"red": 2, "blue": 1}[family.ID])
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
	// 弹层不收起，直接再点当前字段反转方向。
	toolbarClick(t, tt, a.T("sort_name"))
	if !a.Query.Descending || a.Visible[0].ID != "c" {
		t.Fatalf("名称反向排序：%+v", a.Query)
	}
	toolbarClick(t, tt, a.T("filter"))
	toolbarClick(t, tt, a.T("color_count", a.T("color_red"), 2))
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

func TestToolbarFormatsAndCountsFollowSnapshotAndScope(t *testing.T) {
	a := toolbarTestApp()
	snapshot := a.Snapshot
	snapshot.Assets = slices.Clone(snapshot.Assets)
	snapshot.Assets[0].Width, snapshot.Assets[0].Height = 1920, 1080
	snapshot.Assets[1].Width, snapshot.Assets[1].Height = 1080, 1920
	snapshot.Assets[2].Width, snapshot.Assets[2].Height = 1920, 1080
	snapshot.Assets[2].Trashed, snapshot.Assets[2].Kind = true, model.KindVideo
	a.applySnapshot(snapshot)
	if !snapshotHasKind(a, model.KindVideo) || snapshotHasKind(a, model.KindLive) {
		t.Fatal("可选种类范围与图库素材不一致")
	}
	a.Query.Shapes = []string{catalog.ShapePortrait}
	a.Query.TagIDs = []string{"ref"}
	a.Refine()
	query := a.browseQuery(a.Query)
	extensions, kinds := a.typeCounts()
	wantExtensions, wantKinds := a.Index.TypeCounts(query)
	wantShapes := a.Index.ShapeCounts(query)
	query.TagIDs = nil
	if !maps.Equal(extensions, wantExtensions) || !maps.Equal(kinds, wantKinds) ||
		!maps.Equal(a.shapeCounts(), wantShapes) || !maps.Equal(a.tagCounts(), a.Index.TagCounts(query)) {
		t.Fatal("分组计数没有保留其他组的筛选条件")
	}

	snapshot.Assets[2].Trashed, snapshot.Assets[2].Kind, snapshot.Assets[2].Extension = false, model.KindLive, "heic"
	a.clearFilters()
	a.applySnapshot(snapshot)
	if got := a.toolbarExtensions(); !slices.Equal(got, []string{"heic", "jpg", "png"}) ||
		snapshotHasKind(a, model.KindVideo) || !snapshotHasKind(a, model.KindLive) {
		t.Fatalf("更新快照后仍使用旧扩展名或种类：%v", got)
	}
	if shapes := a.shapeCounts(); shapes[catalog.ShapeLandscape] != 2 {
		t.Fatalf("更新快照后的横图计数：%v", shapes)
	}

	a.SetScope("recent", "")
	if a.shapeCounts()[catalog.ShapeLandscape] != 0 {
		t.Fatal("尚未使用的素材计入了最近使用")
	}
	a.applyAssetsUsed([]string{"a"}, time.Now().Add(-time.Second))
	if a.shapeCounts()[catalog.ShapeLandscape] != 1 {
		t.Fatal("写入使用时间后没有刷新最近使用的分组计数")
	}
}

func filterCountPreviewApp(language, theme string) *App {
	a := toolbarTestApp()
	a.Preferences.Language, a.Preferences.Theme = language, theme
	a.Texts.Language = language
	snapshot := a.Snapshot
	snapshot.Assets[0].Width, snapshot.Assets[0].Height = 1920, 1080
	snapshot.Assets[1].Width, snapshot.Assets[1].Height = 1080, 1920
	snapshot.Assets[2].Width, snapshot.Assets[2].Height = 1000, 1000
	snapshot.Assets = append(snapshot.Assets,
		model.Asset{ID: "video", Extension: "mp4", Kind: model.KindVideo, Width: 1920, Height: 1080},
		model.Asset{ID: "live", Extension: "heic", Kind: model.KindLive, Width: 1500, Height: 1000},
	)
	if language == "zh-CN" {
		snapshot.Tags[0].Name = "灵感参考"
	}
	a.applySnapshot(snapshot)
	return a
}

func TestFilterCountLabelsFit(t *testing.T) {
	for _, language := range []string{"en", "ja"} {
		t.Run(language, func(t *testing.T) {
			a := filterCountPreviewApp(language, "light")
			labels := []string{"filter_shapes", "filter_kind_video", "filter_kind_live", "shape_landscape"}
			measured := make([]ui.Rect, len(labels))
			tt := ui.NewTester(func(c *ui.Context) {
				a.applyTheme(c)
				a.filterPanel(c)
				for i, label := range labels {
					size, weight := float32(fontSecondary), 400
					if label == "filter_shapes" {
						size, weight = fontGroup, 600
					}
					measured[i].W, measured[i].H = c.MeasureText(0, ui.Span{
						Text: a.T(label), Font: c.Theme().Font, Size: size, Weight: weight, Lang: c.Locale(),
					})
				}
			}, popFilterWidth, 460)
			tt.SetLocale(a.Texts.Locale())
			for i, label := range labels {
				box, ok := tt.Find(a.T(label))
				if !ok || box.W+1 < measured[i].W || box.H > measured[i].H+2 ||
					box.X+box.W > popFilterWidth || box.Y+box.H > 460 {
					t.Fatalf("筛选计数文案截断或换行：%s，实际 %+v，测量 %+v", a.T(label), box, measured[i])
				}
			}
		})
	}
}

func TestBatch2PerfScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_BATCH2_E_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_BATCH2_E_SCREENSHOTS=1 生成筛选和侧栏截图")
	}
	const dir = "/tmp/momento-batch2-shots"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, theme := range []string{"light", "dark"} {
		a := filterCountPreviewApp("zh-CN", theme)
		filter := ui.NewTester(func(c *ui.Context) {
			a.applyTheme(c)
			ui.Column(c).FillWidth().Background(panelColor(c.Theme())).Children(func() { a.filterPanel(c) })
		}, popFilterWidth, popHeaderHeight+popBodyMax+1)
		filter.SetLocale(a.Texts.Locale())
		if !filter.HasText(a.T("filter_kind_live")) || !filter.HasText(a.T("shape_landscape")) {
			t.Fatal("筛选截图缺少实况类型或形状计数")
		}
		writeScreenshot(t, filepath.Join(dir, "e-filter-zh-CN-"+theme+".png"), filter)

		a = smartTestApp()
		a.Preferences.Language, a.Preferences.Theme = "zh-CN", theme
		a.Texts.Language = "zh-CN"
		a.Library.Manifest.DisplayName = "参考图库"
		a.Snapshot.Folders[0].Name = "设计参考"
		a.Snapshot.SmartFolders[0].Name = "横向参考"
		a.Snapshot.SmartFolders[1].Name = "竖向参考"
		a.Snapshot.Tags[0].Name = "资料"
		a.applySnapshot(a.Snapshot)
		a.BrowseSmartFolder("smart-a")
		sidebar := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); a.sidebar(c) }, 236, 800)
		sidebar.SetLocale(a.Texts.Locale())
		sidebar.SetWindowActive(true)
		for _, mode := range []string{"counts", "no-counts"} {
			a.Preferences.SidebarCounts = mode == "counts"
			sidebar.Frame()
			if !sidebar.HasText("横向参考") || !sidebar.HasText("竖向参考") ||
				sidebar.HasText("1") != a.Preferences.SidebarCounts {
				t.Fatalf("智能侧栏截图状态错误：%s", mode)
			}
			writeScreenshot(t, filepath.Join(dir, "e-sidebar-"+mode+"-zh-CN-"+theme+".png"), sidebar)
		}
	}
}

func filterBenchmarkSnapshot() model.Snapshot {
	snapshot := model.Snapshot{
		Assets: make([]model.Asset, 100_000),
		Tags:   []model.Tag{{ID: "ref", Name: "Reference"}},
	}
	for i := range snapshot.Assets {
		snapshot.Assets[i] = model.Asset{
			ID: fmt.Sprint(i), Name: "Reference", Extension: "jpg",
			Width: 1920, Height: 1080, TagIDs: []string{"ref"},
		}
	}
	return snapshot
}

func BenchmarkFilterPanelData100K(b *testing.B) {
	for _, mode := range []string{"Warm", "Cold"} {
		b.Run(mode, func(b *testing.B) {
			a := &App{}
			a.applySnapshot(filterBenchmarkSnapshot())
			a.shapeCounts()
			a.typeCounts()
			a.tagCounts()
			b.ReportAllocs()
			for b.Loop() {
				if mode == "Cold" {
					a.Toolbar = toolbarState{}
				}
				if len(a.toolbarExtensions()) != 1 || snapshotHasKind(a, model.KindVideo) || snapshotHasKind(a, model.KindLive) {
					b.Fatal("文件类型样本错误")
				}
				if shapes := a.shapeCounts(); shapes[catalog.ShapeLandscape] != 100_000 {
					b.Fatal("形状计数错误")
				}
				if extensions, _ := a.typeCounts(); extensions["jpg"] != 100_000 {
					b.Fatal("文件类型计数错误")
				}
				if tags := a.tagCounts(); tags["ref"] != 100_000 {
					b.Fatal("标签计数错误")
				}
			}
		})
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
	tt.ClickAt(slider.Bounds.X+slider.Bounds.W*.9, slider.Bounds.Y+slider.Bounds.H/2)
	if a.thumbnailScale() <= before {
		t.Fatal("浮层中的缩略图滑块没有生效")
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
	tt.Advance(time.Second)
	if _, ok := tt.AccessByRole(ui.RoleButton, a.T("thumbnail_size")); ok {
		t.Fatal("列表视图仍显示缩略图入口")
	}
	a.ViewMode = 0
	tt.SetSize(1560, 640)
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleButton, a.T("thumbnail_size")); ok {
		t.Fatal("宽窗口仍只显示缩略图入口")
	}
	if slider, ok = tt.AccessByRole(ui.RoleSlider, a.T("thumbnail_size")); !ok || slider.Disabled {
		t.Fatal("宽窗口没有完整滑块")
	}
}

// 列表视图用不上缩略图大小：刻度尺收起淡出，回到瀑布流时展开，中间帧在两者之间。
func TestToolbarThumbnailRulerHidesInList(t *testing.T) {
	a := assetViewFixture()
	a.Preferences = DefaultPreferences()
	a.Preferences.Theme = "light"
	tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); a.toolbar(c) }, 1500, MainToolbarHeight)
	tt.Advance(time.Second)
	slider, ok := tt.AccessByRole(ui.RoleSlider, a.T("thumbnail_size"))
	if !ok {
		t.Fatal("瀑布流没有显示刻度尺")
	}
	region := image.Rect(int(slider.Bounds.X), 0, int(slider.Bounds.X+slider.Bounds.W), MainToolbarHeight)
	crop := func() []byte {
		img := tt.Image()
		var out []byte
		for y := region.Min.Y; y < region.Max.Y; y++ {
			out = append(out, img.Pix[img.PixOffset(region.Min.X, y):img.PixOffset(region.Max.X, y)]...)
		}
		return out
	}
	shown := crop()
	a.ViewMode = 2
	tt.Frame()
	tt.Advance(time.Second)
	if _, ok := tt.AccessByRole(ui.RoleSlider, a.T("thumbnail_size")); ok {
		t.Fatal("列表视图仍显示刻度尺")
	}
	hidden := crop()
	a.ViewMode = 0
	tt.Frame()
	tt.Advance(80 * time.Millisecond)
	middle := crop()
	tt.Advance(time.Second)
	if bytes.Equal(middle, shown) || bytes.Equal(middle, hidden) {
		t.Fatal("刻度尺展开没有过渡")
	}
	if !bytes.Equal(crop(), shown) {
		t.Fatal("刻度尺展开后没有回到原样")
	}
}

// 回收站页右上角是「清空回收站」，不是导入；回收站空时一个按钮都不放。
func TestTrashToolbarEmptiesInsteadOfImport(t *testing.T) {
	a := toolbarTestApp()
	a.Query.Scope = "trash"
	a.Refine()
	tt := toolbarTester(a)
	if tt.HasText(a.T("import")) || tt.HasText(a.T("empty_trash_short")) {
		t.Fatal("回收站是空的，不应出现导入和清空回收站")
	}

	a.Snapshot.Assets[0].Trashed = true
	a.Index = catalog.New(a.Snapshot)
	a.Refine()
	tt.Frame()
	toolbarClick(t, tt, a.T("empty_trash_short"))
	if !a.ConfirmOpen || a.ConfirmTitle != a.T("empty_trash") || a.Busy {
		t.Fatal("清空回收站没有先要求确认")
	}
}

// 筛选里勾着的标签被删掉后，筛选条件跟着去掉，列表恢复。
func TestDeletedTagLeavesFilter(t *testing.T) {
	a := toolbarTestApp()
	tt := toolbarTester(a)
	a.Query.TagIDs = []string{"ref"}
	a.Refine()
	if len(a.Visible) != 1 {
		t.Fatalf("标签筛选结果：%v", a.Visible)
	}
	snapshot := a.Snapshot
	snapshot.Tags = nil
	snapshot.Assets = slices.Clone(snapshot.Assets)
	snapshot.Assets[0].TagIDs = nil
	a.applySnapshot(snapshot)
	tt.Frame()
	if len(a.Query.TagIDs) != 0 || len(a.Visible) != 3 {
		t.Fatalf("删掉的标签还留在筛选里：tags=%v visible=%d", a.Query.TagIDs, len(a.Visible))
	}
}

// 筛选生效时按钮带条件数，悬停出现 ×，点一下清空全部筛选。
func TestToolbarFilterShowsCountAndClearsOnHover(t *testing.T) {
	a := toolbarTestApp()
	a.Query.ColorFamilies = []string{"red"}
	a.Query.Extensions = []string{"jpg"}
	a.Refine()
	tt := toolbarTester(a)
	label := a.T("filter_count", 2)
	box, ok := tt.Find(label)
	if !ok {
		t.Fatalf("筛选按钮没有显示条件数 %q：%v", label, tt.Texts())
	}
	tt.Move(box.X+box.W/2, box.Y+box.H/2)
	tt.Advance(time.Second)
	var clear ui.Rect
	for _, item := range tt.AccessAll() {
		if item.Name == a.T("clear_filters") && item.Bounds.W == toolbarClearSize {
			clear = item.Bounds
		}
	}
	if clear.W == 0 {
		t.Fatal("悬停在生效的筛选按钮上没有出现 ×")
	}
	tt.ClickAt(clear.X+clear.W/2, clear.Y+clear.H/2)
	if a.filterCount() != 0 || len(a.Visible) != 3 {
		t.Fatalf("点 × 没有清空筛选：%+v", a.Query)
	}
	if !tt.HasText(a.T("filter")) || tt.HasText(label) {
		t.Fatal("清空后筛选按钮没有恢复")
	}
}

// 没有条件时弹层里不放「清除筛选」；颜色悬停提示带数量，没有素材的类型、标签、形状不列出来。
func TestFilterPanelOnlyListsOptionsWithAssets(t *testing.T) {
	a := toolbarTestApp()
	a.Query.Search = "Beach"
	a.Refine()
	a.Toolbar.FilterOpen = true
	tt := toolbarTester(a)
	if tt.HasText(a.T("clear_filters")) {
		t.Fatal("没有筛选条件时不该显示清除筛选")
	}
	if !tt.HasText("PNG") || tt.HasText("JPG") {
		t.Fatalf("文件类型应只列当前位置有素材的：%v", tt.Texts())
	}
	if tt.HasText("Reference") || tt.HasText(a.T("filter_tags")) {
		t.Fatalf("当前位置没有带标签的素材，标签段应整段不显示：%v", tt.Texts())
	}
	if tt.HasText(a.T("filter_shapes")) {
		t.Fatal("一个能选的形状都没有时形状段应整段不显示")
	}
	if _, ok := tt.Access(a.T("color_count", a.T("color_blue"), 1)); !ok {
		t.Fatal("有素材的颜色没有带数量")
	}
	if _, ok := tt.Access(a.T("color_count", a.T("color_red"), 0)); !ok {
		t.Fatal("没有素材的颜色没有写成 0")
	}
}

func TestFilterTagSearchWithoutResultSaysNotFound(t *testing.T) {
	a := toolbarTestApp()
	snapshot := a.Snapshot
	snapshot.Assets = slices.Clone(snapshot.Assets)
	snapshot.Tags = nil
	for i := range popTagSearchFrom + 1 {
		id := fmt.Sprintf("t%d", i)
		snapshot.Tags = append(snapshot.Tags, model.Tag{ID: id, Name: "Tag" + id})
		snapshot.Assets[0].TagIDs = append(snapshot.Assets[0].TagIDs, id)
	}
	a.applySnapshot(snapshot)
	a.Toolbar.FilterOpen = true
	a.Toolbar.TagSearch = "zzz"
	tt := toolbarTester(a)
	if !tt.HasText(a.T("no_tag_results", "zzz")) || tt.HasText(a.T("no_tags")) {
		t.Fatalf("标签搜索无结果的文案不对：%v", tt.Texts())
	}
}

// 最近使用和随机浏览里排序没有作用，不放排序按钮；别的位置有。
func TestToolbarSortHiddenWhereItDoesNothing(t *testing.T) {
	for scope, want := range map[string]bool{"all": true, "recent": false, "random": false} {
		a := toolbarTestApp()
		a.Query.Scope = scope
		a.Refine()
		tt := toolbarTester(a)
		if _, ok := tt.AccessByRole(ui.RoleButton, a.T("sort")); ok != want {
			t.Fatalf("%s 位置的排序按钮：有=%v，应有=%v", scope, ok, want)
		}
	}
	a := toolbarTestApp()
	if a.sortChanged() {
		t.Fatal("默认排序（添加时间、新的在前）不算改过")
	}
	a.Query.Descending = false
	if !a.sortChanged() {
		t.Fatal("方向改了应算改过")
	}
	a.Query.Descending, a.Query.Sort = true, "name"
	if !a.sortChanged() {
		t.Fatal("字段改了应算改过")
	}
}

// 图库里一个素材都没有时，只留标题、搜索和导入。
func TestToolbarBareWhenLibraryEmpty(t *testing.T) {
	a := toolbarTestApp()
	a.applySnapshot(model.Snapshot{})
	tt := toolbarTester(a)
	for _, key := range []string{"filter", "sort", "view_masonry", "thumbnail_size"} {
		if _, ok := tt.Access(a.T(key)); ok {
			t.Fatalf("空图库的工具栏不该有 %s", key)
		}
	}
	if _, ok := tt.AccessByRole(ui.RoleSlider, a.T("thumbnail_size")); ok {
		t.Fatal("空图库的工具栏不该有缩略图滑块")
	}
	if !tt.HasText(a.T("import")) {
		t.Fatal("空图库的工具栏要留导入")
	}
	if _, ok := tt.AccessByRole(ui.RoleTextField, a.T("search_assets")); !ok {
		t.Fatal("空图库的工具栏要留搜索")
	}
}

// 回收站：选中素材时出现品牌色「恢复」，「清空回收站」是它左边的普通按钮。
func TestTrashToolbarRestoreAppearsWithSelection(t *testing.T) {
	a := toolbarTestApp()
	a.Snapshot.Assets[0].Trashed = true
	a.Index = catalog.New(a.Snapshot)
	a.Query.Scope = "trash"
	a.Refine()
	tt := toolbarTester(a)
	if _, ok := tt.AccessByRole(ui.RoleButton, a.T("restore")); ok {
		t.Fatal("没选中素材时不该显示恢复")
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, a.T("empty_trash_short")); !ok {
		t.Fatal("回收站有素材时要有清空回收站")
	}
	a.Selected.Add("a")
	tt.Frame()
	tt.Advance(time.Second)
	restore, ok := tt.AccessByRole(ui.RoleButton, a.T("restore"))
	if !ok {
		t.Fatal("选中回收站素材后没有恢复按钮")
	}
	empty, _ := tt.AccessByRole(ui.RoleButton, a.T("empty_trash_short"))
	if empty.Bounds.X+empty.Bounds.W > restore.Bounds.X {
		t.Fatalf("清空回收站应在恢复左边：%+v %+v", empty.Bounds, restore.Bounds)
	}
}

// 只有图标时悬停提示带快捷键；有搜索词时窄栏的放大镜带小圆点，提示写搜索词。
func TestToolbarIconTipsCarryShortcutAndSearchTerm(t *testing.T) {
	a := toolbarTestApp()
	if tip := a.toolbarTip(a.T("filter"), "filter"); tip != a.T("filter")+"  "+displayShortcut(a.Shortcut("filter")) {
		t.Fatalf("提示没有带快捷键：%q", tip)
	}
	a.Query.Search = "alpine"
	tt := ui.NewTester(func(c *ui.Context) {
		a.Preferences.Theme = "light"
		a.applyTheme(c)
		ui.Column(c).Fill().Children(func() { a.toolbar(c); ui.Spacer(c) })
	}, 330, 200)
	tt.Frame()
	box, ok := tt.Find(a.T("search_assets"))
	if !ok {
		t.Fatal("窄栏没有搜索按钮")
	}
	tt.Move(box.X+box.W/2, box.Y+box.H/2)
	tt.Advance(time.Second)
	if !tt.HasText(a.T("search_with_term", "alpine")) {
		t.Fatalf("悬停提示没有写搜索词：%v", tt.Texts())
	}
}

// 主搜索框 Esc：有内容先清空并留在输入框，没内容再失焦。
func TestToolbarSearchEscapeClearsThenBlurs(t *testing.T) {
	a := toolbarTestApp()
	tt := toolbarTester(a)
	toolbarClick(t, tt, a.T("search_assets"))
	tt.Type("alpine")
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.Query.Search != "" || len(a.Visible) != 3 {
		t.Fatalf("第一次 Esc 没有清空搜索：%q", a.Query.Search)
	}
	if active, _ := tt.TextInputActive(); !active {
		t.Fatal("第一次 Esc 后应留在输入框")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if active, _ := tt.TextInputActive(); active {
		t.Fatal("没内容时 Esc 没有让输入框失焦")
	}
}

// 窗口变窄、刻度尺收成按钮的过程中，缩略图那一组只缩宽度，不能从旧位置滑过搜索框。
func TestToolbarThumbnailGroupNeverOverlapsSearch(t *testing.T) {
	a := toolbarTestApp()
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Fill().Children(func() {
			ui.Box(c).Width(236)
			ui.Column(c).Grow(1).FillHeight().MinWidth(0).Children(func() { a.toolbar(c); ui.Spacer(c) })
		})
	}, 1500, 200)
	tt.Advance(time.Second)
	tt.SetSize(1000, 200)
	for range 30 {
		tt.Frame()
		tt.Advance(16 * time.Millisecond)
		search, ok := tt.AccessByRole(ui.RoleTextField, a.T("search_assets"))
		if !ok {
			t.Fatal("没有搜索框")
		}
		for _, item := range tt.AccessAll() {
			if item.Name == a.T("thumbnail_size") && item.Bounds.W > 0 && item.Bounds.X < search.Bounds.X+search.Bounds.W-1 && item.Bounds.X+item.Bounds.W > search.Bounds.X+1 {
				t.Fatalf("缩略图控件压在搜索框上：%+v 搜索框 %+v", item.Bounds, search.Bounds)
			}
		}
	}
}

// 正在忙时导入灰掉，悬停写原因。
func TestToolbarImportBusyTooltip(t *testing.T) {
	a := toolbarTestApp()
	a.Busy = true
	tt := toolbarTester(a)
	box, ok := tt.Find(a.T("import"))
	if !ok {
		t.Fatal("没有导入按钮")
	}
	tt.Move(box.X+box.W/2, box.Y+box.H/2)
	tt.Advance(time.Second)
	if !tt.HasText(a.T("import_busy")) {
		t.Fatalf("灰掉的导入没有写原因：%v", tt.Texts())
	}
}
