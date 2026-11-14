package ui

import (
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func TestPreviewShowInactivePreservesKeyboardFocus(t *testing.T) {
	if os.Getenv("MOMENTO_PREVIEW_FOCUS_PROBE") != "1" {
		t.Skip("设置 MOMENTO_PREVIEW_FOCUS_PROBE=1 检查原生预览窗口焦点")
	}
	content := ui.View(func(c *ui.Context) {
		c.Root().Background(ui.Transparent)
		ui.Column(c).Fill().Padding(16).Radius(12).Background(c.Theme().Background).Children(func() {
			ui.Text(c, "预览焦点检查").Bold()
			ui.Text(c, "此窗口不请求键盘焦点。")
		})
	})
	var parentFocus atomic.Int32
	parent := mygo.NewWindow(mygo.WindowOptions{Title: "Momento 焦点检查父窗", Width: 360, Height: 220, Hidden: true, Content: content})
	defer parent.Destroy()
	parent.OnFocus(func() { parentFocus.Add(1) })
	if parent.IsVisible() || parent.IsFocused() {
		t.Fatal("隐藏父窗在显示前已可见或取得焦点")
	}
	parent.ShowInactive()
	if !parent.IsVisible() {
		t.Fatal("父窗未实际显示，不能验证非激活显示")
	}
	parentBefore := parent.IsFocused()
	if parentBefore || parentFocus.Load() != 0 {
		t.Fatal("父窗 ShowInactive 取得了键盘焦点")
	}
	for _, ownership := range []string{"unowned", "owned"} {
		t.Run(ownership, func(t *testing.T) {
			var focusEvents atomic.Int32
			options := mygo.WindowOptions{Title: "Momento 预览焦点检查", Width: 300, Height: 180,
				Hidden: true, Frameless: true, Transparent: true, CornerRadius: 12,
				BackgroundColor: "rgba(0,0,0,0)", Content: content}
			if ownership == "owned" {
				options.Parent = parent
			}
			preview := mygo.NewWindow(options)
			defer preview.Destroy()
			preview.OnFocus(func() { focusEvents.Add(1) })
			if preview.IsVisible() || preview.IsFocused() {
				t.Fatal("隐藏预览窗在显示前已可见或取得焦点")
			}
			preview.ShowInactive()
			if !preview.IsVisible() {
				t.Fatal("预览窗未实际显示，不能验证非激活显示")
			}
			deadline := time.NewTimer(350 * time.Millisecond)
			defer deadline.Stop()
			ticker := time.NewTicker(25 * time.Millisecond)
			defer ticker.Stop()
			check := func() {
				if preview.IsFocused() || focusEvents.Load() != 0 {
					t.Fatalf("预览窗取得键盘焦点：focused=%v，focus events=%d", preview.IsFocused(), focusEvents.Load())
				}
				if parent.IsFocused() != parentBefore || parentFocus.Load() != 0 {
					t.Fatalf("预览显示改变了父窗焦点：before=%v，after=%v，focus events=%d", parentBefore, parent.IsFocused(), parentFocus.Load())
				}
			}
			check()
			for {
				select {
				case <-ticker.C:
					check()
				case <-deadline.C:
					check()
					t.Logf("显示后350ms：preview visible=%v，focused=%v，focus events=%d；parent focused=%v，focus events=%d",
						preview.IsVisible(), preview.IsFocused(), focusEvents.Load(), parent.IsFocused(), parentFocus.Load())
					return
				}
			}
		})
	}
	t.Log("仅验证普通窗口非激活显示；未构造父窗持有键盘，也未验证真实空格事件连续性。")
}

func TestPreviewNativeAnimationSettlesExactBounds(t *testing.T) {
	if os.Getenv("MOMENTO_PREVIEW_FOCUS_PROBE") != "1" {
		t.Skip("设置 MOMENTO_PREVIEW_FOCUS_PROBE=1 检查原生窗口动画终点")
	}
	var a *App
	var w *previewWindow
	var parent *mygo.Window
	mygo.RunOnMain(func() {
		parent = mygo.NewWindow(mygo.WindowOptions{Title: "Momento 预览尺寸检查", Width: 1100, Height: 800, Hidden: true})
		parent.ShowInactive()
		a = readyPreviewApp()
		a.Window = parent
		w = a.Preview.active
		w.openedAt = time.Now()
		w.create(a)
	})
	defer mygo.RunOnMain(func() { a.CloseAllPreviewWindows(); parent.Destroy() })
	for _, phase := range []string{"open", "pin"} {
		if phase == "pin" {
			mygo.RunOnMain(func() {
				w.picture.Held = true
				w.pressedAt = time.Now().Add(-previewPinThreshold)
				w.advanceHold(a, time.Now())
				w.invalidate()
			})
		}
		time.Sleep(350 * time.Millisecond)
		if _, err := w.window.CapturePage(); err != nil {
			t.Fatal(err)
		}
		t.Log("窗口可能被其他Space遮挡；CapturePage推进当前时刻，只验证动画终点，不宣称自然350ms帧时长。")
		mygo.RunOnMain(func() {
			if got := w.window.Bounds(); got != w.frame {
				t.Errorf("%s终点尺寸%+v，预期%+v", phase, got, w.frame)
			}
			if got := w.window.Opacity(); got != 1 {
				t.Errorf("%s终点透明度%v，预期1", phase, got)
			}
			if w.window.IsFocused() {
				t.Error("非激活显示取得了键盘")
			}
			t.Logf("%s: bounds=%+v opacity=%v focused=%v", phase, w.window.Bounds(), w.window.Opacity(), w.window.IsFocused())
		})
	}
}
