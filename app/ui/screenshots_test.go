package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

const finalScreenshotDirectory = "/tmp/momento-titlebar-settings-preview"

type screenshotRecord struct {
	File             string      `json:"file"`
	State            string      `json:"state"`
	Language         string      `json:"language"`
	Appearance       string      `json:"appearance"`
	Width            int         `json:"width"`
	Height           int         `json:"height"`
	Assets           int         `json:"assets"`
	Visible          int         `json:"visible"`
	Platform         string      `json:"platform"`
	TitleBar         ui.TitleBar `json:"simulatedNativeControls"`
	MenuPresentation string      `json:"menuPresentation,omitempty"`
	SettingsTabs     []ui.Rect   `json:"settingsTabs,omitempty"`
}

// 显式运行时才生成截图，样例库只写入测试临时目录。
func TestStateScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_SCREENSHOTS=1 生成离线界面截图")
	}
	fixture := newScreenshotFixture(t)
	dir := finalScreenshotDirectory
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	states := []string{
		"welcome", "empty", "small", "single", "multiple", "large", "search-empty", "importing", "trash",
		"preview", "settings", "export", "new-library", "error",
		"grid", "list", "single-details", "single-details-bottom", "tag-picker", "folder-picker", "filter", "sort", "tags",
		"search-results", "search-cleared", "grid-selected", "list-selected", "masonry-hover", "grid-hover", "list-hover",
		"single-details-mixed-dpi", "single-details-mixed-dpi-bottom", "filter-selected", "filter-color-hover",
	}
	for _, key := range []string{"toggle_sidebar", "filter", "sort", "view_masonry", "view_grid", "view_list", "import", "toggle_inspector"} {
		states = append(states, "toolbar-hover-"+key)
	}
	var records []screenshotRecord
	for _, language := range []string{"zh-CN", "en", "ja"} {
		for _, state := range states {
			for _, appearance := range []string{"light", "dark"} {
				for _, height := range []int{800, 640} {
					width := 1100
					if state == "settings" {
						if height == 640 {
							continue
						}
						width, height = 560, settingsTestApp(language, appearance).settingsHeight(UpdateCheckStatus{State: UpdateIdle})
					}
					name := fmt.Sprintf("%s-%s-%s-%dx%d", state, language, appearance, width, height)
					t.Run(name, func(t *testing.T) {
						a := fixture.app(t, state, language, appearance)
						view := a.View
						bar := ui.TitleBar{Height: MainToolbarHeight, Left: 80, Bottom: 37}
						if state == "settings" {
							a = settingsTestApp(language, appearance)
							view = func(c *ui.Context) { a.settingsView(c, UpdateCheckStatus{State: UpdateIdle}) }
							bar = ui.TitleBar{Height: 36, Left: 80, Bottom: 25}
						}
						tt := ui.NewTester(view, width, height)
						tt.SetLocale(a.Texts.Locale())
						tt.SetTitleBar(bar)
						tt.SetWindowActive(true)
						var tabs []ui.Rect
						if state == "settings" {
							height = settleFinalSettingsScreenshot(t, a, tt, UpdateCheckStatus{State: UpdateIdle})
							tabs = finalSettingsTabBounds(t, a, tt)
						}
						if strings.HasSuffix(state, "-bottom") {
							tt.Scroll(960, float32(height-100), 0, 2000)
						}
						prepareScreenshotState(t, a, tt, state)
						checkScreenshotState(t, a, tt, state)
						file := fmt.Sprintf("%s-%s-%s-%dx%d.png", state, language, appearance, width, height)
						writeScreenshot(t, filepath.Join(dir, file), tt)
						if a.Thumbnails != nil && len(a.Thumbnails.pending) != 0 {
							t.Fatal("截图仍在异步加载缩略图")
						}
						records = append(records, screenshotRecord{File: file, State: state, Language: language, Appearance: appearance, Width: width, Height: height, Assets: len(a.Snapshot.Assets), Visible: len(a.Visible), Platform: "macos", TitleBar: bar, SettingsTabs: tabs})
					})
				}
			}
		}
	}
	records = append(records, renderFinalSettingsScreenshots(t, dir)...)
	records = append(records, renderFinalTitlebarScreenshots(t, fixture, dir)...)
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
}

type screenshotFixture struct {
	root     string
	manifest model.Manifest
	snapshot model.Snapshot
	bitmaps  map[string]*ui.Bitmap
}

func newScreenshotFixture(t *testing.T) screenshotFixture {
	t.Helper()
	root := filepath.Join(t.TempDir(), "Momento Samples.momento")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/sample-library", "-path", root, "-count", "14")
	cmd.Dir = ".."
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("生成全新截图样例库：%v\n%s", err, output)
	}
	lib, err := library.Open(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := lib.Snapshot(ctx)
	closeErr := lib.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("读取截图样例：%v，关闭：%v", err, closeErr)
	}
	if len(snapshot.Assets) != 14 || len(snapshot.Folders) != 0 || len(snapshot.Tags) != 0 {
		t.Fatal("小图库必须有14张素材、没有文件夹和标签")
	}
	slices.SortFunc(snapshot.Assets, func(a, b model.Asset) int { return a.CreatedAt.Compare(b.CreatedAt) })
	formats := make(map[string]int)
	fixture := screenshotFixture{root: root, manifest: lib.Manifest, snapshot: snapshot, bitmaps: make(map[string]*ui.Bitmap)}
	for _, asset := range snapshot.Assets {
		formats[asset.Extension]++
		bitmap, _, err := decodeThumbnail(filepath.Join(root, "thumbnails", asset.ID+".png"))
		if err != nil {
			t.Fatal(err)
		}
		fixture.bitmaps[asset.ID] = bitmap
	}
	if formats["jpg"] != 10 || formats["png"] != 4 {
		t.Fatalf("小图库格式数错误：%v", formats)
	}
	return fixture
}

func (fixture screenshotFixture) app(t *testing.T, state, language, appearance string) *App {
	t.Helper()
	snapshot := model.Snapshot{Assets: slices.Clone(fixture.snapshot.Assets)}
	sources := make(map[string]*ui.Bitmap, len(fixture.bitmaps))
	for id, bitmap := range fixture.bitmaps {
		sources[id] = bitmap
	}
	if state == "empty" || state == "welcome" {
		snapshot.Assets = nil
	}
	if state == "large" {
		for i := range 30 {
			snapshot.Folders = append(snapshot.Folders, model.Folder{ID: fmt.Sprintf("folder-%02d", i), Name: fmt.Sprintf("参考素材 %02d", i+1), SortIndex: i})
		}
		for i := range 80 {
			snapshot.Tags = append(snapshot.Tags, model.Tag{ID: fmt.Sprintf("tag-%02d", i), Name: fmt.Sprintf("Tag %02d", i+1)})
		}
		snapshot.Assets = make([]model.Asset, 12000)
		for i := range snapshot.Assets {
			asset := fixture.snapshot.Assets[i%len(fixture.snapshot.Assets)]
			source := fixture.bitmaps[asset.ID]
			asset.ID, asset.Name = fmt.Sprintf("%064x", i), fmt.Sprintf("Reference %05d", i+1)
			asset.CreatedAt = time.Date(2026, 1, 1, 12, 0, i, 0, time.UTC)
			asset.FolderIDs = []string{snapshot.Folders[i%30].ID}
			asset.TagIDs = []string{snapshot.Tags[i%80].ID}
			asset.Favorite = i%17 == 0
			snapshot.Assets[i] = asset
			sources[asset.ID] = source
		}
	}
	if state == "trash" {
		for i := range 3 {
			snapshot.Assets[i].Trashed = true
		}
	}
	if state == "tag-picker" || state == "folder-picker" || strings.HasPrefix(state, "filter") || state == "tags" {
		snapshot.Folders = []model.Folder{{ID: "design", Name: "Design references"}, {ID: "ui", Name: "UI layouts", ParentID: "design"}}
		snapshot.Tags = []model.Tag{{ID: "reference", Name: "Reference"}, {ID: "landscape", Name: "Landscape 山水"}, {ID: "inspiration", Name: "Inspiration"}, {ID: "typography", Name: "Typography"}}
		for i := range snapshot.Assets {
			snapshot.Assets[i].TagIDs = []string{snapshot.Tags[i%len(snapshot.Tags)].ID}
			snapshot.Assets[i].FolderIDs = []string{snapshot.Folders[i%len(snapshot.Folders)].ID}
		}
	}
	detailsID := ""
	if strings.HasPrefix(state, "single-details") {
		for i := range snapshot.Assets {
			asset := &snapshot.Assets[i]
			if asset.Extension != "jpg" || len(asset.Colors) != 8 {
				continue
			}
			detailsID = asset.ID
			asset.Name = "Landscape reference with morning light and layered mountains"
			asset.SourceURL = "https://example.com/references/landscape?collection=morning-light"
			asset.EXIF = map[string]string{
				"fileModifiedAt": "2026-01-01 12:34:56", "contentCreatedAt": "2025-12-12 08:42:10",
				"pixelWidth": fmt.Sprint(asset.Width), "pixelHeight": fmt.Sprint(asset.Height),
				"dpiWidth": "300", "dpiHeight": "300", "colorModel": "RGB", "profileName": "Display P3",
				"cameraMake": "FUJIFILM", "cameraModel": "FUJIFILM X-T5", "lensModel": "XF16-55mmF2.8 R LM WR",
				"exposureTime": "1/250 s", "focalLength": "35 mm", "isoSpeedRatings": "200", "flashFired": "false",
				"fNumber": "f/2.8", "exposureProgram": "Aperture priority", "meteringMode": "Multi-segment", "whiteBalance": "Auto",
			}
			if strings.Contains(state, "mixed-dpi") {
				asset.EXIF["dpiHeight"] = "72"
			}
			break
		}
	}
	a := &App{
		Library:        &library.Library{Root: fixture.root, Manifest: fixture.manifest},
		Texts:          i18n.Catalog{Language: language, SystemLanguage: language},
		Preferences:    Preferences{Language: language, Appearance: appearance},
		SidebarVisible: true, InspectorVisible: true, SidebarWidth: 236,
		Query: catalog.Query{Sort: "added", Descending: true}, selectedIndex: -1, anchor: -1,
		ctx: context.Background(), cancel: func() {},
	}
	a.Thumbnails = newThumbnailCache(func(func()) { t.Error("截图不允许异步补入尚未准备的缩略图") })
	a.applySnapshot(snapshot)
	if state == "welcome" {
		a.Library = nil
	}
	if state == "search-empty" {
		a.Query.Search = "moonlight poster"
		a.Refine()
	}
	if state == "search-results" || state == "search-cleared" {
		a.Query.Search = a.Visible[0].Name
		a.Refine()
	}
	if state == "trash" {
		a.Query.Scope = "trash"
		a.Refine()
	}
	if state == "single" {
		for _, asset := range a.Visible {
			if asset.Extension == "jpg" && len(asset.Colors) == 8 {
				a.Selected.Add(asset.ID)
				break
			}
		}
		if a.Selected.Len() != 1 {
			t.Fatal("单选状态需要一张带8色色板的JPEG样本")
		}
	}
	if state == "grid-selected" || state == "list-selected" || state == "masonry-hover" || state == "grid-hover" || state == "list-hover" {
		a.Selected.Add(a.Visible[0].ID)
	}
	if detailsID != "" {
		a.Selected.Add(detailsID)
	}
	if state == "multiple" || state == "tag-picker" || state == "folder-picker" {
		for _, asset := range a.Visible[:3] {
			a.Selected.Add(asset.ID)
		}
	}
	if state == "grid" || state == "grid-selected" || state == "grid-hover" {
		a.ViewMode = 1
	}
	if state == "list" || state == "list-selected" || state == "list-hover" {
		a.ViewMode = 2
	}
	if state == "tag-picker" || state == "folder-picker" {
		a.Inspector.LibraryID = a.Library.Manifest.LibraryID
		a.Inspector.TagsOpen, a.Inspector.FoldersOpen = state == "tag-picker", state == "folder-picker"
	}
	if strings.HasPrefix(state, "filter") {
		a.Toolbar.FilterOpen = true
	}
	if state == "filter-selected" || state == "filter-color-hover" {
		for _, family := range toolbarFamilies {
			a.Query.ColorFamilies = append(a.Query.ColorFamilies, family.ID)
		}
		a.Refine()
	}
	if state == "sort" {
		a.Toolbar.SortOpen = true
	}
	if state == "tags" {
		a.TagPage = true
	}
	if state == "importing" {
		a.Busy = true
		a.Progress = &model.ImportProgress{Stage: "importing", Total: 120, Processed: 37, Imported: 32, Skipped: 5, CurrentFile: "design-reference-037.jpg"}
	}
	if state == "preview" {
		asset := a.Visible[1]
		img, _, _, _, err := assets.PrepareImage(filepath.Join(fixture.root, filepath.FromSlash(asset.RelativePath)))
		if err != nil {
			t.Fatalf("读取预览原文件：%v", err)
		}
		a.Selected.Add(asset.ID)
		a.Preview.ID, a.Preview.Open = asset.ID, true
		a.Preview.target(fixture.root, asset.ID)
		a.Preview.Bitmap = ui.NewBitmap(img)
	}
	if state == "export" {
		a.Selected.Add(a.Visible[0].ID)
		a.ExportOpen = true
		a.Dialog.ExportWasOpen = true
		a.Dialog.Format, a.Dialog.Quality = model.ExportJPEG, 0.9
	}
	if state == "new-library" {
		a.NewLibraryOpen = true
	}
	if state == "error" {
		a.Error = a.T("error_import_busy")
	}
	// 只预装首屏及邻近区域，仍遵守正式缓存的数量和内存上限。
	preload := a.Visible
	if state == "search-cleared" {
		preload = snapshot.Assets
	}
	for _, asset := range preload[:min(64, len(preload))] {
		bitmap := sources[asset.ID]
		if bitmap == nil {
			t.Fatalf("样例素材没有真实缩略图：%s", asset.ID)
		}
		width, height := bitmap.Size()
		bytes := int64(width*height*4) * 4 / 3
		key := thumbnailKey{root: filepath.Clean(fixture.root), id: asset.ID}
		a.Thumbnails.entries[key] = a.Thumbnails.order.PushFront(thumbnailEntry{key: key, bitmap: bitmap, bytes: bytes})
		a.Thumbnails.bytes += bytes
	}
	if a.Thumbnails.bytes > thumbnailByteLimit || len(a.Thumbnails.entries) > thumbnailItemLimit {
		t.Fatal("截图预装超过正式缩略图缓存上限")
	}
	return a
}

func prepareScreenshotState(t *testing.T, a *App, tt *ui.Tester, state string) {
	t.Helper()
	if state == "search-cleared" {
		if err := tt.Click(a.T("search_assets")); err != nil {
			t.Fatal(err)
		}
		tt.Key(ui.Cmd, ui.KeyA)
		tt.Key(0, ui.KeyBackspace)
	}
	label := ""
	if key, ok := strings.CutPrefix(state, "toolbar-hover-"); ok {
		label = a.T(key)
	}
	if state == "filter-color-hover" {
		label = a.T("color_gray")
	}
	if state == "masonry-hover" || state == "grid-hover" || state == "list-hover" {
		label = a.Visible[1].Name
	}
	if label != "" {
		box, ok := tt.Find(label)
		if !ok || box.W <= 0 || box.H <= 0 {
			t.Fatalf("找不到悬停目标 %s", label)
		}
		tt.Move(box.X+box.W/2, box.Y+box.H/2)
		tt.Advance(time.Second)
	}
}

func checkScreenshotState(t *testing.T, a *App, tt *ui.Tester, state string) {
	t.Helper()
	expect := func(text string) {
		if !tt.HasText(text) {
			t.Fatalf("状态 %s 缺少 %q，实际文案：%v", state, text, tt.Texts())
		}
	}
	switch state {
	case "welcome":
		expect(a.T("welcome"))
		expect(a.T("new_library"))
		expect(a.T("open_library"))
		if tt.HasText(a.T("inspector")) || tt.HasText(a.T("all")) {
			t.Fatal("欢迎页不应出现侧栏或检查器")
		}
	case "empty":
		expect(a.T("drop_images"))
		expect(a.T("install_extension"))
		expect(a.T("no_selection"))
	case "single", "grid-selected", "list-selected", "masonry-hover", "grid-hover", "list-hover":
		expect(a.T("title"))
		expect(a.T("file_type"))
		selected := a.SelectedAssets()[0]
		expect(strings.ToUpper(selected.Extension))
		for _, color := range selected.Colors {
			if _, ok := tt.Find(a.T("copy_color", color.Hex)); !ok {
				t.Fatalf("色板缺少 %s", color.Hex)
			}
		}
	case "multiple":
		expect(a.T("selected_assets", 3))
	case "large":
		expect("12,000")
		if len(a.Snapshot.Folders) != 30 || len(a.Snapshot.Tags) != 80 {
			t.Fatal("大图库缺少30文件夹和80标签")
		}
		if len(tt.AccessAll()) > 1500 {
			t.Fatal("大图库一次构建了过多界面元素")
		}
	case "search-empty":
		expect(a.T("search_empty", a.Query.Search))
		expect(a.T("result_count", "0"))
	case "search-results":
		if len(a.Visible) != 1 {
			t.Fatalf("搜索应只匹配一张素材，实际 %d", len(a.Visible))
		}
		expect(a.T("result_count", "1"))
	case "search-cleared":
		if a.Query.Search != "" || len(a.Visible) != 14 || tt.HasText(a.T("result_count", "14")) {
			t.Fatal("清空搜索后应恢复14张素材并隐藏工具栏数量")
		}
	case "importing":
		expect("37 / 120")
		expect(a.T("import_counts", 32, 5))
		expect("design-reference-037.jpg")
	case "trash":
		if len(a.Visible) != 3 || a.Index.Counts().Trash != 3 {
			t.Fatal("回收站必须只显示3张素材")
		}
	case "preview":
		expect(a.T("preview_position", 2, 14))
		if a.Preview.Bitmap == nil || a.Preview.Loading || a.Preview.targetRoot != a.Library.Root || a.Preview.targetID != a.Preview.ID {
			t.Fatal("预览必须展示已经解码的真实原图")
		}
		for _, key := range []string{"previous_asset", "next_asset", "close_preview"} {
			if _, ok := tt.Find(a.T(key)); !ok {
				t.Fatalf("预览缺少本地化控件：%s", key)
			}
		}
	case "settings":
		for _, key := range []string{"settings_general", "settings_extension", "settings_about", "launch_at_login", "appearance", "language", "automatic_updates", "check_updates"} {
			expect(a.T(key))
		}
	case "export":
		for _, key := range []string{"export", "export_format", "export_original", "export_jpeg", "export_png", "jpeg_quality"} {
			expect(a.T(key))
		}
		expect(a.T("export_count", 1))
		expect("90%")
	case "new-library":
		for _, key := range []string{"new_library", "new_library_detail", "library_name", "location", "choose_location", "create_library"} {
			expect(a.T(key))
		}
	case "error":
		for _, key := range []string{"operation_failed", "error_import_busy", "log_location", "copy_error", "close"} {
			expect(a.T(key))
		}
	case "single-details", "single-details-bottom", "single-details-mixed-dpi", "single-details-mixed-dpi-bottom":
		expect("example.com")
		expect(a.T("exif_metadata"))
		selected := a.SelectedAssets()
		if len(selected) != 1 || len(selected[0].EXIF) != 19 {
			t.Fatal("长检查器必须单选并带19个EXIF字段")
		}
		fields := 0
		for _, key := range inspectorEXIFKeys {
			if value := inspectorEXIFValue(selected[0].EXIF, key); value != "" {
				fields++
				expect(a.T("exif_" + key))
			}
		}
		if fields != 16 {
			t.Fatalf("19项原始EXIF去除重复尺寸并合并DPI后应显示16项，实际%d", fields)
		}
		for _, key := range []string{"pixelWidth", "pixelHeight", "dpiWidth", "dpiHeight"} {
			if tt.HasText(a.T("exif_" + key)) {
				t.Fatalf("检查器仍重复显示%s", key)
			}
		}
		resolution := "300 dpi"
		if strings.Contains(state, "mixed-dpi") {
			resolution = "300 × 72 dpi"
		}
		expect(resolution)
	case "tag-picker":
		expect(a.T("search_tags"))
		expect(a.T("create_tag"))
		expect("Landscape 山水 (1/3)")
	case "folder-picker":
		expect(a.T("search_folders"))
		expect(a.T("create_folder"))
		expect("Design references / UI layouts (2/3)")
	case "filter", "filter-selected", "filter-color-hover":
		for _, key := range []string{"filter_colors", "filter_tags", "filter_types"} {
			expect(a.T(key))
		}
		if state != "filter" {
			for _, family := range toolbarFamilies {
				access, ok := tt.Access(a.T("color_" + family.ID))
				if !ok || !access.Checked {
					t.Fatalf("颜色族%s没有显示选中状态", family.ID)
				}
			}
		}
		if state == "filter-color-hover" {
			expect(a.T("color_gray"))
		}
	case "sort":
		for _, key := range []string{"sort_added", "sort_name", "sort_size", "sort_ascending", "sort_descending"} {
			expect(a.T(key))
		}
	case "tags":
		expect(a.T("new_tag"))
		expect(a.T("tag_total", "4"))
		expect("Landscape 山水")
	default:
		expect(a.T("no_selection"))
		if counts := a.Index.Counts(); counts.All != 14 || counts.Uncategorized != 14 || counts.Untagged != 14 {
			t.Fatalf("小图库侧栏计数错误：%+v", counts)
		}
	}
	if key, ok := strings.CutPrefix(state, "toolbar-hover-"); ok {
		expect(a.T(key))
	}
	if state != "welcome" {
		if slices.Contains(tt.Texts(), a.T("inspector")) {
			t.Fatal("检查器仍显示重复标题")
		}
		if _, ok := tt.Find(a.T("close_inspector")); ok {
			t.Fatal("检查器仍有重复关闭按钮")
		}
		if a.Query.Search == "" && tt.HasText(a.T("result_count", toolbarNumber(len(a.Visible)))) {
			t.Fatal("非搜索状态仍显示重复结果数")
		}
	}
	for _, text := range tt.Texts() {
		if strings.Contains(text, "%!") {
			t.Fatalf("界面出现格式化错误：%s", text)
		}
		if strings.Contains(text, "⟦") || strings.Contains(text, "⟧") {
			t.Fatalf("界面出现未翻译的文案键：%s", text)
		}
	}
	if state == "small" || state == "trash" {
		if err := tt.RightClick(a.Visible[0].Name); err != nil {
			t.Fatal(err)
		}
		menu := tt.Menu()
		keys := []string{"preview", "export", "reveal", "move_to_trash", "refresh_thumbnail", "reanalyze_colors"}
		if state == "trash" {
			keys = []string{"preview", "export", "reveal", "restore", "delete_permanently"}
			if slices.Contains(menu, a.T("move_to_trash")) {
				t.Fatal("回收站菜单不应再次移到回收站")
			}
		}
		for _, key := range keys {
			if !slices.Contains(menu, a.T(key)) {
				t.Fatalf("状态 %s 的右键菜单缺少 %s：%v", state, key, menu)
			}
		}
		tt.CloseMenu()
	}
}

func writeScreenshot(t *testing.T, path string, tt *ui.Tester) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, tt.Image()); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}
