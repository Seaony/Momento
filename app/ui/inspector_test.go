package ui

import (
	"errors"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/i18n"
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
	if field, ok := tt.AccessByRole(ui.RoleTextField, a.T("title")); !ok || field.Value != "Landscape.jpg" {
		t.Fatal("检查器没有显示可编辑的素材名")
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
			filename, ok := tt.AccessByRole(ui.RoleTextField, a.T("title"))
			if !ok || filename.Value != a.Visible[0].Name || filename.Bounds.X < 0 || filename.Bounds.X+filename.Bounds.W > 280 || filename.Bounds.Y < button.Y+button.H {
				t.Fatalf("预览下的文件名没有完整保留，或与收起按钮重叠：%+v，%+v", filename, button)
			}
		})
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
			if got := inspectorEXIFValue(values, "resolution"); got != tc.want {
				t.Fatalf("分辨率显示 %q，期望 %q", got, tc.want)
			}
		})
	}
}

func TestInspectorEXIFOmitsDuplicateDimensions(t *testing.T) {
	asset := model.Asset{ID: "a", Width: 2400, Height: 1600, EXIF: map[string]string{
		"pixelWidth": "2400", "pixelHeight": "1600", "dpiWidth": "300", "dpiHeight": "72",
	}}
	a := inspectorTestApp(asset)
	tt := ui.NewTester(a.inspector, 280, 640)
	if !tt.HasText("2400 × 1600") || !tt.HasText("300 × 72 dpi") || !tt.HasText(a.T("exif_resolution")) {
		t.Fatalf("缺少尺寸或合并的分辨率: %v", tt.Texts())
	}
	for _, key := range []string{"exif_pixelWidth", "exif_pixelHeight", "exif_dpiWidth", "exif_dpiHeight"} {
		if tt.HasText(a.T(key)) {
			t.Fatalf("仍显示重复字段 %s", key)
		}
	}
	if hasInspectorEXIF(map[string]string{"pixelWidth": "2400", "pixelHeight": "1600"}) {
		t.Fatal("只有已展示过的像素尺寸时不应再显示 EXIF 区")
	}
}

func TestInspectorMultipleSelectionWithoutTags(t *testing.T) {
	a := inspectorTestApp(model.Asset{ID: "a"}, model.Asset{ID: "b"}, model.Asset{ID: "c"})
	tt := ui.NewTester(a.inspector, 280, 640)
	if !tt.HasText(a.T("selected_no_tags")) || tt.HasText(a.T("no_tags")) {
		t.Fatalf("多选空标签文案不明确: %v", tt.Texts())
	}
}

func TestInspectorSingleSelectionAndCopy(t *testing.T) {
	asset := model.Asset{
		ID: "a", Name: "Reference", Extension: "jpg", Width: 2400, Height: 1600, Size: 1258291,
		CreatedAt: time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC),
		Colors:    []model.Color{{Hex: "#A1B2C3", Proportion: 1}},
		EXIF:      map[string]string{"cameraModel": "Test Camera", "lensModel": ""},
	}
	a := inspectorTestApp(asset)
	tt := ui.NewTester(a.inspector, 280, 800)
	for _, text := range []string{"JPG", "2400 × 1600", "1.2 MB", "Test Camera"} {
		if !tt.HasText(text) {
			t.Errorf("缺少 %q: %v", text, tt.Texts())
		}
	}
	if tt.HasText(a.T("exif_lensModel")) {
		t.Fatal("显示了没有值的 EXIF 字段")
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

func TestInspectorMultipleSelectionAndSearch(t *testing.T) {
	a := inspectorTestApp(
		model.Asset{ID: "a", TagIDs: []string{"landscape"}, FolderIDs: []string{"child"}},
		model.Asset{ID: "b", TagIDs: []string{"landscape"}},
		model.Asset{ID: "c"},
	)
	a.Snapshot.Tags = []model.Tag{{ID: "landscape", Name: "Landscape"}, {ID: "portrait", Name: "Portrait"}}
	a.Snapshot.Folders = []model.Folder{{ID: "parent", Name: "References"}, {ID: "child", Name: "Photos", ParentID: "parent"}}
	tt := ui.NewTester(a.inspector, 280, 800)
	for _, text := range []string{a.T("selected_assets", 3), "2/3", "1/3", "References / Photos"} {
		if !tt.HasText(text) {
			t.Errorf("缺少 %q: %v", text, tt.Texts())
		}
	}
	if err := tt.Click(a.T("add_tags")); err != nil {
		t.Fatal(err)
	}
	if !a.Inspector.TagsOpen || !tt.HasText("Portrait") {
		t.Fatalf("标签选择器未打开: %v", tt.Texts())
	}
	if err := tt.Click(a.T("search_tags")); err != nil {
		t.Fatal(err)
	}
	tt.Type("land")
	if tt.HasText("Portrait") || !tt.HasText("Landscape (2/3)") {
		t.Fatalf("标签搜索未筛选: %v", tt.Texts())
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
	button, ok := tt.Find(a.T("add_folders"))
	if !ok || button.Y < MainToolbarHeight || button.Y+button.H > 640 {
		t.Fatalf("滚到底后文件夹操作仍在窗口之外：%+v", button)
	}
}

func TestInspectorLocalizedStructureAndMetadataWrapping(t *testing.T) {
	for _, language := range []string{"en", "ja"} {
		t.Run(language, func(t *testing.T) {
			asset := model.Asset{ID: "a", Name: "Landscape reference.jpg", Extension: "jpg", Width: 2400, Height: 1600, Size: 1258291,
				Note: "Two lines\nReference notes", SourceURL: "https://example.com/reference", TagIDs: []string{"tag"}, FolderIDs: []string{"folder"},
				Colors: []model.Color{{Hex: "#A1B2C3"}}}
			a := inspectorTestApp(asset)
			a.Texts.Language, a.Preferences.Language = language, language
			a.Snapshot.Tags = []model.Tag{{ID: "tag", Name: "Landscape references"}}
			a.Snapshot.Folders = []model.Folder{{ID: "folder", Name: "Design references"}}
			tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); a.inspector(c) }, 280, 1000)
			tt.SetLocale(a.Texts.Locale())
			tt.Frame()
			preview, _ := tt.Find(a.T("image_preview"))
			filename, _ := tt.AccessByRole(ui.RoleTextField, a.T("title"))
			summary, _ := tt.Find("2400 × 1600 · JPG")
			swatch, _ := tt.Find(a.T("copy_color", "#A1B2C3"))
			info, _ := tt.Find(a.T("file_size"))
			tag, _ := tt.Find("Landscape references")
			folder, _ := tt.Find("Design references")
			note, _ := tt.AccessByRole(ui.RoleTextField, a.T("asset_note"))
			source, _ := tt.AccessByRole(ui.RoleTextField, a.T("source_url"))
			for i, bounds := range []ui.Rect{preview, filename.Bounds, summary, swatch, info, tag, folder, note.Bounds, source.Bounds} {
				if bounds.W <= 0 || bounds.H <= 0 || bounds.X < 0 || bounds.X+bounds.W > 280 {
					t.Fatalf("第%d个检查器区域没有显示或横向溢出：%+v", i, bounds)
				}
			}
			if preview.Y >= filename.Bounds.Y || filename.Bounds.Y >= summary.Y || summary.Y >= swatch.Y || swatch.Y >= info.Y || info.Y >= tag.Y || tag.Y >= folder.Y || folder.Y >= note.Bounds.Y || note.Bounds.Y >= source.Bounds.Y {
				t.Fatal("检查器没有按预览、文件名、尺寸格式、色板、信息、标签、文件夹、备注、来源排列")
			}
			if filename.Value != asset.Name || note.Value != asset.Note || source.Value != asset.SourceURL {
				t.Fatal("重排丢失了文件名、备注或来源的完整值")
			}
			asset.EXIF = map[string]string{"exposureProgram": "Aperture Priority"}
			a.Visible = []model.Asset{asset}
			var needHeight float32
			metadata := ui.NewTester(func(c *ui.Context) {
				a.applyTheme(c)
				a.inspector(c)
				_, needHeight = c.MeasureText(space48*2, ui.Span{Text: a.T("exif_exposureProgram"), Font: c.Theme().Font, Size: fontSecondary, Lang: language})
				needHeight *= float32(fontSecondary) / max(c.Theme().FontSize, fontSecondary)
			}, 280, 1000)
			metadata.SetLocale(a.Texts.Locale())
			metadata.Frame()
			label, ok := metadata.Find(a.T("exif_exposureProgram"))
			if !ok || label.H+1 < needHeight {
				t.Fatalf("长元数据标签没有按可用宽度换行：%+v，需要高度%f", label, needHeight)
			}
		})
	}
}

func TestInspectorSourceDisplayKeepsCompleteEditableValue(t *testing.T) {
	original := "https://example.com/references/landscape/early-morning-light?collection=mountain-study&image=warm-mist-2026"
	a := inspectorTestApp(model.Asset{ID: "a", Name: "Landscape.jpg", SourceURL: original})
	tt := ui.NewTester(a.inspector, 280, 1100)
	field, ok := tt.AccessByRole(ui.RoleTextField, a.T("source_url"))
	if !ok || field.Value != original {
		t.Fatal("未编辑时来源字段丢失了完整地址")
	}
	field, ok = tt.AccessByRole(ui.RoleTextField, a.T("source_url"))
	if !ok {
		t.Fatal("来源输入框不存在")
	}
	tt.ClickAt(field.Bounds.X+field.Bounds.W/2, field.Bounds.Y+field.Bounds.H/2)
	if !tt.Focused(a.T("source_url")) {
		t.Fatal("省略文字层阻挡了来源字段聚焦")
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(ui.Cmd, ui.KeyC)
	if tt.Clipboard() != original {
		t.Fatal("编辑来源时无法选中并复制完整地址")
	}
	replacement := original + "&language=ja"
	tt.Type(replacement)
	if a.Inspector.SourceURL != replacement {
		t.Fatalf("来源编辑值被省略显示截短：%q", a.Inspector.SourceURL)
	}
	if err := tt.Click(a.T("title")); err != nil {
		t.Fatal(err)
	}
	draft, dirty := a.inspectorDraft()
	if !dirty || !draft.Details || draft.Source != replacement {
		t.Fatalf("失焦后保存草稿未保留完整来源：%+v", draft)
	}
	field, ok = tt.AccessByRole(ui.RoleTextField, a.T("source_url"))
	if !ok {
		t.Fatal("来源输入框不存在")
	}
	tt.ClickAt(field.Bounds.X+field.Bounds.W/2, field.Bounds.Y+field.Bounds.H/2)
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(ui.Cmd, ui.KeyC)
	if tt.Clipboard() != replacement {
		t.Fatal("再次编辑来源时完整地址没有恢复")
	}
}

func TestInspectorEmptyNoteAndSourcePlaceholdersFit(t *testing.T) {
	for _, language := range []string{"zh-CN", "en", "ja"} {
		t.Run(language, func(t *testing.T) {
			a := inspectorTestApp(model.Asset{ID: "a", Name: "Landscape.jpg"})
			a.Texts.Language, a.Preferences.Language = language, language
			var noteWidth, sourceWidth float32
			tt := ui.NewTester(func(c *ui.Context) {
				a.applyTheme(c)
				a.inspector(c)
				noteWidth, _ = c.MeasureText(0, ui.Span{Text: a.T("asset_note_placeholder"), Font: c.Theme().Font, Size: fontBody, Lang: language})
				sourceWidth, _ = c.MeasureText(0, ui.Span{Text: a.T("source_url_placeholder"), Font: c.Theme().Font, Size: fontBody, Lang: language})
			}, 280, 1000)
			tt.SetLocale(a.Texts.Locale())
			tt.Frame()
			for _, check := range []struct {
				label, placeholder string
				width              float32
			}{{a.T("asset_note"), a.T("asset_note_placeholder"), noteWidth}, {a.T("source_url"), a.T("source_url_placeholder"), sourceWidth}} {
				field, ok := tt.AccessByRole(ui.RoleTextField, check.label)
				if !ok || field.Value != "" || field.Bounds.W < check.width+2*space12 || field.Placeholder != check.placeholder {
					t.Fatalf("空字段占位没有完整显示：%s，%+v，需要宽度%f", check.placeholder, field, check.width)
				}
			}
			if a.Inspector.SourceURL != "" || a.Inspector.Note != "" || a.InspectorEditsPending() {
				t.Fatal("占位文字被写进了草稿")
			}
		})
	}
}
