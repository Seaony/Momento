package ui

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestSpacePreviewPrefersHoverWithoutChangingSelection(t *testing.T) {
	for _, view := range []int{0, 1, 2} {
		for _, mode := range []string{"hover-only", "selected-and-hover", "selected-only"} {
			t.Run(string(rune('0'+view))+"/"+mode, func(t *testing.T) {
				a := sidebarTestApp()
				a.ViewMode = view
				tt := ui.NewTester(a.View, 1100, 640)
				if mode != "hover-only" {
					if err := tt.Click("Landscape"); err != nil {
						t.Fatal(err)
					}
					tt.Advance(300 * time.Millisecond)
				}
				want := "a"
				if mode != "selected-only" {
					card, ok := tt.AccessByRole(ui.RoleButton, "Portrait")
					if !ok {
						t.Fatal("缺少悬停素材")
					}
					tt.Move(card.Bounds.X+card.Bounds.W/2, card.Bounds.Y+card.Bounds.H/2)
					want = "b"
				} else {
					tt.Move(0, 0)
				}
				selection, inspector := a.Selected.Len(), a.InspectorVisible
				tt.KeyDown(ui.KeySpace)
				tt.KeyUp(ui.KeySpace)
				if !a.PreviewOpen() || a.PreviewAssetID() != want {
					t.Fatalf("预览=%q，预期%q", a.PreviewAssetID(), want)
				}
				a.FollowPreviewSelection()
				if a.PreviewAssetID() != want || a.Selected.Len() != selection || a.InspectorVisible != inspector {
					t.Fatal("悬停预览改变选择、检查器，或被原选择切回")
				}
				if mode != "hover-only" && (!a.Selected.Has("a") || a.Selected.Has("b")) {
					t.Fatal("悬停预览替换了主窗口选中项")
				}
				if mode != "selected-only" {
					tt.Key(0, ui.KeyLeft)
					if a.PreviewAssetID() != "a" {
						t.Fatalf("上一张没有从当前悬停预览图开始翻页：current=%q，items=%+v", a.PreviewAssetID(), a.Preview.active.items)
					}
					tt.Key(0, ui.KeyRight)
					if a.PreviewAssetID() != "b" {
						t.Fatal("下一张没有回到悬停预览图")
					}
				}
				tt.Key(0, ui.KeyEscape)
				if a.PreviewOpen() || a.Selected.Len() != selection || a.InspectorVisible != inspector {
					t.Fatal("关闭悬停预览改变了选择或检查器")
				}
			})
		}
	}
}

func TestHoveredSpacePreviewCanPinWithoutSelecting(t *testing.T) {
	a := sidebarTestApp()
	tt := ui.NewTester(a.View, 1100, 640)
	card, ok := tt.AccessByRole(ui.RoleButton, "Portrait")
	if !ok {
		t.Fatal("缺少悬停素材")
	}
	tt.Move(card.Bounds.X+card.Bounds.W/2, card.Bounds.Y+card.Bounds.H/2)
	tt.KeyDown(ui.KeySpace)
	w := a.Preview.active
	if w == nil || !a.PreviewHeld() {
		t.Fatal("悬停没有进入按住预览")
	}
	tt.Advance(3 * time.Second)
	tt.KeyUp(ui.KeySpace)
	if len(a.Preview.pinned) != 1 || a.Preview.pinned[0] != w || !w.pinned || w.asset.ID != "b" || a.Selected.Len() != 0 || a.InspectorVisible {
		t.Fatal("悬停长按固定改变了素材、选择或检查器")
	}
}

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
					if called != filepath.Join(a.Library.Root, "assets/landscape.png") || a.PreviewOpen() {
						t.Fatalf("空格没有调用系统快速查看：%q", called)
					}
				} else {
					if !a.PreviewOpen() || a.PreviewAssetID() != "a" {
						t.Fatal("卡片有焦点时空格没有打开预览")
					}
					tt.KeyUp(ui.KeySpace)
					if !a.PreviewOpen() {
						t.Fatal("快速松开空格没有保持预览")
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
			if !a.PreviewOpen() || a.PreviewHeld() {
				t.Fatal("自定义 Enter 未打开持续预览")
			}
		})
	}
}
