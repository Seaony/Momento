package ui

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

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

func TestSettingsTabsAndControlStates(t *testing.T) {
	for _, language := range []string{"zh-CN", "en", "ja"} {
		t.Run(language, func(t *testing.T) {
			a := settingsTestApp(language, "light")
			tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c, UpdateCheckStatus{State: UpdateIdle}) }, 560, a.settingsHeight(UpdateCheckStatus{State: UpdateIdle}))
			for _, key := range []string{"launch_at_login", "automatic_updates"} {
				if control, ok := tt.AccessByRole(ui.RoleSwitch, a.T(key)); !ok || control.Disabled {
					t.Fatalf("设置开关不可用：%s %+v", key, control)
				}
			}
			if option, ok := tt.AccessByRole(ui.RoleMenuButton, a.T("language")); !ok || option.Bounds.X+option.Bounds.W > 540 {
				t.Fatalf("语言菜单超出窗口：%+v", option)
			}
			if err := tt.Click(a.T("settings_extension")); err != nil {
				t.Fatal(err)
			}
			if a.Settings.Tab != 1 || !tt.HasText(a.T("extension_running")+"  "+browser.Address) {
				t.Fatal("未切换到真实扩展状态")
			}
			a.Settings.ExtensionBusy = true
			tt.Frame()
			if control, ok := tt.AccessByRole(ui.RoleSwitch, a.T("extension_allow")); !ok || !control.Disabled {
				t.Fatal("服务切换期间仍可重复切换")
			}
			if err := tt.Click(a.T("settings_about")); err != nil {
				t.Fatal(err)
			}
			if a.Settings.Tab != 2 || !tt.HasText(a.T("open_logs")) || !tt.HasText("Momento") {
				t.Fatal("关于页内容不完整")
			}
		})
	}
}

func TestSettingsUpdateAndLoginFeedback(t *testing.T) {
	for _, language := range []string{"zh-CN", "en", "ja"} {
		for _, state := range []UpdateCheckState{UpdateChecking, UpdateLatest, UpdateAvailable, UpdateFailed, UpdateUnconfigured} {
			t.Run(language+"/"+string(state), func(t *testing.T) {
				a := settingsTestApp(language, "light")
				status := UpdateCheckStatus{State: state, Version: "2.1.0", Err: errors.New("test failure")}
				tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c, status) }, 560, a.settingsHeight(status))
				if state == UpdateChecking {
					button, ok := tt.AccessByRole(ui.RoleButton, a.T("updates_checking"))
					if !ok || !button.Disabled {
						t.Fatal("检查中按钮没有禁用")
					}
				}
				key := map[UpdateCheckState]string{UpdateChecking: "updates_checking", UpdateLatest: "updates_latest", UpdateAvailable: "updates_available", UpdateFailed: "updates_failed", UpdateUnconfigured: "updates_unconfigured"}[state]
				message := a.T(key)
				if state == UpdateAvailable {
					message = a.T(key, status.Version)
				} else if state == UpdateFailed {
					message = a.T(key, a.Texts.Error(status.Err))
				}
				if !tt.HasText(message) {
					t.Fatalf("没有显示更新结果：%s", message)
				}
			})
		}
		t.Run(language+"/login-error", func(t *testing.T) {
			a := settingsTestApp(language, "dark")
			a.Settings.LoginError = "permission denied"
			tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c, UpdateCheckStatus{State: UpdateIdle}) }, 560, a.SettingsHeight())
			control, ok := tt.AccessByRole(ui.RoleSwitch, a.T("launch_at_login"))
			if !ok || control.Checked || !tt.HasText(a.T("error_launch_at_login", a.Settings.LoginError)) {
				t.Fatal("登录启动失败没有同时显示真实关状态和错误")
			}
		})
	}
}

func TestSettingsWindowScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_SETTINGS_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_SETTINGS_SCREENSHOTS=1 生成设置窗口截图")
	}
	dir := "/tmp/momento-settings-window"
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	var records []map[string]any
	for _, language := range []string{"zh-CN", "en", "ja"} {
		for _, appearance := range []string{"light", "dark"} {
			for _, state := range []string{"general", "extension", "about", "login-on", "login-error", "checking", "latest", "available", "update-failed", "unconfigured", "extension-off", "extension-occupied"} {
				a := settingsTestApp(language, appearance)
				status := UpdateCheckStatus{State: UpdateIdle}
				switch state {
				case "extension":
					a.Settings.Tab = 1
				case "about":
					a.Settings.Tab = 2
				case "login-on":
					a.Settings.LoginEnabled = true
				case "login-error":
					a.Settings.LoginError = "permission denied"
				case "checking":
					status.State = UpdateChecking
				case "latest":
					status.State = UpdateLatest
				case "available":
					status.State, status.Version = UpdateAvailable, "2.1.0"
				case "update-failed":
					status.State, status.Err = UpdateFailed, errors.New("connection refused")
				case "unconfigured":
					status.State = UpdateUnconfigured
				case "extension-off":
					a.Settings.Tab, a.Preferences.BrowserExtensionEnabled = 1, false
					a.ExtensionStatus.State = browser.Disabled
				case "extension-occupied":
					a.Settings.Tab, a.ExtensionStatus.State = 1, browser.PortOccupied
				}
				height := a.settingsHeight(status)
				tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c, status) }, 560, height)
				tt.SetTitleBar(ui.TitleBar{Height: 36, Left: 80, Bottom: 25})
				height = fitSettingsTester(a, tt, status)
				name := fmt.Sprintf("%s-%s-%s.png", state, language, appearance)
				writeScreenshot(t, filepath.Join(dir, name), tt)
				records = append(records, map[string]any{"file": name, "state": state, "language": language, "appearance": appearance, "width": 560, "height": height})
			}
		}
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
}

func openSettingsLanguageMenu(t *testing.T, a *App, tt *ui.Tester) {
	t.Helper()
	menu, ok := tt.AccessByRole(ui.RoleMenuButton, a.T("language"))
	if !ok {
		t.Fatal("没有语言菜单")
	}
	tt.ClickAt(menu.Bounds.X+menu.Bounds.W/2, menu.Bounds.Y+menu.Bounds.H/2)
	if expected := []string{a.T("follow_system"), "English", "简体中文", "日本語"}; !slices.Equal(tt.Menu(), expected) {
		t.Fatalf("语言菜单顺序或语言名称错误：%v", tt.Menu())
	}
}

func TestSettingsAppearanceAndLanguageChangesImmediately(t *testing.T) {
	for _, initialLanguage := range []string{"zh-CN", "en", "ja"} {
		t.Run(initialLanguage, func(t *testing.T) {
			a := settingsTestApp(initialLanguage, "light")
			a.Texts.SystemLanguage = "ja-JP"
			tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c, UpdateCheckStatus{State: UpdateIdle}) }, 560, a.settingsHeight(UpdateCheckStatus{State: UpdateIdle}))
			light := tt.Image().RGBAAt(0, 120)
			if err := tt.Click(a.T("appearance_dark")); err != nil {
				t.Fatal(err)
			}
			dark := tt.Image().RGBAAt(0, 120)
			if a.Preferences.Theme != "dark" || dark.R >= light.R {
				t.Fatal("选择深色后偏好和界面没有立即变深")
			}
			if option, ok := tt.AccessByRole(ui.RoleRadio, a.T("appearance_dark")); !ok || !option.Checked {
				t.Fatal("深色选中状态没有同步")
			}
			if err := tt.Click(a.T("appearance_light")); err != nil {
				t.Fatal(err)
			}
			if a.Preferences.Theme != "light" || tt.Image().RGBAAt(0, 120).R <= dark.R {
				t.Fatal("选择浅色后界面没有立即变浅")
			}
			tt.SetDark(true)
			if err := tt.Click(a.T("follow_system")); err != nil {
				t.Fatal(err)
			}
			if a.Preferences.Theme != "system" || tt.Image().RGBAAt(0, 120).R != dark.R {
				t.Fatal("跟随系统没有使用系统深色")
			}
			for _, item := range []struct{ label, code, locale, appearance string }{
				{"English", "en", "en", "Appearance"},
				{"简体中文", "zh-CN", "zh-Hans", "外观"},
				{"日本語", "ja", "ja", "外観"},
				{a.T("follow_system"), "system", "ja", "外観"},
			} {
				openSettingsLanguageMenu(t, a, tt)
				if item.code == "system" {
					item.label = a.T("follow_system")
				}
				if err := tt.ChooseMenuItem(item.label); err != nil {
					t.Fatal(err)
				}
				if a.Preferences.Language != item.code || a.Texts.Locale() != item.locale || !tt.HasText(item.appearance) {
					t.Fatalf("选择%s后偏好、界面语言或文案没有同步：%s/%s", item.label, a.Preferences.Language, a.Texts.Locale())
				}
			}
		})
	}
}

func TestSettingsAppearanceTooltips(t *testing.T) {
	for _, language := range []string{"zh-CN", "en", "ja"} {
		a := settingsTestApp(language, "light")
		tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c, UpdateCheckStatus{State: UpdateIdle}) }, 560, a.settingsHeight(UpdateCheckStatus{State: UpdateIdle}))
		for _, key := range []string{"follow_system", "appearance_light", "appearance_dark"} {
			label := a.T(key)
			option, ok := tt.AccessByRole(ui.RoleRadio, label)
			if !ok {
				t.Fatalf("没有外观选项%s", label)
			}
			tt.Move(option.Bounds.X+option.Bounds.W/2, option.Bounds.Y+option.Bounds.H/2)
			tt.Advance(700 * time.Millisecond)
			if !tt.HasText(label) {
				t.Fatalf("外观图标悬停没有完整提示：%s/%s", language, label)
			}
		}
	}
}

func TestSettingsFix3Screenshots(t *testing.T) {
	if os.Getenv("MOMENTO_SETTINGS_FIX3_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_SETTINGS_FIX3_SCREENSHOTS=1 生成设置修改单3截图")
	}
	dir := "/tmp/momento-settings-fix-3"
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	var records []map[string]any
	for _, item := range []struct {
		language, appearance string
		dropdown             bool
	}{
		{"zh-CN", "light", false}, {"en", "dark", false}, {"ja", "light", false}, {"zh-CN", "light", true},
	} {
		a := settingsTestApp(item.language, item.appearance)
		status := UpdateCheckStatus{State: UpdateIdle}
		tt := ui.NewTester(func(c *ui.Context) {
			if item.dropdown {
				c.SetMenuPresentation(ui.PopoverMenus)
			}
			a.settingsView(c, status)
		}, 560, a.settingsHeight(status))
		tt.SetTitleBar(ui.TitleBar{Height: 36, Left: 80, Bottom: 25})
		fitSettingsTester(a, tt, status)
		name := fmt.Sprintf("general-%s-%s", item.language, item.appearance)
		if item.dropdown {
			openSettingsLanguageMenu(t, a, tt)
			name += "-dropdown-popover-menu"
		}
		writeScreenshot(t, filepath.Join(dir, name+".png"), tt)
		records = append(records, map[string]any{"file": name + ".png", "language": item.language, "appearance": item.appearance, "dropdown": item.dropdown, "popoverPresentation": item.dropdown, "menu": tt.Menu()})
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
}

func fitSettingsTester(a *App, tt *ui.Tester, status UpdateCheckStatus) int {
	tt.Frame()
	height := a.settingsHeight(status)
	tt.SetSize(560, height)
	tt.Frame()
	return a.settingsHeight(status)
}

func TestSettingsExtensionFailureFeedback(t *testing.T) {
	for _, language := range []string{"zh-CN", "en", "ja"} {
		for _, state := range []browser.State{browser.Failed, browser.PortOccupied} {
			t.Run(language+"/"+string(state), func(t *testing.T) {
				a := settingsTestApp(language, "light")
				a.Settings.Tab = 1
				a.ExtensionStatus.State = state
				a.ExtensionStatus.Err = errors.New("listen tcp4 127.0.0.1:47641: permission denied")
				status := UpdateCheckStatus{State: UpdateIdle}
				tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c, status) }, 560, a.settingsHeight(status))
				fitSettingsTester(a, tt, status)
				key := "extension_failed"
				if state == browser.PortOccupied {
					key = "extension_port_in_use"
				}
				if !tt.HasText(a.T(key)) || tt.HasText(a.T("error")) || !tt.HasText(a.T("error_extension_listen", a.Texts.Error(a.ExtensionStatus.Err))) {
					t.Fatal("扩展失败没有显示具体状态和错误原因")
				}
			})
		}
	}
}

func TestSettingsAboutUpdateActions(t *testing.T) {
	for _, language := range []string{"zh-CN", "en", "ja"} {
		for _, state := range []UpdateCheckState{UpdateIdle, UpdateChecking, UpdateLatest, UpdateAvailable, UpdateFailed, UpdateUnconfigured} {
			t.Run(language+"/"+string(state), func(t *testing.T) {
				a := settingsTestApp(language, "light")
				a.Settings.Tab = 2
				status := UpdateCheckStatus{State: state, Version: "2.1.0", Err: errors.New("connection refused")}
				tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c, status) }, 560, a.settingsHeight(status))
				fitSettingsTester(a, tt, status)
				if state == UpdateAvailable {
					if _, ok := tt.AccessByRole(ui.RoleButton, a.T("update_to_version", status.Version)); !ok {
						t.Fatal("关于页没有带目标版本的更新按钮")
					}
					for _, item := range tt.AccessAll() {
						if item.Role == ui.RoleButton && (item.Name == a.T("check_updates") || item.Name == a.T("update")) {
							t.Fatal("关于页同时显示了多余的检查或更新按钮")
						}
					}
				}
				buttons := 0
				for _, item := range tt.AccessAll() {
					if item.Role == ui.RoleButton {
						buttons++
					}
					if item.Role == ui.RoleStatus {
						version, ok := tt.Find(a.T("current_version", mygo.App.Version()))
						if !ok || item.Bounds.Y < version.Y+version.H {
							t.Fatal("关于页更新结果没有放在版本下方")
						}
					}
				}
				if buttons != 2 {
					t.Fatalf("关于页应只有更新操作和日志两个按钮，实际%d", buttons)
				}
			})
		}
	}
}

func TestSettingsContentHeightFollowsPageAndFeedback(t *testing.T) {
	a := settingsTestApp("en", "light")
	status := UpdateCheckStatus{State: UpdateIdle}
	tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c, status) }, 560, a.settingsHeight(status))
	general := fitSettingsTester(a, tt, status)
	if err := tt.Click(a.T("settings_extension")); err != nil {
		t.Fatal(err)
	}
	extension := fitSettingsTester(a, tt, status)
	if err := tt.Click(a.T("settings_about")); err != nil {
		t.Fatal(err)
	}
	about := fitSettingsTester(a, tt, status)
	if general <= extension || general <= about || extension == about {
		t.Fatalf("窗口没有跟随三个页面的实际内容高度：%d/%d/%d", general, extension, about)
	}
	status.State, status.Err = UpdateFailed, errors.New("connection refused")
	withFeedback := fitSettingsTester(a, tt, status)
	if withFeedback <= about {
		t.Fatal("出现更新反馈后窗口没有给内容增加空间")
	}
	if stable := fitSettingsTester(a, tt, status); stable != withFeedback {
		t.Fatalf("连续布局导致高度循环变化：%d/%d", withFeedback, stable)
	}
}

func TestSettingsFollowSystemJapanese(t *testing.T) {
	a := settingsTestApp("ja", "light")
	if got := a.T("follow_system"); got != "システムに合わせる" {
		t.Fatalf("跟随系统的日语表达错误：%s", got)
	}
}

func TestSettingsFix4Screenshots(t *testing.T) {
	if os.Getenv("MOMENTO_SETTINGS_FIX4_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_SETTINGS_FIX4_SCREENSHOTS=1 生成设置修改单4截图")
	}
	dir := "/tmp/momento-settings-fix-4"
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	var records []map[string]any
	for _, language := range []string{"zh-CN", "en", "ja"} {
		for _, appearance := range []string{"light", "dark"} {
			for _, state := range []string{"general", "general-login-error", "general-update-failed", "extension", "extension-failed", "extension-occupied", "about", "about-checking", "about-latest", "about-available", "about-failed", "about-unconfigured", "language-system-menu", "appearance-system-tooltip"} {
				a := settingsTestApp(language, appearance)
				status := UpdateCheckStatus{State: UpdateIdle}
				switch state {
				case "general-login-error":
					a.Settings.LoginError = "permission denied"
				case "general-update-failed":
					status.State, status.Err = UpdateFailed, errors.New("connection refused")
				case "extension", "extension-failed", "extension-occupied":
					a.Settings.Tab = 1
					if state == "extension-failed" {
						a.ExtensionStatus.State, a.ExtensionStatus.Err = browser.Failed, errors.New("listen tcp4 127.0.0.1:47641: permission denied")
					} else if state == "extension-occupied" {
						a.ExtensionStatus.State, a.ExtensionStatus.Err = browser.PortOccupied, errors.New("listen tcp4 127.0.0.1:47641: bind: address already in use")
					}
				case "about", "about-checking", "about-latest", "about-available", "about-failed", "about-unconfigured":
					a.Settings.Tab = 2
					switch state {
					case "about-checking":
						status.State = UpdateChecking
					case "about-latest":
						status.State = UpdateLatest
					case "about-available":
						status.State, status.Version = UpdateAvailable, "2.1.0"
					case "about-failed":
						status.State, status.Err = UpdateFailed, errors.New("connection refused")
					case "about-unconfigured":
						status.State = UpdateUnconfigured
					}
				case "language-system-menu":
					a.Preferences.Language, a.Texts.Language, a.Texts.SystemLanguage = "system", "system", language
				}
				tt := ui.NewTester(func(c *ui.Context) {
					if state == "language-system-menu" {
						c.SetMenuPresentation(ui.PopoverMenus)
					}
					a.settingsView(c, status)
				}, 560, a.settingsHeight(status))
				tt.SetTitleBar(ui.TitleBar{Height: 36, Left: 80, Bottom: 25})
				height := fitSettingsTester(a, tt, status)
				if state == "language-system-menu" {
					openSettingsLanguageMenu(t, a, tt)
				}
				if state == "appearance-system-tooltip" {
					option, ok := tt.AccessByRole(ui.RoleRadio, a.T("follow_system"))
					if !ok {
						t.Fatal("没有跟随系统外观选项")
					}
					tt.Move(option.Bounds.X+option.Bounds.W/2, option.Bounds.Y+option.Bounds.H/2)
					tt.Advance(700 * time.Millisecond)
				}
				name := fmt.Sprintf("%s-%s-%s.png", state, language, appearance)
				writeScreenshot(t, filepath.Join(dir, name), tt)
				records = append(records, map[string]any{"file": name, "state": state, "language": language, "appearance": appearance, "width": 560, "height": height, "popoverPresentation": state == "language-system-menu"})
			}
		}
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
}
