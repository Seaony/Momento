package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func polishedSettingsTester(a *App, status UpdateCheckStatus, measure map[string]float32) *ui.Tester {
	tt := ui.NewTester(func(c *ui.Context) {
		c.SetMenuPresentation(ui.PopoverMenus)
		a.settingsView(c, status)
		for text := range measure {
			weight := 400
			if text == a.T(settingsTabs[a.Settings.Tab].key) {
				weight = 600
			}
			measure[text], _ = c.MeasureText(0, ui.Span{Text: text, Font: c.Theme().Font, Size: fontBody, Weight: weight, Lang: c.Locale()})
		}
	}, 760, 560)
	tt.SetDark(a.Preferences.Theme == "dark")
	tt.SetLocale(a.Texts.Locale())
	tt.SetTitleBar(ui.TitleBar{Height: 44, Left: 80, Bottom: 30})
	tt.SetWindowActive(true)
	tt.Frame()
	return tt
}

func TestSettingsPolishLocalizedLayout(t *testing.T) {
	for _, language := range []string{"zh-CN", "en", "ja"} {
		for _, theme := range []string{"light", "dark"} {
			for tab, definition := range settingsTabs {
				t.Run(fmt.Sprintf("%s/%s/%s", language, theme, definition.key), func(t *testing.T) {
					a := settingsTestApp(language, theme)
					if tab == SettingsImport {
						a, _ = finalSettingsScreenshotState(language, theme, "import")
					}
					a.Settings.Tab = tab
					measure := map[string]float32{}
					for _, navigation := range settingsTabs {
						measure[a.T(navigation.key)] = 0
					}
					for _, key := range []string{"close_background", "close_quit", "rename", "folder_expand", "momento_preview", "default_app", "system_preview", "wheel_navigate", "wheel_zoom", "preview_smart", "preview_fit_window", "preview_smooth", "preview_pixelated", "preview_checker", "preview_solid", "all", "tag_management", "trash", "favorites", "uncategorized", "untagged", "recent", "random_browse", "quick_access", "smart_folders", "folders"} {
						measure[a.T(key)] = 0
					}
					tt := polishedSettingsTester(a, UpdateCheckStatus{}, measure)
					checkFinalSettingsScreenshot(t, a, tt, UpdateCheckStatus{})
					for _, navigation := range settingsTabs {
						item, ok := tt.AccessByRole(ui.RoleTab, a.T(navigation.key))
						if !ok || measure[item.Name] > item.Bounds.W-46 {
							t.Fatalf("导航文字被截断: %s", a.T(navigation.key))
						}
					}
					var controls []ui.AccessInfo
					for _, item := range tt.AccessAll() {
						if item.Bounds.W == 0 || item.Bounds.H == 0 || item.Bounds.X < 200 {
							continue
						}
						switch item.Role {
						case ui.RoleRadio, ui.RoleCheckBox:
							padding := float32(24)
							if item.Role == ui.RoleCheckBox {
								padding = 28
							}
							if width, ok := measure[item.Name]; ok && width > item.Bounds.W-padding+1 {
								t.Errorf("选项文字被截断: %q, 文字 %.1f, 可用 %.1f", item.Name, width, item.Bounds.W-padding)
							}
							fallthrough
						case ui.RoleButton, ui.RoleMenuButton, ui.RoleSwitch, ui.RoleTextField:
							if item.Bounds.X < 223 || item.Bounds.X+item.Bounds.W > 737 {
								t.Errorf("控件超出内容区: %s %+v", item.Name, item.Bounds)
							}
							controls = append(controls, item)
						}
					}
					for i, first := range controls {
						for _, second := range controls[i+1:] {
							x := min(first.Bounds.X+first.Bounds.W, second.Bounds.X+second.Bounds.W) - max(first.Bounds.X, second.Bounds.X)
							y := min(first.Bounds.Y+first.Bounds.H, second.Bounds.Y+second.Bounds.H) - max(first.Bounds.Y, second.Bounds.Y)
							if x > 1 && y > 1 {
								t.Errorf("控件互相遮挡: %s / %s", first.Name, second.Name)
							}
						}
					}
					for _, text := range tt.Texts() {
						if strings.ContainsAny(text, "⟦⟧") || strings.Contains(text, "%!") {
							t.Errorf("翻译错误: %s", text)
						}
					}
				})
			}
		}
	}
}

func TestSettingsPolishScreenshots(t *testing.T) {
	batch := os.Getenv("MOMENTO_SETTINGS_POLISH_SCREENSHOTS")
	if batch != "1" && batch != "2" {
		t.Skip("设置 MOMENTO_SETTINGS_POLISH_SCREENSHOTS=1 或 2 分批生成设置截图")
	}
	states := []string{"general", "appearance", "sidebar", "controls", "preview", "import"}
	if batch == "2" {
		states = []string{"shortcuts", "notifications", "password", "extension", "about"}
	}
	dir := filepath.Join("/tmp/momento-polish/shots/settings", "0"+batch)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, theme := range []string{"light", "dark"} {
		for _, state := range states {
			t.Run(theme+"/"+state, func(t *testing.T) {
				a, status := finalSettingsScreenshotState("zh-CN", theme, state)
				tt := polishedSettingsTester(a, status, nil)
				writeScreenshot(t, filepath.Join(dir, fmt.Sprintf("settings-%s-zh-CN-%s.png", state, theme)), tt)
			})
		}
	}
}

func TestSettingsPolishShortcutScrollAndReturn(t *testing.T) {
	for _, language := range []string{"zh-CN", "en", "ja"} {
		for _, theme := range []string{"light", "dark"} {
			a := settingsTestApp(language, theme)
			a.Settings.Tab = SettingsShortcuts
			tt := polishedSettingsTester(a, UpdateCheckStatus{}, nil)
			tt.Scroll(480, 400, 0, 10000)
			label := a.T("shortcut_binding", a.T("next_asset"))
			item, ok := tt.AccessByRole(ui.RoleButton, label)
			if !ok || item.Bounds.Y < 44 || item.Bounds.H < 28 || item.Bounds.Y+item.Bounds.H > 536 {
				t.Fatalf("%s/%s 最后一项快捷键无法滚动到完整可见: %+v", language, theme, item.Bounds)
			}
			if err := tt.Click(label); err != nil {
				t.Fatal(err)
			}
			if err := tt.Click(a.T("settings_preview")); err != nil {
				t.Fatal(err)
			}
			if a.Settings.Tab != SettingsPreview || a.Settings.RecordingShortcut != "" || a.Settings.Scroll.Y != 0 {
				t.Fatal("切换设置页没有复位滚动或取消录制")
			}
		}
	}
}
