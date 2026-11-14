package ui

import (
	"image"
	"os"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestChromeWidthsAnimateAndInspectorFollowsSelection(t *testing.T) {
	fixture := newScreenshotFixture(t)
	a := fixture.app(t, "small", "zh-CN", "light")
	a.Selected.Clear()
	a.InspectorVisible = false
	tt := ui.NewTester(a.View, 1440, 900)
	tt.Advance(time.Second)
	if a.InspectorShownWidth() != 0 {
		t.Fatal("没有选择时检查器仍占宽")
	}
	a.selectAsset(a.Visible[0].ID, 0, 0)
	tt.Frame()
	tt.Advance(87 * time.Millisecond)
	if width := a.InspectorShownWidth(); width <= 0 || width >= 280 {
		t.Fatalf("检查器没有中间宽度：%v", width)
	}
	tt.Advance(163 * time.Millisecond)
	tt.Frame()
	if a.InspectorShownWidth() != 280 {
		t.Fatalf("检查器未完成展开：%v", a.InspectorShownWidth())
	}
	tt.Key(0, ui.KeyEscape)
	tt.Advance(250 * time.Millisecond)
	tt.Frame()
	if a.Selected.Len() != 0 || a.InspectorVisible || a.InspectorShownWidth() != 0 {
		t.Fatal("Esc没有清选并收起检查器")
	}
	tt.Press(236, 450)
	tt.Move(300, 450)
	tt.Release(300, 450)
	tt.Advance(time.Second)
	if a.SidebarWidth < 290 || a.SidebarWidth > 310 {
		t.Fatalf("拖动侧栏分隔线未保存宽度：%v", a.SidebarWidth)
	}
	a.SidebarWidth = 236
	tt.Frame()
	tt.Advance(time.Second)
	a.SidebarVisible = false
	tt.Frame()
	tt.Advance(87 * time.Millisecond)
	if width := a.SidebarShownWidth(); width <= 0 || width >= 236 {
		t.Fatalf("侧栏没有中间宽度：%v", width)
	}
	tt.Advance(163 * time.Millisecond)
	tt.Frame()
	if a.SidebarShownWidth() != 0 {
		t.Fatal("侧栏未完成收起")
	}
}

func TestChromeMotionScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_CHROME_MOTION_SCREENSHOTS") != "1" {
		t.Skip("显式生成左右栏动画联系图")
	}
	fixture := newScreenshotFixture(t)
	for _, kind := range []string{"sidebar", "inspector"} {
		a := fixture.app(t, "small", "zh-CN", "light")
		a.Selected.Clear()
		a.InspectorVisible = false
		tt := ui.NewTester(a.View, 1440, 900)
		tt.SetTitleBar(ui.TitleBar{Height: MainToolbarHeight, Left: 80, Bottom: 37})
		tt.Advance(time.Second)
		if kind == "sidebar" {
			a.SidebarVisible = false
		} else {
			a.selectAsset(a.Visible[0].ID, 0, 0)
		}
		tt.Frame()
		first := motionFramesOver(tt, image.Rect(0, 0, 1440, 600), 340*time.Millisecond)
		tt.Advance(time.Second)
		if kind == "sidebar" {
			a.SidebarVisible = true
		} else {
			a.CollapseInspector()
		}
		tt.Frame()
		second := motionFramesOver(tt, image.Rect(0, 0, 1440, 600), 340*time.Millisecond)
		writeToolbarMotionSheet(t, "chrome-"+kind, first, second)
	}
}

func TestAssetDoubleClickSurvivesInspectorOpening(t *testing.T) {
	fixture := newScreenshotFixture(t)
	for _, mode := range []int{0, 1, 2} {
		a := fixture.app(t, "small", "zh-CN", "light")
		a.ViewMode = mode
		a.InspectorVisible = false
		a.Selected.Clear()
		tt := ui.NewTester(a.View, 1440, 900)
		tt.Advance(time.Second)
		var id, name string
		var target ui.Rect
		for _, asset := range a.Visible {
			if r, ok := tt.Find(asset.Name); ok && r.X+r.W > 1400 && r.Y > 200 && r.Y+r.H < 850 {
				id, name, target = asset.ID, asset.Name, r
				break
			}
		}
		if id == "" {
			t.Fatalf("视图%d没有找到右侧非首行素材", mode)
		}
		x, y := target.X+target.W-40, target.Y+target.H/2
		tt.ClickAt(x, y)
		tt.Advance(100 * time.Millisecond)
		tt.ClickAt(x, y)
		if !a.PreviewOpen() || a.PreviewAssetID() != id {
			t.Fatalf("视图%d检查器展开吞掉了%s的第二次点击：选中%v，预览%v/%s", mode, name, a.Selected.Len(), a.PreviewOpen(), a.PreviewAssetID())
		}
	}
}

func TestInspectorAvailabilityUpdatesOnlyItsMenuItem(t *testing.T) {
	a := &App{}
	full := 0
	var available []bool
	a.MenuChanged = func() { full++ }
	a.InspectorAvailabilityChanged = func(on bool) { available = append(available, on) }
	tt := ui.NewTester(func(c *ui.Context) { ui.Box(c).Fill(); a.SyncMenuFocus(c) }, 100, 100)
	tt.SetWindowActive(true)
	a.Selected.Add("a")
	tt.Frame()
	a.RevealInspector()
	a.CollapseInspector()
	a.RevealInspector()
	tt.Frame()
	if full != 0 || len(available) != 1 || !available[0] {
		t.Fatalf("收展检查器重建了整套菜单：%d，%v", full, available)
	}
	a.Selected.Clear()
	tt.Frame()
	if full != 0 || len(available) != 2 || available[1] {
		t.Fatalf("清选没有单独禁用检查器菜单：%d，%v", full, available)
	}
}

func TestThumbnailSizeOnlyPersistsItsPreference(t *testing.T) {
	a := &App{Preferences: DefaultPreferences()}
	reconfigured := 0
	a.MenuChanged = func() { reconfigured++ }
	a.PreferencesChanged = func() { reconfigured++ }
	a.SetThumbnailSize(1.2)
	if a.Preferences.ThumbnailSize != 1.2 || reconfigured != 0 {
		t.Fatalf("调缩略图重新配置了其他服务：%v，%d", a.Preferences.ThumbnailSize, reconfigured)
	}
}
