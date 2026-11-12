package ui

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/browser"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func renderFinalSettingsScreenshots(t *testing.T, dir string) []screenshotRecord {
	states := []string{"general", "appearance", "sidebar", "controls", "preview", "import", "shortcuts", "notifications", "password", "extension", "about", "login-error", "extension-occupied", "notifications-denied", "shortcut-conflict", "auto-import-missing", "about-available", "about-failed", "password-cooldown"}
	var records []screenshotRecord
	for _, language := range []string{"zh-CN", "en", "ja"} {
		for _, appearance := range []string{"light", "dark"} {
			for _, state := range states {
				name := fmt.Sprintf("settings-%s-%s-%s", state, language, appearance)
				t.Run(name, func(t *testing.T) {
					a, status := finalSettingsScreenshotState(language, appearance, state)
					bar := ui.TitleBar{Height: 44, Left: 80, Bottom: 30}
					tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c, status) }, 760, 560)
					tt.SetDark(appearance == "dark")
					tt.SetLocale(a.Texts.Locale())
					tt.SetTitleBar(bar)
					tt.SetWindowActive(true)
					tt.Frame()
					checkFinalSettingsScreenshot(t, a, tt, status)
					file := name + ".png"
					writeScreenshot(t, dir+"/"+file, tt)
					records = append(records, screenshotRecord{File: file, State: "settings-" + state, Language: language, Appearance: appearance, Width: 760, Height: 560, Platform: "macos", TitleBar: bar, SettingsTabs: finalSettingsTabBounds(t, a, tt)})
				})
			}
		}
	}
	return records
}

func finalSettingsScreenshotState(language, appearance, state string) (*App, UpdateCheckStatus) {
	a := settingsTestApp(language, appearance)
	status := UpdateCheckStatus{State: UpdateIdle}
	for i, key := range []string{"general", "appearance", "sidebar", "controls", "preview", "import", "shortcuts", "notifications", "password", "extension", "about"} {
		if key == state {
			a.Settings.Tab = i
		}
	}
	switch state {
	case "login-error":
		a.Settings.Tab = SettingsGeneral
		a.Settings.LoginError = "permission denied"
	case "extension-occupied":
		a.Settings.Tab = SettingsExtension
		a.ExtensionStatus.State = browser.PortOccupied
		a.ExtensionStatus.Err = errors.New("listen tcp4 127.0.0.1:47641: bind: address already in use")
	case "notifications-denied":
		a.Settings.Tab = SettingsNotifications
		a.Settings.NotificationDenied = true
	case "shortcut-conflict":
		a.Settings.Tab = SettingsShortcuts
		a.Settings.ShortcutErrors = map[string]string{"show_app": a.T("shortcut_conflict", a.T("new_library"))}
	case "auto-import-missing":
		a.Settings.Tab = SettingsImport
	case "password-cooldown":
		a.Settings.Tab = SettingsPassword
		a.Preferences.PasswordHash = "screenshot-only"
		a.Lock.DialogOpen = true
		a.Lock.Mode = "disable"
		a.Lock.ErrorKey = "password_cooldown"
		a.Lock.CooldownUntil = time.Now().Add(30 * time.Second)
	case "about-available":
		a.Settings.Tab = SettingsAbout
		status.State = UpdateAvailable
		status.Version = "2.1.0"
	case "about-failed":
		a.Settings.Tab = SettingsAbout
		status.State = UpdateFailed
		status.Err = errors.New("connection refused")
	}
	if a.Settings.Tab == SettingsImport {
		a.Library = &library.Library{Manifest: model.Manifest{LibraryID: "settings-screenshot"}}
		a.Snapshot.Folders = []model.Folder{{ID: "references", Name: "Design references"}}
		path := "/Users/example/Pictures/Incoming reference images"
		if appearance == "dark" {
			path = "/Users/example/Pictures/Design references for the next collection"
		}
		a.Preferences.AutoImport = map[string][]AutoImportRule{"settings-screenshot": {{Path: path, FolderID: "references", Enabled: true}}}
		if state == "auto-import-missing" {
			a.Settings.AutoImportErrors = map[string]string{path: a.T("auto_import_missing")}
		}
	}
	return a, status
}

func checkFinalSettingsScreenshot(t *testing.T, a *App, tt *ui.Tester, status UpdateCheckStatus) {
	t.Helper()
	for _, tab := range settingsTabs {
		if !tt.HasText(a.T(tab.key)) {
			t.Fatalf("缺少导航%s", tab.key)
		}
	}
	for _, text := range tt.Texts() {
		if strings.Contains(text, "%!") || strings.ContainsAny(text, "⟦⟧") {
			t.Fatalf("文案未翻译或参数错误:%s", text)
		}
	}
	for _, item := range tt.AccessAll() {
		if item.Role == ui.RoleTab && (item.Bounds.X < 0 || item.Bounds.X+item.Bounds.W > 200.1 || item.Bounds.Y+item.Bounds.H > 560.1) {
			t.Fatalf("导航超出侧栏:%+v", item)
		}
	}
	switch a.Settings.Tab {
	case SettingsGeneral:
		if item, ok := tt.AccessByRole(ui.RoleSwitch, a.T("launch_at_login")); !ok || item.Checked != a.Settings.LoginEnabled {
			t.Fatal("登录启动状态不一致")
		}
	case SettingsNotifications:
		if a.Settings.NotificationDenied && !tt.HasText(a.T("notifications_denied")) {
			t.Fatal("拒绝通知状态缺失")
		}
	case SettingsExtension:
		if a.ExtensionStatus.State == browser.PortOccupied && !tt.HasText(a.T("extension_port_in_use")) {
			t.Fatal("扩展占用错误缺失")
		}
	}
}

func settleFinalSettingsScreenshot(t *testing.T, a *App, tt *ui.Tester, status UpdateCheckStatus) int {
	t.Helper()
	tt.SetSize(760, 560)
	tt.Frame()
	return 560
}

func finalSettingsTabBounds(t *testing.T, a *App, tt *ui.Tester) []ui.Rect {
	t.Helper()
	if a.Lock.DialogOpen {
		return nil
	}
	var bounds []ui.Rect
	for i, tab := range settingsTabs {
		item, ok := tt.AccessByRole(ui.RoleTab, a.T(tab.key))
		if !ok || item.Selected != (i == a.Settings.Tab) {
			t.Fatalf("导航选中态错误:%s", tab.key)
		}
		bounds = append(bounds, item.Bounds)
	}
	return bounds
}

func renderFinalTitlebarScreenshots(t *testing.T, fixture screenshotFixture, dir string) []screenshotRecord {
	var records []screenshotRecord
	for _, platform := range chromeTestBars[1:] {
		for _, appearance := range []string{"light", "dark"} {
			for _, state := range []string{"small", "no-sidebar", "no-inspector", "both-hidden", "tags", "welcome"} {
				name := fmt.Sprintf("titlebar-%s-%s-%s", platform.name, state, appearance)
				t.Run(name, func(t *testing.T) {
					a := fixture.app(t, state, "en", appearance)
					switch state {
					case "no-sidebar":
						a.SidebarVisible = false
					case "no-inspector":
						a.InspectorVisible = false
					case "both-hidden":
						a.SidebarVisible, a.InspectorVisible = false, false
					}
					tt := ui.NewTester(a.View, 1100, 640)
					tt.SetTitleBar(platform.bar)
					tt.SetLocale("en")
					tt.SetWindowActive(true)
					file := name + ".png"
					writeScreenshot(t, filepath.Join(dir, file), tt)
					records = append(records, screenshotRecord{File: file, State: state, Language: "en", Appearance: appearance, Width: 1100, Height: 640, Assets: len(a.Snapshot.Assets), Visible: len(a.Visible), Platform: platform.name, TitleBar: platform.bar})
				})
			}
			for tab, state := range []string{"general", "appearance", "sidebar", "controls", "preview", "import", "shortcuts", "notifications", "password", "extension", "about"} {
				name := fmt.Sprintf("titlebar-%s-settings-%s-%s", platform.name, state, appearance)
				t.Run(name, func(t *testing.T) {
					a := settingsTestApp("en", appearance)
					a.Settings.Tab = tab
					status := UpdateCheckStatus{State: UpdateIdle}
					height := a.settingsHeight(status)
					bar := platform.bar
					bar.Height, bar.Bottom = 44, 44
					tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c, status) }, 760, height)
					tt.SetTitleBar(bar)
					tt.SetLocale("en")
					tt.SetWindowActive(true)
					height = settleFinalSettingsScreenshot(t, a, tt, status)
					checkFinalSettingsScreenshot(t, a, tt, status)
					tabs := finalSettingsTabBounds(t, a, tt)
					file := name + ".png"
					writeScreenshot(t, filepath.Join(dir, file), tt)
					records = append(records, screenshotRecord{File: file, State: "settings-" + state, Language: "en", Appearance: appearance, Width: 760, Height: height, Platform: platform.name, TitleBar: bar, SettingsTabs: tabs})
				})
			}
		}
	}
	return records
}
