package ui

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"testing"
)

func TestExplicitQuitDoesNotHideAndCancelWindowClose(t *testing.T) {
	a := &App{Preferences: DefaultPreferences()}
	event := &mygo.CloseEvent{}
	a.closeMainWindow(event)
	if !event.DefaultPrevented() {
		t.Fatal("普通关窗不应退出进程")
	}
	a.Quitting = true
	canceled := false
	a.cancel = func() { canceled = true }
	event = &mygo.CloseEvent{}
	a.closeMainWindow(event)
	if event.DefaultPrevented() || !canceled {
		t.Fatal("显式退出被后台运行拦截，或未停止后台任务")
	}
}

func TestInactiveWindowDoesNotReplaceFocusedEditorMenu(t *testing.T) {
	a := &App{Preferences: DefaultPreferences()}
	var mainText, settingsText string
	main := ui.NewTester(func(c *ui.Context) { ui.TextInput(c, &mainText).Label("Main editor"); a.SyncMenuFocus(c) }, 300, 100)
	settings := ui.NewTester(func(c *ui.Context) { ui.TextInput(c, &settingsText).Label("Settings editor"); a.SyncMenuFocus(c) }, 300, 100)
	settings.SetWindowActive(true)
	if err := settings.Click("Settings editor"); err != nil {
		t.Fatal(err)
	}
	if !a.TextInputFocused {
		t.Fatal("当前文本输入框没有安装编辑菜单")
	}
	main.SetWindowActive(false)
	main.Frame()
	if !a.TextInputFocused {
		t.Fatal("后台主窗口重画抢走设置输入框的复制粘贴菜单")
	}
}
