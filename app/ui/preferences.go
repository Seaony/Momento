package ui

import (
	"encoding/json"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
)

func DefaultPreferences() Preferences {
	return Preferences{
		Language: "system", Theme: "system", UIZoom: 1, AutoCheckUpdates: true, BrowserExtensionEnabled: true,
		CloseAction: "background", TrayIcon: true, DockWhenHidden: true, SidebarCounts: true,
		FolderDoubleClick: "expand", DoubleClick: "preview", SpaceKey: "preview", PreviewWheel: "navigate",
		PreviewFit: "smart", PreviewChecker: true, SelectImported: true,
		AutoImport: map[string][]AutoImportRule{}, Shortcuts: map[string]string{},
		Notify: true, NotifyEvents: []string{"extension", "auto-import", "duplicate"}, NotifySound: true, ThumbnailSize: 1,
		Sort: "added", SortDescending: true, SidebarVisible: true, SidebarWidth: 236,
		// 智能文件夹默认关闭，用户在设置 → 侧栏里手动打开。
		SmartFolders: false,
	}
}

// 缺字段沿用默认值，旧主题只在读取时迁移。
func (p *Preferences) UnmarshalJSON(data []byte) error {
	type plain Preferences
	value := plain(DefaultPreferences())
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	var legacy struct {
		Appearance string  `json:"appearance"`
		Theme      *string `json:"theme"`
	}
	if err := json.Unmarshal(data, &legacy); err != nil {
		return err
	}
	if legacy.Theme == nil && slices.Contains([]string{"system", "light", "dark"}, legacy.Appearance) {
		value.Theme = legacy.Appearance
	}
	*p = NormalizePreferences(Preferences(value))
	return nil
}

func NormalizePreferences(p Preferences) Preferences {
	if !slices.Contains([]string{"system", "zh-CN", "zh-Hans", "zh", "en", "ja"}, p.Language) {
		p.Language = "system"
	}
	switch p.Theme {
	case "light-gray":
		p.Theme = "light"
	case "dark-gray", "navy", "purple":
		p.Theme = "dark"
	}
	if !slices.Contains(themeKeys, p.Theme) {
		p.Theme = "system"
	}
	if !slices.Contains([]float32{.8, .9, 1, 1.1, 1.25, 1.5}, p.UIZoom) {
		p.UIZoom = 1
	}
	if p.CloseAction != "quit" {
		p.CloseAction = "background"
	}
	if !p.TrayIcon {
		p.DockWhenHidden = true
	}
	if p.FolderDoubleClick != "rename" {
		p.FolderDoubleClick = "expand"
	}
	if p.DoubleClick != "external" {
		p.DoubleClick = "preview"
	}
	if p.SpaceKey != "system" {
		p.SpaceKey = "preview"
	}
	if p.PreviewWheel != "zoom" {
		p.PreviewWheel = "navigate"
	}
	if p.PreviewFit != "fit" {
		p.PreviewFit = "smart"
	}
	// "smart" 只为兼容旧数据保留；智能文件夹是否显示只看 SmartFolders。
	hidden := make([]string, 0, len(p.SidebarHidden))
	for _, key := range p.SidebarHidden {
		if slices.Contains([]string{"favorites", "unfiled", "untagged", "random", "folders", "recent", "quick", "smart"}, key) && !slices.Contains(hidden, key) {
			hidden = append(hidden, key)
		}
	}
	p.SidebarHidden = hidden
	events := make([]string, 0, len(p.NotifyEvents))
	for _, key := range p.NotifyEvents {
		if slices.Contains([]string{"extension", "auto-import", "duplicate"}, key) && !slices.Contains(events, key) {
			events = append(events, key)
		}
	}
	p.NotifyEvents = events
	if p.AutoImport == nil {
		p.AutoImport = map[string][]AutoImportRule{}
	}
	if p.Shortcuts == nil {
		p.Shortcuts = map[string]string{}
	}
	if math.IsNaN(float64(p.ThumbnailSize)) || math.IsInf(float64(p.ThumbnailSize), 0) || p.ThumbnailSize == 0 {
		p.ThumbnailSize = 1
	}
	p.ThumbnailSize = max(.5, min(2, p.ThumbnailSize))
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

func clonePreferences(p Preferences) Preferences {
	p.SidebarHidden = slices.Clone(p.SidebarHidden)
	p.NotifyEvents = slices.Clone(p.NotifyEvents)
	shortcuts := make(map[string]string, len(p.Shortcuts))
	for key, value := range p.Shortcuts {
		shortcuts[key] = value
	}
	p.Shortcuts = shortcuts
	imports := make(map[string][]AutoImportRule, len(p.AutoImport))
	for key, value := range p.AutoImport {
		imports[key] = slices.Clone(value)
	}
	p.AutoImport = imports
	return p
}

func (a *App) RestoreUIState() {
	a.Preferences = NormalizePreferences(a.Preferences)
	p := a.Preferences
	a.ViewMode, a.Query.Sort, a.Query.Descending = p.ViewMode, p.Sort, p.SortDescending
	a.SidebarVisible, a.SidebarWidth = p.SidebarVisible, p.SidebarWidth
	a.InspectorVisible = false
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
	a.ApplyWindowZoom()
	a.ConfigureAutoImport()
	if a.PreferencesChanged != nil {
		a.PreferencesChanged()
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
	p.SidebarVisible, p.SidebarWidth = a.SidebarVisible, a.SidebarWidth
	p = NormalizePreferences(p)
	if reflect.DeepEqual(p, a.Preferences) {
		return false
	}
	a.Preferences = p
	return true
}

func (a *App) queuePreferences() {
	if a.PasswordSavePending() {
		return
	}
	if a.configDir == "" {
		return
	}
	p, version := clonePreferences(a.Preferences), a.prefVersion.Add(1)
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
