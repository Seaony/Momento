package main

import (
	"runtime"

	appui "github.com/Seaony/Momento/app/ui"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func showSettingsWindow(a *appui.App) {
	// 已开着的窗口再次打开也重新读「登录时启动」，系统设置里可能改过。
	a.RefreshLaunchAtLogin()
	if a.SettingsWindow != nil {
		a.SettingsWindow.Show()
		a.SettingsWindow.Restore()
		a.SettingsWindow.Focus()
		return
	}
	a.SettingsWindow = mygo.NewWindow(mygo.WindowOptions{
		Title: a.T("settings"), Width: 760, Height: 560, UseContentSize: true,
		MinWidth: 760, MinHeight: 560, DisableResize: true, DisableMaximize: true, DisableFullScreen: true,
		TitleBarStyle: mygo.TitleBarHiddenInset, TitleBarHeight: 44,
		TrafficLightPosition: appui.TrafficLights(),
		BackgroundColor:      appui.WindowBackgroundColor(), Content: ui.View(a.SettingsView),
	})
	a.ApplyWindowZoom()
	a.SettingsWindow.OnClosed(func() {
		a.SettingsWindow = nil
		a.SettingsOpen = false
		a.ResetSettingsSession()
	})
	if runtime.GOOS != "darwin" {
		installMenu(a)
	}
}
