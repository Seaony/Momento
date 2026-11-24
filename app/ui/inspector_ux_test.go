package ui

import (
	"testing"

	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

// 文本输入框的确认按钮文字由调用方定；没指定用「保存」，关掉后不留给下一次。
func TestAskWithConfirmLabel(t *testing.T) {
	a := &App{}
	name := ""
	a.AskWith("hashtag", "New Tag", "", a.T("create"), func(value string) { name = value })
	tt := ui.NewTester(a.dialogs, 1100, 640)
	if _, ok := tt.AccessByRole(ui.RoleButton, a.T("create")); !ok {
		t.Fatal("确认按钮没有用调用方给的文字")
	}
	tt.Type("Travel")
	tt.Key(0, ui.KeyEnter)
	if name != "Travel" || a.PromptOpen {
		t.Fatalf("提交结果：%q open=%v", name, a.PromptOpen)
	}
	a.Ask("pencil-simple", "Rename", "x", func(string) {})
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleButton, a.T("save")); !ok {
		t.Fatal("普通 Ask 应回到「保存」")
	}
}

func TestErrorDialogHasOpenLogsButton(t *testing.T) {
	a := &App{Error: "disk full"}
	tt := ui.NewTester(a.dialogs, 1100, 640)
	if _, ok := tt.AccessByRole(ui.RoleButton, a.T("open_logs")); !ok {
		t.Fatalf("错误框没有「打开日志文件夹」：%v", tt.Texts())
	}
}

// 全是视频时导出对话框不出现格式、质量、尺寸；图视频混选、转格式时说一句视频保持原格式。
func TestExportDialogByContent(t *testing.T) {
	a := inspectorTestApp(model.Asset{ID: "v1", Kind: model.KindVideo, Extension: "mp4"}, model.Asset{ID: "v2", Kind: model.KindVideo, Extension: "mov"})
	a.ExportOpen = true
	a.Dialog.ExportReady, a.Dialog.Format = true, model.ExportJPEG
	tt := ui.NewTester(a.dialogs, 1100, 760)
	if !tt.HasText(a.T("export_selected_count", 2)) || tt.HasText(a.T("export_format")) || tt.HasText(a.T("export_video_original")) {
		t.Fatalf("全是视频时不该出现格式选项：%v", tt.Texts())
	}
	if options, problem := a.exportOptions(); problem != "" || options.Format != model.ExportOriginal {
		t.Fatalf("全是视频时记着的 JPEG 不能影响导出：%+v %q", options, problem)
	}

	a = inspectorTestApp(model.Asset{ID: "i", Kind: model.KindImage, Extension: "png", Rotation: 90}, model.Asset{ID: "v", Kind: model.KindVideo, Extension: "mp4"})
	a.ExportOpen = true
	a.Dialog.ExportReady, a.Dialog.Format = true, model.ExportOriginal
	tt = ui.NewTester(a.dialogs, 1100, 760)
	if !tt.HasText(a.T("export_format")) || tt.HasText(a.T("export_video_original")) || !tt.HasText(a.T("export_rotation_metadata_hint")) {
		t.Fatalf("原文件格式：应显示旋转提示、不显示视频说明：%v", tt.Texts())
	}
	a.Dialog.Format = model.ExportPNG
	tt.Frame()
	if !tt.HasText(a.T("export_video_original")) || tt.HasText(a.T("export_rotation_metadata_hint")) {
		t.Fatalf("转格式：应显示视频说明、不显示旋转提示：%v", tt.Texts())
	}
}

func TestInspectorSingleAddSourceLink(t *testing.T) {
	a := inspectorTestApp(model.Asset{ID: "a", Name: "Reference"})
	tt := ui.NewTester(a.inspector, 280, 800)
	if !tt.HasText(a.T("add_source_link")) {
		t.Fatalf("没有链接时应显示「添加来源链接」：%v", tt.Texts())
	}
	if err := tt.Click(a.T("source_url")); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.Inspector.Editing != "source" || !tt.Focused(a.T("source_url")) {
		t.Fatalf("点占位没有进入链接编辑：%q", a.Inspector.Editing)
	}
	tt.Type("https://example.com/a")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.Inspector.SourceURL != "https://example.com/a" {
		t.Fatalf("链接没有保存到草稿：%q", a.Inspector.SourceURL)
	}
	if draft, dirty := a.inspectorDraft(); !dirty || !draft.Details || draft.Source != "https://example.com/a" {
		t.Fatalf("新链接没有进入保存草稿：%+v", draft)
	}
}

// 标签选择器点一下加上、选择器不关；已加上的再点一下取消。
func TestInspectorTagPickerStaysOpenAndToggles(t *testing.T) {
	a := inspectorTestApp(model.Asset{ID: "a", TagIDs: []string{"landscape"}})
	a.Library, a.Busy = &library.Library{}, true
	a.Snapshot.Tags = []model.Tag{{ID: "landscape", Name: "Landscape"}, {ID: "portrait", Name: "Portrait"}}
	tt := ui.NewTester(a.inspector, 280, 800)
	if err := tt.Click(a.T("add_tags")); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Portrait"); err != nil {
		t.Fatal(err)
	}
	if !a.Inspector.TagsOpen || len(a.queuedRuns) != 1 {
		t.Fatalf("选标签后选择器应保持打开且排队写入：open=%v queued=%d", a.Inspector.TagsOpen, len(a.queuedRuns))
	}
	asset := model.Asset{}
	a.queuedRuns[0].patch.apply(&asset)
	if len(asset.TagIDs) != 1 || asset.TagIDs[0] != "portrait" {
		t.Fatalf("点没加的标签应加上：%v", asset.TagIDs)
	}
	done, ok := tt.AccessByRole(ui.RoleButton, "Landscape")
	if !ok {
		t.Fatalf("没找到已加上的标签按钮：%v", tt.Texts())
	}
	tt.ClickAt(done.Bounds.X+done.Bounds.W/2, done.Bounds.Y+done.Bounds.H/2)
	if len(a.queuedRuns) != 2 {
		t.Fatal("点已加上的标签没有排队取消")
	}
	asset = model.Asset{TagIDs: []string{"landscape", "x"}}
	a.queuedRuns[1].patch.apply(&asset)
	if len(asset.TagIDs) != 1 || asset.TagIDs[0] != "x" {
		t.Fatalf("点已加上的标签应取消：%v", asset.TagIDs)
	}
}

// 标签选择器回车：有完全同名的就加它，不挑部分匹配。
func TestInspectorTagPickerEnterPrefersExactName(t *testing.T) {
	a := inspectorTestApp(model.Asset{ID: "a"})
	a.Library, a.Busy = &library.Library{}, true
	a.Snapshot.Tags = []model.Tag{{ID: "t2023", Name: "Travel 2023"}, {ID: "travel", Name: "Travel"}}
	tt := ui.NewTester(a.inspector, 280, 800)
	if err := tt.Click(a.T("add_tags")); err != nil {
		t.Fatal(err)
	}
	tt.Type("travel")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if len(a.queuedRuns) != 1 || a.queuedRuns[0].label != "tags" {
		t.Fatalf("应加上同名标签：%+v", a.queuedRuns)
	}
	asset := model.Asset{}
	a.queuedRuns[0].patch.apply(&asset)
	if len(asset.TagIDs) != 1 || asset.TagIDs[0] != "travel" {
		t.Fatalf("回车加错了标签：%v", asset.TagIDs)
	}
}

// 多选改备注：Esc 保存并提示改了几张。
func TestInspectorMultiNoteNotifies(t *testing.T) {
	a := inspectorTestApp(model.Asset{ID: "a"}, model.Asset{ID: "b"})
	a.Library, a.Busy = &library.Library{}, true
	a.assetByID = map[string]model.Asset{"a": {ID: "a"}, "b": {ID: "b"}}
	tt := ui.NewTester(a.inspector, 280, 800)
	if err := tt.Click(a.T("asset_note")); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Type("hello")
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.Notice != a.T("assets_note_updated", 2) || a.Inspector.Editing != "" {
		t.Fatalf("多选改备注后提示不对：%q editing=%q", a.Notice, a.Inspector.Editing)
	}
}

// 回收站里的素材：标题、备注、标签只读，没有加号；「恢复」只在工具栏，检查器里不重复放。
func TestInspectorTrashedIsReadOnly(t *testing.T) {
	a := inspectorTestApp(model.Asset{ID: "a", Name: "Reference", Note: "Old note", Trashed: true, TagIDs: []string{"landscape"}})
	a.Snapshot.Tags = []model.Tag{{ID: "landscape", Name: "Landscape"}}
	tt := ui.NewTester(a.inspector, 280, 800)
	if _, ok := tt.AccessByRole(ui.RoleButton, a.T("restore")); ok {
		t.Fatal("检查器里不该再放一个「恢复」")
	}
	for _, label := range []string{a.T("edit_title"), a.T("asset_note"), a.T("add_tags"), a.T("add_folders"), a.T("source_url")} {
		if _, ok := tt.Access(label); ok {
			t.Errorf("回收站里的素材不该能编辑：%q", label)
		}
	}
	for _, text := range []string{"Reference", "Old note", "Landscape"} {
		if !tt.HasText(text) {
			t.Errorf("只读展示缺少 %q：%v", text, tt.Texts())
		}
	}
	if tt.HasText(a.T("add_source_link")) {
		t.Error("回收站里的素材没有链接时不该显示添加占位")
	}
}
