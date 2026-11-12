package ui

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/browser"
	"github.com/egoist/mygo/ui"
)

func renderFinalSettingsScreenshots(t *testing.T, dir string) []screenshotRecord {
	states := []string{
		"extension", "about", "login-on", "login-error", "theme-system", "language-system", "automatic-off",
		"checking", "latest", "available", "update-failed", "unconfigured",
		"extension-off", "extension-occupied", "extension-failed",
		"about-checking", "about-available", "about-failed", "about-unconfigured",
		"appearance-system-hover", "appearance-light-hover", "appearance-dark-hover", "language-menu-open",
	}
	var records []screenshotRecord
	for _, state := range states {
		for _, language := range []string{"zh-CN", "en", "ja"} {
			for _, appearance := range []string{"light", "dark"} {
				name := fmt.Sprintf("settings-%s-%s-%s", state, language, appearance)
				t.Run(name, func(t *testing.T) {
					a, status := finalSettingsScreenshotState(language, appearance, state)
					height := a.settingsHeight(status)
					bar := ui.TitleBar{Height: 36, Left: 80, Bottom: 25}
					tt := ui.NewTester(func(c *ui.Context) {
						if state == "language-menu-open" {
							c.SetMenuPresentation(ui.PopoverMenus)
						}
						a.settingsView(c, status)
					}, 560, height)
					tt.SetDark(appearance == "dark")
					tt.SetLocale(a.Texts.Locale())
					tt.SetTitleBar(bar)
					tt.SetWindowActive(true)
					checkFinalSettingsScreenshot(t, a, tt, status)
					menuPresentation := ""
					if state == "language-menu-open" {
						button, ok := tt.AccessByRole(ui.RoleMenuButton, a.T("language"))
						if !ok {
							t.Fatal("找不到语言菜单")
						}
						tt.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
						for _, label := range []string{a.T("follow_system"), "English", "简体中文", "日本語"} {
							if !tt.HasText(label) {
								t.Fatalf("实际菜单模型展开后缺少%s", label)
							}
						}
						menuPresentation = "framework-popover-from-system-menu-model"
					}
					if appearance, hover := strings.CutPrefix(state, "appearance-"); hover {
						appearance = strings.TrimSuffix(appearance, "-hover")
						key := "appearance_" + appearance
						if appearance == "system" {
							key = "follow_system"
						}
						option, ok := tt.AccessByRole(ui.RoleRadio, a.T(key))
						if !ok {
							t.Fatalf("找不到外观选项%s", key)
						}
						tt.Move(option.Bounds.X+option.Bounds.W/2, option.Bounds.Y+option.Bounds.H/2)
						tt.Advance(time.Second)
						if !tt.HasText(a.T(key)) {
							t.Fatalf("外观图标悬停没有完整提示%s", key)
						}
					}
					file := name + ".png"
					writeScreenshot(t, filepath.Join(dir, file), tt)
					records = append(records, screenshotRecord{File: file, State: "settings-" + state, Language: language, Appearance: appearance, Width: 560, Height: height, Platform: "macos", TitleBar: bar, MenuPresentation: menuPresentation})
				})
			}
		}
	}
	return records
}

func finalSettingsScreenshotState(language, appearance, state string) (*App, UpdateCheckStatus) {
	a := settingsTestApp(language, appearance)
	status := UpdateCheckStatus{State: UpdateIdle}
	if strings.HasPrefix(state, "about") {
		a.Settings.Tab = 2
		state = strings.TrimPrefix(state, "about-")
	}
	switch state {
	case "extension":
		a.Settings.Tab = 1
	case "login-on":
		a.Settings.LoginEnabled = true
	case "login-error":
		a.Settings.LoginError = "permission denied"
	case "theme-system":
		a.Preferences.Appearance = "system"
	case "language-system":
		a.Preferences.Language, a.Texts.Language = "system", "system"
	case "automatic-off":
		a.Preferences.AutoCheckUpdates = false
	case "checking":
		status.State = UpdateChecking
	case "latest":
		status.State = UpdateLatest
	case "available":
		status.State, status.Version = UpdateAvailable, "2.1.0"
	case "update-failed", "failed":
		status.State, status.Err = UpdateFailed, errors.New("connection refused")
	case "unconfigured":
		status.State = UpdateUnconfigured
	case "extension-off":
		a.Settings.Tab, a.Preferences.BrowserExtensionEnabled = 1, false
		a.ExtensionStatus.State = browser.Disabled
	case "extension-occupied":
		a.Settings.Tab, a.ExtensionStatus.State = 1, browser.PortOccupied
	case "extension-failed":
		a.Settings.Tab, a.ExtensionStatus.State = 1, browser.Failed
		a.ExtensionStatus.Err = errors.New("listen tcp4 127.0.0.1:47641: permission denied")
	}
	return a, status
}

func checkFinalSettingsScreenshot(t *testing.T, a *App, tt *ui.Tester, status UpdateCheckStatus) {
	t.Helper()
	expect := func(text string) {
		if !tt.HasText(text) {
			t.Fatalf("设置截图缺少%q，实际%v", text, tt.Texts())
		}
	}
	for _, tab := range settingsTabs {
		expect(a.T(tab.key))
	}
	switch a.Settings.Tab {
	case 0:
		for _, key := range []string{"launch_at_login", "launch_at_login_hint", "appearance_language", "appearance", "language", "automatic_updates"} {
			expect(a.T(key))
		}
		login, ok := tt.AccessByRole(ui.RoleSwitch, a.T("launch_at_login"))
		if !ok || login.Checked != a.Settings.LoginEnabled {
			t.Fatal("开机启动截图开关与真实状态不一致")
		}
		if a.Settings.LoginError != "" {
			expect(a.T("error_launch_at_login", a.Settings.LoginError))
		}
		automatic, ok := tt.AccessByRole(ui.RoleSwitch, a.T("automatic_updates"))
		if !ok || automatic.Checked != a.Preferences.AutoCheckUpdates {
			t.Fatal("自动检查更新截图开关与偏好不一致")
		}
		picker, ok := tt.AccessByRole(ui.RoleMenuButton, a.T("language"))
		if !ok || picker.Bounds.W <= 0 || picker.Bounds.X < 20 || picker.Bounds.X+picker.Bounds.W > 540 {
			t.Fatalf("语言菜单超出560宽窗口：%+v", picker)
		}
		label := map[string]string{"system": a.T("follow_system"), "en": "English", "zh-CN": "简体中文", "ja": "日本語"}[a.Preferences.Language]
		expect(label)
	case 1:
		expect(a.T("extension_allow"))
		expect(a.T("extension_local_only"))
		allow, ok := tt.AccessByRole(ui.RoleSwitch, a.T("extension_allow"))
		if !ok || allow.Checked != a.Preferences.BrowserExtensionEnabled {
			t.Fatal("浏览器扩展截图开关与偏好不一致")
		}
		switch a.ExtensionStatus.State {
		case browser.Running:
			expect(a.T("extension_running") + "  " + browser.Address)
		case browser.Disabled:
			expect(a.T("extension_disabled"))
		case browser.PortOccupied:
			expect(a.T("extension_port_in_use"))
		case browser.Failed:
			expect(a.T("error_extension_listen", a.Texts.Error(a.ExtensionStatus.Err)))
		}
	case 2:
		expect("Momento")
		expect(a.T("open_logs"))
	}
	if a.Settings.Tab != 1 {
		expect(a.T("current_version", "2.0.0"))
		if status.State == UpdateChecking {
			button, ok := tt.AccessByRole(ui.RoleButton, a.T("updates_checking"))
			if !ok || !button.Disabled {
				t.Fatal("更新检查中的按钮应禁用")
			}
		} else {
			expect(a.T("check_updates"))
		}
		switch status.State {
		case UpdateLatest:
			expect(a.T("updates_latest"))
		case UpdateAvailable:
			expect(a.T("updates_available", status.Version))
			expect(a.T("update"))
		case UpdateFailed:
			expect(a.T("updates_failed", a.Texts.Error(status.Err)))
		case UpdateUnconfigured:
			expect(a.T("updates_unconfigured"))
		}
	}
	for _, text := range tt.Texts() {
		if strings.Contains(text, "%!") || strings.ContainsAny(text, "⟦⟧") {
			t.Fatalf("设置截图出现未翻译文案或参数错误：%s", text)
		}
	}
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
			for tab, state := range []string{"general", "extension", "about"} {
				name := fmt.Sprintf("titlebar-%s-settings-%s-%s", platform.name, state, appearance)
				t.Run(name, func(t *testing.T) {
					a := settingsTestApp("en", appearance)
					a.Settings.Tab = tab
					status := UpdateCheckStatus{State: UpdateIdle}
					height := a.settingsHeight(status)
					bar := platform.bar
					bar.Height, bar.Bottom = 36, 36
					tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c, status) }, 560, height)
					tt.SetTitleBar(bar)
					tt.SetLocale("en")
					tt.SetWindowActive(true)
					checkFinalSettingsScreenshot(t, a, tt, status)
					file := name + ".png"
					writeScreenshot(t, filepath.Join(dir, file), tt)
					records = append(records, screenshotRecord{File: file, State: "settings-" + state, Language: "en", Appearance: appearance, Width: 560, Height: height, Platform: platform.name, TitleBar: bar})
				})
			}
		}
	}
	return records
}
