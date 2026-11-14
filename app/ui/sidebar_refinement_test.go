package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func TestSidebarLibraryCountLocalizedTextFits(t *testing.T) {
	for _, language := range []string{"en", "ja"} {
		for _, count := range []int{0, 1, 12000} {
			for _, sidebarWidth := range []int{196, 236} {
				t.Run(fmt.Sprintf("%s/%d/%d", language, count, sidebarWidth), func(t *testing.T) {
					a := sidebarTestApp()
					a.Preferences.Language = language
					a.Texts.Language = language
					assets := make([]model.Asset, count)
					for i := range assets {
						assets[i].ID = fmt.Sprint(i)
					}
					a.Index = catalog.New(model.Snapshot{Assets: assets})
					label := a.T("library_image_count", sidebarCount(count))
					if count == 1 {
						label = a.T("library_image_count_one")
					}
					var textWidth float32
					tt := ui.NewTester(func(c *ui.Context) {
						a.applyTheme(c)
						a.sidebarLibrary(c)
						textWidth, _ = c.MeasureText(0, ui.Span{Text: label, Font: c.Theme().Font, Size: fontSecondary, Lang: language})
					}, sidebarWidth-space24, 80)
					tt.SetLocale(a.Texts.Locale())
					bounds, ok := tt.Find(label)
					if !ok || bounds.W < textWidth || bounds.H > space24 {
						t.Fatalf("图库图片数被截断或换行：%s，边界%+v，需要宽度%.2f", label, bounds, textWidth)
					}
				})
			}
		}
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
