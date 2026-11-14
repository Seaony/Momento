package ui

import (
	"github.com/egoist/mygo/ui"
	"os"
	"path/filepath"
	"testing"
)

func TestSymbolPolishScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_SYMBOL_POLISH_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_SYMBOL_POLISH_SCREENSHOTS=1 生成小符号与主题选择截图")
	}
	dir := "/tmp/momento-polish/shots/symbol-polish"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := newScreenshotFixture(t)
	for _, theme := range []string{"light", "dark"} {
		for _, state := range []string{"main", "library", "appearance", "onboarding-theme", "locked-disabled", "new-library-disabled"} {
			t.Run(theme+"/"+state, func(t *testing.T) {
				a := fixture.app(t, "small", "zh-CN", theme)
				width, height := 1440, 900
				view := a.View
				switch state {
				case "main":
					a.InspectorVisible = false
				case "library":
					a.Sidebar.LibraryOpen = true
				case "appearance":
					a = settingsTestApp("zh-CN", theme)
					a.Settings.Tab = SettingsAppearance
					view = a.SettingsView
					width, height = 760, 560
				case "onboarding-theme":
					a, view = polishedOverlayApp("zh-CN", theme, "onboarding-theme")
				case "locked-disabled":
					a, view = polishedOverlayApp("zh-CN", theme, "locked")
					a.Lock.ErrorKey = ""
				case "new-library-disabled":
					a, view = polishedOverlayApp("zh-CN", theme, "new-library")
				}
				tt := ui.NewTester(view, width, height)
				tt.SetLocale(a.Texts.Locale())
				tt.SetTitleBar(ui.TitleBar{Height: MainToolbarHeight, Left: 80, Bottom: 37})
				tt.SetWindowActive(true)
				tt.Frame()
				writeScreenshot(t, filepath.Join(dir, state+"-"+theme+".png"), tt)
			})
		}
	}
}
