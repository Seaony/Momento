package ui

import (
	"errors"
	"fmt"

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

// 录快捷键或输密码时主窗口藏到后台，回来后快捷键照常，输了一半的密码已清掉；密码正在保存时不动。
func TestHidingMainWindowEndsSettingsSession(t *testing.T) {
	a := &App{Preferences: DefaultPreferences()}
	a.Settings.RecordingShortcut = "search"
	a.Lock.DialogOpen, a.Lock.Mode, a.Lock.NewPassword = true, "enable", "half"
	a.HideMainWindow()
	if a.Settings.RecordingShortcut != "" {
		t.Fatal("藏起主窗口后还在录快捷键，所有快捷键都失效")
	}
	if a.Lock.DialogOpen || a.Lock.NewPassword != "" {
		t.Fatal("输了一半的密码还留着")
	}
	a.Lock.DialogOpen, a.Lock.Saving = true, true
	a.ResetSettingsSession()
	if !a.Lock.DialogOpen {
		t.Fatal("密码正在保存时把对话框关了，结果没处显示")
	}
}

func TestSettingsReopenReadsLaunchAtLogin(t *testing.T) {
	previous := launchAtLogin
	t.Cleanup(func() { launchAtLogin = previous })
	enabled := true
	launchAtLogin = func() bool { return enabled }
	a := &App{Preferences: DefaultPreferences()}
	a.RefreshLaunchAtLogin()
	enabled = false
	a.RefreshLaunchAtLogin()
	if a.Settings.LoginEnabled {
		t.Fatal("再次打开设置时没读回系统的登录启动状态")
	}
}

func TestNotificationDeniedHintClearsAfterSuccess(t *testing.T) {
	a := &App{Preferences: DefaultPreferences()}
	a.notificationShown(fmt.Errorf("show: %w", mygo.ErrNotificationsDenied))
	if !a.Settings.NotificationDenied {
		t.Fatal("通知被拒绝没有提示")
	}
	a.notificationShown(errors.New("timeout"))
	if !a.Settings.NotificationDenied {
		t.Fatal("别的错误不该改变拒绝提示")
	}
	a.notificationShown(nil)
	if a.Settings.NotificationDenied {
		t.Fatal("重新允许通知后提示没有消失")
	}
}
