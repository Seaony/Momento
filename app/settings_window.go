package main

import (
	"runtime"

	appui "github.com/Seaony/Momento/app/ui"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func showSettingsWindow(a *appui.App) {
	if a.SettingsWindow != nil {
		a.SettingsWindow.Restore()
		a.SettingsWindow.Focus()
		return
	}
	a.Settings.LoginEnabled = mygo.App.OpenAtLogin()
	a.SettingsWindow = mygo.NewWindow(mygo.WindowOptions{
		Title: a.T("settings"), Width: 560, Height: a.SettingsHeight(), UseContentSize: true,
		MinWidth: 560, MaxWidth: 560, DisableResize: true, DisableMaximize: true, DisableFullScreen: true,
		TitleBarStyle: mygo.TitleBarHiddenInset, TitleBarHeight: 36,
		TrafficLightPosition: &mygo.Point{X: 14, Y: 11},
		BackgroundColor:      "light-dark(#f4f4f5, #27272a)", Content: ui.View(a.SettingsView),
	})
	a.SettingsWindow.OnClosed(func() {
		a.SettingsWindow = nil
		a.SettingsOpen = false
	})
	if runtime.GOOS != "darwin" {
		installMenu(a)
	}
}
