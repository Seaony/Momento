package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/egoist/mygo/ui"
)

func TestOnboardingEligibilityAndPersistentExit(t *testing.T) {
	a := preferenceTestApp(t)
	if !a.needsOnboarding() {
		t.Fatal("首次启动没有显示引导")
	}
	a.Recent = []library.Recent{{Path: "existing.momento"}}
	if a.needsOnboarding() {
		t.Fatal("已有图库仍显示引导")
	}
	a.Recent = nil
	a.finishOnboarding()
	if err := a.FlushPreferences(); err != nil {
		t.Fatal(err)
	}
	reopened := &App{Preferences: readTestPreferences(t, a.configDir)}
	if reopened.needsOnboarding() {
		t.Fatal("完成引导后再次启动仍出现")
	}
}

func TestOnboardingNavigationAndSkip(t *testing.T) {
	for _, language := range []string{"zh-Hans", "en", "ja"} {
		t.Run(language, func(t *testing.T) {
			a := &App{Texts: i18n.Catalog{Language: language}, Preferences: DefaultPreferences()}
			tt := ui.NewTester(a.welcome, 1100, 640)
			if !tt.HasText(a.T("onboarding_theme")) {
				t.Fatal("第一步没有显示主题选择")
			}
			if err := tt.Click(a.T("onboarding_next")); err != nil {
				t.Fatal(err)
			}
			if a.Welcome.Step != 1 || !tt.HasText(a.T("onboarding_extension_hint")) {
				t.Fatal("下一步没有显示扩展介绍")
			}
			if err := tt.Click(a.T("onboarding_previous")); err != nil {
				t.Fatal(err)
			}
			if a.Welcome.Step != 0 {
				t.Fatal("上一步没有回到主题")
			}
			if err := tt.Click(a.T("onboarding_skip")); err != nil {
				t.Fatal(err)
			}
			if !a.Preferences.Onboarded || !tt.HasText(a.T("welcome")) {
				t.Fatal("跳过后没有显示正常欢迎页")
			}
		})
	}
}

func TestOnboardingOpeningNewLibraryDialogDoesNotCompleteFlow(t *testing.T) {
	a := preferenceTestApp(t)
	a.Welcome.Step = 2
	tt := ui.NewTester(func(c *ui.Context) { a.welcome(c); a.newLibraryDialog(c) }, 1100, 640)
	if err := tt.Click(a.T("new_library")); err != nil {
		t.Fatal(err)
	}
	if !a.NewLibraryOpen {
		t.Fatal("没有打开新建图库表单")
	}
	if a.Preferences.Onboarded {
		t.Fatal("仅打开新建表单就提前完成了引导")
	}
	if err := tt.Click(a.T("cancel")); err != nil {
		t.Fatal(err)
	}
	if a.NewLibraryOpen || a.Preferences.Onboarded || !tt.HasText(a.T("onboarding_library")) {
		t.Fatal("取消新建图库后没有留在最后一步")
	}
	if err := a.FlushPreferences(); err != nil {
		t.Fatal(err)
	}
	if readTestPreferences(t, a.configDir).Onboarded {
		t.Fatal("取消后的偏好仍把引导记为完成")
	}
}

func TestOnboardingLastStepHasOnlyLibraryActionsAndBack(t *testing.T) {
	for _, language := range []string{"zh-Hans", "en", "ja"} {
		t.Run(language, func(t *testing.T) {
			a := &App{Texts: i18n.Catalog{Language: language}, Preferences: DefaultPreferences(), Welcome: welcomeState{Step: 2}}
			tt := ui.NewTester(a.welcome, 1100, 640)
			for _, key := range []string{"onboarding_library", "welcome_hint", "new_library", "open_library", "onboarding_previous"} {
				if !tt.HasText(a.T(key)) {
					t.Errorf("最后一步缺少%s", key)
				}
			}
			if tt.HasText(a.T("welcome")) {
				t.Error("最后一步仍用空图库标题")
			}
			allowed := map[string]bool{a.T("new_library"): true, a.T("open_library"): true, a.T("onboarding_previous"): true}
			buttons := 0
			for _, control := range tt.AccessAll() {
				if control.Role == ui.RoleButton {
					buttons++
					if !allowed[control.Name] {
						t.Errorf("最后一步不应显示按钮：%s", control.Name)
					}
				}
			}
			if buttons != len(allowed) {
				t.Errorf("最后一步应只有三个操作，实际%d个", buttons)
			}
			if err := tt.Click(a.T("onboarding_previous")); err != nil {
				t.Fatal(err)
			}
			if a.Welcome.Step != 1 || a.Preferences.Onboarded {
				t.Fatal("上一步没有回到扩展页，或提前完成了引导")
			}
		})
	}
}

func TestOnboardingScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_ONBOARDING_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_ONBOARDING_SCREENSHOTS=1 导出引导截图")
	}
	dir := "/tmp/momento-eagle/onboarding/review-fix/screenshots"
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, language := range []string{"zh-Hans", "en", "ja"} {
		for _, theme := range []string{"light", "dark"} {
			for step := range 3 {
				a := &App{Texts: i18n.Catalog{Language: language}, Preferences: DefaultPreferences(), Welcome: welcomeState{Step: step}}
				a.Preferences.Theme = theme
				tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); a.welcome(c) }, 1100, 640)
				path := filepath.Join(dir, fmt.Sprintf("%s-%s-step-%d.png", language, theme, step+1))
				writeScreenshot(t, path, tt)
			}
			settings := settingsTestApp(language, theme)
			settings.Settings.Tab = 1
			tt := ui.NewTester(settings.SettingsView, 760, 560)
			tt.SetTitleBar(ui.TitleBar{Height: 44, Left: 80, Bottom: 30})
			writeScreenshot(t, filepath.Join(dir, fmt.Sprintf("%s-%s-settings-appearance.png", language, theme)), tt)
		}
	}
}
