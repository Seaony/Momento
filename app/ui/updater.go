package ui

import (
	"log/slog"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/updater"
	"github.com/egoist/mygo/plugins/updater/native"
)

var availableUpdateVersion string

// 在 App.Run 前注册，更新窗口也使用原生界面。
func InitUpdater() {
	mygo.Use(native.Plugin)
	updater.OnChange(func() {
		availableUpdateVersion = updater.AvailableVersion()
		slog.Info("更新状态已变化", "availableVersion", availableUpdateVersion, "lastCheck", updater.LastCheck())
	})
	updater.OnInstalled(func() { slog.Info("更新已安装，等待重新启动") })
}

func CheckForUpdates() {
	slog.Info("手动检查更新")
	updater.CheckForUpdates()
}

func UpdateMenuItem() *mygo.MenuItem {
	item := updater.MenuItem()
	item.Click = func(*mygo.MenuItem, *mygo.Window) { CheckForUpdates() }
	return item
}

func AvailableUpdateVersion() string { return availableUpdateVersion }

func OnUpdateChange(changed func()) func() { return updater.OnChange(changed) }
