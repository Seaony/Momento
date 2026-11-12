package ui

import (
	"log/slog"

	"github.com/Seaony/Momento/app/internal/logging"
	"github.com/egoist/mygo"
)

var setLaunchAtLogin = mygo.App.SetOpenAtLogin
var launchAtLogin = mygo.App.OpenAtLogin

func (a *App) SetLaunchAtLogin(enabled bool) {
	a.Settings.LoginError = ""
	err := setLaunchAtLogin(enabled)
	a.Settings.LoginEnabled = launchAtLogin()
	a.Preferences.LaunchAtLogin = a.Settings.LoginEnabled
	if err != nil {
		slog.Error("设置开机启动失败", "error", err)
		a.Settings.LoginError = err.Error()
	}
	a.SavePreferences()
}

func (a *App) SetExtensionEnabled(enabled bool) {
	if a.Settings.ExtensionBusy {
		return
	}
	a.Preferences.BrowserExtensionEnabled = enabled
	a.Settings.ExtensionBusy = true
	a.SavePreferences()
	go func() {
		if err := a.Extension.SetEnabled(enabled); err != nil {
			slog.Error("切换浏览器扩展服务失败", "enabled", enabled, "error", err)
		}
		a.Window.Update(func() {
			a.ExtensionStatus = a.Extension.Status()
			a.Settings.ExtensionBusy = false
			a.InvalidateWindows()
		})
	}()
}

func (a *App) OpenLogDirectory() {
	if err := mygo.Shell.OpenPath(logging.Directory()); err != nil {
		a.ShowError(err)
		a.Window.Focus()
		a.Window.Invalidate()
	}
}
