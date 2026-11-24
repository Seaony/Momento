package ui

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/egoist/mygo/ui"
)

func TestWelcomeHasOneHeadingAndCreateAction(t *testing.T) {
	for _, language := range []string{"zh-CN", "en"} {
		t.Run(language, func(t *testing.T) {
			a := &App{Texts: i18n.Catalog{Language: language}, Preferences: Preferences{Onboarded: true}}
			tt := ui.NewTester(a.welcome, 1100, 640)
			if tt.HasText("Momento") || !tt.HasText(a.T("welcome")) || !tt.HasText(a.T("welcome_hint")) {
				t.Fatalf("欢迎页标题或说明不正确：%v", tt.Texts())
			}
			if err := tt.Click(a.T("new_library")); err != nil {
				t.Fatal(err)
			}
			if !a.NewLibraryOpen {
				t.Fatal("新建图库操作未触发")
			}
		})
	}
}

func TestWelcomeScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_UI_FIX_D") != "1" {
		t.Skip("设置 MOMENTO_UI_FIX_D=1 导出欢迎页截图")
	}
	dir := "/tmp/momento-ui-fix-d"
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, language := range []string{"zh-CN", "en"} {
		for _, dark := range []bool{false, true} {
			for _, height := range []int{640, 800} {
				a := &App{Texts: i18n.Catalog{Language: language}, Preferences: Preferences{Onboarded: true}}
				tt := ui.NewTester(a.welcome, 1100, height)
				tt.SetDark(dark)
				theme := "light"
				if dark {
					theme = "dark"
				}
				path := filepath.Join(dir, fmt.Sprintf("welcome-%s-%s-1100x%d.png", language, theme, height))
				f, err := os.Create(path)
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(f, tt.Image())
				closeErr := f.Close()
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

// 引导第一步没有「上一步」；最近图库每一行悬停都有「…」菜单，不只是找不到的。
func TestOnboardingFirstStepHasNoBackAndRecentRowsHaveMenu(t *testing.T) {
	a := &App{Texts: i18n.Catalog{Language: "en"}, Preferences: Preferences{}}
	tt := ui.NewTester(a.welcome, 1100, 640)
	if tt.HasText(a.T("onboarding_previous")) {
		t.Fatal("第一步不该显示上一步")
	}
	a.Welcome.Step = 1
	tt.Frame()
	tt.Advance(time.Second)
	if !tt.HasText(a.T("onboarding_previous")) {
		t.Fatal("第二步应该有上一步")
	}

	dir := t.TempDir()
	b := &App{Texts: i18n.Catalog{Language: "en"}, Preferences: Preferences{Onboarded: true},
		Recent: []library.Recent{{Path: filepath.Join(dir, "Reference.momento"), DisplayName: "Reference"}}, Welcome: welcomeState{RecentChecked: true}}
	wt := ui.NewTester(b.welcome, 1100, 640)
	box, ok := wt.Find("Reference")
	if !ok {
		t.Fatal("欢迎页没有最近图库")
	}
	wt.Move(box.X+box.W/2, box.Y+box.H/2)
	wt.Advance(time.Second)
	if err := wt.Click(b.T("library_more")); err != nil {
		t.Fatal(err)
	}
	wt.Advance(time.Second)
	if !wt.HasText(b.T("remove_recent_library")) || !wt.HasText(b.T("reveal_library")) {
		t.Fatalf("没有找不到标记的图库也应有显示和移除：%v", wt.Texts())
	}
}
