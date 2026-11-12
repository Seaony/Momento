package ui

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestFocusedAssetPreviewShortcut(t *testing.T) {
	for _, view := range []int{0, 1, 2} {
		for _, mode := range []string{"preview", "system"} {
			t.Run(string(rune('0'+view))+"/"+mode, func(t *testing.T) {
				if mode == "system" && runtime.GOOS != "darwin" {
					t.Skip("系统快速查看仅在 macOS 提供")
				}
				a := sidebarTestApp()
				a.Preferences.SpaceKey = mode
				a.ViewMode = view
				a.Library.Root = t.TempDir()
				a.Snapshot.Assets[0].RelativePath = "assets/landscape.png"
				a.applySnapshot(a.Snapshot)
				called := ""
				old := previewSystemItem
				previewSystemItem = func(path string) error { called = path; return nil }
				t.Cleanup(func() { previewSystemItem = old })
				tt := ui.NewTester(a.View, 1100, 640)
				if err := tt.Click("Landscape"); err != nil {
					t.Fatal(err)
				}
				if !tt.Focused("Landscape") {
					t.Fatal("点击卡片后没有取得焦点")
				}
				tt.KeyDown(ui.KeySpace)
				if mode == "system" {
					if called != filepath.Join(a.Library.Root, "assets/landscape.png") || a.Preview.Open {
						t.Fatalf("空格没有调用系统快速查看：%q", called)
					}
				} else {
					if !a.Preview.Open || a.Preview.ID != "a" {
						t.Fatal("卡片有焦点时空格没有打开预览")
					}
					tt.KeyUp(ui.KeySpace)
					if a.Preview.Open {
						t.Fatal("松开空格后预览未关闭")
					}
				}
			})
		}
	}
}

func TestFocusedAssetEnterPreviewShortcut(t *testing.T) {
	for _, view := range []int{0, 1, 2} {
		t.Run(string(rune('0'+view)), func(t *testing.T) {
			a := sidebarTestApp()
			a.ViewMode = view
			a.Preferences.Shortcuts = map[string]string{"preview": "Enter"}
			tt := ui.NewTester(a.View, 1100, 640)
			if err := tt.Click("Landscape"); err != nil {
				t.Fatal(err)
			}
			tt.Key(0, ui.KeyEnter)
			if !a.Preview.Open || a.Preview.Held {
				t.Fatal("自定义 Enter 未打开持续预览")
			}
		})
	}
}
