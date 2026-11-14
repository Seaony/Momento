package ui

import (
	"fmt"
	"log/slog"
	"maps"
	"runtime"
	"strings"

	"github.com/egoist/mygo/ui"
)

type ShortcutDefinition struct {
	Action, Key, Group, Default string
	Global                      bool
}

var ShortcutDefinitions = []ShortcutDefinition{
	{"show_app", "shortcut_show_app", "global", "", true},
	{"show_search", "shortcut_show_search", "global", "", true},
	{"new_library", "new_library", "file", "CmdOrCtrl+N", false},
	{"open_library", "open_library", "file", "CmdOrCtrl+O", false},
	{"import", "import", "file", "CmdOrCtrl+I", false},
	{"import_folder", "import_folder", "file", "CmdOrCtrl+Shift+I", false},
	{"export", "export", "file", "CmdOrCtrl+E", false},
	{"settings", "settings", "file", "CmdOrCtrl+,", false},
	{"lock", "lock_now", "file", "CmdOrCtrl+L", false},
	{"close_window", "shortcut_close_window", "file", "CmdOrCtrl+W", false},
	{"quit", "shortcut_quit", "file", "CmdOrCtrl+Q", false},
	{"undo", "shortcut_undo", "edit", "CmdOrCtrl+Z", false},
	{"redo", "shortcut_redo", "edit", "CmdOrCtrl+Shift+Z", false},
	{"cut", "shortcut_cut", "edit", "CmdOrCtrl+X", false},
	{"copy", "shortcut_copy", "edit", "CmdOrCtrl+C", false},
	{"paste", "paste_import", "edit", "CmdOrCtrl+V", false},
	{"select_all", "select_all", "edit", "CmdOrCtrl+A", false},
	{"delete", "delete", "edit", "Backspace", false},
	{"view_masonry", "view_masonry", "view", "CmdOrCtrl+1", false},
	{"view_grid", "view_grid", "view", "CmdOrCtrl+2", false},
	{"view_list", "view_list", "view", "CmdOrCtrl+3", false},
	{"search", "search", "view", "CmdOrCtrl+K", false},
	{"filter", "filter", "view", "CmdOrCtrl+Alt+F", false},
	{"sort", "sort", "view", "CmdOrCtrl+Alt+S", false},
	{"inspector", "inspector", "view", "CmdOrCtrl+Alt+I", false},
	{"sidebar", "sidebar", "view", "", false},
	{"thumbnail_larger", "thumbnail_larger", "view", "CmdOrCtrl+=", false},
	{"thumbnail_smaller", "thumbnail_smaller", "view", "CmdOrCtrl+-", false},
	{"fullscreen", "shortcut_fullscreen", "view", "Ctrl+Super+F", false},
	{"preview", "momento_preview", "preview", "Space", false},
	{"preview_zoom_in", "preview_zoom_in", "preview", "CmdOrCtrl+=", false},
	{"preview_zoom_out", "preview_zoom_out", "preview", "CmdOrCtrl+-", false},
	{"preview_fit", "preview_fit_window", "preview", "CmdOrCtrl+0", false},
	{"preview_actual", "preview_actual_size", "preview", "CmdOrCtrl+1", false},
	{"preview_previous", "previous_asset", "preview", "Left", false},
	{"preview_next", "next_asset", "preview", "Right", false},
}

func shortcutDefinition(action string) (ShortcutDefinition, bool) {
	for _, definition := range ShortcutDefinitions {
		if definition.Action == action {
			return definition, true
		}
	}
	return ShortcutDefinition{}, false
}

func (a *App) Shortcut(action string) string {
	return shortcutValue(a.Preferences.Shortcuts, action)
}

func shortcutValue(overrides map[string]string, action string) string {
	if value, ok := overrides[action]; ok {
		return value
	}
	definition, _ := shortcutDefinition(action)
	if action == "fullscreen" && runtime.GOOS != "darwin" {
		return "F11"
	}
	return definition.Default
}

func (a *App) ActionShortcut(c *ui.Context, action string) bool {
	if a.Settings.RecordingShortcut != "" {
		return false
	}
	mods, key, ok := parseShortcut(a.Shortcut(action))
	return ok && c.Shortcut(mods, key)
}

func (a *App) SetShortcut(action, accelerator string) bool {
	definition, ok := shortcutDefinition(action)
	if !ok {
		return false
	}
	if a.Settings.ShortcutErrors == nil {
		a.Settings.ShortcutErrors = make(map[string]string)
	}
	delete(a.Settings.ShortcutErrors, action)
	if accelerator != "" {
		mods, key, valid := parseShortcut(accelerator)
		if !valid {
			a.Settings.ShortcutErrors[action] = a.T("shortcut_invalid")
			return false
		}
		if other, conflict := conflictingShortcut(definition, mods, key, a.Preferences.Shortcuts); conflict {
			a.Settings.ShortcutErrors[action] = a.T("shortcut_conflict", a.T(other.Key))
			return false
		}
	}
	if a.Preferences.Shortcuts == nil {
		a.Preferences.Shortcuts = make(map[string]string)
	}
	defaultValue := definition.Default
	if action == "fullscreen" && runtime.GOOS != "darwin" {
		defaultValue = "F11"
	}
	sameDefault := accelerator == defaultValue
	if !sameDefault {
		mods, key, ok := parseShortcut(accelerator)
		defaultMods, defaultKey, defaultOK := parseShortcut(defaultValue)
		sameDefault = ok && defaultOK && mods == defaultMods && key == defaultKey
	}
	if sameDefault {
		delete(a.Preferences.Shortcuts, action)
	} else {
		a.Preferences.Shortcuts[action] = accelerator
	}
	a.SavePreferences()
	return true
}

func conflictingShortcut(definition ShortcutDefinition, mods ui.Modifiers, key ui.Key, overrides map[string]string) (ShortcutDefinition, bool) {
	for _, other := range ShortcutDefinitions {
		if other.Action == definition.Action || !definition.Global && !other.Global && (definition.Group == "preview") != (other.Group == "preview") {
			continue
		}
		otherMods, otherKey, valid := parseShortcut(shortcutValue(overrides, other.Action))
		if valid && mods == otherMods && key == otherKey {
			return other, true
		}
	}
	return ShortcutDefinition{}, false
}

func (a *App) ResetShortcuts(group string) {
	candidate := maps.Clone(a.Preferences.Shortcuts)
	for _, definition := range ShortcutDefinitions {
		if definition.Group == group {
			delete(candidate, definition.Action)
		}
	}
	conflicts := make(map[string]string)
	for _, definition := range ShortcutDefinitions {
		mods, key, valid := parseShortcut(shortcutValue(candidate, definition.Action))
		if !valid {
			continue
		}
		if other, conflict := conflictingShortcut(definition, mods, key, candidate); conflict {
			conflicts[definition.Action] = a.T("shortcut_conflict", a.T(other.Key))
		}
	}
	if len(conflicts) > 0 {
		if a.Settings.ShortcutErrors == nil {
			a.Settings.ShortcutErrors = make(map[string]string)
		}
		for action, message := range conflicts {
			a.Settings.ShortcutErrors[action] = message
		}
		a.InvalidateWindows()
		return
	}
	a.Preferences.Shortcuts = candidate
	for _, definition := range ShortcutDefinitions {
		if definition.Group == group {
			if a.Settings.RecordingShortcut == definition.Action {
				a.Settings.RecordingShortcut = ""
			}
			delete(a.Settings.ShortcutErrors, definition.Action)
		}
	}
	a.SavePreferences()
}

var shortcutKeyNames = map[ui.Key]string{
	ui.KeyEnter: "Enter", ui.KeyEscape: "Escape", ui.KeyBackspace: "Backspace", ui.KeyTab: "Tab", ui.KeySpace: "Space", ui.KeyDelete: "Delete", ui.KeyInsert: "Insert", ui.KeyHome: "Home", ui.KeyEnd: "End", ui.KeyPageUp: "PageUp", ui.KeyPageDown: "PageDown", ui.KeyLeft: "Left", ui.KeyRight: "Right", ui.KeyUp: "Up", ui.KeyDown: "Down",
	ui.KeyMinus: "-", ui.KeyEqual: "=", ui.KeyComma: ",", ui.KeyPeriod: ".", ui.KeySlash: "/", ui.KeySemicolon: ";", ui.KeyQuote: "'", ui.KeyBracketLeft: "[", ui.KeyBracketRight: "]", ui.KeyBackslash: "\\", ui.KeyBackquote: "`",
}

func init() {
	for i, key := range []ui.Key{ui.KeyA, ui.KeyB, ui.KeyC, ui.KeyD, ui.KeyE, ui.KeyF, ui.KeyG, ui.KeyH, ui.KeyI, ui.KeyJ, ui.KeyK, ui.KeyL, ui.KeyM, ui.KeyN, ui.KeyO, ui.KeyP, ui.KeyQ, ui.KeyR, ui.KeyS, ui.KeyT, ui.KeyU, ui.KeyV, ui.KeyW, ui.KeyX, ui.KeyY, ui.KeyZ} {
		shortcutKeyNames[key] = string(rune('A' + i))
	}
	for i, key := range []ui.Key{ui.Key0, ui.Key1, ui.Key2, ui.Key3, ui.Key4, ui.Key5, ui.Key6, ui.Key7, ui.Key8, ui.Key9} {
		shortcutKeyNames[key] = string(rune('0' + i))
	}
	for i, key := range []ui.Key{ui.KeyF1, ui.KeyF2, ui.KeyF3, ui.KeyF4, ui.KeyF5, ui.KeyF6, ui.KeyF7, ui.KeyF8, ui.KeyF9, ui.KeyF10, ui.KeyF11, ui.KeyF12} {
		shortcutKeyNames[key] = fmt.Sprintf("F%d", i+1)
	}
}

func parseShortcut(accelerator string) (ui.Modifiers, ui.Key, bool) {
	if accelerator == "" {
		return 0, 0, false
	}
	var mods ui.Modifiers
	parts := strings.Split(accelerator, "+")
	for _, part := range parts[:len(parts)-1] {
		switch strings.ToLower(part) {
		case "cmdorctrl", "commandorcontrol":
			mods |= ui.Cmd
		case "cmd", "command", "super", "meta":
			mods |= ui.Super
		case "ctrl", "control":
			mods |= ui.Ctrl
		case "alt", "option":
			mods |= ui.Alt
		case "shift":
			mods |= ui.Shift
		default:
			return 0, 0, false
		}
	}
	name := parts[len(parts)-1]
	for key, keyName := range shortcutKeyNames {
		if strings.EqualFold(name, keyName) {
			return mods, key, true
		}
	}
	return 0, 0, false
}

func recordedAccelerator(event ui.InputEvent) string {
	key := event.Key
	if key == ui.KeyUnknown {
		key = event.PhysicalKey
	}
	name, ok := shortcutKeyNames[key]
	if !ok {
		return ""
	}
	parts := []string{}
	if event.Mods&ui.Super != 0 {
		parts = append(parts, "Super")
	}
	if event.Mods&ui.Ctrl != 0 {
		parts = append(parts, "Ctrl")
	}
	if event.Mods&ui.Alt != 0 {
		parts = append(parts, "Alt")
	}
	if event.Mods&ui.Shift != 0 {
		parts = append(parts, "Shift")
	}
	return strings.Join(append(parts, name), "+")
}

func displayShortcut(value string) string {
	if runtime.GOOS == "darwin" {
		replacer := strings.NewReplacer("CmdOrCtrl+", "⌘", "Super+", "⌘", "Ctrl+", "⌃", "Alt+", "⌥", "Shift+", "⇧", "Backspace", "⌫", "Space", "Space", "Left", "←", "Right", "→", "Up", "↑", "Down", "↓")
		return replacer.Replace(value)
	}
	return strings.ReplaceAll(value, "CmdOrCtrl", "Ctrl")
}

func (a *App) shortcutsSettings(c *ui.Context) {
	ui.SearchField(c.Key("shortcut-search"), &a.Settings.ShortcutSearch).Label(a.T("shortcut_search")).Placeholder(a.T("shortcut_search")).Height(32).Radius(8).FillWidth()
	for _, group := range []string{"global", "file", "edit", "view", "preview"} {
		definitions := []ShortcutDefinition{}
		for _, definition := range ShortcutDefinitions {
			if definition.Group == group && shortcutSearchMatches(a.Settings.ShortcutSearch, a.T(definition.Key), displayShortcut(a.Shortcut(definition.Action))) {
				definitions = append(definitions, definition)
			}
		}
		if len(definitions) == 0 {
			continue
		}
		ui.Column(c.Key(group)).FillWidth().Gap(8).Children(func() {
			ui.Row(c).FillWidth().Padding(0, 4).AlignItems(ui.Center).Children(func() {
				ui.Text(c, a.T("shortcut_group_"+group)+fmt.Sprintf(" (%d)", len(definitions))).FontSize(12).Bold().TextColor(c.Theme().TextMuted).Grow(1)
				if settingsButton(c, a.T("restore_defaults")).Clicked() {
					a.ResetShortcuts(group)
				}
			})
			settingsSection(c, "", "", func() {
				for i, definition := range definitions {
					if i > 0 {
						settingsSeparator(c)
					}
					a.shortcutRow(c.Key(definition.Action), definition)
				}
			})
		})
	}
}

func (a *App) shortcutRow(c *ui.Context, definition ShortcutDefinition) {
	settingsRow(c, func() {
		ui.Text(c, a.T(definition.Key)).Grow(1).MinWidth(0)
		label := displayShortcut(a.Shortcut(definition.Action))
		if label == "" {
			label = a.T("shortcut_record")
		}
		recording := a.Settings.RecordingShortcut == definition.Action
		if recording {
			label = a.T("shortcut_recording")
		}
		button := settingsButton(c.Key("record"), label).MinWidth(132).Border(1, c.Theme().Border).Label(a.T("shortcut_binding", a.T(definition.Key)))
		if recording {
			button.Border(1, c.Theme().Accent).Focus().HandleInput(func(event ui.InputEvent) bool {
				if event.Kind != ui.InputKeyDown {
					return event.Kind == ui.InputText || event.Kind == ui.InputKeyUp
				}
				if event.Key == ui.KeyEscape {
					a.endShortcutRecording()
					return true
				}
				if event.Key == ui.KeyBackspace && event.Mods == 0 {
					a.SetShortcut(definition.Action, "")
					a.endShortcutRecording()
					return true
				}
				accelerator := recordedAccelerator(event)
				if accelerator == "" {
					return true
				}
				if a.SetShortcut(definition.Action, accelerator) {
					a.endShortcutRecording()
				}
				return true
			})
		}
		if button.Clicked() {
			a.Settings.RecordingShortcut = definition.Action
			if a.MenuChanged != nil {
				a.MenuChanged()
			}
		}
	})
	if message := a.Settings.ShortcutErrors[definition.Action]; message != "" {
		settingsError(c, message)
	}
}

func (a *App) endShortcutRecording() {
	a.Settings.RecordingShortcut = ""
	if a.MenuChanged != nil {
		a.MenuChanged()
	}
}

func (a *App) SetShortcutRegistrationError(action string, err error) {
	if a.Settings.ShortcutErrors == nil {
		a.Settings.ShortcutErrors = make(map[string]string)
	}
	if err == nil {
		delete(a.Settings.ShortcutErrors, action)
		return
	}
	slog.Error("注册全局快捷键失败", "action", action, "error", err)
	a.Settings.ShortcutErrors[action] = a.T("shortcut_register_failed", a.Texts.Error(err))
}

func (a *App) CancelShortcutRecording() {
	if a.Settings.RecordingShortcut != "" {
		a.endShortcutRecording()
	}
}
