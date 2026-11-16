package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/library"
	"github.com/egoist/mygo/ui"
)

func welcomeMissingApp(language, theme string, menu bool) *App {
	a := settingsTestApp(language, theme)
	a.Preferences.Onboarded = true
	a.Recent = []library.Recent{{Path: "/Volumes/Photos/设计灵感.momento", DisplayName: "设计灵感"}, {Path: "/examples/photos.momento", DisplayName: "旅行与摄影"}}
	a.RecentMissing = map[string]bool{a.Recent[0].Path: true}
	a.Welcome.RecentChecked = true
	if menu {
		open := true
		a.Sidebar.LibraryMore = map[string]*bool{a.Recent[0].Path: &open}
	}
	return a
}

// 没有打开的图库时只有欢迎页：找不到的图库要在这里标出来，并能重新定位或移出列表。
func TestWelcomeMarksMissingLibraryWithActions(t *testing.T) {
	a := welcomeMissingApp("en", "light", true)
	tt := ui.NewTester(a.welcome, 1100, 640)
	tt.Advance(time.Second)
	if !tt.HasText(a.T("library_not_found")) {
		t.Fatal("欢迎页没有标出找不到的图库")
	}
	if !tt.HasText(a.T("relocate_library")) || !tt.HasText(a.T("remove_recent_library")) {
		t.Fatal("欢迎页找不到的图库没有重新定位和移除入口")
	}
}

func TestWelcomeMissingScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_STAGE3_SHOTS") != "1" {
		t.Skip("设置 MOMENTO_STAGE3_SHOTS=1 生成欢迎页找不到图库截图")
	}
	dir := "/tmp/momento-stage3/welcome"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, theme := range []string{"light", "dark"} {
		a := welcomeMissingApp("zh-CN", theme, true)
		tt := ui.NewTester(a.welcome, 1100, 640)
		tt.SetLocale(a.Texts.Locale())
		tt.SetDark(theme == "dark")
		tt.SetTitleBar(ui.TitleBar{Height: MainToolbarHeight, Left: 80, Bottom: 37})
		tt.Advance(time.Second)
		writeScreenshot(t, filepath.Join(dir, "missing-zh-CN-"+theme+".png"), tt)
	}
}
