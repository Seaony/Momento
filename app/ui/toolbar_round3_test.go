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
				var searchWidth, lineHeight float32
				tt := ui.NewTester(func(c *ui.Context) {
					a.applyTheme(c)
					ui.Column(c).Fill().Children(func() { a.toolbar(c); ui.Spacer(c) })
					searchWidth, lineHeight = c.MeasureText(0, ui.Span{Text: a.T("search_assets"), Font: c.Theme().Font, Size: toolbarFont, Lang: language})
				}, 1100, 640)
				tt.SetLocale(language)
				tt.Advance(time.Second)
				// 导入是图标按钮，只检查它在最右边。
				button, ok := tt.Access(a.T("import"))
				if !ok {
					t.Fatal("找不到导入按钮")
				}
				// 照设计稿搜索框可以缩窄：英文放得下完整提示；日文按钮文字长，提示可以截断，但搜索框不能小于最小宽度、不能换行。
				input, ok := tt.AccessByRole(ui.RoleTextField, a.T("search_assets"))
				if !ok || input.Bounds.H < lineHeight*.9 || input.Bounds.H > lineHeight*1.5 || (language == "en" && input.Bounds.W < searchWidth) {
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

func TestToolbarDisabledImportBlocksMenu(t *testing.T) {
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
			tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
			tt.Key(0, ui.KeyEnter)
			if len(tt.Menu()) != 0 {
				t.Fatal("禁用导入仍能打开菜单")
			}
		})
	}
}
