package main

import (
	"runtime"

	appui "github.com/Seaony/Momento/app/ui"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func showSettingsWindow(a *appui.App) {
	if a.SettingsWindow != nil {
		a.SettingsWindow.Show()
		a.SettingsWindow.Restore()
		a.SettingsWindow.Focus()
		return
	}
	a.Settings.LoginEnabled = mygo.App.OpenAtLogin()
	a.SettingsWindow = mygo.NewWindow(mygo.WindowOptions{
		Title: a.T("settings"), Width: 760, Height: 560, UseContentSize: true,
		MinWidth: 760, MinHeight: 560, DisableResize: true, DisableMaximize: true, DisableFullScreen: true,
		TitleBarStyle: mygo.TitleBarHiddenInset, TitleBarHeight: 44,
		TrafficLightPosition: &mygo.Point{X: 14, Y: 15},
		BackgroundColor:      "light-dark(#f4f4f5, #27272a)", Content: ui.View(a.SettingsView),
	})
	a.ApplyWindowZoom()
	a.SettingsWindow.OnClosed(func() {
		a.SettingsWindow = nil
		a.SettingsOpen = false
		a.CancelShortcutRecording()
	})
	if runtime.GOOS != "darwin" {
		installMenu(a)
	}
}
