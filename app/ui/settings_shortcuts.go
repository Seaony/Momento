package ui

import (
	"fmt"
	"log/slog"
	"maps"
	"runtime"
	"strings"

	"github.com/egoist/mygo/ui"
)

// shortcutClaim 记着没录成的那次按键：Other 是占着这个按键的另一项，为空表示不是冲突。
type shortcutClaim struct{ Accelerator, Other string }

type shortcutGroupNote struct {
	Text  string
	Error bool
}

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
	{"copy_tags", "copy_tags", "edit", "CmdOrCtrl+Shift+C", false},
	{"paste_tags", "paste_tags", "edit", "CmdOrCtrl+Shift+V", false},
	{"add_to_folder", "add_to_folder", "edit", "Shift+F", false},
	{"select_all", "select_all", "edit", "CmdOrCtrl+A", false},
	{"delete", "delete", "edit", "Backspace", false},
	{"view_masonry", "view_masonry", "view", "CmdOrCtrl+1", false},
	{"view_justified", "view_justified", "view", "CmdOrCtrl+2", false},
	{"view_list", "view_list", "view", "CmdOrCtrl+3", false},
	{"slideshow", "slideshow", "view", "", false},
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
	{"preview_flip", "preview_flip", "preview", "H", false},
	{"preview_grayscale", "preview_grayscale", "preview", "G", false},
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
	delete(a.Settings.ShortcutPending, action)
	a.Settings.ShortcutGroupNotes = nil
	reject := func(message string, claim shortcutClaim) bool {
		a.Settings.ShortcutErrors[action] = message
		if a.Settings.ShortcutPending == nil {
			a.Settings.ShortcutPending = make(map[string]shortcutClaim)
		}
		a.Settings.ShortcutPending[action] = claim
		return false
	}
	if accelerator != "" {
		mods, key, valid := parseShortcut(accelerator)
		if !valid {
			return reject(a.T("shortcut_invalid"), shortcutClaim{})
		}
		// 注册到系统的全局快捷键没有修饰键，会截走整个系统里的这个按键。
		if definition.Global && mods&(ui.Cmd|ui.Super|ui.Ctrl|ui.Alt) == 0 {
			text := a.T("shortcut_global_modifier")
			if runtime.GOOS != "darwin" {
				text = a.T("shortcut_global_modifier_other")
			}
			return reject(text, shortcutClaim{})
		}
		if other, conflict := conflictingShortcut(definition, mods, key, a.Preferences.Shortcuts); conflict {
			return reject(a.T("shortcut_conflict_in_use", displayShortcut(accelerator), a.T(other.Key)), shortcutClaim{Accelerator: accelerator, Other: other.Action})
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

// 预览窗口里才生效的快捷键；「空格预览」虽然列在预览组，但是在主界面按的，算主界面。
func previewScoped(definition ShortcutDefinition) bool {
	return definition.Group == "preview" && definition.Action != "preview"
}

func conflictingShortcut(definition ShortcutDefinition, mods ui.Modifiers, key ui.Key, overrides map[string]string) (ShortcutDefinition, bool) {
	for _, other := range ShortcutDefinitions {
		if other.Action == definition.Action || !definition.Global && !other.Global && previewScoped(definition) != previewScoped(other) {
			continue
		}
		otherMods, otherKey, valid := parseShortcut(shortcutValue(overrides, other.Action))
		if valid && mods == otherMods && key == otherKey {
			return other, true
		}
	}
	return ShortcutDefinition{}, false
}

// 菜单项实际挂的快捷键：录快捷键时全部摘掉；预览没开时摘掉预览里的；
// 预览开着时，主界面菜单项如果和预览里的实际快捷键相同就摘掉，让按键交给预览（如列表视图改成 ⌘0 时，⌘0 仍是适合窗口）。
func (a *App) MenuAccelerator(action string) string {
	definition, ok := shortcutDefinition(action)
	accelerator := a.Shortcut(action)
	if !ok || a.Settings.RecordingShortcut != "" {
		return ""
	}
	if previewScoped(definition) {
		if !a.PreviewOpen() {
			return ""
		}
		return accelerator
	}
	mods, key, valid := parseShortcut(accelerator)
	if !valid || definition.Global || !a.PreviewOpen() {
		return accelerator
	}
	for _, other := range ShortcutDefinitions {
		if !previewScoped(other) {
			continue
		}
		otherMods, otherKey, otherValid := parseShortcut(a.Shortcut(other.Action))
		// 预览里放大也认 ⇧= 和 =，和预览窗口处理按键时一致。
		if otherValid && otherKey == key && (otherMods == mods || other.Action == "preview_zoom_in" && otherKey == ui.KeyEqual && otherMods|ui.Shift == mods) {
			return ""
		}
	}
	return accelerator
}

// 这一组里有没有改过的快捷键。
func (a *App) groupHasCustomShortcut(group string) bool {
	for _, definition := range ShortcutDefinitions {
		if _, custom := a.Preferences.Shortcuts[definition.Action]; custom && definition.Group == group {
			return true
		}
	}
	return false
}

func (a *App) ResetShortcuts(group string) {
	candidate := maps.Clone(a.Preferences.Shortcuts)
	for _, definition := range ShortcutDefinitions {
		if definition.Group == group {
			delete(candidate, definition.Action)
		}
	}
	for _, definition := range ShortcutDefinitions {
		if definition.Group != group {
			continue
		}
		mods, key, valid := parseShortcut(shortcutValue(candidate, definition.Action))
		if !valid {
			continue
		}
		if other, conflict := conflictingShortcut(definition, mods, key, candidate); conflict {
			// 恢复默认会和别处冲突：整组都不恢复，在组标题旁说清楚。
			if a.Settings.ShortcutGroupNotes == nil {
				a.Settings.ShortcutGroupNotes = make(map[string]shortcutGroupNote)
			}
			a.Settings.ShortcutGroupNotes[group] = shortcutGroupNote{Text: a.T("shortcut_restore_conflict", a.T(definition.Key), a.T(other.Key)), Error: true}
			a.InvalidateWindows()
			return
		}
	}
	a.Preferences.Shortcuts = candidate
	for _, definition := range ShortcutDefinitions {
		if definition.Group == group {
			if a.Settings.RecordingShortcut == definition.Action {
				a.Settings.RecordingShortcut = ""
			}
			delete(a.Settings.ShortcutErrors, definition.Action)
			delete(a.Settings.ShortcutPending, definition.Action)
		}
	}
	a.Settings.ShortcutGroupNotes = map[string]shortcutGroupNote{group: {Text: a.T("shortcut_restored")}}
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
	p := paletteFor(c.Theme())
	field, _ := searchField(c.Key("shortcut-search"), &a.Settings.ShortcutSearch, a.T("shortcut_search"), a.T("clear_search"))
	field.FillWidth()
	matched := false
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
		matched = true
		ui.Column(c.Key(group)).FillWidth().Gap(space8).Children(func() {
			// 标题和其他设置分组一样；「恢复默认」是和「清除筛选」同款的小号文字按钮，只在这一组有改动时出现。
			ui.Row(c).FillWidth().Padding(0, 0, 0, space4).Gap(space8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, a.T("shortcut_group_"+group)).FontSize(fontGroup).FontWeight(600).LetterSpacing(.6).TextColor(p.TextSecondary).Grow(1).MinWidth(0)
				if note, ok := a.Settings.ShortcutGroupNotes[group]; ok {
					color := p.Success
					if note.Error {
						color = p.Danger
					}
					fadeMotion(ui.Text(c.Key("group-note"), note.Text).FontSize(fontSecondary).TextColor(color).SingleLine().ContentTransition(replaceMotion)).Role(ui.RoleStatus)
				}
				if !a.groupHasCustomShortcut(group) {
					return
				}
				reset := ui.ButtonBase(c.Key("reset")).Height(20).Padding(0, popClearPad).Radius(radiusSmall).Cursor(ui.CursorPointer).Label(a.T("restore_defaults"))
				background, text := ui.Transparent, p.TextTertiary
				if reset.Pressed() {
					background, text = p.Pressed, p.Text
				} else if reset.Hovered() {
					background, text = p.Hover, p.Text
				}
				animateColors(reset, background, text)
				reset.Children(func() { ui.Text(c, a.T("restore_defaults")).FontSize(fontGroup).SingleLine() })
				if reset.Clicked() {
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
	if !matched {
		ui.Text(c.Key("shortcut-no-match"), a.T("shortcut_no_match")).FontSize(fontSecondary).TextColor(p.TextTertiary).FillWidth().TextAlign(ui.Center).Role(ui.RoleStatus)
	}
}

func (a *App) shortcutRow(c *ui.Context, definition ShortcutDefinition) {
	p := paletteFor(c.Theme())
	recording := a.Settings.RecordingShortcut == definition.Action
	var record ui.Element
	settingsRow(c, func() {
		ui.Text(c, a.T(definition.Key)).Grow(1).MinWidth(0)
		label := displayShortcut(a.Shortcut(definition.Action))
		if label == "" {
			label = a.T("shortcut_record")
		}
		if recording {
			label = a.T("shortcut_recording")
		}
		button := settingsMenuButton(c.Key("record"), label).Padding(0, space12).Justify(ui.Center).MinWidth(132).
			Label(a.T("shortcut_binding", a.T(definition.Key))).Children(func() {
			ui.Text(c, label).FontSize(fontBody).SingleLine()
		})
		record = button
		if recording {
			button.Border(1, p.Accent).Focus().HandleInput(func(event ui.InputEvent) bool {
				if event.Kind != ui.InputKeyDown {
					return event.Kind == ui.InputText || event.Kind == ui.InputKeyUp
				}
				if event.Key == ui.KeyEscape {
					a.cancelShortcutRecording()
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
			a.clearShortcutPending()
			a.Settings.RecordingShortcut = definition.Action
			if a.MenuChanged != nil {
				a.MenuChanged()
			}
		}
	})
	claimPressed := false
	if message := a.Settings.ShortcutErrors[definition.Action]; message != "" {
		claim := a.Settings.ShortcutPending[definition.Action]
		revealMotion(ui.Row(c.Key("settings-error"))).FillWidth().Padding(space4, space16, space12, space16).Gap(space12).AlignItems(ui.Center).Children(func() {
			ui.Text(c, message).FontSize(fontSecondary).TextColor(p.Danger).Grow(1).MinWidth(0).Role(ui.RoleAlert)
			if claim.Other == "" {
				return
			}
			// 点这个按钮不算「点了别处」，也不能让录制按钮丢焦点。
			use := settingsButton(c.Key("use-anyway"), a.T("shortcut_use_anyway")).Height(controlSmall).KeepFocus()
			claimPressed = use.Pressed()
			if use.Clicked() {
				claimPressed = true
				a.claimShortcut(definition.Action)
			}
		})
	}
	// 按钮失焦、点了别处、设置窗口失焦，都算取消录制，免得菜单和快捷键一直被摘着。
	if recording && a.Settings.RecordingShortcut == definition.Action && (record.PressedOutside() && !claimPressed || record.FocusLeft() || !c.WindowActive()) {
		a.cancelShortcutRecording()
		a.InvalidateWindows()
	}
}

// 把按下的组合键从占着它的那一项上抢过来：对方清空，当前项设上。
func (a *App) claimShortcut(action string) {
	claim, ok := a.Settings.ShortcutPending[action]
	if !ok || claim.Other == "" {
		return
	}
	a.SetShortcut(claim.Other, "")
	if a.SetShortcut(action, claim.Accelerator) {
		a.endShortcutRecording()
	}
}

// 没录成的提示（冲突、不合规）随下一次录制、取消、换页一起清掉。
func (a *App) clearShortcutPending() {
	for action := range a.Settings.ShortcutPending {
		delete(a.Settings.ShortcutErrors, action)
	}
	a.Settings.ShortcutPending = nil
}

func (a *App) endShortcutRecording() {
	a.Settings.RecordingShortcut = ""
	if a.MenuChanged != nil {
		a.MenuChanged()
	}
}

// 取消录制：这一行没录成的提示一并清掉。
func (a *App) cancelShortcutRecording() {
	if action := a.Settings.RecordingShortcut; action != "" {
		if _, pending := a.Settings.ShortcutPending[action]; pending {
			delete(a.Settings.ShortcutErrors, action)
			delete(a.Settings.ShortcutPending, action)
		}
	}
	a.endShortcutRecording()
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
		a.cancelShortcutRecording()
	}
}
