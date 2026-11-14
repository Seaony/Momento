package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestToolbarResponsiveActionsAndSearchShortcut(t *testing.T) {
	for _, language := range []string{"zh-CN", "en", "ja"} {
		for _, width := range []int{480, 584, 924, 1204} {
			t.Run(fmt.Sprintf("%s/%d", language, width), func(t *testing.T) {
				a := toolbarTestApp()
				a.Texts.Language = language
				a.SidebarVisible, a.InspectorVisible = false, false
				tt := ui.NewTester(func(c *ui.Context) {
					a.applyTheme(c)
					ui.Column(c).Fill().Children(func() { a.toolbar(c); ui.Spacer(c) })
				}, width, 640)
				tt.Frame()
				for _, key := range []string{"filter", "sort", "import"} {
					e, ok := tt.Access(a.T(key))
					if !ok || e.Bounds.X < 0 || e.Bounds.X+e.Bounds.W > float32(width) || e.Bounds.Y+e.Bounds.H > MainToolbarHeight {
						t.Fatalf("%s 超出工具栏：%+v", key, e)
					}
				}
				if width < 384 {
					toolbarClick(t, tt, a.T("view_mode"))
				}
				toolbarClick(t, tt, a.T("view_grid"))
				if a.ViewMode != 1 {
					t.Fatal("窄工具栏无法切换视图")
				}
				tt.Key(0, ui.KeyEscape)
				mods := ui.Ctrl
				if runtime.GOOS == "darwin" {
					mods = ui.Cmd
				}
				tt.Key(mods, ui.KeyK)
				tt.Type("alpine")
				if a.Query.Search != "alpine" || len(a.Visible) != 1 {
					t.Fatalf("搜索快捷键未聚焦：%q", a.Query.Search)
				}
				toolbarClick(t, tt, a.T("clear_search"))
				if a.Query.Search != "" || len(a.Visible) != 3 {
					t.Fatal("清空搜索没有恢复素材")
				}
				importButton, _ := tt.Access(a.T("import"))
				for _, key := range []string{"filter", "sort", "view_mode"} {
					control, ok := tt.Access(a.T(key))
					if ok && control.Bounds.X+control.Bounds.W > importButton.Bounds.X {
						t.Fatalf("导入不是最右入口：%s", key)
					}
				}
				for _, text := range tt.Texts() {
					if strings.ContainsAny(text, "⟦⟧") || strings.Contains(text, "%!") {
						t.Fatalf("文案格式错误：%s", text)
					}
				}
			})
		}
	}
}

func TestToolbarPolishScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_TOOLBAR_POLISH_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_TOOLBAR_POLISH_SCREENSHOTS=1 输出工具栏截图")
	}
	dir := "/tmp/momento-polish/shots2/toolbar"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := newScreenshotFixture(t)
	for _, theme := range []string{"light", "dark"} {
		for _, state := range []string{"wide", "narrow", "max-sidebar", "filters", "sort", "thumbnail", "no-sidebar"} {
			t.Run(theme+"/"+state, func(t *testing.T) {
				a := fixture.app(t, "single", "zh-CN", theme)
				width := 1100
				if state == "wide" {
					width = 1440
					a.InspectorVisible = false
				}
				if state == "max-sidebar" {
					a.SidebarWidth = 340
				}
				if state == "no-sidebar" {
					a.SidebarVisible = false
				}
				tt := ui.NewTester(a.View, width, 900)
				tt.SetLocale(a.Texts.Locale())
				tt.SetTitleBar(ui.TitleBar{Height: MainToolbarHeight, Left: 80, Bottom: 37})
				tt.SetWindowActive(true)
				keys := map[string]string{"filters": "filter", "sort": "sort", "thumbnail": "thumbnail_size", "max-sidebar": "search_assets"}
				if key := keys[state]; key != "" {
					toolbarClick(t, tt, a.T(key))
				}
				tt.Advance(time.Second)
				writeScreenshot(t, filepath.Join(dir, state+"-"+theme+".png"), tt)
			})
		}
	}
}

func TestToolbarConfiguredSearchOpensNarrowPopover(t *testing.T) {
	for _, binding := range []string{"CmdOrCtrl+F", "Alt+G"} {
		t.Run(binding, func(t *testing.T) {
			a := toolbarTestApp()
			a.SidebarVisible, a.InspectorVisible = false, false
			a.Preferences.Shortcuts["search"] = binding
			tt := ui.NewTester(func(c *ui.Context) {
				a.applyTheme(c)
				ui.Column(c).Fill().Children(func() { a.toolbar(c); ui.Spacer(c) })
				a.shortcuts(c)
			}, 480, 640)
			mods, key, _ := parseShortcut(binding)
			tt.Key(mods, key)
			tt.Type("alpine")
			if a.Query.Search != "alpine" {
				t.Fatalf("改绑搜索没有展开并聚焦窄栏搜索：%q", a.Query.Search)
			}
		})
	}
}

func TestToolbarSearchShortcutRespectsDisabledAndRecording(t *testing.T) {
	for _, state := range []string{"disabled", "recording", "rebound"} {
		t.Run(state, func(t *testing.T) {
			a := toolbarTestApp()
			a.SidebarVisible, a.InspectorVisible = false, false
			switch state {
			case "disabled":
				a.Preferences.Shortcuts["search"] = ""
			case "recording":
				a.Settings.RecordingShortcut = "import"
			case "rebound":
				a.Preferences.Shortcuts["search"] = "Alt+G"
			}
			tt := ui.NewTester(func(c *ui.Context) {
				a.applyTheme(c)
				ui.Column(c).Fill().Children(func() { a.toolbar(c); ui.Spacer(c) })
				a.shortcuts(c)
			}, 480, 640)
			mods := ui.Ctrl
			if runtime.GOOS == "darwin" {
				mods = ui.Cmd
			}
			tt.Key(mods, ui.KeyK)
			if a.Toolbar.SearchOpen {
				t.Fatal("禁用、录制或改绑后仍被硬编码搜索快捷键展开")
			}
		})
	}
}
