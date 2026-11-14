package main

import (
	"errors"
	"testing"

	"github.com/Seaony/Momento/app/internal/i18n"
	appui "github.com/Seaony/Momento/app/ui"
	"github.com/egoist/mygo"
)

func desktopTestApp() *appui.App {
	return &appui.App{Preferences: appui.DefaultPreferences(), Texts: i18n.Catalog{Language: "en"}}
}
func menuAction(items []*mygo.MenuItem, id string) *mygo.MenuItem {
	for _, item := range items {
		if item.ID == id {
			return item
		}
		if child := menuAction(item.Submenu, id); child != nil {
			return child
		}
	}
	return nil
}
func TestMenusUseRecordedShortcutsAndPreserveTextPaste(t *testing.T) {
	a := desktopTestApp()
	a.Preferences.Shortcuts["settings"] = "Ctrl+Alt+S"
	for _, platform := range []string{"darwin", "windows", "linux"} {
		items := menuItems(a, platform)
		if got := menuAction(items, "settings"); got == nil || got.Accelerator != "Ctrl+Alt+S" {
			t.Fatalf("%s自定义快捷键未进菜单: %+v", platform, got)
		}
		if got := menuAction(items, "paste"); got == nil || got.Click == nil {
			t.Fatal("无文本焦点时没有图片粘贴入口")
		}
		a.TextInputFocused = true
		items = menuItems(a, platform)
		if got := menuAction(items, "paste"); got == nil || got.Role != mygo.RolePaste || got.Click != nil {
			t.Fatal("文本粘贴被图片导入抢走")
		}
		a.TextInputFocused = false
	}
	a.Settings.RecordingShortcut = "settings"
	var check func([]*mygo.MenuItem)
	check = func(items []*mygo.MenuItem) {
		for _, item := range items {
			if item.Accelerator != "" {
				t.Fatal("录制时仍有菜单快捷键")
			}
			check(item.Submenu)
		}
	}
	check(menuItems(a, "darwin"))
}
func TestLockedMenusDoNotExposeLibraryActions(t *testing.T) {
	a := desktopTestApp()
	a.Preferences.PasswordHash = "configured"
	a.InitPasswordLock()
	for _, platform := range []string{"darwin", "windows", "linux"} {
		items := menuItems(a, platform)
		for _, id := range []string{"settings", "paste", "import", "open_library", "export"} {
			if menuAction(items, id) != nil {
				t.Fatalf("锁定时仍能通过菜单执行%s", id)
			}
		}
	}
	if len(trayItems(a)) != 3 {
		t.Fatal("锁定托盘没有限制为显示和退出")
	}
}
func TestGlobalShortcutFailureRemainsVisibleAndCanRetry(t *testing.T) {
	previousRegister, previousUnregister := registerGlobalShortcut, unregisterGlobalShortcut
	t.Cleanup(func() { registerGlobalShortcut = previousRegister; unregisterGlobalShortcut = previousUnregister })
	a := desktopTestApp()
	a.Preferences.Shortcuts["show_app"] = "Ctrl+Alt+M"
	attempts := 0
	var removed []string
	registerGlobalShortcut = func(acc string, fn func()) error {
		attempts++
		if attempts == 1 {
			return errors.New("occupied")
		}
		return nil
	}
	unregisterGlobalShortcut = func(acc string) { removed = append(removed, acc) }
	d := &desktopIntegration{app: a}
	d.refreshShortcuts()
	if a.Settings.ShortcutErrors["show_app"] != "occupied" {
		t.Fatal("注册失败未显示")
	}
	d.refreshShortcuts()
	if attempts != 2 || a.Settings.ShortcutErrors["show_app"] != "" {
		t.Fatal("重试没有恢复注册状态")
	}
	a.Preferences.Shortcuts["show_app"] = ""
	d.refreshShortcuts()
	if len(removed) != 1 || removed[0] != "Ctrl+Alt+M" {
		t.Fatal("清除按键未注销原注册")
	}
}

func TestInspectorMenuFollowsSelectionAndHelpIsAvailable(t *testing.T) {
	a := desktopTestApp()
	for _, platform := range []string{"darwin", "windows", "linux"} {
		items := menuItems(a, platform)
		if got := menuAction(items, "inspector"); got == nil || !got.Disabled {
			t.Fatal("无选择时检查器菜单没有置灰")
		}
		if menuAction(items, "back") != nil || menuAction(items, "forward") != nil {
			t.Fatal("菜单保留了历史导航")
		}
		if menuAction(items, "user_guide") == nil {
			t.Fatal("缺少使用说明入口")
		}
		a.Selected.Add("selected")
		if got := menuAction(menuItems(a, platform), "inspector"); got == nil || got.Disabled {
			t.Fatal("有选择时检查器菜单仍禁用")
		}
		a.Selected.Clear()
	}
	for _, tc := range []struct{ language, url string }{
		{"zh-CN", "https://github.com/Seaony/Momento/blob/master/README.zh-CN.md"},
		{"en", "https://github.com/Seaony/Momento#readme"},
		{"ja", "https://github.com/Seaony/Momento#readme"},
	} {
		a.Texts.Language = tc.language
		if a.HelpURL() != tc.url {
			t.Fatalf("帮助链接错误：%s", a.HelpURL())
		}
	}
}
