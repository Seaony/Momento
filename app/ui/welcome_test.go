package ui

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/Seaony/Momento/app/internal/i18n"
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
