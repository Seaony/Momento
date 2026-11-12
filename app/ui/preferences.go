package ui

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
)

func DefaultPreferences() Preferences {
	return Preferences{
		Language: "system", Appearance: "system", AutoCheckUpdates: true, BrowserExtensionEnabled: true,
		Sort: "added", SortDescending: true, SidebarVisible: true, InspectorVisible: true, SidebarWidth: 236,
	}
}

func (a *App) SavePreferences() {
	a.Texts.Language = a.Preferences.Language
	if a.Window != nil || a.SettingsWindow != nil {
		a.applyLocale()
		a.applyAppearance()
	}
	if a.MenuChanged != nil {
		a.MenuChanged()
	}
	a.InvalidateWindows()
	a.queuePreferences()
}

func (a *App) queuePreferences() {
	if a.configDir == "" {
		return
	}
	p, version := a.Preferences, a.prefVersion.Add(1)
	go func() {
		if err := a.writePreferences(p, version); err != nil {
			a.preferenceError(err)
		}
	}()
}

// 退出前同步写入最新状态，旧的后台写入不能在它之后覆盖文件。
func (a *App) writePreferences(p Preferences, version uint64) error {
	a.prefMu.Lock()
	defer a.prefMu.Unlock()
	if version != a.prefVersion.Load() {
		return nil
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(a.configDir, 0700); err != nil {
		return err
	}
	path := filepath.Join(a.configDir, "preferences.json")
	if err = os.WriteFile(path+".tmp", data, 0600); err != nil {
		return err
	}
	return os.Rename(path+".tmp", path)
}

func (a *App) preferenceError(err error) {
	slog.Error("保存偏好失败", "error", err)
	if a.Window != nil {
		a.Window.Update(func() { a.Error = a.Texts.Error(err); a.InvalidateWindows() })
	} else {
		a.Error = a.Texts.Error(err)
	}
}
