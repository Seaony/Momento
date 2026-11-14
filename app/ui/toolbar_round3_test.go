package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestToolbarRound3LocalizedTextFits(t *testing.T) {
	for _, language := range []string{"en", "ja"} {
		for _, theme := range []string{"light", "dark"} {
			t.Run(language+"/"+theme, func(t *testing.T) {
				a := toolbarTestApp()
				a.Texts.Language = language
				a.Preferences.Theme = theme
				a.SidebarVisible, a.InspectorVisible = false, false
				var importWidth, searchWidth, lineHeight float32
				tt := ui.NewTester(func(c *ui.Context) {
					a.applyTheme(c)
					ui.Column(c).Fill().Children(func() { a.toolbar(c); ui.Spacer(c) })
					importWidth, lineHeight = c.MeasureText(0, ui.Span{Text: a.T("import"), Font: c.Theme().Font, Size: fontBody, Weight: 700, Lang: language})
					searchWidth, _ = c.MeasureText(0, ui.Span{Text: a.T("search_assets"), Font: c.Theme().Font, Size: fontBody, Lang: language})
				}, 1100, 640)
				tt.SetLocale(language)
				tt.Advance(time.Second)
				button, ok := tt.Access(a.T("import"))
				if !ok || button.Bounds.W < importWidth+18+space8+2*space12 || button.Bounds.H < lineHeight {
					t.Fatalf("导入文字被截断或换行：文字%g×%g，按钮%+v", importWidth, lineHeight, button.Bounds)
				}
				input, ok := tt.AccessByRole(ui.RoleTextField, a.T("search_assets"))
				if !ok || input.Bounds.W < searchWidth || input.Bounds.H < lineHeight {
					t.Fatalf("宽工具栏搜索提示被截断或换行：文字%g×%g，输入%+v", searchWidth, lineHeight, input.Bounds)
				}
				if button.Bounds.X+button.Bounds.W > 1100-space12 || input.Bounds.X >= button.Bounds.X {
					t.Fatal("搜索与导入顺序或边界错误")
				}
			})
		}
	}
}

func TestToolbarRound3Screenshots(t *testing.T) {
	if os.Getenv("MOMENTO_TOOLBAR_ROUND3_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_TOOLBAR_ROUND3_SCREENSHOTS=1 输出第三轮工具栏截图")
	}
	dir := "/tmp/momento-polish/shots3/toolbar"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := newScreenshotFixture(t)
	for _, theme := range []string{"light", "dark"} {
		t.Run(theme, func(t *testing.T) {
			a := fixture.app(t, "single", "zh-CN", theme)
			tt := ui.NewTester(a.View, 1440, 900)
			tt.SetLocale(a.Texts.Locale())
			tt.SetTitleBar(ui.TitleBar{Height: MainToolbarHeight, Left: 80, Bottom: 37})
			tt.SetWindowActive(true)
			tt.Advance(time.Second)
			writeScreenshot(t, filepath.Join(dir, fmt.Sprintf("toolbar-%s.png", theme)), tt)
		})
	}
}

func TestToolbarDisabledImportKeepsRaisedSurface(t *testing.T) {
	for _, theme := range []string{"light", "dark"} {
		t.Run(theme, func(t *testing.T) {
			a := toolbarTestApp()
			a.Preferences.Theme, a.Busy = theme, true
			a.SidebarVisible, a.InspectorVisible = false, false
			tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); a.toolbar(c) }, 1100, MainToolbarHeight)
			tt.SetWindowActive(true)
			button, ok := tt.Access(a.T("import"))
			if !ok || !button.Disabled {
				t.Fatal("导入没有保留禁用语义")
			}
			r := button.Bounds
			pixel := tt.Image().RGBAAt(int(r.X+5), int(r.Y+r.H/2))
			palette := paletteFor(momentoTheme(theme, false))
			if pixel.R != palette.Raised.R || pixel.G != palette.Raised.G || pixel.B != palette.Raised.B {
				t.Fatalf("禁用导入底色不一致：%v", pixel)
			}
			tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
			tt.Key(0, ui.KeyEnter)
			if len(tt.Menu()) != 0 {
				t.Fatal("禁用导入仍能打开菜单")
			}
		})
	}
}
