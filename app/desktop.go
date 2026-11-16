package main

import (
	"github.com/Seaony/Momento/app/resources"
	appui "github.com/Seaony/Momento/app/ui"
	"github.com/egoist/mygo"
	"log/slog"
	"runtime"
)

var registerGlobalShortcut = mygo.GlobalShortcut.Register
var unregisterGlobalShortcut = mygo.GlobalShortcut.Unregister

var newTray = mygo.NewTray

type desktopIntegration struct {
	app        *appui.App
	tray       *mygo.Tray
	trayFailed bool // 托盘建不出来（如 GNOME 没有托盘）后不再自动重试，免得每次切焦点、改设置都弹同一个错误
	shortcuts  map[string]string
}

func installDesktop(a *appui.App) *desktopIntegration {
	d := &desktopIntegration{app: a}
	a.WindowShown = func() { d.refreshDock() }
	a.PreferencesChanged = func() { d.refresh() }
	mygo.App.OnActivate(func(visible bool) {
		if !visible {
			a.ShowMainWindow()
		}
	})
	mygo.App.OnDidBecomeActive(func() { mygo.ClearNotifications() })
	d.refresh()
	return d
}

func (d *desktopIntegration) refreshDock() {
	a := d.app
	if runtime.GOOS != "darwin" {
		return
	}
	policy := mygo.ActivationPolicyRegular
	if a.Window != nil && !a.Window.IsVisible() && a.Preferences.TrayIcon && !a.Preferences.DockWhenHidden {
		policy = mygo.ActivationPolicyAccessory
	}
	mygo.App.SetActivationPolicy(policy)
}

func (d *desktopIntegration) refresh() {
	a := d.app
	if !a.Preferences.TrayIcon {
		if d.tray != nil {
			d.tray.Destroy()
			d.tray = nil
		}
		// 用户关掉托盘设置后再打开，会重新试一次。
		d.trayFailed = false
	}
	if a.Preferences.TrayIcon && !d.trayFailed {
		menu := mygo.NewMenu(trayItems(a))
		if d.tray == nil {
			tray, err := newTray(mygo.TrayOptions{ID: "momento", Icon: resources.TrayIcon, IconIsTemplate: true, ToolTip: "Momento", Menu: menu})
			if err != nil {
				slog.Error("创建托盘失败", "error", err)
				a.ShowError(err)
				a.Preferences.DockWhenHidden = true
				d.trayFailed = true
			} else {
				d.tray = tray
			}
		} else {
			d.tray.SetMenu(menu)
		}
	}
	d.refreshDock()
	d.refreshShortcuts()
}

func trayItems(a *appui.App) []*mygo.MenuItem {
	item := func(key string, fn func()) *mygo.MenuItem {
		return &mygo.MenuItem{Label: a.T(key), Click: func(*mygo.MenuItem, *mygo.Window) { fn() }}
	}
	items := []*mygo.MenuItem{item("show_momento", a.ShowMainWindow)}
	if !a.IsLocked() {
		items = append(items, item("paste_import", a.PasteImport), item("settings", a.ShowSettings))
	}
	return append(items, mygo.Separator(), item("quit_momento", mygo.App.Quit))
}

func (d *desktopIntegration) refreshShortcuts() {
	a := d.app
	next := map[string]string{}
	for _, definition := range appui.ShortcutDefinitions {
		if definition.Global {
			next[definition.Action] = a.Shortcut(definition.Action)
		}
	}
	for action, old := range d.shortcuts {
		if next[action] != old && old != "" {
			unregisterGlobalShortcut(old)
		}
	}
	if a.Settings.ShortcutErrors == nil {
		a.Settings.ShortcutErrors = map[string]string{}
	}
	for action, accelerator := range next {
		if d.shortcuts[action] == accelerator {
			continue
		}
		delete(a.Settings.ShortcutErrors, action)
		if accelerator == "" {
			continue
		}
		err := registerGlobalShortcut(accelerator, func() {
			a.ShowMainWindow()
			if action == "show_search" && !a.IsLocked() {
				a.FocusSearch()
			}
		})
		if err != nil {
			// 记日志并显示本地化的「注册失败：原因」。
			a.SetShortcutRegistrationError(action, err)
			next[action] = ""
		}
	}
	d.shortcuts = next
}
