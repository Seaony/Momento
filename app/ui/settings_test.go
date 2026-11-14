package ui

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
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
	if os.Getenv("MOMENTO_PREVIEW_FOCUS_PROBE") == "1" {
		mygo.App.SetName("Momento Preview Probe")
		mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)
		mygo.App.OnWindowAllClosed(func() {})
		mygo.App.WhenReady(func() { go func() { os.Exit(m.Run()) }() })
		if err := mygo.App.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
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
		a.Settings.Tab = SettingsGeneral
		tt.Frame()
		key := "system_tray_icon"
		if platform == "darwin" {
			key = "tray_icon"
		}
		if !tt.HasText(a.T(key)) || tt.HasText(a.T("dock_when_hidden")) != (platform == "darwin") {
			t.Fatalf("%s的后台设置平台文案错误", platform)
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
	if !a.SetShortcut("search", recordedAccelerator(ui.InputEvent{Key: ui.KeyK, Mods: ui.Cmd})) || len(a.Preferences.Shortcuts) != 0 {
		t.Fatal("录制默认按键不应保留覆盖项")
	}
	a.ResetShortcuts("view")
	if a.Shortcut("search") != "CmdOrCtrl+K" || len(a.Preferences.Shortcuts) != 0 {
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

func TestSettingsAutoImportRuleEditsStayInCurrentLibrary(t *testing.T) {
	a, _ := finalSettingsScreenshotState("en", "light", "import")
	a.Preferences.AutoImport["other"] = []AutoImportRule{{Path: "other-path", Enabled: true}}
	tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c, UpdateCheckStatus{}) }, 760, 560)
	rule := a.Preferences.AutoImport["settings-screenshot"][0]
	toggle, ok := tt.AccessByRole(ui.RoleSwitch, a.T("auto_import_enabled", rule.Path))
	if !ok {
		t.Fatal("缺少监视开关")
	}
	tt.ClickAt(toggle.Bounds.X+toggle.Bounds.W/2, toggle.Bounds.Y+toggle.Bounds.H/2)
	if a.Preferences.AutoImport["settings-screenshot"][0].Enabled {
		t.Fatal("监视开关未保存")
	}
	if err := tt.Click(a.T("remove")); err != nil {
		t.Fatal(err)
	}
	if len(a.Preferences.AutoImport["settings-screenshot"]) != 0 || len(a.Preferences.AutoImport["other"]) != 1 {
		t.Fatal("删除规则影响了其他图库")
	}
}

func TestSettingsNotificationsDisabledKeepChoices(t *testing.T) {
	a := settingsTestApp("en", "dark")
	a.Settings.Tab = SettingsNotifications
	a.Preferences.Notify = false
	tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c, UpdateCheckStatus{}) }, 760, 560)
	for _, key := range []string{"notify_extension", "notify_auto_import", "notify_duplicate"} {
		item, ok := tt.AccessByRole(ui.RoleCheckBox, a.T(key))
		if !ok || !item.Disabled || !item.Checked {
			t.Fatalf("关闭通知应保留事件选择且禁用:%s", key)
		}
	}
	sound, ok := tt.AccessByRole(ui.RoleSwitch, a.T("notify_sound"))
	if !ok || !sound.Disabled || !sound.Checked {
		t.Fatal("关闭通知应保留声音选择且禁用")
	}
}

func TestSettingsSidebarBrowsingSectionsCanBeHidden(t *testing.T) {
	for _, item := range []struct{ key, label string }{{"recent", "recent"}, {"quick", "quick_access"}, {"smart", "smart_folders"}} {
		t.Run(item.key, func(t *testing.T) {
			a := settingsTestApp("en", "light")
			a.Settings.Tab = SettingsSidebar
			tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c, UpdateCheckStatus{}) }, 760, 560)
			control, ok := tt.AccessByRole(ui.RoleCheckBox, a.T(item.label))
			if !ok || control.Disabled || !control.Checked {
				t.Fatal("新增分组应默认显示且可隐藏")
			}
			if err := tt.Click(a.T(item.label)); err != nil {
				t.Fatal(err)
			}
			a.RestoreUIState()
			if !slices.Contains(a.Preferences.SidebarHidden, item.key) {
				t.Fatal("隐藏分组没有保留到偏好")
			}
			tt.Frame()
			if err := tt.Click(a.T(item.label)); err != nil {
				t.Fatal(err)
			}
			if slices.Contains(a.Preferences.SidebarHidden, item.key) {
				t.Fatal("取消隐藏分组没有保存")
			}
		})
	}
}

func TestSettingsSegmentsKeepKeyboardSelection(t *testing.T) {
	a := settingsTestApp("en", "light")
	a.Settings.Tab = SettingsSidebar
	tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c, UpdateCheckStatus{}) }, 760, 560)
	if err := tt.Click(a.T("rename")); err != nil {
		t.Fatal(err)
	}
	if a.Preferences.FolderDoubleClick != "rename" {
		t.Fatal("分段点击没有保存")
	}
	tt.Key(0, ui.KeyRight)
	if a.Preferences.FolderDoubleClick != "expand" {
		t.Fatal("分段方向键没有保存")
	}
}

func TestSettingsZoomUsesMenuAndChangesImmediately(t *testing.T) {
	a := settingsTestApp("en", "light")
	a.Settings.Tab = SettingsAppearance
	tt := ui.NewTester(func(c *ui.Context) { c.SetMenuPresentation(ui.PopoverMenus); a.settingsView(c, UpdateCheckStatus{}) }, 760, 560)
	openSettingsZoomMenu(t, a, tt)
	for _, choice := range []string{"80%", "90%", "100%", "110%", "125%", "150%"} {
		if !tt.HasText(choice) {
			t.Fatalf("缩放菜单缺少%s", choice)
		}
	}
	if err := tt.Click("80%"); err != nil {
		t.Fatal(err)
	}
	if a.Preferences.UIZoom != .8 {
		t.Fatal("缩放菜单没有即时保存")
	}
}

func openSettingsZoomMenu(t *testing.T, a *App, tt *ui.Tester) {
	t.Helper()
	button, ok := tt.AccessByRole(ui.RoleMenuButton, a.T("ui_zoom"))
	if !ok {
		t.Fatal("缩放不是系统菜单按钮")
	}
	tt.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
}

func TestSettingsZoomMenuScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_SETTINGS_CONTROL_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_SETTINGS_CONTROL_SCREENSHOTS=1 生成缩放菜单截图")
	}
	dir := "/tmp/momento-eagle/settings-polish"
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, language := range []string{"zh-CN", "en", "ja"} {
		for _, theme := range []string{"light", "dark"} {
			a := settingsTestApp(language, theme)
			a.Settings.Tab = SettingsAppearance
			tt := ui.NewTester(func(c *ui.Context) { c.SetMenuPresentation(ui.PopoverMenus); a.settingsView(c, UpdateCheckStatus{}) }, 760, 560)
			tt.SetDark(theme == "dark")
			tt.SetLocale(a.Texts.Locale())
			tt.SetTitleBar(ui.TitleBar{Height: 44, Left: 80, Bottom: 30})
			tt.SetWindowActive(true)
			tt.Frame()
			openSettingsZoomMenu(t, a, tt)
			for _, choice := range []string{"80%", "90%", "100%", "110%", "125%", "150%"} {
				if !tt.HasText(choice) {
					t.Fatalf("缩放菜单缺少%s", choice)
				}
			}
			writeScreenshot(t, filepath.Join(dir, fmt.Sprintf("settings-zoom-menu-%s-%s.png", language, theme)), tt)
		}
	}
}

func TestSettingsShortcutResetRejectsCrossGroupConflictsAtomically(t *testing.T) {
	a := settingsTestApp("en", "light")
	for _, change := range [][2]string{{"search", "Alt+G"}, {"import", "CmdOrCtrl+K"}, {"filter", "Alt+H"}} {
		if !a.SetShortcut(change[0], change[1]) {
			t.Fatalf("复现准备失败:%s", change[0])
		}
	}
	before := maps.Clone(a.Preferences.Shortcuts)
	saved := 0
	a.MenuChanged = func() { saved++ }
	a.Settings.RecordingShortcut = "search"
	a.ResetShortcuts("view")
	if !maps.Equal(before, a.Preferences.Shortcuts) || saved != 0 {
		t.Fatal("冲突的整组恢复被保存或部分应用")
	}
	if a.Shortcut("search") != "Alt+G" || a.Shortcut("filter") != "Alt+H" || a.Shortcut("import") != "CmdOrCtrl+K" {
		t.Fatal("冲突改坏了原来的快捷键")
	}
	if a.Settings.ShortcutErrors["search"] != a.T("shortcut_conflict", a.T("import")) {
		t.Fatal("恢复默认未明确指出与导入冲突")
	}
	if a.Settings.RecordingShortcut != "search" {
		t.Fatal("失败的恢复不应清除录制状态")
	}
}

func TestSettingsShortcutResetKeepsPreviewContextOverlap(t *testing.T) {
	a := settingsTestApp("en", "light")
	if !a.SetShortcut("preview_actual", "Alt+J") {
		t.Fatal("无法修改预览快捷键")
	}
	a.ResetShortcuts("preview")
	if a.Shortcut("preview_actual") != "CmdOrCtrl+1" || a.Shortcut("view_masonry") != "CmdOrCtrl+1" || len(a.Preferences.Shortcuts) != 0 || len(a.Settings.ShortcutErrors) != 0 {
		t.Fatal("恢复预览默认不应拦住不同上下文的默认按键")
	}
}

func TestSettingsShortcutResetConflictScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_SETTINGS_CONTROL_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_SETTINGS_CONTROL_SCREENSHOTS=1 生成恢复默认冲突截图")
	}
	dir := "/tmp/momento-eagle/settings-shortcut-reset"
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, language := range []string{"zh-CN", "en", "ja"} {
		for _, theme := range []string{"light", "dark"} {
			a := settingsTestApp(language, theme)
			a.Settings.Tab = SettingsShortcuts
			a.Settings.ShortcutSearch = a.T("search")
			if !a.SetShortcut("search", "Alt+G") || !a.SetShortcut("import", "CmdOrCtrl+K") {
				t.Fatal("无法准备恢复冲突")
			}
			a.ResetShortcuts("view")
			tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c, UpdateCheckStatus{}) }, 760, 560)
			tt.SetDark(theme == "dark")
			tt.SetLocale(a.Texts.Locale())
			tt.SetTitleBar(ui.TitleBar{Height: 44, Left: 80, Bottom: 30})
			tt.SetWindowActive(true)
			tt.Frame()
			if !tt.HasText(a.T("shortcut_conflict", a.T("import"))) || a.Shortcut("search") != "Alt+G" {
				t.Fatal("恢复失败的错误或原快捷键不可见")
			}
			writeScreenshot(t, filepath.Join(dir, fmt.Sprintf("settings-shortcut-reset-%s-%s.png", language, theme)), tt)
		}
	}
}

func TestSettingsPreviewTransparencyOrderAndSelection(t *testing.T) {
	for _, language := range []string{"zh-CN", "en", "ja"} {
		a := settingsTestApp(language, "light")
		a.Settings.Tab = SettingsPreview
		tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c, UpdateCheckStatus{}) }, 760, 560)
		checker, checkerOK := tt.AccessByRole(ui.RoleRadio, a.T("preview_checker"))
		solid, solidOK := tt.AccessByRole(ui.RoleRadio, a.T("preview_solid"))
		if !checkerOK || !solidOK || checker.Bounds.X >= solid.Bounds.X || !checker.Checked || solid.Checked {
			t.Fatal("透明背景应先显示棋盘格并默认选中")
		}
		smooth, smoothOK := tt.AccessByRole(ui.RoleRadio, a.T("preview_smooth"))
		pixelated, pixelatedOK := tt.AccessByRole(ui.RoleRadio, a.T("preview_pixelated"))
		if !smoothOK || !pixelatedOK || smooth.Bounds.X >= pixelated.Bounds.X || !smooth.Checked {
			t.Fatal("放大细节应保持平滑在左并默认选中")
		}
		if err := tt.Click(a.T("preview_solid")); err != nil {
			t.Fatal(err)
		}
		if a.Preferences.PreviewChecker {
			t.Fatal("点击纯色没有保存")
		}
		if err := tt.Click(a.T("preview_checker")); err != nil {
			t.Fatal(err)
		}
		if !a.Preferences.PreviewChecker {
			t.Fatal("点击棋盘格没有保存")
		}
	}
}
