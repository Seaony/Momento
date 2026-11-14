package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// 切换器宽度随名称变化，长名称在侧栏内省略，上下箭头不被挤出去。
func TestSidebarLibraryTriggerFitsSidebar(t *testing.T) {
	for _, language := range []string{"en", "ja"} {
		for _, sidebarWidth := range []int{196, 236} {
			t.Run(fmt.Sprintf("%s/%d", language, sidebarWidth), func(t *testing.T) {
				a := sidebarTestApp()
				a.Preferences.Language = language
				a.Texts.Language = language
				a.Library.Manifest.DisplayName = "Brand Moodboard Archive for the Spring Campaign"
				width := float32(sidebarWidth - space24)
				tt := ui.NewTester(func(c *ui.Context) {
					a.applyTheme(c)
					ui.Column(c).Width(width).Children(func() { a.sidebarLibrary(c) })
				}, int(width), 80)
				tt.SetLocale(a.Texts.Locale())
				trigger, ok := tt.Find(a.T("library_menu"))
				if !ok || trigger.X+trigger.W > width || trigger.H != sidebarLibraryHeight {
					t.Fatalf("切换器超出侧栏或高度不对：%+v，可用宽度%.0f", trigger, width)
				}
			})
		}
	}
	a := sidebarTestApp()
	a.Library.Manifest.DisplayName = "Art"
	tt := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		ui.Column(c).Width(212).Children(func() { a.sidebarLibrary(c) })
	}, 212, 80)
	if trigger, ok := tt.Find(a.T("library_menu")); !ok || trigger.W >= 212/2 {
		t.Fatalf("短名称时切换器应随内容收窄：%+v", trigger)
	}
}

func TestSidebarRefinementScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_SIDEBAR_REFINEMENT_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_SIDEBAR_REFINEMENT_SCREENSHOTS=1 生成第三轮侧栏截图")
	}
	dir := "/tmp/momento-polish/shots3/sidebar"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, appearance := range []string{"light", "dark"} {
		t.Run(appearance, func(t *testing.T) {
			a := sidebarTestApp()
			a.Preferences.Language, a.Preferences.Theme = "zh-CN", appearance
			a.Texts.Language = "zh-CN"
			a.Library.Manifest.DisplayName = "灵感素材图库"
			a.Snapshot.Folders[0].Name, a.Snapshot.Folders[1].Name = "设计参考", "界面细节"
			open := true
			a.Sidebar.Open = map[string]*bool{"root": &open}
			a.SidebarWidth = 236
			a.SetScope("all", "")
			tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); a.sidebar(c) }, 236, 800)
			tt.SetTitleBar(ui.TitleBar{Height: MainToolbarHeight, Left: 80, Bottom: 37})
			tt.SetLocale(a.Texts.Locale())
			tt.SetWindowActive(true)
			tt.Frame()
			tt.Advance(time.Second)
			writeScreenshot(t, filepath.Join(dir, "sidebar-zh-CN-"+appearance+".png"), tt)
		})
	}
}
