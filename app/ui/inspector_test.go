package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func inspectorTestApp(selected ...model.Asset) *App {
	a := &App{Preferences: DefaultPreferences(), Texts: i18n.Catalog{Language: "en"}, Visible: selected, InspectorVisible: true}
	for _, asset := range selected {
		a.Selected.Add(asset.ID)
	}
	return a
}

func TestInspectorCopyFailure(t *testing.T) {
	a := inspectorTestApp(model.Asset{ID: "a", Colors: []model.Color{{Hex: "#A1B2C3"}}})
	tt := ui.NewTester(a.inspector, 280, 800)
	tt.FailClipboard(errors.New("clipboard unavailable"))
	if err := tt.Click(a.T("copy_color", "#A1B2C3")); err != nil {
		t.Fatal(err)
	}
	if a.Error == "" || a.Notice != "" {
		t.Fatalf("剪贴板失败没有显示错误或误报成功: error=%q notice=%q", a.Error, a.Notice)
	}
}

func TestInspectorNoSelection(t *testing.T) {
	a := inspectorTestApp()
	a.InspectorVisible = false
	a.RevealInspector()
	a.ToggleInspector()
	if a.InspectorVisible || a.InspectorAvailable() {
		t.Fatal("没有选择时仍能打开检查器")
	}
	tt := ui.NewTester(a.inspector, 280, 640)
	if len(tt.Texts()) != 0 || len(tt.AccessAll()) != 0 {
		t.Fatalf("没有选择时仍渲染检查器：%v", tt.Texts())
	}
}

func TestInspectorFollowsSelectionAndReopensSameAsset(t *testing.T) {
	a := inspectorTestApp(model.Asset{ID: "a", Name: "Landscape.jpg"}, model.Asset{ID: "b", Name: "Portrait.jpg"})
	a.Selected.Clear()
	a.InspectorVisible = false
	a.selectAsset("a", 0, 0)
	if !a.InspectorVisible || !a.InspectorAvailable() {
		t.Fatal("单选后没有展开检查器")
	}
	tt := ui.NewTester(a.inspector, 280, 640)
	if !tt.HasText("Landscape.jpg") {
		t.Fatal("检查器没有显示素材名")
	}
	if _, ok := tt.AccessByRole(ui.RoleTextField, a.T("edit_title")); ok {
		t.Fatal("标题没点就已经是输入框")
	}
	if err := tt.Click(a.T("close_inspector")); err != nil {
		t.Fatal(err)
	}
	if a.InspectorVisible || a.Selected.Len() != 1 {
		t.Fatal("收起检查器没有保留选择")
	}
	a.selectAsset("a", 0, 0)
	if !a.InspectorVisible {
		t.Fatal("再次点击同一素材没有重开检查器")
	}
	a.selectAsset("b", 1, ui.Cmd)
	tt.Frame()
	if !a.InspectorVisible || a.Selected.Len() != 2 || !tt.HasText(a.T("selected_assets", 2)) {
		t.Fatal("多选检查器没有展开或标题不符")
	}
	a.selectAsset("a", 0, ui.Cmd)
	a.selectAsset("b", 1, ui.Cmd)
	a.RevealInspector()
	a.ToggleInspector()
	tt.Frame()
	if a.InspectorAvailable() || a.InspectorVisible || len(tt.Texts()) != 0 {
		t.Fatal("取消全部选择后仍可打开检查器")
	}
}

func TestInspectorLocalizedHeaderKeepsCollapseVisible(t *testing.T) {
	for _, language := range []string{"en", "ja"} {
		t.Run(language, func(t *testing.T) {
			a := inspectorTestApp(model.Asset{ID: "a", Name: "A very long landscape reference name that must leave room for the collapse button.jpg"})
			a.Texts.Language = language
			tt := ui.NewTester(a.inspector, 280, 640)
			button, ok := tt.Find(a.T("close_inspector"))
			if !ok || button.X < 0 || button.X+button.W > 280 || button.Y < 0 || button.Y+button.H > 44 {
				t.Fatalf("长素材名挤出了收起按钮：%+v", button)
			}
			title, ok := tt.Find(a.T("edit_title"))
			if !ok || title.X < 0 || title.X+title.W > 280 || title.Y < button.Y+button.H || !tt.HasText(a.Visible[0].Name) {
				t.Fatalf("预览下的标题没有完整保留，或与收起按钮重叠：%+v，%+v", title, button)
			}
		})
	}
}

// 标题、链接、备注平时是文字，点了才是输入框；回车或失焦保存，Esc 放弃。
func TestInspectorInlineEditing(t *testing.T) {
	a := inspectorTestApp(model.Asset{ID: "a", Name: "Reference", Note: "Original", SourceURL: "https://example.com/path"})
	tt := ui.NewTester(a.inspector, 280, 1000)
	if err := tt.Click(a.T("edit_title")); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	field, ok := tt.AccessByRole(ui.RoleTextField, a.T("edit_title"))
	if !ok || field.Value != "Reference" || !tt.Focused(a.T("edit_title")) {
		t.Fatalf("点标题没有变成已聚焦的输入框：%+v", field)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("Renamed")
	// 光标在中间按回车也只是保存，不插换行、不拆词。
	tt.Key(0, ui.KeyLeft)
	tt.Key(0, ui.KeyLeft)
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.Inspector.Editing != "" || a.Inspector.Title != "Renamed" {
		t.Fatalf("回车没有结束标题编辑：editing=%q title=%q", a.Inspector.Editing, a.Inspector.Title)
	}
	if draft, dirty := a.inspectorDraft(); !dirty || !draft.Rename || draft.Name != "Renamed" {
		t.Fatalf("新标题没有进入保存草稿：%+v", draft)
	}
	if _, ok := tt.AccessByRole(ui.RoleTextField, a.T("edit_title")); ok || !tt.HasText("Renamed") {
		t.Fatal("保存后标题没有回到文字显示")
	}

	if err := tt.Click(a.T("edit_title")); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyBackspace)
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	// 没有图库时改名不会真正写入，所以退回的是原名。
	if a.Inspector.Title != a.Inspector.SourceTitle || a.Inspector.Editing != "" {
		t.Fatalf("清空标题不应保存成空名：%q", a.Inspector.Title)
	}

	if err := tt.Click(a.T("asset_note")); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("Two lines\nSecond line")
	if a.Inspector.Note != "Two lines\nSecond line" || a.Inspector.Editing != "note" {
		t.Fatalf("备注没有保留多行：%q", a.Inspector.Note)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.Inspector.Note != "Original" || a.Inspector.Editing != "" {
		t.Fatalf("Esc 没有放弃备注修改：%q", a.Inspector.Note)
	}

	link, ok := tt.Find(a.T("open_source_url"))
	if !ok {
		t.Fatal("有链接时没有显示")
	}
	tt.Move(link.X+link.W/2, link.Y+link.H/2)
	if err := tt.Click(a.T("edit_link")); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.Focused(a.T("source_url")) {
		t.Fatal("点编辑后链接没有进入编辑")
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyBackspace)
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.Inspector.Editing != "" || a.Inspector.SourceURL != "" {
		t.Fatalf("清空链接回车没有保存：%q", a.Inspector.SourceURL)
	}
	if tt.HasText(a.T("link")) {
		t.Fatal("没有链接时仍显示链接一行")
	}
}

func TestInspectorResolutionCombinesAxes(t *testing.T) {
	for _, tc := range []struct {
		name, width, height, want string
	}{
		{"相同", "300", "300", "300 dpi"},
		{"不同", "300", "72", "300 × 72 dpi"},
		{"只有水平", "300", "", "300 dpi"},
		{"只有垂直", "", "72", "72 dpi"},
		{"没有分辨率", "", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := map[string]string{"dpiWidth": tc.width, "dpiHeight": tc.height}
			if got := inspectorEXIFValue(i18n.Catalog{Language: "en"}, values, "resolution"); got != tc.want {
				t.Fatalf("分辨率显示 %q，期望 %q", got, tc.want)
			}
		})
	}
}

// 图库里存的原始值要换成人看的格式，不能直接露出时间戳和枚举号。
func TestInspectorEXIFFormatting(t *testing.T) {
	values := map[string]string{
		"contentCreatedAt": "2026-10-07T15:55:29.858569817Z", "exposureTime": "0.0004", "focalLength": "50",
		"fNumber": "4", "exposureProgram": "3", "meteringMode": "5", "whiteBalance": "9", "flashFired": "false",
	}
	en, zh := i18n.Catalog{Language: "en"}, i18n.Catalog{Language: "zh-CN"}
	for _, tc := range []struct {
		texts     i18n.Catalog
		key, want string
	}{
		{en, "contentCreatedAt", "Oct 7, 2026 at 3:55 PM"},
		{zh, "contentCreatedAt", "2026年10月7日 15:55"},
		{en, "exposureTime", "1/2500 s"},
		{en, "focalLength", "50 mm"},
		{en, "fNumber", "f/4"},
		{en, "exposureProgram", "Aperture Priority"},
		{zh, "meteringMode", "矩阵测光"},
		{en, "whiteBalance", "9"},
		{zh, "flashFired", "未闪光"},
	} {
		if got := inspectorEXIFValue(tc.texts, values, tc.key); got != tc.want {
			t.Errorf("%s 显示 %q，期望 %q", tc.key, got, tc.want)
		}
	}
	local := time.Date(2026, 1, 2, 9, 5, 0, 0, time.Local)
	if got := inspectorEXIFValue(en, map[string]string{"fileModifiedAt": local.UTC().Format(time.RFC3339Nano)}, "fileModifiedAt"); got != "Jan 2, 2026 at 9:05 AM" {
		t.Errorf("文件时间没有换成本地时间：%q", got)
	}
}

func TestInspectorEXIFOmitsDuplicateDimensions(t *testing.T) {
	asset := model.Asset{ID: "a", Width: 2400, Height: 1600, EXIF: map[string]string{
		"pixelWidth": "2400", "pixelHeight": "1600", "dpiWidth": "300", "dpiHeight": "72",
	}}
	a := inspectorTestApp(asset)
	tt := ui.NewTester(a.inspector, 280, 800)
	if !tt.HasText("2400 × 1600") || !tt.HasText("300 × 72 dpi") || !tt.HasText(a.T("exif_resolution")) {
		t.Fatalf("缺少尺寸或合并的分辨率: %v", tt.Texts())
	}
	for _, key := range []string{"exif_pixelWidth", "exif_pixelHeight", "exif_dpiWidth", "exif_dpiHeight"} {
		if tt.HasText(a.T(key)) {
			t.Fatalf("仍显示重复字段 %s", key)
		}
	}
	if hasInspectorEXIF(a.Texts, map[string]string{"pixelWidth": "2400", "pixelHeight": "1600"}) {
		t.Fatal("只有已展示过的像素尺寸时不应再显示 EXIF 区")
	}
}

func TestInspectorSingleSelectionAndCopy(t *testing.T) {
	asset := model.Asset{
		ID: "a", Name: "Reference", Extension: "jpg", Width: 2400, Height: 1600, Size: 1258291,
		CreatedAt: time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC),
		Colors:    []model.Color{{Hex: "#A1B2C3", Proportion: .123}, {Hex: "#112233", Proportion: .05}},
		EXIF:      map[string]string{"cameraModel": "Test Camera", "lensModel": ""},
	}
	a := inspectorTestApp(asset)
	tt := ui.NewTester(a.inspector, 280, 800)
	for _, text := range []string{"JPG", "2400 × 1600", "1.2 MB", "Test Camera", a.Texts.DateTime(asset.CreatedAt.Local())} {
		if !tt.HasText(text) {
			t.Errorf("缺少 %q: %v", text, tt.Texts())
		}
	}
	if tt.HasText(a.T("exif_lensModel")) {
		t.Fatal("显示了没有值的 EXIF 字段")
	}
	swatch, ok := tt.Find(a.T("copy_color", "#A1B2C3"))
	if !ok {
		t.Fatal("没有色块")
	}
	tt.Move(swatch.X+swatch.W/2, swatch.Y+swatch.H/2)
	tt.Advance(300 * time.Millisecond)
	if !tt.HasText("#A1B2C3 (12.3%)") {
		t.Fatalf("悬停色块没有显示色值和占比: %v", tt.Texts())
	}
	if err := tt.Click(a.T("copy_color", "#A1B2C3")); err != nil {
		t.Fatal(err)
	}
	if got := tt.Clipboard(); got != "#A1B2C3" {
		t.Fatalf("复制的色值为 %q", got)
	}
	if a.Notice != a.T("copied_color", "#A1B2C3") {
		t.Fatalf("复制后没有反馈: %q", a.Notice)
	}
}

func TestInspectorMultipleSelectionAndPickers(t *testing.T) {
	a := inspectorTestApp(
		model.Asset{ID: "a", TagIDs: []string{"landscape"}, FolderIDs: []string{"child"}},
		model.Asset{ID: "b", TagIDs: []string{"landscape"}},
		model.Asset{ID: "c"},
	)
	a.Snapshot.Tags = []model.Tag{{ID: "landscape", Name: "Landscape"}, {ID: "portrait", Name: "Portrait"}}
	a.Snapshot.Folders = []model.Folder{{ID: "parent", Name: "References"}, {ID: "child", Name: "Photos", ParentID: "parent"}}
	tt := ui.NewTester(a.inspector, 280, 800)
	for _, text := range []string{a.T("selected_assets", 3), "2/3", "1/3", "Photos"} {
		if !tt.HasText(text) {
			t.Errorf("缺少 %q: %v", text, tt.Texts())
		}
	}
	if err := tt.Click(a.T("add_tags")); err != nil {
		t.Fatal(err)
	}
	if !a.Inspector.TagsOpen || !tt.HasText("Portrait") || !tt.HasText(a.T("all_tags")+" (2)") {
		t.Fatalf("标签选择器未打开: %v", tt.Texts())
	}
	tt.Type("land")
	if tt.HasText("Portrait") || !tt.HasText("Landscape (2/3)") {
		t.Fatalf("标签搜索未筛选: %v", tt.Texts())
	}
	tt.Type("x")
	if !tt.HasText(a.T("create_named", "landx")) || tt.HasText(a.T("no_tags")) {
		t.Fatalf("搜不到时应只给创建行: %v", tt.Texts())
	}
	// 回车：搜不到就创建（测试里没有图库，只看选择器收起、搜索词清空）。
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.Inspector.TagsOpen || a.Inspector.TagSearch != "" {
		t.Fatalf("回车没有创建并收起标签选择器：open=%v search=%q", a.Inspector.TagsOpen, a.Inspector.TagSearch)
	}
	if err := tt.Click(a.T("add_folders")); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("References") || tt.HasText("Photos (1/3)") {
		t.Fatalf("文件夹选择器应只显示收起的顶层: %v", tt.Texts())
	}
	if err := tt.Click(a.T("toggle_folder", "References")); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Photos (1/3)") {
		t.Fatalf("展开父文件夹后没有显示子文件夹: %v", tt.Texts())
	}
}

func TestInspectorScrollReachesMemberships(t *testing.T) {
	asset := model.Asset{ID: "a", Name: "Reference", SourceURL: "https://example.com/reference", EXIF: map[string]string{}}
	for _, key := range inspectorEXIFKeys {
		asset.EXIF[key] = "Metadata value"
	}
	a := inspectorTestApp(asset)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			ui.Box(c).Height(MainToolbarHeight).Shrink(0)
			a.inspector(c)
		})
	}, 280, 640)
	tt.Scroll(180, 500, 0, 5000)
	label, ok := tt.Find(a.T("exif_creator"))
	if !ok || label.Y < MainToolbarHeight || label.Y+label.H > 640 {
		t.Fatalf("滚到底后最后一行 EXIF 仍在窗口之外：%+v", label)
	}
}

func TestInspectorLocalizedStructureAndMetadataWrapping(t *testing.T) {
	for _, language := range []string{"en", "ja"} {
		t.Run(language, func(t *testing.T) {
			asset := model.Asset{ID: "a", Name: "Landscape reference.jpg", Extension: "jpg", Width: 2400, Height: 1600, Size: 1258291,
				Note: "Two lines\nReference notes", SourceURL: "https://example.com/reference", TagIDs: []string{"tag"}, FolderIDs: []string{"folder"},
				Colors: []model.Color{{Hex: "#A1B2C3"}}, EXIF: map[string]string{"exposureProgram": "3"}}
			a := inspectorTestApp(asset)
			a.Texts.Language, a.Preferences.Language = language, language
			a.Snapshot.Tags = []model.Tag{{ID: "tag", Name: "Landscape references"}}
			a.Snapshot.Folders = []model.Folder{{ID: "folder", Name: "Design references"}}
			var needHeight float32
			tt := ui.NewTester(func(c *ui.Context) {
				a.applyTheme(c)
				a.inspector(c)
				_, needHeight = c.MeasureText(0, ui.Span{Text: a.T("exif_exposureProgram"), Font: c.Theme().Font, Size: fontSecondary, Lang: language})
			}, 280, 1200)
			tt.SetLocale(a.Texts.Locale())
			tt.Frame()
			var order []ui.Rect
			for _, label := range []string{a.T("image_preview"), a.T("copy_color", "#A1B2C3"), a.T("edit_title"), a.T("file_size"), a.T("open_source_url"),
				"Landscape references", "Design references", a.T("asset_note"), a.T("exif_exposureProgram")} {
				bounds, ok := tt.Find(label)
				if !ok || bounds.W <= 0 || bounds.H <= 0 || bounds.X < 0 || bounds.X+bounds.W > 280 {
					t.Fatalf("检查器区域 %q 没有显示或横向溢出：%+v", label, bounds)
				}
				order = append(order, bounds)
			}
			for i := 1; i < len(order); i++ {
				if order[i].Y <= order[i-1].Y {
					t.Fatalf("检查器没有按预览、色板、标题、详细信息、链接、标签、文件夹、备注、EXIF 排列（第%d项）", i)
				}
			}
			if !tt.HasText(asset.Note) || !tt.HasText(asset.SourceURL) {
				t.Fatal("备注或链接没有完整显示")
			}
			label, _ := tt.Find(a.T("exif_exposureProgram"))
			if label.H < 1.8*needHeight*float32(fontSecondary)/max(float32(fontBody), fontSecondary) {
				t.Fatalf("长元数据标签没有在 74 宽里换行：%+v，单行高%f", label, needHeight)
			}
			if strings.Contains(strings.Join(tt.Texts(), " "), "⟦") {
				t.Fatal("出现未翻译的文案")
			}
		})
	}
}

func TestInspectorEXIFCollapses(t *testing.T) {
	a := inspectorTestApp(model.Asset{ID: "a", Name: "Reference", EXIF: map[string]string{"cameraModel": "Test Camera", "fileCreatedAt": "2026-01-02T03:04:05Z"}})
	tt := ui.NewTester(a.inspector, 280, 800)
	if !tt.HasText("Test Camera") || !tt.HasText("Created") {
		t.Fatalf("EXIF 默认应展开，创建时间用短标签: %v", tt.Texts())
	}
	if err := tt.Click(a.T("exif")); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if !a.Preferences.EXIFCollapsed || tt.HasText("Test Camera") {
		t.Fatal("点 EXIF 标题没有收起")
	}
	if err := tt.Click(a.T("exif")); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if a.Preferences.EXIFCollapsed || !tt.HasText("Test Camera") {
		t.Fatal("再点没有展开")
	}
}

// 检查器里创建文件夹先弹出新建对话框选图标，建好后放进所选素材。
func TestInspectorCreateFolderOpensIconDialog(t *testing.T) {
	a := inspectorTestApp(model.Asset{ID: "a"}, model.Asset{ID: "b"})
	a.Library = &library.Library{}
	tt := ui.NewTester(func(c *ui.Context) { a.inspector(c); a.folderDialog(c) }, 900, 800)
	if err := tt.Click(a.T("add_folders")); err != nil {
		t.Fatal(err)
	}
	tt.Type("Moodboard")
	if err := tt.Click(a.T("create_named", "Moodboard")); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	state := a.Dialog.Folder
	if !state.Open || state.Name != "Moodboard" || state.ID != "" || len(state.Assets) != 2 || a.Inspector.FoldersOpen {
		t.Fatalf("没有弹出带名称和素材的新建文件夹对话框：%+v", state)
	}
	if err := tt.Click(a.T("folder_icon")); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(a.T("folder_icon_star")); err != nil {
		t.Fatal(err)
	}
	if a.Dialog.Folder.Icon != "star" || a.Dialog.Folder.PickerOpen {
		t.Fatalf("选图标后没有记下或没有收起：%+v", a.Dialog.Folder)
	}
}

// 文件夹对话框开着时换了图库，草稿要作废，不能写进新图库。
func TestFolderDialogClosesWhenLibraryChanges(t *testing.T) {
	a := inspectorTestApp(model.Asset{ID: "a"})
	a.Library = &library.Library{}
	a.ShowNewFolder("", "Moodboard", []string{"a"})
	tt := ui.NewTester(func(c *ui.Context) { a.folderDialog(c) }, 900, 800)
	tt.Frame()
	if !a.Dialog.Folder.Open {
		t.Fatal("对话框没有打开")
	}
	a.generation++
	tt.Frame()
	if a.Dialog.Folder.Open || a.Dialog.Folder.Name != "" {
		t.Fatalf("换图库后对话框还留着旧草稿：%+v", a.Dialog.Folder)
	}
	a.ShowNewFolder("", "Moodboard", nil)
	a.generation++
	a.saveFolderDialog("Moodboard")
	if a.Busy {
		t.Fatal("换图库后保存仍然开始写入")
	}
}
