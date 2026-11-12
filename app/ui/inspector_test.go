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
	a := &App{Texts: i18n.Catalog{Language: "en"}, Visible: selected, InspectorVisible: true}
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
	tt := ui.NewTester(a.inspector, 280, 640)
	if !tt.HasText(a.T("no_selection")) || tt.HasText(a.T("exif_metadata")) {
		t.Fatalf("空选状态错误: %v", tt.Texts())
	}
	if tt.HasText(a.T("inspector")) {
		t.Fatal("检查器仍显示重复标题")
	}
	if _, ok := tt.Find(a.T("close_inspector")); ok {
		t.Fatal("检查器仍显示重复关闭按钮")
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
