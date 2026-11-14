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
	if !tt.HasText("Landscape.jpg") {
		t.Fatal("检查器顶部没有显示素材名")
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
			heading, ok := tt.Find(a.Visible[0].Name)
			if !ok || heading.X+heading.W > button.X {
				t.Fatalf("单行标题与收起按钮重叠：%+v，%+v", heading, button)
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
