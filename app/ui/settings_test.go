package ui

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Seaony/Momento/app/internal/browser"
	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func TestMain(m *testing.M) {
	data, err := os.ReadFile("../mygo.json")
	var config struct {
		Version string `json:"version"`
	}
	if err == nil {
		err = json.Unmarshal(data, &config)
	}
	if err == nil && config.Version == "" {
		err = errors.New("应用配置缺少版本号")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	mygo.App.SetVersion(config.Version)
	os.Exit(m.Run())
}

func settingsTestApp(language, appearance string) *App {
	preferences := DefaultPreferences()
	preferences.Language, preferences.Theme = language, appearance
	return &App{Texts: i18n.Catalog{Language: language, SystemLanguage: language}, Preferences: preferences, ExtensionStatus: browser.Status{State: browser.Running, Address: browser.Address}}
}

func TestSettingsPagesAndNavigation(t *testing.T) {
	for _, language := range []string{"zh-CN", "en", "ja"} {
		a := settingsTestApp(language, "light")
		tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c, UpdateCheckStatus{State: UpdateIdle}) }, 760, 560)
		tt.SetTitleBar(ui.TitleBar{Height: 44, Left: 80, Bottom: 30})
		for i, tab := range settingsTabs {
			if err := tt.Click(a.T(tab.key)); err != nil {
				t.Fatal(err)
			}
			if a.Settings.Tab != i {
				t.Fatalf("未进入第%d页", i)
			}
			item, ok := tt.AccessByRole(ui.RoleTab, a.T(tab.key))
			if !ok || !item.Selected {
				t.Fatalf("导航选中态错误: %s", tab.key)
			}
			if a.settingsHeight(UpdateCheckStatus{State: UpdateFailed}) != 560 {
				t.Fatal("设置高度不应随页面变化")
			}
			for _, text := range tt.Texts() {
				if strings.Contains(text, "%!") || strings.ContainsAny(text, "⟦⟧") {
					t.Fatalf("未翻译或参数错误: %s", text)
				}
			}
		}
	}
}

func TestSettingsChangesPreferencesImmediately(t *testing.T) {
	a := settingsTestApp("en", "light")
	a.Settings.Tab = SettingsPreview
	tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c, UpdateCheckStatus{}) }, 760, 560)
	if err := tt.Click(a.T("preview_pixelated")); err != nil {
		t.Fatal(err)
	}
	if !a.Preferences.PreviewPixelated {
		t.Fatal("像素化选择未生效")
	}
	if err := tt.Click(a.T("preview_solid")); err != nil {
		t.Fatal(err)
	}
	if a.Preferences.PreviewChecker {
		t.Fatal("纯色背景选择未生效")
	}
	a.Settings.Tab = SettingsAppearance
	tt.Frame()
	toggle, ok := tt.AccessByRole(ui.RoleSwitch, a.T("sidebar_counts"))
	if !ok {
		t.Fatal("没有侧栏数量开关")
	}
	tt.ClickAt(toggle.Bounds.X+toggle.Bounds.W/2, toggle.Bounds.Y+toggle.Bounds.H/2)
	if a.Preferences.SidebarCounts {
		t.Fatal("侧栏数量开关未生效")
	}
}

func TestSettingsDockRemainsReachable(t *testing.T) {
	before := settingsPlatform
	t.Cleanup(func() { settingsPlatform = before })
	settingsPlatform = func() string { return "darwin" }
	a := settingsTestApp("en", "light")
	a.Preferences.TrayIcon = false
	a.Preferences.DockWhenHidden = false
	tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c, UpdateCheckStatus{}) }, 760, 560)
	toggle, ok := tt.AccessByRole(ui.RoleSwitch, a.T("dock_when_hidden"))
	if !ok || !toggle.Disabled || !toggle.Checked || !a.Preferences.DockWhenHidden {
		t.Fatal("托盘关闭后Dock必须显示且不能关闭")
	}
}

func TestSettingsImportRequiresLibrary(t *testing.T) {
	a := settingsTestApp("en", "light")
	a.Settings.Tab = SettingsImport
	tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c, UpdateCheckStatus{}) }, 760, 560)
	item, ok := tt.AccessByRole(ui.RoleButton, a.T("add_watch_folder"))
	if !ok || !item.Disabled || !tt.HasText(a.T("open_library_first")) {
		t.Fatal("无图库时自动导入应禁用并说明原因")
	}
}

func TestSettingsSidebarFixedEntries(t *testing.T) {
	a := settingsTestApp("en", "light")
	a.Settings.Tab = SettingsSidebar
	tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c, UpdateCheckStatus{}) }, 760, 560)
	for _, key := range []string{"all", "tag_management", "trash"} {
		item, ok := tt.AccessByRole(ui.RoleCheckBox, a.T(key))
		if !ok || !item.Disabled || !item.Checked {
			t.Fatalf("固定项未勾选或可修改:%s", key)
		}
	}
	if err := tt.Click(a.T("favorites")); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(a.Preferences.SidebarHidden, "favorites") {
		t.Fatal("侧栏隐藏项未保存")
	}
}

func TestSettingsQuickLookIsMacOnly(t *testing.T) {
	before := settingsPlatform
	t.Cleanup(func() { settingsPlatform = before })
	for _, platform := range []string{"darwin", "windows", "linux"} {
		settingsPlatform = func() string { return platform }
		a := settingsTestApp("en", "light")
		a.Settings.Tab = SettingsControls
		tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c, UpdateCheckStatus{}) }, 760, 560)
		if tt.HasText(a.T("system_preview")) != (platform == "darwin") {
			t.Fatalf("%s的快速查看选项错误", platform)
		}
	}
}

func TestSettingsShortcutConflictDoesNotSave(t *testing.T) {
	a := settingsTestApp("en", "light")
	if a.SetShortcut("show_app", "CmdOrCtrl+N") || a.Settings.ShortcutErrors["show_app"] == "" {
		t.Fatal("全局快捷键冲突没有拦截")
	}
	if _, exists := a.Preferences.Shortcuts["show_app"]; exists {
		t.Fatal("冲突快捷键被保存")
	}
	if !a.SetShortcut("search", "Alt+G") || a.Shortcut("search") != "Alt+G" {
		t.Fatal("合法快捷键未保存")
	}
	if !a.SetShortcut("search", "") || a.Shortcut("search") != "" {
		t.Fatal("清除快捷键未保存")
	}
	if !a.SetShortcut("search", recordedAccelerator(ui.InputEvent{Key: ui.KeyF, Mods: ui.Cmd})) || len(a.Preferences.Shortcuts) != 0 {
		t.Fatal("录制默认按键不应保留覆盖项")
	}
	a.ResetShortcuts("view")
	if a.Shortcut("search") != "CmdOrCtrl+F" || len(a.Preferences.Shortcuts) != 0 {
		t.Fatal("恢复默认未清除覆盖项")
	}
}

func TestSettingsShortcutRecording(t *testing.T) {
	a := settingsTestApp("en", "light")
	definition, _ := shortcutDefinition("search")
	tt := ui.NewTester(func(c *ui.Context) { a.shortcutRow(c, definition) }, 500, 120)
	if err := tt.Click(a.T("shortcut_binding", a.T("search"))); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Alt, ui.KeyG)
	if a.Shortcut("search") != "Alt+G" || a.Settings.RecordingShortcut != "" {
		t.Fatal("录制组合键未生效")
	}
	if err := tt.Click(a.T("shortcut_binding", a.T("search"))); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEscape)
	if a.Shortcut("search") != "Alt+G" {
		t.Fatal("Esc取消改了快捷键")
	}
	if err := tt.Click(a.T("shortcut_binding", a.T("search"))); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyBackspace)
	if a.Shortcut("search") != "" {
		t.Fatal("Backspace未清除")
	}
}

func TestShortcutDefinitionsAndParsing(t *testing.T) {
	seen := map[string]bool{}
	for _, definition := range ShortcutDefinitions {
		if seen[definition.Action] {
			t.Fatalf("重复动作%s", definition.Action)
		}
		seen[definition.Action] = true
		if definition.Default != "" {
			if _, _, ok := parseShortcut(definition.Default); !ok {
				t.Fatalf("默认快捷键无法解析:%s", definition.Default)
			}
		}
	}
	for _, test := range []struct {
		query, name, key string
		want             bool
	}{{"search", "Search", "⌘F", true}, {"⌘f", "Search", "⌘F", true}, {"import", "Search", "⌘F", false}} {
		if shortcutSearchMatches(test.query, test.name, test.key) != test.want {
			t.Fatal("快捷键搜索未同时按名称和按键匹配")
		}
	}
}

func TestSettingsFeedbackAndAboutActions(t *testing.T) {
	for _, language := range []string{"zh-CN", "en", "ja"} {
		a := settingsTestApp(language, "dark")
		a.Settings.LoginError = "permission denied"
		tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c, UpdateCheckStatus{}) }, 760, 560)
		if !tt.HasText(a.T("error_launch_at_login", a.Settings.LoginError)) {
			t.Fatal("开机启动错误不可见")
		}
		a.Settings.Tab = SettingsExtension
		a.ExtensionStatus.State = browser.PortOccupied
		a.ExtensionStatus.Err = errors.New("bind: address already in use")
		tt.Frame()
		if !tt.HasText(a.T("extension_port_in_use")) || !tt.HasText(a.T("error_extension_listen", a.Texts.Error(a.ExtensionStatus.Err))) {
			t.Fatal("扩展端口错误不可见")
		}
		a.Settings.Tab = SettingsNotifications
		a.Settings.NotificationDenied = true
		tt.Frame()
		if !tt.HasText(a.T("notifications_denied")) {
			t.Fatal("系统拒绝通知错误不可见")
		}
		a.Settings.Tab = SettingsAbout
		status := UpdateCheckStatus{State: UpdateAvailable, Version: "2.1.0"}
		tt = ui.NewTester(func(c *ui.Context) { a.settingsView(c, status) }, 760, 560)
		if !tt.HasText(a.T("update_to_version", status.Version)) || !tt.HasText(a.T("open_logs")) {
			t.Fatal("关于页操作不完整")
		}
	}
}

func TestSettingsWindowScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_SETTINGS_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_SETTINGS_SCREENSHOTS=1 生成设置截图")
	}
	dir := "/tmp/momento-eagle/settings"
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	records := renderFinalSettingsScreenshots(t, dir)
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
}
