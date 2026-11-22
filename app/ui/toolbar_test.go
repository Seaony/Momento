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
	// 弹层不收起，直接再点当前字段反转方向。
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

func TestToolbarFormatsAndCountsFollowSnapshotAndScope(t *testing.T) {
	a := toolbarTestApp()
	snapshot := a.Snapshot
	snapshot.Assets = slices.Clone(snapshot.Assets)
	snapshot.Assets[0].Width, snapshot.Assets[0].Height = 1920, 1080
	snapshot.Assets[1].Width, snapshot.Assets[1].Height = 1080, 1920
	snapshot.Assets[2].Width, snapshot.Assets[2].Height = 1920, 1080
	snapshot.Assets[2].Trashed, snapshot.Assets[2].Kind = true, model.KindVideo
	a.applySnapshot(snapshot)
	if !a.libraryHasKind(model.KindVideo) || a.libraryHasKind(model.KindLive) {
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
		a.libraryHasKind(model.KindVideo) || !a.libraryHasKind(model.KindLive) {
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
				if len(a.toolbarExtensions()) != 1 || a.libraryHasKind(model.KindVideo) || a.libraryHasKind(model.KindLive) {
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
	tt.SetSize(1500, 640)
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

// 回收站页右上角是「清空回收站」，不是导入；回收站空时不可点。
func TestTrashToolbarEmptiesInsteadOfImport(t *testing.T) {
	a := toolbarTestApp()
	a.Query.Scope = "trash"
	a.Refine()
	tt := toolbarTester(a)
	if _, ok := tt.Find(a.T("import")); ok {
		t.Fatal("回收站页不应出现导入")
	}
	if err := tt.Click(a.T("empty_trash")); err == nil && a.ConfirmOpen {
		t.Fatal("回收站是空的，清空按钮不应能点")
	}

	a.Snapshot.Assets[0].Trashed = true
	a.Index = catalog.New(a.Snapshot)
	a.Refine()
	tt.Frame()
	toolbarClick(t, tt, a.T("empty_trash"))
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
