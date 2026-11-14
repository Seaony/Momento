package ui

import (
	"strings"
	"testing"

	"github.com/Seaony/Momento/app/internal/logging"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func TestDialogsPromptRequiresNameAndSubmitsOnce(t *testing.T) {
	a := &App{}
	calls := 0
	name := ""
	a.Ask("Rename", "", func(value string) { calls++; name = value })
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
	a.Confirm("Delete", "Delete these images?", func() { calls++ })
	tt := ui.NewTester(a.dialogs, 1100, 640)
	tt.Key(0, ui.KeyEscape)
	if calls != 0 || a.ConfirmOpen {
		t.Fatal("Escape 必须取消")
	}
	a.Confirm("Delete", "Delete these images?", func() { calls++ })
	tt.Frame()
	if err := tt.Click(a.T("confirm")); err != nil {
		t.Fatal(err)
	}
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
	a.Confirm("Delete", "Delete these images?", func() { calls++ })
	tt := ui.NewTester(a.dialogs, 1100, 640)
	if alert, ok := tt.AccessByRole(ui.RoleAlertDialog, "Delete"); !ok || alert.Description != "Delete these images?" {
		t.Fatal("确认框没有保留读屏标题和说明")
	}
	if !tt.Focused(a.T("confirm")) {
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
	a.Confirm("Delete", "Delete these images?", func() { calls++ })
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
