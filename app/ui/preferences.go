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

func NormalizePreferences(p Preferences) Preferences {
	if p.Language == "" {
		p.Language = "system"
	}
	switch p.Appearance {
	case "system", "light", "dark":
	default:
		p.Appearance = "system"
	}
	if p.ViewMode < 0 || p.ViewMode > 2 {
		p.ViewMode = 0
	}
	switch p.Sort {
	case "added", "name", "size":
	default:
		p.Sort, p.SortDescending = "added", true
	}
	p.SidebarWidth = max(196, min(340, p.SidebarWidth))
	return p
}

func (a *App) RestoreUIState() {
	a.Preferences = NormalizePreferences(a.Preferences)
	p := a.Preferences
	a.ViewMode, a.Query.Sort, a.Query.Descending = p.ViewMode, p.Sort, p.SortDescending
	a.SidebarVisible, a.InspectorVisible, a.SidebarWidth = p.SidebarVisible, p.InspectorVisible, p.SidebarWidth
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

func (a *App) RememberUIState() {
	if a.captureUIState() {
		a.queuePreferences()
	}
}

func (a *App) captureUIState() bool {
	p := a.Preferences
	p.ViewMode, p.Sort, p.SortDescending = a.ViewMode, a.Query.Sort, a.Query.Descending
	if p.Sort == "" {
		p.Sort, p.SortDescending = "added", true
	}
	p.SidebarVisible, p.InspectorVisible, p.SidebarWidth = a.SidebarVisible, a.InspectorVisible, a.SidebarWidth
	p = NormalizePreferences(p)
	if p == a.Preferences {
		return false
	}
	a.Preferences = p
	return true
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
func (a *App) FlushPreferences() error {
	a.captureUIState()
	if a.configDir == "" {
		return nil
	}
	err := a.writePreferences(a.Preferences, a.prefVersion.Add(1))
	if err != nil {
		a.preferenceError(err)
	}
	return err
}

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
