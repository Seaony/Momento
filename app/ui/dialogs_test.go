package ui

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/logging"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func TestDialogsPromptRequiresNameAndSubmitsOnce(t *testing.T) {
	a := &App{}
	calls := 0
	name := ""
	a.Ask("pencil-simple", "Rename", "", func(value string) { calls++; name = value })
	tt := ui.NewTester(a.dialogs, 1100, 640)
	if button, ok := tt.AccessByRole(ui.RoleButton, a.T("save")); !ok || !button.Disabled {
		t.Fatal("空名称时保存按钮应禁用")
	}
	tt.Type("  Sunset  ")
	tt.Key(0, ui.KeyEnter)
	if calls != 1 || name != "Sunset" || a.PromptOpen {
		t.Fatalf("提交结果：calls=%d name=%q open=%v", calls, name, a.PromptOpen)
	}
	tt.Frame()
	if calls != 1 {
		t.Fatal("下一帧重复执行了提交")
	}
}

func TestDialogsConfirmationCancelDoesNotRunAction(t *testing.T) {
	a := &App{}
	calls := 0
	a.Confirm("Delete", "Delete these images?", a.T("delete"), func() { calls++ })
	tt := ui.NewTester(a.dialogs, 1100, 640)
	tt.Key(0, ui.KeyEscape)
	if calls != 0 || a.ConfirmOpen {
		t.Fatal("Escape 必须取消")
	}
	a.Confirm("Delete", "Delete these images?", a.T("delete"), func() { calls++ })
	tt.Frame()
	button, ok := tt.AccessByRole(ui.RoleButton, a.T("delete"))
	if !ok {
		t.Fatal("确认框没有显示删除按钮")
	}
	tt.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
	if calls != 1 || a.ConfirmOpen {
		t.Fatal("确认应执行一次并关闭")
	}
}

func TestDialogsExportDefaultsAndErrorCopy(t *testing.T) {
	a := &App{ExportOpen: true}
	a.Selected.Add("selected")
	tt := ui.NewTester(a.dialogs, 1100, 640)
	if a.Dialog.Format != model.ExportOriginal || a.Dialog.Quality != 0.9 {
		t.Fatal("导出应默认原文件和 0.9 JPEG 质量")
	}
	tt.Key(0, ui.KeyEscape)
	a.Error = "disk full"
	tt.Frame()
	if err := tt.Click(a.T("copy_error")); err != nil {
		t.Fatal(err)
	}
	if clipboard := tt.Clipboard(); !strings.Contains(clipboard, "disk full") || !strings.Contains(clipboard, logging.Directory()) {
		t.Fatalf("复制内容缺少错误或日志位置：%q", clipboard)
	}
	if err := tt.Click(a.T("close")); err != nil {
		t.Fatal(err)
	}
	if a.Error != "" {
		t.Fatal("关闭后没有清掉错误")
	}
}

func TestConfirmationKeepsDefaultActionAndIgnoresOutsideClick(t *testing.T) {
	a := &App{}
	calls := 0
	a.Confirm("Delete", "Delete these images?", a.T("delete"), func() { calls++ })
	tt := ui.NewTester(a.dialogs, 1100, 640)
	if alert, ok := tt.AccessByRole(ui.RoleAlertDialog, "Delete"); !ok || alert.Description != "Delete these images?" {
		t.Fatal("确认框没有保留读屏标题和说明")
	}
	if !tt.Focused(a.T("delete")) {
		t.Fatal("确认操作没有默认键盘焦点")
	}
	tt.Key(0, ui.KeyEnter)
	if a.ConfirmOpen || calls != 1 {
		t.Fatal("默认确认操作应只执行一次")
	}
	tt.Frame()
	if calls != 1 {
		t.Fatal("下一帧重复执行确认")
	}
	a.Confirm("Delete", "Delete these images?", a.T("delete"), func() { calls++ })
	tt.Frame()
	tt.ClickAt(12, 12)
	if !a.ConfirmOpen || calls != 1 {
		t.Fatal("点击遮罩不应取消或执行确认")
	}
	tt.Key(0, ui.KeyEscape)
	if a.ConfirmOpen || calls != 1 {
		t.Fatal("Escape 应取消确认")
	}
}

// 转格式时可以限制最长边：输错了不能导出也不能存预设；存成预设后重新打开对话框，点预设就恢复同样的设置；同名预设覆盖。
func TestExportMaxEdgeAndPresets(t *testing.T) {
	a := &App{ExportOpen: true, Texts: i18n.Catalog{Language: "zh-CN"}}
	a.Selected.Add("selected")
	tt := ui.NewTester(a.dialogs, 1100, 760)
	if tt.HasText(a.T("export_max_edge")) {
		t.Fatal("原格式导出不该出现限制最长边")
	}
	a.Dialog.Format = model.ExportJPEG
	tt.Frame()
	if err := tt.Click(a.T("export_max_edge")); err != nil {
		t.Fatal(err)
	}
	if !a.Dialog.MaxEdgeOn {
		t.Fatal("勾选限制最长边没生效")
	}
	a.Dialog.MaxEdge = "abc"
	tt.Frame()
	if _, problem := a.exportOptions(); problem == "" || !tt.HasText(problem) {
		t.Fatal("最长边输错时应提示")
	}
	if button, ok := tt.AccessByRole(ui.RoleButton, a.T("export")); !ok || !button.Disabled {
		t.Fatal("最长边输错时导出按钮应禁用")
	}
	a.Dialog.MaxEdge, a.Dialog.Quality = " 1600 ", .8
	options, problem := a.exportOptions()
	if problem != "" || options != (model.ExportOptions{Format: model.ExportJPEG, Quality: .8, MaxEdge: 1600}) {
		t.Fatalf("导出设置不对：%+v %q", options, problem)
	}
	a.saveExportPreset("网页")
	a.Dialog.MaxEdge = "800"
	a.saveExportPreset("网页")
	if len(a.Preferences.ExportPresets) != 1 || a.Preferences.ExportPresets[0].MaxEdge != 800 {
		t.Fatalf("同名预设应覆盖：%+v", a.Preferences.ExportPresets)
	}
	// 关掉再打开，设置回到默认；点预设恢复。
	a.ExportOpen = false
	tt.Frame()
	a.ExportOpen = true
	tt.Frame()
	if a.Dialog.Format != model.ExportOriginal || a.Dialog.MaxEdgeOn {
		t.Fatal("重新打开应回到默认设置")
	}
	if err := tt.Click("网页"); err != nil {
		t.Fatal(err)
	}
	if options, _ := a.exportOptions(); options != (model.ExportOptions{Format: model.ExportJPEG, Quality: .8, MaxEdge: 800}) {
		t.Fatalf("点预设没有恢复设置：%+v", options)
	}
	// 另存一个不限尺寸的预设；最长边输错时它不能被当成「当前设置」高亮。
	a.Dialog.MaxEdgeOn = false
	a.saveExportPreset("原尺寸")
	a.Dialog.MaxEdgeOn, a.Dialog.MaxEdge = true, "0"
	tt.Frame()
	if chip, ok := tt.Access("原尺寸"); !ok || chip.Checked {
		t.Fatal("最长边输错时不该高亮不限尺寸的预设")
	}
	a.Dialog.MaxEdgeOn = false
	tt.Frame()
	if chip, ok := tt.Access("原尺寸"); !ok || !chip.Checked {
		t.Fatal("设置和预设一样时应高亮")
	}
}

// 预设存在 preferences.json 里，嵌进去的导出设置要原样读回来。
func TestExportPresetsPersist(t *testing.T) {
	in := Preferences{ExportPresets: []ExportPreset{{Name: "网页", ExportOptions: model.ExportOptions{Format: model.ExportJPEG, Quality: .8, MaxEdge: 2048, Motion: true}}}}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"maxEdge":2048`) {
		t.Fatalf("预设字段没有展开存：%s", data)
	}
	var out Preferences
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(out.ExportPresets, in.ExportPresets) {
		t.Fatalf("读回的预设不同：%+v", out.ExportPresets)
	}
}
