package ui

import (
	"fmt"
	"image/png"
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

func TestOnboardingLibraryActionCompletesFlow(t *testing.T) {
	a := &App{Texts: i18n.Catalog{Language: "en"}, Preferences: DefaultPreferences(), Welcome: welcomeState{Step: 2}}
	tt := ui.NewTester(a.welcome, 1100, 640)
	if err := tt.Click(a.T("new_library")); err != nil {
		t.Fatal(err)
	}
	if !a.Preferences.Onboarded || !a.NewLibraryOpen {
		t.Fatal("新建图库没有完成引导或打开表单")
	}
}

func TestOnboardingDoneReturnsToWelcome(t *testing.T) {
	a := &App{Texts: i18n.Catalog{Language: "en"}, Preferences: DefaultPreferences(), Welcome: welcomeState{Step: 2}}
	tt := ui.NewTester(a.welcome, 1100, 640)
	if err := tt.Click(a.T("onboarding_finish")); err != nil {
		t.Fatal(err)
	}
	if !a.Preferences.Onboarded || !tt.HasText(a.T("welcome")) || tt.HasText(a.T("onboarding_skip")) {
		t.Fatal("完成后没有退出引导")
	}
}

func TestOnboardingScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_ONBOARDING_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_ONBOARDING_SCREENSHOTS=1 导出引导截图")
	}
	dir := "/tmp/momento-eagle/onboarding/screenshots"
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
				file, err := os.Create(path)
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(file, tt.Image())
				closeErr := file.Close()
				if err != nil {
					t.Fatal(err)
				}
				if closeErr != nil {
					t.Fatal(closeErr)
				}
			}
		}
	}
}
