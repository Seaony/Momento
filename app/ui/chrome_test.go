package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

var chromeTestBars = []struct {
	name string
	bar  ui.TitleBar
}{
	{"macos", ui.TitleBar{Height: MainToolbarHeight, Left: 80, Bottom: 37}},
	{"windows", ui.TitleBar{Height: MainToolbarHeight, Right: 138, Bottom: MainToolbarHeight}},
	{"linux", ui.TitleBar{Height: MainToolbarHeight, Right: 112, Bottom: MainToolbarHeight}},
}

func TestNativeWindowControlsLeaveActionsAccessible(t *testing.T) {
	for _, platform := range chromeTestBars {
		t.Run(platform.name, func(t *testing.T) {
			a := toolbarTestApp()
			a.SidebarVisible, a.InspectorVisible = false, false
			tt := ui.NewTester(a.toolbar, 1100, MainToolbarHeight)
			tt.SetTitleBar(platform.bar)
			left, leftOK := tt.Find(a.T("toggle_sidebar"))
			right, rightOK := tt.Find(a.T("toggle_inspector"))
			if !leftOK || !rightOK || left.X < platform.bar.Left || right.X+right.W > 1100-platform.bar.Right {
				t.Fatalf("工具栏操作被系统窗口按钮覆盖：左%+v，右%+v，按钮区域%+v", left, right, platform.bar)
			}
			if err := tt.Click(a.T("toggle_sidebar")); err != nil {
				t.Fatal(err)
			}
			side := ui.NewTester(a.sidebar, 236, 640)
			side.SetTitleBar(platform.bar)
			library, ok := side.Find(a.T("library_menu"))
			if !ok {
				t.Fatal("图库切换器不可见")
			}
			if platform.bar.Left > 0 {
				if library.Y < platform.bar.Bottom {
					t.Fatalf("图库切换器被系统窗口按钮覆盖：%+v", library)
				}
			} else if library.Y+library.H/2 != MainToolbarHeight/2 {
				t.Fatalf("没有左侧窗口按钮时，图库切换器应在工具栏首行居中：%+v", library)
			}
			a.Preferences.Onboarded = true
			welcome := ui.NewTester(a.welcome, 1100, 640)
			welcome.SetTitleBar(platform.bar)
			create, ok := welcome.Find(a.T("new_library"))
			if !ok || create.Y < platform.bar.Bottom {
				t.Fatalf("欢迎页操作被系统窗口按钮覆盖：%+v", create)
			}
		})
	}
}

func TestTitleBarInsetScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_TITLEBAR_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_TITLEBAR_SCREENSHOTS=1 导出窗口按钮留位截图")
	}
	dir := "/tmp/momento-titlebar-layout"
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	fixture := newScreenshotFixture(t)
	var records []struct {
		File string      `json:"file"`
		Bar  ui.TitleBar `json:"simulatedNativeControls"`
	}
	for _, platform := range chromeTestBars {
		for _, appearance := range []string{"light", "dark"} {
			for _, state := range []string{"small", "no-sidebar", "no-inspector", "welcome", "tags"} {
				a := fixture.app(t, "small", "en", appearance)
				switch state {
				case "no-sidebar":
					a.SidebarVisible = false
				case "no-inspector":
					a.InspectorVisible = false
				case "welcome":
					a.Library = nil
				case "tags":
					a.TagPage = true
				}
				tt := ui.NewTester(a.View, 1100, 640)
				tt.SetTitleBar(platform.bar)
				file := fmt.Sprintf("%s-%s-%s.png", platform.name, state, appearance)
				writeScreenshot(t, filepath.Join(dir, file), tt)
				records = append(records, struct {
					File string      `json:"file"`
					Bar  ui.TitleBar `json:"simulatedNativeControls"`
				}{file, platform.bar})
			}
		}
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
}
